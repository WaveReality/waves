// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"math"
	"math/rand/v2"
	"testing"

	"cogentcore.org/core/math32"
)

// Tests for the 26/18-neighbor stencils. Each one is checked on two levels:
// that the WEIGHTS satisfy the normalization and isotropy identities, and that
// the actual Laplacian19 / Gradient10 code paths reproduce known derivatives.
//
// The normalization conditions make a stencil correct. The isotropy conditions
// make its leading error direction-independent, which is the whole point of
// using 26 neighbors instead of 6 -- a stencil can be perfectly normalized and
// still be badly anisotropic, which is exactly what 1/d^2 (Laplacian) and 1/d
// (gradient) weighting are.

const stSize = 6 // interior cells per side; full array is stSize+2

func stencilSim() *Sim {
	ss := &Sim{}
	ss.Config = &Config{}
	ss.Config.Defaults()
	ss.Config.GPU = false
	ss.Config.GUI = false
	ss.Config.Equation = Electroweak
	ss.Config.Size.Set(stSize, stSize, stSize)
	ss.ConfigVars()
	ss.Params = &Params[0]
	ss.StateVars = EWStatesN
	ss.ConfigState()
	GetCtx(0).Init()
	State.SetZeros()
	return ss
}

// stCtr is the cell the stencils are evaluated at; fill uses coordinates
// relative to it so the analytic derivatives are evaluated at the origin.
const stCtr = (stSize + 2) / 2

// stFill writes f(x,y,z) into state variable v over the WHOLE array, edges
// included, so the stencil at stCtr has real values in every neighbor.
func stFill(v int32, f func(x, y, z float64) float64) {
	n := stSize + 2
	for z := range n {
		for y := range n {
			for x := range n {
				State.Set(float32(f(float64(x-stCtr), float64(y-stCtr), float64(z-stCtr))),
					z, y, x, int(v), 0)
			}
		}
	}
}

func stLap(v int32) float64 {
	c := int32(stCtr)
	return float64(Laplacian19(c, c, c, v, 0, State.Value(stCtr, stCtr, stCtr, int(v), 0)))
}

func stGrad(v int32) math32.Vector3 {
	c := int32(stCtr)
	return Gradient10(c, c, c, v, 0)
}

// --- Laplacian19 -----------------------------------------------------------

func TestLaplacian19Exact(t *testing.T) {
	stencilSim()
	v := int32(EWHsCa)
	// Tolerances are per-case because the sum runs over 26 float32 terms: the
	// cancelling cases are exact, while r^2 accumulates the round-off of three
	// separate x^2 contributions.
	for _, tc := range []struct {
		name string
		f    func(x, y, z float64) float64
		want float64
		tol  float64
	}{
		{"constant", func(x, y, z float64) float64 { return 3 }, 0, 1e-6},
		{"linear x", func(x, y, z float64) float64 { return x }, 0, 1e-6},
		{"xy", func(x, y, z float64) float64 { return x * y }, 0, 1e-6},
		// the quadratic anisotropy: a stencil with the wrong second moments
		// returns something non-zero here
		{"2x^2-y^2-z^2", func(x, y, z float64) float64 { return 2*x*x - y*y - z*z }, 0, 1e-6},
		{"x^2", func(x, y, z float64) float64 { return x * x }, 2, 1e-5},
		{"r^2", func(x, y, z float64) float64 { return x*x + y*y + z*z }, 6, 1e-5},
	} {
		stFill(v, tc.f)
		if got := stLap(v); math.Abs(got-tc.want) > tc.tol {
			t.Errorf("lap(%s) = %g, want %g (tol %g)", tc.name, got, tc.want, tc.tol)
		}
	}
}

// TestLaplacian19Isotropy checks the two weight identities directly.
func TestLaplacian19Isotropy(t *testing.T) {
	stencilSim()
	var norm, q4, q22 float64
	for j := range 26 {
		w := float64(NeighWts.Value(int(LaplacianWts), j))
		dx := float64(NeighOffs.Value(j, int(math32.X)))
		dy := float64(NeighOffs.Value(j, int(math32.Y)))
		norm += w * dx * dx
		q4 += w * dx * dx * dx * dx
		q22 += w * dx * dx * dy * dy
	}
	t.Logf("sum w dx^2 = %.6f (want 2)   sum w dx^4 = %.6f   3*sum w dx^2dy^2 = %.6f", norm, q4, 3*q22)
	if math.Abs(norm-2) > 1e-6 {
		t.Errorf("normalization sum w dx^2 = %g, want 2", norm)
	}
	if math.Abs(3*q22/q4-1) > 1e-6 {
		t.Errorf("4th-order isotropy: 3*sum w dx^2dy^2 / sum w dx^4 = %g, want 1", 3*q22/q4)
	}
}

// --- Gradient10 ------------------------------------------------------------

func TestGradient10Exact(t *testing.T) {
	stencilSim()
	v := int32(EWHsCa)
	rng := rand.New(rand.NewPCG(1, 2))
	worst := 0.0
	for range 100 { // gradient of a.r must be exactly a, in any direction
		a := [3]float64{rng.NormFloat64(), rng.NormFloat64(), rng.NormFloat64()}
		stFill(v, func(x, y, z float64) float64 { return a[0]*x + a[1]*y + a[2]*z })
		g := stGrad(v)
		for i, gi := range []float32{g.X, g.Y, g.Z} {
			worst = math.Max(worst, math.Abs(float64(gi)-a[i]))
		}
	}
	t.Logf("gradient of a linear function: max error %.3e", worst)
	if worst > 1e-5 {
		t.Errorf("gradient not exact on linear functions: max error %g", worst)
	}
	// even functions must give exactly zero (the stencil is antipodal, so all
	// even-order error terms cancel -- that is what makes it 2nd order)
	for _, tc := range []struct {
		name string
		f    func(x, y, z float64) float64
	}{
		{"x^2", func(x, y, z float64) float64 { return x * x }},
		{"xy", func(x, y, z float64) float64 { return x * y }},
		{"r^2", func(x, y, z float64) float64 { return x*x + y*y + z*z }},
	} {
		stFill(v, tc.f)
		g := stGrad(v)
		if m := math.Max(math.Abs(float64(g.X)), math.Max(math.Abs(float64(g.Y)),
			math.Abs(float64(g.Z)))); m > 1e-6 {
			t.Errorf("grad(%s) at origin = %v, want 0 (antipodal pairing)", tc.name, g)
		}
	}
}

// TestGradient10Isotropy checks the weight identities for the +x stencil.
// Note sum_j w_j is pinned at 1/2 by the normalization for ANY ratios, since
// every pair sits at x = +-1: only the isotropy is tunable.
func TestGradient10Isotropy(t *testing.T) {
	stencilSim()
	var t111, t122 float64
	for j := range 9 {
		w := float64(NeighWts.Value(int(Grad10Wts), j))
		ex := float64(FaceOffs.Value(int(math32.X), int(Plus1), j, int(math32.X)))
		ey := float64(FaceOffs.Value(int(math32.X), int(Plus1), j, int(math32.Y)))
		t111 += w * ex * ex * ex
		t122 += w * ex * ey * ey
	}
	t.Logf("T111 = %.6f (want 0.5)   3*T122 = %.6f", t111, 3*t122)
	if math.Abs(t111-0.5) > 1e-6 {
		t.Errorf("normalization T111 = %g, want 0.5", t111)
	}
	if math.Abs(3*t122/t111-1) > 1e-6 {
		t.Errorf("3rd-order isotropy: 3*T122/T111 = %g, want 1", 3*t122/t111)
	}
}

// --- plane-wave anisotropy, the end-to-end statement -----------------------

// TestStencilPlaneWaveIsotropy sweeps wave directions and bounds how much the
// effective wavenumber each stencil reports varies with direction. This is the
// quantity that actually matters: it is the directional spread of the wave
// dispersion relation, i.e. how much faster a wave travels along a lattice
// diagonal than along an axis.
func TestStencilPlaneWaveIsotropy(t *testing.T) {
	stencilSim()
	rng := rand.New(rand.NewPCG(3, 4))
	dirs := make([][3]float64, 2000)
	for i := range dirs {
		d := [3]float64{rng.NormFloat64(), rng.NormFloat64(), rng.NormFloat64()}
		n := math.Sqrt(d[0]*d[0] + d[1]*d[1] + d[2]*d[2])
		dirs[i] = [3]float64{d[0] / n, d[1] / n, d[2] / n}
	}
	for _, km := range []float64{0.3, 0.8} {
		lapLo, lapHi := math.Inf(1), math.Inf(-1)
		grdLo, grdHi := math.Inf(1), math.Inf(-1)
		for _, d := range dirs {
			k := [3]float64{km * d[0], km * d[1], km * d[2]}
			// Laplacian: returns -k_eff^2 on exp(i k.r)
			s := 0.0
			for j := range 26 {
				w := float64(NeighWts.Value(int(LaplacianWts), j))
				ph := 0.0
				for a := range 3 {
					ph += k[a] * float64(NeighOffs.Value(j, a))
				}
				s += w * (math.Cos(ph) - 1)
			}
			r := -s / (km * km)
			lapLo, lapHi = math.Min(lapLo, r), math.Max(lapHi, r)
			// Gradient: returns i*k_eff
			var ke [3]float64
			for ax := range 3 {
				g := 0.0
				for j := range 9 {
					w := float64(NeighWts.Value(int(Grad10Wts), j))
					ph := 0.0
					for a := range 3 {
						ph += k[a] * float64(FaceOffs.Value(ax, int(Plus1), j, a))
					}
					g += w * 2 * math.Sin(ph)
				}
				ke[ax] = g
			}
			r = math.Sqrt(ke[0]*ke[0]+ke[1]*ke[1]+ke[2]*ke[2]) / km
			grdLo, grdHi = math.Min(grdLo, r), math.Max(grdHi, r)
		}
		lapSp, grdSp := 100*(lapHi-lapLo), 100*(grdHi-grdLo)
		t.Logf("|k|=%.1f  Laplacian19 spread %.4f%%   Gradient10 spread %.4f%%", km, lapSp, grdSp)
		// Bounds are ~3x the measured isotropic values, well under the 1.97%
		// and 5.04% that the 1/d^2 and 1/d weightings produced at |k|=0.8.
		if km == 0.8 {
			if lapSp > 0.2 {
				t.Errorf("Laplacian19 directional spread %.4f%% at |k|=0.8, want < 0.2%%", lapSp)
			}
			if grdSp > 0.6 {
				t.Errorf("Gradient10 directional spread %.4f%% at |k|=0.8, want < 0.6%%", grdSp)
			}
		}
	}
}

// TestLaplacian19CFL pins the stability limit. The leapfrog update is stable
// while C^2 |lambda_min| < 4, where lambda_min is the most negative eigenvalue
// of the stencil. For the isotropic weights that is exactly -16/3, attained at
// k = (pi, pi, 0), so C must stay below sqrt(3)/2.
//
// The older anisotropic 1/d^2 weighting had lambda_min = -4 exactly and so
// allowed C < 1: isotropy costs about 13% of the timestep. If this test starts
// failing after a weight change, the C limit in Parameters needs updating with
// it.
func TestLaplacian19CFL(t *testing.T) {
	stencilSim()
	eig := func(k [3]float64) float64 {
		s := 0.0
		for j := range 26 {
			w := float64(NeighWts.Value(int(LaplacianWts), j))
			ph := 0.0
			for a := range 3 {
				ph += k[a] * float64(NeighOffs.Value(j, a))
			}
			s += w * (math.Cos(ph) - 1)
		}
		return s
	}
	const n = 60
	worst := 0.0
	for i := range n + 1 {
		for j := range n + 1 {
			for l := range n + 1 {
				k := [3]float64{math.Pi * float64(i) / n, math.Pi * float64(j) / n,
					math.Pi * float64(l) / n}
				worst = math.Min(worst, eig(k))
			}
		}
	}
	atCorner := eig([3]float64{math.Pi, math.Pi, 0})
	cMax := 2 / math.Sqrt(-worst)
	t.Logf("lambda_min = %.6f (want -16/3 = %.6f, attained at k=(pi,pi,0): %.6f)",
		worst, -16.0/3.0, atCorner)
	t.Logf("CFL limit: C < %.6f (sqrt(3)/2 = %.6f)", cMax, math.Sqrt(3)/2)
	// tolerances accommodate the float32 weight tensor (~1e-7 relative)
	if math.Abs(worst+16.0/3.0) > 1e-5 {
		t.Errorf("spectral radius %g, want -16/3 = %g", worst, -16.0/3.0)
	}
	if math.Abs(cMax-math.Sqrt(3)/2) > 1e-6 {
		t.Errorf("CFL limit %g, want sqrt(3)/2 = %g", cMax, math.Sqrt(3)/2)
	}
	if p := Params[0].C; float64(p) >= cMax {
		t.Errorf("Params.C default %g is at or above the CFL limit %g", p, cMax)
	}
}

// --- Curl10 / Divergence10 -------------------------------------------------

// stFill3 writes a vector field into three CONSECUTIVE state variables
// starting at v, which is the layout Curl10 and Divergence10 expect.
func stFill3(v int32, f func(x, y, z float64) [3]float64) {
	n := stSize + 2
	for z := range n {
		for y := range n {
			for x := range n {
				a := f(float64(x-stCtr), float64(y-stCtr), float64(z-stCtr))
				for c := range 3 {
					State.Set(float32(a[c]), z, y, x, int(v)+c, 0)
				}
			}
		}
	}
}

// TestCurl10 checks the curl on fields whose answer is known exactly.
// Curl10 is built entirely from Grad10Wts and FaceOffs, so it inherits the
// gradient's normalization and isotropy; what needs checking here is that the
// six directional derivatives are paired into the right antisymmetric
// combinations, and with the right signs.
func TestCurl10(t *testing.T) {
	stencilSim()
	v := int32(EWHsCa)
	c := int32(stCtr)

	for _, tc := range []struct {
		name string
		f    func(x, y, z float64) [3]float64
		want [3]float64
	}{
		{"constant", func(x, y, z float64) [3]float64 { return [3]float64{2, -3, 5} },
			[3]float64{0, 0, 0}},
		{"rigid rotation (-y,x,0)", func(x, y, z float64) [3]float64 { return [3]float64{-y, x, 0} },
			[3]float64{0, 0, 2}},
		{"(0,0,y)", func(x, y, z float64) [3]float64 { return [3]float64{0, 0, y} },
			[3]float64{1, 0, 0}},
		{"(z,0,0)", func(x, y, z float64) [3]float64 { return [3]float64{z, 0, 0} },
			[3]float64{0, 1, 0}},
		// curl of a gradient vanishes identically; a quadratic potential makes
		// the gradient field linear, so this is exact rather than asymptotic
		{"grad(x^2+2yz)", func(x, y, z float64) [3]float64 {
			return [3]float64{2 * x, 2 * z, 2 * y}
		}, [3]float64{0, 0, 0}},
	} {
		stFill3(v, tc.f)
		g := Curl10(c, c, c, v, 0)
		got := [3]float64{float64(g.X), float64(g.Y), float64(g.Z)}
		for i := range 3 {
			if math.Abs(got[i]-tc.want[i]) > 1e-5 {
				t.Errorf("curl(%s) = %v, want %v", tc.name, got, tc.want)
				break
			}
		}
	}

	// a general linear field A_i = M_ij r_j has curl_i = eps_ijk M_kj exactly
	rng := rand.New(rand.NewPCG(7, 8))
	worst := 0.0
	for range 50 {
		var m [3][3]float64
		for i := range 3 {
			for j := range 3 {
				m[i][j] = rng.NormFloat64()
			}
		}
		stFill3(v, func(x, y, z float64) [3]float64 {
			r := [3]float64{x, y, z}
			var a [3]float64
			for i := range 3 {
				for j := range 3 {
					a[i] += m[i][j] * r[j]
				}
			}
			return a
		})
		g := Curl10(c, c, c, v, 0)
		want := [3]float64{m[2][1] - m[1][2], m[0][2] - m[2][0], m[1][0] - m[0][1]}
		got := [3]float64{float64(g.X), float64(g.Y), float64(g.Z)}
		for i := range 3 {
			worst = math.Max(worst, math.Abs(got[i]-want[i]))
		}
	}
	t.Logf("curl of 50 random linear vector fields: max error %.3e", worst)
	if worst > 1e-5 {
		t.Errorf("curl not exact on linear fields: max error %g", worst)
	}
}

// TestDivergence10 checks div on linear fields, where it equals trace(M).
// It also checks the per-direction out-params, which the previous version of
// the function never wrote.
func TestDivergence10(t *testing.T) {
	stencilSim()
	v := int32(EWHsCa)
	c := int32(stCtr)
	rng := rand.New(rand.NewPCG(9, 10))
	worst, worstC := 0.0, 0.0
	for range 50 {
		var m [3][3]float64
		for i := range 3 {
			for j := range 3 {
				m[i][j] = rng.NormFloat64()
			}
		}
		stFill3(v, func(x, y, z float64) [3]float64 {
			r := [3]float64{x, y, z}
			var a [3]float64
			for i := range 3 {
				for j := range 3 {
					a[i] += m[i][j] * r[j]
				}
			}
			return a
		})
		var dx, dy, dz float32
		got := float64(Divergence10(c, c, c, v, 0, &dx, &dy, &dz))
		want := m[0][0] + m[1][1] + m[2][2]
		worst = math.Max(worst, math.Abs(got-want))
		// the out-params are the diagonal terms d_i A_i individually
		for i, d := range []float32{dx, dy, dz} {
			worstC = math.Max(worstC, math.Abs(float64(d)-m[i][i]))
		}
	}
	t.Logf("divergence of 50 random linear fields: max error %.3e (components %.3e)", worst, worstC)
	if worst > 1e-5 || worstC > 1e-5 {
		t.Errorf("divergence not exact on linear fields: %g (components %g)", worst, worstC)
	}
}
