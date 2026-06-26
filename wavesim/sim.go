// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import "cogentcore.org/lab/tensorfs"

//go:generate core generate -add-types -add-funcs -gosl

// Sim contains everything for the simulation.
type Sim struct {
	// Params contains the current simulation parameters.
	Params *Parameters `display:"add-fields"`

	// Config contains the broader running configuration.
	Config Config

	// Root is the root tensorfs directory, where all stats and other misc sim data goes.
	Root *tensorfs.Node `display:"-"`

	// Stats has the stats directory within Root.
	Stats *tensorfs.Node `display:"-"`

	// Current has the current stats values within Stats.
	Current *tensorfs.Node `display:"-"`
}
