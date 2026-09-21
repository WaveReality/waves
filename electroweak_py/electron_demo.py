"""
A free electron and a free neutrino in motion, and the fields they carry.

Same wave packet for both -- identical mass, momentum and width -- so the only
difference is the charges. Q = T^3 + Y gives the electron Q = -1 and the neutrino
Q = 0, while the Z vector couplings g_V = T^3 - 2 Q sin^2(theta_W) come out
-0.054 for the electron (accidentally tiny) and +0.500 for the neutrino. So the
two particles trade places: the electron carries a long-range electric field and
almost no Z, the neutrino carries no electric field at all and a Z field nine
times the electron's.

The fermion is a genuine Dirac wave packet: a 4-component spinor, projected onto
positive energy, evolved EXACTLY in momentum space (psi(k,t) = e^{-iE_k t} psi(k,0)).
No lattice Dirac operator, so no doublers and no dispersion error. Its conserved
current J^mu = (psi^dag psi, psi^dag alpha^i psi) then sources the gauge fields.

A UNIFORMLY MOVING SOURCE DOES NOT RADIATE, so the fields are rigidly attached to
the packet and the steady-state solution is exact. For a source translating at
velocity v along z, in Lorenz gauge,

    A^mu(k) = S^mu(k) / (k_perp^2 + k_z^2/gamma^2)                  photon
    Z^mu(k) = S_Z^mu(k) / (k_perp^2 + k_z^2/gamma^2 + m_Z^2)        Z

    E = -grad A^0 + v d_z A        B = curl A

which is where the Lorentz pancaking comes from: the k_z^2/gamma^2 in the
denominator is a 1/gamma contraction along the direction of motion.

WHY A HEAVY LEPTON. A real electron's Compton wavelength is 386 fm while the Z
range is 0.0022 fm -- a ratio of 1.8e5. No single lattice holds both. So this uses
a 100 GeV lepton, whose wave packet is comparable to the Z range, with the real
electron's charge and couplings. The electromagnetic structure is scale free and
looks identical for a real electron; only the Z panel depends on the choice.

WHY THERE IS NO W FIELD, for either of them. The charged current is off-diagonal
in particle species (it turns an electron into a neutrino), so a one-lepton state
has <J^+> = 0 identically. Nothing to plot -- see w_source_demo.py for what does
drive it.

Run:  python electron_demo.py            ->  electron_fields.png
      python electron_demo.py --replot   ->  redraw from cache
"""

import pickle
import sys

import numpy as np

import higgs_gauge as hg

CACHE = "electron_results.pkl"

# --- Dirac algebra, Dirac representation -----------------------------------
_s = [np.array([[0, 1], [1, 0]], complex),
      np.array([[0, -1j], [1j, 0]]),
      np.array([[1, 0], [0, -1]], complex)]
_I, _O = np.eye(2, dtype=complex), np.zeros((2, 2), complex)
ALPHA = np.array([np.block([[_O, s], [s, _O]]) for s in _s])
BETA = np.block([[_I, _O], [_O, -_I]])


# ---------------------------------------------------------------------------


class Packet:
    """Positive-energy Dirac wave packet, evolved exactly in momentum space."""

    def __init__(self, N=96, a=0.003, mass=100.0, pz=200.0, sigma=0.012):
        self.N, self.a, self.mass, self.pz = N, a, mass, pz
        self.L = N * a
        k1 = 2 * np.pi * np.fft.fftfreq(N, a)
        self.K = np.meshgrid(k1, k1, k1, indexing="ij")
        self.E = np.sqrt(sum(k**2 for k in self.K) + mass**2)
        self.E_c = np.sqrt(mass**2 + pz**2)
        self.v = pz / self.E_c
        self.gamma = self.E_c / mass

        g = np.exp(-0.5 * sigma**2 * (self.K[0] ** 2 + self.K[1] ** 2
                                      + (self.K[2] - pz) ** 2))
        g = g * np.exp(-1j * sum(self.K) * (self.L / 2))      # centre in the box
        chi = np.zeros((4,) + g.shape, complex)
        chi[0] = g
        self.psi_hat = 0.5 * (chi + self._H(chi) / self.E)     # positive-energy projector
        nrm = np.sqrt((np.abs(self.psi_hat) ** 2).sum() * a**3 / N**3)
        self.psi_hat /= nrm                                    # integral |psi|^2 d3x = 1

    def _H(self, p):
        out = np.einsum("ij,j...->i...", BETA, p) * self.mass
        for A, k in zip(ALPHA, self.K):
            out = out + np.einsum("ij,j...->i...", A, p) * k
        return out

    def psi(self, t):
        return np.fft.ifftn(self.psi_hat * np.exp(-1j * self.E * t), axes=(1, 2, 3))

    def current(self, t):
        """Returns (J0, [Jx, Jy, Jz]) -- the conserved Dirac current."""
        p = self.psi(t)
        J0 = np.real(np.einsum("i...,i...->...", p.conj(), p))
        Jv = [np.real(np.einsum("i...,ij,j...->...", p.conj(), A, p)) for A in ALPHA]
        return J0, Jv

    def mean_velocity(self, t=0.0):
        J0, Jv = self.current(t)
        return np.array([J.sum() for J in Jv]) / J0.sum()


# ---------------------------------------------------------------------------


def moving_fields(pk, J0, Jv, coupling, mass=0.0):
    """Steady-state E and B of a source rigidly translating at v along z.

    Exact for uniform motion (which does not radiate). mass=0 -> photon.
    """
    KX, KY, KZ = pk.K
    D = KX**2 + KY**2 + KZ**2 / pk.gamma**2 + mass**2
    if mass == 0.0:
        D = D.copy()
        D[0, 0, 0] = np.inf        # neutralising background: drop the k=0 mode

    S0 = np.fft.fftn(coupling * J0)
    Sv = [np.fft.fftn(coupling * J) for J in Jv]
    A0 = S0 / D
    Av = [S / D for S in Sv]

    # E = -grad A0 + v d_z A ;  B = curl A
    Eh = [-1j * K * A0 + 1j * pk.v * KZ * A for K, A in zip(pk.K, Av)]
    Bh = [1j * (KY * Av[2] - KZ * Av[1]),
          1j * (KZ * Av[0] - KX * Av[2]),
          1j * (KX * Av[1] - KY * Av[0])]
    E = [np.real(np.fft.ifftn(h)) for h in Eh]
    B = [np.real(np.fft.ifftn(h)) for h in Bh]
    return E, B


def potential_profile(pk, J0, coupling, mass, nb=48):
    """Radial profile of r*|A^0| (Lorenz gauge). Photon -> flat, Yukawa -> e^{-m r},
    with no 1/r prefactor left to bias an exponential fit."""
    KX, KY, KZ = pk.K
    D = KX**2 + KY**2 + KZ**2 / pk.gamma**2 + mass**2
    if mass == 0.0:
        D = D.copy()
        D[0, 0, 0] = np.inf
    A0 = np.real(np.fft.ifftn(np.fft.fftn(coupling * J0) / D))
    idx = np.arange(pk.N) * pk.a - pk.L / 2
    X, Y, Z = np.meshgrid(idx, idx, idx, indexing="ij")
    r = np.sqrt(X**2 + Y**2 + Z**2).ravel()
    v = np.abs(A0).ravel() * r
    edges = np.linspace(0, pk.L / 2, nb + 1)
    w = np.digitize(r, edges) - 1
    rs, fs = [], []
    for b in range(nb):
        sel = w == b
        if sel.sum() > 16:
            rs.append(r[sel].mean())
            fs.append(v[sel].mean())
    return np.array(rs), np.array(fs)


def radial_profile(pk, F, nb=48):
    """|F| binned by distance from the box centre (minimum-image)."""
    idx = np.arange(pk.N) * pk.a - pk.L / 2
    X, Y, Z = np.meshgrid(idx, idx, idx, indexing="ij")
    r = np.sqrt(X**2 + Y**2 + Z**2).ravel()
    mag = np.sqrt(sum(f**2 for f in F)).ravel()
    edges = np.linspace(0, pk.L / 2, nb + 1)
    w = np.digitize(r, edges) - 1
    rs, fs = [], []
    for b in range(nb):
        sel = w == b
        if sel.sum() > 16:
            rs.append(r[sel].mean())
            fs.append(mag[sel].mean())
    return np.array(rs), np.array(fs)


def validate_boost(v, sigma=0.012, boxes=((96, 0.004), (128, 0.006), (160, 0.007))):
    """Pure rigid Gaussian charge: E_perp/E_par -> gamma^3 as the box grows.

    Separates the field solver from the Dirac packet. The asymptotic point-charge
    ratio needs gamma*sigma << r << L/4; a periodic box always squeezes that window.
    """
    gam = 1.0 / np.sqrt(1 - v * v)
    rows = []
    for N, a in boxes:
        L = N * a
        k1 = 2 * np.pi * np.fft.fftfreq(N, a)
        KX, KY, KZ = np.meshgrid(k1, k1, k1, indexing="ij")
        idx = np.arange(N) * a - L / 2
        X, Y, Z = np.meshgrid(idx, idx, idx, indexing="ij")
        rho = np.exp(-0.5 * (X**2 + Y**2 + Z**2) / sigma**2)
        rho /= rho.sum() * a**3
        D = KX**2 + KY**2 + KZ**2 / gam**2
        D[0, 0, 0] = np.inf
        A0 = np.fft.fftn(rho) / D
        Az = np.fft.fftn(rho * v) / D
        E = [np.real(np.fft.ifftn(-1j * K * A0 + 1j * v * KZ * A))
             for K, A in zip((KX, KY, KZ), (0 * Az, 0 * Az, Az))]
        c, d = N // 2, int(round(0.11 / a))
        et = np.sqrt(sum(f[c + d, c, c] ** 2 for f in E))
        el = np.sqrt(sum(f[c, c, c + d] ** 2 for f in E))
        rows.append((N, a, L, d * a, et / el))
    return gam**3, rows


# ---------------------------------------------------------------------------


def main(replot=False):
    if replot:
        with open(CACHE, "rb") as fh:
            make_figure(**pickle.load(fh))
        print("redrew electron_fields.png from " + CACHE)
        return

    P = hg.Params()
    sw2 = np.sin(P.theta_W) ** 2
    e_em = np.sqrt(4 * np.pi / 137.036)
    pref = P.g / (2 * np.cos(P.theta_W))
    gV = -0.5 + 2 * sw2                       # electron: T3=-1/2, Q=-1
    gZ = pref * gV
    gV_nu = +0.5                              # neutrino: T3=+1/2, Q=0
    gZ_nu = pref * gV_nu

    print("=" * 76)
    print("A SINGLE FREE LEPTON IN MOTION, AND ITS FIELDS")
    print("=" * 76)
    print(f"\n  m_Z = {P.m_Z:.2f} GeV, range 1/m_Z = {0.1973 / P.m_Z:.5f} fm")
    print(f"  a real electron's Compton wavelength is {0.1973 / 0.511e-3:.0f} fm"
          f"  ->  ratio {(1 / 0.511e-3) / (1 / P.m_Z):.1e}")
    print("  so this uses a 100 GeV lepton with the electron's charge and couplings.")
    print(f"\n  EM coupling   e*Q = {e_em * -1:+.4f}")
    print(f"  Z  coupling  g_Z*g_V = {gZ:+.4f}   (g_V = -1/2 + 2sin^2 = {gV:+.4f},"
          f" accidentally small)")
    print(f"  |Z source| / |EM source| = {abs(gZ / e_em):.4f}")
    print(f"\n  {'':<10}{'Q':>6}{'T3':>7}{'Y':>7}{'g_V':>9}{'EM src':>10}{'Z src':>10}")
    for nm, T3, Y in (("electron", -0.5, -0.5), ("neutrino", +0.5, -0.5)):
        Q = T3 + Y
        gv = T3 - 2 * Q * sw2
        print(f"  {nm:<10}{Q:>+6.0f}{T3:>+7.1f}{Y:>+7.1f}{gv:>+9.4f}"
              f"{e_em * Q:>+10.4f}{pref * gv:>+10.4f}")
    print(f"  neutrino Z source is {gV_nu / abs(gV):.2f}x the electron's Z source,"
          f" and {pref * gV_nu / e_em:.2f}x the electron's EM source")

    pk = Packet(N=128, a=0.005, sigma=0.016)
    print(f"\n  lattice N={pk.N}, a={pk.a} GeV^-1, box {pk.L:.3f} = {pk.L * P.m_Z:.0f}/m_Z")
    print(f"  m = {pk.mass} GeV, p_z = {pk.pz} GeV  ->  E = {pk.E_c:.2f},"
          f"  v = {pk.v:.4f},  gamma = {pk.gamma:.4f}")

    # --- checks -----------------------------------------------------------
    J0, Jv = pk.current(0.0)
    norm = J0.sum() * pk.a**3
    vmean = pk.mean_velocity(0.0)
    dt = 5e-4
    idx = np.arange(pk.N) * pk.a
    zc = lambda t: ((pk.current(t)[0].sum(axis=(0, 1)) * idx).sum()
                    / pk.current(t)[0].sum())
    drift = (zc(dt) - zc(0.0)) / dt
    print(f"\n  norm  = {norm:.8f}   (exactly conserved: evolution is a phase)")
    print(f"  <v>   = ({vmean[0]:+.2e}, {vmean[1]:+.2e}, {vmean[2]:.5f})")
    print(f"  d<z>/dt = {drift:.5f}   -> Ehrenfest holds to {abs(drift / vmean[2] - 1):.1e}")
    print(f"  (<v_z> < p/E = {pk.v:.4f} because the packet has a momentum spread)")

    # --- fields -----------------------------------------------------------
    print("\n  solving fields ...")
    E_mv, B_mv = moving_fields(pk, J0, Jv, -e_em, 0.0)
    E_Z, _ = moving_fields(pk, J0, Jv, gZ, P.m_Z)
    E_nu, _ = moving_fields(pk, J0, Jv, gZ_nu, P.m_Z)   # same packet, neutrino charges

    pk_rest = Packet(N=pk.N, a=pk.a, mass=pk.mass, pz=0.0, sigma=0.016)
    J0r, Jvr = pk_rest.current(0.0)
    E_rest, B_rest = moving_fields(pk_rest, J0r, Jvr, -e_em, 0.0)

    c = pk.N // 2
    Eperp = np.sqrt(E_mv[0] ** 2 + E_mv[1] ** 2)
    print(f"  |B|max / |E|max (moving) = "
          f"{np.sqrt(sum(b**2 for b in B_mv)).max() / np.sqrt(sum(f**2 for f in E_mv)).max():.4f}"
          f"   (should approach v = {pk.v:.4f})")
    Bm_rest = np.sqrt(sum(b**2 for b in B_rest)).max()
    net = [abs(J.sum()) * pk_rest.a**3 for J in Jvr]
    print(f"  |B|max (at rest)         = {Bm_rest:.2e}   NOT zero: this is the")
    print(f"     electron's SPIN magnetic moment. Net current integrates to "
          f"{max(net):.1e} (pure dipole,")
    print(f"     no monopole) and |B| falls as 1/r^3 -- it fell out of the Dirac "
          f"equation, unasked.")

    # E_perp/E_par at equal distance should be gamma^3 for a point charge
    d = 20
    e_tr = np.sqrt(sum(f[c + d, c, c] ** 2 for f in E_mv))
    e_lo = np.sqrt(sum(f[c, c, c + d] ** 2 for f in E_mv))
    ratio_meas = e_tr / e_lo
    print(f"  at r = {d * pk.a:.3f}:  E_perp/E_par = {ratio_meas:.3f}"
          f"   vs point-charge gamma^3 = {pk.gamma**3:.3f}")
    g3, rows = validate_boost(pk.v)
    print(f"\n  solver check (rigid Gaussian, no Dirac): E_perp/E_par -> gamma^3 "
          f"= {g3:.2f}")
    for N_, a_, L_, r_, rat_ in rows:
        print(f"    N={N_:4d} a={a_:.3f}  L={L_:.2f}  r={r_:.3f}  ratio {rat_:6.3f}")
    print("    (the packet panels use a smaller box, so they under-read; the window"
          "\n     gamma*sigma << r << L/4 is what a periodic box squeezes)")

    del E_Z
    fine = Packet(N=96, a=0.003, mass=pk.mass, pz=0.0, sigma=0.012)
    J0f, Jvf = fine.current(0.0)
    E_emf, _ = moving_fields(fine, J0f, Jvf, -e_em, 0.0)
    E_Zf, _ = moving_fields(fine, J0f, Jvf, gZ, P.m_Z)
    r_em, f_em = potential_profile(fine, J0f, -e_em, 0.0)
    r_z, f_z = potential_profile(fine, J0f, gZ, P.m_Z)
    r_nu, f_nu = potential_profile(fine, J0f, gZ_nu, P.m_Z)
    del E_emf, E_Zf
    print(f"\n  Z panel on a finer grid: N={fine.N}, a={fine.a}, "
          f"a*m_Z = {fine.a * P.m_Z:.3f}, box = {fine.L * P.m_Z:.0f}/m_Z")

    d = dict(
        a=pk.a, L=pk.L, N=pk.N, v=pk.v, gamma=pk.gamma, mass=pk.mass, pz=pk.pz,
        m_Z=P.m_Z, sigma=0.012,
        rho_rest=J0r[:, c, :], rho_mv=J0[:, c, :],
        Ex_rest=E_rest[0][:, c, :], Ez_rest=E_rest[2][:, c, :],
        Ex_mv=E_mv[0][:, c, :], Ez_mv=E_mv[2][:, c, :],
        Ex_nu=E_nu[0][:, c, :], Ez_nu=E_nu[2][:, c, :],
        r_nu=r_nu, f_nu=f_nu, sw2=sw2,
        By_mv=B_mv[1][:, c, :], By_rest=B_rest[1][:, c, :],
        Etr=np.sqrt(sum(f[:, c, c] ** 2 for f in E_mv)),
        Elo=np.sqrt(sum(f[c, c, :] ** 2 for f in E_mv)),
        Etr0=np.sqrt(sum(f[:, c, c] ** 2 for f in E_rest)),
        r_em=r_em, f_em=f_em, r_z=r_z, f_z=f_z,
    )
    del E_nu
    with open(CACHE, "wb") as fh:
        pickle.dump(d, fh)
    make_figure(**d)
    print("\nwrote electron_fields.png  (cached in " + CACHE + ")")


def make_figure(**kw):
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    from matplotlib.colors import LogNorm

    N, a, L, m_Z = kw["N"], kw["a"], kw["L"], kw["m_Z"]
    v, gamma, mass, pz, sw2 = kw["v"], kw["gamma"], kw["mass"], kw["pz"], kw["sw2"]

    plt.rcParams.update({
        "font.size": 13, "axes.titlesize": 15, "axes.labelsize": 14,
        "xtick.labelsize": 12, "ytick.labelsize": 12, "legend.fontsize": 11.5,
    })
    fig, axes = plt.subplots(2, 3, figsize=(19.5, 11.2))
    g = (np.arange(N) * a - L / 2) * m_Z
    win = 6.0

    trio = ((kw["Ex_rest"], kw["Ez_rest"]), (kw["Ex_mv"], kw["Ez_mv"]),
            (kw["Ex_nu"], kw["Ez_nu"]))
    inwin = np.abs(g) <= win
    sub = np.ix_(inwin, inwin)
    # one shared colour scale across all three, so the panels are comparable
    vhi = max(np.sqrt(Ex**2 + Ez**2)[sub].max() for Ex, Ez in trio)
    vlo = vhi / 120.0

    def panel(ax, rho, Ex, Ez, title, showB=None, lines=True):
        mesh = ax.pcolormesh(g, g, np.clip(np.sqrt(Ex**2 + Ez**2), vlo, vhi),
                             norm=LogNorm(vmin=vlo, vmax=vhi), cmap="inferno",
                             shading="auto", rasterized=True)
        if lines:
            ax.streamplot(g, g, Ez, Ex, color="w", density=1.1, linewidth=0.8,
                          arrowsize=0.8)
        ax.contour(g, g, rho / rho.max(), levels=[0.15, 0.6], colors="w",
                   linewidths=1.0, linestyles="--", alpha=0.7)
        if showB is not None:
            lev = np.array([-0.6, -0.3, 0.3, 0.6]) * np.abs(showB).max()
            ax.contour(g, g, showB, levels=lev, colors="#4fc3f7",
                       linewidths=1.4, linestyles="-")
        ax.set_xlim(-win, win)
        ax.set_ylim(-win, win)
        ax.set_aspect("equal")
        ax.set_xlabel(r"$m_Z\,z$   (direction of motion)")
        ax.set_ylabel(r"$m_Z\,x$")
        ax.set_title(title)
        return mesh

    tag = dict(fc="0.12", ec="none", alpha=0.65, pad=2.5)
    panel(axes[0, 0], kw["rho_rest"], kw["Ex_rest"], kw["Ez_rest"],
          r"(a)  electron at rest — Coulomb + spin dipole", showB=kw["By_rest"])
    axes[0, 0].text(0.03, 0.03, "colour $=|E|$    white $=$ field lines\n"
                    "dashed $=$ charge density    cyan $=B$",
                    transform=axes[0, 0].transAxes, fontsize=10.5, color="w", bbox=tag)

    panel(axes[0, 1], kw["rho_mv"], kw["Ex_mv"], kw["Ez_mv"],
          rf"(b)  electron, $v={v:.3f}c$ — pancaked", showB=kw["By_mv"])
    axes[0, 1].annotate("", xy=(4.6, 0), xytext=(2.4, 0),
                        arrowprops=dict(arrowstyle="-|>", color="w", lw=2.6))

    # no streamlines here: drawing field lines across the black region would
    # imply a field that is not there
    mesh = panel(axes[0, 2], kw["rho_mv"], kw["Ex_nu"], kw["Ez_nu"],
                 rf"(c)  neutrino, $v={v:.3f}c$ — $Z$ field only", lines=False)
    axes[0, 2].annotate("", xy=(4.6, 0), xytext=(2.4, 0),
                        arrowprops=dict(arrowstyle="-|>", color="w", lw=2.6))
    axes[0, 2].text(0.03, 0.03, "identical wave packet.\n"
                    "$Q=0$, so no electric field\nat any distance.",
                    transform=axes[0, 2].transAxes, fontsize=10.5, color="w", bbox=tag)
    fig.colorbar(mesh, ax=axes[0, 2], shrink=0.88, pad=0.02,
                 label=r"field strength (shared scale)")

    # (d) transverse vs longitudinal --------------------------------------
    ax = axes[1, 0]
    half = slice(N // 2 + 1, N)
    rr = g[half]
    ok = rr <= 8.0
    rr = rr[ok]
    ax.loglog(rr, kw["Etr"][half][ok], color="#c0392b", lw=2.4, label="moving, transverse")
    ax.loglog(rr, kw["Elo"][half][ok], color="#2471a3", lw=2.4, ls="--",
              label="moving, longitudinal")
    ax.loglog(rr, kw["Etr0"][half][ok], color="0.35", lw=1.8, ls=":", label="at rest")
    j = len(rr) - 6
    guide = rr >= 2.0
    ref = kw["Etr"][half][ok][j] * (rr / rr[j]) ** -2
    ax.loglog(rr[guide], ref[guide], color="0.55", lw=1.4, alpha=0.9)
    ax.text(rr[j] * 0.80, ref[j] * 2.4, r"$1/r^2$", fontsize=13, color="0.4")
    i = np.argmin(np.abs(rr - 6.0))
    ax.text(0.30, 0.90,
            rf"$E_\perp/E_\parallel = {kw['Etr'][half][ok][i] / kw['Elo'][half][ok][i]:.1f}$"
            rf" at $m_Zr=6$" "\n" rf"point-charge limit: $\gamma^3={gamma**3:.1f}$",
            transform=ax.transAxes, fontsize=11.5, color="0.2",
            bbox=dict(fc="w", ec="none", alpha=0.85, pad=2.5))
    ax.set_xlabel(r"distance,  $m_Z\,r$")
    ax.set_ylabel(r"$|E|$")
    ax.set_title(r"(d)  electron: boosted $\perp$, suppressed $\parallel$")
    ax.legend(loc="lower left", framealpha=0.95)
    ax.grid(alpha=0.2, which="both")

    # (e) all three reaches on one plot ------------------------------------
    ax = axes[1, 1]
    ax.semilogy(kw["r_em"] * m_Z, kw["f_em"], color="#c0392b", lw=2.8,
                label=r"electron, photon")
    ax.semilogy(kw["r_nu"] * m_Z, kw["f_nu"], color="#8e44ad", lw=2.8,
                label=r"neutrino, $Z$")
    ax.semilogy(kw["r_z"] * m_Z, kw["f_z"], color="#1e8449", lw=2.4, ls="--",
                label=r"electron, $Z$")
    ax.axvline(1.0, color="0.5", lw=1.2, ls=":")
    ax.text(1.15, 3e-7, r"$r = 1/m_Z$", fontsize=11, color="0.4")
    ax.set_xlim(0, 9.2)
    ax.set_xlabel(r"distance,  $m_Z\,r$")
    ax.set_ylabel(r"$r\,|A^0|$   (Lorenz gauge)")
    ax.set_title("(e)  flat $=$ infinite reach, sloped $=$ screened")
    ax.legend(loc="upper right", framealpha=0.95)
    ax.grid(alpha=0.2, which="both")
    rn, fn, fe = kw["r_nu"] * m_Z, kw["f_nu"], kw["f_em"]
    i0, i1 = int(np.argmax(fn)), int(np.argmin(np.abs(rn - 8.0)))

    def pw(x):
        ex = int(np.floor(np.log10(x)))
        return rf"{x / 10**ex:.1f}\times 10^{{{ex}}}"

    ax.text(0.035, 0.16,
            "neutrino $Z$ / electron photon:\n"
            rf"${fn[i0] / fe[i0]:.2f}$ at $m_Zr={rn[i0]:.1f}$,"
            rf"   ${pw(fn[i1] / fe[i1])}$ at $m_Zr=8$",
            transform=ax.transAxes, fontsize=11, color="0.25",
            bbox=dict(fc="w", ec="none", alpha=0.85, pad=2.0))

    # (f) the quantum numbers ----------------------------------------------
    ax = axes[1, 2]
    ax.axis("off")
    e_em = np.sqrt(4 * np.pi / 137.036)
    pref = 0.6533 / (2 * np.cos(np.arcsin(np.sqrt(sw2))))
    rows = [("", "Q", "T³", "Y", "g_V", "EM", "Z")]
    for nm, T3, Y in (("electron", -0.5, -0.5), ("neutrino", +0.5, -0.5)):
        Q = T3 + Y
        gv = T3 - 2 * Q * sw2
        rows.append((nm, f"{Q:+.0f}", f"{T3:+.1f}", f"{Y:+.1f}", f"{gv:+.3f}",
                     f"{e_em * Q:+.3f}", f"{pref * gv:+.3f}"))
    xs = [0.02, 0.36, 0.47, 0.59, 0.73, 0.87, 1.00]
    ax.text(0.5, 1.0, r"$Q = T^3 + Y$   and   $g_V = T^3 - 2Q\sin^2\theta_W$",
            ha="center", fontsize=13, transform=ax.transAxes)
    for ri, row in enumerate(rows):
        yy = 0.90 - ri * 0.085
        for xi, (x, cell) in enumerate(zip(xs, row)):
            ax.text(x, yy, cell, fontsize=12.5, family="monospace",
                    weight="bold" if ri == 0 else "normal",
                    ha="left" if xi == 0 else "right", transform=ax.transAxes,
                    color="0.15" if ri == 0 else "0.25")
    ax.plot([0.02, 1.0], [0.60, 0.60], color="0.8", lw=1.0,
            transform=ax.transAxes, clip_on=False)
    ax.text(0.02, 0.535,
            "The two swap roles. The electron's own $Z$\n"
            r"coupling is accidentally tiny because"
            "\n"
            r"$\sin^2\theta_W \approx 1/4$ nearly cancels $T^3$."
            "\n\n"
            "And note the couplings themselves:\n"
            r"$g = 0.653$ against $e = 0.303$."
            "\n"
            "The weak coupling is twice the\n"
            "electromagnetic one. What makes the weak\n"
            r"force weak is $m_Z$, not the charge —"
            "\n"
            "panel (e), not this table.",
            fontsize=11.8, va="top", linespacing=1.45,
            transform=ax.transAxes, color="0.2")

    fig.suptitle(
        rf"An electron and a neutrino as identical positive-energy Dirac packets "
        rf"($m={mass:.0f}$ GeV, $p_z={pz:.0f}$ GeV) — only the charges differ"
        "\n"
        r"Neither carries a $W$ field: the charged current turns one into the other, "
        r"so a one-lepton state has $\langle J^+\rangle = 0$",
        fontsize=15,
    )
    fig.tight_layout(rect=(0, 0, 1, 0.925))
    fig.savefig("electron_fields.png", dpi=135)


if __name__ == "__main__":
    main(replot="--replot" in sys.argv)
