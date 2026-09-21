// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"

	"cogentcore.org/core/math32"
)

// physical constants in SI units (m, s, kg, A, N, W)
const (
	// C is the speed of light, m/s
	C = 299792458.0

	// Hbar is the normalized Planck constant h / (2 Pi) = J s = m^2 kg / s
	Hbar = 1.054571628e-34

	// eV/c^2 to kg
	EVcsqToKg = 1.7826619216279e-36

	// GeV/c^2 to kg
	GEVcsqToKg = 1.7826619216279e-27

	// E is the unit of electrical charge: C = A s
	E = 1.602176487e-19

	// EMass is the electron rest mass kg
	EMass = 9.10938215e-31

	// Mu0 is the magnetic constant, permeability of free space
	// N/A^2 = (m kg) / (s^2 A^2)
	Mu0 = 4.0 * math32.Pi * 1e-7

	// Eps0 is the electric constant, permittivity of free space = 1 / (mu0 c^2)
	// F/m = (s^4 A^2) / (m^3 kg)
	Eps0 = 1.0 / (Mu0 * C * C)

	// Alpha is the fine structure constant: e^2 / (hbar c 4pi eps0) (dimensionless)
	Alpha = 7.2973525376e-3

	// LambdaE is the Compton wavelength of the electron rest mass:
	// h / (m0 c) = (2 pi hbar) / (m0 c) = 2.42631e-12 m
	LambdaE = (2.0 * math32.Pi * Hbar) / (EMass * C)

	// LambdaBarE is the Compton wavelength of the electron rest mass in hbar
	// hbar / (m0 c) = 3.8615926771197e-13 m
	LambdaBarE = Hbar / (EMass * C)

	// A0 is the Bohr radius in hbar: hbar / (m0 c alpha) = 5.2917720859e-11 m
	A0 = Hbar / (EMass * C * Alpha)

	// HiggsLambda is the quartic coupling of the Higgs potential,
	// m_h^2 / (2 v^2) = 0.1291. In 4 spacetime dimensions lambda is
	// DIMENSIONLESS, so it carries into cube units completely unchanged --
	// provided the Higgs field itself is measured in inverse-length (1/cube)
	// units, the same units as the masses. See [Units.Update].
	HiggsLambda = 0.1291

	// HiggsMuGeV is the mu parameter of the Higgs potential, m_h / sqrt(2),
	// in GeV/c^2. Retained for reference only: it CANNOT be expressed in the
	// electron-anchored cube units, because the Higgs and electron Compton
	// wavelengths differ by a factor of 2.4e5 -- at ComptonE = 16 the Higgs
	// Compton wavelength is 1e-4 of a single cube. The electroweak sector
	// therefore sets its own length scale, via Units.HiggsCompton.
	HiggsMuGeV = 88.47

	// HiggsMhOverV = m_h / v = sqrt(2 * HiggsLambda) = 0.50813 is the only
	// dimensionless number the Higgs potential actually contains. Everything
	// else about it is a choice of units.
	HiggsMhOverV = 0.50813

	// GW is g, the SU(2)_L gauge coupling. Dimensionless, so like
	// HiggsLambda it carries into cube units unchanged.
	GW = 0.6533

	// GpW is g', the U(1)_Y gauge coupling. Dimensionless.
	// sin^2(theta_W) = g'^2/(g^2+g'^2) = 0.2230.
	GpW = 0.3500

	//////// Planck scale constants

	// Planck length, m
	Lp = 1.616252e-35

	// Planck time, s
	Tp = 5.39124e-44

	// Planck mass, kg
	Mp = 2.17644e-8

	// Planck current, A
	Ip = 3.47899e25
)

// Units establishes a consistent set of units for computing
// physics-based [Parameter] factors. Length units are in terms
// of individual cubic elements.
type Units struct {
	// ComptonE is the compton hbar wavelength of an electron in cubic elements,
	// i.e., how many cubes long is the Compton wavelength of the electron.
	// This fixes the length dimension of a cube, as the inverse of this times
	// the numerical value of this quantity (LambdaBarE). It also fixes the EMass.
	ComptonE float64 `default:"16" min:"4"`

	// C is the speed of light in a vacuum in units of cube length / time step.
	// For Dirac waves, 0.5 is the maximum stable value. This fixes the time
	// scale given the length scale from ComptonE.
	C float64

	// Hbar = h / 2pi = reduced Planck constant, which determines the mass scale
	// given length (from ComptonE) and time (from C).
	Hbar float64

	// E is the electric charge constant in cubic units, which determines the
	// electric potential units, C = A s
	// 0.302822 causes Mu0 and Eps0 to both be 1, if C and Hbar are both 1
	E float64

	// EMass is the rest mass of the electron, in cubic units.
	EMass float64 `edit:"-"`

	// Mu0 is the computed Mu0 magnetic constant, permeability of free space
	// N/A^2 = m kg / s^2 A^2
	Mu0 float64 `edit:"-"`

	// Eps0 is the computed Eps0 electric constant, permittivity of free space
	// F/m = (s^4 A^2) / (m^3 kg)
	Eps0 float64 `edit:"-"`

	// HiggsCompton is the reduced Compton wavelength of the Higgs boson,
	// in cubic elements: how many cubes long is hbar / (m_h c). This is the
	// single free scale knob for the electroweak sector, playing the same role
	// there that ComptonE plays for the electron. It is independent of ComptonE
	// because the two scales are 2.4e5 apart and cannot share a lattice.
	// Values of 16 or more keep m_h, m_W and m_Z all well resolved.
	HiggsCompton float64 `default:"16" min:"4"`

	// HiggsMh is the Higgs mass in inverse cube units (1/cube),
	// = 1 / HiggsCompton.
	HiggsMh float64 `edit:"-"`

	// HiggsLambda is the quartic coupling of the Higgs potential. It is
	// dimensionless, so this is just the Standard Model value, unconverted.
	HiggsLambda float64 `edit:"-"`

	// HiggsMu is the mu parameter of the Higgs potential, m_h / sqrt(2),
	// in INVERSE cube units (1/cube) -- an inverse Compton wavelength, not a
	// mass. Its square fills the same slot in the update as (m c / hbar)^2
	// does in Klein-Gordon.
	HiggsMu float64 `edit:"-"`

	// HiggsV is the Higgs vacuum expectation value, mu / sqrt(lambda), in
	// inverse cube units (1/cube). This is the radius of the minimum of the
	// potential, i.e. the value the neutral component settles at.
	HiggsV float64 `edit:"-"`

	// GW is g, the SU(2)_L gauge coupling: dimensionless, hence the
	// Standard Model value unconverted.
	GW float64 `edit:"-"`

	// GpW is g', the U(1)_Y gauge coupling: likewise dimensionless.
	GpW float64 `edit:"-"`

	// MW is the resulting W boson mass, g v / 2, in 1/cube.
	MW float64 `edit:"-"`

	// MZ is the resulting Z boson mass, v sqrt(g^2+g'^2) / 2, in 1/cube.
	MZ float64 `edit:"-"`

	// ThetaW is the weak mixing angle, atan(g'/g), in degrees.
	ThetaW float64 `edit:"-"`

	// CuM is the computed length of a cubic element, in meters.
	CuM float64 `edit:"-"`

	// CuS is the computed duration of a time step update, in seconds.
	CuS float64 `edit:"-"`

	// CuKg is the computed cube unit of mass, in Kg.
	CuKg float64 `edit:"-"`

	// CuN is the computed unit of force, in Newtons: kg m / s^2.
	CuN float64 `edit:"-"`

	// CuJ is the computed unit of energy, in Joules: N m = kg m^2 / s^2.
	CuJ float64 `edit:"-"`

	// CuW is the computed unit of power, in Watts: J / s = kg m^2 / s^3.
	CuW float64 `edit:"-"`

	// CuA is the computed unit of current, in Ampheres: A = C / s; Esi / (E * S).
	CuA float64 `edit:"-"`

	// CuC is the computed cube unit of charge, in Coulombs: C = A * s.
	CuC float64 `edit:"-"`

	// CuV is the computed unit of electrical potential, in Volts: V = W / A = kg m^2.
	CuV float64 `edit:"-"`

	// CuF is the computed unit of capacitance, in Farads = C / V: 1/kg 1/m^2 s^4 A^2
	CuF float64 `edit:"-"`

	//////// SI units: m, kg, s, A

	// Csi is the speed of light, m/s
	Csi float64 `edit:"-"`

	// Hbar is the normalized Planck constant h / (2 Pi) = J s = m^2 kg / s
	HbarSi float64 `edit:"-"`

	// Esi is the unit of electrical charge: C = A s
	Esi float64 `edit:"-"`

	// EMassSi is the electron rest mass kg
	EMassSi float64 `edit:"-"`

	// Mu0 is the magnetic constant, permeability of free space
	// N/A^2 = (m kg) / (s^2 A^2)
	Mu0si float64 `edit:"-"`

	// Eps0 is the electric constant, permittivity of free space = 1 / (mu0 c^2)
	// F/m = (s^4 A^2) / (m^3 kg)
	Eps0si float64 `edit:"-"`

	// Alpha is the fine structure constant: e^2 / (hbar c 4pi eps0) (dimensionless)
	Alpha float64 `edit:"-"`

	// LambdaEsi is the Compton wavelength of the electron rest mass:
	// h / (m0 c) = (2 pi hbar) / (m0 c) = 2.42631e-12 m
	LambdaEsi float64 `edit:"-"`

	// LambdaBarEsi is the Compton wavelength of the electron rest mass in hbar
	// hbar / (m0 c) = 3.8615926771197e-13 m
	LambdaBarEsi float64 `edit:"-"`

	// A0 is the Bohr radius in hbar: hbar / (m0 c alpha) = 5.2917720859e-11 m
	A0si float64 `edit:"-"`
}

func (un *Units) Defaults() {
	un.ComptonE = 16
	un.HiggsCompton = 16
	un.C = 0.5
	un.Hbar = 1
	un.E = 1.0
	un.EMass = 0.125 // for ComptonE = 16
	un.Csi = C
	un.HbarSi = Hbar
	un.Esi = E
	un.EMassSi = EMass
	un.Mu0si = Mu0
	un.Eps0si = Eps0
	un.Alpha = Alpha
	un.LambdaEsi = LambdaE
	un.LambdaBarEsi = LambdaBarE
	un.A0si = A0
	un.Update()
}

func (un *Units) Update() {
	un.CuM = LambdaBarE / un.ComptonE
	un.CuS = (un.CuM * un.C) / C
	un.CuKg = ((un.CuS / (un.CuM * un.CuM)) * Hbar) / un.Hbar
	un.CuN = (un.CuKg * un.CuM) / (un.CuS * un.CuS)
	un.CuJ = un.CuN * un.CuM
	un.CuW = un.CuJ / un.CuS

	un.CuA = E / (un.E * un.CuS) // un.E = Esi / (un.A * un.S); A * E = Esi / S; A = Esi / (E * S)
	un.CuC = un.CuA * un.CuS
	un.CuV = un.CuW / un.CuA
	un.CuF = un.CuC / un.CuV

	un.EMass = EMass / un.CuKg
	un.Mu0 = Mu0 * ((un.CuS * un.CuS * un.CuA * un.CuA) / (un.CuM * un.CuKg))
	un.Eps0 = 1.0 / (un.Mu0 * un.C * un.C)

	// The electroweak sector sets its own length scale. Anchoring it to the
	// electron is impossible: m_h / m_e = 2.4e5, so at ComptonE = 16 the Higgs
	// Compton wavelength would be 1e-4 of one cube -- unresolvable by 4 orders
	// of magnitude. HiggsCompton anchors it directly instead.
	//
	// Measuring the Higgs field in inverse-length (1/cube) units, the same units
	// as the masses, makes lambda dimensionless and g, g' dimensionless too, so
	// every Standard Model coupling passes through with NO conversion at all.
	// Only the one length scale needs setting:
	//
	//	m_h = 1 / HiggsCompton   [1/cube]
	//	mu  = m_h / sqrt(2)      [1/cube]   -- mu^2 goes in the mass slot
	//	v   = mu / sqrt(lambda)  [1/cube]   -- check: m_h / v = sqrt(2 lambda)
	un.HiggsMh = 1.0 / un.HiggsCompton
	un.HiggsLambda = HiggsLambda
	un.HiggsMu = un.HiggsMh / math.Sqrt2
	un.HiggsV = un.HiggsMu / math.Sqrt(un.HiggsLambda)
	un.GW = GW
	un.GpW = GpW
	un.MW = un.GW * un.HiggsV / 2
	un.MZ = un.HiggsV * math.Hypot(un.GW, un.GpW) / 2
	un.ThetaW = math.Atan2(un.GpW, un.GW) * 180 / math.Pi
}
