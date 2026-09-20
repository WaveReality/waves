"""
What the W field actually does when you drive it: evanescent below cutoff,
radiating above it.

Beta decay drives the charged current at ~1 MeV while m_W = 80.4 GeV, so the
source sits at omega/m_W ~ 1e-5 -- five orders of magnitude below cutoff. A
massive field obeys omega^2 = k^2 + m^2, so a source slower than m cannot excite
any propagating mode: k would have to be imaginary. The field still responds, but
the response is evanescent -- exponentially localized, and gone the moment the
source stops. That is what "virtual particle" means, written as a wave equation.

Two experiments:

  1. STATIC SOURCE, full 3D SU(2)xU(1)+Higgs (uses higgs_gauge.py).
     Relax to the static response and fit the profile. It is a Yukawa blob,
     W(r) ~ exp(-m_W r)/r, and the fitted decay rate recovers m_W -- which was
     never put in by hand; it comes from the Higgs condensate.

     The source must be a CONSERVED current. This discretization takes the
     Yang-Mills force as a BACKWARD difference of F, so the Gauss constraint is
     d^-_m j_m = 0 and the source is built as a lattice curl using backward
     differences. Get that wrong and the longitudinal mode is driven with no
     restoring force: the relaxation never converges. A conserved static current
     has no monopole moment, so the field is a screened dipole -- the far field
     still decays as exp(-m r)/r, with a subleading 1/r^2 piece that is removed
     by extrapolating the local log-slope to 1/r -> 0.

  2. DRIVEN SOURCE, 1D massive field.
     Sweep the drive frequency through m_W and measure the time-averaged energy
     flux far from the source. The 1D reduction is used because resolving many
     wavelengths in 3D is expensive, and cutoff physics is dimension independent.
     (What does change with dimension is the static Green's function: exp(-m|x|)
     in 1D, exp(-m r)/r in 3D. Panel (a) is 3D and shows the 1/r.)

Run:  python w_source_demo.py            ->  w_source.png
      python w_source_demo.py --replot   ->  redraw from cache
"""

import pickle
import sys

import numpy as np

import higgs_gauge as hg

CACHE = "w_source_results.pkl"


# ---------------------------------------------------------------------------
# 1. static conserved source in the full 3D theory
# ---------------------------------------------------------------------------


def static_yukawa(N=44, a=0.0038, charge=1e-3, sigma_cells=2.0, n_steps=1800):
    p = hg.Params(N=N, a=a)
    idx = (np.arange(N) - N // 2) * a
    X, Y, Z = np.meshgrid(idx, idx, idx, indexing="ij")
    r = np.sqrt(X**2 + Y**2 + Z**2)

    sigma = sigma_cells * a
    Az = np.exp(-0.5 * (r / sigma) ** 2)
    Az *= 1.0 / (Az.sum() * a**3)

    # j = curl(0, 0, Az) with BACKWARD differences -> d^-_m j_m = 0 exactly,
    # which is what this discretization's Gauss constraint requires.
    def bw(f, ax):
        return (f - np.roll(f, 1, axis=ax)) / a

    st = hg.vacuum_state(p)
    jW = np.zeros_like(st.W)
    jW[0, 0] = charge * bw(Az, 1)
    jW[0, 1] = -charge * bw(Az, 0)
    div = bw(jW[0, 0], 0) + bw(jW[0, 1], 1)

    dt = hg.cfl_dt(p, 0.15)
    gamma = 1.0 * p.m_W
    acc = None
    for _ in range(n_steps):
        acc = hg.step(st, p, dt, acc, jW=jW)
        f = np.exp(-gamma * dt)
        st.Wd *= f
        st.Bd *= f
        st.Pi *= f
    residual = np.abs(acc[1]).max()

    Wm = np.sqrt(st.W[0, 0] ** 2 + st.W[0, 1] ** 2)
    nb = 36
    edges = np.linspace(0, N // 2 * a, nb + 1)
    which = np.digitize(r.ravel(), edges) - 1
    rs, ws = [], []
    for b in range(nb):
        sel = which == b
        if sel.sum() > 16:
            rs.append(r.ravel()[sel].mean())
            ws.append(Wm.ravel()[sel].mean())
    rs, ws = np.array(rs), np.array(ws)
    keep = (rs > 4 * sigma) & (rs < 0.45 * N * a) & (ws > 0)
    rs, ws = rs[keep], ws[keep]

    logrW = np.log(rs * ws)
    m_naive = -np.polyfit(rs, logrW, 1)[0]
    # local slope, extrapolated to 1/r -> 0 to drop the dipole's 1/r^2 term
    slope = np.diff(logrW) / np.diff(rs)
    rmid = 0.5 * (rs[1:] + rs[:-1])
    m_ext = -np.polyfit(1.0 / rmid, slope, 1)[1]
    # what the lattice Laplacian actually gives for a screened field
    kappa_lat = (2.0 / a) * np.arcsinh(a * p.m_W / 2.0)

    return dict(p_N=N, p_a=a, m_W=p.m_W, rs=rs, ws=ws, m_naive=m_naive,
                m_ext=m_ext, kappa_lat=kappa_lat, sigma=sigma,
                div=np.abs(div).max(), jmax=np.abs(jW).max(), residual=residual)


# ---------------------------------------------------------------------------
# 2. driven source, 1D:  (d_t^2 - d_x^2 + m^2) W = J(x,t)
# ---------------------------------------------------------------------------


def solve_1d(m, omega, *, Nx=2400, a=0.0025, t_total=3.0, sponge_frac=0.16,
             sigma_cells=2.0, amp=1.0, det_frac=0.042, avg_from=0.60,
             ramp_periods=3.0, record_window=None):
    x = (np.arange(Nx) - Nx // 2) * a
    sigma = sigma_cells * a
    src = np.exp(-0.5 * (x / sigma) ** 2)
    src /= src.sum() * a                      # unit integral, so J_0 = amp

    nsp = int(sponge_frac * Nx)               # absorbing sponge, quadratic ramp
    gamma = np.zeros(Nx)
    ramp = np.linspace(0.0, 1.0, nsp) ** 2
    gmax = 12.0 / (nsp * a)
    gamma[-nsp:] = gmax * ramp
    gamma[:nsp] = gmax * ramp[::-1]

    dt = 0.40 * a
    n_steps = int(t_total / dt)
    T = 2 * np.pi / omega
    ramp_t = min(ramp_periods * T, 0.25 * t_total)
    det = Nx // 2 + int(det_frac * Nx)

    W = np.zeros(Nx)
    v = np.zeros(Nx)
    ft, fs, frames, frame_t = [], [], [], []
    sel = np.abs(x) <= record_window if record_window else None

    for n in range(n_steps):
        t = n * dt
        env = 0.5 * (1.0 - np.cos(np.pi * min(t / ramp_t, 1.0)))
        lap = (np.roll(W, -1) - 2.0 * W + np.roll(W, 1)) / a**2
        acc = lap - m**2 * W + amp * env * np.sin(omega * t) * src
        v = (v + dt * acc) / (1.0 + dt * gamma)
        W = W + dt * v
        ft.append(t)
        fs.append(-v[det] * (W[det + 1] - W[det - 1]) / (2 * a))   # S = -(d_t W)(d_x W)
        if sel is not None:
            frames.append(W[sel].copy())
            frame_t.append(t)

    ft, fs = np.array(ft), np.array(fs)
    late = ft > avg_from * t_total
    out = dict(mean_flux=fs[late].mean(), x_det=x[det], T=T)
    if sel is not None:
        out.update(frames=np.array(frames), frame_t=np.array(frame_t),
                   frame_x=x[sel])
    return out


def theory_flux(m, omega, a=0.0025, sigma=0.005):
    """<S> = omega |J(k)|^2 / (8k), k from the lattice dispersion. Zero below m."""
    omega = np.atleast_1d(omega)
    out = np.zeros_like(omega, dtype=float)
    above = omega > m
    if above.any():
        arg = np.clip(a * np.sqrt(omega[above] ** 2 - m**2) / 2.0, 0, 1)
        k = (2.0 / a) * np.arcsin(arg)
        out[above] = omega[above] * np.exp(-(k**2) * sigma**2) / (8.0 * k)
    return out


def frequency_sweep(m):
    ratios = np.concatenate([np.linspace(0.40, 0.95, 7),
                             np.linspace(1.05, 2.40, 14)])
    flux = np.array([solve_1d(m, r * m)["mean_flux"] for r in ratios])
    return ratios, flux


# ---------------------------------------------------------------------------


def main(replot=False):
    if replot:
        with open(CACHE, "rb") as fh:
            d = pickle.load(fh)
        make_figure(**d)
        print("redrew w_source.png from " + CACHE)
        return

    m = hg.Params().m_W
    print("=" * 76)
    print("DRIVING THE W FIELD:  evanescent below cutoff, radiating above")
    print("=" * 76)
    print(f"\nm_W = {m:.3f} GeV (= g v / 2, from the Higgs condensate);"
          f"  range 1/m_W = {0.1973 / m:.5f} fm")
    print(f"beta decay sits at omega/m_W ~ {1e-3 / m:.1e}")

    print("\n--- 1. static conserved source, full 3D SU(2)xU(1)+Higgs --------------")
    y = static_yukawa()
    print(f"  lattice N={y['p_N']}, a={y['p_a']} GeV^-1, box = "
          f"{y['p_N'] * y['p_a'] * m:.1f}/m_W")
    print(f"  source conservation |d^-.j| = {y['div']:.2e}  (vs |j| = {y['jmax']:.2e})")
    print(f"  relaxation residual max|acc| = {y['residual']:.2e}")
    print(f"  fitted decay rate, naive ln(rW) fit   : {y['m_naive']:7.3f} GeV")
    print(f"  fitted, log-slope extrapolated 1/r->0 : {y['m_ext']:7.3f} GeV")
    print(f"  lattice expectation for this a        : {y['kappa_lat']:7.3f} GeV")
    print(f"  m_W from the Higgs condensate         : {m:7.3f} GeV")
    print(f"  extrapolated / lattice = {y['m_ext'] / y['kappa_lat']:.4f}")

    print("\n--- 2. driven source, 1D ----------------------------------------------")
    below = solve_1d(m, 0.60 * m, t_total=0.32, record_window=0.16,
                     ramp_periods=1.2, avg_from=0.5)
    above = solve_1d(m, 1.60 * m, t_total=0.32, record_window=0.16,
                     ramp_periods=1.2, avg_from=0.5)
    print(f"  space-time windows recorded at 0.60 m_W and 1.60 m_W")

    print("\n  frequency sweep ...")
    ratios, flux = frequency_sweep(m)
    th = theory_flux(m, ratios * m)
    print(f"{'omega/m_W':>11}{'flux (sim)':>14}{'flux (theory)':>16}{'ratio':>9}")
    for r_, f_, t_ in zip(ratios, flux, th):
        rr = f"{f_ / t_:.4f}" if t_ > 0 else "   --"
        print(f"{r_:>11.3f}{f_:>14.4e}{t_:>16.4e}{rr:>9}")
    lo = flux[ratios < 1.0].max()
    hi = flux[ratios > 1.0].max()
    print(f"\n  peak flux below cutoff {lo:.3e}   above cutoff {hi:.3e}"
          f"   -> suppression {hi / lo:.1e}")

    d = dict(m=m, y=y, below=below, above=above, ratios=ratios, flux=flux)
    with open(CACHE, "wb") as fh:
        pickle.dump(d, fh)
    make_figure(**d)
    print("\nwrote w_source.png  (results cached in " + CACHE + ")")


def make_figure(m, y, below, above, ratios, flux):
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt

    plt.rcParams.update({
        "font.size": 13, "axes.titlesize": 15, "axes.labelsize": 14,
        "xtick.labelsize": 12, "ytick.labelsize": 12, "legend.fontsize": 11.5,
    })
    CC = {"sim": "#c0392b", "evan": "#2471a3", "th": "#1e8449"}

    fig, axes = plt.subplots(2, 2, figsize=(14.5, 10.5))

    # (a) 3D static Yukawa ----------------------------------------------------
    ax = axes[0, 0]
    rs, ws = y["rs"], y["ws"]
    ax.semilogy(rs * m, rs * ws, "o", color=CC["sim"], ms=6.5, label="3D simulation")
    A = (rs * ws * np.exp(y["m_ext"] * rs)).mean()
    ax.semilogy(rs * m, A * np.exp(-y["m_ext"] * rs), color="0.35", lw=2.0, ls="--",
                label=rf"$e^{{-mr}}$ fit,  $m={y['m_ext']:.1f}$ GeV")
    ax.set_xlabel(r"$m_W\,r$")
    ax.set_ylabel(r"$r\,|W|$")
    ax.set_title(r"(a)  static source $\rightarrow$ Yukawa blob, $e^{-m_W r}/r$")
    ax.legend(loc="upper right", framealpha=0.95)
    ax.grid(alpha=0.2, which="both")
    ax.text(0.03, 0.06,
            f"$m_W$ input (Higgs): {m:.1f} GeV\n"
            f"fitted decay rate:  {y['m_ext']:.1f} GeV\n"
            f"lattice expectation: {y['kappa_lat']:.1f} GeV",
            transform=ax.transAxes, fontsize=11, color="0.2")

    # (b,c) space-time --------------------------------------------------------
    for ax, dat, lab, ratio in ((axes[0, 1], below, "(b)", 0.60),
                                (axes[1, 0], above, "(c)", 1.60)):
        F, xs, ts = dat["frames"], dat["frame_x"], dat["frame_t"]
        vmax = np.abs(F).max()
        ax.pcolormesh(xs * m, ts * m, F, cmap="RdBu_r", vmin=-vmax, vmax=vmax,
                      shading="auto", rasterized=True)
        tc = np.array([0.0, ts[-1]])
        for s in (-1, 1):
            ax.plot(s * tc * m, tc * m, color="0.1", lw=1.5, ls=":")
        ax.set_xlim(xs[0] * m, xs[-1] * m)
        ax.set_ylim(0, ts[-1] * m)
        ax.set_xlabel(r"$m_W\,x$")
        ax.set_ylabel(r"$m_W\,t$")
        sub = "evanescent: nothing leaves" if ratio < 1 else "radiating: waves escape"
        ax.set_title(rf"{lab}  $\omega = {ratio:.2f}\,m_W$ — {sub}")
        ax.text(0.98, 0.03, "dotted = light cone", transform=ax.transAxes,
                fontsize=10.5, color="0.1", ha="right")

    # (d) flux vs frequency ---------------------------------------------------
    ax = axes[1, 1]
    w = np.linspace(1.001, 2.40, 400)
    ax.semilogy(w, theory_flux(m, w * m), color=CC["th"], lw=2.0,
                label=r"theory  $\omega|\tilde J(k)|^2/8k$")
    lo, hi = ratios < 1.0, ratios > 1.0
    ax.semilogy(ratios[hi], flux[hi], "o", color=CC["sim"], ms=7,
                label="simulation, radiating")
    ax.semilogy(ratios[lo], np.abs(flux[lo]), "s", color=CC["evan"], ms=7,
                mfc="none", label="simulation, evanescent")
    ax.axvline(1.0, color="0.25", lw=1.8, ls="--")
    ax.axvspan(0.35, 1.0, color=CC["evan"], alpha=0.08, lw=0)
    ax.text(0.985, 0.30, r"cutoff $\omega=m_W$", transform=ax.get_xaxis_transform(),
            fontsize=12, color="0.25", va="center", ha="right", rotation=90,
            bbox=dict(fc="w", ec="none", alpha=0.8, pad=1.5))
    ax.text(0.05, 0.80, "no propagating\nmode exists", transform=ax.transAxes,
            fontsize=12, color=CC["evan"], va="top",
            bbox=dict(fc="w", ec="none", alpha=0.8, pad=2.5))
    ax.set_xlim(0.35, 2.45)
    ax.set_ylim(1e-10, 3.0)
    ax.set_xlabel(r"drive frequency  $\omega/m_W$")
    ax.set_ylabel(r"time-averaged energy flux  $\langle S\rangle$")
    ax.set_title(rf"(d)  radiated power switches on at the mass "
             rf"($10^{{{int(round(np.log10(flux[hi].max() / np.abs(flux[lo]).min())))}}}$ jump)")
    ax.legend(loc="lower right", framealpha=0.95)
    ax.grid(alpha=0.2, which="both")

    fig.suptitle(
        r"Driving the W field:  $(\partial_t^2-\nabla^2+m_W^2)\,W = J$,   "
        rf"$m_W = {m:.1f}$ GeV generated by the Higgs condensate"
        "\n"
        r"Beta decay drives at $\omega/m_W \sim 10^{-5}$ — off the left edge of (d),"
        " deep in the evanescent regime",
        fontsize=15,
    )
    fig.tight_layout(rect=(0, 0, 1, 0.925))
    fig.savefig("w_source.png", dpi=145)


if __name__ == "__main__":
    main(replot="--replot" in sys.argv)
