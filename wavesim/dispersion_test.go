// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/math32"
)

// dispMean is the mean of a rate stat, skipping the leading samples that read
// zero while the [StatGroupVelWindow] window fills.
func dispMean(vals []float64) float64 {
	if len(vals) <= StatGroupVelWindow {
		return 0
	}
	vals = vals[StatGroupVelWindow:]
	tot := 0.0
	for _, v := range vals {
		tot += v
	}
	return tot / float64(len(vals))
}

// TestStatWidth checks the width measurement itself against gaussians of known
// sigma written straight into the state, before any equation is involved.
//
// The case that matters is the one centred on zero: half the packet sits at
// each end of the axis, and a plain second moment would call that nearly the
// width of the box. The circular form in [CircularWidth] does not care where
// the packet sits.
func TestStatWidth(t *testing.T) {
	const ln = 128
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU, ss.Config.GUI = false, false
	ss.Config.Equation = Wave
	ss.Config.Size.Set(ln, 1, 1)
	ss.ConfigSim()
	ss.Init()

	vr := WaveVel
	vri := int(vr.Int64())
	cur := int(GetCtx(0).CurState)
	vrs := []enums.Enum{vr}
	ctrs, wids, wts := make([]float64, 1), make([]float64, 1), make([]float64, 1)

	// a gaussian of known sigma, wrapped, at a known centre
	set := func(sigma, ctr float64) {
		State.SetZeros()
		for x := range ln {
			d := float64(x) - ctr
			for d > ln/2 {
				d -= ln
			}
			for d < -ln/2 {
				d += ln
			}
			State.Set(float32(math.Exp(-d*d/(2*sigma*sigma))), 1, 1, x+1, vri, cur)
		}
	}
	for _, sigma := range []float64{2, 4, 8, 16} {
		for _, ctr := range []float64{ln / 2, 0} {
			set(sigma, ctr)
			StateMoments(ss.Config.Size, math32.X, int32(cur), false, vrs, ctrs, wids, wts)
			err := wids[0]/sigma - 1
			where := "centred"
			if ctr == 0 {
				where = "straddling the wrap"
			}
			t.Logf("sigma %4.0f %-19s: measured %7.4f (%+.3f%%), centroid %6.2f",
				sigma, where, wids[0], 100*err, ctrs[0])
			if math.Abs(err) > 0.01 {
				t.Errorf("sigma %g %s: measured %g, off by %.2f%%", sigma, where, wids[0], 100*err)
			}
		}
	}
	// filling the box is the most spread a distribution can be, and the width
	// must report that rather than running away
	State.SetZeros()
	for x := range ln {
		State.Set(1, 1, 1, x+1, vri, cur)
	}
	StateMoments(ss.Config.Size, math32.X, int32(cur), false, vrs, ctrs, wids, wts)
	want := CircularWidthMax * ln
	t.Logf("uniform: measured %.4f, saturates at L/sqrt(12) = %.4f", wids[0], want)
	if math.Abs(wids[0]-want) > 1.0e-6 {
		t.Errorf("uniform width %g, want the saturation value %g", wids[0], want)
	}
}

// TestPacketEnvelope sets the two envelopes against each other. Every
// travelling-packet config goes through [Sim.PacketDist], so [Config.PacketSlab]
// switches them all together: a planar slab, or a 3D gaussian blob.
//
// The contrast separates two things that both look like "spreading" on screen,
// and every equation records both: Vd along X is dispersion, Vd along Z is
// diffraction. A blob has a finite transverse extent, so it is a spread of
// TRANSVERSE wavenumbers, and it fans out whether or not the equation
// disperses at all. A slab has none, so what is left is the dispersion.
//
// Massless Weyl is the case that makes the point: it has exactly no dispersion,
// the slab holds its width to 1%, and the blob still grows by half again --
// along the direction of travel as well, since a tilted k has less of itself
// left along X. Schrodinger disperses isotropically instead, so its blob grows
// the same amount every way and its slab grows by the same amount as the blob
// does along X.
func TestPacketEnvelope(t *testing.T) {
	const nst = 80
	for _, tc := range []struct {
		name string
		eq   Equations
		init func(*Sim)
		disp bool // does the equation itself disperse
	}{
		{name: "Weyl neutrino", eq: Weyl, init: NeutrinoPacket},
		{name: "Schrodinger", eq: Schrodinger, init: FreePacket, disp: true},
	} {
		var xg, zg [2]float64 // fractional width growth, [slab] and [blob]
		for i, slab := range []bool{true, false} {
			ss := &Sim{}
			ss.Config = &Config{}
			ss.Config.Defaults()
			ss.Config.GPU, ss.Config.GUI = false, false
			ss.Config.Equation = tc.eq
			ss.Config.Size.Set(128, 48, 48)
			ss.Config.Wavelength, ss.Config.PacketWidth = 16, 12
			ss.Config.PacketSlab = slab
			ss.ConfigSim()
			ss.Params.ThreeD.SetBool(true) // diffraction needs the other two axes
			ss.Params.Update()
			var probe enums.Enum = CabMag
			if tc.eq == Weyl {
				probe = WeylMag
			}
			ss.InitFunc = tc.init
			ss.Init()
			for range nst {
				ss.StepRun()
			}
			xw := ss.StatVals(StatWidthName(probe, math32.X))
			zw := ss.StatVals(StatWidthName(probe, math32.Z))
			xg[i] = xw[len(xw)-1]/xw[0] - 1
			zg[i] = zw[len(zw)-1]/zw[0] - 1
			shape := "slab"
			if !slab {
				shape = "blob"
			}
			t.Logf("%-14s %s: X %5.2f -> %5.2f (%+5.1f%%)   Z %5.2f -> %5.2f (%+5.1f%%)",
				tc.name, shape, xw[0], xw[len(xw)-1], 100*xg[i], zw[0], zw[len(zw)-1], 100*zg[i])
		}
		// the slab is uniform across Z, so its Z width sits at the saturation
		// value and must not move; the blob is localized and must fan out
		if math.Abs(zg[0]) > 1.0e-6 {
			t.Errorf("%s: the slab is uniform across Z, so its Z width cannot change: %+.3f%%",
				tc.name, 100*zg[0])
		}
		if zg[1] < 0.1 {
			t.Errorf("%s: a blob of finite transverse extent must diffract, but its Z width grew only %+.1f%%",
				tc.name, 100*zg[1])
		}
		if tc.disp {
			// dispersion does not know about direction: the blob spreads the
			// same amount every way, and along X the slab matches it
			if math.Abs(zg[1]/xg[1]-1) > 0.1 {
				t.Errorf("%s: dispersion is isotropic, so the blob should grow alike in X (%+.1f%%) and Z (%+.1f%%)",
					tc.name, 100*xg[1], 100*zg[1])
			}
			if math.Abs(xg[0]/xg[1]-1) > 0.1 {
				t.Errorf("%s: along X the slab (%+.1f%%) should match the blob (%+.1f%%): same dispersion, and the slab has nothing else",
					tc.name, 100*xg[0], 100*xg[1])
			}
		} else {
			// no dispersion at all: the slab holds, and everything the blob
			// does -- including along X -- is diffraction
			if xg[0] > 0.03 {
				t.Errorf("%s: a slab under an equation with no dispersion must hold its width, but it grew %+.1f%%",
					tc.name, 100*xg[0])
			}
			if xg[1] < 5*xg[0] {
				t.Errorf("%s: the blob should spread along X far more than the slab (%+.1f%% vs %+.1f%%), from transverse k alone",
					tc.name, 100*xg[1], 100*xg[0])
			}
		}
	}
}

// dispCase is one equation and the packet to measure it with.
type dispCase struct {
	name  string
	eq    Equations
	probe enums.Enum
	init  func(*Sim) // nil for the equation's own default
	rest  bool       // started at rest, so there is nothing to compare Vg to
	maxVd float64    // 0 means no bound: logged but not asserted
	minVd float64
}

// TestDispersion is the cross-equation comparison: the same packet, the same
// box and the same measurement under every wave equation, so the numbers mean
// something against each other.
//
// The split it asserts is the structural one. An equation whose spatial
// operator is a GRADIENT -- WaveCDir, and Weyl with no mass -- carries a packet
// without changing its shape at all, because every k in it moves at the same
// speed. An equation whose operator is a LAPLACIAN does not, and neither does
// one with a mass term: those have a curved omega(k), so the packet's own
// spread of wavenumbers spreads the packet.
//
// The second-order equations pick up a further contribution from the lattice
// alone, which is why Wave is not at zero either despite being non-dispersive
// in the continuum.
//
// Both Weyl cases probe WeylMag, the conserved total, rather than one
// chirality: the mass trades the halves back and forth, so neither on its own
// is the particle.
//
// The Weyl packets read a small NEGATIVE Vd, and that is a sampling artefact
// rather than a packet pulling itself together. A slab with a carrier on it is
// not an eigenstate of the lattice equation, so its width breathes -- the
// neutrino by 3% and the electron by 7%, over something longer than this run --
// and a mean taken over part of a cycle lands wherever the cycle happens to be.
// Read Vd for a spreading rate only against a packet that is actually
// spreading; the width trace itself says the rest.
func TestDispersion(t *testing.T) {
	const sz = 384
	// enough steps for the packet to cover the same ground it used to at a
	// higher C: a rate averaged over less than a breathing cycle reads
	// whichever half of the cycle it landed in
	const nst = 300
	// a packet several wavelengths wide, so it is narrow in k and the lattice
	// curvature is not what is being measured. Slabs, at the Config default:
	// see TestPacketEnvelope for why a blob would not measure dispersion.
	const wl, pw = 16, 32
	got := map[string]float64{}
	for _, tc := range []dispCase{
		{name: "Wave", eq: Wave, probe: WavePos},
		{name: "KleinGordon", eq: KleinGordon, probe: WavePos},
		{name: "Maxwell photon", eq: Maxwell, probe: AYs, init: PolarizedPhoton},
		{name: "WaveC", eq: WaveC, probe: WaveCMag},
		{name: "WaveCDir", eq: WaveCDir, probe: WaveCMag, maxVd: 0.005},
		{name: "Schrodinger", eq: Schrodinger, probe: CabMag, init: FreePacket},
		{name: "KleinGordonC", eq: KleinGordonC, probe: CabAs, init: ChargedPacket},
		{name: "Weyl neutrino", eq: Weyl, probe: WeylMag, init: NeutrinoPacket, maxVd: 0.005},
		{name: "Weyl electron", eq: Weyl, probe: WeylMag, init: ElectronPacket},
		{name: "Dirac at rest", eq: Dirac, probe: DiracMag, init: SpinAtRest, rest: true, minVd: 0.004},
	} {
		ss := &Sim{}
		ss.Config = &Config{}
		ss.Config.Defaults()
		ss.Config.GPU, ss.Config.GUI = false, false
		ss.Config.Equation = tc.eq
		ss.Config.Size.Set(sz, 4, 4)
		ss.Config.Wavelength, ss.Config.PacketWidth = wl, pw
		ss.ConfigSim()
		if tc.init != nil {
			ss.InitFunc = tc.init
			ss.Init()
		}
		for range nst {
			ss.StepRun()
		}
		wd := ss.StatVals(StatWidthName(tc.probe, math32.X))
		if len(wd) == 0 {
			t.Errorf("%s: no width stat recorded", tc.name)
			continue
		}
		vg := dispMean(ss.StatVals(StatGroupVelName(tc.probe, math32.X)))
		vd := dispMean(ss.StatVals(StatDispersionName(tc.probe, math32.X)))
		grow := 0.0
		if wd[0] > 0 {
			grow = wd[len(wd)-1]/wd[0] - 1
		}
		at := ""
		if tc.rest {
			at = "  (at rest)"
		}
		t.Logf("%-14s Vg %+.4f  Vd %+.5f   width %6.3f -> %6.3f (%+.1f%%)%s",
			tc.name, vg, vd, wd[0], wd[len(wd)-1], 100*grow, at)
		if math.IsNaN(vd) {
			t.Errorf("%s: dispersion went bad", tc.name)
		}
		if tc.maxVd > 0 && math.Abs(vd) > tc.maxVd {
			t.Errorf("%s: a gradient operator carries every k at the same speed, so it "+
				"should not spread: Vd = %g, want under %g", tc.name, vd, tc.maxVd)
		}
		if tc.minVd > 0 && vd < tc.minVd {
			t.Errorf("%s: a massive particle spreads even sitting still: Vd = %g, want over %g",
				tc.name, vd, tc.minVd)
		}
		got[tc.name] = vd
	}
	// the headline contrast, as a ratio rather than two separate bounds
	if r := math.Abs(got["Wave"] / got["Weyl neutrino"]); r < 10 {
		t.Errorf("the massless Weyl packet should hold its shape far better than the "+
			"second-order wave: only %.1fx less dispersion", r)
	}
}
