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
A request whose handler takes ownership of the raw writer (HTTPServe, or a websocket upgrade that hijacks the connection); govalin performs no status flush or response finalization for it.
_Avoid_: framework-managed finalization on raw handlers, double-writing a bypassed response

**Allocation budget**:
The number of allocations a request shape is allowed to cost, asserted in CI. Measured from what the
code does today, not aimed at; exact rather than statistical, because an allocation count is the same
on every run of a build while wall-clock time on a shared runner is not. Raising one is a decision
taken in the diff that needs it, with the reason in the commit.
_Avoid_: timing thresholds as a blocking gate, nudging the number until the gate passes

**Committed response**:
A response whose header has reached the wire, whoever put it there — a `Call` body sink, an
`http.ServeContent` call, a raw write, third-party middleware. Recorded by the writer itself rather
than declared by the caller, so the framework never has to be told.
_Avoid_: caller-declared "I wrote it" flags, per-call-site bypass as the way to avoid a double write

**Written status**:
The status of a **Committed response**, as recorded on the way out. This — not the **Buffered
status** — is what the access log reports; the buffered value is the fallback for a response that
never reached the wire through govalin.
_Avoid_: logging the intended status, "the status is whatever `Status()` says"

**Streamed body**:
A response body copied straight from a reader onto the connection, never held in memory. Of unknown
length and not seekable, so it is chunked and carries no ranges.
_Avoid_: buffer-then-write for large bodies, "read it all first to know the length"

**Ranged content**:
A response body the client may ask for part of, byte range by byte range, because the content can be
addressed at an offset — by seeking, or by reading at an offset with a known total size. What makes a
large download resumable and a large file seekable.
_Avoid_: whole-body-only downloads, buffering a remote object to make it seekable

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

**Body size limit**:
The maximum request body the server will accept and read into memory, covering JSON/raw bodies and
URL-encoded forms. Enforced on every path that reads a body, never left to net/http's own default.
_Avoid_: per-path limits, "the framework caps it somewhere"

**Multipart size limit**:
The total size ceiling for a multipart request body. Separate from the **Body size limit**, whose
default is sized for JSON payloads and would reject every upload.
_Avoid_: one limit for all body shapes, uncapped uploads

**Multipart memory budget**:
How much of a multipart form is held in memory before parts spill to temporary files. Bounds memory
pressure only — it is not a limit on what the server accepts.
_Avoid_: budget-as-size-cap, zero budget forcing every upload to disk

**Declared-length rejection**:
Refusing a body whose `Content-Length` already exceeds its limit, before reading a byte, rather than
letting the read trip the limiter. Keeps a marginally-oversized request from costing a connection
teardown.
_Avoid_: read-then-discard, limiter-only enforcement

**Actionable runtime warning**:
A non-fatal log emitted when an auxiliary runtime operation fails (e.g. mDNS registration/broadcast) must state what went wrong and what the user can do to fix it — not merely that an operation failed. Misconfiguration, by contrast, is caught early and is fatal.
_Avoid_: "failed to start/configure" with no cause or remedy, silent runtime degradation

**Root-confined static access**:
Every file read below a static mount goes through an `os.Root` opened on the configured directory, so
a name that leaves that directory — including through a symlink — is refused by the OS rather than by
comparing cleaned path strings.
_Avoid_: prefix-comparison containment, cleaned-path-as-proof, stat-then-open resolution

**Trailing-slash directory claim**:
A trailing slash on a static URL is the request claiming the name addresses a directory. The name
handed to the file system never carries one, because `io/fs` refuses it, so the claim is answered
separately: a directory serves its index or its listing, and a name that turns out to hold a file is
redirected to its slashless URL.
_Avoid_: a trailing slash reaching an fs name, one file served at both spellings

**Refused name**:
A name the mounted file system will not resolve, for any reason it gives — a `..` segment, a symlink
leaving the root, a directory it may not read. Answered as a missing file, deliberately
indistinguishable from one, because an answer that varied with the reason tells a prober which names
are there. The bluntness is the point, which is why a name govalin malformed itself must never reach
it: there is nothing downstream that can tell the two apart.
_Avoid_: 500 for a name that escapes the root, the reason a name failed visible in the response

**SPA fallback**:
What a mount in SPA mode answers with when a URL addresses no file it holds: the shell, on the
grounds that the route belongs to the client. A directory is not a file it holds, so the fallback
covers that too and a SPA bundle is never enumerated.
_Avoid_: a generated listing of a SPA bundle, 404 for a client-side route

**Request-controlled value**:
A value the client alone decides — request path, `User-Agent`. Unbounded in length and unconstrained
in content until govalin bounds it.
_Avoid_: "it's just a header", trusting net/http to have filtered it

**Log-safe value**:
A **Request-controlled value** made fit to write to a log sink: length bounded, truncation marked so
a cut value is never read as what the client sent, control characters dropped. The bound is the
substance — no sink provides one; the scrub is defence in depth, since both standard `slog` handlers
already escape control characters.
_Avoid_: unbounded client values in a log line, silent truncation

**Correlation ID**:
The call ID, taken verbatim from an inbound `X-Govalin-Id` when the caller sends one and generated
otherwise. It is deliberately *not* a **Log-safe value**: a caller propagates its own ID so it can
find the request in govalin's log, and appearing verbatim in every record is the point of having one.
_Avoid_: sanitizing the ID for the log, framework-imposed ID formats that break propagation

**Cache validation**:
A client asking whether the copy it already holds is still current, and being answered 304 with no
body when it is. A bandwidth feature, on safe methods only. Distinct from a precondition on an unsafe
method (`If-Match`, 412), which borrows the same header vocabulary to answer a different question —
lost updates, not stale copies.
_Avoid_: "ETag support" as one undivided feature, If-None-Match meaning the same thing on a write as
on a read

**Validator**:
The opaque token naming a version of a resource. Supplied by the caller out of something it already
knows — a row version, an updated_at, a digest — never computed by the framework from a response body
it has already paid to produce.
_Avoid_: post-hoc body hashing, buffering a response in order to validate it

**Derived validator**:
The **Validator** govalin computes for a static asset without being asked: from when the file changed
if the file system knows, from what the file contains if it does not. A file system reporting no
modification time is declaring it has no change signal, so content is the only thing left that
identifies the file.
_Avoid_: size-only tags for embedded assets, mtime tags from a file system with no clock

**Strong validator**:
A **Validator** asserting byte-exact identity, sent unprefixed. The weak form (`W/`) asserts only
equivalence, and a range resume cannot be built on it: the client's `If-Range` is refused and the
whole body is sent again. Every **Derived validator** is strong.
_Avoid_: weak tags on rangeable content, `W/` as a default hedge

**Revalidation short-circuit**:
What a matched **Validator** buys a handler: permission to skip producing a body, never an obligation
to. The 304 is buffered the moment the match is found, so a handler that produces a body anyway still
sends a correct empty 304 — it has wasted only its own work.
_Avoid_: correctness that depends on the caller reading a return value, a body inside a 304

**Freshness**:
How long a client may reuse a response it has stored before it has to ask again, declared by the
server as a lifetime and counted down by the client. The other half of caching from **Cache
validation**: a **Validator** makes a revalidation cheap, only freshness makes it unnecessary.
Binding, not advisory — a lifetime cannot be withdrawn from a cache the server never sees, which is
why a static mount declares none until it is asked to.
_Avoid_: "caching" as one undivided feature, freshness as a stronger validator, a framework-chosen
default lifetime

**Cache scope**:
Which caches may store a response — any of them, only the client that asked (`private`), or none at
all (`no-store`). A separate question from **Freshness**, which says only how long: a response behind
a login is a scope decision and a lifetime decision, answered deliberately rather than inferred from
each other.
_Avoid_: `public` as a synonym for cacheable, a scope read off how long something stays fresh

**Selecting header**:
The request header a response was chosen from, which is what `Vary` names — and only a request header
can be one. A response header put there names nothing a request carries, so the cache keys on the URL
alone and a single stored copy answers everyone. Naming a header the answer does not depend on is the
opposite mistake, and costs storage rather than correctness.
_Avoid_: a response header in `Vary`, a `Vary` sent only on the responses that came out varying

**Implied HEAD**:
A HEAD answered by the route's GET handler, because HEAD is GET without a body. A handler registered
for HEAD replaces it wherever it sits in the route table — that is how a route buys the right to skip
producing a body, since net/http discards the one an implied HEAD writes. OPTIONS gets no
equivalent: its response is not a representation of the resource, and CORS already owns that path.
_Avoid_: 404 on a path that has a GET, an implied HEAD shadowing a registered one, GET-derived OPTIONS

**Optional trailing slash**:
A trailing slash on a registered route pattern makes the URL's trailing slash optional: both
spellings reach the handler, which is the layer that decides which of them is canonical. A doubled
slash is not the route — the matcher decides which URLs are the route's own rather than relying on
the mux in front of it to have cleaned them first.
_Avoid_: a route reachable only at the spelling it was registered with, a doubled slash matching,
routing that assumes upstream path cleanup

## Relationships

- A **Race-clean build** is a required outcome of the **Stability gate**
- A failed **Stability gate** is a **Release blocker**
- **Root-cause-first remediation** addresses shared gate failures before localized refactors
- **Compatibility-preserving first fix** is the default for urgent release blockers
- **Scope-frozen phase one** limits changes to race elimination only
- **Phase-one done criteria** require both race and default test passes with unchanged helper signatures
- A **Policy-enforced gate** ensures race-cleanliness is continuously verified pre-merge
- An **Allocation budget** is a **Stability gate** for cost, enforced in its own job on the baseline
  toolchain for the same reason race detection is: an isolated, trustworthy signal
- Benchmark timings are advisory context for an **Allocation budget**, never the gate themselves
- **Gate-first execution order** resolves blocking quality gates before static-route redesign
- **Canonical static mount URL** is the trailing-slash path, with redirect from non-slash form
- An **Optional trailing slash** is what puts a **Canonical static mount URL** within reach: both
  spellings of the mount root route to it, so the redirect from the non-slash form has somewhere to
  come from
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
- A **Status flush** never touches a **Committed response**; that, not **Lifecycle bypass**, is what
  keeps the framework from writing a status twice
- **Lifecycle bypass** is about ownership of the raw writer, not about avoiding a double write
- The access log records the **Written status** of a **Committed response**, falling back to the
  **Buffered status** only when nothing was committed through govalin
- A **Committed response** counts as a handled request: the lifecycle appends no not-found body to it
- A **Streamed body** and **Ranged content** are the two shapes of a body too large to buffer; the
  content decides which one it can be
- The **App-under-test** relies on **Startup-event readiness**, which reads the **Bound port** only after the startup event fires
- **Test-scoped failure** and **Status-agnostic body access** define the public test client's failure semantics: transport errors fail the test, HTTP error statuses do not
- The **Plugin complexity boundary** does not exclude test tooling from the core module: a test package adds no dependency cost to consumers who never import it
- The **Body size limit** and the **Multipart size limit** bound what the server accepts; the **Multipart memory budget** bounds only where an accepted upload is stored
- **Declared-length rejection** is the first enforcement step for every limit, with the limiter as the fallback for bodies of unknown length
- **Root-confined static access** replaces path-string containment: the static mount is a resource the OS confines, not a prefix the framework checks
- A **Trailing-slash directory claim** carries **Canonical static mount URL** below the mount root: one spelling serves the resource, the other redirects to it
- A **Refused name** is where **Root-confined static access** stops being visible: what the OS refused is not reported back, only that nothing was found
- A **SPA fallback** takes precedence over a generated listing, but never over a **Trailing-slash directory claim**: a spelling that disagrees with the file system is redirected in either mode
- A **Request-controlled value** reaching a log becomes a **Log-safe value** first; there is no path from a header to a log sink that skips it
- The **Correlation ID** is the deliberate exception: client-chosen and logged verbatim, because propagating it is the feature
- **Cache validation** is answered on safe methods only; the same header on an unsafe method is a precondition, and out of scope
- A **Derived validator** exists only for static assets; every other response's **Validator** comes from the caller
- A **Revalidation short-circuit** buffers 304 like any other **Buffered status**, so **Status flush** and **Committed response** govern it unchanged
- A 304 carries no representation, so the **Status flush** drops the headers that describe one
- A **Strong validator** is what keeps **Ranged content** resumable across a revalidation
- **Freshness** and **Cache validation** are the two halves of one feature, not alternatives: the
  first removes the request, the second answers it cheaply when it comes
- A **Derived validator** is on by default and **Freshness** never is, and the asymmetry is the point:
  an advisory validator costs a client nothing to ignore, a lifetime binds it until the clock runs out
- A 304 carries the **Freshness** of the response it stands in for, so a **Revalidation
  short-circuit** renews the stored copy rather than leaving the client asking on every reuse
- A **SPA fallback** is the one response govalin gives a **Freshness** of its own without being
  asked, and it is the absence of one: the shell names the fingerprinted assets, so **Freshness** on
  a SPA mount is for what the shell points at and never for the shell
- A shell that is never fresh is affordable because its **Derived validator** is: never stale costs a
  304, not a bundle
- An **Implied HEAD** is what lets a static mount answer a cache probe: the **Derived validator** and
  the length are already on the response its GET handler produces
- An **Implied HEAD** is logged as the HEAD it is; that a GET handler answered it is routing, not
  something the client did
- A **Selecting header** is what a cache keys on in addition to the URL, so it is declared by every
  response the plugin reading it produced — including the one it declined to add CORS headers to
- **Freshness** decides how long a missing **Selecting header** goes on answering the wrong request:
  a stored response with a lifetime is reused rather than revalidated

## Example dialogue

> **Dev:** "Regular tests pass, but race tests fail. Can we still release?"
> **Maintainer:** "No. A failed stability gate is a release blocker, and race-clean is required."

## Flagged ambiguities

- "tests pass" was used to mean only default test execution and also all quality gates; resolved: release readiness requires passing mandatory stability gates, including race checks.
- CI currently runs default tests but does not run race tests; resolved: race-clean will be enforced in CI as a mandatory gate.
- Existing HTTP->HTTPS plugin redirect currently uses only URL path and drops query parameters; resolved: query-preserving behavior applies framework-wide and this behavior must be aligned.
- "ETag support" was used to mean both cache validation and optimistic concurrency control; resolved: this work is cache validation only, and a precondition on an unsafe method is a separate feature that would need its own name.
- "caching" was used to mean both freshness and cache validation; resolved: they are the two halves of caching, and a request that never happens is the one freshness is for.
- `Vary` was read as naming the headers a response carries; resolved: it names the request headers the response was selected from, and a response header there is a cache key of nothing.
