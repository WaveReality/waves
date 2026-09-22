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
	ss.Params.Edges = EdgesWrap
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
