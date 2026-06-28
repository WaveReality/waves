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
	wavesim.Run(
		func(sim *wavesim.Sim) {
			sim.Config.Equation = wavesim.Wave3D
			sim.Config.Size.Set(100, 100, 1)
		},
		func(sim *wavesim.Sim) {
			// sim.Sine(wavesim.WavePos, math32.X, 27, 0, 1, 0)
			sim.MovingWavePacket(wavesim.WavePos, math32.X, math32.Vec3i(50, 50, 0), -1, 16, 16, 0, 1.5)
		})
}
