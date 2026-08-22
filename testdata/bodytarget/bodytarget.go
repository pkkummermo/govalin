// Package bodytarget is not built: every call below is a body target the generic
// BodyAs cannot infer T from. TestUninferableBodyTargetDoesNotCompile builds it
// and asserts each compile error is still reported.
package bodytarget

import "github.com/pkkummermo/govalin"

type user struct {
	Name string
}

// A target passed by value.
func valueTarget(call *govalin.Call, u user) error {
	return call.BodyAs(u)
}

// A pointer behind an interface, which *T cannot be inferred from.
func interfaceTarget(call *govalin.Call, u *user) error {
	var target any = u
	return call.BodyAs(target)
}
