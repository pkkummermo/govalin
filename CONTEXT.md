# Govalin

Govalin is an HTTP server framework context focused on predictable request handling and safe extension behavior. This context defines the language for quality and runtime-safety decisions that govern releases.

## Language

**Race-clean build**:
A build and test run with no detected data races under race instrumentation.
_Avoid_: mostly-safe, probably-safe

**Release blocker**:
A condition that must be resolved before a version can be shipped.
_Avoid_: warning-only failure, non-blocking issue

**Stability gate**:
A mandatory quality check that protects runtime correctness and predictable behavior.
_Avoid_: optional check, advisory check

**Root-cause-first remediation**:
Fix shared causes that fail broad quality gates before secondary design cleanup.
_Avoid_: parallel mixed remediation, cosmetic-first fixes

**Compatibility-preserving first fix**:
The first remediation step keeps public helper signatures unchanged to minimize migration risk.
_Avoid_: breaking-first cleanup, broad API rewrite in urgent fix

**Scope-frozen phase one**:
The first remediation phase addresses only the declared release-blocking fault class.
_Avoid_: opportunistic cleanup, multi-goal phase mixing

**Phase-one done criteria**:
Phase one is complete only when race tests and default tests pass without changing the helper API.
_Avoid_: partial green status, implicit completion standards

**Policy-enforced gate**:
A release policy is implemented as mandatory CI checks rather than team convention alone.
_Avoid_: manual-only enforcement, optional pipeline checks

**Gate-first execution order**:
Implementation starts by restoring mandatory quality gates before non-blocking design refactors.
_Avoid_: refactor-first execution, mixed-priority delivery

**Canonical static mount URL**:
The mount root URL is the trailing-slash form; non-slash form redirects to canonical URL.
_Avoid_: dual canonical forms, content served on both root variants

**Permanent canonical redirect**:
Canonical URL normalization for static mount roots uses permanent redirect semantics.
_Avoid_: provisional redirect phases, temporary canonicalization

**Query-preserving canonicalization**:
Canonical redirects normalize path shape while preserving the original query string.
_Avoid_: query dropping, parameter mutation during URL normalization

**Framework-wide redirect integrity**:
All framework-generated redirects must preserve request query parameters unless explicitly documented otherwise.
_Avoid_: plugin-specific redirect semantics, silent intent loss

**Raw-query passthrough**:
Redirect query preservation uses the original raw query bytes without parse/re-encode normalization.
_Avoid_: query canonicalization side effects, reordered parameters

**Pre-merge gate enforcement**:
Mandatory stability gates run on pull requests before merge, not only after push.
_Avoid_: post-merge-only enforcement, delayed blocker detection

**Dedicated race gate job**:
Race detection runs in a separate CI job from default unit tests for isolated quality signals.
_Avoid_: mixed-failure jobs, opaque blocker diagnosis

**Current-toolchain alignment**:
Project and CI Go versions are actively maintained to current supported releases, not left on stale versions.
_Avoid_: long-lived version lag, accidental legacy lock-in

**Toolchain maintenance cadence**:
Go patch levels stay at latest stable for the chosen minor, with monthly minor-review and immediate security patch adoption.
_Avoid_: ad-hoc upgrades, undefined update responsibility

**Selective race coverage**:
Race detection runs only on the current Go baseline while compatibility matrix coverage applies to non-race tests.
_Avoid_: race-on-all-matrix strategy, unbounded CI runtime growth

**Dual-minor compatibility matrix**:
Non-race CI tests run on current stable Go and previous minor Go versions.
_Avoid_: single-version-only compatibility signal, oversized version matrices

**Two-PR rollout**:
Release-blocking stabilization and design refactors are delivered in separate pull requests.
_Avoid_: mixed-risk delivery, bundled stabilization and redesign

**Scope-locked PR1**:
PR1 includes only race-helper stabilization and CI gate/toolchain updates, excluding redirect and static behavior changes.
_Avoid_: scope bleed into routing semantics, mixed stabilization and feature behavior

**Validation rule**:
A check applied to an input that asserts a condition and leaves the value unchanged; it either passes or yields a validation error.
_Avoid_: value-mutating check, silent correction

**Transform step**:
A step in an input-handling chain that normalizes the value (such as trimming surrounding whitespace) and always succeeds, never producing a validation error.
_Avoid_: erroring normalizer, validation-as-transform

**Chain order significance**:
Rules and transforms apply in written order; a transform only affects checks placed after it in the chain.
_Avoid_: implicit reordering, position-independent transforms

**Buffered status**:
A status code set via `Status()` that is held on the call and not yet committed to the response writer.
_Avoid_: eager status write, status sent immediately on set

**Status flush**:
Committing the buffered status to the writer. Happens on the first body write (JSON/Text/HTML/Redirect) or, when no body is written, exactly once at the end of the request lifecycle.
_Avoid_: implicit 200 on unflushed status, multiple status writes

**Lifecycle bypass**:
A request whose handler takes ownership of the raw writer (HTTPServe); govalin performs no status flush or response finalization for it.
_Avoid_: framework-managed finalization on raw handlers, double-writing a bypassed response

**Bound port**:
The port the server is actually listening on, read from the live listener (`listener.Addr()`), as opposed to the configured port. When the configured port is `0` the OS assigns a free port, so only the bound port is authoritative. Exposed to plugins via `App.Port()`.
_Avoid_: configured-port-as-truth, advertising a port the server is not listening on

**Service advertisement**:
Publishing the server on the local network as a full DNS-SD service instance (instance name + service type in the `.local` domain, with SRV/TXT/PTR records) over mDNS — not merely a resolvable hostname. A registered service is both resolvable and browsable by standard discovery tooling.
_Avoid_: hostname-only A-record publishing, "discovery means just resolving myapp.local"

**Plugin complexity boundary**:
GOVALIN core stays simple, lean, and easy to configure; any added complexity or heavier dependency (including CGo) belongs in a plugin, never core, because plugins are opt-in and not required for the framework to work. Core's dependency cost is paid by every user; a plugin's is paid only by those who import it.
_Avoid_: feature creep into core, mandatory-by-default dependencies, CGo in the core import graph

**App-under-test**:
The user's own constructed `App` — built by their production constructor with their config, plugins, and routes — handed to the test harness as-is. The harness starts, observes, and stops it; it never builds or reconfigures the app on the user's behalf.
_Avoid_: harness-built app, setup-callback-only testing, duplicated app wiring in tests

**Startup-event readiness**:
The test harness knows the server is ready because a server startup event fired — an explicit signal with a happens-before edge to reading the bound port — never by sleeping or polling.
_Avoid_: sleep-based readiness, port polling, pre-allocated free ports

**Status-agnostic body access**:
Test-client helpers that return the response body return it regardless of status code; asserting on status is a separate, explicit act. Tests that exercise error handlers depend on this.
_Avoid_: helpers that auto-fail on non-2xx, status assertions hidden inside body accessors

**Test-scoped failure**:
A test utility failure fails the calling test (`t.Fatalf`/`t.Errorf`), never the process. Process exit in a helper kills the whole test binary and destroys the test report.
_Avoid_: `os.Exit` in test helpers, one failed request aborting the suite

**Actionable runtime warning**:
A non-fatal log emitted when an auxiliary runtime operation fails (e.g. mDNS registration/broadcast) must state what went wrong and what the user can do to fix it — not merely that an operation failed. Misconfiguration, by contrast, is caught early and is fatal.
_Avoid_: "failed to start/configure" with no cause or remedy, silent runtime degradation

## Relationships

- A **Race-clean build** is a required outcome of the **Stability gate**
- A failed **Stability gate** is a **Release blocker**
- **Root-cause-first remediation** addresses shared gate failures before localized refactors
- **Compatibility-preserving first fix** is the default for urgent release blockers
- **Scope-frozen phase one** limits changes to race elimination only
- **Phase-one done criteria** require both race and default test passes with unchanged helper signatures
- A **Policy-enforced gate** ensures race-cleanliness is continuously verified pre-merge
- **Gate-first execution order** resolves blocking quality gates before static-route redesign
- **Canonical static mount URL** is the trailing-slash path, with redirect from non-slash form
- **Permanent canonical redirect** applies to static mount root normalization
- **Query-preserving canonicalization** keeps request query parameters unchanged during static mount redirect
- **Framework-wide redirect integrity** applies query preservation to every framework-generated redirect
- **Raw-query passthrough** defines exact query preservation semantics for framework redirects
- **Pre-merge gate enforcement** requires race/stability checks during pull request validation
- **Dedicated race gate job** separates race signal from default test signal in CI
- **Current-toolchain alignment** keeps quality gates representative of the intended modern runtime baseline
- **Toolchain maintenance cadence** defines how “up to date” is continuously enforced
- **Selective race coverage** keeps race checks strict on baseline without multiplying matrix cost
- **Dual-minor compatibility matrix** provides compatibility signal on 1.25.x and 1.24.x
- **Two-PR rollout** isolates race-gate stabilization from static-routing redesign
- **Scope-locked PR1** constrains initial delivery to test-helper race fix and CI enforcement updates
- A **Validation rule** asserts on an input without changing it; a **Transform step** changes the value and never fails
- **Chain order significance** means a **Transform step** only reaches a **Validation rule** placed after it
- A **Buffered status** becomes visible to the client only after a **Status flush**
- A **Status flush** happens at most once per request: on first body write, or otherwise at end of lifecycle
- **Lifecycle bypass** suppresses **Status flush** — the handler owns response finalization
- The **App-under-test** relies on **Startup-event readiness**, which reads the **Bound port** only after the startup event fires
- **Test-scoped failure** and **Status-agnostic body access** define the public test client's failure semantics: transport errors fail the test, HTTP error statuses do not
- The **Plugin complexity boundary** does not exclude test tooling from the core module: a test package adds no dependency cost to consumers who never import it

## Example dialogue

> **Dev:** "Regular tests pass, but race tests fail. Can we still release?"
> **Maintainer:** "No. A failed stability gate is a release blocker, and race-clean is required."

## Flagged ambiguities

- "tests pass" was used to mean only default test execution and also all quality gates; resolved: release readiness requires passing mandatory stability gates, including race checks.
- CI currently runs default tests but does not run race tests; resolved: race-clean will be enforced in CI as a mandatory gate.
- Existing HTTP->HTTPS plugin redirect currently uses only URL path and drops query parameters; resolved: query-preserving behavior applies framework-wide and this behavior must be aligned.
