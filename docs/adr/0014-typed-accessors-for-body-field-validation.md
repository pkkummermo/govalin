# A body field is named by an accessor, not by a string

## Status

accepted

## Context and decision

`ValidateField("Name").MinLength(2)` resolved the field with
`reflect.ValueOf(data).Elem().FieldByName("Name")` at request time. Three mistakes therefore
survived compilation and shipped: a misspelled field name, a rule applied to a field of another
type, and a rule applied to a body that has no such field at all. The first answered every request
with `"Field does not exist"`; the other two were worse, because a `Kind()` check that does not
match simply falls through and the rule silently passes.

A field is now named by an accessor the caller writes, `StringField("name", func(u User) string
{ return u.Name })`, and the compiler answers all three: `u.Nmae` does not exist, an `int` accessor
is not a `func(User) string`, and `MinLength` is not on `IntFieldValidator`.
`testdata/typedfields` is built by a test that asserts each of those three is still rejected.

Go 1.27 generic methods are what make this expressible: `ValidatedBody` infers `T` from the target
it is given, so an accessor is checked against the body type without the caller naming that type.

The name stays an explicit first argument. It is what
`validation.NewParameterErrorDetail(field, reason)` reports, and an accessor cannot yield it
without the reflection this change removes. That turns out to be the fix the API needed anyway:
reflection reported the **Go** name `"Name"` to a client that sent `"name"`, and the caller now
passes the name the client used.

## Considered options

- **One entry point per field type, rules as methods (chosen)** — see above.
- **A single `Field[F any](name string, acc func(T) F, rules ...Rule[F])`** — type-safe and one
  entry point instead of two, but Go cannot declare `MinLength` on `Field[T, string]` only, so the
  sugar would have to re-check `F` at runtime. That is reflection's failure mode in a new costume.
  It also forces the caller to import `validation` for `Required()` and `MinLength(2)`, and not
  needing that import is one of the things the curried API buys.
- **Keeping `ValidateField` alongside the typed form** — two ways to say the same thing, one of
  which is the one being removed for being unsafe.
- **Struct tags (`validate:"required,min=2"`)** — moves the rules onto the type, so they cannot
  differ per route, and the tag is a string the compiler does not read either.

## Consequences

- Breaking, for v0.4.0: `ValidateField`, `BodyFieldValidator` and its six rule methods are gone.
  A chain migrates field by field, and the `.Get()` that used to close each field is dropped —
  `StringFieldValidator` and `IntFieldValidator` embed `*BodyValidator[T]`, so the next
  `StringField`, `IntField` or the final `Get` is promoted straight through.
- `MinLength`/`MaxLength` on a body field count runes. The reflection path counted bytes
  (`len(field.String())`), so a name of four accented characters failed a minimum of five; the
  query parameter validator has always counted runes and the two now agree.
- A field-level `Custom` still receives the whole body, so a check may depend on other fields while
  reporting under this field's name. Because it shadows the promoted body-level `Custom`, a
  `Custom` written after a field call is attributed to that field — a body-attributed one must come
  before the first field.
- Only string and int fields have entry points, which is what the reflection path validated too.
  A `FloatField` is a few lines the day a caller needs one; anything else goes through `Custom`.
- `reflect` is gone from `validation.go`.
- Each rule is defined once, in `internal/validation`, and both the parameter validators and the body
  field validators spend it. The predicates and messages had been written out three times, so the
  reflection path counting bytes where the query parameter path counted runes was a divergence
  waiting to happen rather than an oversight.
