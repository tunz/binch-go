package binch

import (
	"github.com/bnagy/gapstone"
	"github.com/keystone-engine/keystone/bindings/go/keystone"
	"github.com/tunz/binch-go/pkg/io"
	"log"
	"sort"
	"strings"
)

const maxInstrBytes = 15 // Maximum bytes of x86 instructions.

type addrIdxInfo struct {
	SectionBase uint64
	ArrIdx      int
}

type changeInfo struct {
	addr uint64
	data []byte
}

// Project groups binary and assembly engines.
type Project struct {
	binary       *bcio.Binary
	assembler    *keystone.Keystone
	disassembler *gapstone.Engine
	section2code map[uint64][]*Instruction
	addr2idx     map[uint64]addrIdxInfo
	changes      []changeInfo
}

// Instruction is a simplified struct of gapstone.Instruction
type Instruction struct {
	Name    string
	Address uint64
	Bytes   []byte
	Str     string
}

// MakeAssembler creates a keystone engine.
func makeAssembler(machineType string) *keystone.Keystone {
	var ks *keystone.Keystone
	var err error
	switch machineType {
	case "EM_AARCH64":
		ks, err = keystone.New(keystone.ARCH_ARM, keystone.MODE_64)
	case "EM_386":
		ks, err = keystone.New(keystone.ARCH_X86, keystone.MODE_32)
	case "EM_X86_64":
		ks, err = keystone.New(keystone.ARCH_X86, keystone.MODE_64)
	}

	if err != nil {
		panic(err)
	}
	return ks
}

// MakeDisassembler create a capstone engine.
func makeDisassembler(machineType string) *gapstone.Engine {
	var cs gapstone.Engine
	var err error
	switch machineType {
	case "EM_AARCH64":
		cs, err = gapstone.New(gapstone.CS_ARCH_ARM, gapstone.CS_MODE_64)
	case "EM_386":
		cs, err = gapstone.New(gapstone.CS_ARCH_X86, gapstone.CS_MODE_32)
	case "EM_X86_64":
		cs, err = gapstone.New(gapstone.CS_ARCH_X86, gapstone.CS_MODE_64)
	}

	if err != nil {
		panic(err)
	}
	return &cs
}

func (p *Project) makeInstruction(ins gapstone.Instruction) *Instruction {
	opStr := ins.Mnemonic + " " + ins.OpStr
	addr := uint64(ins.Address)
	symbolName := p.binary.Addr2Symbol[addr]
	return &Instruction{
		Name:    symbolName,
		Address: addr,
		Bytes:   ins.Bytes,
		Str:     opStr,
	}
}

func (p *Project) disasmAll(sectionBase uint64, size uint64) []*Instruction {
	buf := p.binary.ReadMemory(sectionBase, size)
	if buf == nil {
		return nil
	}
	insns, err := p.disassembler.Disasm(buf, sectionBase, 0)
	if err != nil {
		return nil
	}

	result := make([]*Instruction, 0, len(insns))
	for idx, ins := range insns {
		result = append(result, p.makeInstruction(ins))
		p.addr2idx[uint64(ins.Address)] = addrIdxInfo{
			SectionBase: sectionBase,
			ArrIdx:      idx,
		}
	}
	return result
}

func (p *Project) findSectionIdx(addr uint64) int {
	return sort.Search(len(p.binary.CodeSections), func(i int) bool {
		return p.binary.CodeSections[i].Addr > addr
	}) - 1
}

func (p *Project) getSectionCodeFromBase(base uint64) []*Instruction {
	if code, exists := p.section2code[base]; exists {
		return code
	}

	sectionIdx := p.findSectionIdx(base)
	size := p.binary.CodeSections[sectionIdx].Size
	p.section2code[base] = p.disasmAll(base, size)
	return p.section2code[base]
}

func (p *Project) findSectionCode(addr uint64) []*Instruction {
	idx := p.findSectionIdx(addr)
	if idx < 0 {
		return nil
	}
	base := p.binary.CodeSections[idx].Addr
	sz := p.binary.CodeSections[idx].Size
	if addr < base || addr >= base+sz {
		return nil
	}
	return p.getSectionCodeFromBase(base)
}

// FindPrevInstruction finds a previous instruction by checking every code
// section.
func (p *Project) FindPrevInstruction(addr uint64) *Instruction {
	info, exists := p.addr2idx[addr]
	if !exists {
		log.Panicln("FindPrevInstructions is called from unknown address.")
	}

	if info.SectionBase == addr {
		sectionIdx := p.findSectionIdx(addr)
		if p.binary.CodeSections[sectionIdx].Addr != addr {
			log.Panicln("p.binary.CodeSections[sectionIdx].Addr != addr")
		}
		if sectionIdx == 0 {
			return nil
		}
		base := p.binary.CodeSections[sectionIdx-1].Addr
		code := p.getSectionCodeFromBase(base)
		return code[len(code)-1]
	}

	code := p.getSectionCodeFromBase(info.SectionBase)
	return code[info.ArrIdx-1]
}

// FindNextInstruction finds a next instruction by checking every code
// section.
func (p *Project) FindNextInstruction(addr uint64) *Instruction {
	info, exists := p.addr2idx[addr]
	if !exists {
		log.Panicln("FindNextInstructions is called from unknown address.")
	}

	code := p.getSectionCodeFromBase(info.SectionBase)
	if info.ArrIdx == len(code)-1 {
		sectionIdx := sort.Search(len(p.binary.CodeSections), func(i int) bool {
			return p.binary.CodeSections[i].Addr >= info.SectionBase
		})
		if p.binary.CodeSections[sectionIdx].Addr != info.SectionBase {
			log.Panicln("p.binary.CodeSections[sectionIdx].Addr != infoSectionBase")
		}
		if sectionIdx+1 == len(p.binary.CodeSections) {
			return nil
		}
		base := p.binary.CodeSections[sectionIdx+1].Addr
		code := p.getSectionCodeFromBase(base)
		return code[0]
	}

	return code[info.ArrIdx+1]
}

// GetInstruction find and returns an instruction.
func (p *Project) GetInstruction(addr uint64) *Instruction {
	if info, exists := p.addr2idx[addr]; exists {
		code := p.getSectionCodeFromBase(info.SectionBase)
		return code[info.ArrIdx]
	}
	code := p.findSectionCode(addr)
	if code == nil {
		return nil
	}
	info := p.addr2idx[addr]
	return code[info.ArrIdx]
}

// Entry returns binary entry point.
func (p *Project) Entry() uint64 {
	return p.binary.Entry
}

// Assemble returns byte codes of a given instruction.
func (p *Project) Assemble(instr string, addr uint64) []byte {
	// LLVM has some weird syntax check. It does not catch syntax errors for
	// mismatched brackets. So, we catch them here.
	if strings.Count(instr, "[") != strings.Count(instr, "]") {
		return nil
	}
	encoding, _, ok := p.assembler.Assemble(instr, addr)
	if !ok {
		return nil
	}
	return encoding
}

// Disassemble returns an instruction of a given byte code.
func (p *Project) Disassemble(buf []byte, addr uint64) *Instruction {
	if insns, err := p.disassembler.Disasm(buf, addr, 1); err == nil {
		return p.makeInstruction(insns[0])
	}
	return nil
}

func copyData(data []byte) []byte {
	newData := make([]byte, len(data))
	copy(newData, data)
	return newData
}

// WriteMemory write data into memory, and remove code caches if necessary.
func (p *Project) WriteMemory(addr uint64, data []byte) {
	origData := copyData(p.binary.ReadMemory(addr, uint64(len(data))))
	p.changes = append(p.changes, changeInfo{addr: addr, data: origData})
	p.recWriteMemory(addr, data)
}

func (p *Project) recWriteMemory(addr uint64, data []byte) {
	sectionIdx := p.findSectionIdx(addr)
	delete(p.section2code, p.binary.CodeSections[sectionIdx].Addr)
	r := p.binary.WriteMemory(addr, data)
	if r < len(data) {
		p.recWriteMemory(addr+uint64(r), data[r:])
	}
}

// Undo latest patch.
func (p *Project) Undo() {
	if len(p.changes) == 0 {
		return
	}
	last := p.changes[len(p.changes)-1]
	p.changes = p.changes[:len(p.changes)-1]
	p.recWriteMemory(last.addr, last.data)
}

// Save just calls the save method of binary.
func (p *Project) Save() {
	p.binary.Save()
}

// MakeProject creates a binch project object.
func MakeProject(b *bcio.Binary) *Project {
	return &Project{
		binary:       b,
		assembler:    makeAssembler(b.MachineType),
		disassembler: makeDisassembler(b.MachineType),
		section2code: make(map[uint64][]*Instruction),
		addr2idx:     make(map[uint64]addrIdxInfo),
		changes:      make([]changeInfo, 0),
	}
}
