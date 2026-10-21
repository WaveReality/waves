// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yaegiwaves exports waves packages to the yaegi interpreter
// and provides functions for connecting with Cogent Lab.
package yaegiwaves

import (
	"reflect"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/yaegicore"
	"cogentcore.org/lab/goal/interpreter"
	"cogentcore.org/lab/yaegilab/labsymbols"
	"cogentcore.org/lab/yaegilab/tensorsymbols"
	"github.com/cogentcore/yaegi/interp"
)

func init() {
	yaegicore.Interpreters["Goal"] = func(options interp.Options) yaegicore.Interpreter {
		return NewInterpreter(options)
	}
}

// BindTextEditor is [yaegicore.BindTextEditor], with the Goal interpreter
// of this package (with the waves symbols) registered. Content apps whose
// code blocks are compiled into them (by the core tool) set
// content.BindTextEditor to it to run edited code, and list it in the
// Web.DeferredFunctions of their core.toml, so that js builds load yaegi
// and its symbols only when code is first edited:
//
//	[Web]
//	  DeferredFunctions = ["github.com/WaveReality/waves/yaegiwaves.BindTextEditor"]
//
// This interpreter does not record the variables of Goal math mode in
// tensorfs (unlike Cogent Lab's yaegilab), so the compiled code must not
// either, and the code blocks can use the wavesim package without importing
// it:
//
//	[ContentCode]
//	  GoalRecord = false
//	  Imports = ["github.com/WaveReality/waves/wavesim"]
func BindTextEditor(ed *textcore.Editor, parent *core.Frame, language string) {
	yaegicore.BindTextEditor(ed, parent, language)
}

// Interpreter implements [yaegicore.Interpreter] using the [interpreter.Interpreter] for Goal.
type Interpreter struct {
	*interpreter.Interpreter
}

// NewInterpreter returns a new [Interpreter] initialized with the given options.
func NewInterpreter(options interp.Options) *Interpreter {
	return &Interpreter{interpreter.NewInterpreter(options)}
}

func (in *Interpreter) Use(values interp.Exports) error {
	return in.Interp.Use(values)
}

func (in *Interpreter) ImportUsed() {
	errors.Log(in.Use(tensorsymbols.Symbols))
	errors.Log(in.Use(labsymbols.Symbols))
	errors.Log(in.Use(Symbols))
	in.Config()
}

func (in *Interpreter) Eval(src string) (res reflect.Value, err error) {
	res, _, err = in.Interpreter.Eval(src)
	return
}
