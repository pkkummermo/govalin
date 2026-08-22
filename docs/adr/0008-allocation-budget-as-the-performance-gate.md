# Allocation budget as the performance gate

## Status

accepted

## Context and decision

The response lifecycle change (ADR 0007) put a wrapper in front of every write in the framework. It
was measured before merging and costs one allocation per request, but nothing stops the next change
from adding another, and per-request overhead is the one cost an HTTP framework's users cannot
opt out of. Govalin already enforces correctness as a **policy-enforced gate** in CI; performance had
no gate at all.

The obvious gate — run the benchmarks, fail on a slowdown — does not work on the runners CI has.
Shared-vCPU GitHub runners swing wide enough between runs that a threshold tight enough to catch a
real 5% regression fires constantly on noise, and a gate that fails at random is a gate someone
switches off.

**Allocation counts are the signal that survives a noisy runner.** For a given build they are
identical on every run, so the gate is exact rather than statistical: no threshold, no p-value, no
flake. They are also the right proxy for framework overhead, which is mostly allocation and the GC
pressure that follows it — the 14% the static path gained in ADR 0007 came with an allocation fewer,
and the +2% every other route paid came with one more.

`TestAllocationBudget` asserts a measured allocation count per request shape (text, JSON, status
only, before/after handlers, 404, raw handler, static file) using `testing.AllocsPerRun`. The budgets
are what the code does today, not aspirations: raising one is a decision taken in the diff that needs
it, with the reason in the commit message. The failure message says so, because a gate whose fix is
"edit the number until it passes" gates nothing.

Two scoping decisions:

- The gate runs in a **dedicated job on the baseline toolchain and operating system**, mirroring the
  **Dedicated race gate job** and **Selective race coverage**. Escape analysis and inlining change
  between Go releases, so a compatibility-matrix job on a newer version could fail on a difference
  that is not a regression. The platform matters for the same reason: the static file shape allocates
  differently on darwin than on ubuntu, because the standard library takes a different path. The
  budgets are therefore the counts measured on ubuntu, and "identical on every run" holds for a given
  build on a given platform — a developer reproducing them elsewhere may legitimately see other
  numbers.
- It is **off by default** (`GOVALIN_PERF_GATE=1`), so a contributor running `go test ./...` on
  whatever Go version they have does not get a failure that means nothing.

Timings are still reported, as an **advisory** `perf-compare` job that benchstats the PR against its
merge base into the job summary and never fails the build. The base runs its own benchmark files
rather than having the PR's copied in: a benchmark the PR adds or changes would not compile there,
and one that did compile would be measuring a file against a tree it was not written for.

## Considered options

- **Allocation budgets as the gate, timings advisory (chosen)** — see above.
- **benchstat threshold on sec/op as the gate** — the direct reading of "performance regression", and
  unusable on shared runners: the noise floor is wider than the regressions worth catching.
- **A dedicated benchmark machine or a hosted service (Bencher, CodSpeed)** — gives trustworthy
  timing and continuous history. Rejected for now as infrastructure this project does not have; the
  advisory job covers the "did something get much slower" case at no cost.
- **Asserting bytes per operation as well** — bytes move with buffer sizing decisions that are not
  regressions (a larger read buffer allocates more and copies less). The count is the stable signal.
- **Always-on budgets in the normal test run** — simplest, and it makes every contributor's local Go
  version part of the contract. Rejected in favour of an explicit gate job.

## Consequences

- A change that adds a per-request allocation fails CI and has to say so in its diff.
- The budgets need a deliberate update when the framework legitimately allocates more, and when a Go
  release shifts them. The gate job reports the measured number alongside the budget, so the update
  is mechanical once the decision is made.
- Budgets have to be read off the reference platform, not the machine the change was written on.
  Setting them from a local darwin run lands numbers that fail in CI, which is how the per-OS
  difference was found.
- Benchmarks are compiled by the gate job, so one that stops building fails CI rather than rotting
  unnoticed.
- The advisory comparison is silent when the merge base has no comparable benchmarks — including for
  the PR that introduces them.
