// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"

	"cogentcore.org/core/colors/colormap"
	"cogentcore.org/core/math32"
)

// vwSim builds a state tensor of the given interior size and fills the chosen
// variable with a value that encodes its own coordinates, so that any sampling
// mistake shows up as the wrong cell rather than the wrong number.
func vwSim(size math32.Vector3i, vr EWStates) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU = false
	ss.Config.GUI = false
	ss.Config.Equation = Electroweak
	ss.Config.Size = size
	ss.ConfigSim()
	ss.StateVars = EWStatesN
	ss.ConfigState()
	GetCtx(0).Init()
	State.SetZeros()
	cur := int(GetCtx(0).CurState)
	for z := int32(0); z < size.Z+2; z++ {
		for y := int32(0); y < size.Y+2; y++ {
			for x := int32(0); x < size.X+2; x++ {
				State.Set(vwCode(x, y, z), int(z), int(y), int(x), int(vr.Int64()), cur)
			}
		}
	}
	return ss
}

// vwCode packs a coordinate into one value, distinct per cell and small enough
// that ValColor's fallback (clamp to +-1) leaves it untouched. The digits are
// kept well separated because the color pipeline rescales through
// 0.5*(v+1) and back, which costs a float32 several digits.
func vwCode(x, y, z int32) float32 {
	return float32(x*100+y*10+z) / 1000
}

func vwView(ss *Sim, vr EWStates, depth math32.Dims, slice int32) *View {
	vw := &View{}
	vw.sim = ss
	vw.Settings.Defaults()
	vw.colorMap = colormap.AvailableMaps[string(vw.Settings.ColorMap)]
	vw.Size = ss.Config.Size
	vw.Depth = depth
	vw.Panels[0].Var = vr
	vw.Panels[0].Mode = Plane
	vw.Panels[0].CurPrev = Current
	vw.Start.X = 1
	vw.Start.SetDim(depth, 1)
	vw.Start.SetDim(vw.SliceDim(), slice)
	return vw
}

// TestViewSliceMapping renders the height field directly and checks that every
// vertex carries the value of the state cell it is supposed to: display X is
// always state X, display depth is View.Depth, and the remaining dimension is
// pinned at the slice level.
func TestViewSliceMapping(t *testing.T) {
	// deliberately unequal, so a swapped axis cannot pass by coincidence
	size := math32.Vec3i(7, 5, 3)
	ss := vwSim(size, EWHs0a)
	_ = ss

	for _, tc := range []struct {
		depth math32.Dims
		slice int32
	}{
		{math32.Z, 2}, // X-Z plane at a Y level
		{math32.Y, 2}, // X-Y plane at a Z level
	} {
		vw := vwView(ss, EWHs0a, tc.depth, tc.slice)
		nx := int(vw.Size.X)
		nz := int(vw.DepthSize())
		if nz < 2 {
			t.Fatalf("depth %v gives only %d cells", tc.depth, nz)
		}
		pm := &PlaneMesh{view: vw, panelNo: 0}
		nVtx, nIndex, _ := pm.MeshSize()
		if nVtx != nz*nx {
			t.Errorf("depth %v: MeshSize gave %d vertices, want %d", tc.depth, nVtx, nz*nx)
		}
		vtx := make(math32.ArrayF32, 3*nVtx)
		norm := make(math32.ArrayF32, 3*nVtx)
		tex := make(math32.ArrayF32, 2*nVtx)
		clr := make(math32.ArrayF32, 4*nVtx)
		idx := make(math32.ArrayU32, nIndex)
		pm.SetPlane(vtx, norm, tex, clr, idx)

		for zi := 0; zi < nz; zi++ {
			for xi := 0; xi < nx; xi++ {
				vi := (nz-1-zi)*nx + xi
				var v math32.Vector3
				vtx.GetVector3(vi*3, &v)
				// the plane is laid out with X across and -Z going back
				if int(v.X) != xi || int(-v.Z) != zi {
					t.Fatalf("depth %v: vertex %d at display (%d,%d) placed at (%g,%g)",
						tc.depth, vi, xi, zi, v.X, -v.Z)
				}
				want := vw.StateCoord(vw.Start, int32(xi), int32(zi))
				if want.Dim(vw.SliceDim()) != tc.slice {
					t.Fatalf("slice dim drifted: %v", want)
				}
				if got := 0.5 * vwCode(want.X, want.Y, want.Z); math.Abs(float64(v.Y-got)) > 1e-5 {
					t.Errorf("depth %v display (%d,%d): height %g, want %g for state %v",
						tc.depth, xi, zi, v.Y, got, want)
				}
			}
		}
		t.Logf("depth %v: %d x %d cells, slice %v = %d, all vertices map to the right state cell",
			tc.depth, nx, nz, vw.SliceDim(), tc.slice)
	}
}

// TestViewDisplayVector: a state vector must point along the axis its own
// dimension is drawn on, in either view.
func TestViewDisplayVector(t *testing.T) {
	v := math32.Vec3(1, 2, 3)
	xz := (&View{Depth: math32.Z}).DisplayVector(v)
	if xz != v {
		t.Errorf("X-Z view should pass a vector through unchanged: got %v", xz)
	}
	xy := (&View{Depth: math32.Y}).DisplayVector(v)
	// state Y is drawn going back into the screen (display Z), state Z is up
	if xy != math32.Vec3(1, 3, 2) {
		t.Errorf("X-Y view should swap Y and Z: got %v", xy)
	}
}

// TestViewMoveSlice: the slice level moves within bounds and nothing else does.
func TestViewMoveSlice(t *testing.T) {
	size := math32.Vec3i(7, 5, 3)
	ss := vwSim(size, EWHs0a)
	_ = ss
	vw := vwView(ss, EWHs0a, math32.Z, 2)
	sd := vw.SliceDim()
	if sd != math32.Y {
		t.Fatalf("X-Z view should slice in Y, got %v", sd)
	}
	before := vw.Start
	vw.MoveSlice(1)
	if vw.Start.Dim(sd) != before.Dim(sd)+1 {
		t.Errorf("slice did not move: %v -> %v", before, vw.Start)
	}
	if vw.Start.X != before.X || vw.Start.Dim(vw.Depth) != before.Dim(vw.Depth) {
		t.Errorf("in-plane corner moved: %v -> %v", before, vw.Start)
	}
	for range 100 { // must clamp, not run off the end
		vw.MoveSlice(1)
	}
	if got, mx := vw.Start.Dim(sd), size.Dim(sd)+1; got != mx {
		t.Errorf("slice clamped to %d, want %d", got, mx)
	}
	for range 100 {
		vw.MoveSlice(-1)
	}
	if got := vw.Start.Dim(sd); got != 0 {
		t.Errorf("slice clamped to %d, want 0", got)
	}
}
