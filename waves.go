// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
	_ "cogentcore.org/lab/yaegilab"
)

//go:generate core generate

//go:embed icon.svg
var icon string

func main() {
	core.AppIcon = icon
	b := core.NewBody("Waves")
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				// ctx.LinkButton(w, "https://github.com/WaveReality/waves")
				w.SetText("GitHub").SetIcon(icons.GitHub)
			})
		})
	})

	b.RunMainWindow()
}
