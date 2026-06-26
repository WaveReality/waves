// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"cogentcore.org/core/cli"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/tree"
	"cogentcore.org/lab/base/randx"
	"cogentcore.org/lab/tensor"
	"cogentcore.org/lab/tensorfs"
)

//go:generate core generate -add-types -add-funcs -gosl

// Sim contains everything for the simulation.
type Sim struct {
	// Params contains the current simulation parameters.
	Params *Parameters `new-window:"+" display:"no-inline"`

	// Config contains the broader running configuration.
	Config *Config `new-window:"+" display:"no-inline"`

	// Root is the root tensorfs directory, where all stats and other misc sim data goes.
	Root *tensorfs.Node `display:"-"`

	// Stats has the stats directory within Root.
	Stats *tensorfs.Node `display:"-"`

	// Current has the current stats values within Stats.
	Current *tensorfs.Node `display:"-"`

	// GUI manages all the GUI elements
	GUI GUI `display:"-"`

	// StateVars points the current state variables in effect.
	StateVars enums.Enum `display:"-"`

	// Rand is the random number generator for the network.
	// all random calls must use this.
	// Set seed here for weight initialization values.
	Rand randx.Rand `display:"-"`

	// Random seed to be set at the start of configuring
	// the network and initializing the weights.
	// Set this to get a different set of weights.
	RandSeed int64 `display:"-"`

	// RandSeeds is a list of random seeds to use for each run.
	RandSeeds randx.Seeds `display:"-"`
}

func Run() {
	cfg := &Config{}
	cfg.Defaults()
	opts := cli.DefaultOptions("Waves", "Waves")
	opts.DefaultFiles = append(opts.DefaultFiles, "config.toml")
	// opts.SearchUp = true // so that the sim can be run from the command subdirectory
	opts.IncludePaths = append(opts.IncludePaths, "./configs")

	cli.Run(opts, cfg, RunSim)
}

func RunSim(cfg *Config) error {
	sim := &Sim{}
	sim.Config = cfg
	sim.ConfigSim()
	if cfg.GUI {
		sim.Init()
		sim.ConfigGUI(nil)
		sim.GUI.Body.RunMainWindow()
	} else {
		sim.RunNoGUI()
	}
	return nil
}

func (ss *Sim) ConfigSim() {
	ss.Root, _ = tensorfs.NewDir("Root")
	tensorfs.CurRoot = ss.Root
	ss.RandSeeds.Init(100) // max 100 runs
	ss.InitRandSeed(0)
	if ss.Config.GPU {
		// gpu.DebugAdapter = true
		// gpu.SelectAdapter = ss.Config.Run.GPUDevice
		GPUInit()
		UseGPU = true
	}
	ss.ConfigVars()
	switch ss.Config.Equation {
	case Wave1D, Wave3D:
		ss.WaveConfig()
	}
	ss.ConfigState()
	// if ss.Config.GPU {
	// 	fmt.Println(axon.GPUSystem.Vars().StringDoc())
	// }
}

func (ss *Sim) ConfigState() {
	ctx := GetCtx(0)
	ctx.Size.SetV(ss.Config.Size)
	nvar := int(ss.StateVars.Int64())
	if State == nil {
		State = tensor.NewFloat32()
	}
	State.SetShapeSizes(int(ctx.Size.Z)+2, int(ctx.Size.Y)+2, int(ctx.Size.X)+2, nvar, 2)
}

func (ss *Sim) InitRandSeed(run int) {
	ss.RandSeeds.Set(run)
}

func (ss *Sim) Init() {
}

func (ss *Sim) Run() {

}

func (ss *Sim) StepRun() {
	ctx := GetCtx(0)
	ns := int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z)
	switch ss.Config.Equation {
	case Wave3D:
		RunWave3DKernel(ns)
	}
}

func (ss *Sim) ConfigGUI(b tree.Node) {
	ss.GUI.MakeBody(b, ss, ss.Root, "Waves", "Waves", "Wave simulator")
	ss.GUI.FinalizeGUI(false)
}

func (ss *Sim) RunNoGUI() {
}
