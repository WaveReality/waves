// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"

	"cogentcore.org/core/core"
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
	// eqs := wavesim.Dirac
	eqs := wavesim.Weyl
	// eqs := wavesim.Electroweak
	// eqs := wavesim.Spinfield

	// ctrPos := math32.Vec3(-1, -1, -1)
	// ctrInt := math32.Vec3i(-1, -1, -1)

	// note: max 3d size is slightly above 400^3 for KGParticles
	// Total memory size in floats: 4,157,747,712 GB: 16,630,990,848 num vars: 32 buf cap: 21,474,836,480
	// nbufs = 10, total vars = 16 -- could squeeze neigh vars to get 12
	// and actually just push nvars higher given prior experience.
	// actually runs in reasonable time on the macbook at that size!

	wavesim.Run(
		func(sim *wavesim.Sim) {
			sim.Config.Equation = eqs
			sim.Params.ThreeD.SetBool(threed)
			sim.ViewInitFunc = wavesim.ViewInitFour
			switch eqs {
			case wavesim.Wave:
				if threed {
					sim.Config.Size.Set(64, 64, 64)
				} else {
					// sim.Config.Size.Set(80, 1, 1)
					sim.Config.Size.Set(1000, 1, 1)
				}
			case wavesim.WaveC:
				if threed {
					sim.Config.Size.Set(64, 64, 64)
				} else {
					sim.Config.Size.Set(500, 1, 1)
				}
			case wavesim.WaveCDir:
				if threed {
					sim.Config.Size.Set(64, 64, 64)
				} else {
					sim.Config.Size.Set(500, 1, 1)
				}
			case wavesim.KleinGordon:
				if threed {
					sim.Config.Size.Set(64, 64, 64)
				} else {
					sim.Config.Size.Set(500, 1, 1)
				}
			case wavesim.Schrodinger:
				if threed {
					sim.Config.Size.Set(64, 64, 64)
				} else {
					sim.Config.Size.Set(500, 1, 1)
				}
			case wavesim.Maxwell:
				sim.Config.Size.Set(64, 64, 64)
			case wavesim.KleinGordonC:
				sim.Config.Size.Set(64, 64, 64)
			case wavesim.Dirac:
				sim.Config.Size.Set(128, 128, 128)
			case wavesim.Weyl:
				sim.Config.Size.Set(64, 64, 64)
				sim.Config.Size.Set(64, 64, 64)
			case wavesim.Electroweak:
				sim.Config.Size.Set(64, 64, 64)
			case wavesim.Spinfield:
				// sim.Config.Size.Set(100, 100, 100)
				sim.Config.Size.Set(200, 200, 200)
			}
		},
		func(sim *wavesim.Sim) {})
}
