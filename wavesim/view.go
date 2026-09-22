// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"fmt"
	"image/color"
	"reflect"
	"strconv"
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
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

// PanelView for what each panel in the View renders.
type PanelView struct {
	// Variable to display.
	Var enums.Enum

	// Select which state to view
	CurPrev CurPrev

	// Mode is how the state values are displayed for this panel.
	Mode ViewModes

	// Offset is an additional offset from the global Start,
	// enforced to be within the displayable size.
	Offset math32.Vector3i
}

// View is a Cogent Core Widget that provides a 3D view into state.
type View struct {
	core.Frame

	// Var determines the set of variables being used.
	// actual variable to view is in the PanelView.
	Var enums.Enum `set:"-"`

	// Pannels are the view settings per panel (4 max).
	Panels [4]PanelView

	// Starting front-left corner location within state.
	Start math32.Vector3i

	// Size of planes
	Size math32.Vector3i

	// Depth is the state dimension drawn going back into the screen.
	// The horizontal display axis is always state X and the height is the
	// value, so this is what picks the plane. [math32.Z] is the standard
	// orientation and the default: the X-Z plane, sliced at a Y level. The
	// alternative, [math32.Y], gives the X-Y plane sliced at a Z level, which
	// is only useful for a flat sim that has a single Z and so no X-Z plane to
	// look at. The dimension left over is the slice level, [View.SliceDim],
	// whose position [View.Start] holds along with the in-plane corner.
	Depth math32.Dims `set:"-"`

	// parameters for the list of variables to view
	VarSettings map[enums.Enum]*VarSettings

	// Settings are parameters controlling how the view is rendered
	Settings Settings

	// Counters are displayed at the bottom: time, etc.
	Counters string `set:"-" display:"-"`

	// color map for mapping values to colors -- set by name in Settings
	colorMap *colormap.Map

	// which panel are we currently updating
	curPanel int

	// current number of panels rendered -- if changes, do full rebuild.
	curNPanels int

	selCube math32.Vector3i

	sim      *Sim
	midFrame *core.Frame
	scene    *Scene
	// SceneDBG  *Scene
	counters  *core.Text
	varsFrame *core.Frame
	toolbar   *core.Toolbar
	viewbar   *core.Toolbar

	sync.Mutex
}

func (vw *View) Init() {
	vw.Frame.Init()
	vw.Settings.Defaults()
	vw.colorMap = colormap.AvailableMaps[string(vw.Settings.ColorMap)]
	for i := range 4 {
		vw.Panels[i].Mode = vw.Settings.Mode
		vw.Panels[i].Var = vw.Var
	}
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
			// vw.SceneDBG = w
			w.View = vw
			se := w.SceneXYZ()
			vw.ViewDefaults(se)
			planesGp := xyz.NewGroup(se)
			planesGp.Name = "Planes"
			se.SetCamera(strconv.Itoa(vw.Settings.Camera))
		})
		w.OnShow(func(e events.Event) {
			vw.RebuildView()
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

// SetVar sets the variable to view and updates the display,
// for given panel number. If panelNo is -1, then this sets
// the global default for all panels, and doesn't update display.
func (vw *View) SetVar(vr enums.Enum, panelNo int) {
	vw.Lock()
	if panelNo < 0 {
		vw.Var = vr
		for i := range 4 {
			vw.Panels[i].Var = vr
		}
		vw.VarsListUpdate()
		vw.Unlock()
		return
	}
	vw.Panels[panelNo].Var = vr
	if vw.varsFrame == nil {
		vw.Unlock()
		return
	}
	vw.varsFrame.Update()
	vw.Unlock()
	vw.toolbar.Update()
	vw.UpdateView()
}

// SetMode sets the display mode for given panel number.
// if panelNo < 0 then sets default for all panels.
func (vw *View) SetMode(mode ViewModes, panelNo int) {
	vw.Lock()
	if panelNo < 0 {
		vw.Settings.Mode = mode
		for i := range 4 {
			vw.Panels[i].Mode = mode
		}
		vw.Unlock()
		return
	}
	vw.Panels[panelNo].Mode = mode
	if vw.varsFrame == nil {
		vw.Unlock()
		return
	}
	vw.varsFrame.Update()
	vw.toolbar.Update()
	vw.UpdateView()
}

// SetCurPrev sets the current vs. previous state viewing
func (vw *View) SetCurPrev(curprv CurPrev, panelNo int) {
	vw.Panels[panelNo].CurPrev = curprv
	vw.UpdateView()
}

// SetVarMinMax sets the min and max range for given variable.
func (vw *View) SetVarMinMax(vr enums.Enum, mn, mx float32) {
	vp, err := vw.GetVarSettings(vr)
	if errors.Log(err) != nil {
		return
	}
	vp.Range.SetMin(mn)
	vp.Range.SetMax(mx)
	vw.UpdateView()
}

// SelectCamera selects the given pre-configured camera view,
// which have different angles. 1= top-down, 2 = head-on
func (vw *View) SelectCamera(camNo int) {
	se := vw.SceneXYZ()
	if se == nil {
		return
	}
	se.SetCamera(strconv.Itoa(camNo))
}

// GoUpdateView is the update call to make from another go routine
// it does the proper blocking to coordinate with GUI updates
// generated on the main GUI thread.
func (vw *View) GoUpdateView() {
	if !vw.IsVisible() {
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
	if !vw.IsVisible() {
		return
	}
	sw := vw.scene
	vw.UpdateImpl()
	sw.NeedsRender()
}

// RebuildView does a full update and then rebuild of view data
func (vw *View) RebuildView() {
	vw.UpdateView()
	vw.SceneXYZ().Rebuild()
}

// UpdateImpl does the guts of updating -- backend for Update or GoUpdate
func (vw *View) UpdateImpl() {
	vw.Lock()
	if vw.Settings.TrackParticle >= 0 {
		_, _, pos, _ := GetParticleAt(int32(vw.Settings.TrackParticle))
		if pos != (math32.Vector3i{}) {
			szh := vw.Size.DivScalar(2)
			vw.Start = pos.Sub(szh)
			// the slice dimension is not a corner: follow the particle exactly.
			vw.Start.SetDim(vw.SliceDim(), pos.Dim(vw.SliceDim()))
		}
	}
	for i := range 4 {
		if vw.Panels[i].Var == nil {
			fmt.Println("nil var", i)
			vw.Panels[i].Var = vw.Var
		}
	}
	vp, err := vw.GetVarSettings(vw.Panels[vw.curPanel].Var)
	if errors.Log(err) != nil {
		vw.Unlock()
		return
	}

	if false && (!vp.Range.FixMin || !vp.Range.FixMax) { // todo: not yet
		needUpdate := false
		// need to autoscale
		// min, max, ok := vw.Data.VarRange(vw.Var)
		min, max, ok := float32(-1), float32(1), true
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
				tb := vw.toolbar
				tb.UpdateTree()
				tb.NeedsRender()
			}
		}
	}

	vw.SetCounters(vw.Counters)
	vw.Unlock()
	vw.UpdatePlanes()
}

func (vw *View) SceneXYZ() *xyz.Scene {
	if vw.scene == nil {
		return nil
	}
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

func (vw *View) GetVarSettings(vr enums.Enum) (*VarSettings, error) {
	vp, ok := vw.VarSettings[vr]
	if !ok {
		return nil, fmt.Errorf("Variable: %v settings not found", vw.Var)
	}
	return vp, nil
}

func (vw *View) GetVarSettingsPanel(panelNo int) (*VarSettings, error) {
	vp, ok := vw.VarSettings[vw.Panels[panelNo].Var]
	if !ok {
		return nil, fmt.Errorf("Variable: %v settings not found", vw.Var)
	}
	return vp, nil
}

// VarsListUpdate updates the list of network variables
func (vw *View) VarsListUpdate() {
	if reflectx.IsNil(reflect.ValueOf(vw.Var)) {
		return
	}
	vals := vw.Var.Values()
	if len(vals) == len(vw.VarSettings) {
		return
	}
	vw.VarSettings = make(map[enums.Enum]*VarSettings, len(vals))
	for _, v := range vals {
		vp := &VarSettings{Var: v}
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
			s.Direction = styles.Column
			s.Grow.Set(0, 1)
			s.Min.X.Em(10)
			s.Overflow.Y = styles.OverflowAuto
		})
		vals := vw.Var.Values()
		tree.AddChildAt(w, "curprv", func(w *core.Switch) {
			w.SetText("Current").SetChecked(true).
				SetTooltip("Selects whether to show the current or previous state values")
			w.OnChange(func(e events.Event) {
				cp := Current
				if !w.IsChecked() {
					cp = Previous
				}
				vw.SetCurPrev(cp, vw.curPanel)
			})
			w.Updater(func() {
				if vw.Panels[vw.curPanel].CurPrev == Current {
					w.SetText("Current").SetChecked(true)
				} else {
					w.SetText("Previous").SetChecked(false)
				}
			})
		})
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
					vw.SetVar(v, vw.curPanel)
				})
				w.Updater(func() {
					w.SetSelected(v == vw.Panels[vw.curPanel].Var)
				})
			})
		}
	})
}

// ViewDefaults are the default 3D view params
func (vw *View) ViewDefaults(se *xyz.Scene) {
	se.Camera.Near = 0.1

	se.Camera.Pose.Pos.Set(0, 1.9, 2.7)
	se.Camera.LookAt(math32.Vec3(0, 0.2, -.8), math32.Vec3(0, 1, 0))
	se.SaveCamera("2")

	se.Camera.Pose.Pos.Set(0, 2.7, 1.2)
	se.Camera.LookAt(math32.Vec3(0, 0.2, -.8), math32.Vec3(0, 1, 0))
	se.SaveCamera("1")
	se.SaveCamera("default")

	vw.Styler(func(s *styles.Style) {
		se.Background = colors.Scheme.Surface
	})
	xyz.NewAmbient(se, "ambient", 0.1, xyz.DirectSun)
	xyz.NewDirectional(se, "directional", 0.5, xyz.DirectSun).Pos.Set(0, 2, 5)
	xyz.NewPoint(se, "point", .2, xyz.DirectSun).Pos.Set(0, 2, -5)
}

var NilColor = color.RGBA{0x20, 0x20, 0x20, 0x40}

// ValColor returns the scaled value, max value, and color representation
// for given raw value.
func (vw *View) ValColor(raw float32, panelNo int) (scaled, maxval float32, clr color.RGBA) {
	vp, err := vw.GetVarSettings(vw.Panels[panelNo].Var)
	var clp, norm, op float32
	if err != nil {
		maxval = 1
		clp = math32.Clamp(raw, -maxval, maxval)
		norm = 0.5 * (clp + 1)
		scaled = float32(2*norm - 1)
		op = (vw.Settings.ZeroAlpha + (1-vw.Settings.ZeroAlpha)*math32.Abs(scaled))
	} else {
		maxval = max(math32.Abs(vp.Range.Max), math32.Abs(vp.Range.Min))
		clp = vp.Range.ClampValue(raw)
		norm = vp.Range.NormValue(clp)
		if vp.ZeroCtr {
			scaled = float32(2*norm - 1)
			op = (vw.Settings.ZeroAlpha + (1-vw.Settings.ZeroAlpha)*math32.Abs(scaled))
		} else {
			scaled = float32(norm)
			op = (vw.Settings.ZeroAlpha + (1-vw.Settings.ZeroAlpha)*0.8) // no meaningful alpha -- just set at 80\%
		}
	}
	clr = colors.WithAF32(vw.colorMap.Map(norm), op)
	return
}

func (vw *View) planeName(no int) string {
	return "plane_" + strconv.Itoa(no)
}

func (vw *View) Planes() *xyz.Group {
	se := vw.SceneXYZ()
	lgpi := se.ChildByName("Planes", 0)
	if lgpi == nil {
		return nil
	}
	return lgpi.(*xyz.Group)
}

// PlaneAtNumber returns the xyz.Group that represents given plane number.
// nil if not found.
func (vw *View) PlaneAtNumber(no int) *xyz.Group {
	plgp := vw.Planes()
	pl := plgp.ChildByName(vw.planeName(no), 0)
	if pl == nil {
		return nil
	}
	return pl.(*xyz.Group)
}

// SliceDim returns the state dimension held fixed by the current view: the
// one that is neither horizontal (always X) nor [View.Depth].
func (vw *View) SliceDim() math32.Dims {
	if vw.Depth == math32.Z {
		return math32.Y
	}
	return math32.Z
}

// DepthSize returns the number of state cells drawn going back into the screen.
func (vw *View) DepthSize() int32 {
	return vw.Size.Dim(vw.Depth)
}

// SetDepth sets the state dimension drawn going back into the screen, which
// selects the display plane, and re-centers the slice level on the dimension
// that is now off screen. Only [math32.Y] and [math32.Z] are meaningful:
// the horizontal axis is always state X.
func (vw *View) SetDepth(dim math32.Dims) {
	if dim != math32.Y && dim != math32.Z {
		return
	}
	vw.Depth = dim
	fs := GetCtx(0).SizeFull()
	sd := vw.SliceDim()
	vw.Start.SetDim(dim, 1)
	vw.Start.SetDim(sd, max(fs.Dim(sd)/2, 1))
	// the depth extent changes, so the mesh has to be rebuilt, not just redrawn
	vw.RebuildView()
}

// DisplayVector maps a state-space vector into display space, where X is
// across, Y is up and Z goes back into the screen. In an X-Z view the two
// already agree; in an X-Y view the state Y and Z components swap, so that a
// vector always points along the axis its own dimension is drawn on.
func (vw *View) DisplayVector(v math32.Vector3) math32.Vector3 {
	if vw.Depth == math32.Z {
		return v
	}
	return math32.Vec3(v.X, v.Z, v.Y)
}

// StateCoord returns the state coordinate sampled by the display cell xi
// across and di back into the screen, given the already offset start corner.
func (vw *View) StateCoord(st math32.Vector3i, xi, di int32) math32.Vector3i {
	c := st
	c.X = st.X + xi
	c.SetDim(vw.Depth, st.Dim(vw.Depth)+di)
	return c
}

func (vw *View) ZoomInSize(n int32) {
	if vw.Size.X <= 4 {
		return
	}
	dd := vw.Depth
	n = min(n, vw.Size.X-4)
	vw.Size.X -= 2 * n
	vw.Start.X += n
	if vw.Size.Dim(dd) > 4 {
		vw.Size.SetDim(dd, vw.Size.Dim(dd)-2*n)
		vw.Start.SetDim(dd, vw.Start.Dim(dd)+n)
	}
	vw.UpdateView()
}

func (vw *View) ZoomOutSize(n int32) {
	ctx := GetCtx(0)
	sz := ctx.Size.V()
	fs := ctx.SizeFull()
	dd := vw.Depth
	vw.Size.X += 2 * n
	if vw.Start.X > 1 {
		vw.Start.X -= n
	}
	if vw.Size.X >= sz.X {
		vw.Size.X = sz.X
	}
	if vw.Start.X+vw.Size.X >= fs.X {
		vw.Start.X = (fs.X - 1) - vw.Size.X
	}
	if sz.Dim(dd) > 1 {
		vw.Size.SetDim(dd, vw.Size.Dim(dd)+2*n)
		if vw.Start.Dim(dd) > 1 {
			vw.Start.SetDim(dd, vw.Start.Dim(dd)-n)
		}
		if vw.Size.Dim(dd) >= sz.Dim(dd) {
			vw.Size.SetDim(dd, sz.Dim(dd))
		}
		if vw.Start.Dim(dd)+vw.Size.Dim(dd) >= fs.Dim(dd) {
			vw.Start.SetDim(dd, (fs.Dim(dd)-1)-vw.Size.Dim(dd))
		}
	}
	vw.UpdateView()
}

// MoveSlice moves the slice plane by n along [View.SliceDim]: the one state
// dimension that is not on screen.
func (vw *View) MoveSlice(n int32) {
	sd := vw.SliceDim()
	sz := GetCtx(0).Size.V()
	p := vw.Start.Dim(sd) + n
	p = max(p, 0)
	p = min(p, sz.Dim(sd)+1)
	vw.Start.SetDim(sd, p)
	vw.UpdateView()
}

// MoveStart moves the displayed region within the slice plane, by mv.X across
// and mv.Y back into the screen. Use [View.MoveSlice] to change which slice.
func (vw *View) MoveStart(mv math32.Vector3i) {
	fs := GetCtx(0).SizeFull()
	dd := vw.Depth
	vw.Start.X += mv.X
	vw.Start.SetDim(dd, vw.Start.Dim(dd)+mv.Y)
	if vw.Start.X < 0 {
		vw.Start.X = 0
	}
	if vw.Start.Dim(dd) < 0 {
		vw.Start.SetDim(dd, 0)
	}
	if vw.Start.X+vw.Size.X >= fs.X {
		vw.Start.X = (fs.X - 1) - vw.Size.X
	}
	if vw.Start.Dim(dd)+vw.Size.Dim(dd) >= fs.Dim(dd) {
		vw.Start.SetDim(dd, (fs.Dim(dd)-1)-vw.Size.Dim(dd))
	}
	vw.UpdateView()
}
