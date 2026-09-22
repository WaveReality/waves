// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"

	"cogentcore.org/core/math32"
)

// emSim builds a 3D Maxwell sim and runs one of the MaxwellConfigs on it.
func emSim(sz int32, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU = false
	ss.Config.GUI = false
	ss.Config.Equation = Maxwell
	ss.Config.Size.Set(sz, sz, sz)
	ss.ConfigSim()
	ss.Params.ThreeD.SetBool(true)
	ss.Params.Update()
	ss.StateVars = EMStatesN
	ss.ConfigState()
	ss.MaxwellStats()
	ss.InitFunc = init
	ss.Init()
	return ss
}

func emStep(ss *Sim) {
	ctx := GetCtx(0)
	ctx.StepInc()
	ns := int(ctx.Size.X * ctx.Size.Y * ctx.Size.Z)
	RunMaxwellKernel(ns)
	switch ss.Params.Edges {
	case EdgesWrap:
		RunEdgesWrapKernel(int(ctx.EdgesN()))
	case EdgesDamp:
		RunMaxwellDampKernel(int(ctx.EdgesN()))
	}
	ss.RunStats(false)
}

// emLast is the most recent value of the named stat.
func emLast(ss *Sim, name string) float64 {
	v := ss.StatVals(name)
	if len(v) == 0 {
		return math.NaN()
	}
	return v[len(v)-1]
}

// TestElectricPotential: the relaxed potential must be the Dirichlet Green's
// function k(1/r - 1/R), the same in both directions sampled.
// emFitMax is the outermost radius index the 1/r fit and the isotropy check
// use: beyond it the grounded wall is close enough to distort the field, and
// the wall is a cube.
const emFitMax = 3

func TestElectricPotential(t *testing.T) {
	const sz = 24
	ss := emSim(sz, func(s *Sim) {
		s.Config.Wavelength = 2 // radii 2, 4, 6, 8 inside a 24 box
		ElectricPotential(s)
	})
	for range 3000 {
		emStep(ss)
	}
	wl := float64(ss.Config.Wavelength)
	for _, dir := range []string{StatRadialAxis, StatRadialDiag} {
		// least squares of A0 against 1/r: the Dirichlet solution is
		// k(1/r - 1/R), so A0 is linear in 1/r with slope k and intercept
		// -k/R. Only out to emFitMax: past that the wall dominates, and a
		// CUBICAL wall is not spherical -- it is 12 cubes away along the axis
		// but 21 along the diagonal, so no single R describes both.
		var sx, sy, sxx, sxy, n float64
		a := make([]float64, EMNRadii+1)
		for i := 1; i <= EMNRadii; i++ {
			a[i] = emLast(ss, StatRadialName("A0", dir, i))
			if i > emFitMax {
				continue
			}
			x := 1 / (wl * float64(i))
			sx += x
			sy += a[i]
			sxx += x * x
			sxy += x * a[i]
			n++
		}
		k := (n*sxy - sx*sy) / (n*sxx - sx*sx)
		b0 := (sy - k*sx) / n
		for i := 1; i <= EMNRadii; i++ {
			r := wl * float64(i)
			want := k/r + b0
			t.Logf("%s r=%4.1f: A0 %.5f  fit k/r - k/R %.5f  (%+.1f%%)",
				dir, r, a[i], want, 100*(a[i]/want-1))
			if i <= emFitMax && math.Abs(a[i]/want-1) > 0.05 {
				t.Errorf("%s r=%g: A0 %g, want %g for the 1/r form", dir, r, a[i], want)
			}
		}
		t.Logf("%s fit: k = %.4f, R = %.2f cubes (box half-width %d)", dir, k, -k/b0, sz/2)
	}
	// E is the gradient, so it loses the constant and should go as 1/r^2.
	// Only the middle band is a fair test: at r = 2 you are still inside the
	// charge, which NeighAverage27 smears over a 3x3x3 block, and at r = 8 the
	// grounded wall is close -- and a cubical wall is 12 cubes away along the
	// axis but 21 along the diagonal, so the box itself is not spherical there.
	for _, dir := range []string{StatRadialAxis, StatRadialDiag} {
		for i := 1; i < EMNRadii; i++ {
			e0 := emLast(ss, StatRadialName("Emag", dir, i))
			e1 := emLast(ss, StatRadialName("Emag", dir, i+1))
			r0, r1 := wl*float64(i), wl*float64(i+1)
			pw := math.Log(e0/e1) / math.Log(r1/r0)
			t.Logf("%s |E| r=%.0f->%.0f: %.4e -> %.4e, power %.2f (want 2)", dir, r0, r1, e0, e1, pw)
			if i == 2 && math.Abs(pw-2) > 0.15 {
				t.Errorf("%s: |E| goes as r^-%.2f between r=%g and %g, want r^-2", dir, pw, r0, r1)
			}
		}
	}
	// the point of an isotropic stencil: a spherical field must not know which
	// way the lattice axes point. Same band, for the same reasons.
	for i := 1; i <= EMNRadii; i++ {
		ax := emLast(ss, StatRadialName("A0", StatRadialAxis, i))
		dg := emLast(ss, StatRadialName("A0", StatRadialDiag, i))
		t.Logf("anisotropy at r=%2.0f: axis %.5f diag %.5f  ratio %.4f", wl*float64(i), ax, dg, ax/dg)
		if i >= 2 && i <= emFitMax && math.Abs(ax/dg-1) > 0.02 {
			t.Errorf("A0 at r=%g differs by %.1f%% between axis and diagonal",
				wl*float64(i), 100*math.Abs(ax/dg-1))
		}
	}
}

// emRMS is the root mean square of a variable over the interior.
func emRMS(sz int32, vr EMStates) float64 {
	var s float64
	cur := int(GetCtx(0).CurState)
	n := 0
	for z := int32(1); z <= sz; z++ {
		for y := int32(1); y <= sz; y++ {
			for x := int32(1); x <= sz; x++ {
				v := float64(State.Value(int(z), int(y), int(x), int(vr), cur))
				s += v * v
				n++
			}
		}
	}
	return math.Sqrt(s / float64(n))
}

// emLine samples a variable along X at fixed y, z.
func emLine(sz int32, vr EMStates, y, z int32) []float64 {
	cur := int(GetCtx(0).CurState)
	v := make([]float64, sz)
	for x := int32(1); x <= sz; x++ {
		v[x-1] = float64(State.Value(int(z), int(y), int(x), int(vr), cur))
	}
	return v
}

// TestPolarizedPhoton: only the vector potential is set, and everything else
// follows. E must come out along the polarization, B along the other
// transverse direction, and nothing along the direction of travel.
//
// |E|/|B| is c only in the continuum: E comes from a time derivative whose
// lattice symbol is 2 sin(k/2) and B from a space derivative with a different
// one, so the ratio is c times a factor that goes to 1 as the wavelength grows.
// The test is that it converges, at the second-order rate.
func TestPolarizedPhoton(t *testing.T) {
	const sz = 32
	for _, tc := range []struct {
		name           string
		pol            math32.Dims
		wantE, wantB   EMStates
		otherE, otherB EMStates
	}{
		{"Y", math32.Y, EY, BZ, EZ, BY},
		{"Z", math32.Z, EZ, BY, EY, BZ},
	} {
		var errs [2]float64
		for wi, wl := range []float32{8, 16} {
			ss := emSim(sz, func(s *Sim) {
				s.Config.Polarization = tc.pol
				s.Config.Wavelength = wl
				s.Config.PacketWidth = 2 * wl
				PolarizedPhoton(s)
			})
			for range 20 {
				emStep(ss)
			}
			e := emRMS(sz, tc.wantE)
			b := emRMS(sz, tc.wantB)
			eo := emRMS(sz, tc.otherE)
			bo := emRMS(sz, tc.otherB)
			ex := emRMS(sz, EX)
			bx := emRMS(sz, BX)
			c := float64(ss.Params.C)
			errs[wi] = math.Abs(e/b/c - 1)
			t.Logf("%s-polarized wl=%2.0f: |%v|=%.4f |%v|=%.4f  E/B=%.4f (c=%.2f, %+.1f%%)   other transverse %.1e/%.1e  along travel %.1e/%.1e",
				tc.name, wl, tc.wantE, e, tc.wantB, b, e/b, c, 100*(e/b/c-1), eo, bo, ex, bx)
			if eo > 1e-6*e || bo > 1e-6*b {
				t.Errorf("%s: the other transverse direction is not empty: E %g B %g", tc.name, eo, bo)
			}
			if ex > 1e-6*e || bx > 1e-6*b {
				t.Errorf("%s: something along the direction of travel: E %g B %g", tc.name, ex, bx)
			}
		}
		if errs[1] > 0.5*errs[0] {
			t.Errorf("%s: |E|/|B| error %.3f -> %.3f on doubling the wavelength, want 4x better",
				tc.name, errs[0], errs[1])
		}
	}
}

// TestCircularPolarization: adding the two transverse components a quarter
// cycle apart must give a transverse |E| that barely varies along the packet,
// where a linear wave swings to zero twice a wavelength.
func TestCircularPolarization(t *testing.T) {
	const sz = 32
	ripple := func(init func(*Sim)) (float64, *Sim) {
		ss := emSim(sz, func(s *Sim) {
			s.Config.Wavelength = 8
			s.Config.PacketWidth = 40 // near flat: envelope ripple would mask it
			init(s)
		})
		emStep(ss) // E is kernel-written: one step, before the packet moves
		ey := emLine(sz, EY, sz/2, sz/2)
		ez := emLine(sz, EZ, sz/2, sz/2)
		// over one wavelength at the packet center, where the envelope is flat
		lo, hi := math.Inf(1), 0.0
		ctr := int(float32(sz) * 0.25)
		for x := ctr - 4; x <= ctr+4; x++ {
			m := ey[x]*ey[x] + ez[x]*ez[x]
			lo = math.Min(lo, m)
			hi = math.Max(hi, m)
		}
		return (hi - lo) / (hi + lo), ss
	}
	lin, _ := ripple(PolarizedPhoton)
	cir, _ := ripple(CircularPolarization)
	t.Logf("|E_transverse|^2 ripple over one wavelength: linear %.3f, circular %.3f", lin, cir)
	if lin < 0.9 {
		t.Errorf("a linear wave should swing to zero: ripple %g", lin)
	}
	if cir > 0.1 {
		t.Errorf("a circular wave should have near-constant |E|: ripple %g", cir)
	}
}

// TestStandingWave: in a travelling wave E and B rise and fall together; in a
// standing one their nodes are a quarter wavelength apart, so where one is
// large the other is small.
func TestStandingWave(t *testing.T) {
	const sz = 48
	corr := func(init func(*Sim), nst int) float64 {
		ss := emSim(sz, func(s *Sim) {
			// 3 whole wavelengths in the box. Short wavelengths cost the
			// travelling case its correlation: E is built from the velocity at
			// the half step, so E and B are slightly staggered in time, which
			// at wl=8 already pulls the correlation down to 0.81.
			s.Config.Wavelength = 16
			s.Config.PacketWidth = 40
			init(s)
		})
		for range nst {
			emStep(ss)
		}
		e := emLine(sz, EY, sz/2, sz/2)
		b := emLine(sz, BZ, sz/2, sz/2)
		var se, sb, see, sbb, seb, n float64
		for x := range int(sz) {
			ea, ba := math.Abs(e[x]), math.Abs(b[x])
			se += ea
			sb += ba
			see += ea * ea
			sbb += ba * ba
			seb += ea * ba
			n++
		}
		cov := seb/n - (se/n)*(sb/n)
		return cov / math.Sqrt((see/n-(se/n)*(se/n))*(sbb/n-(sb/n)*(sb/n)))
	}
	trav := corr(PolarizedPhoton, 5)
	stand := corr(StandingWave, 5)
	t.Logf("correlation of |E| with |B| along X: travelling %+.3f, standing %+.3f", trav, stand)
	if trav < 0.9 {
		t.Errorf("travelling wave: |E| and |B| should move together, got %g", trav)
	}
	if stand > -0.5 {
		t.Errorf("standing wave: |E| and |B| nodes should be a quarter wave apart, got %g", stand)
	}
}

// TestMovingCharge: a charged line moving along itself makes B circle it, with
// |B|/|E| = v/c^2. At rest there is no B at all.
//
// Measured a few cubes out, not over the whole box: A0 relaxes while A obeys
// the wave equation, so the two meet the damped wall differently and the ratio
// drifts upward as the wall gets close -- 1.01 at r=2 but 1.22 at r=8 of 12.
func TestMovingCharge(t *testing.T) {
	const sz = 24
	at := func(ss *Sim, vr EMStates, d int) float64 {
		c := int(sz / 2)
		return float64(State.Value(c, c+d, c, int(vr), int(GetCtx(0).CurState)))
	}
	run := func(v float32) *Sim {
		ss := emSim(sz, func(s *Sim) {
			s.Config.Velocity.X = v
			MovingCharge(s)
		})
		for range 1500 {
			emStep(ss)
		}
		return ss
	}
	rest := run(0)
	t.Logf("at rest:  E_y = %+.5f, B_z = %+.5e", at(rest, EY, 4), at(rest, BZ, 4))
	if b := math.Abs(at(rest, BZ, 4)); b > 1e-12 {
		t.Errorf("a charge at rest should make no magnetic field at all: %g", b)
	}
	v := float32(0.25)
	ss := run(v)
	c := float64(ss.Params.C)
	want := float64(v) / (c * c)
	for _, d := range []int{2, 3, 4, 6, 8} {
		e, b := at(ss, EY, d), at(ss, BZ, d)
		t.Logf("v=%.2f r=%d: E_y=%+.5f B_z=%+.5f  B/E=%.4f (want v/c^2 = %.4f)", v, d, e, b, b/e, want)
		if d <= 4 {
			if b <= 0 {
				t.Errorf("r=%d: B should circulate as x cross y, got B_z = %g", d, b)
			}
			if math.Abs(b/e/want-1) > 0.05 {
				t.Errorf("r=%d: |B|/|E| = %g, want v/c^2 = %g", d, b/e, want)
			}
		}
	}
}
