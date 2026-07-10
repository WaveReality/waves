// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"slices"

	"cogentcore.org/core/math32"
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

	// Schrodinger is the Schrodinger wave function on complex state.
	Schrodinger

	// Maxwell is Maxwell's equations for electromagnetic (EM) waves.
	Maxwell

	// Dirac is Dirac's wave equation coupled with electromagnetic (EM) waves.
	Dirac

	// ParticleKGC is the Klein-Gordon complex version of stochastic particles.
	ParticleKGC
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

	// Move determines if particles actually move according to their momentums.
	Move slbool.Bool

	// C is the speed of light factor. Generally should not exceed 1!
	C float32

	// Diff is the particle diffusion factor: proportion decay relative to neighbors.
	Diff float32

	// CSq = C^2
	CSq float32 `display:"-"`

	// Inv2CSq = 1 / 2C^2
	Inv2CSq float32 `display:"-"`

	// Hbar = h / 2pi = reduced Planck constant.
	Hbar float32

	// Mass is a general mass term, e.g., for the KleinGordon equations.
	Mass float32

	// MOverHSq = Mass^2 / Hbar^2 is the mass drag factor in KleinGordon
	// and related equations. Note: C^2 factor is added in basic vel += c^2 force
	MOverHSq float32 `display:"-"`

	// HSqOver2M = Hbar^2 / 2 Mass is the factor for Schrodinger's equation.
	HSqOver2M float32 `display:"-"`

	// HEOver2MCSq = (Hbar*e) / (2 Mass * CSq) for computing charge.
	HEOver2MCSq float32 `display:"-"`

	// HOverMC = (Hbar) / (Mass * C) for computing particle momentum from phase.
	HOverMC float32 `display:"-"`

	// MOver2 = Mass / 2 for computing kinetic energy.
	MOver2 float32 `display:"-"`

	// MCSq = (Mass^2 * C^2) for computing total momentum squared
	MCSq float32 `display:"-"`

	// C6M2= (C^6 * Mass^2) is the numerator for computing total particle energy
	C6M2 float32 `display:"-"`

	// E is the electric charge constant, which determines the
	// electric potential units, C = A s
	// 0.302822 causes Mu0 and Eps0 to both be 1, if C and Hbar are both 1
	E float32

	// Mu0 is mu_0, or the permeability of free space, which weights
	// the impact of current on the magnetic vector potential.
	Mu0 float32

	// Eps0 is epsilon_0, or the permittivity of free space, which weights
	// the impact of charge on the electrical scalar potential = 1 / (mu0 c^2)
	Eps0 float32 `edit:"-"`

	// OneoEps0 = 1 / Eps0
	OneoEps0 float32 `display:"-"`

	// Edges determines how to handle the edges.
	Edges Edges

	pad, pad1, pad2 float32
}

func (pr *Parameters) Update() {
	pr.CSq = pr.C * pr.C
	pr.Inv2CSq = 1.0 / (2 * pr.CSq)
	pr.MOverHSq = (pr.Mass * pr.Mass) / (pr.Hbar * pr.Hbar)
	pr.HSqOver2M = (pr.Hbar * pr.Hbar) / (2.0 * pr.Mass)
	pr.HEOver2MCSq = (pr.Hbar * pr.E) / (2.0 * pr.Mass * pr.CSq)
	pr.HOverMC = (0.5 * pr.Hbar) / (pr.Mass * pr.C * math32.Cos(math32.DegToRad(45)))
	pr.MOver2 = pr.Mass / 2.0
	pr.MCSq = pr.Mass * pr.Mass * pr.CSq
	pr.C6M2 = pr.CSq * pr.CSq * pr.MCSq
	pr.Eps0 = 1.0 / (pr.Mu0 * pr.C * pr.C)
	pr.OneoEps0 = 1.0 / pr.Eps0
}

//gosl:end

func (pr *Parameters) Defaults() {
	pr.C = 0.5
	pr.Diff = 0.98
	pr.Hbar = 1.0
	pr.Mass = 1.0
	pr.E = 1.0
	pr.Mu0 = 1.0
	pr.Energy.SetBool(true)
	pr.Move.SetBool(true)
	pr.Update()
}

func (pr *Parameters) ShouldDisplay(field string) bool {
	if ParamsShouldDisplay != nil {
		return slices.Contains(ParamsShouldDisplay, field)
	}
	return true
}
