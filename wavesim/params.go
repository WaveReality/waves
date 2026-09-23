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

	// WaveC is the first-order COMPLEX wave equation, using the Laplacian, to
	// set against the second-order Wave: two real numbers per point either
	// way, and direction carried by the phase rather than by a velocity.
	WaveC

	// WaveCDir is the first-order DIRECTIONAL version, using the gradient:
	// the factor of the second-order operator that goes one way.
	WaveCDir

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

	// Weyl is the first-order chiral equation on a left- and a right-handed
	// 2-spinor: a massless neutrino in one of them, and an electron when the
	// mass term binds the two together.
	Weyl

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

	// SelfField determines whether the wave sources its own electromagnetic
	// field, which runs MaxwellKernel from the Charge and Current it writes.
	// Off, the field is whatever was put there at init and stays fixed: an
	// EXTERNAL potential, which is the right setup for watching a wave be
	// pushed around by something. Requires EM.
	SelfField slbool.Bool

	// EM determines if EM field coupling is activated.
	EM slbool.Bool

	// A0NoWave applies the neighborhood force directly as a velocity, instead
	// of as a force that adds to the velocity, for the scalar potential, A0.
	// Same form as Sommerfield edge damping. It increases the sensitivity, so
	// C must be less than 1.
	//
	// This is the COULOMB gauge condition, not the Lorenz one: the update
	// becomes Jacobi relaxation toward Laplacian(A0) = -rho/eps0, so A0 is
	// slaved to the charge instead of propagating. Off, A0 obeys the Lorenz
	// wave equation and a bump in it travels at c -- which is the unphysical
	// scalar mode, cancelled in the continuum by the longitudinal one but free
	// to run away on a lattice. Turning this on deletes that sector rather
	// than relying on the cancellation, which is why it is the default.
	//
	// The cost is that the A sector still assumes Lorenz: Laplacian-wave A
	// driven by the full current gives Ampere-Maxwell only when
	// div A + dA0/dt / c^2 = 0. Coulomb for A0 and Lorenz for A agree only
	// where A0 is static and div A constant -- static sources, or a
	// divergence-free current. Outside that the fields are approximate.
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
	HiggsMu float32 `default:"0.0441942"`

	// HiggsLambda is the Higgs potential weight factor that determines
	// effectively how strongly the complex magnitude minus mu_H contributes.
	HiggsLambda float32 `default:"0.1291"`

	// Temp is the temperature T of the thermal bath the Higgs sits in, in the
	// same 1/cube units as HiggsMu. This is added externally to the potential.
	//	mu^2 -> mu^2 - ThermalC * Temp^2
	//
	// Transitioning Temp through TempCrit moves the system between broken
	// and unbroken symmetry, as a function of HiggsV, MW and MZ.
	Temp float32

	// TempCrit = HiggsMu / sqrt(ThermalC) is where the thermal mass cancels
	// mu^2: above it the symmetry is restored and the gauge bosons massless.
	TempCrit float32 `edit:"-"`

	// ThermalC is the coefficient c in the thermal mass mu^2 -> mu^2 - c T^2.
	// In the Standard Model it comes from the particles in the bath:
	//	c = (3 g^2 + g'^2 + 4 y_t^2 + 8 lambda) / 16
	// which is about 0.397, dominated by the top Yukawa y_t.
	ThermalC float32 `default:"0.3973"`

	// GW is g, the SU(2)_L weak isospin gauge coupling, which couples the
	// three W^a fields to the Higgs doublet. Dimensionless. Standard Model
	// value 0.6533. Sets m_W = g v / 2.
	GW float32

	// GpW is g', the U(1)_Y weak hypercharge gauge coupling, which couples
	// the B field to the Higgs doublet. Dimensionless. Standard Model value
	// 0.3500. Together with g it sets the weak mixing angle and m_Z.
	GpW float32

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
	// extra force split (no extra neighbor reads). From Boris 1970, the plasma
	// particle pusher; Qin et al. 2013 show it preserves phase space volume.
	Boris slbool.Bool

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
	// in 1/cube^2. Applied as vel += CSq * (Laplacian + ... - MOverHSq*psi),
	// matching d2psi/dt2 = c^2 [Laplacian(psi) - (m c / hbar)^2 psi]. The C^2
	// inside is NOT the outer CSq: one converts mass to inverse length, the
	// other the bracket to an acceleration.
	MOverHSq float32 `display:"-"`

	// HSqOver2M = Hbar^2 / 2 Mass is the factor for Schrodinger's equation.
	HSqOver2M float32 `display:"-"`

	// HEOverMCSq = (Hbar*e) / (Mass * CSq) is the charge density coefficient,
	// as it appears in the COMPONENT form
	//	rho = HEOverMCSq (phi_b d phi_a - phi_a d phi_b)
	// The covariant form carries a 1/2, but that only cancels the 2 from
	// chi* d chi - chi d chi* = 2i (...), so it has no business here.
	HEOverMCSq float32 `display:"-"`

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

	// SigmaF is the strength (and sign) of the Dirac spin term,
	// sigma . (cB + iE), as e / hbar. It is not free: it is what makes the
	// magnetic moment come out at g = 2.
	SigmaF float32 `display:"-"`

	// EsqOverMCSq = e^2 / (Mass * C^2) is the coefficient of the A0 term in
	// the charge density of the EM-coupled complex KG wave, and with one
	// factor of C removed, of the A term in its current.
	EsqOverMCSq float32 `display:"-"`

	// HiggsMuSq is the EFFECTIVE mu^2 the kernel uses, in 1/cube^2:
	// HiggsMu^2 - ThermalC * Temp^2. The Higgs equivalent of MOverHSq,
	// occupying the same slot in the update. Negative above TempCrit, where
	// the origin becomes the minimum.
	HiggsMuSq float32 `display:"-"`

	// HiggsV = sqrt(HiggsMuSq / HiggsLambda) is the Higgs vacuum expectation
	// value: the radius of the minimum of the potential, in 1/cube. The
	// neutral (lower) doublet component is initialized to this. This is the
	// thermal v(T), falling to zero at TempCrit.
	HiggsV float32 `edit:"-"`

	// MW = GW * HiggsV / 2 is the W boson mass in 1/cube. Nothing in the
	// kernel uses it -- the mass is generated dynamically by the Higgs
	// current. It is here to check that against. Follows HiggsV, so it is the
	// thermal mass and vanishes above TempCrit.
	MW float32 `edit:"-"`

	// MZ = HiggsV * sqrt(GW^2 + GpW^2) / 2 is the Z boson mass in 1/cube,
	// likewise for reference only. The photon mass is zero at any temperature.
	MZ float32 `edit:"-"`

	// SinThetaW, CosThetaW are sin and cos of the weak mixing angle,
	// tan(theta_W) = g'/g, rotating (W^3, B) into the physical (Z, photon):
	//	A_mu = sin(theta_W) W^3_mu + cos(theta_W) B_mu   (massless)
	//	Z_mu = cos(theta_W) W^3_mu - sin(theta_W) B_mu   (mass m_Z)
	// The photon direction is the one Q = T^3 + Y annihilates.
	SinThetaW float32 `display:"-"`
	CosThetaW float32 `display:"-"`

	// WeylQ is the electric charge of the Weyl field, in units of E. It is an
	// INPUT here, not a consequence: set it to 0 and you have a neutrino, to 1
	// and you have an electron, and nothing in this equation prefers either.
	//
	// Neutrality only becomes a RESULT one level up, where Q = T^3 + Y and the
	// photon is the combination of B and W^3 that the doublet's upper
	// component is blind to. There the neutrino's isospin cancels its
	// hypercharge exactly, and e_L and e_R land on the same -1 from different
	// assignments -- which is why electromagnetism does not care about
	// chirality although the weak force does.
	WeylQ float32 `default:"1"`

	// WaveDir is which way the first-order wave equation carries things, +1
	// or -1. A first-order equation has to be told; a second-order one does
	// not, because it does both.
	WaveDir float32 `default:"1"`

	// Dispersion turns the mass term on in the plain Wave equation, which is
	// the whole of what separates it from KleinGordon. Off, the equation has
	// no scale in it -- c is a ratio, not a length -- so omega can only be
	// proportional to k, and every wavelength travels at c. On, the mass
	// supplies the one scale there is, the Compton wavelength, and after that
	// it cannot be proportional to anything.
	//
	// Note WHICH waves it slows. The Laplacian couples a cell to its
	// neighbours and its restoring force is the DIFFERENCE from them, which
	// fades as the wave gets longer; the mass is a spring to ground and its
	// does not. So short waves barely notice and long ones are dominated --
	// the opposite way round from the usual lattice error.
	Dispersion slbool.Bool

	// gosl requires the total struct size to be a multiple of 16 bytes.
	pad, pad1, pad2 float32
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
	pr.HEOverMCSq = (pr.Hbar * pr.E) / (pr.Mass * pr.CSq)
	pr.Omega0 = (pr.Mass * pr.CSq) / pr.Hbar
	pr.HOverMC = (0.5 * pr.Hbar) / (pr.Mass * pr.C * math32.Cos(math32.DegToRad(45)))
	pr.MOver2 = pr.Mass / 2.0
	pr.MCSq = pr.Mass * pr.Mass * pr.CSq
	pr.C6M2 = pr.CSq * pr.CSq * pr.MCSq
	pr.Eps0 = 1.0 / (pr.Mu0 * pr.C * pr.C)
	pr.OneoEps0 = 1.0 / pr.Eps0
	pr.E2OverH = (2.0 * pr.E) / pr.Hbar
	pr.EOverHSq = (pr.E * pr.E) / hsq
	pr.EsqOverMCSq = (pr.E * pr.E) / (pr.Mass * pr.CSq)
	pr.SigmaF = pr.E / pr.Hbar
	// the thermal mass adds to mu^2 with the opposite sign, so raising Temp
	// closes the broken minimum; above TempCrit it is negative and the only
	// minimum is the origin, where v(T) = 0.
	pr.HiggsMuSq = pr.HiggsMu*pr.HiggsMu - pr.ThermalC*pr.Temp*pr.Temp
	pr.TempCrit = 0
	if pr.ThermalC > 0 {
		pr.TempCrit = pr.HiggsMu / math32.Sqrt(pr.ThermalC)
	}
	pr.HiggsV = 0
	if pr.HiggsLambda > 0 && pr.HiggsMuSq > 0 {
		pr.HiggsV = math32.Sqrt(pr.HiggsMuSq / pr.HiggsLambda)
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
	pr.WeylQ = 1
	pr.WaveDir = 1
	pr.Mu0 = 1.0
	pr.Move.SetBool(true)
	// Standard Model values at HiggsCompton = 16 cubes (see Units.Update):
	// m_h = 1/16 = 0.0625 /cube, mu = m_h/sqrt2, lambda the SM value unchanged.
	// Gives v = 0.1230, m_W Compton 24.9 cubes, m_Z Compton 21.9 cubes.
	pr.HiggsMu = 0.0441942
	pr.HiggsLambda = 0.1291
	pr.Temp = 0
	pr.ThermalC = 0.3973 // (3g^2 + g'^2 + 4y_t^2 + 8lambda)/16, SM values
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
