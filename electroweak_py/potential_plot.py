"""
The Higgs potential in the STANDARD (canonical) normalization:

    V(phi) = -(1/2) mu^2 phi^2 + (1/4) lambda phi^4

This is the real-field form of  V = mu_c^2 Phi^dag Phi + lambda (Phi^dag Phi)^2
with mu_c^2 = -mu^2 < 0, using Phi^dag Phi = phi^2 / 2. It is the form in which
the textbook parameter values work directly:

    phi_min  = mu / sqrt(lambda) = v              = 246.22 GeV
    m_h      = sqrt(2) mu = sqrt(2 lambda) v      = 125.11 GeV
    lambda   = m_h^2 / (2 v^2)                    = 0.1291
    mu       = m_h / sqrt(2)                      = 88.47 GeV
    V_min    = -mu^4 / (4 lambda) = -lambda v^4/4
    V = 0 at sqrt(2) * v                          = 348.21 GeV

One real component gives a DOUBLE WELL (a 1D slice). The Mexican hat needs at
least two components, where V depends only on the radius r^2 = phi_1^2 + phi_2^2.

Run:  python potential_plot.py   ->  higgs_potential.png
"""

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np

V_GEV = 246.22
MH_GEV = 125.11

LAM = MH_GEV**2 / (2.0 * V_GEV**2)  # 0.1291 -- the standard quartic
MU = MH_GEV / np.sqrt(2.0)          # 88.47 GeV

PHI_MIN = MU / np.sqrt(LAM)         # = v
V_MIN = -(MU**4) / (4.0 * LAM)
PHI_ZERO = np.sqrt(2.0) * PHI_MIN   # outer zero crossing

C = {"broken": "#c0392b", "sym": "#2471a3", "vac": "#1e8449"}

# Sized to stay legible when the figure is scaled well down.
plt.rcParams.update({
    "font.size": 17,
    "axes.titlesize": 21,
    "axes.labelsize": 19,
    "xtick.labelsize": 16,
    "ytick.labelsize": 16,
    "legend.fontsize": 17,
})


def V(phi_sq, sign=-1.0):
    """sign=-1 -> broken (mu^2 term negative); sign=+1 -> symmetric."""
    return 0.5 * sign * MU**2 * phi_sq + 0.25 * LAM * phi_sq**2


def main():
    fig = plt.figure(figsize=(16.0, 7.0))
    # Explicit margins rather than tight_layout: the 3D axes reserves a lot of
    # slack that tight_layout will not reclaim, leaving a wide gutter.
    gs = fig.add_gridspec(1, 2, width_ratios=[1.0, 1.06],
                          left=0.070, right=0.970, top=0.805, bottom=0.115,
                          wspace=0.015)
    scale = 1e-6  # display V in units of 10^6 GeV^4

    # ---- (a) one real component: a double well -----------------------------
    ax = ax_a = fig.add_subplot(gs[0])
    phi = np.linspace(-1.55 * PHI_MIN, 1.55 * PHI_MIN, 800)
    ax.plot(phi, V(phi**2, -1.0) * scale, color=C["broken"], lw=3.4,
            label=r"broken  ($-\mu^2$)")
    ax.plot(phi, V(phi**2, +1.0) * scale, color=C["sym"], lw=2.8, ls="--",
            label=r"symmetric  ($+\mu^2$)")
    ax.axhline(0, color="0.75", lw=1.1)
    ax.axvline(0, color="0.9", lw=1.1)

    for s in (-1, 1):
        ax.plot(s * PHI_MIN, V_MIN * scale, "o", color=C["vac"], ms=13, zorder=5)
        ax.plot(s * PHI_ZERO, 0, "s", color="0.45", ms=9, zorder=5, mfc="none")
    box = dict(fc="w", ec="none", alpha=0.85, pad=2.0)
    ax.annotate(r"$\phi_{\min}=\mu/\sqrt{\lambda}=v$",
                xy=(PHI_MIN, V_MIN * scale),
                xytext=(-PHI_MIN * 0.62, V_MIN * scale * 1.92), ha="left",
                fontsize=18, color=C["vac"], bbox=box,
                arrowprops=dict(arrowstyle="->", color=C["vac"], lw=1.8))
    ax.annotate(r"$V=0$ at $\sqrt{2}\,v$", xy=(PHI_ZERO, 0),
                xytext=(PHI_ZERO * 0.10, 330.0),
                fontsize=17, color="0.35", bbox=box,
                arrowprops=dict(arrowstyle="->", color="0.5", lw=1.7))
    ax.set_ylim(V_MIN * scale * 2.25, 1320.0)
    ax.set_xlabel(r"$\phi$  [GeV]", labelpad=2)
    ax.set_ylabel(r"$V$  [$10^6\,$GeV$^4$]", labelpad=2)
    ax.set_title("(a)  one real component: a double well", pad=10)
    ax.legend(loc="upper center", framealpha=0.95)
    ax.grid(alpha=0.18)

    # ---- (b) two components: the actual hat --------------------------------
    ax = ax_b = fig.add_subplot(gs[1], projection="3d")

    # Surface on a POLAR mesh: V is a surface of revolution, so this gives a clean
    # circular rim. A square domain instead puts huge r^4 spikes at the corners
    # and visually swamps the central bump that makes it a hat.
    lim3d = PHI_ZERO  # cut the surface exactly at V=0, level with the central bump
    R, TH = np.meshgrid(np.linspace(0, lim3d, 70), np.linspace(0, 2 * np.pi, 180))
    Xp, Yp = R * np.cos(TH), R * np.sin(TH)
    ax.plot_surface(Xp, Yp, V(R**2, -1.0) * scale, cmap="RdYlBu_r",
                    rstride=1, cstride=1, linewidth=0, antialiased=True, alpha=0.95)

    th = np.linspace(0, 2 * np.pi, 400)
    ax.plot(PHI_MIN * np.cos(th), PHI_MIN * np.sin(th),
            np.full_like(th, V_MIN * scale), color=C["vac"], lw=3.6, zorder=10)
    g3 = np.linspace(-lim3d, lim3d, 400)
    ax.plot(g3, np.zeros_like(g3), V(g3**2, -1.0) * scale,
            color=C["broken"], lw=3.4, zorder=11)
    ax.plot([0], [0], [0], "o", color="0.15", ms=9, zorder=12)

    ax.set_xlabel(r"$\phi_1$", labelpad=2)
    ax.set_ylabel(r"$\phi_2$", labelpad=2)
    ax.set_zlabel(r"$V$", labelpad=-4)
    ax.tick_params(labelsize=13, pad=-1)
    ax.view_init(elev=20, azim=-58)
    # zoom enlarges the drawn content inside the axes, which is what makes the
    # flat sombrero read at the same height as panel (a)
    ax.set_box_aspect((1.0, 1.0, 0.46), zoom=1.22)
    # A 3D axes renders its cube centred inside the cell with generous internal
    # margins, so the drawn floor floats well above the bottom of the cell. Drop
    # the whole axes to line the floor up with panel (a)'s baseline, then push the
    # title back up by the same distance so the two panel headings stay level.
    DROP = 0.085
    p = ax.get_position()
    ax.set_position([p.x0, p.y0 - DROP, p.width, p.height])
    ax.set_title("(b)  two components: the Mexican hat",
                 pad=10 + DROP * fig.get_figheight() * 72)

    for i, (txt, col) in enumerate([
        ("red = the slice from (a)", C["broken"]),
        ("green = vacuum manifold (flat)", C["vac"]),
        ("black dot = unstable origin", "0.15"),
        (r"cut at $V=0$", "0.45"),
    ]):
        ax.text2D(0.005, 0.97 - 0.062 * i, txt, transform=ax.transAxes,
                  fontsize=16, color=col)

    fig.suptitle(
        r"$V=-\frac{1}{2}\mu^2\phi^2+\frac{1}{4}\lambda\phi^4$    "
        rf"$\mu={MU:.2f}$ GeV,  $\lambda={LAM:.4f}$    "
        rf"$\Rightarrow$    $\phi_{{\min}}=v={PHI_MIN:.2f}$ GeV,   "
        rf"$m_h=\sqrt{{2}}\,\mu={np.sqrt(2) * MU:.2f}$ GeV",
        fontsize=20, y=0.972,
    )
    fig.savefig("higgs_potential.png", dpi=150)

    print("wrote higgs_potential.png")
    print(f"  mu      = {MU:.4f} GeV      (= m_h / sqrt(2))")
    print(f"  lambda  = {LAM:.6f}        (= m_h^2 / (2 v^2))")
    print(f"  phi_min = {PHI_MIN:.4f} GeV     target v   = {V_GEV}")
    print(f"  m_h     = {np.sqrt(2) * MU:.4f} GeV     target m_h = {MH_GEV}")
    print(f"  V_min   = {V_MIN:.4e} GeV^4 (= -lambda v^4 / 4)")
    print(f"  V=0 at  = {PHI_ZERO:.4f} GeV     (= sqrt(2) * v)")


if __name__ == "__main__":
    main()
