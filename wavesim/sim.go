// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"fmt"
	"io"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/gzipx"
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
	Params *Parameters `display:"add-fields"`

	// Units convert between real-world SI units and per-cube computational units.
	Units Units `new-window:"+" display:"no-inline"`

	// Config contains the broader running configuration.
	Config *Config `new-window:"+" display:"no-inline"`

	// ConfigFunc is run at initial configuration, after all default configuration,
	// and can then change any parameters etc.
	ConfigFunc func(sim *Sim) `display:"-"`

	// InitFunc is run at initialization, and should be used to set
	// the initial State, using functions in init.
	InitFunc func(sim *Sim) `display:"-"`

	// viewInitFuncs are run at initialization of the GUI wave View.
	// use ViewInit method to add.
	viewInitFuncs []func(view *View) `display:"-"`

	// StatFuncs are the stats functions that have been added.
	StatFuncs []func(init bool) `display:"-"`

	// Root is the root tensorfs directory, where all stats and other misc sim data goes.
	Root *tensorfs.Node `display:"-"`

	// Stats has the stats directory within Root.
	Stats *tensorfs.Node `display:"-"`

	// Current has the current stats values within Stats.
	Current *tensorfs.Node `display:"-"`

	// GUI manages all the GUI elements
	GUI GUI // `display:"-"`

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

func Run(configFunc, initFunc func(sim *Sim)) *Sim {
	cfg := &Config{}
	cfg.Defaults()
	opts := cli.DefaultOptions("Waves", "Waves")
	opts.DefaultFiles = append(opts.DefaultFiles, "config.toml")
	opts.SearchUp = true // so that the sim can be run from the command subdirectory
	opts.IncludePaths = append(opts.IncludePaths, "./configs")

	var sim *Sim
	cli.Run(opts, cfg, func(cfg *Config) error {
		sim = RunSim(cfg, configFunc, initFunc)
		return nil
	})
	return sim
}

func RunSim(cfg *Config, configFunc, initFunc func(sim *Sim)) *Sim {
	sim := &Sim{}
	sim.Config = cfg
	sim.ConfigFunc = configFunc
	sim.InitFunc = initFunc
	sim.ConfigSim()
	if cfg.GUI {
		sim.Init()
		sim.ConfigGUI(nil)
		sim.GUI.Body.RunMainWindow()
	} else {
		sim.RunNoGUI()
	}
	return sim
}

func Embed(parent tree.Node, configFunc, initFunc func(sim *Sim)) *Sim { //yaegi:add
	cfg := &Config{}
	cfg.Defaults()
	cfg.GUI = true
	sim := &Sim{}
	sim.Config = cfg
	sim.ConfigFunc = configFunc
	sim.InitFunc = initFunc
	sim.ConfigSim()
	sim.ConfigGUI(parent)
	sim.Init()
	return sim
}

func (ss *Sim) ConfigSim() {
	ss.Root, _ = tensorfs.NewDir("Root")
	tensorfs.CurRoot = ss.Root
	ss.Stats = ss.Root.Dir("Stats")
	ss.RandSeeds.Init(100) // max 100 runs
	ss.InitRandSeed(0)
	ss.ConfigVars()
	if ss.ConfigFunc != nil {
		ss.ConfigFunc(ss)
	}
	ss.Params.Update()
	if ss.Config.GPU {
		// gpu.DebugAdapter = true
		// gpu.SelectAdapter = ss.Config.Run.GPUDevice
		GPUInit()
		UseGPU = true
	}
	switch ss.Config.Equation {
	case Wave1D, Wave3D:
		ss.WaveConfig()
	case KleinGordon1D, KleinGordon3D:
		ss.KleinGordonConfig()
	case Schrodinger1D, Schrodinger3D:
		ss.SchrodingerConfig()
	}
	ss.ConfigState()
	// if ss.Config.GPU {
	// 	fmt.Println(axon.GPUSystem.Vars().StringDoc())
	// }
	ss.Init()
}

func (ss *Sim) ConfigState() {
	ctx := GetCtx(0)
	ctx.Size.SetV(ss.Config.Size)
	nvar := int(ss.StateVars.Int64())
	if State == nil {
		State = tensor.NewFloat32()
	}
	fs := ctx.SizeFull()
	fmt.Println(fs)
	State.SetShapeSizes(int(fs.Z), int(fs.Y), int(fs.X), nvar, 2)
}

func (ss *Sim) InitRandSeed(run int) {
	ss.RandSeeds.Set(run)
}

// Init initializes the state and prepares everything for running.
func (ss *Sim) Init() {
	ss.InitRandSeed(0) // todo: run param
	ctx := GetCtx(0)
	ctx.Init()
	State.SetZeros()
	if ss.InitFunc != nil {
		ss.InitFunc(ss)
	}
	ToGPUTensorStrides()
	ToGPU(ParamsVar, CtxVar, NeighOffsVar, LaplacianWtsVar, StateVar)
	ss.RunStats(true)
}

// Run runs until stopped or Step > MaxSteps. Must be called by goroutine.
func (ss *Sim) Run() {
	ctx := GetCtx(0)
	for {
		if ss.GUI.StopNow() || int(ctx.Step) > ss.Config.MaxSteps {
			break
		}
		ss.StepRun()
	}
	ss.Stopped()
}

// StepN runs given number of steps. Must be called by goroutine.
func (ss *Sim) StepN(n int) {
	for range n {
		if ss.GUI.StopNow() {
			break
		}
		ss.StepRun()
	}
	ss.Stopped()
}

// StepRun does one step of running. Must be called from goroutine.
func (ss *Sim) StepRun() {
	ctx := GetCtx(0)
	ctx.StepInc()
	ToGPU(CtxVar)
	ns := int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z)
	switch ss.Config.Equation {
	case Wave3D:
		RunWave3DKernel(ns)
	case Wave1D:
		RunWave1DKernel(ns)
	case KleinGordon1D:
		RunKleinGordon1DKernel(ns)
	case KleinGordon3D:
		RunKleinGordon3DKernel(ns)
	case Schrodinger3D:
		RunSchrodinger3DKernel(ns)
	case Schrodinger1D:
		RunSchrodinger1DKernel(ns)
	}
	if int(ctx.Step)%ss.Config.ViewInterval != 0 {
		RunDone()
	} else {
		RunDone(StateVar)
		ss.RunStats(false)
		ss.UpdateView()
	}
}

// Stopped should be called whenever running stops.
func (ss *Sim) Stopped() {
	if !ss.GUI.Active {
		return
	}
	ss.GUI.Stopped()
}

// ViewInit adds given function to view initialization functions.
// Called in reverse of order added. Equations typically set default init
// for specific equations (e.g., variable), added at the end.
func (ss *Sim) ViewInit(fun func(view *View)) {
	ss.viewInitFuncs = append(ss.viewInitFuncs, fun)
}

func (ss *Sim) callViewInit(view *View) {
	n := len(ss.viewInitFuncs)
	for i := n - 1; i >= 0; i-- {
		fun := ss.viewInitFuncs[i]
		fun(view)
	}
}

func (ss *Sim) ConfigGUI(b tree.Node) {
	ss.GUI.MakeBody(b, ss, ss.Root, "Waves", "Waves", "Wave simulator")
	vw := ss.GUI.AddView("View")
	vw.sim = ss
	vw.Size = ss.Config.Size
	vw.Size.Z = 3
	vw.Start.X = 1
	vw.Start.Y = 1
	vw.Start.Z = vw.Size.Z / 2
	if vw.Start.Z == 0 {
		vw.Start.Z = 1
	}
	ss.callViewInit(vw)
	ss.RunStats(true)
	ss.GUI.FinalizeGUI(false)
}

func (ss *Sim) UpdateView() {
	if !ss.GUI.Active || ss.GUI.View == nil {
		return
	}
	ctx := GetCtx(0)
	if int(ctx.Step)%ss.Config.ViewInterval != 0 {
		return
	}
	ss.GUI.View.SetCounters(fmt.Sprintf("Step: %d Cur: %d", ctx.Step, ctx.CurState))
	ss.GUI.View.GoUpdateView()
}

// SaveState saves the state to given file.
// If filename ends in .gz, it is gzipped.
func (ss *Sim) SaveState(filename fsx.Filename) error { //types:add
	err := gzipx.Save(string(filename), func(w io.Writer) error {
		return tensor.WriteCSV(State, w, tensor.Tab)
	})
	return errors.Log(err)
}

// OpenState opens the state from given file.
// If filename ends in .gz, it is un-gzipped.
func (ss *Sim) OpenState(filename fsx.Filename) error { //types:add
	err := gzipx.Open(string(filename), func(r io.Reader) error {
		return tensor.ReadCSV(State, r, tensor.Tab)
	})
	return errors.Log(err)
}

func (ss *Sim) RunNoGUI() {
}
