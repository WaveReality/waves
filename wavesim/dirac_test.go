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

func drSim(sz int32, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU = false
	ss.Config.GUI = false
	ss.Config.Equation = Dirac
	ss.Config.Size.Set(sz, sz, sz)
	ss.ConfigSim()
	ss.Params.ThreeD.SetBool(true)
	ss.Params.Update()
	ss.StateVars = DiracStatesN
	ss.ConfigState()
	ss.InitFunc = init
	ss.Init()
	return ss
}

// drStepExt runs the Dirac kernel alone: a FIXED external field.
func drStepExt(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	RunDiracKernel(int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z))
	if ss.Params.Edges == EdgesWrap {
		RunEdgesWrapKernel(int(ctx.EdgesN()))
	}
	ss.RunStats(false)
}

func drSum(sz int32, vr enums.Enum) float64 {
	return StateSum(GetCtx(0).Size.V(), vr, GetCtx(0).CurState)
}

// drSpin returns the total <sigma_x>, <sigma_y>, <sigma_z> over the box.
func drSpin(sz int32) (sx, sy, sz2 float64) {
	cur := int(GetCtx(0).CurState)
	for z := int32(1); z <= sz; z++ {
		for y := int32(1); y <= sz; y++ {
			for x := int32(1); x <= sz; x++ {
				a1 := float64(State.Value(int(z), int(y), int(x), int(Dirac1As), cur))
				b1 := float64(State.Value(int(z), int(y), int(x), int(Dirac1Bs), cur))
				a2 := float64(State.Value(int(z), int(y), int(x), int(Dirac2As), cur))
				b2 := float64(State.Value(int(z), int(y), int(x), int(Dirac2Bs), cur))
				sx += 2 * (a1*a2 + b1*b2)
				sy += 2 * (a1*b2 - b1*a2)
				sz2 += (a1*a1 + b1*b1) - (a2*a2 + b2*b2)
			}
		}
	}
	return
}

// drRestSpinX puts a uniform spinor at rest with its spin along +x.
func drRestSpinX(ss *Sim, amp float32) {
	h := amp / float32(math.Sqrt2)
	ss.Fill(Dirac1As, Both, h)
	ss.Fill(Dirac2As, Both, h)
	ss.Fill(Dirac1Bv, Both, ss.Params.Omega0*h)
	ss.Fill(Dirac2Bv, Both, ss.Params.Omega0*h)
}

// TestDiracLarmor: the spin term's strength is not a free parameter. In a
// uniform B along z, the spin must precess at the Larmor frequency
//
//	omega = g e B / (2 m c),  with g = 2
//
// which is the second-order Dirac equation's best known prediction, and the
// reason the sigma . F term has the coefficient it has.
//
// The field is set directly into BZ rather than made from a vector potential:
// that isolates the spin term from the orbital ones, which is what is being
// measured. A = A0 = 0, so nothing else in the equation is active.
func TestDiracLarmor(t *testing.T) {
	const sz = 4
	for _, b := range []float32{3e-4, 6e-4} {
		ss := drSim(sz, func(s *Sim) {
			s.Params.EM.SetBool(true)
			s.Params.Update()
			s.Fill(BZ, Both, b)
			drRestSpinX(s, 1)
		})
		p := ss.Params
		want := float64(p.E) * float64(b) / (float64(p.Mass) * float64(p.C))
		// track the phase of (sx, sy) and unwrap
		prev := 0.0
		total := 0.0
		nst := 3000
		for range nst {
			drStepExt(ss)
			sx, sy, _ := drSpin(sz)
			ph := math.Atan2(sy, sx)
			d := ph - prev
			for d > math.Pi {
				d -= 2 * math.Pi
			}
			for d < -math.Pi {
				d += 2 * math.Pi
			}
			total += d
			prev = ph
		}
		got := total / float64(nst)
		t.Logf("B = %.1e: precession %+.5e rad/step, Larmor g=2 is %+.5e, g = %.4f",
			b, got, want, 2*math.Abs(got)/want)
		if math.Abs(math.Abs(got)/want-1) > 0.05 {
			t.Errorf("B = %g: precession %g, want %g for g = 2", b, got, want)
		}
	}
}

// TestDiracCharge: the same conserved charge as the complex KG version, now
// summed over both components. Exact with no field.
func TestDiracCharge(t *testing.T) {
	const sz = 16
	ss := drSim(sz, SpinAtRest)
	drStepExt(ss)
	q0 := drSum(sz, Charge)
	cc := drSum(sz, DiracMag)
	want := -float64(ss.Params.E) * cc
	t.Logf("SpinAtRest: total charge %+.5f, want -e |psi|^2 = %+.5f (%+.1e)", q0, want, q0/want-1)
	if math.Abs(q0/want-1) > 1e-4 {
		t.Errorf("charge %g, want %g", q0, want)
	}
	lo, hi := q0, q0
	for range 2000 {
		drStepExt(ss)
		q := drSum(sz, Charge)
		lo, hi = math.Min(lo, q), math.Max(hi, q)
	}
	t.Logf("drift over 2000 steps %.2e (relative %.2e)", hi-lo, (hi-lo)/math.Abs(q0))
	if (hi-lo)/math.Abs(q0) > 1e-5 {
		t.Errorf("charge not conserved: range %g on %g", hi-lo, q0)
	}
}

// TestDiracFreeNoSpin: with no field the equation is four uncoupled KG waves,
// so the spin direction must not move at all. This is the write-up's own
// puzzle -- the second-order form has no spin term of its own -- stated as a
// test, and it is the expected answer rather than a defect.
func TestDiracFreeNoSpin(t *testing.T) {
	const sz = 8
	ss := drSim(sz, func(s *Sim) {
		s.DiracBlob(math32.Vec3(-1, -1, -1), s.Config.PacketWidth, s.Config.Amplitude, math32.X, 1)
	})
	drStepExt(ss)
	sx0, sy0, sz0 := drSpin(sz)
	n0 := math.Sqrt(sx0*sx0 + sy0*sy0 + sz0*sz0)
	for range 1000 {
		drStepExt(ss)
	}
	sx, sy, sz2 := drSpin(sz)
	n := math.Sqrt(sx*sx + sy*sy + sz2*sz2)
	dot := (sx*sx0 + sy*sy0 + sz2*sz0) / (n * n0)
	t.Logf("free: spin direction dot product after 1000 steps = %.8f (|S| %.4g -> %.4g)", dot, n0, n)
	if dot < 1-1e-6 {
		t.Errorf("a free particle has no spin term: direction moved, dot = %g", dot)
	}
}

// TestMovingWavePacket: the rewritten packet must actually travel, at the
// lattice group velocity, instead of splitting into two halves going opposite
// ways. A split packet has a centroid that barely moves, so the speed is a
// decisive test of it.
func TestMovingWavePacket(t *testing.T) {
	const sz = 64
	ss := drSim(sz, func(s *Sim) {
		s.Config.Wavelength = 16
		s.Config.PacketWidth = 16
		s.MovingWavePacketConfig(Dirac1As, Dirac1Av, math32.X, math32.Vec3(-1, -1, -1), 1, 0, 1)
	})
	p := ss.Params
	k := 2 * math.Pi / float64(ss.Config.Wavelength)
	kh := 2 * math.Sin(k/2)
	m := math.Sqrt(float64(p.MOverHSq))
	want := kh * math.Cos(k/2) / math.Sqrt(kh*kh+m*m) // in units of c
	ctr := func() float64 {
		var num, den float64
		cur := int(GetCtx(0).CurState)
		c := int(sz / 2)
		for x := int32(1); x <= sz; x++ {
			v := float64(State.Value(c, c, int(x), int(Dirac1As), cur))
			num += v * v * float64(x)
			den += v * v
		}
		return num / den
	}
	drStepExt(ss)
	c0 := ctr()
	nst := 30 // short enough that the packet cannot reach the wrapped edge
	for range nst {
		drStepExt(ss)
	}
	got := (ctr() - c0) / float64(nst) / float64(p.C)
	t.Logf("packet centroid %.2f -> %.2f over %d steps: v = %.4f c, group velocity %.4f c",
		c0, ctr(), nst, got, want)
	if math.Abs(got/want-1) > 0.15 {
		t.Errorf("packet speed %g c, want %g c: a split packet barely moves", got, want)
	}
}

// TestSpinInPotential: the well must actually pull the lump toward it, and the
// run must survive. An external field, so A0 stays exactly as set.
func TestSpinInPotential(t *testing.T) {
	const sz = 32
	ss := drSim(sz, func(s *Sim) {
		s.Config.PacketWidth = 4
		SpinInPotential(s)
	})
	if ss.Params.SelfField.IsTrue() {
		t.Errorf("SpinInPotential must leave SelfField off: the field is external")
	}
	ctr := func() float64 {
		var num, den float64
		cur := int(GetCtx(0).CurState)
		c := int(sz / 2)
		for x := int32(1); x <= sz; x++ {
			v := float64(State.Value(c, c, int(x), int(DiracMag), cur))
			num += v * float64(x)
			den += v
		}
		return num / den
	}
	drStep := func() {
		ctx := GetCtx(0)
		ctx.StepInc()
		RunDiracKernel(int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z))
		RunMaxwellDampKernel(int(ctx.EdgesN()))
		ss.RunStats(false)
	}
	drStep()
	c0 := ctr()
	a00 := float64(State.Value(int(sz/2), int(sz/2), int(sz/2)+3, int(A0s), int(GetCtx(0).CurState)))
	for range 1500 {
		drStep()
	}
	c1 := ctr()
	a01 := float64(State.Value(int(sz/2), int(sz/2), int(sz/2)+3, int(A0s), int(GetCtx(0).CurState)))
	well := float64(sz) / 2
	t.Logf("lump centroid %.2f -> %.2f (well at %.1f), A0 probe %.5f -> %.5f", c0, c1, well, a00, a01)
	if math.IsNaN(c1) {
		t.Fatalf("blew up")
	}
	if math.Abs(a01-a00) > 1e-9 {
		t.Errorf("the external field moved: %g -> %g", a00, a01)
	}
	if c1 >= c0 {
		t.Errorf("the well should pull the lump in: centroid %g -> %g, well at %g", c0, c1, well)
	}
}

// TestDiracHydrogenBound: an electron in a Coulomb well stays bound, and the
// spin term is actually live while it does.
//
// That second half is the point of the test. The kernel reads E and B as
// state, and MaxwellKernel is what fills them -- but it does not run with
// SelfField off, so a well written straight into A0 carries no E field unless
// something puts one there. Without it sigma . (cB + iE) is identically zero
// and the Dirac equation collapses into two independent Klein-Gordon ones.
func TestDiracHydrogenBound(t *testing.T) {
	const sz = 64
	for _, tc := range []struct {
		name string
		init func(*Sim)
	}{{"DiracHydrogen", DiracHydrogen}, {"DiracHydrogenP", DiracHydrogenP}} {
		ss := drSim(sz, tc.init)
		drStepExt(ss)
		cur := GetCtx(0).CurState
		ez := StateMax(GetCtx(0).Size.V(), EZ, cur)
		if ez <= 0 {
			t.Errorf("%s: E field is zero, so the spin term does nothing", tc.name)
		}
		r0 := atomRMS(sz, DiracMag)
		worst := 0.0
		for i := range 200 {
			drStepExt(ss)
			if (i+1)%20 == 0 { // sampling: the rms sweep costs more than a step
				worst = math.Max(worst, atomRMS(sz, DiracMag)/r0-1)
			}
		}
		if worst > 0.4 {
			t.Errorf("%s spread by %.1f%%, not bound", tc.name, 100*worst)
		}
		t.Logf("%-15s rms %6.3f, spreads at most %+.1f%%, max E %.4g", tc.name, r0, 100*worst, ez)
	}
}

// drChiralUniform is a proper rest state on a uniform field: the control for
// TestDiracChirality, which must come out an exact 50/50 of the chiralities.
func drChiralUniform(ss *Sim) {
	ss.Params.EM.SetBool(false)
	ss.Params.Edges = EdgesWrap
	ss.Params.Update()
	ss.Fill(Dirac1As, Both, ss.Config.Amplitude)
	ss.DiracRest(1)
}

// TestDiracChirality checks the left-chiral spinor the kernel recovers, and
// with it the claim that an electron is two massless waves bound by mass.
//
// Two things have to hold. A particle AT REST is an exact 50/50 mixture, since
// chirality is not helicity. And a state started purely right-chiral converts
// ENTIRELY into the left one and back, at the rest mass frequency, with the
// sum conserved -- which is what the mass term does and all it does.
func TestDiracChirality(t *testing.T) {
	const sz = 16
	sums := func() (l, r float64) {
		c := GetCtx(0)
		return StateSum(c.Size.V(), DiracLMag, c.CurState), StateSum(c.Size.V(), DiracMag, c.CurState)
	}

	rs := drSim(sz, drChiralUniform)
	drStepExt(rs)
	l, r := sums()
	if math.Abs(l/r-1) > 0.01 {
		t.Errorf("at rest |psi_L|^2 / |psi_R|^2 = %.5f, want 1: chirality is not helicity", l/r)
	}
	t.Logf("at rest  |psi_L|^2 / |psi_R|^2 = %.5f", l/r)

	ss := drSim(sz, ChiralOscillation)
	drStepExt(ss)
	l, r = sums()
	tot := l + r
	if l/tot > 0.01 {
		t.Errorf("starts %.2f%% left-chiral, should be purely right", 100*l/tot)
	}
	// half a period of |psi|^2 is where the right half has gone entirely away
	want := math.Pi / float64(ss.Params.Omega0)
	lo, hi, rmin, at := tot, tot, r, 0
	for i := range int(1.2 * want) {
		drStepExt(ss)
		l, r = sums()
		lo, hi = math.Min(lo, l+r), math.Max(hi, l+r)
		if r < rmin {
			rmin, at = r, i+1
		}
	}
	if rmin/tot > 0.01 {
		t.Errorf("right half only ever falls to %.2f%%, so it does not fully convert", 100*rmin/tot)
	}
	if got := 2 * float64(at); math.Abs(got-want)/want > 0.05 {
		t.Errorf("conversion period %.1f steps, want pi hbar / m c^2 = %.1f", got, want)
	}
	if (hi-lo)/tot > 0.01 {
		t.Errorf("total varies by %.3f%%, but |psi_L|^2 + |psi_R|^2 must be conserved", 100*(hi-lo)/tot)
	}
	t.Logf("right half falls to %.3f%% at step %d, so the period is %d vs pi hbar / m c^2 = %.1f; total varies %.4f%%",
		100*rmin/tot, at, 2*at, want, 100*(hi-lo)/tot)
}
