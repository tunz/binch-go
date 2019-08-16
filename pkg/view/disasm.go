package bcview

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/tunz/binch-go/pkg/core"
	"log"
	"strconv"
	"strings"
)

/********/
/* Draw *
/********/

func (h *handler) updateBuffer() {
	v, _ := h.gui.View("disasm")
	v.Clear()
	emptyLine := strings.Repeat(" ", 138)
	for idx, line := range h.lines {
		if idx == h.cursor {
			fmt.Fprintf(v, "\x1b[0;30;47m")
		}
		switch line.kind {
		case instrKind:
			instr := line.data.(*binch.Instruction)
			fmt.Fprintf(v, "0x%-16x% -45x%-75s", instr.Address, instr.Bytes, instr.Str)
		case symbolKind:
			name := line.data.(string)
			fmt.Fprintf(v, "; %s", name)
		case emptyKind:
			fmt.Fprintf(v, "%s", emptyLine)
		}
		if idx == h.cursor {
			fmt.Fprintf(v, "\x1b[m")
		}
		fmt.Fprintf(v, "\n")
	}
}

func (h *handler) redraw() {
	origCursor := h.cursor
	idx := h.findNextLine(0)
	firstInstr := h.lines[idx].data.(*binch.Instruction)
	h.drawFromTop(firstInstr.Address)
	h.cursor = origCursor
	h.updateBuffer()
}

func (h *handler) drawFromTop(addr uint64) {
	instr := h.project.GetInstruction(addr)
	if instr == nil {
		h.popupEvents <- fmt.Sprintf("No Such Instruction (Address: 0x%x)", addr)
		log.Println("Failed to find a start instruction (drawFromTop)")
		return
	}

	for i := 0; i < h.maxLines; i++ {
		if instr == nil {
			// Clear remaining lines.
			for j := i; j < h.maxLines; j++ {
				h.lines[j].kind = emptyKind
				h.lines[j].editable = false
			}
			break
		}
		if instr.Name != "" {
			h.lines[i].kind = symbolKind
			h.lines[i].editable = false
			h.lines[i].data = instr.Name
			i++
		}
		if i < h.maxLines {
			h.lines[i].kind = instrKind
			h.lines[i].editable = true
			h.lines[i].data = instr
		}
		if i+1 < h.maxLines {
			instr = h.project.FindNextInstruction(instr.Address)
		}
	}

	// Find the first editable line, and set cursor to the line.
	if h.cursor = h.findNextLine(0); h.cursor == -1 {
		log.Panicf("Invalid next cursor: %d", h.cursor)
	}

	h.updateBuffer()
}

func (h *handler) drawFromBottom(addr uint64) {
	instr := h.project.GetInstruction(addr)
	if instr == nil {
		h.popupEvents <- fmt.Sprintf("No Such Instruction (Address: 0x%x)", addr)
		log.Println("Failed to find a start instruction (drawFromBottom)")
		return
	}

	for i := h.maxLines - 1; i >= 0; i-- {
		if instr == nil {
			// Clear remaining lines.
			for j := i; j >= 0; j-- {
				h.lines[j].kind = emptyKind
				h.lines[j].editable = false
			}
			break
		}

		h.lines[i].kind = instrKind
		h.lines[i].editable = true
		h.lines[i].data = instr
		if instr.Name != "" && i > 0 {
			i--
			h.lines[i].kind = symbolKind
			h.lines[i].editable = false
			h.lines[i].data = instr.Name
		}
		if i != 0 {
			instr = h.project.FindPrevInstruction(instr.Address)
		}
	}

	// Find the first editable line, and set cursor to the line.
	if h.cursor = h.findNextPrevLine(h.maxLines - 1); h.cursor == -1 {
		log.Panicf("Invalid next cursor: %d", h.cursor)
	}

	h.updateBuffer()
}

/****************/
/* Cursor Moves *
/****************/

func (h *handler) findNextLine(start int) int {
	for i := start; i < h.maxLines; i++ {
		if h.lines[i].editable {
			return i
		}
	}
	return -1
}

func (h *handler) findNextPrevLine(start int) int {
	for i := start; i >= 0; i-- {
		if h.lines[i].editable {
			return i
		}
	}
	return -1
}

/* Move cursor using j/k or arrow keys */
func (h *handler) cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}
	if nextIdx := h.findNextPrevLine(h.cursor - 1); nextIdx <= 0 {
		curInstr := h.lines[h.cursor].data.(*binch.Instruction)
		if instr := h.project.FindPrevInstruction(curInstr.Address); instr != nil {
			h.drawFromTop(instr.Address)
		}
	} else {
		h.cursor = nextIdx
		h.updateBuffer()
	}
	return nil
}

func (h *handler) cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}
	if nextIdx := h.findNextLine(h.cursor + 1); nextIdx == -1 {
		lastInstr := h.lines[h.cursor].data.(*binch.Instruction)
		if instr := h.project.FindNextInstruction(lastInstr.Address); instr != nil {
			h.drawFromBottom(instr.Address)
		}
	} else {
		h.cursor = nextIdx
		h.updateBuffer()
	}
	return nil
}

/* Move cursor using ctrl+f/b */
func (h *handler) pageDown(g *gocui.Gui, v *gocui.View) error {
	if nextIdx := h.findNextPrevLine(h.maxLines - 1); nextIdx != -1 {
		lastInstr := h.lines[nextIdx].data.(*binch.Instruction)
		if nextInstr := h.project.FindNextInstruction(lastInstr.Address); nextInstr != nil {
			h.drawFromTop(nextInstr.Address)
		}
	} else {
		log.Panicln("pageDown fail")
	}
	return nil
}

func (h *handler) pageUp(g *gocui.Gui, v *gocui.View) error {
	if nextIdx := h.findNextLine(0); nextIdx != -1 {
		firstInstr := h.lines[nextIdx].data.(*binch.Instruction)
		if nextInstr := h.project.FindPrevInstruction(firstInstr.Address); nextInstr != nil {
			h.drawFromBottom(nextInstr.Address)
		}
	} else {
		log.Panicln("pageDown fail")
	}
	return nil
}

/* Move cursor using goto command */
func (h *handler) gotoAddr(g *gocui.Gui, v *gocui.View) error {
	exitGoto(g, v)
	line, _ := v.Line(0)
	if addr, err := strconv.ParseUint(line, 16, 64); err == nil {
		h.drawFromTop(addr)
	}
	return nil
}

func showGoto(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("goto", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Go to Address"
		v.Editable = true
		v.Wrap = true
		v.Clear()
		if _, err := setCurrentViewOnTop(g, "goto"); err != nil {
			return err
		}
	}
	return nil
}

func exitGoto(g *gocui.Gui, v *gocui.View) error {
	return exitView(g, "goto")
}
