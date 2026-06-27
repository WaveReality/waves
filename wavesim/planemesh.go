// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"cogentcore.org/core/gpu/shape"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/xyz"
	"github.com/emer/emergent/v2/emer"
)

// PlaneMesh is a xyz.Mesh that represents an X-Y plane through the state.
// It is dynamically updated using theUpdate method which only resets the
// essential Vertex elements. The geometry is literal in the size:
// 0,0,0 lower-left corner and increasing X,Z in display for the X,Y plane.
// Display applies an overall scaling to make it fit within the larger view.
type PlaneMesh struct {
	xyz.MeshBase

	// Starting front-left corner location within state.
	Start math32.Vector3i

	// Size of plane
	Size math32.Vector2i
}

// NewPlaneMesh adds PlaneMesh mesh to given scene for given layer
func NewPlaneMesh(sc *xyz.Scene, nv *NetView, lay emer.Layer) *PlaneMesh {
	pm := &PlaneMesh{}
	pm.View = nv
	pm.Lay = lay
	pm.Name = lay.Label()
	sc.SetMesh(pm)
	return pm
}

func (pm *PlaneMesh) MeshSize() (nVtx, nIndex int, hasColor bool) {
	pm.Transparent = true
	pm.HasColor = true
	if pm.Lay == nil {
		return 0, 0, true
	}
	shp := &pm.Lay.AsEmer().Shape
	pm.Shape.CopyFrom(shp)
	if pm.View.Settings.Raster.On {
		if shp.NumDims() == 4 {
			pm.NumVertex, pm.NumIndex = pm.RasterSize4D()
		} else {
			pm.NumVertex, pm.NumIndex = pm.RasterSize2D()
		}
	} else {
		if shp.NumDims() == 4 {
			pm.NumVertex, pm.NumIndex = pm.Size4D()
		} else {
			pm.NumVertex, pm.NumIndex = pm.Size2D()
		}
	}
	return pm.NumVertex, pm.NumIndex, pm.HasColor
}

func (pm *PlaneMesh) Size2D() (nVtx, nIndex int) {
	nz := pm.Shape.DimSize(0)
	nx := pm.Shape.DimSize(1)
	segs := 1

	vtxSz, idxSz := shape.PlaneN(segs, segs)
	nVtx = vtxSz * 5 * nz * nx
	nIndex = idxSz * 5 * nz * nx
	return
}

func (pm *PlaneMesh) Size4D() (nVtx, nIndex int) {
	npz := pm.Shape.DimSize(0) // p = pool
	npx := pm.Shape.DimSize(1)
	nuz := pm.Shape.DimSize(2) // u = unit
	nux := pm.Shape.DimSize(3)

	segs := 1

	vtxSz, idxSz := shape.PlaneN(segs, segs)
	nVtx = vtxSz * 5 * npz * npx * nuz * nux
	nIndex = idxSz * 5 * npz * npx * nuz * nux
	return
}

func (pm *PlaneMesh) Set(vtxAry, normAry, texAry, clrAry math32.ArrayF32, idxAry math32.ArrayU32) {
	if pm.Lay == nil || pm.Shape.NumDims() == 0 {
		return // nothing
	}
	if pm.View.Settings.Raster.On {
		if pm.View.Settings.Raster.XAxis {
			if pm.Shape.NumDims() == 4 {
				pm.RasterSet4DX(vtxAry, normAry, texAry, clrAry, idxAry)
			} else {
				pm.RasterSet2DX(vtxAry, normAry, texAry, clrAry, idxAry)
			}
		} else {
			if pm.Shape.NumDims() == 4 {
				pm.RasterSet4DZ(vtxAry, normAry, texAry, clrAry, idxAry)
			} else {
				pm.RasterSet2DZ(vtxAry, normAry, texAry, clrAry, idxAry)
			}
		}
	} else {
		if pm.Shape.NumDims() == 4 {
			pm.Set4D(vtxAry, normAry, texAry, clrAry, idxAry)
		} else {
			pm.Set2D(vtxAry, normAry, texAry, clrAry, idxAry)
		}
	}
}

// MinUnitHeight ensures that there is always at least some dimensionality
// to the unit cubes -- affects transparency rendering etc
var MinUnitHeight = float32(1.0e-6)

func (pm *PlaneMesh) Set2D(vtxAry, normAry, texAry, clrAry math32.ArrayF32, idxAry math32.ArrayU32) {
	nz := pm.Shape.DimSize(0)
	nx := pm.Shape.DimSize(1)

	fnz := float32(nz)
	fnx := float32(nx)

	uw := pm.View.Settings.UnitSize
	uo := (1.0 - uw)
	segs := 1

	vtxSz, idxSz := shape.PlaneN(segs, segs)
	pidx := 0 // plane index
	pos := math32.Vector3{}

	pm.View.ReadLock()
	for zi := nz - 1; zi >= 0; zi-- {
		z0 := uo - float32(zi+1)
		for xi := 0; xi < nx; xi++ {
			poff := pidx * vtxSz * 5
			ioff := pidx * idxSz * 5
			x0 := uo + float32(xi)
			_, scaled, clr, _ := pm.View.UnitValue(pm.Lay, []int{zi, xi})
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
	pm.View.ReadUnlock()

	pm.BBox.SetBounds(math32.Vec3(0, -0.5, -fnz), math32.Vec3(fnx, 0.5, 0))
}

func (pm *PlaneMesh) Set4D(vtxAry, normAry, texAry, clrAry math32.ArrayF32, idxAry math32.ArrayU32) {
	npz := pm.Shape.DimSize(0) // p = pool
	npx := pm.Shape.DimSize(1)
	nuz := pm.Shape.DimSize(2) // u = unit
	nux := pm.Shape.DimSize(3)

	fnpz := float32(npz)
	fnpx := float32(npx)
	fnuz := float32(nuz)
	fnux := float32(nux)

	usz := pm.View.Settings.UnitSize
	uo := (1.0 - usz) // offset = space

	// for 4D, we build in spaces between groups without changing the overall size of layer
	// by shrinking the spacing of each unit according to the spaces we introduce
	xsc := (fnpx * fnux) / ((fnpx-1)*uo + (fnpx * fnux))
	zsc := (fnpz * fnuz) / ((fnpz-1)*uo + (fnpz * fnuz))

	xuw := xsc * usz
	zuw := zsc * usz

	segs := 1

	vtxSz, idxSz := shape.PlaneN(segs, segs)
	pidx := 0 // plane index
	pos := math32.Vector3{}

	pm.View.ReadLock()
	for zpi := npz - 1; zpi >= 0; zpi-- {
		zp0 := zsc * (-float32(zpi) * (uo + fnuz))
		for xpi := 0; xpi < npx; xpi++ {
			xp0 := xsc * (float32(xpi)*uo + float32(xpi)*fnux)
			for zui := nuz - 1; zui >= 0; zui-- {
				z0 := zp0 + zsc*(uo-float32(zui+1))
				for xui := 0; xui < nux; xui++ {
					poff := pidx * vtxSz * 5
					ioff := pidx * idxSz * 5
					x0 := xp0 + xsc*(uo+float32(xui))
					_, scaled, clr, _ := pm.View.UnitValue(pm.Lay, []int{zpi, xpi, zui, xui})
					v4c := math32.NewVector4Color(clr)
					shape.SetColor(clrAry, poff, 5*vtxSz, v4c)
					ht := 0.5 * math32.Abs(scaled)
					if ht < MinUnitHeight {
						ht = MinUnitHeight
					}
					if scaled >= 0 {
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff, ioff, math32.X, math32.Y, -1, -1, xuw, ht, x0, 0, z0, segs, segs, pos)                     // nz
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+1*vtxSz, ioff+1*idxSz, math32.Z, math32.Y, -1, -1, zuw, ht, z0, 0, x0+xuw, segs, segs, pos) // px
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+2*vtxSz, ioff+2*idxSz, math32.Z, math32.Y, 1, -1, zuw, ht, z0, 0, x0, segs, segs, pos)      // nx
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+3*vtxSz, ioff+3*idxSz, math32.X, math32.Z, 1, 1, xuw, zuw, x0, z0, ht, segs, segs, pos)     // py <-
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+4*vtxSz, ioff+4*idxSz, math32.X, math32.Y, 1, -1, xuw, ht, x0, 0, z0+zuw, segs, segs, pos)  // pz
					} else {
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff, ioff, math32.X, math32.Y, 1, -1, xuw, ht, x0, -ht, z0, segs, segs, pos)                     // nz = pz norm
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+1*vtxSz, ioff+1*idxSz, math32.Z, math32.Y, 1, -1, zuw, ht, z0, -ht, x0+xuw, segs, segs, pos) // px = nx norm
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+2*vtxSz, ioff+2*idxSz, math32.Z, math32.Y, 1, -1, zuw, ht, z0, -ht, x0, segs, segs, pos)     // nx
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+3*vtxSz, ioff+3*idxSz, math32.X, math32.Z, 1, 1, xuw, zuw, x0, z0, -ht, segs, segs, pos)     // ny <-
						shape.SetPlane(vtxAry, normAry, texAry, idxAry, poff+4*vtxSz, ioff+4*idxSz, math32.X, math32.Y, 1, -1, xuw, ht, x0, -ht, z0+zuw, segs, segs, pos) // pz
					}
					pidx++
				}
			}
		}
	}
	pm.View.ReadUnlock()

	pm.BBox.SetBounds(math32.Vec3(0, -0.5, -fnpz*fnuz), math32.Vec3(fnpx*fnux, 0.5, 0))
}
