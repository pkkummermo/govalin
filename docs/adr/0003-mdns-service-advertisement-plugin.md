# mDNS service advertisement as an opt-in plugin

## Status

accepted

## Context and decision

GOVALIN apps should be able to expose themselves on the local network for zero-config discovery
(open `myapp.local` in a browser, show up in Bonjour/`avahi-browse` lists) without the developer
hand-wiring any of it. We add this as **mDNS service advertisement only** — publishing the running
server over multicast DNS. Service *discovery/browsing* (acting as a client to find other services)
and any auto-update or binary-distribution machinery are explicitly **out of scope**: GOVALIN is a
lean HTTP framework, and discovery enables a distribution story without GOVALIN owning it.

The feature ships as an **opt-in plugin** (`plugins/mdns`), not core config. mDNS needs a real
dependency (`brutella/dnssd`, which pulls `miekg/dns`); per the **plugin complexity boundary**, that
cost must be paid only by users who import it, never by every core user. The existing plugin
lifecycle is sufficient — the plugin advertises in `Apply` (the listener is already bound by then)
and withdraws (goodbye packets) via an `onServerShutdown` hook registered in `OnInit`. No new
`Plugin` interface method is needed.

The plugin advertises a **full DNS-SD `_http._tcp` service** (resolvable *and* browsable), not a bare
hostname — the service registration is the superset that delivers both. Zero-config defaults:
instance name = executable base name, service type = `_http._tcp`, hostname = `<os-hostname>.local`,
all multicast-capable interfaces, TXT = `{path:/}`. Every default is overridable. Name collisions on
the network are resolved by DNS-SD probing (`(2)`/`(3)` suffixing) provided by the library — a key
reason for the library choice.

One enabling **core** change is required and ships as its **own separate PR ahead of the plugin**: a
new `App.Port() uint16` accessor sourced from `listener.Addr()`. A plugin currently cannot read the
bound port (the field is unexported, no accessor), and `Start()` discards the real port when the
configured port is `0` (OS-assigned). Sourcing from the live listener fixes that discard bug and is
generally useful beyond mDNS.

## Considered options

- **Plugin (chosen)** vs **core `Config.EnableMDNS()`** — core would drag the mDNS/`miekg/dns`
  dependency into every GOVALIN build, violating the lean-core principle. Plugin isolates it.
- **brutella/dnssd (chosen)** vs **grandcat/zeroconf** vs **hashicorp/mdns** — brutella is the only
  pure-Go option that implements probing-based conflict resolution and goodbye packets and is still
  maintained. grandcat's conflict resolution is incomplete (undercutting the chosen rename-on-collision
  behavior); hashicorp/mdns is effectively unmaintained with weak IPv6.
- **Full DNS-SD service (chosen)** vs **hostname-only A record** — hostname-only resolves
  `myapp.local` but stays invisible to discovery tooling, defeating the purpose.

## Consequences

- **Failure asymmetry, deliberate.** Misconfiguration (bad service type, oversized TXT) is caught in
  `OnInit` and is **fatal**, consistent with how the cors plugin fatals on bad config. Runtime
  registration/broadcast failure (no multicast route, blocked UDP 5353, container without host
  networking) is **non-fatal** — the HTTP server is healthy and keeps serving. A future reader seeing
  mDNS failure merely warn while a duplicate-route registration calls `os.Exit(1)` should know this
  split is intentional: config errors fatal, runtime-environment errors warn.
- Runtime warnings must be **actionable** — state the cause and the fix (e.g. the Docker
  host-networking limitation), never just "failed to start mDNS".
- Multicast typically does not cross the default Docker bridge; this is documented and surfaced as an
  actionable warning rather than worked around in code.
- `App.Port()` becomes public API (additive, harder to reverse) and reads the listener rather than the
  configured value — intentional, to report the truly-bound port.
