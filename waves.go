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

	runCfg := 2

	wavesim.Run(
		func(sim *wavesim.Sim) {
			switch runCfg {
			case 0:
				sim.Config.Equation = wavesim.Wave1D
				sim.Params.C = 1
				// sim.Config.Size.Set(80, 1, 1)
				sim.Config.Size.Set(1000, 1, 1)
				sim.ViewInit(wavesim.Wave1DViewAll)
			case 1:
				sim.Config.Equation = wavesim.Wave3D
				sim.Config.Size.Set(100, 100, 1)
			case 2:
				sim.Config.Equation = wavesim.KleinGordon
				sim.Config.Size.Set(100, 100, 1)
			}
		},
		func(sim *wavesim.Sim) {
			switch runCfg {
			case 0:
				// sim.PosWavePacket(wavesim.WavePos, math32.X, math32.Vec3i(50, 0, 0), -1, 8, 8, 0, 1)
				sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(500, 0, 0), -1, 80, 80, 0, 1)
				// 				sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(500, 0, 0), -1, 80, 80, 0, 1)
			case 1:
				sim.MovingWavePacket(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(50, 50, 0), -1, 8, 8, 0, 1.5)
			case 2:
				sim.MovingWavePacketParamWavelength(wavesim.WavePos, wavesim.WaveVel, math32.X, math32.Vec3i(50, 50, 0), -1, 8, 0, 1.5)
			}
		})
}
