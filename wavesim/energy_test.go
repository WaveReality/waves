package wavesim

import (
	"fmt"
	"math"
	"testing"
)

// ewFillWave lays down a single plane wave of wavelength L along a general
// direction, in all nine spatial W components with staggered phases. W_0 and
// B_0 stay zero, as do all velocities -- the sector ElectroweakStaticEnergy
// covers.
//
// It must be ONE plane wave, not a product of two. A product is a superposition
// of modes at |k1+k2| and |k1-k2|, whose force contributions can partly cancel
// at the probe site; the relative mismatch then measures that cancellation
// rather than the discretization, and stops converging.
func ewFillWave(sz int32, L float64, amp float32) {
	n := int(sz) + 2
	k := 2 * math.Pi / L
	d := [3]float64{0.5774, 0.5774, 0.5774}
	for z := range n {
		for y := range n {
			for x := range n {
				arg := k * (d[0]*float64(x) + d[1]*float64(y) + d[2]*float64(z))
				for a := 0; a < 3; a++ {
					for c := 0; c < 3; c++ {
						v := float64(amp) * math.Sin(arg+0.9*float64(a)+1.7*float64(c))
						for t := range 2 {
							State.Set(float32(v), z, y, x, int(EWW1Xs)+a*8+c, t)
						}
					}
				}
			}
		}
	}
}

// ewKernelForce returns the kernel's acceleration on one field at one site,
// divided by c^2 -- the quantity that must equal -dU/dphi. All velocities start
// at zero, so one step leaves exactly the force in the VELOCITY variable (vr
// must be the *v enum, not the *s one: the position after a step is
// W + force, which is a different thing entirely).
func ewKernelForce(ss *Sim, vr EWStates, x, y, z int32) float64 {
	save := State.Clone()
	ewStep(ss)
	cur := int(GetCtx(0).CurState)
	f := State.Value(int(z), int(y), int(x), int(vr), cur)
	State.CopyFrom(save)
	GetCtx(0).Init()
	return float64(f) / float64(ss.Params.CSq)
}

// ewProbes are the (site, adjoint, component) points the gradient check is
// evaluated at. A single probe is fragile: at large amplitude the force passes
// through zero at particular sites, and a RELATIVE error measured against a
// near-zero reference is meaningless. An RMS over many probes is stable.
func ewProbes(sz int32) [][5]int32 {
	var pr [][5]int32
	for _, c := range [][3]int32{{4, 4, 4}, {5, 4, 6}, {6, 5, 4}, {4, 6, 5}} {
		for a := int32(0); a < 3; a++ {
			for d := int32(0); d < 3; d++ {
				pr = append(pr, [5]int32{c[0], c[1], c[2], a, d})
			}
		}
	}
	return pr
}

// ewGradMismatch returns the RMS relative difference between the kernel's force
// and -dU/dphi over all probes, plus the RMS force scale.
func ewGradMismatch(sz int32, L float64, amp float32, ym bool) (float64, float64) {
	ss := ymSim(sz)
	ss.Params.YangMills.SetBool(ym)
	ss.Params.Update()
	ewFillWave(sz, L, amp)

	pr := ewProbes(sz)
	// one step gives every kernel force at once
	save := State.Clone()
	ewStep(ss)
	cur := int(GetCtx(0).CurState)
	fk := make([]float64, len(pr))
	for i, q := range pr {
		vv := int(EWW1Xv) + int(q[3])*8 + int(q[4])
		fk[i] = float64(State.Value(int(q[2]), int(q[1]), int(q[0]), vv, cur)) / float64(ss.Params.CSq)
	}
	State.CopyFrom(save)
	GetCtx(0).Init()

	const h = 1e-3
	var sd, sf float64
	for i, q := range pr {
		vs := int(EWW1Xs) + int(q[3])*8 + int(q[4])
		base := State.Value(int(q[2]), int(q[1]), int(q[0]), vs, 0)
		set := func(v float32) {
			for tt := range 2 {
				State.Set(v, int(q[2]), int(q[1]), int(q[0]), vs, tt)
			}
		}
		set(base + h)
		up := ElectroweakStaticEnergyNear(q[0], q[1], q[2])
		set(base - h)
		dn := ElectroweakStaticEnergyNear(q[0], q[1], q[2])
		set(base)
		fn := -(up - dn) / (2 * h)
		sd += (fk[i] - fn) * (fk[i] - fn)
		sf += fn * fn
	}
	n := float64(len(pr))
	return math.Sqrt(sd/n) / math.Sqrt(sf/n), math.Sqrt(sf / n)
}

// TestElectroweakForceIsEnergyGradient is the check that pins down every
// coefficient of the Yang-Mills self-coupling.
//
// The kernel's force must be -dU/dphi for the static energy U. It cannot be
// EXACTLY so at finite lattice spacing, because Laplacian19 and Gradient10 are
// not adjoint-compatible: their Fourier symbols differ by O(k^2) (2.2% at
// |k| = 0.3, 23% at |k| = 1.0). So this is a CONVERGENCE test -- the mismatch
// must fall as O(a^2) when the configuration is made smoother.
//
// That is exactly what separates a discretization artifact from a wrong
// coefficient: an error in any Yang-Mills term is O(1) in the field amplitude
// and does NOT shrink with wavelength, while the stencil mismatch does. The
// amplitude is deliberately large enough that the self-coupling dominates the
// force, so the test is sensitive to it; the control run with the terms
// disabled shows the mismatch then stalls instead of converging.
func TestElectroweakForceIsEnergyGradient(t *testing.T) {
	const sz = 10
	amp := float32(0.02)
	Ls := []float64{16, 32, 64}

	for _, ym := range []bool{true, false} {
		var prev float64
		var ratios []float64
		for i, L := range Ls {
			rel, fs := ewGradMismatch(sz, L, amp, ym)
			sc := ""
			if i > 0 {
				ratios = append(ratios, prev/rel)
				sc = fmt.Sprintf("  shrank %.2fx", prev/rel)
			}
			prev = rel
			t.Logf("YangMills=%-5v  L=%3.0f  RMS force=%.4e  RMS mismatch=%.4e%s", ym, L, fs, rel, sc)
		}
		if ym {
			for i, r := range ratios {
				if math.IsNaN(r) || r < 3.0 {
					t.Errorf("step %d: mismatch shrank only %.3fx when the wavelength doubled; "+
						"O(a^2) needs ~4, so a Yang-Mills coefficient is wrong", i, r)
				}
			}
		} else {
			// control: the energy still contains the self-coupling while the
			// force no longer does, so the mismatch must NOT converge
			for _, r := range ratios {
				if r > 3.0 {
					t.Errorf("control converged at %.3fx -- the test is not actually "+
						"sensitive to the Yang-Mills terms", r)
				}
			}
		}
	}
}
