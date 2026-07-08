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

	threed := false
	// threed := true
	// eqs := wavesim.Wave
	// eqs := wavesim.KleinGordonC
	// eqs := wavesim.Schrodinger
	// eqs := wavesim.Maxwell
	// eqs := wavesim.Dirac
	eqs := wavesim.ParticleKGC

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
				if threed {
					sim.Params.C = 1
					sim.Params.Edges = wavesim.EdgesDamp
					sim.Params.ThreeD.SetBool(true)
					sim.Config.Size.Set(100, 100, 100)
				} else {
					sim.Config.Size.Set(500, 1, 1)
				}
			case wavesim.Dirac:
				if threed {
					sim.Params.Edges = wavesim.EdgesDamp
					sim.Params.ThreeD.SetBool(true)
					sim.Config.Size.Set(100, 100, 100)
				} else {
					sim.Config.Size.Set(500, 1, 1)
				}
			case wavesim.ParticleKGC:
				if threed {
					sim.Params.ThreeD.SetBool(true)
					sim.Config.Size.Set(100, 100, 100)
				} else {
					sim.Config.Size.Set(500, 1, 1)
				}
				sim.ViewInit(wavesim.ParticleKGCViewAll)
				sim.ParticleKGCStats()
			}
		},
		func(sim *wavesim.Sim) {
			switch eqs {
			case wavesim.Wave:
				if threed {
					sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(50, 50, 50), -1, 8, 8, 0, 1.5)
				} else {
					// sim.PosWavePacket(wavesim.WavePos, math32.X, math32.Vec3i(50, 0, 0), -1, 8, 8, 0, 1)
					sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(500, 0, 0), -1, 80, 80, 0, 1)
					// 				sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(500, 0, 0), -1, 80, 80, 0, 1)
				}
			case wavesim.KleinGordon:
				if threed {
					sim.MovingWavePacketParams(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(50, 50, 50), -1, 0, 1.5)

				} else {
					sim.MovingWavePacketParams(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(250, 0, 0), -1, 0, 1)
				}
			case wavesim.KleinGordonC:
				if threed {
					sim.MovingWavePacketParams(wavesim.CabPosA, wavesim.CabPosB, math32.X, math32.Vec3i(50, 50, 50), -1, 0, 1)
				} else {
					sim.MovingWavePacketParams(wavesim.CabPosA, wavesim.CabPosB, math32.X, math32.Vec3i(250, 0, 0), -1, 0, 1)
				}
			case wavesim.Schrodinger:
				if threed {
					sim.MovingWavePacketParams(wavesim.CabPosA, wavesim.CabPosB, math32.X, math32.Vec3i(50, 50, 50), -1, 0, 1)
				} else {
					sim.MovingWavePacketParams(wavesim.CabPosA, wavesim.CabPosB, math32.X, math32.Vec3i(250, 0, 0), -1, 0, 1)
				}
			case wavesim.Maxwell:
				if threed {
					sim.Point(wavesim.Charge, wavesim.Both, math32.Vec3i(50, 50, 50), 1)
					sim.InvR(wavesim.A0Pos, math32.Vec3i(50, 50, 50), sim.Params.Mu0)
				} else {
					sim.Point(wavesim.Charge, wavesim.Both, math32.Vec3i(250, 0, 0), 1)
				}
			case wavesim.Dirac:
				if threed {
					sim.Point(wavesim.Charge, wavesim.Both, math32.Vec3i(50, 50, 50), 1)
					sim.InvR(wavesim.A0Pos, math32.Vec3i(50, 50, 50), sim.Params.Mu0)
					sim.MovingWavePacketParams(wavesim.DiracPos1A, wavesim.DiracPos1B, math32.X, math32.Vec3i(50, 50, 50), -1, 0, 1)
				} else {
					sim.Point(wavesim.Charge, wavesim.Both, math32.Vec3i(250, 0, 0), 1)
				}
			case wavesim.ParticleKGC:
				if threed {
					// sim.MovingWavePacketParams(wavesim.CabPosA, wavesim.CabPosB, math32.X, math32.Vec3i(50, 50, 50), -1, 0, 1)
					sim.Point(wavesim.CabSelfPosA, wavesim.CurOnly, math32.Vec3i(50, 50, 50), 1)
				} else {
					// sim.MovingWavePacketParams(wavesim.CabPosA, wavesim.CabPosB, math32.X, math32.Vec3i(250, 0, 0), -1, 0, 1)
					sim.Point(wavesim.CabSelfPosA, wavesim.CurOnly, math32.Vec3i(250, 0, 0), 1)
					sim.Point(wavesim.CabSelfPosB, wavesim.PrevOnly, math32.Vec3i(250, 0, 0), -1)
				}
			}
		})
}
