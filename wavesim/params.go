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

	// C is the speed of light factor, in cubes per time step. The stability
	// limit is set by the spectral radius of Laplacian19, which for the
	// isotropic weights is 16/3 (attained at k = (pi,pi,0)), giving
	//	C < 2/sqrt(16/3) = sqrt(3)/2 = 0.8660
	// Note this is TIGHTER than the 1.0 that the older, anisotropic 1/d^2
	// weighting allowed: isotropy costs about 13% of the timestep.
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

	// GW is g, the SU(2)_L weak isospin gauge coupling, which couples the
	// three W^a fields to the Higgs doublet. Dimensionless. Standard Model
	// value 0.6533. Sets m_W = g v / 2.
	GW float32

	// YangMills enables the Yang-Mills self-coupling of the W fields: the
	// terms that make SU(2) genuinely non-abelian rather than three
	// independent copies of U(1). They are quadratic and cubic in W, so they
	// vanish at linear order and change no boson mass; what they add is W
	// self-interaction. Costs 12 extra Gradient10 evaluations per site.
	YangMills slbool.Bool

	// Boris enables the Boris-style push for the velocity-dependent
	// (gauge-connection) part of the force. Those terms generate a rotation of
	// the velocity, and an explicit integrator drives such a system
	// exponentially -- the radius of a pure-gauge rotation doubles about once
	// per period. The Boris step applies that rotation EXACTLY instead, so it
	// is norm-preserving by construction. Second-order accurate, and costs one
	// extra force split (no extra neighbor reads).
	Boris slbool.Bool

	// GpW is g', the U(1)_Y weak hypercharge gauge coupling, which couples
	// the B field to the Higgs doublet. Dimensionless. Standard Model value
	// 0.3500. Together with g it sets the weak mixing angle and m_Z.
	GpW float32

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

	// MW = GW * HiggsV / 2 is the W boson mass in 1/cube. Nothing in the
	// kernel uses it -- the mass is generated dynamically by the Higgs
	// current. It is here to check that against.
	MW float32 `display:"-"`

	// MZ = HiggsV * sqrt(GW^2 + GpW^2) / 2 is the Z boson mass in 1/cube,
	// likewise for reference only. The photon mass is zero.
	MZ float32 `display:"-"`

	// SinThetaW, CosThetaW are sin and cos of the weak mixing angle,
	//	tan(theta_W) = g'/g
	// which rotates the (W^3, B) pair into the physical (Z, photon) pair:
	//	A_mu = sin(theta_W) W^3_mu + cos(theta_W) B_mu   (massless)
	//	Z_mu = cos(theta_W) W^3_mu - sin(theta_W) B_mu   (mass m_Z)
	// The photon direction is exactly the one that Q = T^3 + Y annihilates,
	// which is why it stays massless.
	SinThetaW float32 `display:"-"`
	CosThetaW float32 `display:"-"`
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
	pr.MW = pr.GW * pr.HiggsV / 2.0
	pr.MZ = pr.HiggsV * math32.Sqrt(pr.GW*pr.GW+pr.GpW*pr.GpW) / 2.0
	nw := math32.Sqrt(pr.GW*pr.GW + pr.GpW*pr.GpW)
	if nw > 0 {
		pr.SinThetaW = pr.GpW / nw
		pr.CosThetaW = pr.GW / nw
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
	pr.GW = 0.6533
	pr.GpW = 0.3500
	pr.YangMills.SetBool(true)
	pr.Boris.SetBool(true)
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
