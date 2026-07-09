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
	GUI bool `default:"true"`

	// Equation to run
	Equation Equations `edit:"-"`

	// Size of Universe to run. This is only the active portion, excluding
	// edges at all sizes (add 2 to each dim).
	Size math32.Vector3i

	// ViewInterval is how often to update the view
	ViewInterval int `min:"1"`

	// Wavelength is the wavelength to use for functions that use it
	// (Config suffix). Allows user to manipulate the wavelength easily,
	// e.g., for KG and other matter waves.
	Wavelength float32

	// PacketWidth is the wave packet width to use for functions that use it
	// (Config suffix). Allows user to manipulate the wave parameters easily,
	// e.g., for KG and other matter waves.
	PacketWidth float32

	// Momentum provides the default particle momentum.
	Momentum math32.Vector3

	// MaxSteps is the maximum number of steps to run.
	MaxSteps int
}

func (cfg *Config) Defaults() {
	cfg.Size.Set(100, 100, 1)
	cfg.MaxSteps = 100000
	cfg.ViewInterval = 1
	cfg.Wavelength = 8
	cfg.PacketWidth = 8
	cfg.Momentum.X = 0.2
}

func (cfg *Config) SizeFull() math32.Vector3i {
	return cfg.Size.AddScalar(2)
}
