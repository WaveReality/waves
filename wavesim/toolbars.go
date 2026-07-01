// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"fmt"
	"image"

	"cogentcore.org/core/colors/colormap"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
)

func (vw *View) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(vw.Update).SetIcon(icons.Update).SetTooltip("Update the view to show current state")
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Settings").SetIcon(icons.Settings).
			SetTooltip("set parameters that control display (font size etc)").
			OnClick(func(e events.Event) {
				d := core.NewBody(vw.Name + " Settings")
				core.NewForm(d).SetStruct(&vw.Settings).
					OnChange(func(e events.Event) {
						vw.GoUpdateView()
					})
				d.RunWindowDialog(vw)
			})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("State").SetType(core.ButtonAction).SetMenu(func(m *core.Scene, pos image.Point) {
			fb := core.NewFuncButton(m).SetFunc(vw.sim.SaveState)
			fb.SetIcon(icons.Save)
			fb.Args[0].SetTag(`extension:".tsr,.tsr.gz"`)
			fb = core.NewFuncButton(m).SetFunc(vw.sim.OpenState)
			fb.SetIcon(icons.Open)
			fb.Args[0].SetTag(`extension:".tsr,.tsr.gz"`)
		})
	})
	tree.Add(p, func(w *core.Separator) {})

	pltp := "which panel is active for selecting the variable and other view changes"
	tree.Add(p, func(w *core.Text) {
		w.SetText("Panel:").SetTooltip(pltp)
	})
	tree.Add(p, func(w *core.Spinner) {
		w.SetMin(0).SetMax(4).SetStep(1).SetValue(float32(vw.curPanel)).SetTooltip(pltp)
		w.Styler(func(s *styles.Style) {
			s.Max.X.Ch(9)
			s.Min.X.Ch(9)
		})
		w.OnChange(func(e events.Event) {
			mx := vw.Settings.NPanels.N()
			pl := int(w.Value)
			if pl < mx && pl >= 0 {
				vw.curPanel = pl
			}
			w.SetValue(float32(vw.curPanel))
			vw.varsFrame.Update()
			vw.UpdateView()
		})
	})

	tree.Add(p, func(w *core.Separator) {})

	vp, ok := vw.VarSettings[vw.Var]
	if !ok {
		vp = &VarSettings{}
		vp.Defaults()
	}
	var minSpin, maxSpin *core.Spinner
	var minSwitch, maxSwitch *core.Switch

	tree.Add(p, func(w *core.Separator) {})
	tree.AddAt(p, "minSwitch", func(w *core.Switch) {
		minSwitch = w
		w.SetText("Min").SetType(core.SwitchCheckbox).SetChecked(vp.Range.FixMin).
			SetTooltip("NOTE: not functional yet!  Fix the minimum end of the displayed value range to value shown in next box.  Having both min and max fixed is recommended where possible for speed and consistent interpretability of the colors.").
			OnChange(func(e events.Event) {
				// vp := vw.GetVarSettingsPanel(vw.curPanel)
				// vp.Range.FixMin = w.IsChecked()
				minSpin.UpdateWidget().NeedsRender()
				vw.UpdateView()
			})
		w.Updater(func() {
			vp, _ := vw.GetVarSettingsPanel(vw.curPanel)
			if vp != nil {
				w.SetChecked(vp.Range.FixMin)
			}
		})
	})
	tree.AddAt(p, "minSpin", func(w *core.Spinner) {
		minSpin = w
		w.Styler(func(s *styles.Style) {
			s.Min.X.Ch(15)
			s.Max.X.Ch(15)
		})
		w.SetValue(vp.Range.Min).
			OnChange(func(e events.Event) {
				vp, _ := vw.GetVarSettingsPanel(vw.curPanel)
				if vp == nil {
					return
				}
				vp.Range.SetMin(w.Value)
				vp.Range.FixMin = true
				minSwitch.UpdateWidget().NeedsRender()
				if vp.ZeroCtr && vp.Range.Min < 0 && vp.Range.FixMax {
					vp.Range.SetMax(-vp.Range.Min)
				}
				if vp.ZeroCtr {
					maxSpin.UpdateWidget().NeedsRender()
				}
				vw.UpdateView()
			})
		w.Updater(func() {
			vp, _ := vw.GetVarSettingsPanel(vw.curPanel)
			if vp != nil {
				w.SetValue(vp.Range.Min)
			}
		})
	})

	tree.AddAt(p, "cmap", func(w *core.ColorMapButton) {
		w.MapName = string(vw.Settings.ColorMap)
		w.SetTooltip("Color map for translating values into colors -- click to select alternative.")
		w.Styler(func(s *styles.Style) {
			s.Min.X.Em(10)
			s.Min.Y.Em(1.2)
			s.Grow.Set(0, 1)
		})
		w.OnChange(func(e events.Event) {
			cmap, ok := colormap.AvailableMaps[string(w.MapName)]
			if ok {
				vw.colorMap = cmap
			}
			vw.UpdateView()
		})
	})

	tree.AddAt(p, "maxSwitch", func(w *core.Switch) {
		maxSwitch = w
		w.SetText("Max").SetType(core.SwitchCheckbox).SetChecked(vp.Range.FixMax).
			SetTooltip("Note: not functional yet! Fix the maximum end of the displayed value range to value shown in next box.  Having both min and max fixed is recommended where possible for speed and consistent interpretability of the colors.").
			OnChange(func(e events.Event) {
				// vp := vw.GetVarSettingsPanel(vw.curPanel)
				// vp.Range.FixMax = w.IsChecked()
				maxSpin.UpdateWidget().NeedsRender()
				vw.UpdateView()
			})
		w.Updater(func() {
			vp, _ := vw.GetVarSettingsPanel(vw.curPanel)
			if vp != nil {
				w.SetChecked(vp.Range.FixMax)
			}
		})
	})

	tree.AddAt(p, "maxSpin", func(w *core.Spinner) {
		maxSpin = w
		w.Styler(func(s *styles.Style) {
			s.Min.X.Ch(15)
			s.Max.X.Ch(15)
		})
		w.SetValue(vp.Range.Max).OnChange(func(e events.Event) {
			vp, _ := vw.GetVarSettingsPanel(vw.curPanel)
			if vp == nil {
				return
			}
			vp.Range.SetMax(w.Value)
			vp.Range.FixMax = true
			maxSwitch.UpdateWidget().NeedsRender()
			if vp.ZeroCtr && vp.Range.Max > 0 && vp.Range.FixMin {
				vp.Range.SetMin(-vp.Range.Max)
			}
			if vp.ZeroCtr {
				minSpin.UpdateWidget().NeedsRender()
			}
			vw.UpdateView()
		})
		w.Updater(func() {
			vp, _ := vw.GetVarSettingsPanel(vw.curPanel)
			if vp != nil {
				w.SetValue(vp.Range.Max)
			}
		})
	})

	tree.AddAt(p, "zeroCtrSwitch", func(w *core.Switch) {
		w.SetText("ZeroCtr").SetChecked(vp.ZeroCtr).
			SetTooltip("keep Min - Max centered around 0, and use negative heights for units -- else use full min-max range for height (no negative heights)").
			OnChange(func(e events.Event) {
				vp, _ := vw.GetVarSettingsPanel(vw.curPanel)
				if vp != nil {
					vp.ZeroCtr = w.IsChecked()
					vw.UpdateView()
				}
			})
		w.Updater(func() {
			vp, _ := vw.GetVarSettingsPanel(vw.curPanel)
			if vp != nil {
				w.SetChecked(vp.ZeroCtr)
			}
		})
	})
}

func (vw *View) MakeViewbar(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Update).SetTooltip("reset to default initial display").
			OnClick(func(e events.Event) {
				vw.SceneXYZ().SetCamera("default")
				vw.SceneXYZ().Rebuild()
				vw.UpdateView()
			})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.ZoomIn).SetTooltip("zoom in")
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Zoom(-.05)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.ZoomOut).SetTooltip("zoom out")
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Zoom(.05)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.Text) {
		w.SetText("Rot:").SetTooltip("rotate display")
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowLeft)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Orbit(5, 0)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowUp)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Orbit(0, 5)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowDown)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Orbit(0, -5)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowRight)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Orbit(-5, 0)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Text) {
		w.SetText("Pan:").SetTooltip("pan display")
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowLeft)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Pan(-.2, 0)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowUp)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Pan(0, .2)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowDown)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Pan(0, -.2)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowRight)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		})
		w.OnClick(func(e events.Event) {
			vw.SceneXYZ().Camera.Pan(.2, 0)
			vw.UpdateView()
		})
	})
	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Text) { w.SetText("Save:") })

	for i := 1; i <= 4; i++ {
		nm := fmt.Sprintf("%d", i)
		tree.AddAt(p, "saved-"+nm, func(w *core.Button) {
			w.SetText(nm).
				SetTooltip("first click (or + Shift) saves current view, second click restores to saved state")
			w.OnClick(func(e events.Event) {
				sc := vw.SceneXYZ()
				cam := nm
				if e.HasAllModifiers(e.Modifiers(), key.Shift) {
					sc.SaveCamera(cam)
				} else {
					err := sc.SetCamera(cam)
					if err != nil {
						sc.SaveCamera(cam)
					}
				}
				fmt.Printf("Camera %s: %v\n", cam, sc.Camera.GenGoSet(""))
				vw.UpdateView()
			})
		})
	}
}
