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

	// eqs := wavesim.Wave1D
	eqs := wavesim.KleinGordon1D
	// eqs := wavesim.Schrodinger1D

	wavesim.Run(
		func(sim *wavesim.Sim) {
			switch eqs {
			case wavesim.Wave1D:
				sim.Config.Equation = wavesim.Wave1D
				sim.Params.C = 1
				// sim.Config.Size.Set(80, 1, 1)
				sim.Config.Size.Set(1000, 1, 1)
				sim.ViewInit(wavesim.Wave1DViewAll)
				sim.WaveStats()
			case wavesim.Wave3D:
				sim.Config.Equation = wavesim.Wave3D
				sim.Config.Size.Set(100, 100, 1)
				sim.WaveStats()
			case wavesim.KleinGordon1D:
				sim.Config.Equation = wavesim.KleinGordon1D
				sim.Config.Size.Set(500, 1, 1)
				sim.ViewInit(wavesim.Wave1DViewAll)
				sim.WaveStats()
			case wavesim.KleinGordon3D:
				sim.Config.Equation = wavesim.KleinGordon3D
				sim.Config.Size.Set(100, 100, 1)
				sim.WaveStats()
			case wavesim.Schrodinger1D:
				sim.Config.Equation = wavesim.Schrodinger1D
				sim.Config.Size.Set(500, 1, 1)
				sim.ViewInit(wavesim.Cab1DViewAll)
				sim.SchrodingerStats()
			case wavesim.Schrodinger3D:
				sim.Config.Equation = wavesim.Schrodinger3D
				sim.Config.Size.Set(100, 100, 1)
				sim.SchrodingerStats()
			}
		},
		func(sim *wavesim.Sim) {
			switch eqs {
			case wavesim.Wave1D:
				// sim.PosWavePacket(wavesim.WavePos, math32.X, math32.Vec3i(50, 0, 0), -1, 8, 8, 0, 1)
				sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(500, 0, 0), -1, 80, 80, 0, 1)
				// 				sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(500, 0, 0), -1, 80, 80, 0, 1)
			case wavesim.Wave3D:
				sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(50, 50, 0), -1, 8, 8, 0, 1.5)
			case wavesim.KleinGordon1D:
				sim.MovingWavePacketParams(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(250, 0, 0), -1, 0, 1)
			case wavesim.KleinGordon3D:
				sim.MovingWavePacketParams(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(50, 50, 0), -1, 0, 1.5)
			case wavesim.Schrodinger1D:
				sim.MovingWavePacketParams(wavesim.CabPosA, wavesim.CabPosB, math32.X, math32.Vec3i(250, 0, 0), -1, 0, 1)
			case wavesim.Schrodinger3D:
				sim.MovingWavePacketParams(wavesim.CabPosA, wavesim.CabPosB, math32.X, math32.Vec3i(50, 50, 0), -1, 0, 1)
			}
		})
}
