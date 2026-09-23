// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"
)

func scSim(sz int32, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU = false
	ss.Config.GUI = false
	ss.Config.Equation = Schrodinger
	ss.Config.Size.Set(sz, sz, sz)
	ss.ConfigSim()
	ss.Params.ThreeD.SetBool(true)
	ss.Params.Update()
	ss.StateVars = CabStatesN
	ss.ConfigState()
	ss.InitFunc = init
	ss.Init()
	return ss
}

func scStep(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	RunSchrodinger(int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z))
	ss.RunStats(false)
}

func scNorm(sz int32) float64 {
	return StateSum(GetCtx(0).Size.V(), CabMag, GetCtx(0).CurState)
}

// scCtrX is the |chi|^2 weighted X centroid.
func scCtrX(sz int32) float64 {
	var num, den float64
	cur := int(GetCtx(0).CurState)
	for z := int32(1); z <= sz; z++ {
		for y := int32(1); y <= sz; y++ {
			for x := int32(1); x <= sz; x++ {
				v := float64(State.Value(int(z), int(y), int(x), int(CabMag), cur))
				num += v * float64(x)
				den += v
			}
		}
	}
	return num / den
}

// TestSchrodingerNorm: the staggered scheme must hold the norm. Visscher's scheme
// conserves a modified norm exactly, so the drift should be tiny and bounded
// rather than growing -- which is what the naive simultaneous update does.
func TestSchrodingerNorm(t *testing.T) {
	const sz = 24
	for _, tc := range []struct {
		name string
		init func(*Sim)
	}{
		{"BoxStandingWave", BoxStandingWave},
		{"HarmonicOscillator", HarmonicOscillator},
		{"HydrogenGround", HydrogenGround},
	} {
		ss := scSim(sz, tc.init)
		scStep(ss)
		n0 := scNorm(sz)
		lo, hi := n0, n0
		for range 2000 {
			scStep(ss)
			n := scNorm(sz)
			lo, hi = math.Min(lo, n), math.Max(hi, n)
		}
		t.Logf("%-19s norm %.5f, varies by %.2e (%.3f%%) over 2000 steps",
			tc.name, n0, hi-lo, 100*(hi-lo)/n0)
		if math.IsNaN(hi) {
			t.Errorf("%s blew up", tc.name)
		}
		if (hi-lo)/n0 > 0.02 {
			t.Errorf("%s: norm varies by %.1f%%", tc.name, 100*(hi-lo)/n0)
		}
	}
}

// TestSchrodingerBoxStationary: an energy eigenstate must be stationary -- the two
// components turn into each other and their squares add to a constant.
func TestSchrodingerBoxStationary(t *testing.T) {
	const sz = 16
	ss := scSim(sz, BoxStandingWave)
	scStep(ss)
	c := int(sz / 2)
	cur := func() float64 {
		return float64(State.Value(c, c, c, int(CabMag), int(GetCtx(0).CurState)))
	}
	m0 := cur()
	lo, hi := m0, m0
	for range 2000 {
		scStep(ss)
		lo, hi = math.Min(lo, cur()), math.Max(hi, cur())
	}
	t.Logf("box ground state |chi|^2 at center: %.5f, varies by %.2e (%.3f%%)",
		m0, hi-lo, 100*(hi-lo)/m0)
	if (hi-lo)/m0 > 0.02 {
		t.Errorf("an eigenstate should not move: |chi|^2 varies by %.1f%%", 100*(hi-lo)/m0)
	}
}

// TestSchrodingerCoherent: a displaced ground state in a harmonic well swings at
// omega and does not spread. The period is the one the well was built from,
// doubled because a sim step is half a leapfrog step.
func TestSchrodingerCoherent(t *testing.T) {
	const sz = 40
	ss := scSim(sz, HarmonicOscillator)
	want := float64(ss.Config.OscillatorPeriod) // one sim step is one leapfrog step
	scStep(ss)
	mid := float64(sz) / 2
	// count zero crossings of (centroid - center) to get the period
	prev := scCtrX(sz) - mid
	first, last, n := -1, -1, 0
	nst := int(3 * want)
	for i := range nst {
		scStep(ss)
		c := scCtrX(sz) - mid
		if prev < 0 && c >= 0 {
			if first < 0 {
				first = i
			}
			last = i
			n++
		}
		prev = c
	}
	if n < 2 {
		t.Fatalf("only %d crossings in %d steps: the packet is not swinging", n, nst)
	}
	got := float64(last-first) / float64(n-1)
	t.Logf("coherent state period %.1f sim steps, want %.1f (%+.1f%%)", got, want, 100*(got/want-1))
	if math.Abs(got/want-1) > 0.1 {
		t.Errorf("period %g, want %g", got, want)
	}
}

// TestSchrodingerGroupVel: a free packet must travel at the de Broglie group
// velocity hbar k / m, with the lattice correction k -> sin(k).
func TestSchrodingerGroupVel(t *testing.T) {
	sz := int32(48)
	ss := scSim(sz, FreePacket)
	for range 4 { // CabMag is written by the kernel, so let it fill in
		scStep(ss)
	}
	c0 := scCtrX(sz)
	const n = 30 // any further and the packet reaches the damped edge
	for range n {
		scStep(ss)
	}
	vg := (scCtrX(sz) - c0) / n
	k := 2 * math.Pi / float64(ss.Config.Wavelength)
	want := float64(ss.Params.Hbar/ss.Params.Mass) * math.Sin(k)
	if math.Abs(vg-want)/want > 0.05 {
		t.Errorf("group velocity %.5f, want %.5f (hbar sin(k) / m)", vg, want)
	}
	// the stat measures the same thing over its own trailing window
	tsr := ss.Stats.Float64(StatGroupVelName(CabMag, 0))
	sv := tsr.Float1D(tsr.Len()-1) * float64(ss.Params.C)
	if math.Abs(sv-vg)/vg > 0.1 {
		t.Errorf("StatGroupVel %.5f, but the packet moved at %.5f", sv, vg)
	}
	t.Logf("group velocity %.5f, hbar sin(k) / m = %.5f, stat %.5f", vg, want, sv)
}

// TestSchrodingerBigBox: the harmonic well grows as r^2, so it runs past what
// the staggered leapfrog can carry once the box is big enough -- at the
// defaults, past 48^3. Quadratic clamps at SchrodingerVMax, which is why this
// stays finite instead of blowing in from the edges.
func TestSchrodingerBigBox(t *testing.T) {
	const sz = 64
	ss := scSim(sz, HarmonicOscillator)
	vmax := float64(ss.SchrodingerVMax())
	om := float64(schrodOmega(ss))
	corner := 0.5 * float64(ss.Params.Mass) * om * om * 3 * (sz / 2) * (sz / 2)
	if corner <= vmax {
		t.Fatalf("box too small to test the clamp: corner V %.3f <= %.3f", corner, vmax)
	}
	if v := float64(StateMax(GetCtx(0).Size.V(), CabV, GetCtx(0).CurState)); v > vmax {
		t.Errorf("V reaches %.3f, over the %.3f the integrator can carry", v, vmax)
	}
	scStep(ss)
	n0 := scNorm(sz)
	lo, hi := n0, n0
	for range 300 {
		scStep(ss)
		n := scNorm(sz)
		lo, hi = math.Min(lo, n), math.Max(hi, n)
	}
	if (hi-lo)/n0 > 1e-3 {
		t.Errorf("norm %.5f varies by %.2e (%.3f%%) -- unstable", n0, hi-lo, 100*(hi-lo)/n0)
	}
	t.Logf("uncapped corner V would be %.3f, clamped to %.3f; norm %.5f varies by %.2e (%.3f%%)",
		corner, vmax, n0, hi-lo, 100*(hi-lo)/n0)
}

// TestSchrodingerHydrogenP: the 2p orbital is two lobes of opposite phase with
// a node between them. The node is fixed by symmetry rather than by getting
// the radial function right, so it must stay exact even though the lobes
// breathe against the softened 1/r core.
func TestSchrodingerHydrogenP(t *testing.T) {
	const sz = 48
	const ctr = sz/2 + 1 // interior index of the center cell
	ss := scSim(sz, HydrogenP)
	scStep(ss)
	mag := func(dz int) float64 {
		return float64(State.Value(ctr+dz, ctr, ctr, int(CabMag), int(GetCtx(0).CurState)))
	}
	as := func(dz int) float64 {
		return float64(State.Value(ctr+dz, ctr, ctr, int(CabAs), int(GetCtx(0).CurState)))
	}
	// the lobes peak at the n = 2 decay length, and are mirror images
	peak := mag(int(ss.Config.HydrogenRadius))
	if d := math.Abs(mag(-int(ss.Config.HydrogenRadius)) - peak); d > 1e-6*peak {
		t.Errorf("lobes differ by %.2e, should be mirror images", d)
	}
	if up, dn := as(int(ss.Config.HydrogenRadius)), as(-int(ss.Config.HydrogenRadius)); up*dn >= 0 {
		t.Errorf("lobes have the same sign (%+.4f, %+.4f), so this is not a p orbital", up, dn)
	}
	worst := mag(0)
	for range 1000 {
		scStep(ss)
		worst = math.Max(worst, mag(0))
	}
	if worst > 0.02*peak {
		t.Errorf("node reaches %.3g, %.2f%% of the lobe peak %.3g", worst, 100*worst/peak, peak)
	}
	t.Logf("lobe peak %.5f at dz = +-%g, node stays under %.3g (%.3f%% of peak) over 1000 steps",
		peak, ss.Config.HydrogenRadius, worst, 100*worst/peak)
}
