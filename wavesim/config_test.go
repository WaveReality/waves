package wavesim

import (
	"math"
	"testing"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/math32"
)

// cfgSim builds a sim big enough for a travelling packet and runs one of the
// ElectroweakConfigs init functions on it.
func cfgSim(sz int32, init func(*Sim), probes ...enums.Enum) *Sim {
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
	ss.ElectroweakStats()
	for _, pv := range probes {
		ss.AddStat(ss.StatWeightedMean(cfgHmagName(pv), EWHmag, pv))
	}
	ss.InitFunc = init
	ss.Init()
	return ss
}

// cfgHmagName is the stat holding |Phi|^2 as weighted by the given packet.
func cfgHmagName(vr enums.Enum) string {
	return "Hmag" + vr.String()
}

// cfgStep advances one step and records the stats, as StepRun does.
func cfgStep(ss *Sim) {
	ewStep(ss)
}

// cfgVgMean is the mean group velocity, skipping the leading samples that
// StatGroupVel reads as zero while its window fills.
func cfgVgMean(vals []float64) float64 {
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

// TestConfigHiggsSymmetric: the field must fall off the unstable maximum and
// reach the vacuum magnitude.
func TestConfigHiggsSymmetric(t *testing.T) {
	ss := cfgSim(32, HiggsSymmetric)
	v2 := float64(ss.Params.HiggsV) * float64(ss.Params.HiggsV)
	for range 4000 {
		cfgStep(ss)
	}
	mag := ss.StatVals("Mean" + EWHmag.String())
	start, peak, last := mag[0], 0.0, mag[len(mag)-1]
	for _, m := range mag {
		peak = math.Max(peak, m)
	}
	t.Logf("mean |Phi|^2: start %.3e -> peak %.4f, final %.4f   (v^2 = %.4f)",
		start, peak, last, v2)
	// the overshoot goes to |Phi| = sqrt(2) v, where V returns to zero: with no
	// dissipation the field rings between there and the origin forever.
	if peak < 1.5*v2 || peak > 2.5*v2 {
		t.Errorf("overshoot %g, expected about 2 v^2 = %g", peak, 2*v2)
	}
	if start > 0.01*v2 {
		t.Errorf("did not start near the symmetric point: %g vs v^2 = %g", start, v2)
	}
	if math.IsNaN(last) {
		t.Errorf("blew up")
	}
}

// cfgSpeed runs a pulse config at a given Config.Amplitude and reads back the
// recorded stats: the mean group velocity in units of C, and the lowest
// fraction of vacuum |Phi|^2 seen under the packet.
func cfgSpeed(sz int32, init func(*Sim), probe enums.Enum, amp float32, nst int) (vc, hfrac float64) {
	ss := cfgSim(sz, func(s *Sim) {
		s.Config.Amplitude = amp
		init(s)
	}, probe)
	v2 := float64(ss.Params.HiggsV) * float64(ss.Params.HiggsV)
	for range nst {
		cfgStep(ss)
	}
	vc = cfgVgMean(ss.StatVals(StatGroupVelName(probe, math32.X)))
	hfrac = math.Inf(1)
	for _, h := range ss.StatVals(cfgHmagName(probe)) {
		hfrac = math.Min(hfrac, h/v2)
	}
	return
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
		probe enums.Enum
		mass  func(p *Parameters) float64
	}{
		{"photon", PhotonPulse, AYs, func(p *Parameters) float64 { return 0 }},
		{"Z", ZPulse, EWZY, func(p *Parameters) float64 { return float64(p.MZ) }},
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
		probe   enums.Enum
		minDrop float64 // how far below vacuum the condensate must go at amp 1
		maxDrop float64
	}{
		{"photon", PhotonPulse, AYs, 0.99, 1.01},
		{"Z", ZPulse, EWZY, 0.0, 0.70},
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
	var res [2]float64
	for i, ym := range []bool{false, true} {
		ss := cfgSim(64, func(s *Sim) {
			s.Params.YangMills.SetBool(ym)
			s.Params.Update()
			WCollision(s)
		})
		// the stat sums ALL FOUR W^3 components: the packets are polarised
		// along y and travel along x, so the transport term is silent and what
		// survives sources the LONGITUDINAL W^3_x. Probing W^3_y finds nothing.
		for range 600 {
			cfgStep(ss)
		}
		// no initial-zero guard is needed: if WCollision wrote W^3 directly,
		// the YangMills=false control below would show it too.
		for _, v := range ss.StatVals(EWW3SqStat) {
			res[i] = math.Max(res[i], v)
		}
		t.Logf("YangMills=%-5v  peak sum(W^3^2) generated = %.4e", ym, res[i])
	}
	if res[1] < 1e4*res[0] {
		t.Errorf("collision generated %g with Yang-Mills vs %g without; expected the "+
			"non-abelian coupling to dominate", res[1], res[0])
	}
}
