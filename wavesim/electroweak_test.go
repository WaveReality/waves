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
	ss.ElectroweakInit()
	return ss
}

// ewWrap re-wraps the edge ring. Fill and friends only touch interior cells,
// so any perturbation must be followed by this -- otherwise the edges keep the
// old value and Laplacian26 sees a step at the boundary.
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
