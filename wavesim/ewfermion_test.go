// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"

	"cogentcore.org/core/math32"
)

// ewfSim builds an electroweak sim with the leptons in it.
func ewfSim(sz int32, init func(*Sim)) *Sim {
	return ewfSimBox(math32.Vec3i(sz, sz, sz), init)
}

// ewfSimBox is ewfSim with an explicit box shape, for the packet runs that
// need room to travel without wrapping into themselves.
func ewfSimBox(sz math32.Vector3i, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU, ss.Config.GUI = false, false
	ss.Config.Equation = Electroweak
	ss.Config.Size = sz
	ss.ConfigSim()
	ss.Params.Edges = EdgesWrap
	ss.Params.Update()
	ss.InitFunc = init
	ss.Init()
	return ss
}

// ewfSums returns the summed staggered magnitudes of the three lepton fields.
func ewfSums() (nu, el, er float64) {
	c := GetCtx(0)
	sz := c.Size.V()
	return StateSum(sz, EWNuMag, c.CurState), StateSum(sz, EWELMag, c.CurState),
		StateSum(sz, EWERMag, c.CurState)
}

// TestEWFermionVsWeyl is the reduction that says the lepton kernel is the Weyl
// kernel with the mass moved into a field. In the broken vacuum the charged
// Higgs is zero and the neutral one is v, so the only Yukawa term left is
// -i (y v / sqrt(2)) between e_L and e_R -- which is exactly Omega0 in
// WeylKernel. Run the same initial spinor under both and the two must agree
// step for step.
//
// The neutrino is the other half of the statement: the same field, in the
// upper component of the doublet, where the Yukawa has no partner for it. It
// must come out of the identical box with its magnitude untouched.
func TestEWFermionVsWeyl(t *testing.T) {
	const sz = 16
	const amp = 0.5
	const nst = 200
	const yuk = 0.8 // a watchable mass, not the real 3e-6

	// State is global, so the two runs cannot be alive at once: record the
	// electroweak trajectory first, then replay it under Weyl.
	ewL := make([]float64, nst)
	ewR := make([]float64, nst)
	ewNu := make([]float64, nst)
	ew := ewfSim(sz, func(s *Sim) {
		HiggsBroken(s)
		s.Params.YukawaE = yuk
		s.Params.Update()
		s.Fill(EWEL1a, Both, amp) // e_L, purely left to start
		s.Fill(EWNu1a, Both, amp) // and a neutrino alongside it
	})
	me := float64(ew.Params.MassE)
	v := float64(ew.Params.HiggsV)
	if me <= 0 {
		t.Fatalf("no electron mass: YukawaE %g, HiggsV %g", ew.Params.YukawaE, ew.Params.HiggsV)
	}
	for i := range nst {
		ew.StepRun()
		ewNu[i], ewL[i], ewR[i] = ewfSums()
	}

	// the same spinor under WeylKernel, with that mass as a constant Omega0
	wyL := make([]float64, nst)
	wyR := make([]float64, nst)
	wy := wySim(sz, func(s *Sim) {
		s.Fill(WeylL1a, Both, amp)
	})
	// Omega0 is a RATE and MassE an inverse length: c is the conversion
	wy.Params.Omega0 = float32(me) * ew.Params.C
	for i := range nst {
		wyStep(wy)
		wyL[i], wyR[i] = wySums()
	}

	el0 := ewL[0]
	nu0 := ewNu[0]
	var worst, nuWorst float64
	for i := range nst {
		worst = math.Max(worst, math.Abs(ewL[i]-wyL[i])/el0)
		worst = math.Max(worst, math.Abs(ewR[i]-wyR[i])/el0)
		nuWorst = math.Max(nuWorst, math.Abs(ewNu[i]/nu0-1))
	}
	t.Logf("m_e = %.5f (1/cube) from y = %.2f and v = %.5f; flip rate c m_e = %.5f", me, yuk, v, me*float64(ew.Params.C))
	t.Logf("after %d steps: e_L %.5f (Weyl %.5f), e_R %.5f (Weyl %.5f)",
		nst, ewL[nst-1], wyL[nst-1], ewR[nst-1], wyR[nst-1])
	t.Logf("electron tracks Weyl to %.2e; neutrino moved by %.2e", worst, nuWorst)
	if ewR[nst-1] <= 0 {
		t.Errorf("the Higgs never made any e_R: the Yukawa term is not doing anything")
	}
	if worst > 1e-5 {
		t.Errorf("the electron does not track WeylKernel at the same mass: off by %.2e", worst)
	}
	if nuWorst > 1e-9 {
		t.Errorf("the neutrino picked up %.2e: with no nu_R the Yukawa has nothing to couple it to", nuWorst)
	}
	_ = math32.X
}

// ewfPhase is the phase angle of a lepton spinor's component 1, summed over
// the box: how far it has rotated.
func ewfPhase(ar, ai EWStates) float64 {
	c := GetCtx(0)
	sz := c.Size.V()
	return math.Atan2(StateSum(sz, ai, c.CurState), StateSum(sz, ar, c.CurState))
}

// TestEWFermionCharge is the point of putting the leptons in here at all.
//
// Nothing in the lepton kernel names an electric charge. What it has is a
// hypercharge per field and a weak isospin per doublet component, and the
// photon is the sinTheta_W W^3 + cosTheta_W B combination. Q = T^3 + Y then
// falls out of the algebra: for the neutrino the two terms are
//
//	g sinTheta_W / 2 - g' cosTheta_W / 2 = (g g' - g' g) / 2 sqrt(g^2+g'^2) = 0
//
// exactly, for any couplings. The neutrino is not neutral because anyone set
// it to zero -- it is neutral because those two numbers cancel.
//
// Both electron chiralities must come out at the SAME rate despite reaching it
// differently: e_L from T^3 = -1/2 with Y = -1/2, e_R from hypercharge alone at
// Y = -1 and no SU(2) at all. If they did not agree there could be no mass
// term joining them.
//
// Run with the Higgs at zero, where there is no Yukawa to mix the chiralities
// and no current to move the gauge fields, so a uniform potential just sits
// there and rotates whatever carries charge.
func TestEWFermionCharge(t *testing.T) {
	const sz = 8
	const amp = 0.5
	const nst = 50
	const a0 = 0.0648 // gives the electron about 0.02 rad per step

	ss := ewfSim(sz, func(s *Sim) {
		s.Params.EM.SetBool(true)
		s.Params.YukawaE = 0 // isolate the gauge response
		s.Params.Update()
		sw, cw := s.Params.SinThetaW, s.Params.CosThetaW
		s.Fill(EWW30s, Both, sw*a0) // a pure photon: no Z admixture
		s.Fill(EWB0s, Both, cw*a0)
		s.Fill(EWNu1a, Both, amp)
		s.Fill(EWEL1a, Both, amp)
		s.Fill(EWER1a, Both, amp)
	})
	p := ss.Params
	// e = g g' / sqrt(g^2 + g'^2) is the electric charge the mixing leaves
	e := float64(p.GW*p.GpW) / math.Sqrt(float64(p.GW*p.GW+p.GpW*p.GpW))
	want := float64(p.C) * e * a0 // the c that goes with c sigma.grad
	// measure BETWEEN two later times, not from rest: the three-level
	// leapfrog is given cur = prv here, which is not its own past, and that
	// leaves a fixed phase offset in the first few steps. It biases a rate
	// taken from zero and cancels out of a difference.
	const skip = 20
	for range skip {
		ss.StepRun()
	}
	nu0 := ewfPhase(EWNu1a, EWNu1b)
	el0 := ewfPhase(EWEL1a, EWEL1b)
	er0 := ewfPhase(EWER1a, EWER1b)
	for range nst {
		ss.StepRun()
	}
	nu := (ewfPhase(EWNu1a, EWNu1b) - nu0) / nst
	el := (ewfPhase(EWEL1a, EWEL1b) - el0) / nst
	er := (ewfPhase(EWER1a, EWER1b) - er0) / nst
	t.Logf("phase rate per step: nu %+.6f, e_L %+.6f, e_R %+.6f   (e A0 = %.6f)", nu, el, er, want)
	t.Logf("as charges: nu %+.4f, e_L %+.4f, e_R %+.4f", nu/want, el/want, er/want)
	if math.Abs(nu) > 1e-6 {
		t.Errorf("the neutrino rotated at %.2e in a photon field: Q = T^3 + Y should cancel exactly", nu)
	}
	if math.Abs(math.Abs(el/want)-1) > 0.005 {
		t.Errorf("e_L came out at charge %.4f, want -1 or +1 in magnitude", el/want)
	}
	if math.Abs(er/el-1) > 1e-6 {
		t.Errorf("the two chiralities have different charge, %.4f vs %.4f: no mass term could join them",
			er/want, el/want)
	}
}

// TestEWLeptonPackets is the demo, measured: the same wave in the two halves
// of one doublet, and the Higgs can only slow one of them down.
//
// A long box, because both packets have to travel several times their own
// width without wrapping into themselves -- a 64 cube is far too small for
// this and reads the resulting self-interference as the packet going
// backwards.
func TestEWLeptonPackets(t *testing.T) {
	const nst = 100
	ss := ewfSimBox(math32.Vec3i(192, 8, 8), LeptonPackets)
	p := ss.Params
	me := float64(p.MassE)
	c := float64(p.C)
	// what the lattice dispersion says each of them should do
	_, vgn := ss.ChiralDispersion(ss.Config.Wavelength, 0)
	_, vge := ss.ChiralDispersion(ss.Config.Wavelength, float32(c*me))
	ss.StepRun() // the Mag variables are written by the kernel, not the init
	nu0, el0, er0 := ewfSums()
	for range nst {
		ss.StepRun()
	}
	nu1, el1, er1 := ewfSums()
	vnu := dispMean(ss.StatVals(StatGroupVelName(EWNuMag, math32.X))) * c
	ve := dispMean(ss.StatVals(StatGroupVelName(EWEMag, math32.X))) * c
	t.Logf("y = %.1f gives m_e = %.4f (1/cube), so the halves trade at c m_e = %.4f per step",
		p.YukawaE, me, c*me)
	t.Logf("neutrino: |nu|^2 %.0f -> %.0f, speed %.4f (lattice says %.4f)", nu0, nu1, vnu, vgn)
	t.Logf("electron: L %.0f -> %.0f, R %.0f -> %.0f, speed %.4f (lattice says %.4f)",
		el0, el1, er0, er1, ve, vge)
	t.Logf("the electron is %.1f%% slower than the neutrino, and nothing but the Higgs did that",
		100*(1-ve/vnu))
	if er0 <= 0 {
		t.Errorf("the electron has no right-handed half: the Yukawa term is not doing anything")
	}
	if math.Abs(vnu/float64(vgn)-1) > 0.1 {
		t.Errorf("the neutrino went at %.4f, want the massless lattice speed %.4f", vnu, vgn)
	}
	if math.Abs(ve/float64(vge)-1) > 0.1 {
		t.Errorf("the electron went at %.4f, want the massive lattice speed %.4f", ve, vge)
	}
	if ve >= 0.9*vnu {
		t.Errorf("the electron at %.4f should visibly lag the neutrino at %.4f: that lag IS the mass",
			ve, vnu)
	}
}

// TestEWLeptonDamp: the leptons must leave through a damped boundary rather
// than bounce off it, like every other first-order field here.
func TestEWLeptonDamp(t *testing.T) {
	const nst = 900
	ss := ewfSimBox(math32.Vec3i(32, 32, 32), LeptonPackets)
	ss.Params.Edges = EdgesDamp
	ss.Params.Update()
	ss.Init()
	ss.StepRun()
	nu0, el0, er0 := ewfSums()
	t0 := nu0 + el0 + er0
	if t0 <= 0 {
		t.Fatalf("nothing in the box to damp")
	}
	for range nst {
		ss.StepRun()
	}
	nu1, el1, er1 := ewfSums()
	left := (nu1 + el1 + er1) / t0
	t.Logf("leptons: %.3f%% left after %d steps (nu %.4f, e_L %.4f, e_R %.4f)",
		100*left, nst, nu1/t0, el1/t0, er1/t0)
	if math.Abs(left) > 0.01 {
		t.Errorf("%.3f%% of the leptons are still in the box: they are reflecting", 100*left)
	}
}
