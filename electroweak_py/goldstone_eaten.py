"""
The Goldstone bosons becoming the longitudinal polarizations of W and Z.

Degrees of freedom are conserved across symmetry breaking:

    before:  Phi (4 real) + 4 massless vectors (2 each)   = 12
    after:   h   (1 real) + W+-,Z (3 each) + photon (2)   = 12

So three scalar degrees of freedom have to leave Phi and reappear as the third
(longitudinal) polarization of each massive vector. The term that does it is the
bilinear mixing hiding in |D_mu Phi|^2,

    L_mix = m_Z Z_mu d^mu G0 + i m_W (W-_mu d^mu G+ - W+_mu d^mu G-)

whose coefficient is exactly the mass -- mass generation and "eating" are the same
event, both coming from G_mu <Phi>.

This script measures the transfer directly, in temporal gauge, where the Goldstones
are still explicit fields rather than gauged away. One Fourier mode at momentum k
along z is kicked and its frequency is read off:

    background                 longitudinal W        Goldstone
    -------------------------  --------------------  --------------------
    ungauged (g = g' = 0)      omega = 0 (dead)      omega = khat (massless)
    symmetric <Phi> = 0        omega = 0 (dead)      --
    broken + gauged            sqrt(khat^2 + m_W^2)  sqrt(khat^2 + m_W^2)

In the first two rows the longitudinal vector does not propagate at all (its curl
vanishes, so it feels no force) and the Goldstone is an independent massless scalar.
In the third they have merged: one massive vector polarization, and no mode left at
omega = khat.

Run:  python goldstone_eaten.py     (~20 s, text output only)
"""

import numpy as np

import higgs_gauge as hg

# Lattice: 8 sites and a = 0.02 GeV^-1 put the lowest mode at k ~ 39 GeV, safely
# below m_W. Coarseness costs nothing here -- the massive dispersion is exactly
# sqrt(khat^2 + m^2) on the lattice, since the mass term is a local k=0 term and
# the gradient term gives khat^2 by construction.
N, A = 8, 0.020
EPS = 0.2  # kick amplitude [GeV], small against v = 246

KPH = 2.0 * np.pi / (N * A)  # lowest lattice momentum, directed along z
KHAT = np.sqrt(hg.lattice_momentum_sq(KPH, A))  # what the discrete Laplacian sees
PROF = np.cos(KPH * np.arange(N) * A).reshape(1, 1, N)  # varies along the last axis


# ---------------------------------------------------------------------------
# kicks and probes -- all at momentum k along z, all in the W^1 / charged sector
# ---------------------------------------------------------------------------


def kick_W_long(st):
    """W^1 polarized ALONG k. Curl-free, so pure gauge in the unbroken theory."""
    st.W[0, 2] += EPS * PROF


def kick_W_trans(st):
    """W^1 polarized ACROSS k. The ordinary propagating polarization."""
    st.W[0, 0] += EPS * PROF


def kick_goldstone(st):
    """The charged would-be Goldstone (upper doublet component)."""
    st.Phi[0] += EPS * PROF


# Probe the VELOCITY, not the field. In the broken phase the longitudinal sector
# is a 2x2 system (W_L, G) whose mass matrix [[m^2, mk], [mk, k^2]] has eigenvalues
# 0 and k^2 + m^2 -- the zero eigenvalue being the residual gauge mode. A kick
# excites both, so the field oscillates about a nonzero offset and zero-crossing
# detection fails. The gauge mode is static, so it drops out of the velocity.
def pr_W_long(st):
    return float(np.sum(st.Wd[0, 2] * PROF))


def pr_W_trans(st):
    return float(np.sum(st.Wd[0, 0] * PROF))


def pr_goldstone(st):
    return float(np.sum(np.real(st.Pi[0]) * PROF))


# ---------------------------------------------------------------------------


def frequency(p, kick, probe, ref, n_periods=8.0, safety=0.04):
    """Kick one mode of the vacuum and return its measured angular frequency."""
    st = hg.vacuum_state(p)
    kick(st)
    dt = hg.cfl_dt(p, safety)
    n_steps = max(400, int(n_periods * 2.0 * np.pi / max(ref, 1.0) / dt))
    t, y = hg.evolve_probe(st, p, dt, n_steps, probe)
    return hg.frequency_from_zero_crossings(t, y)


def main():
    gauged = hg.Params(N=N, a=A)
    ungauged = hg.Params(N=N, a=A, g=0.0, gp=0.0)  # broken, but no gauge coupling
    symmetric = hg.Params(N=N, a=A, broken=False)  # <Phi> = 0, free gauge fields

    m_W = gauged.m_W
    w_massive = np.sqrt(KHAT**2 + m_W**2)

    print(f"k_phys = {KPH:.3f}   khat = {KHAT:.4f} GeV   m_W = {m_W:.4f} GeV")
    print(f"massive dispersion  sqrt(khat^2 + m_W^2) = {w_massive:.4f} GeV\n")

    rows = [
        ("ungauged broken (g=g'=0)", "charged Goldstone", ungauged, kick_goldstone,
         pr_goldstone, KHAT, f"khat = {KHAT:.3f}   massless scalar"),
        ("ungauged broken (g=g'=0)", "W transverse", ungauged, kick_W_trans,
         pr_W_trans, KHAT, f"khat = {KHAT:.3f}   massless vector"),
        ("ungauged broken (g=g'=0)", "W longitudinal", ungauged, kick_W_long,
         pr_W_long, KHAT, "0               pure gauge, dead"),
        ("symmetric  <Phi> = 0", "W transverse", symmetric, kick_W_trans,
         pr_W_trans, KHAT, f"khat = {KHAT:.3f}   massless vector"),
        ("symmetric  <Phi> = 0", "W longitudinal", symmetric, kick_W_long,
         pr_W_long, KHAT, "0               pure gauge, dead"),
        ("BROKEN + GAUGED", "W transverse", gauged, kick_W_trans,
         pr_W_trans, w_massive, f"{w_massive:.3f}"),
        ("BROKEN + GAUGED", "W longitudinal", gauged, kick_W_long,
         pr_W_long, w_massive, f"{w_massive:.3f}          <== EATEN"),
        ("BROKEN + GAUGED", "charged Goldstone", gauged, kick_goldstone,
         pr_goldstone, w_massive, f"{w_massive:.3f}          <== EATEN"),
    ]

    print(f"{'background':26s} {'kicked direction':19s} {'measured w':>12s}   expected")
    print("-" * 92)
    last = None
    for bg, name, p, kick, probe, ref, expected in rows:
        if last is not None and bg != last:
            print()
        last = bg
        w = frequency(p, kick, probe, ref)
        print(f"{bg:26s} {name:19s} {w:12.4f}   {expected}")

    # ---- the mixing term caught in the act --------------------------------
    # Start the Higgs EXACTLY at the VEV and disturb only the gauge field. The
    # bilinear m_W W^-_mu d^mu G^+ then sources the Goldstone immediately.
    print("\npure longitudinal-W kick, Higgs started exactly at the VEV:")
    for label, p in (("  gauged  (g = 0.6533)", gauged), ("  ungauged (g = 0)     ", ungauged)):
        st = hg.vacuum_state(p)
        kick_W_long(st)
        dt = hg.cfl_dt(p, 0.04)
        n_steps = int(2.0 * 2.0 * np.pi / w_massive / dt)
        _, y = hg.evolve_probe(st, p, dt, n_steps, lambda s: float(np.max(np.abs(s.Phi[0]))))
        print(f"{label}  ->  max |Phi_upper| = {y.max():.4e} GeV")

    print("\nThe Goldstone is generated out of nothing by the gauge field alone;")
    print("with g = 0 it stays identically zero. That coupling is the eating.")


if __name__ == "__main__":
    main()
