// Package typedfields is not built: every call below is a compile error that the
// typed body validators must keep rejecting. TestTypedFieldRulesDoNotCompile
// builds it and asserts each error is still reported.
package typedfields

import "github.com/pkkummermo/govalin"

type user struct {
	Name string
	Age  int
}

// A string rule on an int field.
func stringRuleOnIntField(call *govalin.Call, u *user) error {
	return call.ValidatedBody(u).
		IntField("age", func(u user) int { return u.Age }).MinLength(2).
		Get()
}

// An accessor returning a type the field validator does not validate.
func accessorOfWrongType(call *govalin.Call, u *user) error {
	return call.ValidatedBody(u).
		StringField("age", func(u user) int { return u.Age }).Required().
		Get()
}

// A field name that does not exist on the body.
func misspelledField(call *govalin.Call, u *user) error {
	return call.ValidatedBody(u).
		StringField("name", func(u user) string { return u.Nmae }).Required().
		Get()
}
