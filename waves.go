// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	_ "cogentcore.org/lab/yaegilab"
	"github.com/WaveReality/waves/wavesim"
)

//go:generate core generate

//go:embed icon.svg
var icon string

func main() {
	core.AppIcon = icon

	// threed := false
	threed := true
	// eqs := wavesim.Wave
	// eqs := wavesim.KleinGordonC
	// eqs := wavesim.Schrodinger
	eqs := wavesim.Maxwell
	// eqs := wavesim.Dirac
	// eqs := wavesim.Higgs
	// eqs := wavesim.Spinfield

	ctrPos := math32.Vec3(-1, -1, -1)
	ctrInt := math32.Vec3i(-1, -1, -1)

	// note: max 3d size is slightly above 400^3 for KGParticles
	// Total memory size in floats: 4,157,747,712 GB: 16,630,990,848 num vars: 32 buf cap: 21,474,836,480
	// nbufs = 10, total vars = 16 -- could squeeze neigh vars to get 12
	// and actually just push nvars higher given prior experience.
	// actually runs in reasonable time on the macbook at that size!

	wavesim.Run(
		func(sim *wavesim.Sim) {
			sim.Config.Equation = eqs
			switch eqs {
			case wavesim.Wave:
				sim.Params.C = 1
				// sim.Params.Edges = wavesim.EdgesWrap
				sim.Params.Edges = wavesim.EdgesDamp
				sim.ViewInit(func(vw *wavesim.View) {
					// vw.Settings.NPanels = wavesim.PanelsTwo
					// vw.SetMode(wavesim.Bars, -1)
				})
				if threed {
					sim.Params.ThreeD.SetBool(true)
					// sim.Config.Size.Set(100, 100, 1)
					sim.Config.Size.Set(100, 100, 100)
				} else {
					// sim.Config.Size.Set(80, 1, 1)
					sim.Config.Size.Set(1000, 1, 1)
					sim.ViewInit(wavesim.Wave1DViewAll)
				}
				sim.WaveStats()
			case wavesim.KleinGordon:
				if threed {
					sim.Params.ThreeD.SetBool(true)
					sim.Config.Size.Set(100, 100, 100)
				} else {
					sim.Config.Size.Set(500, 1, 1)
					sim.ViewInit(wavesim.Wave1DViewAll)
				}
				sim.WaveStats()
			case wavesim.KleinGordonC:
				if threed {
					sim.Params.ThreeD.SetBool(true)
					sim.Config.Size.Set(100, 100, 100)
				} else {
					sim.Config.Size.Set(500, 1, 1)
					sim.ViewInit(wavesim.Cab1DViewAll)
				}
				sim.SchrodingerStats()
			case wavesim.Schrodinger:
				if threed {
					sim.Params.ThreeD.SetBool(true)
					sim.Config.Size.Set(100, 100, 100)
				} else {
					sim.Config.Size.Set(500, 1, 1)
					sim.ViewInit(wavesim.Cab1DViewAll)
				}
				sim.SchrodingerStats()
			case wavesim.Maxwell:
				sim.Params.C = 0.5 // note: Lorentz gauge constraint on A0 requires < 1
				if threed {
					sim.Params.Edges = wavesim.EdgesDamp
					sim.Params.ThreeD.SetBool(true)
					sim.Config.Size.Set(100, 100, 100)
				} else {
					sim.Config.Size.Set(500, 1, 1)
				}
			case wavesim.Higgs:
				sim.Units.E = 0.55
				sim.Params.HiggsMu = 0.02
				sim.Params.HiggsLambda = 0.55
				// sim.Params.Mass = -0.5
				sim.Params.Mass = 0
				sim.Params.Edges = wavesim.EdgesDamp
				sim.Params.ThreeD.SetBool(true)
				sim.Config.Size.Set(100, 100, 100)
				sim.HiggsStats()
				sim.ViewInit(wavesim.HiggsViewAll)
			case wavesim.Dirac:
				if threed {
					sim.Params.Edges = wavesim.EdgesDamp
					sim.Params.ThreeD.SetBool(true)
					sim.Config.Size.Set(100, 100, 100)
				} else {
					sim.Config.Size.Set(500, 1, 1)
				}
			case wavesim.Spinfield:
				sim.Params.Edges = wavesim.EdgesDamp
				sim.Params.Mass = 0.1
				sim.Params.Hbar = 0.1
				sim.Params.Move.SetBool(false)
				sim.Config.Velocity.X = 0
				if threed {
					sim.Params.ThreeD.SetBool(true)
					// sim.Config.Size.Set(100, 100, 100)
					sim.Config.Size.Set(200, 200, 200)
				} else {
					sim.Config.Size.Set(50, 1, 1)
				}
				sim.ViewInit(wavesim.SpinfieldViewAll)
				sim.SpinfieldStats()
			}
		},
		func(sim *wavesim.Sim) {
			switch eqs {
			case wavesim.Wave:
				if threed {
					sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, ctrPos, -1, 8, 8, 0, 1.5)
				} else {
					// sim.PosWavePacket(wavesim.WavePos, math32.X, math32.Vec3i(50, 0, 0), -1, 8, 8, 0, 1)
					sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, ctrPos, -1, 80, 80, 0, 1)
					// 				sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(500, 0, 0), -1, 80, 80, 0, 1)
				}
			case wavesim.KleinGordon:
				sim.MovingWavePacketConfig(wavesim.WavePos, wavesim.WaveVel, math32.X, ctrPos, -1, 0, 1)
			case wavesim.KleinGordonC:
				sim.MovingWavePacketConfig(wavesim.CabPosA, wavesim.CabPosB, math32.X, ctrPos, -1, 0, 1)
			case wavesim.Schrodinger:
				sim.MovingWavePacketConfig(wavesim.CabPosA, wavesim.CabPosB, math32.X, ctrPos, -1, 0, 1)
			case wavesim.Maxwell:
				if threed {
					sim.Point(wavesim.Charge, wavesim.Both, ctrInt, 1)
					sim.InvR(wavesim.A0s, ctrPos, sim.Params.Mu0)
					sim.MovingWavePacketConfig(wavesim.AYs, wavesim.AYv, math32.X, ctrPos, -1, 0, 1)
				} else {
					sim.Point(wavesim.Charge, wavesim.Both, ctrInt, 1)
				}
			case wavesim.Dirac:
				if threed {
					sim.Point(wavesim.Charge, wavesim.Both, ctrInt, 1)
					sim.InvR(wavesim.A0s, ctrPos, sim.Params.Mu0)
					sim.MovingWavePacketConfig(wavesim.DiracPos1A, wavesim.DiracPos1B, math32.X, ctrPos, -1, 0, 1)
				} else {
					sim.Point(wavesim.Charge, wavesim.Both, ctrInt, 1)
				}
			case wavesim.Higgs:
				sim.Point(wavesim.Charge, wavesim.Both, ctrInt, 1)
				sim.InvR(wavesim.A0s, ctrPos, sim.Params.Mu0)
				sim.InvR(wavesim.HiggsHs0a, ctrPos, sim.Params.Mu0)
				sim.InvR(wavesim.HiggsHv0b, ctrPos, -sim.Params.Mu0)
			case wavesim.Spinfield:
				sim.ParticleAtConfig(ctrInt, 1)
				sim.ParticleField(ctrInt, 8)
			}
		})
}
