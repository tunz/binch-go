package bcview

import (
	"github.com/jroimartin/gocui"
	"github.com/tunz/binch-go/pkg/core"
	"github.com/tunz/binch-go/pkg/io"
	"log"
	"sync"
)

type lineKind int

const (
	instrKind  lineKind = 0
	symbolKind lineKind = 1
	emptyKind  lineKind = 2
	rawKind    lineKind = 3
)

type lineInfo struct {
	kind     lineKind
	editable bool
	data     interface{}
}

type handler struct {
	filename    string
	project     *binch.Project
	maxLines    int
	lines       []lineInfo
	cursor      int
	gui         *gocui.Gui
	instrEvents chan string
	byteEvents  chan string
	popupEvents chan string
	mux         sync.Mutex
}

func (h *handler) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	left := maxX / 8
	if v, err := g.SetView("disasm", left, 3, maxX-left, maxY-4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = h.filename

		_, h.maxLines = v.Size()
		if h.lines == nil {
			h.lines = make([]lineInfo, h.maxLines)
			h.drawFromTop(h.project.Entry())
		}

		if _, err := g.SetCurrentView("disasm"); err != nil {
			return err
		}
	}
	if v, err := g.SetView("helper", left, maxY-3, maxX-left, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		h.popupEvents = make(chan string, 20)
		go popupLoop(g, v, h.popupEvents)
		h.popupEvents <- ""
	}
	return nil
}

func (h *handler) saveFile(g *gocui.Gui, v *gocui.View) error {
	h.popupEvents <- "Saved"
	h.project.Save()
	return nil
}

func (h *handler) undo(g *gocui.Gui, v *gocui.View) error {
	log.Println("undo")
	h.project.Undo()
	h.redraw()
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func initKeybindings(g *gocui.Gui, h *handler) error {
	/* Global */
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	key2fn := map[string]map[interface{}]func(g *gocui.Gui, v *gocui.View) error{}

	/* Disasm */
	key2fn["disasm"] = map[interface{}]func(g *gocui.Gui, v *gocui.View) error{
		'k':                h.cursorUp,
		gocui.KeyArrowUp:   h.cursorUp,
		'j':                h.cursorDown,
		gocui.KeyArrowDown: h.cursorDown,
		gocui.KeyCtrlF:     h.pageDown,
		gocui.KeyCtrlB:     h.pageUp,
		'g':                showGoto,
		'q':                quit,
		gocui.KeyEnter:     h.showPatch,
		's':                h.saveFile,
		'd':                h.deleteInstr,
		gocui.KeyCtrlZ:     h.undo,
	}

	/* Goto */
	key2fn["goto"] = map[interface{}]func(g *gocui.Gui, v *gocui.View) error{
		gocui.KeyEsc:   exitGoto,
		gocui.KeyEnter: h.gotoAddr,
	}

	/* Patch */
	key2fn["patchByte"] = map[interface{}]func(g *gocui.Gui, v *gocui.View) error{
		gocui.KeyEsc:       h.exitPatch,
		gocui.KeyTab:       patchMoveFocus,
		gocui.KeyArrowDown: patchMoveFocus,
		gocui.KeyEnter:     h.patchByte,
	}

	key2fn["patchInstr"] = map[interface{}]func(g *gocui.Gui, v *gocui.View) error{
		gocui.KeyEsc:     h.exitPatch,
		gocui.KeyTab:     patchMoveFocus,
		gocui.KeyArrowUp: patchMoveFocus,
		gocui.KeyEnter:   h.patchInstr,
	}

	/* Help */

	for view, m := range key2fn {
		for key, fn := range m {
			if err := g.SetKeybinding(view, key, gocui.ModNone, fn); err != nil {
				return err
			}
		}
	}

	return nil
}

// Run starts up binch UI.
func Run(filename string, b *bcio.Binary) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	h := handler{
		filename: filename,
		project:  binch.MakeProject(b),
		maxLines: 0,
		lines:    nil,
		cursor:   0,
		gui:      g,
	}

	g.InputEsc = true
	g.ASCII = true
	g.SetManagerFunc(h.layout)

	if err := initKeybindings(g, &h); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
