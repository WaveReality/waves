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

// ewfLeptonJ sums a component of the lepton current over the interior.
func ewfLeptonJ(sz int32, mu int32, comp int) float64 {
	c := GetCtx(0)
	prv := c.PrevState()
	var s float64
	for z := int32(1); z <= sz; z++ {
		for y := int32(1); y <= sz; y++ {
			for x := int32(1); x <= sz; x++ {
				j := EWLeptonCurrent(x, y, z, prv, mu)
				switch comp {
				case 2:
					s += float64(j.Z) // W^3
				case 3:
					s += float64(j.W) // B
				}
			}
		}
	}
	return s
}

// TestEWLeptonCurrentCharge: the charge the leptons SOURCE has to be the same
// charge they FEEL, and for the same reason -- Q = T^3 + Y.
//
// Project the current they put into the gauge fields onto the photon,
// sinTheta_W j^3 + cosTheta_W j^Y. The hypercharges collapse it to
// -e(|e_L|^2 + |e_R|^2), exactly, with nothing from the neutrino. That is the
// other half of TestEWFermionCharge: one says the photon does not push the
// neutrino, this says the neutrino does not make a photon.
func TestEWLeptonCurrentCharge(t *testing.T) {
	const sz = 8
	const amp = 0.5
	for _, tc := range []struct {
		name string
		base EWStates
		want float64 // charge in units of e
	}{
		{"neutrino", EWNu1a, 0},
		{"e_L", EWEL1a, -1},
		{"e_R", EWER1a, -1},
	} {
		ss := ewfSim(sz, func(s *Sim) {
			s.Fill(tc.base, Both, amp)
		})
		ss.StepRun() // so prv holds the field
		p := ss.Params
		sw, cw := float64(p.SinThetaW), float64(p.CosThetaW)
		e := float64(p.GW*p.GpW) / math.Sqrt(float64(p.GW*p.GW+p.GpW*p.GpW))
		// the density the packet has, to normalize against
		n := float64(sz*sz*sz) * amp * amp
		rho := sw*ewfLeptonJ(sz, 0, 2) + cw*ewfLeptonJ(sz, 0, 3)
		got := rho / (e * n)
		t.Logf("%-9s photon-direction charge density %+.6f, as a charge %+.6f", tc.name, rho, got)
		if math.Abs(got-tc.want) > 1e-5 {
			t.Errorf("%s sources charge %+.6f, want %+.1f", tc.name, got, tc.want)
		}
	}
}

// ewfMaxAbs is the largest magnitude of a variable over the interior.
func ewfMaxAbs(sz int32, vr EWStates) float64 {
	c := GetCtx(0)
	cur := int(c.CurState)
	var m float64
	for z := int32(1); z <= sz; z++ {
		for y := int32(1); y <= sz; y++ {
			for x := int32(1); x <= sz; x++ {
				m = math.Max(m, math.Abs(float64(State.Value(int(z), int(y), int(x), int(vr), cur))))
			}
		}
	}
	return m
}

// TestEWSelfFieldNeutrino is the back-reaction seen from the other side, and
// the sharpest statement of neutrality this model can make.
//
// With Params.SelfField on, a lepton sources the gauge fields. Put a lump of
// electron in an empty box and an electromagnetic field appears around it. Put
// a lump of neutrino in the same box and A stays at exactly zero -- not small,
// zero -- because every term that could have sourced it cancels in the
// Q = T^3 + Y combination.
//
// The neutrino does still source the Z, which is the whole reason it interacts
// at all. Neutral is not the same as inert.
func TestEWSelfFieldNeutrino(t *testing.T) {
	const sz = 16
	const nst = 30
	got := map[string]float64{}
	for _, tc := range []struct {
		name  string
		base  EWStates
		wantA bool // does it make an electromagnetic field
	}{
		{"electron", EWEL1a, true},
		{"neutrino", EWNu1a, false},
	} {
		ss := ewfSim(sz, func(s *Sim) {
			s.Params.SelfField.SetBool(true)
			s.Params.EM.SetBool(true)
			s.Params.Update()
			// no Higgs at all: the leptons are the only source in the box
			s.Gauss(tc.base, Both, math32.Vec3(-1, -1, -1), s.Config.PacketWidth, 0.5, 0)
		})
		for range nst {
			ss.StepRun()
		}
		a0 := ewfMaxAbs(sz, EWStates(A0s))
		z0 := ewfMaxAbs(sz, EWZ0)
		t.Logf("%-9s after %d steps: max |A0| %.3e, max |Z0| %.3e", tc.name, nst, a0, z0)
		got[tc.name] = a0
		if tc.wantA && a0 <= 0 {
			t.Errorf("%s sourced no electromagnetic field at all", tc.name)
		}
		if z0 <= 0 {
			t.Errorf("%s sourced no Z field: neutral is not the same as inert", tc.name)
		}
	}
	// The cancellation is exact in the algebra but happens in float32 here,
	// between two terms of order the electron's own field, so what is left is
	// epsilon times that and not a hard zero. Measured against the charged
	// case rather than against nothing.
	r := got["neutrino"] / got["electron"]
	t.Logf("the neutrino's electromagnetic field is %.2e of the electron's", r)
	if r > 1e-5 {
		t.Errorf("the neutrino sourced %.2e of the electron's field: it has no electric charge", r)
	}
}

// TestEWLeptonCurrentVelocity: the flux the leptons source, divided by the
// density, is the speed they travel at.
//
// A left-handed field streams one way and a right-handed one the other, so a
// massive electron's photon current goes as |e_L|^2 - |e_R|^2 against a
// density of |e_L|^2 + |e_R|^2. Their ratio is v/c -- the same relation the
// free Weyl equation gives, arrived at here through the gauge current instead.
func TestEWLeptonCurrentVelocity(t *testing.T) {
	const nst = 100
	ss := ewfSimBox(math32.Vec3i(192, 8, 8), LeptonPackets)
	p := ss.Params
	sw, cw := float64(p.SinThetaW), float64(p.CosThetaW)
	c := float64(p.C)
	ss.StepRun()
	for range nst {
		ss.StepRun()
	}
	sz := ss.Config.Size
	var rho, jx float64
	ctx := GetCtx(0)
	prv := ctx.PrevState()
	for z := int32(1); z <= sz.Z; z++ {
		for y := int32(1); y <= sz.Y; y++ {
			for x := int32(1); x <= sz.X; x++ {
				j0 := EWLeptonCurrent(x, y, z, prv, 0)
				j1 := EWLeptonCurrent(x, y, z, prv, 1)
				rho += sw*float64(j0.Z) + cw*float64(j0.W)
				jx += sw*float64(j1.Z) + cw*float64(j1.W)
			}
		}
	}
	vj := jx / rho
	vg := dispMean(ss.StatVals(StatGroupVelName(EWEMag, math32.X)))
	t.Logf("photon current / density = %.4f, packet group velocity = %.4f c", vj, vg)
	if math.Abs(vj-vg) > 0.05 {
		t.Errorf("the current says the electron moves at %.4f c but the packet moves at %.4f c", vj, vg)
	}
	_ = c
}

// ewfPhiAt is |Phi| at a cell, from the stored doublet.
func ewfPhiAt(x, y, z int32) float64 {
	c := GetCtx(0)
	cur := int(c.CurState)
	return math.Sqrt(float64(State.Value(int(z), int(y), int(x), int(EWHmag), cur)))
}

// TestEWHiggsBackReaction: the Yukawa term runs both ways.
//
// Without it the condensate is an infinite reservoir -- it hands out mass and
// never notices. With Params.SelfField on, a lump of electron pushes |Phi|
// where it sits, and the push is LOCAL to the lump and LINEAR in the coupling,
// which is what -sqrt(2) y (e_R^dag e_L) says it should be.
//
// A neutrino cannot do this at all. The bilinear it would need is e_R^dag
// nu_L, and with no charged Higgs to speak of in the broken vacuum there is
// nothing for it to push on.
//
// The gauge back-reaction plays no part in this: with e_R absent, or the
// couplings zeroed, |Phi| does not move at all. It is the Yukawa term alone.
//
// The push is DOWNWARD, and that is the right sign: the mass term costs energy
// proportional to |Phi|, so a dense enough lump of fermion pays for itself by
// melting the condensate it sits in and becoming lighter. That is the same
// effect that restores the symmetry at high density.
func TestEWHiggsBackReaction(t *testing.T) {
	const sz = 24
	// Two steps, which is one step of motion: EWHmag is written from the
	// position the kernel read, so it lags the update by one. The Higgs starts
	// at rest, so that single step of displacement IS the force -- no
	// propagation, no feedback, no restoring term yet.
	const nst = 2
	const ctr = sz / 2 // as a FULL index: interior coord ctr-1

	var ss0 *Sim
	run := func(yuk float32, self bool, lept EWStates, both bool) (mid, far, v float64) {
		ss := ewfSim(sz, func(s *Sim) {
			HiggsBroken(s)
			s.Params.YukawaE = yuk
			s.Params.SelfField.SetBool(self)
			s.Params.Update()
			// a SMALL lump. The source goes as y times the fermion density,
			// and at the default amplitude of 1 that density is enormous
			// against a condensate of v = 0.123: it drives |Phi| through zero
			// in a few steps. One particle spread over a box is nothing like
			// that dense, so the linear regime is the physical one.
			a := float32(0.01)
			s.Gauss(lept, Both, math32.Vec3(-1, -1, -1), s.Config.PacketWidth, a, 0)
			if both { // the partner the Yukawa needs to have something to pair with
				s.Gauss(EWER1a, Both, math32.Vec3(-1, -1, -1), s.Config.PacketWidth, a, 0)
			}
		})
		ss0 = ss
		v = float64(ss.Params.HiggsV)
		for range nst {
			ss.StepRun()
		}
		return ewfPhiAt(ctr, ctr, ctr), ewfPhiAt(1, 1, 1), v
	}

	// the control: no back-reaction, so the condensate cannot move
	m0, f0, v := run(EWDemoYukawa, false, EWEL1a, true)
	t.Logf("SelfField off: |Phi| at the lump %.6f, in the corner %.6f  (v = %.6f)", m0, f0, v)
	if math.Abs(m0/v-1) > 1e-4 {
		t.Errorf("without back-reaction the condensate moved by %.2e at the lump", m0/v-1)
	}

	// and with it, at one coupling and at double
	m1, f1, _ := run(EWDemoYukawa, true, EWEL1a, true)
	m2, f2, _ := run(2*EWDemoYukawa, true, EWEL1a, true)
	d1, d2 := m1-v, m2-v
	t.Logf("SelfField on,  y = %4.1f: |Phi| at the lump %.6f (%+.3e), corner %.6f (%+.3e)",
		EWDemoYukawa, m1, d1, f1, f1-v)
	t.Logf("SelfField on,  y = %4.1f: |Phi| at the lump %.6f (%+.3e), corner %.6f (%+.3e)",
		2*EWDemoYukawa, m2, d2, f2, f2-v)
	if d1 == 0 {
		t.Fatalf("the electron did not move the condensate at all")
	}
	if d1 > 0 {
		t.Errorf("the condensate rose by %.2e at the lump: a mass term costs energy, so a "+
			"dense fermion should melt the field it sits in, not build it up", d1)
	}
	// The shift must TRACE the fermion bilinear, cell by cell, because that is
	// what the source is. Both e_L and e_R are gaussians of width w, so their
	// product falls as exp(-2 d^2 / w^2) -- and the corner of a box this size
	// is still well inside a lump that wide, which is why it moves at all.
	// ewfPhiAt takes FULL indices, so the two reads sit at interior coords
	// ctr-1 and 0, against a lump centred at interior ctr.
	w := float64(ss0.Config.PacketWidth)
	gw := func(ic float64) float64 {
		d2 := 3 * (ic - ctr) * (ic - ctr) // same offset on all three axes
		return math.Exp(-2 * d2 / (w * w))
	}
	want := gw(0) / gw(ctr-1)
	gotr := (f1 - v) / d1
	t.Logf("corner is %.4f of the peak shift; the fermion bilinear there is %.4f of its value at the peak (w = %.0f)",
		gotr, want, w)
	if math.Abs(gotr/want-1) > 0.03 {
		t.Errorf("the shift falls off as %.4f but the fermion bilinear does as %.4f: "+
			"the source is not the local density", gotr, want)
	}
	// the source is linear in y, so at fixed fermion state doubling y doubles it
	// The source is linear in y, so at a frozen fermion state doubling y
	// doubles it. It is not quite frozen -- doubling y also doubles the mass,
	// so the fermions evolve differently over even these few steps -- which is
	// why this is a few percent over 2 rather than exactly 2.
	r := d2 / d1
	t.Logf("doubling the coupling scaled the push by %.3fx (a source linear in y wants 2)", r)
	if math.Abs(r-2) > 0.2 {
		t.Errorf("the push scaled by %.3fx, not the 2x a source linear in y must give", r)
	}

	// a neutrino has no partner in the broken vacuum: nothing to push with
	mn, _, _ := run(EWDemoYukawa, true, EWNu1a, false)
	t.Logf("neutrino lump: |Phi| %.6f (%+.3e)", mn, mn-v)
	if math.Abs(mn-v) > 0.02*math.Abs(d1) {
		t.Errorf("a neutrino moved the condensate by %.2e: it has no Yukawa partner", mn-v)
	}
}
