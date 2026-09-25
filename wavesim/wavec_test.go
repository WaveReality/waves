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

// wcSimEq builds either complex wave equation.
func wcSimEq(sz int32, threeD bool, eq Equations, dir float32, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU, ss.Config.GUI = false, false
	ss.Config.Equation = eq
	if threeD {
		ss.Config.Size.Set(sz, sz, sz)
	} else {
		ss.Config.Size.Set(sz, 1, 1)
	}
	ss.ConfigSim()
	ss.Params.ThreeD.SetBool(threeD)
	ss.Params.WaveDir = dir
	ss.Params.Update()
	ss.ConfigState()
	ss.InitFunc = init
	ss.Init()
	return ss
}

// wcRMS is the root mean square of a variable over the interior.
func wcRMS(vr enums.Enum) float64 {
	var s float64
	c := GetCtx(0)
	cur := int(c.CurState)
	n := 0
	for z := int32(1); z <= c.Size.Z; z++ {
		for y := int32(1); y <= c.Size.Y; y++ {
			for x := int32(1); x <= c.Size.X; x++ {
				f := float64(State.Value(int(z), int(y), int(x), int(vr.Int64()), cur))
				s += f * f
				n++
			}
		}
	}
	return math.Sqrt(s / float64(n))
}

func wcSim(sz int32, threeD bool, dir float32, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU, ss.Config.GUI = false, false
	ss.Config.Equation = WaveCDir
	if threeD {
		ss.Config.Size.Set(sz, sz, sz)
	} else {
		ss.Config.Size.Set(sz, 1, 1)
	}
	ss.ConfigSim()
	ss.Params.ThreeD.SetBool(threeD)
	ss.Params.WaveDir = dir
	ss.Params.Update()
	ss.StateVars = WaveCStatesN
	ss.ConfigState()
	ss.InitFunc = init
	ss.Init()
	return ss
}

func wcStep(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	RunWaveCDirKernel(int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z))
	if ss.Params.Edges == EdgesWrap {
		RunEdgesWrapKernel(int(ctx.EdgesN()))
	}
	ss.RunStats(false)
}

// wcCtr is the |psi|^2 weighted X centroid and the total.
func wcCtr() (float64, float64) {
	var num, den float64
	c := GetCtx(0)
	cur := int(c.CurState)
	for z := int32(1); z <= c.Size.Z; z++ {
		for y := int32(1); y <= c.Size.Y; y++ {
			for x := int32(1); x <= c.Size.X; x++ {
				v := float64(State.Value(int(z), int(y), int(x), int(WaveCMag), cur))
				num += v * float64(x)
				den += v
			}
		}
	}
	return num / den, den
}

// TestWaveC: the first-order complex wave equation, against which the
// second-order WaveKernel is meant to be read.
//
// Everything in it leaves in ONE piece, in the one direction Params.WaveDir
// names, in 1D and 3D alike, and that is the claim being tested. The pulse
// matters most: it has no phase to tell it where to go, and it still goes,
// where the second-order equation would split it in half because its operator
// contains both factors. It travels at exactly c, and is held to that.
//
// The PACKET is not held to c, only to travelling one way, subluminally, with
// its total conserved. A carrier of finite wavelength sees the lattice: the
// gradient stencil and the leapfrog each bend the dispersion, and the envelope
// comes out at 0.996 c at C = 0.5, falling to 0.765 c by C = 0.25. That is not
// the k-spread of the envelope -- widening the packet eightfold does not move
// it -- and it does not match dw/dk for sin(w) = C sin(k) either, which is the
// relation the scheme should obey. Writing the past from that relation rather
// than from c, as WeylPacket does, changes the answer by under a percent, so
// the initialization is not the cause. Unexplained; the bound here is loose on
// purpose rather than pinning a number nobody has accounted for.
func TestWaveC(t *testing.T) {
	for _, threeD := range []bool{false, true} {
		sz, n, tol := int32(128), 20, 0.01
		if threeD {
			sz, n, tol = 64, 10, 0.05
		}
		for _, tc := range []struct {
			name string
			init func(*Sim)
			atC  bool // held to exactly c
		}{{"packet", WaveCDirPacket, false}, {"pulse", WaveCDirPulse, true}} {
			for _, dir := range []float32{1, -1} {
				ss := wcSim(sz, threeD, dir, tc.init)
				wcStep(ss)
				x0, q0 := wcCtr()
				for range n {
					wcStep(ss)
				}
				x1, q1 := wcCtr()
				v := (x1 - x0) / float64(n)
				c := float64(ss.Params.C)
				want := float64(dir) * c
				frac := v / want // of c, in the intended direction
				switch {
				case tc.atC:
					if math.Abs(v-want)/math.Abs(want) > tol {
						t.Errorf("%v 3D=%v dir %+.0f: speed %+.4f, want %+.4f -- a wave that split would barely move at all",
							tc.name, threeD, dir, v, want)
					}
				case frac < 0.5:
					// a packet that split would sit near zero, and one going
					// the wrong way would come out negative
					t.Errorf("%v 3D=%v dir %+.0f: speed %+.4f is %+.3f of c in the intended direction -- it should travel one way, whole",
						tc.name, threeD, dir, v, frac)
				case frac > 1.01:
					t.Errorf("%v 3D=%v dir %+.0f: speed %+.4f is %+.3f c, and nothing here may outrun c",
						tc.name, threeD, dir, v, frac)
				}
				if math.Abs(q1/q0-1) > 1e-5 {
					t.Errorf("%v 3D=%v dir %+.0f: total changed by %.2e", tc.name, threeD, dir, q1/q0-1)
				}
				t.Logf("3D=%-5v %-6s dir %+.0f: speed %+.4f = %+.3f c, total held",
					threeD, tc.name, dir, v, v/c)
			}
		}
	}
}

func wcPlainStep(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	ss.RunWaveC(int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z))
	if ss.Params.Edges == EdgesWrap {
		RunEdgesWrapKernel(int(ctx.EdgesN()))
	}
	ss.RunStats(false)
}

// TestWaveCPlain is the comparison the equation exists for: the first-order
// complex wave against the second-order real one.
//
// Both carry two real numbers per point. The complex pair are the same kind of
// thing on the same scale, so their rms is EQUAL; the second-order pair are a
// quantity and its rate, so theirs differ by roughly omega. And nothing in
// this equation says which way to go -- the packet follows its phase, so a
// pulse with no phase does not move at all.
func TestWaveCPlain(t *testing.T) {
	for _, threeD := range []bool{false, true} {
		sz := int32(128)
		if threeD {
			sz = 64
		}
		for _, dir := range []float32{1, -1} {
			ss := wcSimEq(sz, threeD, WaveC, dir, WaveCPacket)
			ra, rb := wcRMS(WaveCa), wcRMS(WaveCb) // at init, before anything moves
			wcPlainStep(ss)
			x0, q0 := wcCtr()
			if math.Abs(ra/rb-1) > 0.05 {
				t.Errorf("3D=%v: real and imaginary rms are %.4f and %.4f -- they should be the same size",
					threeD, ra, rb)
			}
			const n = 30
			for range n {
				wcPlainStep(ss)
			}
			x1, q1 := wcCtr()
			v := (x1 - x0) / n
			kap := float64(ss.Params.C)
			k := 2 * math.Pi / float64(ss.Config.Wavelength) * float64(dir)
			want := 2 * kap * math.Sin(k) // d/dk of 2 C (1 - cos k)
			if math.Abs(v-want)/math.Abs(want) > 0.05 {
				t.Errorf("3D=%v dir %+.0f: speed %+.4f, want %+.4f", threeD, dir, v, want)
			}
			if math.Abs(q1/q0-1) > 1e-5 {
				t.Errorf("3D=%v dir %+.0f: total changed by %.2e", threeD, dir, q1/q0-1)
			}
			t.Logf("3D=%-5v dir %+.0f: speed %+.4f (want %+.4f), a/b rms ratio %.4f, total held",
				threeD, dir, v, want, ra/rb)
		}
	}

	// no phase, no direction: a pulse just spreads
	ps := wcSimEq(128, false, WaveC, 1, WaveCPulse)
	wcPlainStep(ps)
	x0, _ := wcCtr()
	for range 60 {
		wcPlainStep(ps)
	}
	x1, _ := wcCtr()
	if math.Abs(x1-x0) > 0.01 {
		t.Errorf("a pulse with no phase drifted %.4f: nothing in the equation should give it a direction", x1-x0)
	}
	t.Logf("pulse drifted %+.5f over 60 steps: it spreads both ways instead", x1-x0)

	// the contrast: the second-order pair are NOT on the same scale
	ws := &Sim{}
	ws.Config = &Config{}
	ws.Config.Defaults()
	ws.Config.GPU, ws.Config.GUI = false, false
	ws.Config.Equation = Wave
	ws.Config.Size.Set(128, 1, 1)
	ws.ConfigSim()
	ws.Params.Update()
	ws.ConfigState()
	ws.InitFunc = func(s *Sim) {
		s.MovingWavePacket(WavePos, WaveVel, math32.X, math32.Vec3(-1, -1, -1), 1,
			s.Config.Wavelength, s.Config.PacketWidth, 0, s.Config.Amplitude)
	}
	ws.Init()
	rp, rv := wcRMS(WavePos), wcRMS(WaveVel)
	if math.Abs(rv/rp-1) < 0.3 {
		t.Errorf("second-order pos and vel rms are %.4f and %.4f, ratio %.3f -- they were supposed to differ",
			rp, rv, rv/rp)
	}
	t.Logf("second order: pos rms %.4f, vel rms %.4f, ratio %.3f -- a quantity and its rate, not two of a kind",
		rp, rv, rv/rp)
}
