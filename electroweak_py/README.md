# The Higgs mechanism on a lattice

Note: This is code written entirely by Claude Opus 5, with prompts recorded in prompts.md.

Real-time classical lattice field theory for the SU(2)<sub>L</sub> × U(1)<sub>Y</sub>
Higgs sector. Four numbers go in — `v`, `λ`, `g`, `g'` — and the whole boson spectrum
comes out, measured from oscillation frequencies rather than read off a formula.

```bash
python verify.py     # correctness tests -- run this first
python demo.py       # spectrum + higgs_mechanism.png   (~2.5 min)
python demo.py --replot   # redraw the figure from cached results
python potential_plot.py  # -> higgs_potential.png
python w_source_demo.py   # -> w_source.png   (~2 min; --replot to redraw)
```

## Files

| file | what it is |
|---|---|
| `higgs_gauge.py` | the physics: fields, energy, forces, integrator, probes |
| `verify.py` | five correctness tests, all passing |
| `demo.py` | the demonstrations and the figure |
| `potential_plot.py` | the potential itself: double well, Mexican hat, vacuum manifold |
| `w_source_demo.py` | driving the W field: Yukawa blob, evanescent vs radiating, cutoff at $m_W$ |

## Conventions

The dominant set (PDG / Peskin / Schwartz):

- `Q = T³ + Y` with `Y_Φ = +1/2`, `Tᵃ = σᵃ/2`
- `V = μ²Φ†Φ + λ(Φ†Φ)²` with `μ² = −λv² < 0`
- `⟨Φ⟩ = (0, v/√2)ᵀ`, `v = 246.22 GeV`
- `m_h = √(2λ)·v`, `m_W = gv/2`, `m_Z = v√(g²+g'²)/2`, `m_γ = 0`

Everything in GeV with `ħ = c = 1`; the lattice spacing `a` is in GeV⁻¹.

## What it does

Fields live on a 3D periodic lattice in temporal gauge (`W^a_0 = B_0 = 0`):
the complex doublet `Φ`, the SU(2) potential `W^a_j`, the U(1) potential `B_j`,
and their time derivatives. The energy functional is

```
E = Σ_x a³ [ Π†Π + Σ_j (D_jΦ)†(D_jΦ) + V(Φ)
             + ½ Σ (Ẇ^a_j)² + ¼ Σ (F^a_jk)²
             + ½ Σ (Ḃ_j)²   + ¼ Σ (Fb_jk)²  ]
```

and the equations of motion are its gradient, integrated with velocity Verlet.
No gauge-boson mass term appears anywhere. The W and Z masses arise entirely from
`|D_μΦ|²` evaluated with `Φ` at its VEV.

## Results

```
mode              measured [GeV]  tree-level [GeV]     ratio
W+- (charged)            80.4286           80.4278  1.000010
Z (neutral)              91.2437           91.2428  1.000010
photon                    0.0000          0 (flat)        --
h (Higgs)               125.1115          125.1127  0.999990
```

The figure shows five things: identical kicks along the W, Z and photon directions
(the photon does not move at all); every mass scaling linearly with the VEV; the W
dispersion relation `ω² = k̂² + m_W²`; the W going from free drift to ringing as the
condensate forms; and the instantaneous W frequency tracking `g⟨r⟩/2` throughout.

## Verification

`verify.py` checks five things. Tests 1 and 4 are exact statements with tight
tolerances; 2 and 3 are convergence statements, which is all a discretization can
promise.

1. **Analytic forces == −∇(energy)**, by central differences — agrees to ~10⁻⁹.
   This is the load-bearing test: it validates every hand-derived term, including
   the non-abelian Yang–Mills force and the Higgs currents.
2. **Verlet energy error** is bounded and scales as `dt²`.
3. **Lattice gauge invariance** is `O(a²)`, measured with an exact finite U(1)
   transformation.
4. **The photon direction is exactly flat** at the VEV (zero energy cost, zero force,
   to machine precision) because `Q` annihilates `⟨Φ⟩`. The Z direction is not.
5. **The k=0 spectrum** matches the tree-level formulas to ~10⁻⁵.

## Scope and limitations

- **Gauge fields are non-compact** — ordinary real fields with naive lattice
  differences, not compact link variables with a Wilson action. Gauge invariance is
  therefore `O(a²)` rather than exact, and there is no exactly conserved Gauss law.
  For topological physics (sphalerons, Chern–Simons number, baryogenesis) you want
  compact SU(2) links instead.
- **k=0 results are exact.** For spatially uniform fields every lattice difference
  vanishes identically, so the mass measurements carry no discretization error at
  all — a 2³ lattice gives the same answer as 64³, for any `a`. That is why
  `demo.py` does its spectroscopy on a 2³ lattice.
- **Classical, not quantum.** No loops, no running couplings, no thermal
  fluctuations. Tree-level relations are what it should reproduce, and does.
- **External sources must be conserved.** `forces(..., jW=...)` adds a prescribed
  current to the SU(2) acceleration. This discretization computes the Yang-Mills force
  as a *backward* difference of `F`, so the Gauss constraint reads `d^-_m j_m = 0`.
  A source that is divergence-free under forward differences will drive the
  longitudinal mode with no restoring force and the relaxation will never converge.
- **`damping` in `step()` is not Standard Model physics.** It stands in for Hubble
  friction in an expanding universe, and is applied to the Higgs only so that a
  gauge probe keeps ringing at readable amplitude. Off by default.
- **Fermions are absent**, so no Yukawa couplings and no top quark.

## Things worth trying

- Set `broken=False` in `Params` for `μ² > 0`: the symmetric phase, where the gauge
  bosons are massless and all four scalars are massive.
- Vary `g'` and watch `m_W/(m_Z cos θ_W) = 1` hold — the tree-level rho parameter.
- Set `g = gp = 0` to recover the ungauged theory and see the three Goldstone modes
  as genuine massless propagating waves.
- Push the drive frequency in `w_source_demo.py` right up to threshold and watch the
  steady state take longer and longer to establish — the group velocity goes to zero
  there, which is the 1D van Hove singularity in the theory curve.
- Add a thermal mass by hand, `μ² → μ² + cT²`, and ramp `T` down through
  `T_c = √(−μ²/c)` to get the textbook symmetry-restoration sequence.
