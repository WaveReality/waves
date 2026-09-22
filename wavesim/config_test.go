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

// TestConfigPulseSpeeds: the photon packet must travel at c and the Z packet
// measurably slower, at its group velocity.
func TestConfigPulseSpeeds(t *testing.T) {
	const sz = 96
	for _, tc := range []struct {
		name  string
		init  func(*Sim)
		probe int32
		mass  func(p *Parameters) float64
	}{
		{"photon", PhotonPulse, int32(AYs), func(p *Parameters) float64 { return 0 }},
		{"Z", ZPulse, int32(EWZY), func(p *Parameters) float64 { return float64(p.MZ) }},
	} {
		ss := cfgSim(sz, tc.init)
		p := ss.Params
		c := float64(p.C)
		m := tc.mass(p)
		// lattice group velocity: omega = c sqrt(khat^2 + m^2) with
		// khat = 2 sin(k/2), so dom/dk = c khat cos(k/2) / sqrt(khat^2 + m^2).
		// The cos(k/2) is why even a massless packet is slightly subluminal.
		// the carrier comes from Config, which ElectroweakConfig sets, so the
		// prediction tracks whatever the config chooses rather than a literal.
		k := 2 * math.Pi / float64(ss.Config.Wavelength)
		khat := 2 * math.Sin(k/2)
		wantV := c * khat * math.Cos(k/2) / math.Sqrt(khat*khat+m*m)

		// AYs / EWZY are written BY the kernel, so they are still zero before
		// the first step: measure the baseline after one step, not before.
		// And keep the run short enough that the packet cannot wrap the box.
		ewStep(ss)
		nst := 100
		c0 := cfgCentroid(sz, tc.probe)
		for range nst {
			ewStep(ss)
		}
		c1 := cfgCentroid(sz, tc.probe)
		got := (c1 - c0) / float64(nst)
		t.Logf("%-7s centroid %.2f -> %.2f over %d steps: v = %.5f  want %.5f  (v/c = %.4f)",
			tc.name, c0, c1, nst, got, wantV, got/c)
		if math.Abs(got/wantV-1) > 0.08 {
			t.Errorf("%s speed %g, want %g", tc.name, got, wantV)
		}
		// The lattice already puts a massless packet at cos(k/2) = 0.98 c, and
		// the finite packet loses a few percent more to dispersion; the point is
		// that it is near c and far above the Z.
		if m == 0 && got/c < 0.9 {
			t.Errorf("photon should move at essentially c: %g vs %g", got, c)
		}
		if m > 0 && got/c > 0.85 {
			t.Errorf("Z should be visibly subluminal, got %g c", got/c)
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
