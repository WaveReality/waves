// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"image/color"
	"log"
	"reflect"
	"sync"
	"time"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/colormap"
	"cogentcore.org/core/core"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
)

// View is a Cogent Core Widget that provides a 3D view into state.
type View struct {
	core.Frame

	// Var is the current variable that we're viewing
	Var enums.Enum

	// color map for mapping values to colors -- set by name in Settings
	ColorMap *colormap.Map

	// parameters for the list of variables to view
	VarSettings map[enums.Enum]*VarSettings

	// Settings are parameters controlling how the view is rendered
	Settings Settings

	// Counters are displayed at the bottom: time, etc.
	Counters string

	// current var params -- only valid during Update of display
	curVarSettings *VarSettings

	midFrame  *core.Frame
	scene     *Scene
	counters  *core.Text
	varsFrame *core.Frame
	toolbar   *core.Toolbar
	viewbar   *core.Toolbar

	sync.Mutex
}

func (vw *View) Init() {
	vw.Frame.Init()
	vw.Settings.Defaults()
	vw.ColorMap = colormap.AvailableMaps[string(vw.Settings.ColorMap)]
	vw.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	tree.AddChildAt(vw, "tbar", func(w *core.Toolbar) {
		vw.toolbar = w
		w.Styler(func(s *styles.Style) {
			s.Wrap = true
		})
		w.Maker(vw.MakeToolbar)
	})
	tree.AddChildAt(vw, "midframe", func(w *core.Frame) {
		vw.midFrame = w
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Grow.Set(1, 1)
		})
		vw.makeVars(w)
		tree.AddChildAt(w, "scene", func(w *Scene) {
			vw.scene = w
			w.View = vw
			se := w.SceneXYZ()
			vw.ViewDefaults(se)
			planesGp := xyz.NewGroup(se)
			planesGp.Name = "Planes"
		})
		w.OnShow(func(e events.Event) {
			vw.Current()
		})
	})
	tree.AddChildAt(vw, "counters", func(w *core.Text) {
		vw.counters = w
		w.SetText("Counters: ").
			Styler(func(s *styles.Style) {
				s.Min.X.Pw(90)
			})
		w.Updater(func() {
			if w.Text != vw.Counters && vw.Counters != "" {
				w.SetText(vw.Counters)
			}
		})
	})
	tree.AddChildAt(vw, "vbar", func(w *core.Toolbar) {
		vw.viewbar = w
		w.Styler(func(s *styles.Style) {
			s.Wrap = true
		})
		w.Maker(vw.MakeViewbar)
	})
}

// SetVar sets the variable to view and updates the display
func (vw *View) SetVar(vr string) {
	vw.Lock()
	vw.Var = vr
	vw.varsFrame.Update()
	vw.Unlock()
	vw.toolbar.Update()
	vw.UpdateView()
}

// HasPlanes returns true if network has any layers -- else no display
func (vw *View) HasPlanes() bool {
	if vw.Net == nil || vw.Net.NumPlanes() == 0 {
		return false
	}
	return true
}

// GoUpdateView is the update call to make from another go routine
// it does the proper blocking to coordinate with GUI updates
// generated on the main GUI thread.
func (vw *View) GoUpdateView() {
	if !vw.IsVisible() || !vw.HasPlanes() {
		return
	}
	sw := vw.scene
	sw.Scene.AsyncLock()
	vw.UpdateImpl()
	sw.NeedsRender()
	sw.Scene.AsyncUnlock()
	if core.TheApp.Platform() == system.Web {
		time.Sleep(time.Millisecond) // critical to prevent hanging!
	}
}

// UpdateView updates the display based on last recorded state of network.
func (vw *View) UpdateView() {
	if !vw.IsVisible() || !vw.HasPlanes() {
		return
	}
	sw := vw.scene
	vw.UpdateImpl()
	sw.NeedsRender()
}

// UpdateImpl does the guts of updating -- backend for Update or GoUpdate
func (vw *View) UpdateImpl() {
	vw.Lock()
	vp, ok := vw.VarSettings[vw.Var]
	if !ok {
		vw.Unlock()
		log.Printf("View: %v variable: %v not found\n", vw.Name, vw.Var)
		return
	}
	vw.curVarSettings = vp

	if !vp.Range.FixMin || !vp.Range.FixMax {
		needUpdate := false
		// need to autoscale
		min, max, ok := vw.Data.VarRange(vw.Var)
		if ok {
			vp.MinMax.Set(min, max)
			if !vp.Range.FixMin {
				nmin := float32(minmax.NiceRoundNumber(float64(min), true)) // true = below
				if vp.Range.Min != nmin {
					vp.Range.Min = nmin
					needUpdate = true
				}
			}
			if !vp.Range.FixMax {
				nmax := float32(minmax.NiceRoundNumber(float64(max), false)) // false = above
				if vp.Range.Max != nmax {
					vp.Range.Max = nmax
					needUpdate = true
				}
			}
			if vp.ZeroCtr && !vp.Range.FixMin && !vp.Range.FixMax {
				bmax := math32.Max(math32.Abs(vp.Range.Max), math32.Abs(vp.Range.Min))
				if !needUpdate {
					if vp.Range.Max != bmax || vp.Range.Min != -bmax {
						needUpdate = true
					}
				}
				vp.Range.Max = bmax
				vp.Range.Min = -bmax
			}
			if needUpdate {
				tb := vw.Toolbar()
				tb.UpdateTree()
				tb.NeedsRender()
			}
		}
	}

	vw.SetCounters(vw.Data.CounterRec(vw.RecNo))
	vw.Unlock()
	vw.UpdatePlanes()
}

func (vw *View) SceneXYZ() *xyz.Scene {
	return vw.scene.SceneXYZ()
}

// SetCounters sets the counters widget view display at bottom of netview
func (vw *View) SetCounters(ctrs string) {
	if ctrs == "" {
		return
	}
	vw.Counters = ctrs
	ct := vw.counters
	ct.UpdateWidget().NeedsRender()
}

// VarsListUpdate updates the list of network variables
func (vw *View) VarsListUpdate() {
	if reflectx.IsNil(reflect.ValueOf(vw.Var)) {
		return
	}
	vals := vr.Values()
	if len(vals) == len(vw.VarSettings) {
		return
	}
	vw.VarSettings = make(map[enums.Enum]*VarSettings, len(vw.Vars))
	for _, v := range vals {
		vp := &VarSettings{Var: v.String()}
		vp.Defaults()
		if vsr, ok := v.(VarSettinger); ok {
			vsr.SetVarSettings(vp)
		}
		vw.VarSettings[v] = vp
	}
}

// makeVars configures the variables
func (vw *View) makeVars(frame *core.Frame) {
	vw.VarsListUpdate()
	if reflectx.IsNil(reflect.ValueOf(vw.Var)) {
		return
	}
	tree.AddChildAt(frame, "vars", func(w *core.Frame) {
		vw.varsFrame = w
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(0, 1)
			s.Overflow.Y = styles.OverflowAuto
		})
		vals := vw.Var.Values()
		for _, v := range vals {
			vn := v.String()
			doc := v.Desc()
			tree.AddChildAt(w, vn, func(w *core.Button) {
				w.SetText(vn)
				if doc != "" {
					w.Tooltip = v.String() + ": " + doc
				}
				w.SetType(core.ButtonAction)
				w.OnClick(func(e events.Event) {
					vw.SetVar(v)
				})
				w.Updater(func() {
					w.SetSelected(v == vw.Var)
				})
			})
		}
	})
}

// ViewDefaults are the default 3D view params
func (vw *View) ViewDefaults(se *xyz.Scene) {
	se.Camera.Pose.Pos.Set(0, 1.5, 2.5) // more "top down" view shows more of layers
	// 	vs.Camera.Pose.Pos.Set(0, 1, 2.75) // more "head on" for larger / deeper networks
	se.Camera.Near = 0.1
	se.Camera.LookAt(math32.Vec3(0, 0, 0), math32.Vec3(0, 1, 0))
	vw.Styler(func(s *styles.Style) {
		se.Background = colors.Scheme.Surface
	})
	xyz.NewAmbient(se, "ambient", 0.1, xyz.DirectSun)
	xyz.NewDirectional(se, "directional", 0.5, xyz.DirectSun).Pos.Set(0, 2, 5)
	xyz.NewPoint(se, "point", .2, xyz.DirectSun).Pos.Set(0, 2, -5)
}

// StateValue returns the raw value, scaled value, and color representation
// for given state index. scaled is in range -1..1
func (vw *View) StateValue(idx ...int) (raw, scaled float32, clr color.RGBA) {
	raw := State.Value(idx...)
	scaled, clr = vw.ValColor(lay, idx1d, raw)
	return
}

var NilColor = color.RGBA{0x20, 0x20, 0x20, 0x40}

// ValColor returns the raw value, scaled value, and color representation
// for given raw value
func (vw *View) ValColor(raw float32) (scaled float32, clr color.RGBA) {
	if vw.curVarSettings == nil || vw.curVarSettings.Var != vw.Var {
		ok := false
		vw.curVarSettings, ok = vw.VarSettings[vw.Var]
		if !ok {
			return
		}
	}
	clp := vw.curVarSettings.Range.ClampValue(raw)
	norm := vw.curVarSettings.Range.NormValue(clp)
	var op float32
	if vw.curVarSettings.ZeroCtr {
		scaled = float32(2*norm - 1)
		op = (vw.Settings.ZeroAlpha + (1-vw.Settings.ZeroAlpha)*math32.Abs(scaled))
	} else {
		scaled = float32(norm)
		op = (vw.Settings.ZeroAlpha + (1-vw.Settings.ZeroAlpha)*0.8) // no meaningful alpha -- just set at 80\%
	}
	clr = colors.WithAF32(vw.ColorMap.Map(norm), op)
	return
}

func (vw *View) Planes() *xyz.Group {
	se := vw.SceneXYZ()
	lgpi := se.ChildByName("Planes", 0)
	if lgpi == nil {
		return nil
	}
	return lgpi.(*xyz.Group)
}

// LabelByName returns given Text2D label (see ConfigLabels).
// nil if not found.
// func (vw *View) LabelByName(lab string) *xyz.Text2D {
// 	lgp := vw.Labels()
// 	txt := lgp.ChildByName(lab, 0)
// 	if txt == nil {
// 		return nil
// 	}
// 	return txt.(*xyz.Text2D)
// }

// LayerByName returns the xyz.Group that represents layer of given name.
// nil if not found.
func (vw *View) LayerByName(lay string) *xyz.Group {
	lgp := vw.Planes()
	ly := lgp.ChildByName(lay, 0)
	if ly == nil {
		return nil
	}
	return ly.(*xyz.Group)
}
