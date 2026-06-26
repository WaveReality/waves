// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import "cogentcore.org/lab/gosl/slbool"

//gosl:start

// Equations are the different implemented equations to simulate.
type Equations int32 //enums:enum

const (
	// Wave1D is the basic wave equation in one dimension (X).
	Wave1D Equations = iota

	// Wave3D is the basic wave equation in three dimensions.
	Wave3D
)

// Edges determines how to handle the edges.
type Edges int32 //enums:enum

const (
	// EdgesFixed keeps the edge values fixed at initial values
	EdgesFixed Edges = iota

	// EdgesWrap copies edge values from other side, effectively wrapping
	// the space around on itself like a torus.
	EdgesWrap
)

// Parameters contains the full set of simulation parameters.
// this is uploaded to the GPU.
type Parameters struct {
	// Equation determines what equations are computed.
	Equation Equations

	// Edges determines how to handle the edges.
	Edges Edges

	// DoEnergy determines if energy is computed (when not necessary).
	DoEnergy slbool.Bool

	pad int32

	// Units are the relevant unit factors.
	Units Units
}

func (pr *Parameters) Defaults() {
	pr.Units.Defaults()
}

func (pr *Parameters) Update() {
	pr.Units.Update()
}

// Display contains display parameters.
type Display struct {
	// On determines if display is updated.
	On bool

	// Interval is the number of time steps between display updates.
	Interval int
}

// Units contains all the relevant units
type Units struct {
	// C is the speed of light factor
	C float32

	// CSq = C^2
	CSq float32 `edit:"-"`

	// Inv2CSq = 1 / 2C^2
	Inv2CSq float32 `edit:"-"`

	pad float32
}

func (un *Units) Defaults() {
	un.C = 0.5
	un.Update()
}

func (un *Units) Update() {
	un.CSq = un.C * un.C
	un.Inv2CSq = 1.0 / (2 * un.CSq)
}

//gosl:end
