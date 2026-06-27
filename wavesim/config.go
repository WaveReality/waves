// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import "cogentcore.org/core/math32"

// Config contains overall simulation configuration options.
type Config struct {
	// GPU determines whether to use the GPU.
	GPU bool `default:"true"`

	// GUI determines whether to show the GUI.
	GUI bool `default:"false"`

	// Equation to run
	Equation Equations

	// Size of Universe to run. This is only the active portion, excluding
	// edges at all sizes (add 2 to each dim).
	Size math32.Vector3i

	// MaxSteps is the maximum number of steps to run.
	MaxSteps int
}

func (cfg *Config) Defaults() {
	cfg.Size.Set(10, 1, 10)
	cfg.MaxSteps = 1000
}

func (cfg *Config) SizeFull() math32.Vector3i {
	return cfg.Size.AddScalar(2)
}
