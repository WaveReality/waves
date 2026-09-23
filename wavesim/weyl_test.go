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

func wySim(sz int32, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU, ss.Config.GUI = false, false
	ss.Config.Equation = Weyl
	ss.Config.Size.Set(sz, sz, sz)
	ss.ConfigSim()
	ss.Params.ThreeD.SetBool(true)
	ss.Params.Update()
	ss.StateVars = WeylStatesN
	ss.ConfigState()
	ss.InitFunc = init
	ss.Init()
	return ss
}

func wyStep(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	RunWeylKernel(int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z))
	if ss.Params.Edges == EdgesWrap {
		RunEdgesWrapKernel(int(ctx.EdgesN()))
	}
	ss.RunStats(false)
}

func wySums() (l, r float64) {
	c := GetCtx(0)
	return StateSum(c.Size.V(), WeylLMag, c.CurState), StateSum(c.Size.V(), WeylRMag, c.CurState)
}

func wyCtr(sz int32, mag enums.Enum, dim math32.Dims) float64 {
	var num, den float64
	cur := int(GetCtx(0).CurState)
	for z := int32(1); z <= sz; z++ {
		for y := int32(1); y <= sz; y++ {
			for x := int32(1); x <= sz; x++ {
				v := float64(State.Value(int(z), int(y), int(x), int(mag.Int64()), cur))
				d := float64(x)
				if dim == math32.Z {
					d = float64(z)
				}
				num += v * d
				den += v
			}
		}
	}
	return num / den
}

// TestWeylNeutrino: a massless left-handed wave, and nothing else. The right
// half must stay EXACTLY empty -- with Mass at zero there is no term anywhere
// that can put anything into it, which is what having only one handedness
// means. And it must travel at c, approaching it as the wavelength grows past
// the lattice.
func TestWeylNeutrino(t *testing.T) {
	const sz = 64
	var speed [2]float64
	for i, wl := range []float32{8, 16} {
		ss := wySim(sz, func(s *Sim) {
			s.Config.Wavelength = wl
			NeutrinoPacket(s)
		})
		wyStep(ss)
		x0, z0 := wyCtr(sz, WeylLMag, math32.X), wyCtr(sz, WeylLMag, math32.Z)
		_, r0 := wySums()
		const n = 40
		var rmax float64
		for range n {
			wyStep(ss)
			_, r := wySums()
			rmax = math.Max(rmax, math.Abs(r))
		}
		l, _ := wySums()
		if rmax > 1e-9*l {
			t.Errorf("wavelength %g: right half reached %.3g, must stay exactly zero", wl, rmax)
		}
		if d := math.Abs(wyCtr(sz, WeylLMag, math32.Z) - z0); d > 0.01 {
			t.Errorf("wavelength %g: drifted %.3f along Z, should move only along its momentum", wl, d)
		}
		speed[i] = (wyCtr(sz, WeylLMag, math32.X) - x0) / n
		if c := float64(ss.Params.C); speed[i] < 0.7*c || speed[i] > c {
			t.Errorf("wavelength %g: speed %.4f, want just under c = %.4f", wl, speed[i], c)
		}
		// the stat must see the same thing, and must see nothing at all in R
		sl := ss.Stats.Float64(StatGroupVelName(WeylLMag, 0)).Float1D(-1) * float64(ss.Params.C)
		sr := ss.Stats.Float64(StatGroupVelName(WeylRMag, 0)).Float1D(-1)
		if math.Abs(sl-speed[i])/speed[i] > 0.05 {
			t.Errorf("wavelength %g: stat says %.4f but the packet moved at %.4f", wl, sl, speed[i])
		}
		if sr != 0 {
			t.Errorf("wavelength %g: right half has a group velocity of %.4f, but it is empty", wl, sr)
		}
		t.Logf("wavelength %2g: speed %.4f = %.3f c, stat %.4f; right half peaks at %.3g (r0 %.3g)",
			wl, speed[i], speed[i]/float64(ss.Params.C), sl, rmax, r0)
	}
	if speed[1] <= speed[0] {
		t.Errorf("longer wavelength is not faster (%.4f vs %.4f): massless must approach c", speed[1], speed[0])
	}
}

// wyFlipUniform is the chiral flip on a uniform field, where the gradients
// vanish and the trade comes out exactly cos^2 and sin^2.
func wyFlipUniform(ss *Sim) {
	ss.Params.Mass = WeylMass
	ss.Params.Update()
	ss.Fill(WeylL1a, Both, ss.Config.Amplitude)
}

// TestWeylChiralFlip: an electron started entirely left-handed converts
// entirely into the right-handed one and back, at the rest mass frequency,
// with the sum conserved. That trade is the whole of what the mass term does.
func TestWeylChiralFlip(t *testing.T) {
	const sz = 16
	ss := wySim(sz, wyFlipUniform)
	wyStep(ss)
	l, r := wySums()
	tot := l + r
	if r/tot > 0.05 {
		t.Errorf("starts %.2f%% right-handed, should be purely left", 100*r/tot)
	}
	want := math.Pi / float64(ss.Params.Omega0)
	lmin, at, lo, hi := tot, 0, tot, tot
	for i := range int(1.2 * want) {
		wyStep(ss)
		l, r = wySums()
		lo, hi = math.Min(lo, l+r), math.Max(hi, l+r)
		if l < lmin {
			lmin, at = l, i+1
		}
	}
	if lmin/tot > 0.02 {
		t.Errorf("left half only falls to %.2f%%, so it does not fully convert", 100*lmin/tot)
	}
	if got := 2 * float64(at); math.Abs(got-want)/want > 0.05 {
		t.Errorf("flip period %.1f steps, want pi hbar / m c^2 = %.1f", got, want)
	}
	if (hi-lo)/tot > 0.01 {
		t.Errorf("total varies by %.3f%%, but the sum must be conserved", 100*(hi-lo)/tot)
	}
	t.Logf("left half falls to %.3f%% at step %d, so the period is %d vs pi hbar / m c^2 = %.1f; total varies %.4f%%",
		100*lmin/tot, at, 2*at, want, 100*(hi-lo)/tot)
}

// TestWeylElectronAtRest: a massive particle at rest is an equal mixture of
// the two chiralities, and stays that way. Chirality is not helicity.
func TestWeylElectronAtRest(t *testing.T) {
	const sz = 16
	ss := wySim(sz, ElectronAtRest)
	wyStep(ss)
	l0, r0 := wySums()
	var worst float64
	for range 400 {
		wyStep(ss)
		l, r := wySums()
		worst = math.Max(worst, math.Abs(l/r-1))
	}
	if math.Abs(l0/r0-1) > 1e-6 || worst > 0.01 {
		t.Errorf("|psi_L|^2 / |psi_R|^2 starts at %.6f and wanders by %.4f, want a flat 1", l0/r0, worst)
	}
	t.Logf("|psi_L|^2 / |psi_R|^2 starts %.6f, wanders at most %.2e over 400 steps", l0/r0, worst)
}

// TestWeylElectronSlower is the payoff comparison: the same wave, same
// wavelength, same box, with and without a mass. The massive one travels
// slower, which is what the L-R trade buys -- both halves still go at c, and a
// thing that keeps reversing gets down the box less quickly than one that does
// not.
func TestWeylElectronSlower(t *testing.T) {
	const sz = 64
	speed := map[string]float64{}
	for _, tc := range []struct {
		name string
		init func(*Sim)
	}{{"neutrino", NeutrinoPacket}, {"electron", ElectronPacket}} {
		ss := wySim(sz, tc.init)
		wyStep(ss)
		l0, r0 := wySums()
		x0 := wyCtr(sz, WeylLMag, math32.X)
		const n = 40
		for range n {
			wyStep(ss)
		}
		l, r := wySums()
		speed[tc.name] = (wyCtr(sz, WeylLMag, math32.X) - x0) / n
		if d := math.Abs((l+r)/(l0+r0) - 1); d > 1e-4 {
			t.Errorf("%s: total changed by %.2e, it must be conserved", tc.name, d)
		}
		// a massive particle carries both chiralities at one helicity; a
		// massless one cannot have the other at all
		if tc.name == "neutrino" && r != 0 {
			t.Errorf("neutrino has %.3g in its right half, must be exactly zero", r)
		}
		if tc.name == "electron" && (r/l < 0.05 || math.Abs(r/l-r0/l0) > 0.1) {
			t.Errorf("electron R/L is %.3f, started %.3f: wanted a real and steady share", r/l, r0/l0)
		}
		t.Logf("%-8s speed %.4f = %.3f c, R/L %.3f -> %.3f",
			tc.name, speed[tc.name], speed[tc.name]/float64(ss.Params.C), r0/l0, r/l)
	}
	if speed["electron"] >= 0.95*speed["neutrino"] {
		t.Errorf("electron %.4f is not visibly slower than neutrino %.4f", speed["electron"], speed["neutrino"])
	}
}
