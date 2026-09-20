"""
The Higgs mechanism, measured rather than assumed.

Run:  python demo.py            (prints the spectrum, writes higgs_mechanism.png)

Nothing below inserts a gauge-boson mass by hand. The only inputs are v, lambda, g
and g'. Every mass is read off from how fast a mode actually oscillates.
Run:  python demo.py --replot    to redraw the figure from cached results.
"""

import pickle
import sys

import numpy as np

import higgs_gauge as hg

CACHE = "demo_results.pkl"

# ---------------------------------------------------------------------------
# k = 0 spectroscopy.
#
# For spatially uniform fields every lattice difference vanishes identically, so the
# discretization is EXACT: a 2^3 lattice gives the same answer as 64^3, for any a.
# The timestep is then set by the oscillation period, not by a CFL condition.
# ---------------------------------------------------------------------------


def k0_frequency(p, setup, probe, m_ref, n_periods=25, steps_per_period=400):
    dt = (2 * np.pi / max(m_ref, 1.0)) / steps_per_period
    st = hg.vacuum_state(p)
    setup(st)
    t, y = hg.evolve_probe(st, p, dt, int(n_periods * steps_per_period), probe)
    return hg.frequency_from_zero_crossings(t, y), (t, y)


def point_params(base, **kw):
    """A minimal lattice -- valid for k=0 work only."""
    d = dict(N=2, a=1.0, v=base.v, lam=base.lam, g=base.g, gp=base.gp)
    d.update(kw)
    return hg.Params(**d)


def spectrum_table(base):
    p = point_params(base)
    eps = 1e-3 * p.v
    rows, traces = [], {}
    for label, (setup, probe), pred in (
        ("W+- (charged)", hg.perturb_W(p, eps), p.m_W),
        ("Z (neutral)", hg.perturb_Z(p, eps), p.m_Z),
        ("photon", hg.perturb_photon(p, eps), 0.0),
        ("h (Higgs)", hg.perturb_higgs(p, eps), p.m_h),
    ):
        w, tr = k0_frequency(p, setup, probe, m_ref=max(pred, p.m_W))
        rows.append((label, w, pred))
        traces[label] = (tr[0], tr[1] / eps)
    return rows, traces


def vev_sweep(base, n=9):
    vs = np.linspace(0.05, 1.0, n) * base.v
    out = {"v": vs, "W": [], "Z": [], "h": []}
    for v in vs:
        p = point_params(base, v=v)
        eps = 1e-3 * base.v
        for key, (setup, probe), pred in (
            ("W", hg.perturb_W(p, eps), p.m_W),
            ("Z", hg.perturb_Z(p, eps), p.m_Z),
            ("h", hg.perturb_higgs(p, eps), p.m_h),
        ):
            w, _ = k0_frequency(p, setup, probe, m_ref=pred, n_periods=15)
            out[key].append(w)
    for k in ("W", "Z", "h"):
        out[k] = np.array(out[k])
    return out


def w_dispersion(base, N=16, a=0.008, n_modes=5):
    """omega^2 = khat^2 + m_W^2 for a transverse W wave (polarization x, travel z)."""
    p = hg.Params(N=N, a=a, v=base.v, lam=base.lam, g=base.g, gp=base.gp)
    idx = np.arange(N) * a
    _, _, Z = np.meshgrid(idx, idx, idx, indexing="ij")
    eps = 1e-3 * p.v
    ks, omegas = [], []
    for nmode in range(n_modes):
        k = 2 * np.pi * nmode / (N * a)
        shape = np.sin(k * Z) if nmode > 0 else np.ones_like(Z)
        w_pred = np.sqrt(hg.lattice_momentum_sq(k, a) + p.m_W**2)
        dt = min((2 * np.pi / w_pred) / 300, 0.15 * a)
        st = hg.vacuum_state(p)
        st.W[0, 0] += eps * shape
        norm = (shape**2).sum()
        t, y = hg.evolve_probe(
            st, p, dt, int(12 * 2 * np.pi / w_pred / dt),
            lambda s: (s.W[0, 0] * shape).sum() / norm,
        )
        ks.append(k)
        omegas.append(hg.frequency_from_zero_crossings(t, y))
    return p, np.array(ks), np.array(omegas)


def mass_acquisition(base, N=4, a=0.004, t_end=0.75, seed=2):
    """Start the Higgs on the hilltop, give the W a kick, and watch the W go from
    free drift (no restoring force) to ringing at m_W as the condensate grows.

    Hubble-like friction is applied to the Higgs only, so the field settles into the
    minimum instead of oscillating forever and the W frequency stays readable.
    """
    p = hg.Params(N=N, a=a, v=base.v, lam=base.lam, g=base.g, gp=base.gp)
    rng = np.random.default_rng(seed)
    st = hg.vacuum_state(p)
    sh = st.Phi.shape
    st.Phi[:] = 0.01 * p.v * (rng.standard_normal(sh) + 1j * rng.standard_normal(sh))
    st.Wd[0, 0] = 0.015 * p.v * p.m_W  # W starts at zero amplitude, moving

    def probe(s):
        r = float(np.sqrt(2.0 * np.real(np.sum(s.Phi.conj() * s.Phi)) / s.Phi[0].size))
        return (float(s.W[0, 0, 0, 0, 0]), r)

    dt = hg.cfl_dt(p, 0.05)
    n = int(t_end / dt)
    t, y = hg.evolve_probe(st, p, dt, n, probe, stride=max(1, n // 2000),
                           damping=0.35 * p.m_h)
    W = np.array([q[0] for q in y])
    r = np.array([q[1] for q in y])

    # instantaneous frequency from consecutive zero crossings
    sb = np.signbit(W)
    i = np.nonzero(sb[:-1] != sb[1:])[0]
    tc = t[i] + (t[i + 1] - t[i]) * (-W[i]) / (W[i + 1] - W[i])
    om = np.pi / np.diff(tc)
    tmid = 0.5 * (tc[1:] + tc[:-1])
    r_at = np.interp(tmid, t, r)
    return p, t, W, r, tmid, om, r_at


# ---------------------------------------------------------------------------


def main(replot=False):
    base = hg.Params()
    if replot:
        with open(CACHE, "rb") as fh:
            d = pickle.load(fh)
        make_figure(base, d["traces"], d["rows"], d["pd"], d["ks"], d["ws"],
                    d["pr"], d["t"], d["W"], d["r"], d["tmid"], d["om"], d["r_at"],
                    d["sw"])
        print("redrew higgs_mechanism.png from " + CACHE)
        return
    print("=" * 78)
    print("THE HIGGS MECHANISM ON A LATTICE")
    print("=" * 78)
    print(base.summary())

    print("\n--- k=0 spectrum, measured from oscillation frequencies ---------------")
    rows, traces = spectrum_table(base)
    print(f"{'mode':<16}{'measured [GeV]':>16}{'tree-level [GeV]':>18}{'ratio':>10}")
    for label, meas, pred in rows:
        ratio = "     --" if pred == 0 else f"{meas / pred:.6f}"
        preds = "0 (flat)" if pred == 0 else f"{pred:.4f}"
        print(f"{label:<16}{meas:>16.4f}{preds:>18}{ratio:>10}")

    print("\n--- would-be Goldstone directions -------------------------------------")
    pg = point_params(base, g=0.0, gp=0.0)  # gauge couplings switched OFF
    eps = 1e-3 * pg.v
    for which, name in ((0, "Re upper"), (1, "Im upper"), (2, "Im lower")):
        w, _ = k0_frequency(pg, *hg.perturb_goldstone(pg, eps, which), m_ref=pg.m_h)
        print(f"  ungauged (g=g'=0), {name:<9}: omega = {w:8.4f} GeV  -> massless Goldstone")
    w, _ = k0_frequency(pg, *hg.perturb_higgs(pg, eps), m_ref=pg.m_h)
    print(f"  ungauged, radial     : omega = {w:8.4f} GeV  -> m_h = {pg.m_h:.4f}")
    print("  Switch g, g' back on and those three directions are pure gauge -- still flat,")
    print("  but now unphysical. They reappear as the longitudinal W+, W-, Z polarizations.")

    print("\n--- degree-of-freedom accounting --------------------------------------")
    print("  unbroken :  4 scalar  +  4 massless vectors x 2 polarizations  = 12")
    print("  broken   :  1 Higgs   +  3 massive vectors x 3  +  photon x 2  = 12")

    print("\n--- W dispersion relation ---------------------------------------------")
    pd, ks, ws = w_dispersion(base)
    print(f"  lattice N={pd.N}, a={pd.a} GeV^-1, box L={pd.N * pd.a:.4f} GeV^-1")
    print(f"{'k [GeV]':>10}{'omega meas':>13}{'sqrt(khat^2+mW^2)':>20}{'ratio':>10}")
    for k, w in zip(ks, ws):
        pred = np.sqrt(hg.lattice_momentum_sq(k, pd.a) + pd.m_W**2)
        print(f"{k:>10.2f}{w:>13.3f}{pred:>20.3f}{w / pred:>10.6f}")

    print("\n--- watching the W acquire its mass in real time -----------------------")
    pr, t, W, r, tmid, om, r_at = mass_acquisition(base)
    print(f"  Higgs starts at <r> = {r[0]:.2f} GeV and settles at {r[-1]:.2f} GeV "
          f"(v = {pr.v:.2f})")
    print(f"  W begins with zero restoring force, then rings; late-time omega "
          f"= {om[-6:].mean():.2f} GeV vs m_W = {pr.m_W:.2f}")
    print(f"{'t [1e-3/GeV]':>14}{'omega_W':>10}{'g<r>/2':>10}")
    for i in range(0, len(om), max(1, len(om) // 8)):
        print(f"{tmid[i] * 1e3:>14.1f}{om[i]:>10.2f}{base.g * r_at[i] / 2:>10.2f}")

    sw = vev_sweep(base)
    with open(CACHE, "wb") as fh:
        pickle.dump(dict(traces=traces, rows=rows, pd=pd, ks=ks, ws=ws, pr=pr, t=t,
                         W=W, r=r, tmid=tmid, om=om, r_at=r_at, sw=sw), fh)
    make_figure(base, traces, rows, pd, ks, ws, pr, t, W, r, tmid, om, r_at, sw)
    print("\nwrote higgs_mechanism.png  (results cached in " + CACHE + ")")


def make_figure(base, traces, rows, pd, ks, ws, pr, t, W, r, tmid, om, r_at, sw):
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt

    fig, axes = plt.subplots(2, 3, figsize=(17.5, 9.4))
    C = {"W": "#c0392b", "Z": "#2471a3", "g": "#d4a017", "h": "#1e8449"}

    # (a) three identical kicks -------------------------------------------------
    ax = axes[0, 0]
    for label, col in (("W+- (charged)", C["W"]), ("Z (neutral)", C["Z"]), ("photon", C["g"])):
        tt, yy = traces[label]
        m = tt <= 3 * 2 * np.pi / base.m_W
        ax.plot(tt[m] * 1e3, yy[m], color=col, lw=1.9, label=label)
    ax.axhline(0, color="0.75", lw=0.8)
    ax.set_ylim(-1.30, 1.34)
    ax.set_xlabel(r"time  [$10^{-3}\,$GeV$^{-1}$]")
    ax.set_ylabel("gauge field / initial amplitude")
    ax.set_title("(a)  same kick, three directions in field space", fontsize=11)
    ax.legend(loc="lower center", fontsize=8.5, framealpha=0.95, ncol=3,
              columnspacing=0.9, handlelength=1.4)
    ax.text(0.5, 0.955, "photon never moves:  $Q\\,\\langle\\Phi\\rangle = 0$",
            transform=ax.transAxes, fontsize=9.5, color="#a07a00", ha="center")

    # (b) mass vs VEV -----------------------------------------------------------
    ax = axes[0, 1]
    vv = np.linspace(0, base.v, 100)
    for key, col, lab, pred in (
        ("h", C["h"], r"$m_h=\sqrt{2\lambda}\,v$", np.sqrt(2 * base.lam) * vv),
        ("Z", C["Z"], r"$m_Z=v\sqrt{g^2+g'^2}/2$", vv * np.hypot(base.g, base.gp) / 2),
        ("W", C["W"], r"$m_W=gv/2$", base.g * vv / 2),
    ):
        ax.plot(vv, pred, color=col, lw=1.2, alpha=0.5)
        ax.plot(sw["v"], sw[key], "o", color=col, ms=6, label=lab)
    ax.plot(vv, 0 * vv, color=C["g"], lw=1.2, alpha=0.5)
    ax.plot(base.v, 0, "o", color=C["g"], ms=6, label=r"$m_\gamma=0$")
    ax.set_xlabel("Higgs VEV  $v$  [GeV]")
    ax.set_ylabel("measured mass  [GeV]")
    ax.set_title("(b)  every mass is proportional to the VEV", fontsize=11)
    ax.legend(loc="upper left", fontsize=9, framealpha=0.95)

    # (c) dispersion ------------------------------------------------------------
    ax = axes[0, 2]
    khat2 = hg.lattice_momentum_sq(ks, pd.a)
    xs = np.linspace(0, khat2.max() * 1.1, 50)
    ax.plot(xs, xs + pd.m_W**2, color="0.4", lw=1.3, label=r"$\omega^2=\hat k^2+m_W^2$")
    ax.plot(khat2, ws**2, "o", color=C["W"], ms=7.5, label=r"measured, vs $\hat k^2$")
    ax.plot(ks**2, ws**2, "s", mfc="none", color="0.55", ms=7.5,
            label=r"same data, vs continuum $k^2$")
    ax.axhline(pd.m_W**2, color=C["W"], ls=":", lw=1.0)
    ax.text(khat2.max() * 0.40, pd.m_W**2 * 1.15, r"intercept $= m_W^2$",
            fontsize=9.5, color=C["W"])
    ax.set_xlabel(r"$\hat k^2$  [GeV$^2$]")
    ax.set_ylabel(r"$\omega^2$  [GeV$^2$]")
    ax.set_title("(c)  a genuine massive vector wave", fontsize=11)
    ax.legend(loc="upper left", fontsize=8.5, framealpha=0.95)

    # (d) real-time mass acquisition -------------------------------------------
    ax = axes[1, 0]
    ax.plot(t * 1e3, W, color=C["W"], lw=1.1)
    ax.axhline(0, color="0.75", lw=0.8)
    ax.set_xlabel(r"time  [$10^{-3}\,$GeV$^{-1}$]")
    ax.set_ylabel(r"$W^1_1$  [GeV]", color=C["W"])
    ax.tick_params(axis="y", colors=C["W"])
    ax2 = ax.twinx()
    ax2.plot(t * 1e3, r / pr.v, color=C["h"], lw=2.0)
    ax2.axhline(1.0, color=C["h"], ls="--", lw=0.9, alpha=0.6)
    ax2.set_ylabel(r"$\langle r\rangle/v$", color=C["h"])
    ax2.tick_params(axis="y", colors=C["h"])
    ax2.set_ylim(0, 1.45)
    ax.set_ylim(W.min() * 1.55, W.max() * 1.22)
    ax.axvspan(0, 70, color="0.85", alpha=0.45, lw=0)
    ax.text(0.015, 0.055, "no restoring force:\nW drifts freely", transform=ax.transAxes,
            fontsize=8.8, color="0.3", va="bottom")
    ax.text(0.97, 0.055, r"condensate formed: W rings at $m_W$", transform=ax.transAxes,
            fontsize=8.8, color="0.3", ha="right", va="bottom")
    ax.set_title("(d)  the W acquiring mass in real time", fontsize=11)

    # (e) instantaneous frequency tracks g<r>/2 ---------------------------------
    ax = axes[1, 1]
    ax.plot(t * 1e3, base.g * r / 2, color=C["h"], lw=1.8, label=r"$g\langle r\rangle/2$")
    ax.plot(tmid * 1e3, om, "o", color=C["W"], ms=6,
            label=r"measured $\omega_W$ (per half-cycle)")
    ax.axhline(base.m_W, color="0.4", ls=":", lw=1.2)
    ax.text(t[-1] * 1e3 * 0.60, base.m_W * 1.07, r"$m_W = gv/2$", fontsize=9.5, color="0.3")
    ax.set_xlabel(r"time  [$10^{-3}\,$GeV$^{-1}$]")
    ax.set_ylabel("frequency / mass  [GeV]")
    ax.set_ylim(0, max(om.max(), base.g * r.max() / 2) * 1.12)
    ax.set_title(r"(e)  $\omega_W(t)$ follows the condensate, not the clock", fontsize=11)
    ax.legend(loc="lower right", fontsize=9, framealpha=0.95)

    # (f) results table ---------------------------------------------------------
    ax = axes[1, 2]
    ax.axis("off")
    lines = [
        ("mode", "measured", "tree level", ""),
        ("", "", "", ""),
    ]
    for label, meas, pred in rows:
        name = label.split()[0]
        if pred == 0:
            lines.append((name, f"{meas:.4f}", "0", "exactly flat"))
        else:
            lines.append((name, f"{meas:.3f}", f"{pred:.3f}", f"{meas / pred:.6f}"))
    y0 = 0.93
    ax.text(0.5, 1.0, "measured vs. predicted  [GeV]", ha="center", fontsize=11.5,
            transform=ax.transAxes)
    for i, (c0, c1, c2, c3) in enumerate(lines):
        yy = y0 - i * 0.075
        bold = "bold" if i == 0 else "normal"
        ax.text(0.02, yy, c0, fontsize=10.5, family="monospace", weight=bold,
                transform=ax.transAxes)
        ax.text(0.40, yy, c1, fontsize=10.5, family="monospace", weight=bold,
                ha="right", transform=ax.transAxes)
        ax.text(0.70, yy, c2, fontsize=10.5, family="monospace", weight=bold,
                ha="right", transform=ax.transAxes)
        ax.text(0.99, yy, c3, fontsize=9.5, family="monospace", weight=bold,
                ha="right", color="0.35", transform=ax.transAxes)
    ax.plot([0.02, 0.99], [0.40, 0.40], color="0.8", lw=1.0,
            transform=ax.transAxes, clip_on=False)
    ax.text(0.02, 0.345,
            "inputs:   $v$, $\\lambda$, $g$, $g\'$  -- four numbers\n"
            "outputs:  the entire spectrum\n"
            "\n"
            "No gauge-boson mass term appears anywhere in\n"
            "the Lagrangian. The W and Z masses come only\n"
            "from $|D_\\mu\\Phi|^2$ with $\\Phi$ at its VEV, and the photon\n"
            "stays massless because $Q\\,\\langle\\Phi\\rangle = 0$.\n"
            "\n"
            "d.o.f.:  $4 + 4{\\times}2 \\;=\\; 12 \\;=\\; 1 + 3{\\times}3 + 2$",
            fontsize=9.6, va="top", linespacing=1.5,
            transform=ax.transAxes, color="0.2")

    for a_ in axes.ravel()[:5]:
        a_.grid(alpha=0.18)
    fig.suptitle(
        "SU(2)$_L\\times$U(1)$_Y$ + Higgs doublet, classical lattice field theory"
        "  --  every mass measured, none imposed",
        fontsize=13,
    )
    fig.tight_layout(rect=(0, 0, 1, 0.962))
    fig.savefig("higgs_mechanism.png", dpi=135)


if __name__ == "__main__":
    main(replot="--replot" in sys.argv)
