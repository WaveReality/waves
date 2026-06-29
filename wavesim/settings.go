// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/math32/minmax"
)

// note: must be added to gosl due to -gosl flag
//gosl:start

// ViewModes are different ways of displaying wave states.
type ViewModes int32 //enums:enum

const (
	// Plane displays a contiguous plane of values -- best for smooth states.
	Plane ViewModes = iota

	// Bars displays discrete bars at each point -- best for more discontinuous states.
	Bars
)

// CurPrev for Current vs Previous state access.
type CurPrev int32 //enums:enum

const (
	// Current selects the current state value (most recently updated).
	Current CurPrev = iota

	// Previous selects the previous state value.
	Previous
)

// NPanels selects number of panels.
type NPanels int32 //enums:enum -trim-prefix=Panels

const (
	// One panel
	PanelsOne NPanels = iota

	// Two side-by-side panels
	PanelsTwo

	// Four bottom-top and side-by-side panels
	PanelsFour
)

func (np NPanels) N() int {
	switch np {
	case PanelsOne:
		return 1
	case PanelsTwo:
		return 2
	case PanelsFour:
		return 4
	}
	return 1
}

//gosl:end

// Settings for how the View is rendered.
type Settings struct {

	// Number of different panels, each capable of displaying a different variable, mode,
	// and location in the state.
	NPanels NPanels

	// Mode is how the state values are displayed.
	Mode ViewModes

	// Height is how high the values are, in normalized units.
	Height float32

	// Camera specifies the initial camera view to show the scene
	// 1 = default = top-down, 2 = side-long
	Camera int

	// size of a single bar element, where 1 = full width and no space.. .9 default
	BarSize float32 `min:"0.1" max:"1" step:"0.1" default:"0.9"`

	// name of color map to use
	ColorMap core.ColorMapName `display:"-"`

	// size of the labels
	LabelSize float32 `min:"0.01" max:".1" step:"0.01" default:"0.05"`

	// opacity (0-1) of zero values. greater magnitude values become increasingly
	// opaque on either side of this minimum.
	ZeroAlpha float32 `min:"0" max:"1" step:"0.1" default:"0.5"`
}

func (nv *Settings) Defaults() {
	nv.NPanels = PanelsOne
	// nv.Mode = Bars
	nv.Height = 0.2
	nv.BarSize = 0.9
	nv.LabelSize = 0.05
	nv.ZeroAlpha = 0.5
	nv.ColorMap = core.ColorMapName("ColdHot")
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
