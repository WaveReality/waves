// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/xyz"
)

// PlaneMesh is a xyz.Mesh that represents an X-Y plane through the state.
// It is dynamically updated using theUpdate method which only resets the
// essential Vertex elements. The geometry is literal in the size:
// 0,0,0 lower-left corner and increasing X,Z in display for the X,Y plane.
// Display applies an overall scaling to make it fit within the larger view.
type PlaneMesh struct {
	xyz.MeshBase

	view *View

	// our panel number
	panelNo int
}

// NewPlaneMesh adds PlaneMesh mesh to given scene for given layer
func NewPlaneMesh(sc *xyz.Scene, view *View, panel int) *PlaneMesh {
	pm := &PlaneMesh{view: view, panelNo: panel}
	pm.Name = view.planeName(panel)
	sc.SetMesh(pm)
	return pm
}

func (pm *PlaneMesh) MeshSize() (nVtx, nIndex int, hasColor bool) {
	pm.Transparent = true
	pm.HasColor = true

	nz := int(pm.view.Size.Y)
	nx := int(pm.view.Size.X)
	segs := 1

	mode := pm.view.Panels[pm.panelNo].Mode

	nper := 1
	if mode == Bars {
		nper = 5
	}
	vtxSz, idxSz := shape.PlaneN(segs, segs)
	nVtx = vtxSz * nper * nz * nx
	nIndex = idxSz * nper * nz * nx

	pm.NumVertex, pm.NumIndex = nVtx, nIndex
	return pm.NumVertex, pm.NumIndex, pm.HasColor
}

// MinUnitHeight ensures that there is always at least some dimensionality
// to the unit cubes -- affects transparency rendering etc
var MinUnitHeight = float32(1.0e-6)

func (pm *PlaneMesh) Set(vtxAry, normAry, texAry, clrAry math32.ArrayF32, idxAry math32.ArrayU32) {
	// mode := pm.view.Panels[pm.panelNo].Mode
	// if pm.mode == Plane {
	// 	pm.SetPlane(vtxAry, texAry, clrAry, idxAry)
	// } else {
	pm.SetBars(vtxAry, normAry, texAry, clrAry, idxAry)
	// }
}

func (pm *PlaneMesh) SetPlane(vtxAry, normAry, texAry, clrAry math32.ArrayF32, idxAry math32.ArrayU32) {
	// nz := pm.Shape.DimSize(0)
	// nx := pm.Shape.DimSize(1)
	//
	// fnz := float32(nz)
	// fnx := float32(nx)
	//
	// uw := pm.View.Settings.UnitSize
	// uo := (1.0 - uw)
	// segs := 1
	//
	// vtxSz, idxSz := shape.PlaneN(segs, segs)
	// pidx := 0 // plane index
	// pos := math32.Vector3{}
	//
	// pm.View.ReadLock()
	// for zi := nz - 1; zi >= 0; zi-- {
	// 	z0 := uo - float32(zi+1)
	// 	for xi := 0; xi < nx; xi++ {
	// 		poff := pidx * vtxSz * 5
	// 		ioff := pidx * idxSz * 5
	// 		x0 := uo + float32(xi)
	// 		_, scaled, clr, _ := pm.View.UnitValue(pm.Lay, []int{zi, xi})
	// 		v4c := math32.NewVector4Color(clr)
	// 		shape.SetColor(clrAry, poff, 5*vtxSz, v4c)
	// 		ht := 0.5 * math32.Abs(scaled)
	// 		if ht < MinUnitHeight {
	// 			ht = MinUnitHeight
	// 		}
	// 		if scaled >= 0 {
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff, ioff, math32.X, math32.Y, -1, -1, uw, ht, x0, 0, z0, segs, segs, pos)                    // nz
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+1*vtxSz, ioff+1*idxSz, math32.Z, math32.Y, -1, -1, uw, ht, z0, 0, x0+uw, segs, segs, pos) // px
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+2*vtxSz, ioff+2*idxSz, math32.Z, math32.Y, 1, -1, uw, ht, z0, 0, x0, segs, segs, pos)     // nx
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+3*vtxSz, ioff+3*idxSz, math32.X, math32.Z, 1, 1, uw, uw, x0, z0, ht, segs, segs, pos)     // py <-
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+4*vtxSz, ioff+4*idxSz, math32.X, math32.Y, 1, -1, uw, ht, x0, 0, z0+uw, segs, segs, pos)  // pz
	// 		} else {
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff, ioff, math32.X, math32.Y, 1, -1, uw, ht, x0, -ht, z0, segs, segs, pos)                    // nz = pz norm
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+1*vtxSz, ioff+1*idxSz, math32.Z, math32.Y, 1, -1, uw, ht, z0, -ht, x0+uw, segs, segs, pos) // px = nx norm
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+2*vtxSz, ioff+2*idxSz, math32.Z, math32.Y, 1, -1, uw, ht, z0, -ht, x0, segs, segs, pos)    // nx
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+3*vtxSz, ioff+3*idxSz, math32.X, math32.Z, 1, 1, uw, uw, x0, z0, -ht, segs, segs, pos)     // ny <-
	// 			shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+4*vtxSz, ioff+4*idxSz, math32.X, math32.Y, 1, -1, uw, ht, x0, -ht, z0+uw, segs, segs, pos) // pz
	// 		}
	// 		pidx++
	// 	}
	// }
	// pm.View.ReadUnlock()
	//
	// pm.BBox.SetBounds(math32.Vec3(0, -0.5, -fnz), math32.Vec3(fnx, 0.5, 0))
}

func (pm *PlaneMesh) SetBars(vtxAry, normAry, texAry, clrAry math32.ArrayF32, idxAry math32.ArrayU32) {
	nz := int(pm.view.Size.Y)
	nx := int(pm.view.Size.X)

	fnz := float32(nz)
	fnx := float32(nx)

	ctx := GetCtx(0)

	offset := pm.view.Panels[pm.panelNo].Offset
	vr := pm.view.Panels[pm.panelNo].Var
	curprv := pm.view.Panels[pm.panelNo].CurPrev
	tidx := int(ctx.CurState)
	if curprv == Previous {
		tidx = int(ctx.PrevState())
	}

	st := pm.view.Start.Add(offset)
	vri := int(vr.Int64())

	uw := pm.view.Settings.BarSize
	uo := (1.0 - uw)
	segs := 1

	vtxSz, idxSz := shape.PlaneN(segs, segs)
	pidx := 0 // plane index
	pos := math32.Vector3{}

	pm.view.Lock()
	for zi := nz - 1; zi >= 0; zi-- {
		z0 := uo - float32(zi+1)
		ys := int(st.Y) + zi
		for xi := range nx {
			xs := int(st.X) + xi
			poff := pidx * vtxSz * 5
			ioff := pidx * idxSz * 5
			x0 := uo + float32(xi)

			val := State.Value(int(st.Z), ys, xs, vri, tidx)
			scaled, clr := pm.view.ValColor(val)
			v4c := math32.NewVector4Color(clr)
			shape.SetColor(clrAry, poff, 5*vtxSz, v4c)
			ht := 0.5 * math32.Abs(scaled)
			if ht < MinUnitHeight {
				ht = MinUnitHeight
			}
			if scaled >= 0 {
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff, ioff, math32.X, math32.Y, -1, -1, uw, ht, x0, 0, z0, segs, segs, pos)                    // nz
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+1*vtxSz, ioff+1*idxSz, math32.Z, math32.Y, -1, -1, uw, ht, z0, 0, x0+uw, segs, segs, pos) // px
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+2*vtxSz, ioff+2*idxSz, math32.Z, math32.Y, 1, -1, uw, ht, z0, 0, x0, segs, segs, pos)     // nx
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+3*vtxSz, ioff+3*idxSz, math32.X, math32.Z, 1, 1, uw, uw, x0, z0, ht, segs, segs, pos)     // py <-
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+4*vtxSz, ioff+4*idxSz, math32.X, math32.Y, 1, -1, uw, ht, x0, 0, z0+uw, segs, segs, pos)  // pz
			} else {
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff, ioff, math32.X, math32.Y, 1, -1, uw, ht, x0, -ht, z0, segs, segs, pos)                    // nz = pz norm
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+1*vtxSz, ioff+1*idxSz, math32.Z, math32.Y, 1, -1, uw, ht, z0, -ht, x0+uw, segs, segs, pos) // px = nx norm
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+2*vtxSz, ioff+2*idxSz, math32.Z, math32.Y, 1, -1, uw, ht, z0, -ht, x0, segs, segs, pos)    // nx
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+3*vtxSz, ioff+3*idxSz, math32.X, math32.Z, 1, 1, uw, uw, x0, z0, -ht, segs, segs, pos)     // ny <-
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+4*vtxSz, ioff+4*idxSz, math32.X, math32.Y, 1, -1, uw, ht, x0, -ht, z0+uw, segs, segs, pos) // pz
			}
			pidx++
		}
	}
	pm.view.Unlock()
	pm.BBox.SetBounds(math32.Vec3(0, -0.5, -fnz), math32.Vec3(fnx, 0.5, 0))
}
