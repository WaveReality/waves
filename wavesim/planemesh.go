// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/xyz"
)

// PlaneMesh is a xyz.Mesh that represents an X-Y plane through the state,
// as either a Heightfield or bars.
// It is dynamically updated using the Set method.
// The geometry is literal in the size:
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
	hasColor = true
	pm.HasColor = hasColor

	nz := int(pm.view.Size.Y)
	nx := int(pm.view.Size.X)
	segs := 1

	mode := pm.view.Panels[pm.panelNo].Mode

	switch mode {
	case Plane:
		nVtx = nz * nx
		nIndex = 6 * (nz - 1) * (nx - 1) // two triangles per square
	case Bars:
		nper := 5
		vtxSz, idxSz := shape.PlaneN(segs, segs)
		nVtx = vtxSz * nper * nz * nx
		nIndex = idxSz * nper * nz * nx
	case Vectors:
		nVtx = nz * nx * 4 * 5   // 4 points * 5 planes
		nIndex = nz * nx * 5 * 6 // two triangles per 5 planes
	}
	pm.NumVertex, pm.NumIndex = nVtx, nIndex
	return
}

// MinUnitHeight ensures that there is always at least some dimensionality
// to the unit cubes -- affects transparency rendering etc
var MinUnitHeight = float32(1.0e-6)

func (pm *PlaneMesh) Set(vtxAry, normAry, texAry, clrAry math32.ArrayF32, idxAry math32.ArrayU32) {
	mode := pm.view.Panels[pm.panelNo].Mode
	switch mode {
	case Plane:
		pm.SetPlane(vtxAry, normAry, texAry, clrAry, idxAry)
	case Bars:
		pm.SetBars(vtxAry, normAry, texAry, clrAry, idxAry)
	case Vectors:
		pm.SetVectors(vtxAry, normAry, texAry, clrAry, idxAry)
	}
}

func (pm *PlaneMesh) SetPlane(vtxAry, normAry, texAry, clrAry math32.ArrayF32, idxAry math32.ArrayU32) {
	nz := int(pm.view.Size.Y)
	nx := int(pm.view.Size.X)

	if nz < 2 {
		return
	}

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

	vidx := 0 // vertex index
	var vtx, norm, a, b, c, d math32.Vector3

	pm.view.Lock()

	// have to write all verticies first, so have access to all the data
	// for computing norms!
	for zi := nz - 1; zi >= 0; zi-- {
		ys := int(st.Y) + zi
		for xi := 0; xi < nx; xi++ {
			xs := int(st.X) + xi
			val := State.Value(int(st.Z), ys, xs, vri, tidx)
			scaled, _, clr := pm.view.ValColor(val, pm.panelNo)
			v4c := math32.NewVector4Color(clr)
			shape.SetColor(clrAry, vidx, 1, v4c)
			ht := 0.5 * scaled
			vtx.Set(float32(xi), ht, -float32(zi))
			vtx.ToSlice(vtxAry, vidx*3)
			vidx++
		}
	}

	// b c
	// a d
	// abd, bcd
	// because z is inverted, a is higher z, b is lower
	nz1 := nz - 1
	vIdx := func(z, x int) int {
		return ((nz1-z)*nx + x)
	}

	nidx := 0
	iidx := 0 // index index
	for zi := nz1; zi >= 0; zi-- {
		for xi := 0; xi < nx; xi++ {
			if zi == 0 { // no index
				if xi == nx-1 { // no right
					vtxAry.GetVector3(vIdx(1, xi-1)*3, &a) // a = zi=1, xi=left
					vtxAry.GetVector3(vIdx(0, xi-1)*3, &b) // b = zi=0, xi=left
					vtxAry.GetVector3(vIdx(0, xi)*3, &c)   // c = zi=0, xi=us
				} else {
					vtxAry.GetVector3(vIdx(1, xi)*3, &a)   // a = zi=1, xi=us
					vtxAry.GetVector3(vIdx(0, xi)*3, &b)   // b = zi=0, xi=us
					vtxAry.GetVector3(vIdx(0, xi+1)*3, &c) // c = zi=0, xi=right
				}
			} else { // do indexes
				if xi == nx-1 { // no right, no index
					vtxAry.GetVector3(vIdx(zi, xi-1)*3, &a)   // a = zi, xi=left
					vtxAry.GetVector3(vIdx(zi-1, xi-1)*3, &b) // b = zi-1, xi=left
					vtxAry.GetVector3(vIdx(zi-1, xi)*3, &c)   // c = zi-1, xi=us
				} else {
					ai := vIdx(zi, xi)     // a = zi, xi=us
					bi := vIdx(zi-1, xi)   // b = zi-1, xi=us
					ci := vIdx(zi-1, xi+1) // c = zi-1, xi=right
					di := vIdx(zi, xi+1)   // d = zi, xi=right
					vtxAry.GetVector3(ai*3, &a)
					vtxAry.GetVector3(bi*3, &b)
					vtxAry.GetVector3(ci*3, &c)
					vtxAry.GetVector3(di*3, &d)

					idxAry.Set(iidx, uint32(ai), uint32(bi), uint32(di), uint32(bi), uint32(ci), uint32(di))
					iidx += 6
				}
			}
			norm = math32.Normal(a, b, c)
			norm.ToSlice(normAry, nidx)
			nidx += 3
		}
	}
	pm.view.Unlock()
	pm.BBox.SetBounds(math32.Vec3(0, -0.5, -fnz), math32.Vec3(fnx, 0.5, 0))
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
	nper := 5

	vtxSz, idxSz := shape.PlaneN(segs, segs)
	pidx := 0 // plane index
	pos := math32.Vector3{}

	pm.view.Lock()
	for zi := nz - 1; zi >= 0; zi-- {
		ys := int(st.Y) + zi
		z0 := uo - float32(zi+1)
		for xi := range nx {
			xs := int(st.X) + xi
			poff := pidx * vtxSz * nper
			ioff := pidx * idxSz * nper
			x0 := uo + float32(xi)

			val := State.Value(int(st.Z), ys, xs, vri, tidx)
			scaled, _, clr := pm.view.ValColor(val, pm.panelNo)
			v4c := math32.NewVector4Color(clr)
			shape.SetColor(clrAry, poff, nper*vtxSz, v4c)
			ht := 0.5 * math32.Abs(scaled)
			if ht < MinUnitHeight {
				ht = MinUnitHeight
			}
			base := float32(0)
			if scaled >= 0 {
				// back
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff, ioff, math32.X, math32.Y, -1, -1, uw, ht, x0, 0, z0, segs, segs, pos)
				// left
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+1*vtxSz, ioff+1*idxSz, math32.Z, math32.Y, -1, -1, uw, ht, z0, base, x0+uw, segs, segs, pos)
				// right
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+2*vtxSz, ioff+2*idxSz, math32.Z, math32.Y, 1, -1, uw, ht, z0, base, x0, segs, segs, pos)
				// top
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+3*vtxSz, ioff+3*idxSz, math32.X, math32.Z, 1, 1, uw, uw, x0, z0, ht, segs, segs, pos)
				// front
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+4*vtxSz, ioff+4*idxSz, math32.X, math32.Y, 1, -1, uw, ht, x0, base, z0+uw, segs, segs, pos)
			} else {
				base = -ht
				// back
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff, ioff, math32.X, math32.Y, 1, -1, uw, ht, x0, base, z0, segs, segs, pos)
				// bottom
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+1*vtxSz, ioff+1*idxSz, math32.X, math32.Z, 1, 1, uw, uw, x0, z0, base, segs, segs, pos)
				// left
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+2*vtxSz, ioff+2*idxSz, math32.Z, math32.Y, 1, -1, uw, ht, z0, base, x0+uw, segs, segs, pos)
				// right
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+3*vtxSz, ioff+3*idxSz, math32.Z, math32.Y, 1, -1, uw, ht, z0, base, x0, segs, segs, pos)
				// front
				shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+4*vtxSz, ioff+4*idxSz, math32.X, math32.Y, 1, -1, uw, ht, x0, base, z0+uw, segs, segs, pos) // pz
			}
			pidx++
		}
	}
	pm.view.Unlock()
	pm.BBox.SetBounds(math32.Vec3(0, -0.5, -fnz), math32.Vec3(fnx, 0.5, 0))
}

func (pm *PlaneMesh) SetVectors(vtxAry, normAry, texAry, clrAry math32.ArrayF32, idxAry math32.ArrayU32) {
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

	uw := pm.view.Settings.VectorSize
	uy := uw * .05
	start := 0.5 - 0.5*uw

	vidx := 0 // vertex index
	nidx := 0
	iidx := 0 // index index
	var bbl, bbr, bfl, bfr, tbl, tbr, tfl, tfr, norm math32.Vector3
	up := math32.Vec3(1, 1, 1)

	setPlane := func(a, b, c, d math32.Vector3) {
		// b c
		// a d
		// abd, bcd
		a.ToSlice(vtxAry, (vidx+0)*3)
		b.ToSlice(vtxAry, (vidx+1)*3)
		c.ToSlice(vtxAry, (vidx+2)*3)
		d.ToSlice(vtxAry, (vidx+3)*3)
		idxAry.Set(iidx, uint32(vidx+0), uint32(vidx+1), uint32(vidx+3), uint32(vidx+1), uint32(vidx+2), uint32(vidx+3))
		vidx += 4
		iidx += 6
		norm = math32.Normal(a, b, c)
		for range 4 {
			norm.ToSlice(normAry, nidx)
			nidx += 3
		}
	}

	pm.view.Lock()
	for zi := nz - 1; zi >= 0; zi-- {
		ys := int(st.Y) + zi
		z0 := start - float32(zi+1)
		for xi := range nx {
			xs := int(st.X) + xi
			x0 := start + float32(xi)

			vx := State.Value(int(st.Z), ys, xs, vri, tidx)
			vy := State.Value(int(st.Z), ys, xs, vri+1, tidx)
			vz := State.Value(int(st.Z), ys, xs, vri+2, tidx)
			ed := math32.Vec3(vx, vy, vz)
			d := ed.Length()
			ned := ed
			if d > 0 {
				ned.SetDivScalar(d)
			}
			sgn := up.Dot(ed)
			if sgn < 0 {
				d = -d
			}
			_, maxval, clr := pm.view.ValColor(d, pm.panelNo)
			if maxval > 0 {
				ed.SetDivScalar(maxval)
			}

			if math32.Abs(ned.X) > .95 { // mostly X
				bbl.Set(x0, 0, z0)
				bbr.Set(x0+ed.X, 0, z0)
				bfl.Set(x0, 0, z0+uw)
				bfr.Set(x0+ed.X, 0, z0+uw)
				tbl.Set(x0, ed.Y+uy, z0+ed.Z)
				tbr.Set(x0+ed.X, ed.Y+uy, z0+ed.Z)
				tfl.Set(x0, ed.Y+uy, z0+uw+ed.Z)
				tfr.Set(x0+ed.X, ed.Y+uy, z0+uw+ed.Z)
			} else if math32.Abs(ned.Z) > .95 { // mostly Z
				bbl.Set(x0, 0, z0)
				bbr.Set(x0+uw, 0, z0)
				bfl.Set(x0, 0, z0+ed.Z)
				bfr.Set(x0+uw, 0, z0+ed.Z)
				tbl.Set(x0+ed.X, ed.Y+uy, z0)
				tbr.Set(x0+uw+ed.X, ed.Y+uy, z0)
				tfl.Set(x0+ed.X, ed.Y+uy, z0+ed.Z)
				tfr.Set(x0+uw+ed.X, ed.Y+uy, z0+ed.Z)
			} else {
				bbl.Set(x0, 0, z0)
				bbr.Set(x0+uw, 0, z0)
				bfl.Set(x0, 0, z0+uw)
				bfr.Set(x0+uw, 0, z0+uw)
				tbl.Set(x0+ed.X, ed.Y, z0+ed.Z)
				tbr.Set(x0+uw+ed.X, ed.Y, z0+ed.Z)
				tfl.Set(x0+ed.X, ed.Y, z0+uw+ed.Z)
				tfr.Set(x0+uw+ed.X, ed.Y, z0+uw+ed.Z)
			}

			v4c := math32.NewVector4Color(clr)
			shape.SetColor(clrAry, vidx, 4*5, v4c) // 4 vertex per plane, 5, planes

			// back
			setPlane(bbl, tbl, tbr, bbr)
			// left
			setPlane(bbl, tbl, tfl, bfl)
			// right
			setPlane(bfr, tfr, tbr, bbr)
			// front
			setPlane(bfl, tfl, tfr, bfr)
			// top
			setPlane(tfl, tbl, tbr, tfr)
		}
	}
	pm.view.Unlock()
	pm.BBox.SetBounds(math32.Vec3(0, -0.5, -fnz), math32.Vec3(fnx, 0.5, 0))
}
