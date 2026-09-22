// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"fmt"
	"math"
	"testing"
)

// ewSim builds a minimal native-Go Electroweak sim (no GPU, no GUI).
func ewSim(sz int32) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU = false
	ss.Config.GUI = false
	ss.Config.Equation = Electroweak
	ss.Config.Size.Set(sz, sz, sz)
	ss.ConfigVars()
	ss.Params = &Params[0] // ConfigVars early-returns if the globals already exist
	ss.Units.Defaults()
	ss.Params.Edges = EdgesWrap
	ss.StateVars = EWStatesN
	ss.ConfigState()
	ss.FromUnits()
	ctx := GetCtx(0)
	ctx.Init()
	State.SetZeros()
	HiggsBroken(ss)
	return ss
}

// ewWrap re-wraps the edge ring. Fill and friends only touch interior cells,
// so any perturbation must be followed by this -- otherwise the edges keep the
// old value and Laplacian19 sees a step at the boundary.
func ewWrap() {
	ctx := GetCtx(0)
	RunEdgesWrapKernel(int(ctx.EdgesN()))
}

// ewStep runs just the Higgs kernel plus edge wrapping.
func ewStep(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	ns := int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z)
	RunElectroweakKernel(ns)
	RunEdgesWrapKernel(int(ctx.EdgesN()))
}

func ewGet(v EWStates) float32 {
	ctx := GetCtx(0)
	return State.Value(1, 1, 1, int(v), int(ctx.CurState))
}

// TestHiggsUnits checks the cube-unit relations among mu, lambda, v and m_h.
func TestHiggsUnits(t *testing.T) {
	ss := ewSim(4)
	un := &ss.Units
	mh := math.Sqrt2 * un.HiggsMu
	if math.Abs(mh-un.HiggsMh) > 1e-12 {
		t.Errorf("m_h = sqrt(2)*mu: got %g want %g", mh, un.HiggsMh)
	}
	if math.Abs(mh*un.HiggsCompton-1.0) > 1e-12 {
		t.Errorf("m_h * HiggsCompton should be 1: got %g", mh*un.HiggsCompton)
	}
	// the one dimensionless number the potential contains
	if r := mh / un.HiggsV; math.Abs(r-HiggsMhOverV) > 1e-5 {
		t.Errorf("m_h/v: got %g want sqrt(2 lambda) = %g", r, HiggsMhOverV)
	}
	t.Logf("m_h=%.6f mu=%.6f v=%.6f lambda=%.4f (all per cube); m_h/v=%.5f",
		un.HiggsMh, un.HiggsMu, un.HiggsV, un.HiggsLambda, mh/un.HiggsV)
	mW := 0.6533 * un.HiggsV / 2
	mZ := un.HiggsV * math.Hypot(0.6533, 0.35) / 2
	t.Logf("with g=0.6533 g'=0.35: m_W=%.6f (%.1f cubes) m_Z=%.6f (%.1f cubes)",
		mW, 1/mW, mZ, 1/mZ)
}

// TestHiggsVacuumStatic verifies the VEV is an exact fixed point: at the
// minimum, mu^2 - lambda*mag vanishes and a uniform field has zero Laplacian.
func TestHiggsVacuumStatic(t *testing.T) {
	ss := ewSim(6)
	v0 := ewGet(EWHs0a)
	if math.Abs(float64(v0-ss.Params.HiggsV)) > 1e-7 {
		t.Fatalf("init: hs0a = %g, want HiggsV = %g", v0, ss.Params.HiggsV)
	}
	for range 500 {
		ewStep(ss)
	}
	drift := math.Abs(float64(ewGet(EWHs0a) - v0))
	vel := math.Abs(float64(ewGet(EWHv0a)))
	t.Logf("after 500 steps: hs0a drift = %.3e, velocity = %.3e", drift, vel)
	if drift > 1e-5 || vel > 1e-6 {
		t.Errorf("vacuum not stationary: drift %g vel %g", drift, vel)
	}
	// the charged components must stay exactly zero
	for _, c := range []EWStates{EWHsCa, EWHsCb, EWHs0b} {
		if g := math.Abs(float64(ewGet(c))); g > 1e-9 {
			t.Errorf("component %v drifted to %g", c, g)
		}
	}
}

// freqZC returns the angular frequency (radians per step) from zero crossings.
func freqZC(y []float64) float64 {
	var tc []float64
	for i := range len(y) - 1 {
		if (y[i] < 0) != (y[i+1] < 0) {
			tc = append(tc, float64(i)+(-y[i])/(y[i+1]-y[i]))
		}
	}
	if len(tc) < 2 {
		return 0
	}
	return math.Pi * float64(len(tc)-1) / (tc[len(tc)-1] - tc[0])
}

// TestHiggsRadialMass kicks the radial direction and checks that it rings at
// omega = c * m_h, and kicks a Goldstone direction and checks it does NOT ring.
// That contrast is the whole content of the Mexican hat: the radial direction
// is massive, the phase directions are exactly flat.
func TestHiggsRadialMass(t *testing.T) {
	ss := ewSim(4)
	p := ss.Params
	wantRadial := float64(p.C) * math.Sqrt2 * float64(p.HiggsMu) // c * m_h
	nsteps := int(24 * math.Pi / wantRadial)

	// --- radial: perturb hs0a, the component holding the VEV ---------------
	eps := float32(0.002)
	ss.Fill(EWHs0a, Both, eps)
	ewWrap()
	radial := make([]float64, nsteps)
	for i := range nsteps {
		ewStep(ss)
		radial[i] = float64(ewGet(EWHs0a) - p.HiggsV)
	}
	got := freqZC(radial)
	t.Logf("radial   : omega = %.6f  want c*m_h = %.6f  (ratio %.5f)",
		got, wantRadial, got/wantRadial)
	if math.Abs(got/wantRadial-1) > 0.02 {
		t.Errorf("radial frequency %g, want %g", got, wantRadial)
	}

	// --- Goldstone: perturb a charged component at k=0 ---------------------
	ss2 := ewSim(4)
	ss2.Fill(EWHsCa, Both, eps)
	ewWrap()
	gold := make([]float64, nsteps)
	for i := range nsteps {
		ewStep(ss2)
		gold[i] = float64(ewGet(EWHsCa))
	}
	gf := freqZC(gold)
	lo, hi := gold[0], gold[0]
	for _, g := range gold {
		lo = math.Min(lo, g)
		hi = math.Max(hi, g)
	}
	t.Logf("goldstone: omega = %.6f  want 0 (flat direction); hsCa stays in [%.6e, %.6e] vs eps = %.1e",
		gf, lo, hi, eps)
	if gf != 0 {
		t.Errorf("Goldstone direction oscillated at %g; it must be massless", gf)
	}
	// the phase direction is flat, so hsCa must hold its value
	if math.Abs(hi-float64(eps)) > 1e-5 || math.Abs(lo-float64(eps)) > 1e-5 {
		t.Errorf("Goldstone amplitude moved: [%g, %g], want ~%g", lo, hi, eps)
	}
}

// ewSpectrum kicks one uniform gauge direction and returns its frequency.
// Uniform (k=0) fields have exactly zero Laplacian and zero gradient on any
// lattice, so these mass measurements carry no discretization error at all --
// a 4^3 box gives the same answer as 64^3.
func ewSpectrum(t *testing.T, eps float32, nsteps int,
	kick func(ss *Sim), probe func() float32) (float64, float64) {
	ss := ewSim(4)
	kick(ss)
	ewWrap()
	y := make([]float64, nsteps)
	peak := 0.0
	for i := range nsteps {
		ewStep(ss)
		y[i] = float64(probe())
		peak = math.Max(peak, math.Abs(y[i]))
	}
	return freqZC(y), peak
}

// TestGaugeBosonMasses is the acceptance test for the whole gauge sector.
// No mass term appears anywhere in the kernel: these frequencies are produced
// entirely by the Higgs current evaluated at the vacuum.
func TestGaugeBosonMasses(t *testing.T) {
	p := ewSim(4).Params // also allocates the package globals
	c := float64(p.C)
	g, gp := float64(p.GW), float64(p.GpW)
	n := math.Hypot(g, gp)
	eps := float32(1e-4)
	nsteps := 3000

	t.Logf("v=%.6f  m_W=%.6f  m_Z=%.6f  theta_W=%.4f deg  (all per cube)",
		p.HiggsV, p.MW, p.MZ, math.Atan2(gp, g)*180/math.Pi)

	// --- W: kick W^1 in the X direction -----------------------------------
	wantW := c * float64(p.MW)
	gotW, peakW := ewSpectrum(t, eps, nsteps,
		func(ss *Sim) { ss.Fill(EWW1Xs, Both, eps) },
		func() float32 { return ewGet(EWW1Xv) })
	t.Logf("W       : omega = %.6f  want c*m_W = %.6f  ratio %.5f", gotW, wantW, gotW/wantW)
	if math.Abs(gotW/wantW-1) > 0.01 {
		t.Errorf("W frequency %g, want %g", gotW, wantW)
	}

	// --- Z: the (W^3, B) combination (g, -g')/N ----------------------------
	cw3, cb := float32(g/n), float32(-gp/n)
	wantZ := c * float64(p.MZ)
	gotZ, _ := ewSpectrum(t, eps, nsteps,
		func(ss *Sim) {
			ss.Fill(EWW3Xs, Both, cw3*eps)
			ss.Fill(EWBXs, Both, cb*eps)
		},
		func() float32 { return cw3*ewGet(EWW3Xv) + cb*ewGet(EWBXv) })
	t.Logf("Z       : omega = %.6f  want c*m_Z = %.6f  ratio %.5f", gotZ, wantZ, gotZ/wantZ)
	if math.Abs(gotZ/wantZ-1) > 0.01 {
		t.Errorf("Z frequency %g, want %g", gotZ, wantZ)
	}

	// --- photon: the orthogonal combination (g', g)/N ----------------------
	// Q = T^3 + Y annihilates the vacuum, so this direction must be EXACTLY
	// flat: not approximately massless, but zero force to machine precision.
	aw3, ab := float32(gp/n), float32(g/n)
	gotA, peakA := ewSpectrum(t, eps, nsteps,
		func(ss *Sim) {
			ss.Fill(EWW3Xs, Both, aw3*eps)
			ss.Fill(EWBXs, Both, ab*eps)
		},
		func() float32 { return aw3*ewGet(EWW3Xv) + ab*ewGet(EWBXv) })
	// Compared against the W, which is the same kick in a massive direction.
	// The residual is float32 round-off, not a small mass: any real coupling
	// would show up within a few orders of magnitude of peakW, not eleven.
	t.Logf("photon  : peak |velocity| = %.3e vs W %.3e -- suppressed %.1e x (omega %.4g is noise)",
		peakA, peakW, peakA/peakW, gotA)
	if peakA > 1e-6*peakW {
		t.Errorf("photon direction is not flat: peak velocity %g vs W %g", peakA, peakW)
	}

	t.Logf("check   : m_W/(m_Z cos theta_W) = %.6f  (rho parameter, must be 1)",
		float64(p.MW)/(float64(p.MZ)*math.Cos(math.Atan2(gp, g))))
}

// TestGaugeTimeComponent isolates the time-component (G_0) sector.
//
// A constant uniform B_0 over a vacuum Higgs is PURE GAUGE: it is the same
// physical state as a vacuum whose phase turns at a constant rate. The exact
// solution has D_0 Psi = 0 identically, giving
//
//	d_t Psi = -c * EWGaugeAct(B_0, Psi)
//
// so the neutral component rotates in the (hs0a, hs0b) plane at
//
//	omega = c * g' * Y_Phi * B_0 = c * g' * B_0 / 2
//
// Nothing else in the kernel can produce this motion: every spatial gradient,
// every spatial current and the potential force vanish identically here, so it
// comes entirely from the three G_0 terms. Written out they give
//
//	d2x/dt2 = -2 w dy/dt + w^2 x      (Coriolis + centrifugal)
//	d2y/dt2 = +2 w dx/dt + w^2 y
//
// the rotating-frame equation -- the mechanical statement that a constant A_0
// is a change of frame, not a force.
//
// CAUTION, and the reason this test measures over a short window: that system
// is only MARGINALLY stable (degenerate eigenvalues +-i w), so any explicit
// integrator drives it. Under the kernel's symplectic Euler the radius grows
// exponentially, roughly doubling once per rotation period, because the
// velocity-dependent Coriolis term is not handled symplectically. The
// continuum physics is exact; the integrator is not. Sustained large A_0 over
// many periods is therefore not yet trustworthy -- see the drift column.
func TestGaugeTimeComponent(t *testing.T) {
	// A FIXED drift window across all three omegas: the window must not scale
	// with the period, or the Higgs potential's restoring response (on its own
	// m_h timescale) mixes in differently at each omega and the O(omega^2)
	// signature is lost.
	const drSteps = 64
	var prevDrift float64
	var ratios []float64
	t.Logf("%8s %10s %12s %9s %13s %8s", "B0", "omega", "measured", "ratio", "drift/step", "x4?")
	for k, b := range []float32{0.2, 0.1, 0.05} {
		ss := ewSim(4)
		p := ss.Params
		v := p.HiggsV
		om := float64(p.C) * float64(p.GpW) * float64(b) / 2.0

		ss.Fill(EWB0s, Both, b)
		ss.Fill(EWHv0b, Both, float32(om)*v) // the velocity making D_0 Psi = 0
		ewWrap()

		// Read the rotation rate as a phase ADVANCE between two early times.
		// Differencing cancels the integrator's constant phase offset, and
		// staying well inside one period keeps the instability negligible.
		n1 := int(0.4 / om)
		mag0 := float64(v) * float64(v)
		nrun := 2 * n1
		if nrun < drSteps {
			nrun = drSteps
		}
		var th1, th2, drift float64
		for i := range nrun {
			ewStep(ss)
			switch i + 1 {
			case n1:
				th1 = math.Atan2(float64(ewGet(EWHs0b)), float64(ewGet(EWHs0a)))
			case 2 * n1:
				th2 = math.Atan2(float64(ewGet(EWHs0b)), float64(ewGet(EWHs0a)))
			case drSteps:
				drift = (mag0 - float64(ewGet(EWHmag))) / (mag0 * drSteps)
			}
		}
		got := (th2 - th1) / float64(n1)

		scale := ""
		if k > 0 {
			ratios = append(ratios, prevDrift/drift)
			scale = fmt.Sprintf("%.2f", prevDrift/drift)
		}
		prevDrift = drift
		t.Logf("%8.3f %10.6f %12.6f %9.5f %13.3e %8s", b, om, got, got/om, drift, scale)

		// Tolerance is loose because the scheme itself has an O(omega) phase
		// error, independently reproduced: ratios 1.0103 / 1.0052 / 1.0026 at
		// these three omegas, halving as omega halves. The sharp assertion is
		// the drift scaling below.
		if math.IsNaN(got) || math.Abs(got/om-1) > 0.02 {
			t.Errorf("B0=%g: phase rotation %g, want c*g'*B0/2 = %g", b, got, om)
		}
		// B_0 is pure gauge: in the continuum the current it sees is exactly
		// D_0 Psi = 0. Discretely the half-step offset between psi and psv
		// leaves j_0 at O(omega^2), so B_0 creeps at that order rather than
		// standing perfectly still.
		if d := math.Abs(float64(ewGet(EWB0s)-b)) / float64(b); d > 0.01 {
			t.Errorf("B0=%g drifted %.3g relative; pure gauge must hold it", b, d)
		}
	}
	// The radius error is a discretization artifact, not a physics error, so
	// it must converge away as O(omega^2): halving B_0 quarters it.
	for i, r := range ratios {
		if math.IsNaN(r) || math.Abs(r-4.0) > 0.5 {
			t.Errorf("step %d: radius drift scaled by %.3f when omega halved, want ~4", i, r)
		}
	}
}

// --- Yang-Mills self-coupling ---------------------------------------------

// ymSim is pure Yang-Mills: HiggsMu = 0 makes vfac = -lambda*mag, so Phi = 0
// is an exact fixed point and every Higgs current vanishes identically. What
// is left is the gauge sector alone.
func ymSim(sz int32) *Sim {
	ss := ewSim(sz)
	ss.Params.HiggsMu = 0
	ss.Params.Update()
	State.SetZeros()
	GetCtx(0).Init()
	return ss
}

// ewWBase returns the first state variable of SU(2) component a (1..3);
// the four potentials are at +0..+3 and their velocities at +4..+7.
func ewWBase(a int) int { return int(EWW10s) + (a-1)*8 }

// TestYangMillsPureGauge is the acceptance test for the self-coupling.
//
// A constant uniform W^3_0 = B over a vacuum gauge field is PURE GAUGE: it is
// the gauge transform, by U = exp(i g B c t T^3), of a static uniform W^1_x.
// So the exact solution has (W^1_x, W^2_x) turning in the adjoint 1-2 plane at
//
//	omega = +c * g * B
//
// with |W| constant and W^3_0 standing still. Consistency of BOTH the nu = x
// and the nu = 0 equations is needed for that, and they involve different
// terms, which is what makes this test pin down all the coefficients at once:
//
//	nu = x: transport (-2g) and quartic (-g^2) must cancel to leave -omega^2 W
//	nu = 0: transpose (+g) and quartic (-g^2) must cancel exactly
//
// Get the factor of 2 on transport wrong, or the relative sign of the transpose
// term, and the radius runs away instead of holding. Reversing the assumed
// rotation sense makes the rate wrong by 2x and W^3_0 collapse.
//
// LIMITATION: this test cannot fix the OVERALL sign of the three cubic terms.
// Flipping them all reverses the rotation sense, and the test then passes with
// the opposite omega -- which is how an earlier sign error survived it.
// TestElectroweakForceIsEnergyGradient is what pins that down.
//
// As in TestGaugeTimeComponent the rotating-frame system is only marginally
// stable under an explicit integrator, so the radius error is measured as a
// convergence rate rather than an absolute.
func TestYangMillsPureGauge(t *testing.T) {
	const drSteps = 64
	var prevDrift float64
	var ratios []float64
	r := float32(0.05)
	t.Logf("%8s %11s %12s %9s %13s %8s %12s", "B", "omega", "measured", "ratio", "drift/step", "x4?", "W30 drift")
	for k, b := range []float32{0.02, 0.01, 0.005} {
		ss := ymSim(4)
		p := ss.Params
		om := float64(p.C) * float64(p.GW) * float64(b)
		ss.Fill(EWW30s, Both, b)
		ss.Fill(EWW1Xs, Both, r)
		ss.Fill(EWW2Xv, Both, float32(om)*r)
		ewWrap()

		n1 := int(0.4 / math.Abs(om))
		nrun := 2 * n1
		if nrun < drSteps {
			nrun = drSteps
		}
		r0 := float64(r) * float64(r)
		var th1, th2, drift float64
		for i := range nrun {
			ewStep(ss)
			switch i + 1 {
			case n1:
				th1 = math.Atan2(float64(ewGet(EWW2Xs)), float64(ewGet(EWW1Xs)))
			case 2 * n1:
				th2 = math.Atan2(float64(ewGet(EWW2Xs)), float64(ewGet(EWW1Xs)))
			case drSteps:
				w1, w2 := float64(ewGet(EWW1Xs)), float64(ewGet(EWW2Xs))
				drift = math.Abs((w1*w1+w2*w2)-r0) / (r0 * drSteps)
			}
		}
		got := (th2 - th1) / float64(n1)
		bd := math.Abs(float64(ewGet(EWW30s)-b)) / float64(b)
		scale := ""
		if k > 0 {
			ratios = append(ratios, prevDrift/drift)
			scale = fmt.Sprintf("%.2f", prevDrift/drift)
		}
		prevDrift = drift
		t.Logf("%8.3f %11.6f %12.6f %9.4f %13.3e %8s %11.2e", b, om, got, got/om, drift, scale, bd)

		// tolerance covers the scheme's own O(omega) phase error, as in
		// TestGaugeTimeComponent; the sharp assertion is the drift scaling
		if math.IsNaN(got) || math.Abs(got/om-1) > 0.02 {
			t.Errorf("B=%g: rotation %g, want +c*g*B = %g", b, got, om)
		}
		if bd > 0.02 {
			t.Errorf("B=%g: W^3_0 drifted %.3g relative; pure gauge must hold it", b, bd)
		}
	}
	for i, rr := range ratios {
		if math.IsNaN(rr) || math.Abs(rr-4.0) > 0.6 {
			t.Errorf("step %d: radius drift scaled by %.3f when omega halved, want ~4", i, rr)
		}
	}
}

// TestYangMillsCovariance checks the eps^{abc} structure with NON-uniform
// fields, which the pure-gauge test cannot reach.
//
// A constant SO(3) rotation of the adjoint index is an exact symmetry of the
// Yang-Mills equations, because eps is invariant under SO(3) and the Laplacian
// and gradients act componentwise. So rotating the initial condition, evolving,
// and rotating back must reproduce plain evolution. Using a signed permutation
// (90 degrees about the 3-axis: W1 -> -W2, W2 -> W1) makes the comparison exact
// in floating point rather than merely close.
func TestYangMillsCovariance(t *testing.T) {
	const sz, nst = 6, 40
	fill := func(ss *Sim) {
		n := int(sz) + 2
		for z := range n {
			for y := range n {
				for x := range n {
					for a := 1; a <= 3; a++ {
						for c := range 4 {
							v := 0.03 * math.Sin(float64(a+c)+0.7*float64(x)+
								0.5*float64(y)-0.3*float64(z))
							State.Set(float32(v), z, y, x, ewWBase(a)+c, 0)
							State.Set(float32(v), z, y, x, ewWBase(a)+c, 1)
						}
					}
				}
			}
		}
	}
	// rotate 90 degrees about adjoint axis 3: (W1, W2, W3) -> (-W2, W1, W3)
	rot := func(inv bool) {
		n := int(sz) + 2
		s := 1.0
		if inv {
			s = -1.0
		}
		for z := range n {
			for y := range n {
				for x := range n {
					for c := range 8 {
						w1 := State.Value(z, y, x, ewWBase(1)+c, 0)
						w2 := State.Value(z, y, x, ewWBase(2)+c, 0)
						State.Set(float32(-s)*w2, z, y, x, ewWBase(1)+c, 0)
						State.Set(float32(s)*w1, z, y, x, ewWBase(2)+c, 0)
					}
				}
			}
		}
	}
	// plain evolution
	ss := ymSim(sz)
	fill(ss)
	ewWrap()
	for range nst {
		ewStep(ss)
	}
	plain := make([]float32, 0, 4096)
	cur := int(GetCtx(0).CurState)
	n := int(sz) + 2
	for z := range n {
		for y := range n {
			for x := range n {
				for a := 1; a <= 3; a++ {
					for c := range 8 {
						plain = append(plain, State.Value(z, y, x, ewWBase(a)+c, cur))
					}
				}
			}
		}
	}

	// rotate -> evolve -> rotate back
	ss2 := ymSim(sz)
	fill(ss2)
	rot(false)
	ewWrap()
	for range nst {
		ewStep(ss2)
	}
	// bring the result back into the cur slot layout rot() edits
	cur2 := int(GetCtx(0).CurState)
	worst := 0.0
	i := 0
	for z := range n {
		for y := range n {
			for x := range n {
				for c := range 8 {
					w1 := State.Value(z, y, x, ewWBase(1)+c, cur2)
					w2 := State.Value(z, y, x, ewWBase(2)+c, cur2)
					// inverse rotation: (W1,W2) -> (W2,-W1)
					got := []float32{w2, -w1, State.Value(z, y, x, ewWBase(3)+c, cur2)}
					for a := range 3 {
						worst = math.Max(worst, math.Abs(float64(got[a]-plain[i+a*8+c])))
					}
				}
				i += 24
			}
		}
	}
	t.Logf("global SO(3) covariance over %d steps on a %d^3 lattice: max deviation %.3e", nst, sz, worst)
	if worst > 1e-6 {
		t.Errorf("Yang-Mills not covariant under a constant adjoint rotation: %g", worst)
	}
}

// TestYangMillsSpectrumUnchanged verifies the defining property of the
// self-coupling: every term is quadratic or cubic in W, so all four vanish at
// linear order and cannot shift a boson mass. Turning them on must leave the
// W frequency alone, up to the O(eps^2) correction they genuinely do produce.
func TestYangMillsSpectrumUnchanged(t *testing.T) {
	eps := float32(1e-4)
	nsteps := 3000
	var freq [2]float64
	for i, on := range []bool{false, true} {
		ss := ewSim(4)
		ss.Params.YangMills.SetBool(on)
		ss.Params.Update()
		ss.Fill(EWW1Xs, Both, eps)
		ewWrap()
		y := make([]float64, nsteps)
		for j := range nsteps {
			ewStep(ss)
			y[j] = float64(ewGet(EWW1Xv))
		}
		freq[i] = freqZC(y)
	}
	rel := math.Abs(freq[1]/freq[0] - 1)
	t.Logf("W frequency: YangMills off %.8f, on %.8f -- relative shift %.2e (kick eps = %.0e)",
		freq[0], freq[1], rel, eps)
	if rel > 1e-5 {
		t.Errorf("Yang-Mills terms shifted the W mass by %g; they must vanish at linear order", rel)
	}
}

// TestBorisStability measures what the Boris push buys on the configuration
// that previously destroyed the integrator: a uniform W^3_0 large enough that
// the pure-gauge rotation has a 96-step period.
//
// The velocity-dependent (Coriolis) part of the force is a rotation, and Boris
// applies it exactly instead of integrating it explicitly. That is the
// difference between exponential blow-up and a bounded solution.
//
// It does NOT conserve, and cannot. Writing z = W^1 + i W^2, the pure-gauge
// sector obeys
//
//	d2z/dt2 = -2 i w dz/dt + w^2 z   ->   lambda^2 + 2 i w lambda - w^2 = 0
//
// whose discriminant vanishes: lambda = -i w is a DOUBLE root, so the continuum
// general solution is (A + B t) exp(-i w t). The secular branch is there in the
// exact equations, not put there by the discretization -- it is the gauge
// direction, a gauge transformation whose parameter grows linearly in time.
// No integrator removes it; only fixing the gauge does.
//
// (The Higgs sector looks perfectly conserved under the same test, but that is
// partly the Mexican-hat potential supplying a restoring force toward |Phi| = v
// which the pure gauge sector has nothing equivalent to.)
func TestBorisStability(t *testing.T) {
	const nst = 1200 // ~12 rotation periods
	b, r := float32(0.2), float32(0.05)
	var res [2]float64
	for i, bo := range []bool{false, true} {
		ss := ymSim(4)
		ss.Params.Boris.SetBool(bo)
		ss.Params.Update()
		p := ss.Params
		om := float64(p.C) * float64(p.GW) * float64(b)
		ss.Fill(EWW30s, Both, b)
		ss.Fill(EWW1Xs, Both, r)
		ss.Fill(EWW2Xv, Both, float32(om)*r)
		ewWrap()
		r0 := float64(r) * float64(r)
		worst := 1.0
		for range nst {
			ewStep(ss)
			w1, w2 := float64(ewGet(EWW1Xs)), float64(ewGet(EWW2Xs))
			rr := (w1*w1 + w2*w2) / r0
			if math.IsNaN(rr) {
				worst = math.Inf(1)
				break
			}
			worst = math.Max(worst, rr)
		}
		res[i] = worst
		t.Logf("Boris=%-5v  peak radius^2 / initial over %d steps (~%d periods): %.4g",
			bo, nst, int(float64(nst)*om/(2*math.Pi)), worst)
	}
	if !math.IsInf(res[0], 1) && res[0] < 100 {
		t.Errorf("control did not blow up (peak %.3g); the test is not exercising the instability", res[0])
	}
	if res[1] > 4.0 {
		t.Errorf("Boris peak radius^2 grew to %.3g; expected bounded", res[1])
	}
	t.Logf("improvement: unbounded -> bounded at %.3g", res[1])
}

// TestMixedBasisPropagation checks the mass-eigenstate views written into the
// Maxwell and Z / W^+- state variables, and that each carries the right
// dispersion relation.
//
// The wave is set up directly in the PHYSICAL basis -- the photon direction
// (sin tw, cos tw) or the Z direction (cos tw, -sin tw) in the (W^3, B) plane
// -- and the frequency is read back out of A0s..AZs / EWZ* / EWWPr*, which the
// kernel recomputes from the gauge basis every step. So it exercises the whole
// round trip: physical -> gauge -> evolve -> physical.
//
//	photon:  omega = c * khat                      (massless, phase velocity c)
//	Z:       omega = c * sqrt(khat^2 + m_Z^2)
//	W+-:     omega = c * sqrt(khat^2 + m_W^2)
//
// The Higgs scale is raised so the boson masses are comparable to khat --
// otherwise all three frequencies agree to a fraction of a percent and the test
// would not actually discriminate between a massless photon and a massive one.
//
// The photon staying exactly massless is the sharp part: it is the direction
// Q = T^3 + Y annihilates, and any error in the mixing angle would leak Z mass
// into it and drag the phase velocity below c.
func TestMixedBasisPropagation(t *testing.T) {
	const sz = 8
	eps := float32(1e-4)
	kph := 2 * math.Pi / float64(sz)
	khat := math.Sqrt(float64(lattice2(kph)))

	fill := func(v int32, amp float32) {
		n := sz + 2
		for z := range n {
			for y := range n {
				for x := range n {
					for tt := range 2 {
						State.Set(amp*float32(math.Cos(kph*float64(z))), z, y, x, int(v), tt)
					}
				}
			}
		}
	}

	for _, tc := range []struct {
		name  string
		setup func(p *Parameters, amp float32)
		probe int32
		mass  func(p *Parameters) float64
	}{
		{"photon -> A_x", func(p *Parameters, a float32) {
			fill(int32(EWW3Xs), p.SinThetaW*a)
			fill(int32(EWBXs), p.CosThetaW*a)
		}, int32(AXs), func(p *Parameters) float64 { return 0 }},
		{"Z -> Z_x", func(p *Parameters, a float32) {
			fill(int32(EWW3Xs), p.CosThetaW*a)
			fill(int32(EWBXs), -p.SinThetaW*a)
		}, int32(EWZX), func(p *Parameters) float64 { return float64(p.MZ) }},
		{"W+- -> Re(W+_x)", func(p *Parameters, a float32) {
			fill(int32(EWW1Xs), a)
		}, int32(EWWPrX), func(p *Parameters) float64 { return float64(p.MW) }},
	} {
		ss := ewSim(sz)
		p := ss.Params
		// raise the electroweak scale so m_W, m_Z are comparable to khat
		p.HiggsMu = 0.58
		p.Update()
		State.SetZeros()
		GetCtx(0).Init()
		HiggsBroken(ss)
		tc.setup(p, eps)
		ewWrap()

		m := tc.mass(p)
		want := discFreq(float64(p.C) * math.Sqrt(khat*khat+m*m))
		nst := int(10 * 2 * math.Pi / want)
		y := make([]float64, nst)
		for i := range nst {
			ewStep(ss)
			y[i] = float64(State.Value(1, 1, 1, int(tc.probe), int(GetCtx(0).CurState)))
		}
		got := freqZC(y)
		t.Logf("%-16s omega = %.6f  want %.6f  ratio %.5f   (m = %.4f, khat = %.4f)",
			tc.name, got, want, got/want, m, khat)
		if math.Abs(got/want-1) > 0.01 {
			t.Errorf("%s: omega %g, want %g", tc.name, got, want)
		}
		if m == 0 {
			// undo the leapfrog dispersion to recover the physical phase velocity
			v := 2 * math.Sin(got/2) / khat
			t.Logf("%-16s phase velocity = %.6f vs c = %.6f  (ratio %.6f)  <= massless",
				"", v, float64(p.C), v/float64(p.C))
			if math.Abs(v/float64(p.C)-1) > 1e-3 {
				t.Errorf("photon phase velocity %g, want c = %g", v, p.C)
			}
		}
	}
}

// discFreq is the frequency the leapfrog actually produces for a mode whose
// continuum frequency is w: the update x[n+1] - 2x[n] + x[n-1] = -w^2 x[n] has
// characteristic frequency 2 asin(w/2). At these wavenumbers that is a 0.6%
// correction, well above the 1% tolerance, so it has to be included.
func discFreq(w float64) float64 { return 2 * math.Asin(w/2) }

func lattice2(k float64) float32 {
	return float32(4.0 * math.Sin(k/2) * math.Sin(k/2))
}
