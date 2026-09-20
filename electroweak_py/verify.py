"""
Correctness tests for higgs_gauge.py. Run this before trusting any physics.

[1] analytic forces == -grad(energy_potential)            (central differences)
[2] velocity-Verlet energy error is bounded and O(dt^2)
[3] lattice gauge invariance is O(a^2)
[4] the photon direction is EXACTLY flat at the VEV, the Z direction is not
[5] the k=0 mass spectrum matches the tree-level formulas

Tests [1] and [4] are exact statements and use tight tolerances. Tests [2] and [3]
are convergence statements: they check the observed error SCALES correctly, which
is the only thing a discretization can promise.
"""

import numpy as np

import higgs_gauge as hg

rng = np.random.default_rng(1234)


# ---------------------------------------------------------------------------


def numerical_force_check(p, amp=0.05, n_samples=12, h_rel=1e-5):
    """Compare analytic forces to central differences of energy_potential()."""
    st = hg.random_state(p, amp, seed=7)
    st.Phi += p.v / np.sqrt(2.0)  # keep the Higgs near a sensible magnitude

    aPhi, aW, aB = hg.forces(st, p)
    c = p.a**3  # energy_potential carries a^3; forces are densities
    results = {}

    # Higgs: Phi_ddot = -dU/dPhi^*, with dU/dPhi^* = (1/2)(dU/dRe + i dU/dIm)
    err = []
    for _ in range(n_samples):
        i = rng.integers(0, 2)
        x = tuple(rng.integers(0, p.N, size=3))
        h = h_rel * p.v
        grad = {}
        for part in ("re", "im"):
            plus, minus = st.copy(), st.copy()
            d = h if part == "re" else 1j * h
            plus.Phi[(i,) + x] += d
            minus.Phi[(i,) + x] -= d
            grad[part] = (hg.energy_potential(plus, p) - hg.energy_potential(minus, p)) / (2 * h)
        num = -0.5 * (grad["re"] + 1j * grad["im"]) / c
        err.append(abs(num - aPhi[(i,) + x]) / max(abs(aPhi[(i,) + x]), 1e-30))
    results["Phi"] = max(err)

    for name, arr, ana in (("W", "W", aW), ("B", "B", aB)):
        err = []
        for _ in range(n_samples):
            idx = (
                (rng.integers(0, 3), rng.integers(0, 3)) if name == "W" else (rng.integers(0, 3),)
            )
            x = tuple(rng.integers(0, p.N, size=3))
            key = idx + x
            h = h_rel / p.a
            plus, minus = st.copy(), st.copy()
            getattr(plus, arr)[key] += h
            getattr(minus, arr)[key] -= h
            num = -(hg.energy_potential(plus, p) - hg.energy_potential(minus, p)) / (2 * h) / c
            err.append(abs(num - ana[key]) / max(abs(ana[key]), 1e-30))
        results[name] = max(err)

    return results


def energy_error_scaling(p, T=1.5, amp=0.02):
    """Energy excursion over a fixed physical time, at successively halved dt."""
    out = []
    for safety in (0.30, 0.15, 0.075, 0.0375):
        st = hg.random_state(p, amp, seed=11)
        st.Phi += p.v / np.sqrt(2.0)
        dt = hg.cfl_dt(p, safety)
        e0 = hg.total_energy(st, p)
        n = int(T / dt)
        es, acc = [e0], None
        for i in range(n):
            acc = hg.step(st, p, dt, acc)
            if i % max(1, n // 200) == 0:
                es.append(hg.total_energy(st, p))
        es = np.array(es)
        q = len(es) // 4
        out.append(
            (
                dt,
                (es.max() - es.min()) / abs(e0),
                abs(es[-q:].mean() - es[:q].mean()) / abs(e0),
            )
        )
    return out


def gauge_invariance_scaling():
    """Exact finite U(1)_Y gauge transformation on a fixed physical configuration."""
    out = []
    for N, a in ((8, 0.0040), (16, 0.0020), (32, 0.0010), (64, 0.0005)):
        p = hg.Params(N=N, a=a)
        k = 2 * np.pi / (N * a)
        idx = np.arange(N) * a
        X, Y, Z = np.meshgrid(idx, idx, idx, indexing="ij")
        st = hg.vacuum_state(p)
        st.Phi[0] = 0.2 * p.v * np.sin(k * X) * (1 + 0.3j)
        st.Phi[1] = p.v / np.sqrt(2.0) * (1 + 0.2 * np.cos(k * Y))
        for j in range(3):
            st.B[j] = 0.2 * p.v * np.cos(k * Z + 0.3 * j)
        e0 = hg.energy_potential(st, p)

        beta = 0.7 * np.sin(k * X + 0.2) + 0.5 * np.cos(k * Y)  # O(1), a-independent
        s = st.copy()
        s.Phi = np.exp(1j * hg.Y_PHI * beta) * st.Phi
        for j in range(3):
            s.B[j] = st.B[j] + (np.roll(beta, -1, axis=j) - beta) / a / p.gp
        out.append((a, abs(hg.energy_potential(s, p) - e0) / abs(e0)))
    return out


def photon_is_flat(p):
    """Q annihilates <Phi>, so a uniform photon-direction field must cost zero energy."""
    e_vac = hg.energy_potential(hg.vacuum_state(p), p)
    amp = 0.05 * p.v

    st = hg.vacuum_state(p)
    cW3, cB = hg.photon_direction(p)
    st.W[2, 0] += cW3 * amp
    st.B[0] += cB * amp
    de_gamma = (hg.energy_potential(st, p) - e_vac) / abs(e_vac)
    _, aW, aB = hg.forces(st, p)
    force = max(np.abs(aW).max(), np.abs(aB).max()) / (amp / p.a**2)

    stz = hg.vacuum_state(p)
    zW3, zB = hg.z_direction(p)
    stz.W[2, 0] += zW3 * amp
    stz.B[0] += zB * amp
    de_Z = (hg.energy_potential(stz, p) - e_vac) / abs(e_vac)
    return de_gamma, force, de_Z


def spectrum(p, eps_rel=1e-3):
    eps = eps_rel * p.v
    out = {}
    for name, (setup, probe), pred in (
        ("W", hg.perturb_W(p, eps), p.m_W),
        ("Z", hg.perturb_Z(p, eps), p.m_Z),
        ("gamma", hg.perturb_photon(p, eps), 0.0),
        ("h", hg.perturb_higgs(p, eps), p.m_h),
    ):
        out[name] = (hg.mode_frequency(p, setup, probe), pred)
    return out


# ---------------------------------------------------------------------------

if __name__ == "__main__":
    ok = True
    p = hg.Params(N=4, a=0.0025)
    print("=" * 76)
    print(f"VERIFICATION   (lattice N={p.N}, a={p.a} GeV^-1)")
    print("=" * 76)

    print("\n[1] analytic force vs numerical gradient of the energy")
    for k, e in numerical_force_check(p).items():
        good = e < 1e-6
        ok &= good
        print(f"      {'OK  ' if good else 'FAIL'}  max relative error, {k:>3} force : {e:.3e}")

    print("\n[2] Verlet energy error: bounded, and O(dt^2)")
    rows = energy_error_scaling(p)
    prev = None
    for dt, span, sec in rows:
        r = f"{prev / span:5.2f}" if prev else "   --"
        print(f"      dt={dt:.3e}   excursion={span:.3e}  ratio={r}   secular={sec:.3e}")
        prev = span
    ratios = [rows[i][1] / rows[i + 1][1] for i in range(len(rows) - 1)]
    good = all(3.4 < r < 4.6 for r in ratios) and rows[-1][2] < 1e-4
    ok &= good
    print(f"      {'OK  ' if good else 'FAIL'}  halving dt divides the error by ~4")

    print("\n[3] lattice gauge invariance is O(a^2)  (exact finite U(1) transformation)")
    rows = gauge_invariance_scaling()
    prev = None
    for a, r in rows:
        rr = f"{prev / r:5.2f}" if prev else "   --"
        print(f"      a={a:.5f}   |dE|/E={r:.4e}   ratio={rr}")
        prev = r
    ratios = [rows[i][1] / rows[i + 1][1] for i in range(len(rows) - 1)]
    good = 3.5 < ratios[-1] < 4.6
    ok &= good
    print(f"      {'OK  ' if good else 'FAIL'}  halving a divides the violation by ~4")

    print("\n[4] photon direction is EXACTLY flat at the VEV")
    de_g, force, de_Z = photon_is_flat(p)
    for label, val, good in (
        ("dE along photon direction", de_g, abs(de_g) < 1e-13),
        ("force along photon direction", force, force < 1e-13),
        ("dE along Z direction (must be > 0)", de_Z, de_Z > 1e-6),
    ):
        ok &= good
        print(f"      {'OK  ' if good else 'FAIL'}  {label:<36} = {val:.3e}")

    print("\n[5] k=0 spectrum vs tree-level formulas")
    for name, (meas, pred) in spectrum(p).items():
        if pred == 0.0:
            good = meas == 0.0
            print(f"      {'OK  ' if good else 'FAIL'}  m_{name:<6} measured {meas:9.4f}   "
                  f"predicted 0 (no oscillation)")
        else:
            good = abs(meas / pred - 1) < 1e-4
            print(f"      {'OK  ' if good else 'FAIL'}  m_{name:<6} measured {meas:9.4f}   "
                  f"predicted {pred:9.4f}   ratio {meas / pred:.6f}")
        ok &= good

    print("\n" + "=" * 76)
    print("ALL CHECKS PASSED" if ok else "SOME CHECKS FAILED")
    print("=" * 76)
    raise SystemExit(0 if ok else 1)
