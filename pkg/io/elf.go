package bcio

import (
	"debug/elf"
	"log"
	"os"
	"sort"
)

type memSegment struct {
	Vaddr   uint64
	Offset  int64
	Memsz   uint64
	Data    []uint8
	changes map[int]byte
}

type codeSection struct {
	Addr uint64
	Size uint64
}

// Binary type stores information about how to load a file to memory.
type Binary struct {
	filename     string
	memory       []memSegment
	Symbol2Addr  map[string]uint64
	Addr2Symbol  map[uint64]string
	CodeSections []codeSection
	Entry        uint64
	MachineType  string
}

// Save overwrites the changes into binary.
func (b *Binary) Save() {
	f, err := os.OpenFile(b.filename, os.O_RDWR, 0644)
	if err != nil {
		panic("No such file")
	}
	defer f.Close()

	for _, m := range b.memory {
		for idx, val := range m.changes {
			f.WriteAt([]byte{val}, m.Offset+int64(idx))
		}
	}
}

// ReadMemory reads memory bytes from binary.
func (b *Binary) ReadMemory(addr uint64, size uint64) []uint8 {
	for _, m := range b.memory {
		if addr >= m.Vaddr && addr < m.Vaddr+m.Memsz {
			if addr-m.Vaddr+size <= uint64(len(m.Data)) {
				return m.Data[addr-m.Vaddr : addr-m.Vaddr+size]
			}
			data := m.Data[addr-m.Vaddr:]
			sz := uint64(len(data))
			return append(data, b.ReadMemory(addr+sz, size-sz)...)
		}
	}
	return nil
}

// WriteMemory write memory bytes into binary. If it tries to write multiple
// memory segments, only update the first segment.
func (b *Binary) WriteMemory(addr uint64, data []byte) int {
	for _, m := range b.memory {
		if addr >= m.Vaddr && addr < m.Vaddr+m.Memsz {
			base := int(addr - m.Vaddr)
			if base+len(data) <= len(m.Data) {
				for i := 0; i < len(data); i++ {
					m.changes[base+i] = data[i]
					m.Data[base+i] = data[i]
				}
				return len(data)
			}

			size := len(m.Data) - base
			for i := 0; i < size; i++ {
				m.changes[base+i] = data[i]
				m.Data[base+i] = data[i]
			}
			return size
		}
	}
	return -1
}

func loadCodeSegments(f *os.File, _elf *elf.File) []memSegment {
	memory := make([]memSegment, 0, len(_elf.Progs))
	for _, prog := range _elf.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}

		align := uint64(0x1000)
		pageoffset := prog.Vaddr & (align - 1)

		memsz := prog.Memsz + pageoffset
		offset := int64(prog.Off - pageoffset)
		filesz := prog.Filesz + pageoffset
		vaddr := prog.Vaddr - pageoffset
		memsz = (memsz + align) & ^(align - 1)

		f.Seek(offset, os.SEEK_SET)
		buf := make([]uint8, filesz, filesz)
		_, err := f.Read(buf)
		if err != nil {
			log.Panicln(err)
		}

		memory = append(memory, memSegment{
			Vaddr:   vaddr,
			Offset:  offset,
			Memsz:   memsz,
			Data:    buf,
			changes: make(map[int]byte),
		})
	}
	return memory
}

func loadSymbols(_elf *elf.File) (map[string]uint64, map[uint64]string) {
	symbol2addr := make(map[string]uint64)
	addr2symbol := make(map[uint64]string)

	symbols, err := _elf.Symbols()
	if err != nil {
		return symbol2addr, addr2symbol
	}

	for _, symbol := range symbols {
		infoType := symbol.Info & 0xf
		if infoType == 1 || infoType == 2 { // Symbol Type is STT_FUNC or STT_OBJECT
			symbol2addr[symbol.Name] = symbol.Value
			addr2symbol[symbol.Value] = symbol.Name
		}
		// TODO: Check Thumb or ARM type for ARM ELFs.
	}
	return symbol2addr, addr2symbol
}

func findCodeSection(_elf *elf.File) []codeSection {
	codeSections := make([]codeSection, 0, len(_elf.Sections)/3)
	for _, section := range _elf.Sections {
		if section.Flags&elf.SHF_EXECINSTR == elf.SHF_EXECINSTR {
			// We simply assume that every executable section is code section
			// such as .text, .init, .plt.
			codeSections = append(codeSections, codeSection{
				Addr: section.Addr,
				Size: section.Size,
			})
		}
	}
	sort.Slice(codeSections, func(i, j int) bool {
		addr1 := codeSections[i].Addr
		addr2 := codeSections[j].Addr
		return addr1 < addr2 || (addr1 == addr2 && codeSections[i].Size < codeSections[j].Size)
	})
	return codeSections
}

// ReadElf loads a ELF binary.
func ReadElf(filename string) *Binary {
	f, err := os.Open(filename)
	if err != nil {
		panic("No such file")
	}
	defer f.Close()

	_elf, err := elf.NewFile(f)
	if err != nil {
		panic("Failed to load the ELF file")
	}

	memory := loadCodeSegments(f, _elf)
	symbol2addr, addr2symbol := loadSymbols(_elf)
	codeSections := findCodeSection(_elf)

	return &Binary{
		filename:     filename,
		memory:       memory,
		Symbol2Addr:  symbol2addr,
		Addr2Symbol:  addr2symbol,
		CodeSections: codeSections,
		Entry:        _elf.Entry,
		MachineType:  _elf.Machine.String(),
	}
}
