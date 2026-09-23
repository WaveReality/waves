// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/math32"
)

// kgSim builds a 3D complex Klein-Gordon sim.
func kgSim(sz int32, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU = false
	ss.Config.GUI = false
	ss.Config.Equation = KleinGordonC
	ss.Config.Size.Set(sz, sz, sz)
	ss.ConfigSim()
	ss.Params.ThreeD.SetBool(true)
	ss.Params.Edges = EdgesWrap
	ss.Params.Update()
	ss.StateVars = CabStatesN
	ss.ConfigState()
	ss.InitFunc = init
	ss.Init()
	return ss
}

func kgStep(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	ns := int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z)
	if ss.Params.EM.IsTrue() {
		RunMaxwellKernel(ns)
	}
	RunKleinGordonCKernel(ns)
	if ss.Params.Edges == EdgesWrap {
		RunEdgesWrapKernel(int(ctx.EdgesN()))
	}
	ss.RunStats(false)
}

func kgSum(sz int32, vr enums.Enum) float64 {
	return StateSum(GetCtx(0).Size.V(), vr, GetCtx(0).CurState)
}

// TestKGChargeConserved: with no field, the total charge must not change at
// all. This is the property the complex version exists to have -- the real
// scalar KG equation has nothing like it.
func TestKGChargeConserved(t *testing.T) {
	const sz = 8
	ss := kgSim(sz, ChargedPacket)
	kgStep(ss)
	q0 := kgSum(sz, Charge)
	var lo, hi float64 = q0, q0
	for range 2000 {
		kgStep(ss)
		q := kgSum(sz, Charge)
		lo, hi = math.Min(lo, q), math.Max(hi, q)
	}
	t.Logf("free packet: total charge %.6e, drift over 2000 steps %.2e (relative %.2e)",
		q0, hi-lo, (hi-lo)/math.Abs(q0))
	if math.Abs(q0) < 1e-9 {
		t.Fatalf("packet carries no charge at all: %g", q0)
	}
	if (hi-lo)/math.Abs(q0) > 1e-5 {
		t.Errorf("charge not conserved: range %g on %g", hi-lo, q0)
	}
}

// TestKGChargeSign: for a wave at rest the charge is exactly -e |chi|^2 at
// every cell, whatever shape it has, and its sign is the direction the complex
// phase turns and nothing else. Same |chi|^2, opposite charge -- which is the
// only difference between a particle and its antiparticle here.
func TestKGChargeSign(t *testing.T) {
	const sz = 16
	var q, cc [2]float64
	names := []string{"ChargeAtRest", "ChargeAtRestAnti"}
	for i, init := range []func(*Sim){ChargeAtRest, ChargeAtRestAnti} {
		ss := kgSim(sz, init)
		kgStep(ss)
		q[i] = kgSum(sz, Charge)
		cc[i] = kgSum(sz, CabMag)
		sign := 1.0
		if i == 1 {
			sign = -1
		}
		want := -sign * float64(ss.Params.E) * cc[i]
		t.Logf("%-17s total charge %+.5f, want -e |chi|^2 = %+.5f (%+.1e)",
			names[i], q[i], want, q[i]/want-1)
		if math.Abs(q[i]/want-1) > 1e-4 {
			t.Errorf("%s: charge %g, want %g", names[i], q[i], want)
		}
	}
	if q[0]*q[1] >= 0 {
		t.Errorf("the two rotation senses should have opposite charge: %g and %g", q[0], q[1])
	}
	if math.Abs(cc[0]/cc[1]-1) > 1e-6 {
		t.Errorf("they should be the same wave: |chi|^2 %g vs %g", cc[0], cc[1])
	}
	// the flat case, where the analytic value is a single number
	ss := kgSim(6, ChargeUniform)
	kgStep(ss)
	amp := float64(ss.Config.Amplitude)
	want := -float64(ss.Params.E) * amp * amp * 6 * 6 * 6
	got := kgSum(6, Charge)
	t.Logf("ChargeUniform     total charge %+.5f, want -e amp^2 N = %+.5f", got, want)
	if math.Abs(got/want-1) > 1e-4 {
		t.Errorf("uniform charge %g, want %g", got, want)
	}
}

// TestKGContinuity: charge is conserved locally, not just globally --
// d(rho)/dt + div J = 0. A convergence statement, since the discrete
// derivatives only agree to order h^2.
//
// The time derivative is CENTERED: rho is recorded a step either side of the
// J sample, since a forward difference sits half a step later than J and the
// mismatch is first order, which swamps what is being measured.
func TestKGContinuity(t *testing.T) {
	const sz = 64
	snap := func() []float64 {
		v := make([]float64, 0, sz*sz*sz)
		cur := int(GetCtx(0).CurState)
		for z := int32(1); z <= sz; z++ {
			for y := int32(1); y <= sz; y++ {
				for x := int32(1); x <= sz; x++ {
					v = append(v, float64(State.Value(int(z), int(y), int(x), int(Charge), cur)))
				}
			}
		}
		return v
	}
	var res [3]float64
	wls := []float32{4, 8, 16}
	for i, wl := range wls {
		ss := kgSim(sz, func(s *Sim) {
			s.Config.Wavelength = wl
			s.Config.PacketWidth = wl // must fit in the box: edges wrap
			ChargedPacket(s)
		})
		kgStep(ss)
		before := snap()
		kgStep(ss)
		mid := int(GetCtx(0).CurState)
		var resid, scale float64
		k := 0
		for z := int32(1); z <= sz; z++ {
			for y := int32(1); y <= sz; y++ {
				for x := int32(1); x <= sz; x++ {
					var dx, dy, dz float32
					div := float64(Divergence10(x, y, z, int32(CurrentX), int32(mid), &dx, &dy, &dz))
					scale += div * div
					_ = before[k]
					k++
				}
			}
		}
		kgStep(ss)
		after := snap()
		k = 0
		for z := int32(1); z <= sz; z++ {
			for y := int32(1); y <= sz; y++ {
				for x := int32(1); x <= sz; x++ {
					var dx, dy, dz float32
					div := float64(Divergence10(x, y, z, int32(CurrentX), int32(mid), &dx, &dy, &dz))
					drho := 0.5 * (after[k] - before[k])
					r := drho + div
					resid += r * r
					k++
				}
			}
		}
		res[i] = math.Sqrt(resid / scale)
		t.Logf("wavelength %2.0f: RMS(d rho/dt + div J) / RMS(div J) = %.4f", wl, res[i])
	}
	for i := 1; i < len(wls); i++ {
		if res[i] > 0.5*res[i-1] {
			t.Errorf("continuity residual %.4f -> %.4f from wavelength %g to %g, want ~4x better",
				res[i-1], res[i], wls[i-1], wls[i])
		}
	}
}

// kgStepExt advances the wave in a FIXED external field: no Maxwell kernel, so
// A0 and A stay exactly as initialized.
func kgStepExt(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	RunKleinGordonCKernel(int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z))
	if ss.Params.Edges == EdgesWrap {
		RunEdgesWrapKernel(int(ctx.EdgesN()))
	}
	ss.RunStats(false)
}

// TestKGGaugeUniform: a uniform constant A0 is pure gauge -- it multiplies chi
// by a phase and changes no observable. Both |chi|^2 and the total charge must
// come out the same as with no field at all.
//
// The charge is the sharper half. Its free part alone is NOT invariant: the
// phase rotation adds to phi_b d(phi_a) - phi_a d(phi_b). The -e^2 A0 |chi|^2
// term in rho is exactly what cancels that, so this checks that the coupling
// term in the CHARGE matches the coupling term in the EQUATION.
//
// The field has to be set BEFORE the wave, because ChargedUniform reads A0 to
// get the turning rate right; start it at the free rate and the state is a
// superposition of the two frequencies, which beats and is not a gauge
// transform of anything.
func TestKGGaugeUniform(t *testing.T) {
	const sz = 6
	run := func(a0 float32) (q, cc float64) {
		ss := kgSim(sz, func(s *Sim) {
			if a0 != 0 {
				s.Params.EM.SetBool(true)
				s.Params.Update()
			}
			s.Fill(A0s, Both, a0)
			ChargeAtRest(s)
		})
		for range 200 {
			kgStepExt(ss)
		}
		return kgSum(sz, Charge), kgSum(sz, CabMag)
	}
	q0, cc0 := run(0)
	for _, a0 := range []float32{0.001, 0.005, 0.01} {
		q, cc := run(a0)
		t.Logf("A0 = %.3f (e A0 / hbar is %.2f of Omega0): charge %+.5f (%+.1e)   |chi|^2 %.5f (%+.1e)",
			a0, float64(a0)/float64(Params[0].Omega0), q, q/q0-1, cc, cc/cc0-1)
		if math.Abs(q/q0-1) > 1e-3 {
			t.Errorf("A0 = %g changed the charge by %.2e: a uniform A0 is pure gauge", a0, q/q0-1)
		}
		// |chi|^2 drifts a little, growing linearly with A0: a discretization
		// effect of applying the rotation at half steps, not a failure of the
		// invariance. The charge above is the sharper statement.
		if math.Abs(cc/cc0-1) > 2e-3 {
			t.Errorf("A0 = %g changed |chi|^2 by %.2e", a0, cc/cc0-1)
		}
	}
}

// TestKGBorisStability: the A0 coupling turns the velocity (d phi_a, d phi_b)
// at rate 2 e A0 / hbar. An explicit step does not turn it, it spirals out, and
// the wave grows without bound -- the instability web/content/complex-kg.md
// describes and works around by updating the two components in sequence.
// Params.Boris applies the rotation exactly instead.
//
// A packet in a 1/r potential, which is the case the write-up reports. A
// uniform A0 does NOT show it: started in the right eigenstate the explicit
// step survives that quite happily, so the runaway needs a field the wave is
// actually being pushed around by.
func TestKGBorisStability(t *testing.T) {
	const sz = 16
	run := func(a0amp float32, boris bool) (first, last, qrange, q float64) {
		ss := kgSim(sz, func(s *Sim) {
			s.Params.EM.SetBool(true)
			s.Params.Boris.SetBool(boris)
			s.Params.Update()
			s.InvR(A0s, math32.Vec3(-1, -1, -1), a0amp)
			ChargedPacket(s)
		})
		kgStepExt(ss)
		first = kgSum(sz, CabMag)
		q = kgSum(sz, Charge)
		lo, hi := q, q
		for range 3000 {
			kgStepExt(ss)
			c := kgSum(sz, Charge)
			lo, hi = math.Min(lo, c), math.Max(hi, c)
		}
		return first, kgSum(sz, CabMag), hi - lo, q
	}
	for _, a0amp := range []float32{0.05, 0.2} {
		f0, l0, _, _ := run(a0amp, false)
		f1, l1, qr, q := run(a0amp, true)
		t.Logf("1/r potential peak %.2f:  explicit |chi|^2 %.4g -> %.4g (x%.3g)", a0amp, f0, l0, l0/f0)
		t.Logf("                          Boris    |chi|^2 %.4g -> %.4g (x%.3g), charge %.4g varies by %.2g (%.2f%%)",
			f1, l1, l1/f1, q, qr, 100*qr/math.Abs(q))
		if !(l0 > 100*f0 || math.IsNaN(l0)) {
			t.Errorf("expected the explicit step to run away at %v: %g -> %g", a0amp, f0, l0)
		}
		if l1 > 2*f1 || math.IsNaN(l1) {
			t.Errorf("Boris should stay bounded at %v: %g -> %g", a0amp, f1, l1)
		}
		// the write-up notes the total charge is not conserved instant to
		// instant in a field, only on average. It should still stay close.
		if qr/math.Abs(q) > 0.01 {
			t.Errorf("charge varies by %.1f%% in the field, expected under 1%%", 100*qr/math.Abs(q))
		}
	}
}

// atomCtrX is the |wave|^2 weighted X centroid, in interior cube coords.
func atomCtrX(sz int32, mag enums.Enum) float64 {
	var num, den float64
	cur := int(GetCtx(0).CurState)
	for z := int32(1); z <= sz; z++ {
		for y := int32(1); y <= sz; y++ {
			for x := int32(1); x <= sz; x++ {
				v := float64(State.Value(int(z), int(y), int(x), int(mag.Int64()), cur))
				num += v * float64(x)
				den += v
			}
		}
	}
	return num / den
}

// atomRMS is the |wave|^2 weighted rms radius about the center, in cubes: how
// spread out the state is, which is what says whether it is bound.
func atomRMS(sz int32, mag enums.Enum) float64 {
	var num, den float64
	cur := int(GetCtx(0).CurState)
	mid := float64(sz) / 2
	for z := int32(1); z <= sz; z++ {
		for y := int32(1); y <= sz; y++ {
			for x := int32(1); x <= sz; x++ {
				v := float64(State.Value(int(z), int(y), int(x), int(mag.Int64()), cur))
				dx, dy, dz := float64(x-1)-mid, float64(y-1)-mid, float64(z-1)-mid
				num += v * (dx*dx + dy*dy + dz*dz)
				den += v
			}
		}
	}
	return math.Sqrt(num / den)
}

// scalarUnbound is ScalarHydrogen with the well removed: the control that says
// the binding in TestScalarHydrogenBound is doing the work, not the damping.
func scalarUnbound(ss *Sim) {
	p := ss.Params
	p.EM.SetBool(false)
	p.SelfField.SetBool(false)
	p.Mass = BoundStateMass
	p.Edges = EdgesDamp
	p.Update()
	ss.Expo(CabAs, Both, math32.Vec3(-1, -1, -1), ss.Config.HydrogenRadius, ss.Config.Amplitude)
	ss.ChargedBound(1, p.Omega0)
}

// TestScalarHydrogenBound: a spin-0 particle in a Coulomb well stays put. The
// initial shape is the nonrelativistic exp(-r/a), an eigenstate only to order
// (Z alpha)^2, so it breathes -- but it must stay the size of an atom, while
// the same wave with no well to hold it runs away.
func TestScalarHydrogenBound(t *testing.T) {
	const sz = 64
	for _, tc := range []struct {
		name  string
		init  func(*Sim)
		bound bool
	}{
		{"ScalarHydrogen", ScalarHydrogen, true},
		{"ScalarHydrogenP", ScalarHydrogenP, true},
		{"unbound control", scalarUnbound, false},
	} {
		ss := kgSim(sz, tc.init)
		kgStepExt(ss)
		r0 := atomRMS(sz, CabMag)
		for range 200 {
			kgStepExt(ss)
		}
		grow := atomRMS(sz, CabMag)/r0 - 1
		if tc.bound && grow > 0.15 {
			t.Errorf("%s spread by %.1f%%, not bound", tc.name, 100*grow)
		}
		if !tc.bound && grow < 0.5 {
			t.Errorf("%s spread by only %.1f%%, so the control is not a control", tc.name, 100*grow)
		}
		t.Logf("%-16s rms %6.3f -> %6.3f (%+.1f%%)", tc.name, r0, r0*(1+grow), 100*grow)
	}
}

// TestScalarOscillator: the relativistic coherent state swings at the well
// frequency slowed by E / m c^2, because the level spacing is
// hbar omega (m c^2 / E) rather than hbar omega. Dirac must agree exactly:
// with no EM field its spin term is zero and it is two copies of this.
func TestScalarOscillator(t *testing.T) {
	const sz = 48
	var period [2]float64
	for i, tc := range []struct {
		name string
		mag  enums.Enum
		init func(*Sim)
		mk   func(int32, func(*Sim)) *Sim
		step func(*Sim)
	}{
		{"ScalarOscillator", CabMag, ScalarOscillator, kgSim, kgStepExt},
		{"DiracOscillator", DiracMag, DiracOscillator, drSim, drStepExt},
	} {
		ss := tc.mk(sz, tc.init)
		tc.step(ss)
		mid := float64(sz)/2 + 1
		p := ss.Params
		om := 2 * math.Pi / float64(ss.Config.OscillatorPeriod)
		w := math.Sqrt(float64(p.Hbar) / (float64(p.Mass) * om))
		mc2 := float64(p.Mass) * float64(p.CSq)
		e := 1.5*float64(p.Hbar)*om + 0.5*float64(p.Mass)*om*om*4*w*w
		want := float64(ss.Config.OscillatorPeriod) * (mc2 + e) / mc2
		var first, last, n int
		prev := atomCtrX(sz, tc.mag) - mid
		for s := range 800 {
			tc.step(ss)
			d := atomCtrX(sz, tc.mag) - mid
			if prev > 0 && d <= 0 { // downward zero crossing: one per swing
				if n == 0 {
					first = s
				}
				last, n = s, n+1
			}
			prev = d
		}
		if n < 2 {
			t.Fatalf("%s: only %d crossings, it is not swinging", tc.name, n)
		}
		period[i] = float64(last-first) / float64(n-1)
		if math.Abs(period[i]-want)/want > 0.05 {
			t.Errorf("%s period %.1f steps, want %.1f (well period x E / m c^2)", tc.name, period[i], want)
		}
		t.Logf("%-17s period %.1f steps, want %.1f (well says %.0f, slowed by E / m c^2 = %.3f)",
			tc.name, period[i], want, ss.Config.OscillatorPeriod, (mc2+e)/mc2)
	}
	if period[0] != period[1] {
		t.Errorf("Dirac period %.1f but scalar %.1f: with no field the spin term must do nothing",
			period[1], period[0])
	}
}
