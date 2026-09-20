"""
Real-time classical lattice simulation of the SU(2)_L x U(1)_Y Higgs sector.

Fields (3D periodic spatial lattice, temporal gauge W^a_0 = B_0 = 0):

    Phi[i,x,y,z]     complex Higgs doublet, i = 0 (upper, charged) / 1 (lower, neutral)
    Pi               = d(Phi)/dt
    W[a,j,x,y,z]     SU(2) gauge potential, a = 1..3 adjoint, j = 1..3 spatial
    Wd               = d(W)/dt      (the SU(2) electric field)
    B[j,x,y,z]       U(1)_Y gauge potential
    Bd               = d(B)/dt

Conventions (the "dominant" set: PDG / Peskin / Schwartz):

    Q = T^3 + Y,  Y_Phi = +1/2,  T^a = sigma^a / 2
    V = mu^2 Phi^dag Phi + lambda (Phi^dag Phi)^2,   mu^2 = -lambda v^2 < 0
    <Phi> = (0, v/sqrt(2))^T  with v = 246.22 GeV
    m_h = sqrt(2 lambda) v,  m_W = g v / 2,  m_Z = v sqrt(g^2 + g'^2) / 2,  m_gamma = 0

Energy functional (hbar = c = 1; everything in GeV, lattice spacing a in GeV^-1):

    E = sum_x a^3 [ Pi^dag Pi                          Higgs kinetic
                  + sum_j (D_j Phi)^dag (D_j Phi)      Higgs gradient
                  + V(Phi)                             Higgs potential
                  + (1/2) sum_{a,j} (Wd^a_j)^2         SU(2) electric
                  + (1/4) sum_{a,j,k} (F^a_jk)^2       SU(2) magnetic
                  + (1/2) sum_j (Bd_j)^2               U(1) electric
                  + (1/4) sum_{j,k} (Fb_jk)^2 ]        U(1) magnetic

Gauge fields are NON-COMPACT (ordinary real fields with naive lattice differences)
rather than compact link variables. Gauge invariance therefore holds only to O(a^2).
That is exact for the spatially uniform (k=0) modes used to measure masses, and fine
for small-amplitude waves, but this code is NOT suitable for topological/sphaleron
physics, where you want compact SU(2) link variables and the Wilson action.

All analytic forces in this module are verified against numerical gradients of
energy_potential() in verify.py -- run that first if you change anything.
"""

from __future__ import annotations

from dataclasses import dataclass, replace

import numpy as np

# ---------------------------------------------------------------------------
# group theory constants
# ---------------------------------------------------------------------------

SIGMA = np.array(
    [
        [[0, 1], [1, 0]],
        [[0, -1j], [1j, 0]],
        [[1, 0], [0, -1]],
    ],
    dtype=complex,
)
TGEN = SIGMA / 2.0  # T^a = sigma^a / 2

EPS = np.zeros((3, 3, 3))
for _i, _j, _k in [(0, 1, 2), (1, 2, 0), (2, 0, 1)]:
    EPS[_i, _j, _k] = 1.0
    EPS[_i, _k, _j] = -1.0

Y_PHI = 0.5  # hypercharge of the Higgs doublet in the Q = T^3 + Y convention


# ---------------------------------------------------------------------------
# parameters
# ---------------------------------------------------------------------------


@dataclass(frozen=True)
class Params:
    """Physical and lattice parameters. Masses in GeV, lengths in GeV^-1."""

    N: int = 8  # sites per spatial direction
    a: float = 0.0025  # lattice spacing [GeV^-1]
    v: float = 246.22  # Higgs VEV (real-field radius) [GeV]
    lam: float = 0.1291  # quartic; m_h = sqrt(2 lam) v
    g: float = 0.6533  # SU(2)_L coupling
    gp: float = 0.3500  # U(1)_Y coupling
    broken: bool = True  # False => mu^2 = +lam v^2, symmetric phase

    @property
    def mu2(self) -> float:
        return -self.lam * self.v**2 if self.broken else +self.lam * self.v**2

    # --- tree-level predictions we will test against -----------------------
    @property
    def m_h(self) -> float:
        return np.sqrt(2.0 * self.lam) * self.v

    @property
    def m_W(self) -> float:
        return self.g * self.v / 2.0

    @property
    def m_Z(self) -> float:
        return self.v * np.sqrt(self.g**2 + self.gp**2) / 2.0

    @property
    def theta_W(self) -> float:
        return np.arctan2(self.gp, self.g)

    def summary(self) -> str:
        return (
            f"v = {self.v:.2f} GeV   lambda = {self.lam:.4f}   "
            f"g = {self.g:.4f}   g' = {self.gp:.4f}\n"
            f"predicted:  m_h = {self.m_h:7.2f}   m_W = {self.m_W:7.2f}   "
            f"m_Z = {self.m_Z:7.2f}   m_gamma = 0\n"
            f"sin^2(theta_W) = {np.sin(self.theta_W)**2:.4f}   "
            f"m_W / (m_Z cos theta_W) = {self.m_W / (self.m_Z * np.cos(self.theta_W)):.6f}"
        )


@dataclass
class State:
    Phi: np.ndarray  # (2,N,N,N) complex
    Pi: np.ndarray  # (2,N,N,N) complex
    W: np.ndarray  # (3,3,N,N,N) real  [adjoint, spatial]
    Wd: np.ndarray  # (3,3,N,N,N) real
    B: np.ndarray  # (3,N,N,N) real    [spatial]
    Bd: np.ndarray  # (3,N,N,N) real

    def copy(self) -> "State":
        return State(*(f.copy() for f in (self.Phi, self.Pi, self.W, self.Wd, self.B, self.Bd)))


# ---------------------------------------------------------------------------
# lattice shifts -- spatial axes are always the LAST three
# ---------------------------------------------------------------------------


def fwd(f: np.ndarray, j: int) -> np.ndarray:
    """f(x + j_hat)"""
    return np.roll(f, -1, axis=f.ndim - 3 + j)


def bwd(f: np.ndarray, j: int) -> np.ndarray:
    """f(x - j_hat)"""
    return np.roll(f, +1, axis=f.ndim - 3 + j)


# ---------------------------------------------------------------------------
# gauge action on the doublet
# ---------------------------------------------------------------------------


def gauge_matrix_act(Wj: np.ndarray, Bj: np.ndarray, Phi: np.ndarray, p: Params) -> np.ndarray:
    """G_j Phi  with  G_j = g W^a_j T^a + g' Y B_j, a Hermitian 2x2 acting on the doublet."""
    su2 = np.einsum("aik,a...,k...->i...", TGEN, Wj, Phi)
    return p.g * su2 + (p.gp * Y_PHI) * Bj * Phi


def covariant_derivative(st: State, p: Params) -> np.ndarray:
    """D_j Phi, shape (3,2,N,N,N).

    D_j Phi(x) = [Phi(x+j) - Phi(x)] / a  -  i G_j(x) * [Phi(x) + Phi(x+j)] / 2

    The gauge term is evaluated at the link MIDPOINT. Putting it on Phi(x) instead
    also has the right continuum limit but degrades lattice gauge invariance from
    O(a^2) to O(a) -- see verify.py test [3].
    """
    out = np.empty((3,) + st.Phi.shape, dtype=complex)
    for j in range(3):
        Phi_f = fwd(st.Phi, j)
        mid = 0.5 * (st.Phi + Phi_f)
        out[j] = (Phi_f - st.Phi) / p.a - 1j * gauge_matrix_act(st.W[:, j], st.B[j], mid, p)
    return out


# ---------------------------------------------------------------------------
# field strengths
# ---------------------------------------------------------------------------


def su2_field_strength(W: np.ndarray, p: Params) -> np.ndarray:
    """F^a_jk, shape (3,3,3,N,N,N), antisymmetric in (j,k).

    F^a_jk = d_j W^a_k - d_k W^a_j + g eps^{abc} W^b_j W^c_k
    """
    N = W.shape[-1]
    F = np.zeros((3, 3, 3, N, N, N))
    for j in range(3):
        for k in range(3):
            if j == k:
                continue
            F[:, j, k] = (
                (fwd(W[:, k], j) - W[:, k]) / p.a
                - (fwd(W[:, j], k) - W[:, j]) / p.a
                + p.g * np.einsum("abc,b...,c...->a...", EPS, W[:, j], W[:, k])
            )
    return F


def u1_field_strength(B: np.ndarray, p: Params) -> np.ndarray:
    """Fb_jk, shape (3,3,N,N,N), antisymmetric in (j,k)."""
    N = B.shape[-1]
    Fb = np.zeros((3, 3, N, N, N))
    for j in range(3):
        for k in range(3):
            if j == k:
                continue
            Fb[j, k] = (fwd(B[k], j) - B[k]) / p.a - (fwd(B[j], k) - B[j]) / p.a
    return Fb


# ---------------------------------------------------------------------------
# energy
# ---------------------------------------------------------------------------


def potential_density(Phi: np.ndarray, p: Params) -> np.ndarray:
    rho = np.real(np.einsum("i...,i...->...", Phi.conj(), Phi))  # Phi^dag Phi
    return p.mu2 * rho + p.lam * rho**2


def energy_potential(st: State, p: Params) -> float:
    """Everything that does NOT involve time derivatives (the 'potential' half)."""
    D = covariant_derivative(st, p)
    grad = float(np.real(np.sum(D.conj() * D)))
    pot = potential_density(st.Phi, p).sum()
    F = su2_field_strength(st.W, p)
    Fb = u1_field_strength(st.B, p)
    mag = 0.25 * (F**2).sum() + 0.25 * (Fb**2).sum()
    return p.a**3 * (grad + pot + mag)


def energy_kinetic(st: State, p: Params) -> float:
    kin_phi = float(np.real(np.sum(st.Pi.conj() * st.Pi)))
    kin_gauge = 0.5 * (st.Wd**2).sum() + 0.5 * (st.Bd**2).sum()
    return p.a**3 * (kin_phi + kin_gauge)


def energy_breakdown(st: State, p: Params) -> dict:
    D = covariant_derivative(st, p)
    F = su2_field_strength(st.W, p)
    Fb = u1_field_strength(st.B, p)
    c = p.a**3
    return {
        "higgs_kinetic": c * float(np.real(np.sum(st.Pi.conj() * st.Pi))),
        "higgs_gradient": c * float(np.real(np.sum(D.conj() * D))),
        "higgs_potential": c * potential_density(st.Phi, p).sum(),
        "su2_electric": c * 0.5 * (st.Wd**2).sum(),
        "su2_magnetic": c * 0.25 * (F**2).sum(),
        "u1_electric": c * 0.5 * (st.Bd**2).sum(),
        "u1_magnetic": c * 0.25 * (Fb**2).sum(),
    }


def total_energy(st: State, p: Params) -> float:
    return energy_kinetic(st, p) + energy_potential(st, p)


# ---------------------------------------------------------------------------
# forces  (all verified against numerical gradients in verify.py)
# ---------------------------------------------------------------------------


def forces(st: State, p: Params, jW: np.ndarray | None = None):
    """Return (Phi_ddot, W_ddot, B_ddot).

    `jW` is an optional EXTERNAL source added to the SU(2) acceleration, with the
    same shape as st.W. It stands for a prescribed matter current -- e.g. the
    transition current of a weak decay -- rather than anything dynamical.

    Phi_ddot = -dU/dPhi^*        W_ddot = -dU/dW        B_ddot = -dU/dB
    (per unit a^3, i.e. densities -- the a^3 cancels out of the equations of motion)
    """
    D = covariant_derivative(st, p)

    # ---- Higgs -----------------------------------------------------------
    # With D_j Phi(x) = P_j(x) Phi(x) + Q_j(x) Phi(x+j),
    #   P_j = -(1/a) - i G_j/2,   Q_j = +(1/a) - i G_j/2,
    # so  dU_grad/dPhi^*(y) = sum_j [ P_j^dag(y) D_j(y) + Q_j^dag(y-j) D_j(y-j) ]
    #                       = sum_j [ (D_j(y-j) - D_j(y))/a
    #                                 + (i/2)( G_j D_j |_y  +  G_j D_j |_{y-j} ) ]
    dU_dPhistar = np.zeros_like(st.Phi)
    for j in range(3):
        GD = gauge_matrix_act(st.W[:, j], st.B[j], D[j], p)
        dU_dPhistar += (bwd(D[j], j) - D[j]) / p.a
        dU_dPhistar += 0.5j * (GD + bwd(GD, j))
    rho = np.real(np.einsum("i...,i...->...", st.Phi.conj(), st.Phi))
    dU_dPhistar += (p.mu2 + 2.0 * p.lam * rho) * st.Phi
    Phi_ddot = -dU_dPhistar

    # ---- SU(2) -----------------------------------------------------------
    F = su2_field_strength(st.W, p)
    W_ddot = np.zeros_like(st.W)
    for m in range(3):
        # Yang-Mills: + sum_j backward_diff_j( F[d,j,m] )  - g eps^{adc} F^a_mk W^c_k
        acc = np.zeros_like(st.W[:, 0])
        for j in range(3):
            Fjm = F[:, j, m]
            acc += (Fjm - bwd(Fjm, j)) / p.a
        for k in range(3):
            acc -= p.g * np.einsum("adc,a...,c...->d...", EPS, F[:, m, k], st.W[:, k])
        # Higgs current: + 2 g Im[ Phi_mid^dag T^d D_m Phi ], Phi_mid on the link
        Phi_mid = 0.5 * (st.Phi + fwd(st.Phi, m))
        cur = np.einsum("i...,dik,k...->d...", Phi_mid.conj(), TGEN, D[m])
        acc += 2.0 * p.g * np.imag(cur)
        W_ddot[:, m] = acc

    # ---- U(1) ------------------------------------------------------------
    Fb = u1_field_strength(st.B, p)
    B_ddot = np.zeros_like(st.B)
    for m in range(3):
        acc = np.zeros_like(st.B[0])
        for j in range(3):
            acc += (Fb[j, m] - bwd(Fb[j, m], j)) / p.a
        Phi_mid = 0.5 * (st.Phi + fwd(st.Phi, m))
        cur = np.einsum("i...,i...->...", Phi_mid.conj(), D[m])
        acc += 2.0 * p.gp * Y_PHI * np.imag(cur)
        B_ddot[m] = acc

    if jW is not None:
        W_ddot = W_ddot + jW

    return Phi_ddot, W_ddot, B_ddot


# ---------------------------------------------------------------------------
# Gauss constraints (conserved by the exact equations of motion)
# ---------------------------------------------------------------------------


def gauss_su2(st: State, p: Params) -> np.ndarray:
    """(D_j Wd_j)^a + 2 g Im[Phi^dag T^a Pi]; should vanish and stay vanishing."""
    out = np.zeros_like(st.Wd[:, 0])
    for j in range(3):
        out += (st.Wd[:, j] - bwd(st.Wd[:, j], j)) / p.a
        out += p.g * np.einsum("abc,b...,c...->a...", EPS, st.W[:, j], st.Wd[:, j])
    cur = np.einsum("i...,aik,k...->a...", st.Phi.conj(), TGEN, st.Pi)
    out += 2.0 * p.g * np.imag(cur)
    return out


def gauss_u1(st: State, p: Params) -> np.ndarray:
    out = np.zeros_like(st.B[0])
    for j in range(3):
        out += (st.Bd[j] - bwd(st.Bd[j], j)) / p.a
    cur = np.einsum("i...,i...->...", st.Phi.conj(), st.Pi)
    out += 2.0 * p.gp * Y_PHI * np.imag(cur)
    return out


# ---------------------------------------------------------------------------
# integrator: velocity Verlet (symplectic, second order)
# ---------------------------------------------------------------------------


def step(st: State, p: Params, dt: float, acc=None, damping: float = 0.0,
         jW: np.ndarray | None = None):
    """One velocity-Verlet step. Pass the previous acceleration to save a force call.

    `damping` (in GeV) adds a friction term -gamma * Pi to the HIGGS velocity only,
    applied by operator splitting at the end of the step. This is NOT part of the
    Standard Model: it stands in for Hubble friction in an expanding universe, which
    is what actually let the early-universe Higgs field settle into its minimum
    instead of oscillating forever. It is applied only to the Higgs so that a gauge
    field used as a probe keeps ringing at constant amplitude and its frequency stays
    readable. Energy is deliberately not conserved when damping > 0.
    """
    if acc is None:
        acc = forces(st, p, jW)
    aPhi, aW, aB = acc

    st.Pi += 0.5 * dt * aPhi
    st.Wd += 0.5 * dt * aW
    st.Bd += 0.5 * dt * aB

    st.Phi += dt * st.Pi
    st.W += dt * st.Wd
    st.B += dt * st.Bd

    acc = forces(st, p, jW)
    aPhi, aW, aB = acc

    st.Pi += 0.5 * dt * aPhi
    st.Wd += 0.5 * dt * aW
    st.Bd += 0.5 * dt * aB

    if damping:
        st.Pi *= np.exp(-damping * dt)

    return acc


def cfl_dt(p: Params, safety: float = 0.15) -> float:
    """Timestep respecting both the lattice light-cone and the heaviest mass."""
    return safety * min(p.a, 1.0 / max(p.m_h, p.m_Z, 1e-12))


# ---------------------------------------------------------------------------
# state construction
# ---------------------------------------------------------------------------


def vacuum_state(p: Params) -> State:
    """Higgs at the VEV in the neutral (lower) component, gauge fields zero."""
    N = p.N
    Phi = np.zeros((2, N, N, N), dtype=complex)
    if p.broken:
        Phi[1] = p.v / np.sqrt(2.0)
    return State(
        Phi=Phi,
        Pi=np.zeros((2, N, N, N), dtype=complex),
        W=np.zeros((3, 3, N, N, N)),
        Wd=np.zeros((3, 3, N, N, N)),
        B=np.zeros((3, N, N, N)),
        Bd=np.zeros((3, N, N, N)),
    )


def random_state(p: Params, amp: float, seed: int = 0) -> State:
    """Small random fields -- used only by the force/gradient tests."""
    rng = np.random.default_rng(seed)
    N = p.N
    sh_phi, sh_W, sh_B = (2, N, N, N), (3, 3, N, N, N), (3, N, N, N)
    return State(
        Phi=amp * (rng.standard_normal(sh_phi) + 1j * rng.standard_normal(sh_phi)),
        Pi=amp * (rng.standard_normal(sh_phi) + 1j * rng.standard_normal(sh_phi)),
        W=amp * rng.standard_normal(sh_W) / p.a,
        Wd=amp * rng.standard_normal(sh_W) / p.a,
        B=amp * rng.standard_normal(sh_B) / p.a,
        Bd=amp * rng.standard_normal(sh_B) / p.a,
    )


# ---------------------------------------------------------------------------
# the neutral gauge-boson rotation:  (W^3, B) -> (Z, A)
# ---------------------------------------------------------------------------


def z_direction(p: Params):
    """Unit vector (cW3, cB) for the Z boson: Z = (g W^3 - g' B)/sqrt(g^2+g'^2)."""
    n = np.hypot(p.g, p.gp)
    return p.g / n, -p.gp / n


def photon_direction(p: Params):
    """Unit vector (cW3, cB) for the photon: A = (g' W^3 + g B)/sqrt(g^2+g'^2)."""
    n = np.hypot(p.g, p.gp)
    return p.gp / n, p.g / n


# ---------------------------------------------------------------------------
# frequency extraction
# ---------------------------------------------------------------------------


def frequency_from_zero_crossings(t: np.ndarray, y: np.ndarray) -> float:
    """Angular frequency of an oscillating signal, from linearly interpolated
    zero crossings. Returns 0.0 if fewer than two crossings occur."""
    y = y - 0.0  # the modes we probe oscillate about zero
    s = np.signbit(y)
    idx = np.nonzero(s[:-1] != s[1:])[0]
    if len(idx) < 2:
        return 0.0
    # linear interpolation for each crossing time
    t0, t1 = t[idx], t[idx + 1]
    y0, y1 = y[idx], y[idx + 1]
    tc = t0 + (t1 - t0) * (-y0) / (y1 - y0)
    # n crossings span (n-1) half periods
    half_periods = len(tc) - 1
    return np.pi * half_periods / (tc[-1] - tc[0])


# ---------------------------------------------------------------------------
# driving a single mode and reading off its frequency
# ---------------------------------------------------------------------------


def evolve_probe(st: State, p: Params, dt: float, n_steps: int, probe, stride: int = 1,
                 damping: float = 0.0):
    """Evolve `st` and record probe(st) as a time series. Returns (t, y)."""
    ts, ys, acc = [], [], None
    for i in range(n_steps):
        acc = step(st, p, dt, acc, damping)
        if (i + 1) % stride == 0:
            ts.append((i + 1) * dt)
            ys.append(probe(st))
    return np.asarray(ts), np.asarray(ys)


def mode_frequency(p: Params, setup, probe, n_periods: float = 12.0,
                   m_ref: float | None = None, safety: float = 0.02) -> float:
    """Perturb the vacuum with setup(state), evolve, and return the angular frequency.

    Returns 0.0 for a mode that does not oscillate (i.e. a massless one at k=0).
    """
    st = vacuum_state(p)
    setup(st)
    dt = cfl_dt(p, safety)
    ref = m_ref if m_ref is not None else max(p.m_W, p.m_h, 1.0)
    n_steps = max(200, int(n_periods * 2.0 * np.pi / ref / dt))
    t, y = evolve_probe(st, p, dt, n_steps, probe)
    return frequency_from_zero_crossings(t, y)


# --- ready-made probes for the four physical k=0 modes ----------------------


def perturb_W(p: Params, eps: float):
    """Uniform W^1 in spatial direction 1 -> the charged W mode."""
    def setup(st):
        st.W[0, 0] += eps
    return setup, (lambda st: st.W[0, 0, 0, 0, 0])


def perturb_Z(p: Params, eps: float):
    cW3, cB = z_direction(p)

    def setup(st):
        st.W[2, 0] += cW3 * eps
        st.B[0] += cB * eps

    return setup, (lambda st: cW3 * st.W[2, 0, 0, 0, 0] + cB * st.B[0, 0, 0, 0])


def perturb_photon(p: Params, eps: float):
    cW3, cB = photon_direction(p)

    def setup(st):
        st.W[2, 0] += cW3 * eps
        st.B[0] += cB * eps

    return setup, (lambda st: cW3 * st.W[2, 0, 0, 0, 0] + cB * st.B[0, 0, 0, 0])


def perturb_higgs(p: Params, eps: float):
    """Radial (physical Higgs) direction."""
    def setup(st):
        st.Phi[1] += eps

    return setup, (lambda st: float(np.real(st.Phi[1, 0, 0, 0]) - p.v / np.sqrt(2.0)))


def perturb_goldstone(p: Params, eps: float, which: int = 0):
    """A tangential (would-be Goldstone) direction.

    which = 0,1 -> the two real components of the upper (charged) doublet entry
    which = 2   -> the imaginary part of the lower (neutral) entry
    """
    if which == 0:
        def setup(st):
            st.Phi[0] += eps
        probe = lambda st: float(np.real(st.Phi[0, 0, 0, 0]))
    elif which == 1:
        def setup(st):
            st.Phi[0] += 1j * eps
        probe = lambda st: float(np.imag(st.Phi[0, 0, 0, 0]))
    else:
        def setup(st):
            st.Phi[1] += 1j * eps
        probe = lambda st: float(np.imag(st.Phi[1, 0, 0, 0]))
    return setup, probe


def lattice_momentum_sq(k_phys: float, a: float) -> float:
    """khat^2 = (4/a^2) sin^2(k a / 2): what the naive lattice Laplacian actually sees."""
    return (4.0 / a**2) * np.sin(k_phys * a / 2.0) ** 2
