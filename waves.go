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
	// these support 1D or 3D:
	// eqs := wavesim.Wave
	// eqs := wavesim.WaveC
	// eqs := wavesim.WaveCDir
	// eqs := wavesim.KleinGordon
	// eqs := wavesim.Schrodinger

	// 3D only:
	// eqs := wavesim.Maxwell
	// eqs := wavesim.KleinGordonC
	eqs := wavesim.Dirac
	// eqs := wavesim.Weyl
	// eqs := wavesim.Electroweak
	// eqs := wavesim.Spinfield

	t1sz := 512
	t3sz := 128
	// note: max 3d size is slightly above 400^3 for KGParticles
	// Total memory size in floats: 4,157,747,712 GB: 16,630,990,848 num vars: 32 buf cap: 21,474,836,480
	// nbufs = 10, total vars = 16 -- could squeeze neigh vars to get 12
	// and actually just push nvars higher given prior experience.
	// actually runs in reasonable time on the macbook at that size!

	size := math32.Vec3i(int32(t1sz), 1, 1)
	if threed {
		size.Set(int32(t3sz), int32(t3sz), int32(t3sz))
	}

	// ctrPos := math32.Vec3(-1, -1, -1)
	// ctrInt := math32.Vec3i(-1, -1, -1)

	wavesim.Run(
		func(sim *wavesim.Sim) {
			sim.Config.Equation = eqs
			sim.Params.ThreeD.SetBool(threed)
			sim.ViewInitFunc = wavesim.ViewInitFour
			sim.Config.Size = size
		},
		func(sim *wavesim.Sim) {})
}
