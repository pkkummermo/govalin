// Package wstarget is not built: every call below is a message target the generic
// WsMessage.As cannot infer T from. TestUninferableMessageTargetDoesNotCompile
// builds it and asserts each compile error is still reported.
package wstarget

import "github.com/pkkummermo/govalin"

type greeting struct {
	Name string
}

// A target passed by value.
func valueTarget(message *govalin.WsMessage, g greeting) error {
	return message.As(g)
}

// A pointer behind an interface, which *T cannot be inferred from.
func interfaceTarget(message *govalin.WsMessage, g *greeting) error {
	var target any = g
	return message.As(target)
}
