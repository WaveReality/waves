// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"fmt"
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/xyzcore"
)

// Scene is a Widget for managing the 3D Scene of the NetView
type Scene struct {
	xyzcore.Scene

	View *View
}

func (sw *Scene) Init() {
	sw.Scene.Init()
	sw.On(events.MouseDown, func(e events.Event) {
		sw.MouseDownEvent(e)
		sw.NeedsRender()
	})
	sw.On(events.Scroll, func(e events.Event) {
		pos := sw.Geom.ContentBBox.Min
		e.SetLocalOff(e.LocalOff().Add(pos))
		sw.SceneXYZ().MouseScrollEvent(e.(*events.MouseScroll))
		sw.NeedsRender()
	})
	sw.On(events.KeyChord, func(e events.Event) {
		sw.SceneXYZ().KeyChordEvent(e)
		sw.NeedsRender()
	})
	// sw.HandleSlideEvents() // TODO: need this
}

func (sw *Scene) MouseDownEvent(e events.Event) {
	pos := e.Pos().Sub(sw.Geom.ContentBBox.Min)
	panelNo, pt := sw.PlaneUnitAtPoint(pos)
	if panelNo < 0 {
		return
	}
	vw := sw.View
	vw.selCube = pt
	vw.UpdateView()
	e.SetHandled()
}

func (sw *Scene) WidgetTooltip(pos image.Point) (string, image.Point) {
	if pos == image.Pt(-1, -1) {
		return "_", image.Point{}
	}
	vw := sw.View
	lpos := pos.Sub(sw.Geom.ContentBBox.Min)

	panelNo, pt := sw.PlaneUnitAtPoint(lpos)
	if panelNo < 0 {
		return "", pos
	}
	cp := vw.Panels[panelNo]
	ctx := GetCtx(0)
	val := State.Value(int(pt.Z), int(pt.Y), int(pt.X), int(cp.Var.Int64()), int(ctx.CurState))

	tt := fmt.Sprintf("%d: %s %v=%g\n", panelNo, cp.Var, pt, val)
	return tt, pos
}

func (sw *Scene) PlaneUnitAtPoint(pos image.Point) (panelNo int, pt math32.Vector3i) {
	panelNo = -1
	sc := sw.SceneXYZ()
	plGpi := sc.ChildByName("Planes", 0)
	if plGpi == nil {
		return
	}
	_, plGp := xyz.AsNode(plGpi)
	vw := sw.View
	// sz, nsc := vw.planeScale()
	sz := vw.Size
	poff := math32.Vector3Scalar(0.5)
	poff.Y = -0.5
	for li, plgi := range plGp.Children {
		plg := plgi.(*xyz.Group)
		ploff := poff
		sp := float32(0.02)
		switch li {
		case 1:
			ploff.X = -sp
		case 2:
			ploff.Z = -0.5
		case 3:
			ploff.X = -sp
			ploff.Z = -0.5
		}
		lp := math32.Vec3(0, 0, 0).Sub(ploff)
		lp.Y = -lp.Y // reverse direction
		lo := plg.Child(0).(*PlaneObj)
		ray := lo.RayPick(pos)
		// plane is in XZ plane with norm pointing up in Y axis
		// offset is 0 in local coordinates
		plane := math32.Plane{Norm: math32.Vec3(0, 1, 0), Off: 0}
		ipt, ok := ray.IntersectPlane(plane)
		if !ok || ipt.Z > 0 { // Z > 0 means clicked "in front" of plane -- where labels are
			continue
		}
		pt.Set(int32(ipt.X), int32(-ipt.Z), 0)
		// fmt.Printf("\tpanel: %d coords: %v\n", li, pt)
		if pt.X < 0 || pt.Y < 0 || pt.X >= sz.X || pt.Y >= sz.Y {
			continue
		}
		panelNo = li
		pl := vw.Panels[li]
		pt = pt.Add(vw.Start).Add(pl.Offset)
		// fmt.Printf("*** selected panel: %d coords: %v\n", li, pt)
		break
	}
	return
}

// FormDialog opens a dialog in a new, separate window
// for viewing / editing the given struct object, in
// the context of the given ctx widget.
func FormDialog(ctx core.Widget, v any, title string) {
	d := core.NewBody(title)
	core.NewForm(d).SetStruct(v)
	if tb, ok := v.(core.ToolbarMaker); ok {
		d.AddTopBar(func(bar *core.Frame) {
			core.NewToolbar(bar).Maker(tb.MakeToolbar)
		})
	}
	d.RunWindowDialog(ctx)
}
