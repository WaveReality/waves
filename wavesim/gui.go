// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"io/fs"
	"sync"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	_ "cogentcore.org/lab/gosl/slbool/slboolcore" // include to get gui views
	"cogentcore.org/lab/lab"
)

// GUI manages all standard elements of a simulation Graphical User Interface
type GUI struct {
	lab.Browser

	// Active is true if the GUI is configured and running
	Active bool `display:"-"`

	// SimForm displays the Sim object fields in the left panel.
	SimForm *core.Form `display:"-"`

	// Body is the entire content of the sim window.
	Body *core.Body `display:"-"`

	// isRunning is true if sim is running.
	isRunning bool

	// stopNow can be set via SetStopNow method under mutex protection
	// to signal the current sim to stop running.
	// It is not used directly in the looper-based control logic, which has
	// its own direct Stop function, but it is set there in case there are
	// other processes that are looking at this flag.
	stopNow bool

	// view if created.
	View *View

	// the sim
	sim *Sim

	runMu sync.Mutex
}

// UpdateWindow triggers an update on window body,
// to be called from within the normal event processing loop.
// See GoUpdateWindow for version to call from separate goroutine.
func (gui *GUI) UpdateWindow() {
	if gui.Toolbar != nil {
		gui.Toolbar.Restyle()
	}
	gui.SimForm.Update()
	gui.Splits.NeedsRender()
	// todo: could update other stuff but not really necessary
}

// GoUpdateWindow triggers an update on window body,
// for calling from a separate goroutine.
func (gui *GUI) GoUpdateWindow() {
	gui.Splits.Scene.AsyncLock()
	defer gui.Splits.Scene.AsyncUnlock()
	gui.UpdateWindow()
}

// StartRun should be called whenever a process starts running.
// It sets stopNow = false and isRunning = true under a mutex.
func (gui *GUI) StartRun() {
	gui.runMu.Lock()
	gui.stopNow = false
	gui.isRunning = true
	gui.runMu.Unlock()
}

// IsRunning returns the state of the isRunning flag, under a mutex.
func (gui *GUI) IsRunning() bool {
	gui.runMu.Lock()
	defer gui.runMu.Unlock()
	return gui.isRunning
}

// StopNow returns the state of the stopNow flag, under a mutex.
func (gui *GUI) StopNow() bool {
	gui.runMu.Lock()
	defer gui.runMu.Unlock()
	return gui.stopNow
}

// SetStopNow sets the stopNow flag to true, under a mutex.
func (gui *GUI) SetStopNow() {
	gui.runMu.Lock()
	gui.stopNow = true
	gui.runMu.Unlock()
}

// Stopped is called when a run method stops running,
// from a separate goroutine (do not call from main event loop).
// Turns off the isRunning flag, calls OnStop,
// and calls GoUpdateWindow to update window state.
func (gui *GUI) Stopped() {
	gui.runMu.Lock()
	gui.isRunning = false
	gui.stopNow = true // in case anyone else is looking
	gui.runMu.Unlock()
	// if gui.OnStop != nil {
	// 	gui.OnStop(mode, level)
	// }
	gui.GoUpdateWindow()
}

// NewGUIBody returns a new GUI, with an initialized Body by calling [gui.MakeBody].
func NewGUIBody(b tree.Node, sim *Sim, fsroot fs.FS, appname, title, about string) *GUI {
	gu := &GUI{}
	gu.MakeBody(b, sim, fsroot, appname, title, about)
	return gu
}

// MakeBody initializes default Body with a top-level [core.Splits] containing
// a [core.Form] editor of the given sim object, and a filetree for the data filesystem
// rooted at fsroot, and with given app name, title, and about information.
// The first arg is an optional existing [core.Body] to make into: if nil then
// a new body is made first.
func (gui *GUI) MakeBody(b tree.Node, sim *Sim, fsroot fs.FS, appname, title, about string) {
	core.NoSentenceCaseFor = append(core.NoSentenceCaseFor, "github.com/WaveReality")
	if b == nil {
		gui.Body = core.NewBody(appname).SetTitle(title)
		b = gui.Body
		core.AppAbout = about
	} else {
		gui.Toolbar = core.NewToolbar(b)
	}
	split := core.NewSplits(b)
	split.Styler(func(s *styles.Style) {
		s.Min.Y.Em(40)
	})
	split.Name = "split"
	gui.sim = sim
	gui.Splits = split
	gui.SimForm = core.NewForm(split).SetStruct(sim)
	gui.SimForm.OnChange(func(e events.Event) {
		sim.UpdateUnits()
	})
	gui.SimForm.Name = "sim-form"
	if gui.Body != nil {
		gui.Body.AddTopBar(func(bar *core.Frame) {
			gui.Toolbar = core.NewToolbar(bar)
			gui.Toolbar.Maker(gui.MakeToolbar)
		})
	} else {
		gui.Toolbar.Maker(gui.MakeToolbar)
	}
	fform := core.NewFrame(split)
	fform.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
		s.Grow.Set(1, 1)
	})
	gui.Files = lab.NewDataTree(fform)
	tabs := lab.NewTabs(split)
	gui.Tabs = tabs
	lab.Lab = tabs
	tabs.Name = "tabs"
	gui.FS = fsroot
	gui.DataRoot = "Root"
	gui.UpdateFiles()
	gui.Files.Tabber = tabs

	split.SetTiles(core.TileSplit, core.TileSpan)
	split.SetSplits(.2, .5, .8)
}

// AddView adds View in tab with given name
func (gui *GUI) AddView(tabName string) *View {
	nv := lab.NewTab(gui.Tabs, tabName, func(tab *core.Frame) *View {
		nv := NewView(tab)
		gui.View = nv
		// gui.NetViews = append(gui.NetViews, nv)
		return nv
	})
	return nv
}

// FinalizeGUI wraps the end functionality of the GUI
func (gui *GUI) FinalizeGUI(closePrompt bool) {
	gui.Active = true
}

func (gui *GUI) MakeToolbar(p *tree.Plan) {
	tree.AddAt(p, "Init", func(w *core.Button) {
		w.SetText("Init").SetIcon(icons.Update).
			SetTooltip("Run simulation until Stop").OnClick(func(e events.Event) {
			if !gui.IsRunning() {
				gui.sim.Init()
				go gui.sim.UpdateView()
			}
		})
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(!gui.IsRunning()) })
	})
	tree.AddAt(p, "Run", func(w *core.Button) {
		w.SetText("Run").SetIcon(icons.PlayArrow).
			SetTooltip("Run simulation until Stop").OnClick(func(e events.Event) {
			tb := gui.Toolbar
			if !gui.IsRunning() {
				gui.StartRun()
				tb.Restyle()
				go gui.sim.Run()
			}
		})
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(!gui.IsRunning()) })
	})
	tree.AddAt(p, "Stop", func(w *core.Button) {
		w.SetText("Stop").SetIcon(icons.Stop).
			SetTooltip("Stop a running simulation").OnClick(func(e events.Event) {
			if gui.IsRunning() {
				gui.SetStopNow()
			}
		})
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(gui.IsRunning()) })
	})
	tree.AddAt(p, "Step 1", func(w *core.Button) {
		w.SetText("Step 1").SetIcon(icons.SkipNext).
			SetTooltip("Step forward 1 time step").OnClick(func(e events.Event) {
			tb := gui.Toolbar
			if !gui.IsRunning() {
				gui.StartRun()
				tb.Restyle()
				go gui.sim.StepN(1)
			}
		})
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(!gui.IsRunning()) })
	})
	tree.AddAt(p, "Step 10", func(w *core.Button) {
		w.SetText("Step 10").SetIcon(icons.SkipNext).
			SetTooltip("Step forward 10 time steps").OnClick(func(e events.Event) {
			tb := gui.Toolbar
			if !gui.IsRunning() {
				gui.StartRun()
				tb.Restyle()
				go gui.sim.StepN(10)
			}
		})
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(!gui.IsRunning()) })
	})
	tree.AddAt(p, "Step 100", func(w *core.Button) {
		w.SetText("Step 100").SetIcon(icons.SkipNext).
			SetTooltip("Step forward 10 time steps").OnClick(func(e events.Event) {
			tb := gui.Toolbar
			if !gui.IsRunning() {
				gui.StartRun()
				tb.Restyle()
				go gui.sim.StepN(100)
			}
		})
		w.FirstStyler(func(s *styles.Style) { s.SetEnabled(!gui.IsRunning()) })
	})
}
