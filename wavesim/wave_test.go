// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"

	"cogentcore.org/core/math32"
)

// wvSim builds Wave or KleinGordon on the same state with the same config, in
// 1D and in a box big enough that the packet never approaches the edges.
func wvSim(sz int32, eq Equations, wl, width float32) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU, ss.Config.GUI = false, false
	ss.Config.Equation = eq
	ss.Config.Size.Set(sz, 1, 1)
	ss.Config.Wavelength = wl
	ss.Config.PacketWidth = width
	ss.ConfigSim()
	ss.Params.Edges = EdgesWrap
	ss.Params.Update()
	ss.ConfigState()
	ss.InitFunc = WavePacket
	ss.Init()
	return ss
}

func wvStep(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	n := int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z)
	if ss.Config.Equation == Wave {
		RunWaveKernel(n)
	} else {
		RunKleinGordonKernel(n)
	}
	RunEdgesWrapKernel(int(ctx.EdgesN()))
}

// wvCtr is the intensity-weighted X centroid of the second-order field.
func wvCtr(sz int32) float64 {
	var num, den float64
	cur := int(GetCtx(0).CurState)
	for x := int32(1); x <= sz; x++ {
		v := float64(State.Value(1, 1, int(x), int(WavePos), cur))
		v *= v
		num += v * float64(x)
		den += v
	}
	return num / den
}

// TestWaveDispersion is what the mass term costs, measured: the same packet,
// the same box, the same config, run under Wave and under KleinGordon at two
// wavelengths.
//
// Wave travels at c whatever the wavelength, because its equation has no
// scale in it -- c is a ratio, not a length, so nothing can make omega
// anything but proportional to k. The mass term supplies the one scale there
// is, the Compton wavelength, and after that omega cannot be proportional to
// anything.
//
// Note WHICH waves it slows. The Laplacian is a coupling to neighbours and its
// restoring force comes from the difference with them, which dies away as the
// wave gets longer; the mass is a spring to ground, and its does not. So short
// waves barely notice the mass and long ones are dominated by it -- the
// opposite of the usual lattice error, which spoils SHORT waves.
func TestWaveDispersion(t *testing.T) {
	const sz = 2048
	const short, long float32 = 16, 64
	speed := map[Equations]map[float32]float64{}
	for _, eq := range []Equations{Wave, KleinGordon} {
		speed[eq] = map[float32]float64{}
		for _, wl := range []float32{short, long} {
			ss := wvSim(sz, eq, wl, 2*wl)
			wvStep(ss)
			x0 := wvCtr(sz)
			const n = 100
			for range n {
				wvStep(ss)
			}
			speed[eq][wl] = (wvCtr(sz) - x0) / n / float64(ss.Params.C)
			t.Logf("%-12v wavelength %2.0f: %.4f c", eq, wl, speed[eq][wl])
		}
	}
	// massless: the same speed at both, and that speed is c
	for _, wl := range []float32{short, long} {
		if math.Abs(speed[Wave][wl]-1) > 0.03 {
			t.Errorf("Wave at wavelength %.0f runs at %.4f c, but a massless wave has no scale to disperse about",
				wl, speed[Wave][wl])
		}
	}
	// massive: distinctly slower at the LONG wavelength, where the mass wins
	drop := speed[KleinGordon][short] - speed[KleinGordon][long]
	if drop < 0.08 {
		t.Errorf("KleinGordon only slowed by %.4f c between the two wavelengths: the mass term should disperse it", drop)
	}
	if speed[KleinGordon][long] >= speed[Wave][long]-0.05 {
		t.Errorf("at wavelength %.0f KleinGordon runs at %.4f c against Wave's %.4f c -- the mass is not costing anything",
			long, speed[KleinGordon][long], speed[Wave][long])
	}
	t.Logf("mass costs %.4f c at wavelength %.0f and %.4f c at wavelength %.0f: the LONG waves are the ones it slows",
		speed[Wave][short]-speed[KleinGordon][short], short,
		speed[Wave][long]-speed[KleinGordon][long], long)
}

// wvMode puts a single pure sine mode of the given wavelength into an
// equation, in 1D with wrapped edges, so its one lattice wavenumber can be
// watched on its own.
func wvMode(eq Equations, wl float32) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU, ss.Config.GUI = false, false
	ss.Config.Equation = eq
	ss.Config.Size.Set(256, 1, 1)
	ss.ConfigSim()
	ss.Params.Edges = EdgesWrap
	if eq == Wave {
		ss.Params.Diffusion.SetBool(true)
	}
	ss.Params.Update()
	ss.InitFunc = func(s *Sim) {
		if eq == Wave {
			s.Sine(WavePos, math32.X, wl, 0, 1, 0)
		} else {
			s.Sine(WaveCa, math32.X, wl, 0, 1, 0)
		}
	}
	ss.Init()
	return ss
}

// TestDiffusion is the last corner of the picture, and the one that says what
// the i in a quantum wave is for.
//
// Params.Diffusion moves the Laplacian from the acceleration to the velocity
// and nothing else: same term, same C^2. WaveC is that same first-order
// equation with an i in front. So a mode of wavenumber k DECAYS at C^2 khat^2
// under diffusion and ROTATES at C^2 khat^2 under WaveC -- the same rate, and
// the i is the entire difference between heat spreading out and a wave
// propagating.
func TestDiffusion(t *testing.T) {
	const wl float32 = 32
	k := 2 * math.Pi / float64(wl)
	kh2 := 4 * math.Sin(k/2) * math.Sin(k/2) // the 1D lattice khat^2
	at := func(v int) float64 {
		return float64(State.Value(1, 1, 9, v, int(GetCtx(0).CurState)))
	}

	d := wvMode(Wave, wl)
	want := float64(d.Params.CSq) * kh2
	a0 := at(int(WavePos))
	const n = 60
	for range n {
		ctx := GetCtx(0)
		ctx.StepInc()
		RunWaveKernel(int(ctx.Size.X))
		RunEdgesWrapKernel(int(ctx.EdgesN()))
	}
	a1 := at(int(WavePos))
	if a1 >= a0 {
		t.Fatalf("with Diffusion on the mode should decay, went %.4f -> %.4f", a0, a1)
	}
	decay := -math.Log(a1/a0) / n

	w := wvMode(WaveC, wl)
	if math.Abs(float64(w.Params.C)-float64(d.Params.CSq)) > 1e-6 {
		t.Fatalf("the two coefficients must match to compare rates: %v vs %v", w.Params.C, d.Params.CSq)
	}
	prev := at(int(WaveCa))
	var first, last, cnt int
	for i := range 2000 {
		w.RunWaveC(int(GetCtx(0).Size.X))
		GetCtx(0).StepInc()
		RunEdgesWrapKernel(int(GetCtx(0).EdgesN()))
		v := at(int(WaveCa))
		if prev > 0 && v <= 0 { // same mode, but it turns instead of dying
			if cnt == 0 {
				first = i
			}
			last, cnt = i, cnt+1
		}
		prev = v
	}
	if cnt < 2 {
		t.Fatalf("WaveC mode did not oscillate: %d crossings", cnt)
	}
	rot := 2 * math.Pi / (float64(last-first) / float64(cnt-1))

	for nm, got := range map[string]float64{"diffusion decay": decay, "WaveC rotation": rot} {
		if math.Abs(got-want)/want > 0.02 {
			t.Errorf("%s is %.6f per step, want C^2 khat^2 = %.6f", nm, got, want)
		}
	}
	if math.Abs(rot/decay-1) > 0.02 {
		t.Errorf("rotation %.6f and decay %.6f differ by more than the lattice should explain", rot, decay)
	}
	t.Logf("same mode, same coefficient: diffusion decays at %.6f per step, WaveC rotates at %.6f (C^2 khat^2 = %.6f) -- the i is the only difference",
		decay, rot, want)
}
