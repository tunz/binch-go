package bcview

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"log"
	"time"
)

func setCurrentViewOnTop(g *gocui.Gui, name string) (*gocui.View, error) {
	if _, err := g.SetCurrentView(name); err != nil {
		return nil, err
	}
	return g.SetViewOnTop(name)
}

func exitView(g *gocui.Gui, name string) error {
	if err := g.DeleteView(name); err != nil {
		return err
	}
	if _, err := g.SetCurrentView("disasm"); err != nil {
		return err
	}
	return nil
}

func flush(g *gocui.Gui) {
	// Call Update with a dummy function for flushing screen.
	g.Update(func(g *gocui.Gui) error { return nil })
}

func popupLoop(g *gocui.Gui, v *gocui.View, popupEvents chan string) {
	for msg := range popupEvents {
		if msg != "" {
			v.Clear()
			log.Println(msg)
			fmt.Fprintf(v, "[*] %s", msg)
			flush(g)
			time.Sleep(time.Second * 2)
		}
		v.Clear()
		fmt.Fprintf(v, "q: quit | Enter: patch | d: delete | s: save | ctrl+z: undo")
		flush(g)
	}
}
