// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/math32/minmax"
)

// Settings for how the View is rendered.
type Settings struct {
	// name of color map to use
	ColorMap core.ColorMapName

	// opacity (0-1) of zero values. greater magnitude values become increasingly
	// opaque on either side of this minimum.
	ZeroAlpha float32 `min:"0" max:"1" step:"0.1" default:"0.5"`
}

func (nv *Settings) Defaults() {
	if nv.ColorMap == "" {
		nv.ColorMap = core.ColorMapName("ColdHot")
	}
}

// VarSettings holds parameters for display of each variable
type VarSettings struct {

	// the variable
	Var enums.Enum

	// keep Min - Max centered around 0, and use negative heights for units
	// else use full min-max range for height (no negative heights)
	ZeroCtr bool

	// range to display
	Range minmax.Range32 `display:"inline"`

	// if not using fixed range, this is the actual range of data
	MinMax minmax.F32 `display:"inline"`
}

// Defaults sets default values if otherwise not set
func (vs *VarSettings) Defaults() {
	if vs.Range.Max == 0 && vs.Range.Min == 0 {
		vs.ZeroCtr = true
		vs.Range.SetMin(-1)
		vs.Range.SetMax(1)
	}
}

// VarSettinger sets variable parameters
type VarSettinger interface {
	SetVarSettings(vs *VarSettings)
}
