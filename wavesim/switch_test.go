// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"
)

// TestEquationSwitch cycles the Equation through every case the way the GUI
// does -- set the field, call Update -- and checks that each one comes up
// clean rather than inheriting the last one's stats, init options or state.
//
// Twice round, because the failure this guards against is accumulation: a
// second pass that registers each stat again would silently double every row.
func TestEquationSwitch(t *testing.T) {
	eqs := []Equations{Wave, WaveC, WaveCDir, KleinGordon, KleinGordonC,
		Schrodinger, Maxwell, Dirac, Weyl, Electroweak}
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU, ss.Config.GUI = false, false
	ss.Config.Equation = Wave
	ss.Config.Size.Set(16, 16, 16)
	ss.ConfigSim()
	ss.Params.ThreeD.SetBool(true)

	nstats := map[Equations]int{}
	nvars := map[Equations]int64{}
	for pass := range 2 {
		for _, eq := range eqs {
			ss.Config.Equation = eq
			ss.Config.Update() // the GUI path
			if ss.Config.curEquation != eq {
				t.Fatalf("%v: Update did not reconfigure", eq)
			}
			if ss.StateVars == nil || ss.StateVars.Int64() == 0 {
				t.Errorf("%v: no state variables", eq)
				continue
			}
			// ThreeD must survive a switch: it is not something an equation owns
			if ss.Params.ThreeD.IsFalse() {
				t.Errorf("%v: ThreeD was reset by the switch", eq)
			}
			if pass == 0 {
				nstats[eq], nvars[eq] = len(ss.StatFuncs), ss.StateVars.Int64()
			} else {
				if len(ss.StatFuncs) != nstats[eq] {
					t.Errorf("%v: %d stats on the second pass, %d on the first -- Reset is not clearing",
						eq, len(ss.StatFuncs), nstats[eq])
				}
				if ss.StateVars.Int64() != nvars[eq] {
					t.Errorf("%v: state vars changed between passes", eq)
				}
			}
			// and it has to actually run
			for range 3 {
				ss.StepRun()
			}
			v := float64(State.Value(1, 1, 1, 0, int(GetCtx(0).CurState)))
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Errorf("%v: state went bad after three steps", eq)
			}
			if pass == 1 && eq == eqs[len(eqs)-1] {
				t.Logf("cycled %d equations twice: stats and state vars stable", len(eqs))
			}
		}
	}
}
