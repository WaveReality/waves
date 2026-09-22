// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"testing"
)

// thSetTemp sets Temp and recomputes derived values, as the GUI form does.
func thSetTemp(ss *Sim, t float32) {
	ss.Params.Temp = t
	ss.Params.Update()
}

// thMag is the stat holding the mean |Phi|^2 over the interior.
var thMag = "Mean" + EWHmag.String()

// thTail returns the values of the named stat recorded since index from.
func thTail(ss *Sim, name string, from int) []float64 {
	v := ss.StatVals(name)
	if from >= len(v) {
		return nil
	}
	return v[from:]
}

func thMin(v []float64) float64 {
	m := math.Inf(1)
	for _, x := range v {
		m = math.Min(m, x)
	}
	return m
}

func thMax(v []float64) float64 {
	m := math.Inf(-1)
	for _, x := range v {
		m = math.Max(m, x)
	}
	return m
}

// TestThermalVEV checks the params against the algebra: mu^2 -> mu^2 - c T^2,
// v(T) = sqrt(mu^2(T)/lambda), and the boson masses following v(T) to zero.
func TestThermalVEV(t *testing.T) {
	ss := ewSim(4)
	p := ss.Params
	mu, lam, c := float64(p.HiggsMu), float64(p.HiggsLambda), float64(p.ThermalC)
	tc := mu / math.Sqrt(c)
	if math.Abs(float64(p.TempCrit)-tc) > 1e-6 {
		t.Errorf("TempCrit = %g, want mu/sqrt(c) = %g", p.TempCrit, tc)
	}
	// T exactly at Tc is a float32 knife edge: mu^2(T) is zero up to rounding,
	// deciding between v = 0 and v ~ 1e-5. Nothing physical, so step around.
	for _, f := range []float64{0, 0.5, 0.9, 0.99, 1.01, 1.2, 2.0} {
		thSetTemp(ss, float32(f*tc))
		musq := mu*mu - c*float64(p.Temp)*float64(p.Temp)
		want := 0.0
		if musq > 0 {
			want = math.Sqrt(musq / lam)
		}
		got := float64(p.HiggsV)
		if math.Abs(got-want) > 1e-6+1e-4*want {
			t.Errorf("T = %.2f Tc: v(T) = %g, want %g", f, got, want)
		}
		wantMW := float64(p.GW) * want / 2
		if math.Abs(float64(p.MW)-wantMW) > 1e-6+1e-4*wantMW {
			t.Errorf("T = %.2f Tc: m_W = %g, want %g", f, p.MW, wantMW)
		}
		t.Logf("T = %.2f Tc: mu^2(T) = %+.3e  v(T) = %.5f  m_W = %.5f  m_Z = %.5f",
			f, musq, p.HiggsV, p.MW, p.MZ)
	}
	thSetTemp(ss, 0)
}

// TestThermalVEVStatic: v(T) must be a fixed point of the KERNEL, not just of
// the formula -- TestHiggsVacuumStatic with Temp carrying part of mu^2.
func TestThermalVEVStatic(t *testing.T) {
	const sz = 6
	ss := ewSim(sz)
	defer thSetTemp(ss, 0)
	for _, f := range []float32{0.5, 0.9} {
		thSetTemp(ss, f*ss.Params.TempCrit)
		v := ss.Params.HiggsV
		State.SetZeros()
		ss.Fill(EWHs0a, Both, v)
		WrapEdges()
		for range 500 {
			ewStep(ss)
		}
		drift := math.Abs(float64(ewGet(EWHs0a) - v))
		vel := math.Abs(float64(ewGet(EWHv0a)))
		t.Logf("T = %.1f Tc: v(T) = %.5f, drift after 500 steps %.3e, velocity %.3e", f, v, drift, vel)
		if drift > 1e-5 || vel > 1e-6 {
			t.Errorf("T = %.1f Tc: thermal vacuum not stationary: drift %g vel %g", f, drift, vel)
		}
	}
}

// TestThermalRestoration: heated above TempCrit, a field left in the old
// broken vacuum is no longer in a minimum and must fall back to zero.
func TestThermalRestoration(t *testing.T) {
	ss := ewSim(6)
	defer thSetTemp(ss, 0)
	v0 := ss.Params.HiggsV // the cold VEV, where ewSim leaves the field
	ewStep(ss)             // EWHmag is kernel-written: step once to fill it
	start := ss.StatVals(thMag)[0]
	thSetTemp(ss, 1.5*ss.Params.TempCrit)
	if ss.Params.HiggsV != 0 {
		t.Fatalf("above TempCrit the VEV should be zero, got %g", ss.Params.HiggsV)
	}
	if ss.Params.MW != 0 || ss.Params.MZ != 0 {
		t.Errorf("above TempCrit the gauge bosons should be massless, got m_W %g m_Z %g",
			ss.Params.MW, ss.Params.MZ)
	}
	from := len(ss.StatVals(thMag))
	for range 4000 {
		ewStep(ss)
	}
	low := thMin(thTail(ss, thMag, from))
	t.Logf("heated to 1.5 Tc from v = %.5f: |Phi|^2 %.3e -> %.3e at its lowest (started %.3e)",
		v0, start, low, float64(v0)*float64(v0))
	if low > 0.05*start {
		t.Errorf("symmetry not restored: |Phi|^2 only fell from %g to %g", start, low)
	}
}

// TestThermalQuench runs what the GUI does: HiggsSymmetric above TempCrit,
// where zero is a real minimum and the field stays put, then cooled to zero so
// it breaks out to the new vacuum.
func TestThermalQuench(t *testing.T) {
	ss := cfgSim(16, func(s *Sim) {
		s.Params.Temp = 1.25 * s.Params.TempCrit
		s.Params.Update()
		HiggsSymmetric(s)
	})
	defer thSetTemp(ss, 0)
	ewStep(ss) // EWHmag is kernel-written: step once to fill it
	hot := ss.StatVals(thMag)[0]
	for range 1000 { // above TempCrit this is a real minimum: must stay put
		ewStep(ss)
	}
	held := ss.StatVals(thMag)
	t.Logf("hot at %.2f Tc: |Phi|^2 %.3e -> %.3e after 1000 steps",
		ss.Params.Temp/ss.Params.TempCrit, hot, held[len(held)-1])
	if held[len(held)-1] > 4*hot+1e-12 {
		t.Errorf("symmetric phase should be stable above TempCrit: %g -> %g",
			hot, held[len(held)-1])
	}

	thSetTemp(ss, 0) // quench
	v2 := float64(ss.Params.HiggsV) * float64(ss.Params.HiggsV)
	from := len(held)
	for range 4000 {
		ewStep(ss)
	}
	cold := thTail(ss, thMag, from)
	peak, final := thMax(cold), cold[len(cold)-1]
	t.Logf("quenched to T = 0: |Phi|^2 peak %.5f, final %.5f  (v^2 = %.5f)", peak, final, v2)
	if peak < 0.5*v2 {
		t.Errorf("quench did not reach the broken vacuum: peak %g vs v^2 %g", peak, v2)
	}
	if math.IsNaN(final) {
		t.Errorf("quench blew up")
	}
}
