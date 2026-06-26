// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import "cogentcore.org/lab/gosl/slvec"

//gosl:start

// Context contains all simulation counters and other context.
// This is only other state shared with GPU.
type Context struct {
	// Size is the 3D size of the state, EXCLUSIVE of edges (add 2 to each dim).
	Size slvec.Vector3i

	// Step is the current simulation timestep.
	Step int32

	// CurState is either 0 or 1, indicating which state variables
	// are currently being updated on this compute pass.
	CurState int32

	pad, pad1 int32
}

func (ctx *Context) Init() {
	ctx.Step = 0
	ctx.CurState = 0
}

// StateCoords returns the x,y,z coordinates for given index into
// the state, where index is in Size units of active states,
// excluding edges. Resulting coords have 1 added to each,
// so they are valid coordinates into actual State.
// returns false if the index is out of range for size.
func (ctx *Context) StateCoords(idx uint32, x, y, z *int32) bool {
	szxy := ctx.Size.X * ctx.Size.Y
	*z = int32(idx) / szxy
	rz := int32(idx) % szxy
	szx := ctx.Size.X
	*y = rz / szx
	*x = rz % szx
	if *z >= ctx.Size.Z || *y >= ctx.Size.Y || *x >= ctx.Size.X {
		return false
	}
	*z += 1
	*y += 1
	*x += 1
	return true
}

// PrevState returns the index for the previous state, relative to CurState.
func (ctx *Context) PrevState() int32 {
	return 1 - ctx.CurState
}

// StepInc increments for next step of processing.
func (ctx *Context) StepInc() {
	ctx.Step++
	ctx.CurState = ctx.PrevState()
}

//gosl:end
