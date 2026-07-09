// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"cogentcore.org/core/math32"
)

// physical constants in SI units (m, s, kg, A, N, W)
const (
	// C is the speed of light, m/s
	C = 299792458.0

	// Hbar is the normalized Planck constant h / (2 Pi) = J s = m^2 kg / s
	Hbar = 1.054571628e-34

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
	// the numerical value of this quantity (LambdaBarE).
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
	un.C = 0.5
	un.Hbar = 1
	un.E = 1.0
	un.EMass = 1
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
	un.CuW = un.CuJ * un.CuM

	un.CuA = E / (un.E * un.CuS) // un.E = Esi / (un.A * un.S); A * E = Esi / S; A = Esi / (E * S)
	un.CuC = un.CuA * un.CuS
	un.CuV = un.CuW / un.CuA
	un.CuF = un.CuC / un.CuV

	un.EMass = EMass / un.CuKg
	un.Mu0 = Mu0 * ((un.CuS * un.CuS * un.CuA * un.CuA) / (un.CuM * un.CuKg))
	un.Eps0 = 1.0 / (un.Mu0 * un.C * un.C)
}
