// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"cogentcore.org/lab/tensor"
)

//go:generate gosl -exclude=Update,Defaults,ShouldDisplay -max-buffer-size=2147483616

//gosl:start

// vars are all the global vars for GPU / CPU computation.
//
//gosl:vars
var (
	// Params contains the full set of simulation parameters.
	//gosl:group Params
	//gosl:read-only
	Params []Parameters

	// NeighOffs are neighborhood offsets for 3D 26 neighbors: [26][3]
	//gosl:dims 2
	NeighOffs *tensor.Int32

	// LaplacianWts are Laplacian weighting factors for 3D 26 neighbors.
	//gosl:dims 1
	LaplacianWts *tensor.Float32

	// AverageWts are neighborhood average weighting factors for 3D 26 neighbors + center.
	//gosl:dims 1
	AverageWts *tensor.Float32

	// Ctx has the Context state values.
	//gosl:group State
	//gosl:read-only
	Ctx []Context

	// State is the overall wave state, with inner-most index being the current
	// and previous states. [Z][Y][X][VarsN][2]
	// The display shows X-Y planes stacked in the Z dimension.
	//gosl:dims 5
	//gosl:nbuffs 6
	State *tensor.Float32
)

//gosl:end
