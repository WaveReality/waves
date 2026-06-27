// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"cogentcore.org/core/xyz"
)

// PlaneObj is the Plane 3D object within the View
type PlaneObj struct {
	xyz.Solid

	panelNo int
	view    *View
}

// UpdatePlanes updates the planes display with any structural or
// current data changes. Very fast if no structural changes.
func (vw *View) UpdatePlanes() {
	sw := vw.scene
	se := sw.SceneXYZ()

	// if vw.Net == nil || vw.Net.NumLayers() == 0 {
	// 	se.DeleteChildren()
	// 	se.Meshes.Reset()
	// 	return
	// }
	if vw.NeedsRebuild() {
		se.Background = colors.Scheme.Surface
	}
	plGp := se.ChildByName("Planes", 0).(*xyz.Group)

	plConfig := tree.TypePlan{}
	for li := range vw.NPanels {
		plConfig.Add(types.For[xyz.Group](), vw.planeName(li))
	}

	tree.Update(plGp, plConfig)
	// for li := range plGp.Children {
	// 	lmesh := errors.Log1(se.MeshByName(vw.planeName(li)))
	// 	se.SetMesh(lmesh) // does update
	// }
	// return

	gpConfig := tree.TypePlan{}
	gpConfig.Add(types.For[PlaneObj](), "plane")
	gpConfig.Add(types.For[xyz.Text2D](), "name")

	sz := vw.Size
	// if vw.NPanels > 1 {
	// 	sz.X *= 2
	// }
	// if vw.NPanels > 2 {
	// 	sz.Z *= 2
	// }

	nsc := math32.Vec3(2/float32(sz.X), 2/float32(sz.Y), 2/float32(sz.Z))
	ht := vw.Settings.Height
	// szc := max(nsc.X, nsc.Y)
	poff := math32.Vector3Scalar(0.5)
	poff.Y = -0.5
	poff.X = 1
	for li, plgi := range plGp.Children {
		plnm := vw.planeName(li)
		plmesh, _ := se.MeshByName(plnm)
		if plmesh == nil {
			plmesh = NewPlaneMesh(se, vw, li)
		} else {
			plmesh.(*PlaneMesh).panelNo = li
		}
		se.SetMesh(plmesh)
		plg := plgi.(*xyz.Group)
		gpConfig[1].Name = plnm // text2d textures use obj name, so must be unique
		tree.Update(plg, gpConfig)
		lp := math32.Vec3(0, 0, 0).Sub(poff)
		lp.Y = -lp.Y // reverse direction
		// lp = lp.Sub(nmin).Mul(nsc).Sub(poff)
		plg.Pose.Pos.Set(lp.X, lp.Z, lp.Y)

		plo := plg.Child(0).(*PlaneObj)
		plo.Defaults()
		plo.panelNo = li
		plo.view = vw
		plo.SetMeshName(plnm)
		plo.Material.Color = colors.FromRGB(255, 100, 255)
		plo.Material.Reflective = 8
		plo.Material.Bright = 8
		plo.Material.Shiny = 30
		plo.Pose.Scale.Set(nsc.X, ht, nsc.Y)
		// note: would actually be better to NOT cull back so you can view underneath
		// but then the front and back fight against each other, causing flickering

		txt := plg.Child(1).(*xyz.Text2D)
		txt.Defaults()
		txt.SetText("Var: " + vw.Vars[li].String())
		// this doesn't help updating, neither does Rebuild!
		// txt.RenderText()
		txt.Pose.Scale = math32.Vector3Scalar(vw.Settings.LabelSize)
		txt.Styles.Background = colors.Uniform(colors.Transparent)
		txt.Styles.Text.Align = text.Start
		txt.Styles.Text.AlignV = text.Start
	}
	sw.NeedsRender()
}
