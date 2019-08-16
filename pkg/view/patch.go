package bcview

import (
	"encoding/hex"
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/tunz/binch-go/pkg/core"
	"log"
	"strings"
	"unicode"
)

func hex2bytes(hexStr string) []byte {
	stripped := strings.ReplaceAll(hexStr, " ", "")
	decoded, err := hex.DecodeString(stripped)
	if err != nil {
		log.Println(hexStr)
		log.Panicln(err)
	}
	return decoded
}

func nopPadding(bytes []byte, size int) []byte {
	for i := 0; i < size-len(bytes); i++ {
		bytes = append(bytes, 0x90)
	}
	return bytes
}

func (h *handler) patchByte(g *gocui.Gui, v *gocui.View) error {
	return nil
}

func (h *handler) patchInstr(g *gocui.Gui, v *gocui.View) error {
	instr := h.lines[h.cursor].data.(*binch.Instruction)
	str, _ := v.Line(0)
	if bytes := h.project.Assemble(str, instr.Address); bytes != nil {
		padded := nopPadding(bytes, len(instr.Bytes))
		h.project.WriteMemory(instr.Address, padded)
		h.redraw()
		return h.exitPatch(g, v)
	}
	return nil
}

func (h *handler) deleteInstr(g *gocui.Gui, v *gocui.View) error {
	instr := h.lines[h.cursor].data.(*binch.Instruction)
	nop := make([]byte, len(instr.Bytes))
	for i := 0; i < len(instr.Bytes); i++ {
		nop[i] = 0x90
	}
	h.project.WriteMemory(instr.Address, nop)
	h.redraw()
	return nil
}

func (h *handler) instrEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	h.mux.Lock()

	curStr, _ := v.Line(0)
	cx, _ := v.Cursor()
	if key == gocui.KeyArrowDown || (key == gocui.KeyArrowRight && cx+1 > len(curStr)) {
		h.mux.Unlock()
		// Do not move to the next line.
		return
	}

	gocui.DefaultEditor.Edit(v, key, ch, mod)

	if key != gocui.KeyArrowLeft && key != gocui.KeyArrowRight {
		str, _ := v.Line(0)
		h.mux.Unlock()
		h.instrEvents <- str
	} else {
		h.mux.Unlock()
	}
}

func isHexadecimal(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isBlank(x int) bool {
	// Every third positions are blanks.
	// e.g.)
	// 11 11 11 11
	//   ^  ^  ^
	return (x+1)%3 == 0
}

func (h *handler) byteEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	h.mux.Lock()

	hexString, _ := v.Line(0)
	cx, _ := v.Cursor()

	switch {
	case key == gocui.KeyArrowRight || ch == 'l':
		if cx+1 < len(hexString) {
			v.MoveCursor(1, 0, false)
			if isBlank(cx + 1) {
				// If updated cursor is poiting a blank, move to the next digit.
				v.MoveCursor(1, 0, false)
			}
		}
	case key == gocui.KeyArrowLeft || ch == 'h':
		v.MoveCursor(-1, 0, false)
		if isBlank(cx - 1) {
			// If updated cursor is poiting a blank, move to the next digit.
			v.MoveCursor(-1, 0, false)
		}
	case isHexadecimal(rune(hexString[cx])) && isHexadecimal(ch):
		v.EditWrite(unicode.ToLower(ch))
		str, _ := v.Line(0)
		h.byteEvents <- str

		if cx+1 >= len(hexString) {
			// EditWrite moves cursor to right, so if it goes over the boundary,
			// then roll back the cursor.
			v.MoveCursor(-1, 0, false)
		} else if isBlank(cx + 1) {
			v.MoveCursor(1, 0, false)
		}
	}
	h.mux.Unlock()
}

func isSameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}

func (h *handler) watchInstrChange(byteView, instrView *gocui.View, addr uint64, origBytes []byte) {
	for instrStr := range h.instrEvents {
		if newBytes := h.project.Assemble(instrStr+"\n", addr); newBytes != nil {
			h.mux.Lock()
			byteView.Clear()
			if isSameBytes(newBytes, origBytes) {
				fmt.Fprintf(byteView, "% x", newBytes)
			} else if !isSameBytes(newBytes, origBytes) && len(newBytes) <= len(origBytes) {
				fmt.Fprintf(byteView, "\x1b[0;32m% x\x1b[m", newBytes)

				// Fill remaining bytes with nops
				for i := len(newBytes); i < len(origBytes); i++ {
					if i != 0 {
						fmt.Fprintf(byteView, " ")
					}
					fmt.Fprintf(byteView, "90")
				}
			} else if len(origBytes) < len(newBytes) {
				fmt.Fprintf(byteView, "\x1b[0;32m% x\x1b[m", newBytes[:len(origBytes)])
				fmt.Fprintf(byteView, " \x1b[0;31m% x\x1b[m", newBytes[len(origBytes):])
			}

			instrView.Clear()
			fmt.Fprintf(instrView, "\x1b[0;37m%s\x1b[m", instrStr)
		} else {
			h.mux.Lock()
			byteView.Clear()
			fmt.Fprintf(byteView, "% x", origBytes)

			instrView.Clear()
			fmt.Fprintf(instrView, "\x1b[0;31m%s\x1b[m", instrStr)
		}

		// New line for gocui to make sure to generate a line.
		fmt.Fprintf(byteView, "\n")
		fmt.Fprintf(instrView, "\n")
		h.mux.Unlock()

		flush(h.gui)
	}
}

func (h *handler) watchByteChange(byteView, instrView *gocui.View, addr uint64) {
	for opcode := range h.byteEvents {
		bytes := hex2bytes(opcode)
		if instr := h.project.Disassemble(bytes, addr); instr != nil {
			instrView.Clear()
			fmt.Fprintf(instrView, "%s\n", instr.Str)

			byteView.Clear()
			fmt.Fprintf(byteView, "\x1b[0;32m% x\x1b[m", instr.Bytes)
			if len(instr.Bytes) < len(bytes) {
				fmt.Fprintf(byteView, " \x1b[0;31m% x\x1b[m", bytes[len(instr.Bytes):])
			}
			fmt.Fprintf(byteView, "\n")
		} else {
			instrView.Clear()
			fmt.Fprintf(instrView, "\x1b[0;31mInvalid Op\x1b[m\n")
		}
		flush(h.gui)
	}
}

func (h *handler) showPatch(g *gocui.Gui, v *gocui.View) error {
	var err error
	var byteView, instrView *gocui.View

	maxX, maxY := g.Size()
	if v, err := g.SetView("patch", maxX/2-40, maxY/2-4, maxX/2+40, maxY/2+3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Patch"
	}
	curInstr := h.lines[h.cursor].data.(*binch.Instruction)
	if byteView, err = g.SetView("patchByte", maxX/2-35, maxY/2-3, maxX/2+35, maxY/2-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		byteView.Title = "Bytes"
		byteView.Editable = true
		byteView.Overwrite = true
		byteView.Editor = gocui.EditorFunc(h.byteEditor)
		fmt.Fprintf(byteView, "% x", curInstr.Bytes)
	}
	if instrView, err = g.SetView("patchInstr", maxX/2-35, maxY/2, maxX/2+35, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		instrView.Title = "Instruction"
		instrView.Editable = true
		instrView.Editor = gocui.EditorFunc(h.instrEditor)
		fmt.Fprintf(instrView, "%s", curInstr.Str)
		if err := instrView.SetCursor(len(curInstr.Str), 0); err != nil {
			return err
		}
	}
	g.Cursor = true

	instrAddress := curInstr.Address
	origBytes := curInstr.Bytes

	h.byteEvents = make(chan string, 20)
	go h.watchByteChange(byteView, instrView, instrAddress)

	h.instrEvents = make(chan string, 20)
	go h.watchInstrChange(byteView, instrView, instrAddress, origBytes)

	if _, err := setCurrentViewOnTop(g, "patchInstr"); err != nil {
		return err
	}

	return nil
}

func patchMoveFocus(g *gocui.Gui, v *gocui.View) error {
	if g.CurrentView().Name() == "patchByte" {
		instrView, _ := g.View("patchInstr")
		line, _ := instrView.Line(0)
		instrView.SetCursor(len(line), 0)

		if _, err := g.SetCurrentView("patchInstr"); err != nil {
			return err
		}
	} else {
		if _, err := g.SetCurrentView("patchByte"); err != nil {
			return err
		}
	}
	return nil
}

func (h *handler) exitPatch(g *gocui.Gui, v *gocui.View) error {
	g.Cursor = false
	exitView(g, "patchByte")
	exitView(g, "patchInstr")
	exitView(g, "patch")
	close(h.byteEvents)
	close(h.instrEvents)
	return nil
}
