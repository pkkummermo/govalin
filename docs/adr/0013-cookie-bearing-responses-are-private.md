# A response that sets a cookie is the asking client's alone

## Status

accepted

## Context and decision

With sessions enabled, every request without a valid session cookie mints one and writes `Set-Cookie`
on that response, before any handler runs. Freshness is opt-in (ADR 0010), so until now that response
said nothing at all about who may store it — and a shared cache is free to store a 200 GET that says
nothing and reuse it under heuristic freshness. The copy it hands the next visitor carries the first
visitor's session cookie, so the two share a session for as long as the copy is reused.

The answer is a **Cache scope**: setting a cookie through `Call` narrows the response to the client
that asked for it, keeping whatever freshness is declared. The session cookie is one caller of that,
not a case of its own — an app setting a cookie of its own has the same response to protect.

The two orders are handled at the two ends, because either alone leaves the other open. A cookie set
over a declared lifetime narrows it (`max-age=3600` becomes `private, max-age=3600`), and a lifetime
declared over a response already setting a cookie is written narrowed. Without the second half the
first loses exactly where it matters: a static mount declares a lifetime of its own, last-call-wins
replaces the scope with it, and the longest-lived, most-shared response an app has is the one carrying
a stranger's `Set-Cookie`. The lifetime is still the route's to choose — what it no longer chooses is
that a response setting a cookie may be held for everyone.

`Vary` cannot answer this, which is why it is not the fix ADR 0012 left behind. The request that mints
a session is the one carrying no cookie to key on, so the response is stored under the empty-cookie
key — the key every other first-time visitor matches. A cache key says which stored response answers a
request; it never says who may hold one.

## Considered options

- **Narrowing at the cookie write and at every freshness declaration made over it (chosen)** — see
  above.
- **Narrowing at the cookie write alone** — correct until the first `config.CacheFor(time.Hour)` on a
  static mount, which is both the response most likely to be shared and the one held longest.
- **Narrowing at the freshness declaration alone** — the order this change was first written in, and
  it holds only for a cookie already on the response. A handler that declares a lifetime and then sets
  a cookie is the same leak with the two lines swapped.
- **`no-store` on a session-minting response** — heavier than the problem. The client's own copy is
  fine and useful; it is the copy held in between that is wrong, and `private` says that exactly.
- **`Vary: Cookie`** — answers a different question (above), and for returning clients it splits a
  shared cache one copy per session, which is not caching.
- **Narrowing at end of lifecycle, where both the cookie and the final `Cache-Control` are visible** —
  there is no such point: `http.ServeContent` commits its own header from inside the handler, so a
  static file would slip past it. Declaration time is the one place every path passes through.
- **Leaving `CacheFor` purely last-call-wins** — that promise was about a route overriding a group's
  default lifetime. A route declaring a lifetime never learns a cookie is on the response, so honouring
  it here would be honouring a choice nobody made.

## Consequences

- A response govalin mints a session on carries `Cache-Control: private`, and no shared cache holds one.
  For an app with sessions enabled that is every first visit, which is the cost this buys deliberately.
- A lifetime declared on such a response comes out `private, max-age=…`: the client still reuses its own
  copy for the full duration, and nothing in between stores one.
- An app's own cookie narrows its response the same way. That is a change beyond sessions, and the
  right one: the response carrying it is as wrong to hand the next client.
- The narrowing follows the cookie through `Call`, whether it is set through `Cookie` or written as a
  `Set-Cookie` header. Only the raw writer bypasses it, that being the escape hatch for a response the
  framework does not manage.
- `CachePrivateFor` and `NoStore` are untouched: both already answer the question being asked.
- Sessions are opt-in, so an app that never enables them sees no change unless it sets cookies itself.
