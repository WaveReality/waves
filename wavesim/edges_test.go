// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"testing"

	"cogentcore.org/core/math32"
)

func dampSim(eq Equations, sz int32, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU, ss.Config.GUI = false, false
	ss.Config.Equation = eq
	ss.Config.Size.Set(sz, sz, sz)
	ss.Config.Wavelength = 8
	ss.Config.PacketWidth = 6
	ss.ConfigSim()
	ss.Params.ThreeD.SetBool(true)
	ss.Params.Edges = EdgesDamp
	ss.Params.Update()
	if init != nil {
		ss.InitFunc = init
	}
	ss.Init()
	return ss
}

// dampTot is the sum of squares over every state variable in the interior:
// how much wave is still in the box, whatever the equation calls it.
func dampTot(ss *Sim) float64 {
	var s float64
	c := GetCtx(0)
	cur := int(c.CurState)
	nv := int(ss.StateVars.Int64())
	for z := int32(1); z <= c.Size.Z; z++ {
		for y := int32(1); y <= c.Size.Y; y++ {
			for x := int32(1); x <= c.Size.X; x++ {
				for v := range nv {
					f := float64(State.Value(int(z), int(y), int(x), v, cur))
					s += f * f
				}
			}
		}
	}
	return s
}

// TestEdgesDamp sends a packet into the boundary in every equation that damps
// and measures what comes back, which is the only thing a damping boundary is
// for.
//
// Two mechanisms are being checked. The second-order equations use Sommerfeld
// damping in the boundary cell, where the force sets the velocity instead of
// adding to it. The first-order ones cannot: they have no acceleration to
// leave out, so they use an open halo plus [EdgeDampFactor], an absorbing
// layer several cells deep. On its own the open halo leaves nearly half a
// Weyl packet in the box, so the layer is doing most of the work.
func TestEdgesDamp(t *testing.T) {
	const sz = 48
	for _, tc := range []struct {
		eq   Equations
		name string
		init func(*Sim)
	}{
		{Wave, "Wave", nil},
		{KleinGordonC, "KleinGordonC", ChargedPacket},
		{Dirac, "Dirac", func(s *Sim) {
			s.MovingWavePacketConfig(Dirac1As, Dirac1Av, math32.X, math32.Vec3(-1, -1, -1), 1, 0, 1)
		}},
		{WaveCDir, "WaveCDir", nil},
		{WaveC, "WaveC", nil},
		{Weyl, "Weyl", nil},
	} {
		ss := dampSim(tc.eq, sz, tc.init)
		ss.StepRun()
		t0 := dampTot(ss)
		if t0 <= 0 {
			t.Errorf("%s: nothing in the box to damp", tc.name)
			continue
		}
		for range 300 {
			ss.StepRun()
		}
		left := dampTot(ss) / t0
		if left > 0.01 {
			t.Errorf("%s: %.3f%% of the wave is still in the box after 300 steps -- it is reflecting",
				tc.name, 100*left)
		}
		t.Logf("%-13s %.3f%% left after 300 steps", tc.name, 100*left)
	}
}
