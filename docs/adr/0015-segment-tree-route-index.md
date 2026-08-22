# A segment tree that keeps registration order

## Status

accepted

## Context and decision

Matching a URL asked every registered route whether it matched. Each route held a compiled regexp,
so a 200 route table ran 200 regexp executions per request: 63% of the request's time, plus the
`sync.Pool` traffic of taking a regexp machine out and putting it back 200 times. What a route cost
depended on where its author had happened to register it — 331ns at the top of the table, 4611ns at
the bottom.

Routes are now indexed by their segments. A literal segment is a key, a parameter segment is the one
child that takes any piece, and a route sits at the node its last segment reaches; the walk descends
the URL's pieces rather than the table. A match costs what the path is, not what the table holds.

### Registration order is the constraint, not a detail to trade away

Every other Go router of this shape resolves overlapping routes by specificity: a static segment
beats a parameter, a parameter beats a wildcard. Govalin's **Registration-order match** says the
opposite — the first route registered whose pattern matches is the one that serves, so a literal
registered after a parameter route that also matches is unreachable *by design*. That rule is the
author's tool for deciding which of two overlapping patterns wins, and an index that quietly replaced
it with specificity would change which handler answers an existing app's requests.

So the tree preserves it. Each node carries the lowest registration order of any route beneath it,
and the walk carries the best order it has found; a subtree that cannot beat it is never entered.
Parameter and literal children are both explored, because either may hold the earlier route.

### The tree indexes segments, so what is not a segment stays in a list

A wildcard spans separators, which is not a shape a per-segment tree can key on. Rather than
complicate the tree with a construct that appears a handful of times in an app — a static mount, a
CORS `Before`, an HTTPS redirect — routes holding a wildcard stay in an ordered list scanned after
the walk, where the lowest order still wins. An app that registers many wildcard routes gets the
linear scan it always had, now over a segment matcher rather than a regexp.

The same reasoning removed the regexps: the patterns the parser built were only ever literals,
`([^/]+?)` and `.+?` joined by slashes, which is a walk over slash-separated pieces expressed in a
language that cannot know that. `PathMatcher` walks its segments directly and captures parameters on
the same pass instead of running the pattern a second time for submatches.

## Considered options

- **A segment tree ordered by registration (chosen)** — see above.
- **A specificity tree, as httprouter and gin use** — simpler, no order bookkeeping, and faster still
  because only one branch is ever explored. Rejected because it silently changes which handler serves
  a URL in any app with overlapping routes.
- **Keeping the linear scan and only replacing the regexp** — this was measured: it took the bottom
  of a 200 route table from 4350ns to 1950ns for a fraction of the code. Rejected as the endpoint
  rather than the step it became, because the cost still scales with the table.
- **Making the tree the whole answer and confirming nothing** — the segments settle everything about
  a match except how the path spells its end. Rather than re-run the full match to check, the matcher
  reports whether its trailing slash is optional and the walk asks at the node.
- **A slice of literal children instead of a map** — measured 5ns faster per request on small tables
  and 40ns slower on a node with 40 children. Rejected: the map is the one that does not reintroduce
  a scan proportional to the route table, which is the thing being fixed.

## Consequences

- A route table's shape no longer changes what a request costs: 120ns at the top of a 200 route
  table and 271ns at the bottom, against 331ns and 4611ns before, and a one route app is unchanged.
- Every route now exists twice, in `pathHandlers` and in the index. `getOrCreatePathHandlerByPath` is
  the single place a route is created, so it is the single place the two can diverge.
- The index is only correct if it agrees with a scan of the table. That is a property rather than a
  case list, so it is tested as one, against 20k generated route tables.
- Before and after handlers still scan, over lists holding only the routes that have one. They match
  every route they hit rather than the first, so the index's answer is not the question they ask.
