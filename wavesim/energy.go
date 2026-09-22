// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import "cogentcore.org/core/math32"

// ElectroweakStaticEnergy is the potential (no time derivative) part of the
// electroweak energy, evaluated over the interior of the lattice:
//
//	U = sum_x [ (1/2) sum_j |D_j Psi|^2                     Higgs gradient
//	          + V(mag)                                      Higgs potential
//	          + (1/4) sum_{a,j,k} (F^a_jk)^2                W magnetic
//	          + (1/4) sum_{j,k} (Fb_jk)^2                   B magnetic
//	          + (1/2) sum_a (div W^a)^2 + (1/2)(div B)^2 ]  gauge fixing
//
// with the full non-abelian field strength
//
//	F^a_jk = d_j W^a_k - d_k W^a_j + g eps^abc W^b_j W^c_k
//
// The gauge-fixing term is the xi = 1 (Feynman) choice the kernel assumes;
// with the magnetic term it makes the abelian part of -dU/dW the plain
// Laplacian.
//
// A host-side diagnostic for checking the kernel's force is -dU/dphi. NOT
// exact at finite spacing: Laplacian19 and Gradient10 are not adjoint
// compatible (Fourier symbols differ by O(k^2)), so the agreement is a
// continuum-limit statement, which TestElectroweakForceIsEnergyGradient
// measures.
//
// Only W_0 = B_0 = 0 is covered: with a time component the conserved
// functional is the indefinite Feynman-gauge one, whose cubic term mixes
// kinetic and potential pieces.
func ElectroweakStaticEnergy() float64 {
	ctx := GetCtx(0)
	p := &Params[0]
	sz := ctx.Size.V()
	tot := 0.0
	for z := int32(1); z <= sz.Z; z++ {
		for y := int32(1); y <= sz.Y; y++ {
			for x := int32(1); x <= sz.X; x++ {
				tot += float64(ewSiteEnergy(x, y, z, p.GW, p))
			}
		}
	}
	return tot
}

// ElectroweakStaticEnergyNear sums the energy over the 27 sites centred on
// (x,y,z). Every stencil reaches at most one cube, so the derivative of the
// TOTAL energy w.r.t. one site lives entirely in this block. That makes the
// numerical-gradient check boundary-independent, so the test field need not be
// lattice-periodic.
func ElectroweakStaticEnergyNear(x, y, z int32) float64 {
	p := &Params[0]
	tot := 0.0
	for dz := int32(-1); dz <= 1; dz++ {
		for dy := int32(-1); dy <= 1; dy++ {
			for dx := int32(-1); dx <= 1; dx++ {
				tot += float64(ewSiteEnergy(x+dx, y+dy, z+dz, p.GW, p))
			}
		}
	}
	return tot
}

// ewWVec returns the adjoint triplet (W^1_c, W^2_c, W^3_c) for spatial
// component c (0=X, 1=Y, 2=Z) at one site.
func ewWVec(x, y, z, c int32, tidx int32) math32.Vector3 {
	return math32.Vec3(
		State.Value(int(z), int(y), int(x), int(EWW1Xs)+int(c), int(tidx)),
		State.Value(int(z), int(y), int(x), int(EWW2Xs)+int(c), int(tidx)),
		State.Value(int(z), int(y), int(x), int(EWW3Xs)+int(c), int(tidx)))
}

func ewSiteEnergy(x, y, z int32, g float32, p *Parameters) float32 {
	prv := int32(0) // the tests evaluate energy on slice 0
	psi := math32.Vec4(
		State.Value(int(z), int(y), int(x), int(EWHsCa), 0),
		State.Value(int(z), int(y), int(x), int(EWHsCb), 0),
		State.Value(int(z), int(y), int(x), int(EWHs0a), 0),
		State.Value(int(z), int(y), int(x), int(EWHs0b), 0))

	// ---- Higgs: covariant gradient + potential ----------------------------
	gCa := Gradient10(x, y, z, int32(EWHsCa), prv)
	gCb := Gradient10(x, y, z, int32(EWHsCb), prv)
	g0a := Gradient10(x, y, z, int32(EWHs0a), prv)
	g0b := Gradient10(x, y, z, int32(EWHs0b), prv)
	e := float32(0)
	for c := int32(0); c < 3; c++ {
		var d math32.Vector4
		switch c {
		case 0:
			d = math32.Vec4(gCa.X, gCb.X, g0a.X, g0b.X)
		case 1:
			d = math32.Vec4(gCa.Y, gCb.Y, g0a.Y, g0b.Y)
		default:
			d = math32.Vec4(gCa.Z, gCb.Z, g0a.Z, g0b.Z)
		}
		w := ewWVec(x, y, z, c, prv)
		b := State.Value(int(z), int(y), int(x), int(EWBXs)+int(c), 0)
		d = d.Add(EWGaugeAct(w.X, w.Y, w.Z, b, psi))
		e += 0.5 * (d.X*d.X + d.Y*d.Y + d.Z*d.Z + d.W*d.W)
	}
	mag := psi.X*psi.X + psi.Y*psi.Y + psi.Z*psi.Z + psi.W*psi.W
	e += -0.5*p.HiggsMuSq*mag + 0.25*p.HiggsLambda*mag*mag

	// ---- gauge field strengths -------------------------------------------
	// gradients of the three spatial components of each field
	var gW [3][3]math32.Vector3 // [adjoint a][component c]
	for a := 0; a < 3; a++ {
		for c := 0; c < 3; c++ {
			gW[a][c] = Gradient10(x, y, z, int32(EWW1Xs)+int32(a*8+c), prv)
		}
	}
	var gB [3]math32.Vector3
	for c := 0; c < 3; c++ {
		gB[c] = Gradient10(x, y, z, int32(EWBXs)+int32(c), prv)
	}
	comp := func(v math32.Vector3, i int) float32 {
		switch i {
		case 0:
			return v.X
		case 1:
			return v.Y
		}
		return v.Z
	}
	wv := [3]math32.Vector3{ewWVec(x, y, z, 0, prv), ewWVec(x, y, z, 1, prv), ewWVec(x, y, z, 2, prv)}
	// magnetic: (1/4) F_jk F_jk, summed over BOTH orderings of (j,k)
	for j := 0; j < 3; j++ {
		for k := 0; k < 3; k++ {
			if j == k {
				continue
			}
			cr := slCross(wv[j], wv[k]) // eps^abc W^b_j W^c_k
			for a := 0; a < 3; a++ {
				f := comp(gW[a][k], j) - comp(gW[a][j], k) + g*comp(cr, a)
				e += 0.25 * f * f
			}
			fb := comp(gB[k], j) - comp(gB[j], k)
			e += 0.25 * fb * fb
		}
	}
	// gauge fixing: (1/2)(div W^a)^2 and (1/2)(div B)^2
	for a := 0; a < 3; a++ {
		dv := gW[a][0].X + gW[a][1].Y + gW[a][2].Z
		e += 0.5 * dv * dv
	}
	dvb := gB[0].X + gB[1].Y + gB[2].Z
	e += 0.5 * dvb * dvb
	return e
}

func slCross(a, b math32.Vector3) math32.Vector3 {
	return math32.Vec3(a.Y*b.Z-a.Z*b.Y, a.Z*b.X-a.X*b.Z, a.X*b.Y-a.Y*b.X)
}
