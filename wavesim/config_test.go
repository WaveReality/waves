package wavesim

import (
	"math"
	"testing"
)

// cfgSim builds a sim big enough for a travelling packet, then runs one of the
// ElectroweakConfigs init functions on it.
func cfgSim(sz int32, init func(*Sim)) *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU = false
	ss.Config.GUI = false
	ss.Config.Equation = Electroweak
	ss.Config.Size.Set(sz, 4, 4) // thin in y,z: the packets are X slabs
	ss.ConfigSim()
	ss.Params.Edges = EdgesWrap
	ss.StateVars = EWStatesN
	ss.ConfigState()
	ss.InitFunc = init
	ss.Init()
	return ss
}

// cfgCentroid returns the energy-weighted X centroid of a state variable,
// which tracks the packet even as it disperses.
func cfgCentroid(sz int32, vr int32) float64 {
	var num, den float64
	cur := int(GetCtx(0).CurState)
	for x := int32(1); x <= sz; x++ {
		v := float64(State.Value(2, 2, int(x), int(vr), cur))
		w := v * v
		num += w * float64(x)
		den += w
	}
	if den == 0 {
		return 0
	}
	return num / den
}

func cfgSumSq(sz int32, vr int32) float64 {
	var s float64
	cur := int(GetCtx(0).CurState)
	for x := int32(1); x <= sz; x++ {
		v := float64(State.Value(2, 2, int(x), int(vr), cur))
		s += v * v
	}
	return s
}

// TestConfigHiggsSymmetric: the field must fall off the unstable maximum and
// reach the vacuum magnitude.
func TestConfigHiggsSymmetric(t *testing.T) {
	const sz = 32
	ss := cfgSim(sz, HiggsSymmetric)
	v2 := float64(ss.Params.HiggsV) * float64(ss.Params.HiggsV)
	start := float64(State.Value(2, 2, 8, int(EWHmag), int(GetCtx(0).CurState)))
	var peak, last float64
	for i := range 4000 {
		ewStep(ss)
		m := float64(State.Value(2, 2, 8, int(EWHmag), int(GetCtx(0).CurState)))
		peak = math.Max(peak, m)
		last = m
		_ = i
	}
	t.Logf("|Phi|^2 at one site: start %.3e -> peak %.4f, final %.4f   (v^2 = %.4f)",
		start, peak, last, v2)
	// the overshoot goes to |Phi| = sqrt(2) v, where V returns to zero: with no
	// dissipation the field rings between there and the origin forever.
	if peak < 1.5*v2 || peak > 2.5*v2 {
		t.Errorf("overshoot %g, expected about 2 v^2 = %g", peak, 2*v2)
	}
	if start > 0.01*v2 {
		t.Errorf("did not start near the symmetric point: %g vs v^2 = %g", start, v2)
	}
	if peak < 0.5*v2 {
		t.Errorf("never reached the vacuum: peak %g vs v^2 = %g", peak, v2)
	}
	if math.IsNaN(last) {
		t.Errorf("blew up")
	}
}

// cfgSpeed runs one of the pulse configs at a given Config.Amplitude and
// returns the packet speed in units of c, plus the lowest fraction of the
// vacuum |Phi|^2 found under the packet along the way.
func cfgSpeed(sz int32, init func(*Sim), probe int, amp float32, nst int) (vc, hfrac float64) {
	ss := cfgSim(sz, func(s *Sim) {
		s.Config.Amplitude = amp
		init(s)
	})
	v2 := float64(ss.Params.HiggsV) * float64(ss.Params.HiggsV)
	hfrac = math.Inf(1)
	// the probe variables are written BY the kernel, so they are still zero
	// before the first step: take the baseline after one step, not before.
	ewStep(ss)
	c0 := cfgCentroid(sz, int32(probe))
	for range nst {
		ewStep(ss)
		// |Phi|^2 weighted by the local packet intensity: what the packet
		// itself sits in, not the box average.
		var num, den float64
		cur := int(GetCtx(0).CurState)
		for x := int32(1); x <= sz; x++ {
			a := float64(State.Value(2, 2, int(x), probe, cur))
			h := float64(State.Value(2, 2, int(x), int(EWHmag), cur))
			num += a * a * h
			den += a * a
		}
		if den > 0 {
			hfrac = math.Min(hfrac, num/den/v2)
		}
	}
	c1 := cfgCentroid(sz, int32(probe))
	return (c1 - c0) / float64(nst) / float64(ss.Params.C), hfrac
}

// cfgGroupVel is the lattice group velocity in units of c, for a carrier of
// the given wavelength and a field of the given mass. omega = c sqrt(khat^2 +
// m^2) with khat = 2 sin(k/2), so dom/dk = c khat cos(k/2) / sqrt(khat^2+m^2).
// The cos(k/2) is why even a massless packet comes out slightly subluminal.
func cfgGroupVel(wavelength, mass float64) float64 {
	k := 2 * math.Pi / wavelength
	kh := 2 * math.Sin(k/2)
	return kh * math.Cos(k/2) / math.Sqrt(kh*kh+mass*mass)
}

// TestConfigPulseSpeeds: the photon packet must travel at c and the Z packet
// measurably slower, at its group velocity.
//
// This runs at a small Config.Amplitude, because the tree-level group velocity
// is a statement about a LINEAR wave on an undisturbed vacuum. The configs
// themselves default to amplitude 1 so they read at the GUI's display scale,
// and there the Z is large enough to move the condensate it travels through;
// TestPulseCondensateBackReaction covers that regime.
func TestConfigPulseSpeeds(t *testing.T) {
	const sz = 96
	const amp = 0.02
	var speed [2]float64
	for ti, tc := range []struct {
		name  string
		init  func(*Sim)
		probe int
		mass  func(p *Parameters) float64
	}{
		{"photon", PhotonPulse, int(AYs), func(p *Parameters) float64 { return 0 }},
		{"Z", ZPulse, int(EWZY), func(p *Parameters) float64 { return float64(p.MZ) }},
	} {
		// the carrier comes from Config, which ElectroweakConfig sets, so the
		// prediction tracks whatever the config chooses rather than a literal.
		ss := cfgSim(sz, func(s *Sim) {})
		wl := float64(ss.Config.Wavelength)
		m := tc.mass(ss.Params)
		if tc.name == "Z" { // ZPulse raises the scale, so take m_Z from there
			ss = cfgSim(sz, func(s *Sim) { ewDemoScale(s, 0.388) })
			m = tc.mass(ss.Params)
		}
		wantV := cfgGroupVel(wl, m)

		got, hfrac := cfgSpeed(sz, tc.init, tc.probe, amp, 100)
		speed[ti] = got
		t.Logf("%-7s amp %.2f: v = %.5f c, want %.5f c (%+.1f%%), condensate %.0f%% of vacuum",
			tc.name, amp, got, wantV, 100*(got/wantV-1), 100*hfrac)
		if math.Abs(got/wantV-1) > 0.08 {
			t.Errorf("%s speed %g c, want %g c", tc.name, got, wantV)
		}
		// The lattice already puts a massless packet at cos(k/2) = 0.98 c, and
		// the finite packet loses a few percent more to dispersion; the point
		// is that it is near c and far above the Z.
		if m == 0 && got < 0.9 {
			t.Errorf("photon should move at essentially c: %g c", got)
		}
		if m > 0 && got > 0.85 {
			t.Errorf("Z should be visibly subluminal, got %g c", got)
		}
	}
	// the claim the demo makes, and the one that holds at any amplitude
	if speed[1] > speed[0]-0.1 {
		t.Errorf("Z at %.4f c should lag the photon at %.4f c by a clear margin",
			speed[1], speed[0])
	}
}

// TestPulseCondensateBackReaction measures what the pulses do to the Higgs
// condensate they travel through, which at the default amplitude is the
// difference between a linear wave and a strongly nonlinear one.
//
// The photon direction is the combination that Q = T^3 + Y annihilates, so it
// costs the condensate nothing no matter how large it gets -- at amplitude 1
// the packet is many times v and |Phi|^2 does not move, and its speed is the
// same as in the linear regime. That is exact masslessness shown directly,
// rather than inferred from a propagation speed.
//
// The Z direction does not annihilate the vacuum, so a packet comparable to v
// partially restores the symmetry underneath itself. The local mass falls with
// the condensate and the packet speeds up, which is why the tree-level group
// velocity only applies to a small Z.
func TestPulseCondensateBackReaction(t *testing.T) {
	const sz = 96
	for _, tc := range []struct {
		name    string
		init    func(*Sim)
		probe   int
		minDrop float64 // how far below vacuum the condensate must go at amp 1
		maxDrop float64
	}{
		{"photon", PhotonPulse, int(AYs), 0.99, 1.01},
		{"Z", ZPulse, int(EWZY), 0.0, 0.70},
	} {
		vLin, hLin := cfgSpeed(sz, tc.init, tc.probe, 0.02, 100)
		vBig, hBig := cfgSpeed(sz, tc.init, tc.probe, 1, 100)
		t.Logf("%-7s amp 0.02: v = %.5f c, condensate %3.0f%%   |   amp 1: v = %.5f c, condensate %3.0f%%  (%+.1f%% faster)",
			tc.name, vLin, 100*hLin, vBig, 100*hBig, 100*(vBig/vLin-1))
		if hBig < tc.minDrop || hBig > tc.maxDrop {
			t.Errorf("%s left the condensate at %.0f%% of vacuum, want %.0f-%.0f%%",
				tc.name, 100*hBig, 100*tc.minDrop, 100*tc.maxDrop)
		}
		if tc.name == "photon" && math.Abs(vBig/vLin-1) > 0.01 {
			t.Errorf("photon speed moved %.1f%% between amplitudes: it should not "+
				"care, since it does not couple to the condensate", 100*(vBig/vLin-1))
		}
		if tc.name == "Z" && vBig <= vLin {
			t.Errorf("Z at amplitude 1 (%g c) should outrun the linear Z (%g c): "+
				"it has melted its own mass down", vBig, vLin)
		}
	}
}

// TestConfigWCollision: W^3 must be generated where the two packets overlap,
// and only when the Yang-Mills self-coupling is on.
func TestConfigWCollision(t *testing.T) {
	const sz = 64
	var res [2]float64
	for i, ym := range []bool{false, true} {
		ss := cfgSim(sz, func(s *Sim) {
			s.Params.YangMills.SetBool(ym)
			s.Params.Update()
			WCollision(s)
		})
		// Sum over ALL FOUR W^3 components. Both packets are transversely
		// polarised along y and travel along x, so W^{b mu} d_mu vanishes
		// identically (nothing varies along the polarisation) and the transport
		// term is silent. What survives is the transpose term, which sources the
		// LONGITUDINAL W^3_x -- probing only W^3_y finds nothing.
		w3 := func() float64 {
			t := 0.0
			for c := int32(0); c < 4; c++ {
				t += cfgSumSq(sz, int32(EWW30s)+c)
			}
			return t
		}
		if s0 := w3(); s0 > 1e-20 {
			t.Fatalf("W^3 not initially zero: %g", s0)
		}
		peak := 0.0
		for range 600 {
			ewStep(ss)
			peak = math.Max(peak, w3())
		}
		res[i] = peak
		t.Logf("YangMills=%-5v  peak sum(W^3^2) generated = %.4e", ym, peak)
	}
	if res[1] < 1e4*res[0] {
		t.Errorf("collision generated %g with Yang-Mills vs %g without; expected the "+
			"non-abelian coupling to dominate", res[1], res[0])
	}
}
