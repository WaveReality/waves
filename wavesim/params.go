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

	// Electroweak simulates the electroweak system, including the Higgs
	Electroweak

	// Spinfield is the Klein-Gordon complex version of stochastic particles.
	Spinfield

	// ParticleMove is just particle motion without any wave field.
	ParticleMove
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

	// Edges determines how to handle the edges.
	Edges Edges

	// Energy determines if energy is computed (when not necessary).
	Energy slbool.Bool

	// C is the speed of light factor. Generally should not exceed 1!
	C float32

	// Hbar = h / 2pi = reduced Planck constant.
	Hbar float32

	// Mass is a general mass term, e.g., for the KleinGordon equations.
	// In general, this should be set to Hbar / (Compton * C) for the
	// target Compton (bar) wavelength, e.g. .125 for C = 0.5 and Compton = 16.
	// If negative, then M^2 is negative!
	Mass float32

	// VPotential is the strength of an external potential field, which can then
	// influence wave propagation. This is just the magnitude of the potential,
	// which is actually negative. In general may need to reduce C below 1 to
	// ensure stability with larger potential values.
	VPotential float32

	// EM determines if EM field coupling is activated.
	EM slbool.Bool

	// A0NoWave applies the neighborhood force directly as a velocity, instead
	// of as a force that adds to the velocity, for the scalar potential, A0.
	// This prevents non-physical longitudinal wave propagation.
	// It increases the sensitivity, such that C must be less than 1.
	// This is the same as Sommerfield edge damping.
	A0NoWave slbool.Bool

	// E is the electric charge constant, which determines the
	// electric potential units, C = A s
	// 0.302822 causes Mu0 and Eps0 to both be 1, if C and Hbar are both 1
	E float32

	// Mu0 is mu_0, or the permeability of free space, which weights
	// the impact of current on the magnetic vector potential.
	Mu0 float32

	// Move determines if particles actually move according to their momentums.
	Move slbool.Bool

	// HiggsMu is the Higgs potential target mass (mu_H), which determines the
	// expected value of the symmetry-broken higgs field squared complex magnitude.
	HiggsMu float32

	// HiggsLambda is the Higgs potential weight factor that determines
	// effectively how strongly the complex magnitude minus mu_H contributes.
	HiggsLambda float32

	// Diff is the particle diffusion rate: how fast to spread distance to neighbors.
	Diff float32

	// Decay is the particle decay rate: portion of neighbor value to
	// retain per unit distance.
	Decay float32

	// CSq = C^2
	CSq float32 `display:"-"`

	// Inv2CSq = 1 / 2C^2
	Inv2CSq float32 `display:"-"`

	// MOverHSq = (Mass * C / Hbar)^2 is the mass drag factor in KleinGordon
	// and related equations: the INVERSE REDUCED COMPTON WAVELENGTH squared,
	// in units of 1/cube^2. The update applies it as
	//	vel += CSq * (Laplacian + ... - MOverHSq*psi)
	// which matches Klein-Gordon written as
	//	d2psi/dt2 = c^2 [ Laplacian(psi) - (m c / hbar)^2 psi ]
	// so the C^2 inside MOverHSq is required and is NOT the same as the outer
	// CSq factor -- one converts mass to inverse length, the other converts
	// the whole bracket to an acceleration.
	MOverHSq float32 `display:"-"`

	// HSqOver2M = Hbar^2 / 2 Mass is the factor for Schrodinger's equation.
	HSqOver2M float32 `display:"-"`

	// HEOver2MCSq = (Hbar*e) / (2 Mass * CSq) for computing charge.
	HEOver2MCSq float32 `display:"-"`

	// Omega0 is (Mass * Csq) / Hbar -- the angular velocity for
	// complex-valued oscillator corresponding to the rest mass energy only.
	Omega0 float32 `display:"-"`

	// HOverMC = (Hbar) / (Mass * C) for computing particle momentum from phase.
	HOverMC float32 `display:"-"`

	// MOver2 = Mass / 2 for computing kinetic energy.
	MOver2 float32 `display:"-"`

	// MCSq = (Mass^2 * C^2) for computing total momentum squared
	MCSq float32 `display:"-"`

	// C6M2= (C^6 * Mass^2) is the numerator for computing total particle energy
	C6M2 float32 `display:"-"`

	// Eps0 is epsilon_0, or the permittivity of free space, which weights
	// the impact of charge on the electrical scalar potential = 1 / (mu0 c^2)
	Eps0 float32 `edit:"-"`

	// OneoEps0 = 1 / Eps0
	OneoEps0 float32 `display:"-"`

	// E2OverH = (2 * E) / Hbar
	E2OverH float32 `display:"-"`

	// EOverHSq = E^2 / Hbar^2
	EOverHSq float32 `display:"-"`

	// HiggsMuSq = HiggsMu*HiggsMu, in 1/cube^2 -- the Higgs equivalent of
	// MOverHSq, occupying the same slot in the update.
	HiggsMuSq float32 `display:"-"`

	// HiggsV = HiggsMu / sqrt(HiggsLambda) is the Higgs vacuum expectation
	// value: the radius of the minimum of the potential, in 1/cube. The
	// neutral (lower) doublet component is initialized to this.
	HiggsV float32 `display:"-"`
}

func (pr *Parameters) Update() {
	pr.CSq = pr.C * pr.C
	pr.Inv2CSq = 1.0 / (2 * pr.CSq)
	pr.MOverHSq = (pr.Mass * pr.Mass * pr.CSq) / (pr.Hbar * pr.Hbar)
	if pr.Mass < 0 {
		pr.MOverHSq = -pr.MOverHSq
	}
	hsq := (pr.Hbar * pr.Hbar)
	pr.HSqOver2M = hsq / (2.0 * pr.Mass)
	pr.HEOver2MCSq = (pr.Hbar * pr.E) / (2.0 * pr.Mass * pr.CSq)
	pr.Omega0 = (pr.Mass * pr.CSq) / pr.Hbar
	pr.HOverMC = (0.5 * pr.Hbar) / (pr.Mass * pr.C * math32.Cos(math32.DegToRad(45)))
	pr.MOver2 = pr.Mass / 2.0
	pr.MCSq = pr.Mass * pr.Mass * pr.CSq
	pr.C6M2 = pr.CSq * pr.CSq * pr.MCSq
	pr.Eps0 = 1.0 / (pr.Mu0 * pr.C * pr.C)
	pr.OneoEps0 = 1.0 / pr.Eps0
	pr.E2OverH = (2.0 * pr.E) / pr.Hbar
	pr.EOverHSq = (pr.E * pr.E) / hsq
	pr.HiggsMuSq = pr.HiggsMu * pr.HiggsMu
	if pr.HiggsLambda > 0 {
		pr.HiggsV = pr.HiggsMu / math32.Sqrt(pr.HiggsLambda)
	}
}

//gosl:end

func (pr *Parameters) Defaults() {
	pr.Energy.SetBool(true)
	pr.C = 0.5
	pr.Hbar = 1.0
	pr.Mass = 0.125
	pr.A0NoWave.SetBool(true)
	pr.E = 1.0
	pr.Mu0 = 1.0
	pr.Move.SetBool(true)
	// Standard Model values at HiggsCompton = 16 cubes (see Units.Update):
	// m_h = 1/16 = 0.0625 /cube, mu = m_h/sqrt2, lambda the SM value unchanged.
	// Gives v = 0.1230, m_W Compton 24.9 cubes, m_Z Compton 21.9 cubes.
	pr.HiggsMu = 0.0441942
	pr.HiggsLambda = 0.1291
	pr.Diff = 0.5
	pr.Decay = 0.98
	pr.Update()
}

func (pr *Parameters) ShouldDisplay(field string) bool {
	if ParamsShouldDisplay != nil {
		return slices.Contains(ParamsShouldDisplay, field)
	}
	return true
}
