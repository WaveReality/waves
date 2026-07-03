// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"slices"

	"cogentcore.org/lab/gosl/slbool"
)

// ParamsShouldDisplay should be set to an equation-specific list of
// Parameters field names to display. To simplify the
// GPU configuration, it is important to only have one struct
// with everything in it, so this simplifies things.
var ParamsShouldDisplay []string

//gosl:start

// Equations are the different implemented equations to simulate.
type Equations int32 //enums:enum

const (
	// Wave is the basic wave equation in one dimension (X).
	Wave Equations = iota

	// KleinGordon is the Klein-Gordon massive particle wave function,
	// on a scalar wave state.
	KleinGordon

	// KleinGordonC is the Klein-Gordon massive particle wave function,
	// on a complex wave state.
	KleinGordonC

	// Schrodinger is the 1D Schrodinger wave function on complex state.
	Schrodinger
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

// The following are constants used across many equations.
const (
	Pi       = 3.14159265358979323846264338327950288419716939937510582097494459
	TwoPi    = 2 * Pi
	InvTwoPi = 1.0 / TwoPi
)

// Parameters contains the full set of simulation parameters,
// for all equations. These are the bare computational values,
// uploaded to the GPU.
// Use Units to set values relative to a particular set of units.
type Parameters struct {
	// ThreeD runs the 3D version of wave equations, else 1D.
	ThreeD slbool.Bool

	// Energy determines if energy is computed (when not necessary).
	Energy slbool.Bool

	// C is the speed of light factor. Generally should not exceed 1!
	C float32

	// CSq = C^2
	CSq float32 `display:"-"`

	// Inv2CSq = 1 / 2C^2
	Inv2CSq float32 `display:"-"`

	// HBar = h / 2pi = reduced Planck constant.
	HBar float32

	// Mass is a general mass term, e.g., for the KleinGordon equations.
	Mass float32

	// MassCOverHBarSq = Mass^2 C^2 / HBar^2 is the mass drag factor
	// in KleinGordon and related equations.
	MassCOverHBarSq float32 `display:"-"`

	// HBarSqOver2Mass = HBar^2 / 2 Mass is the factor for Schrodinger's equation.
	HBarSqOver2Mass float32 `display:"-"`

	// MassOver2 = Mass / 2 for computing kinetic energy.
	MassOver2 float32 `display:"-"`

	// Wavelength is the wavelength to use for functions that use it
	// (Params suffix). Allows user to manipulate the wavelength easily,
	// e.g., for KG and other matter waves.
	Wavelength float32

	// PacketWidth is the wave packet width to use for functions that use it
	// (Params suffix). Allows user to manipulate the wave parameters easily,
	// e.g., for KG and other matter waves.
	PacketWidth float32

	// Edges determines how to handle the edges.
	Edges Edges

	pad, pad1, pad2 float32
}

func (pr *Parameters) Update() {
	pr.CSq = pr.C * pr.C
	pr.Inv2CSq = 1.0 / (2 * pr.CSq)
	pr.MassCOverHBarSq = (pr.Mass * pr.Mass * pr.CSq) / (pr.HBar * pr.HBar)
	pr.HBarSqOver2Mass = (pr.HBar * pr.HBar) / (2.0 * pr.Mass)
	pr.MassOver2 = pr.Mass / 2.0
}

//gosl:end

func (pr *Parameters) Defaults() {
	pr.C = 0.5
	pr.HBar = 1.0
	pr.Mass = 1.0
	pr.Wavelength = 8
	pr.PacketWidth = 8
	pr.Energy.SetBool(true)
	pr.Update()
}

func (pr *Parameters) ShouldDisplay(field string) bool {
	if ParamsShouldDisplay != nil {
		return slices.Contains(ParamsShouldDisplay, field)
	}
	return true
}
