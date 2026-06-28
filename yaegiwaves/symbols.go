// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yaegiwaves exports axon packages to the yaegi interpreter.
package yaegiwaves

//go:generate ./make

import (
	"reflect"
)

var Symbols = map[string]map[string]reflect.Value{}
