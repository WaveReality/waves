package wavesim

import (
	"math"
	"testing"
)

// cfgSim builds a sim big enough for a travelling packet and runs one of the
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

// cfgCentroid is the energy-weighted X centroid of a state variable, which
// tracks the packet as it disperses.
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

// cfgSpeed runs a pulse config at a given Config.Amplitude, returning the
// packet speed in units of c and the lowest fraction of vacuum |Phi|^2 seen
// under the packet.
func cfgSpeed(sz int32, init func(*Sim), probe int, amp float32, nst int) (vc, hfrac float64) {
	ss := cfgSim(sz, func(s *Sim) {
		s.Config.Amplitude = amp
		init(s)
	})
	v2 := float64(ss.Params.HiggsV) * float64(ss.Params.HiggsV)
	hfrac = math.Inf(1)
	// probes are kernel-written: baseline after one step, not before.
	ewStep(ss)
	c0 := cfgCentroid(sz, int32(probe))
	for range nst {
		ewStep(ss)
		// weighted by local packet intensity: what the packet sits in.
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

// cfgGroupVel is the lattice group velocity in units of c: from omega =
// c sqrt(khat^2 + m^2) with khat = 2 sin(k/2), dom/dk = c khat cos(k/2) /
// sqrt(khat^2+m^2). The cos(k/2) is why even a massless packet is subluminal.
func cfgGroupVel(wavelength, mass float64) float64 {
	k := 2 * math.Pi / wavelength
	kh := 2 * math.Sin(k/2)
	return kh * math.Cos(k/2) / math.Sqrt(kh*kh+mass*mass)
}

// TestConfigPulseSpeeds: the photon must travel at c and the Z measurably
// slower, at its group velocity. Runs at small Config.Amplitude, since the
// tree-level velocity describes a LINEAR wave on an undisturbed vacuum; the
// configs default to 1, covered by TestPulseCondensateBackReaction.
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
		// carrier from Config, so the prediction tracks it, not a literal.
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
		// the lattice alone puts a massless packet at cos(k/2) = 0.98 c, and
		// the finite packet loses a few percent more to dispersion.
		if m == 0 && got < 0.9 {
			t.Errorf("photon should move at essentially c: %g c", got)
		}
		if m > 0 && got > 0.85 {
			t.Errorf("Z should be visibly subluminal, got %g c", got)
		}
	}
	// the demo's claim, and the one that holds at any amplitude
	if speed[1] > speed[0]-0.1 {
		t.Errorf("Z at %.4f c should lag the photon at %.4f c by a clear margin",
			speed[1], speed[0])
	}
}

// TestPulseCondensateBackReaction measures what the pulses do to the
// condensate they travel through. Q annihilates the photon direction, so a
// photon packet costs it nothing however large: at amplitude 1 the packet is
// many times v, |Phi|^2 does not move and the speed matches the linear regime
// -- exact masslessness shown directly rather than inferred. The Z direction
// does not, so a packet comparable to v partially restores the symmetry under
// itself, loses mass and speeds up.
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
		// all FOUR W^3 components: the packets are polarised along y and
		// travel along x, so the transport term is silent and what survives
		// sources the LONGITUDINAL W^3_x. Probing W^3_y alone finds nothing.
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
