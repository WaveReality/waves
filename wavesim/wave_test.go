// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"
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
