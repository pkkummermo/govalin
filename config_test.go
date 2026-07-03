package govalin_test

import (
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
)

func TestServerMaxBodyReadSizeConfig(t *testing.T) {
	newApp := govalin.New(func(config *govalin.Config) {
		config.ServerMaxBodyReadSize(4)
	})

	newApp.Post("/bodysize", func(call *govalin.Call) {
		var body string

		err := call.BodyAs(&body)

		if err != nil {
			call.Error(err)
		} else {
			call.JSON(body)
		}
	})

	govalintest.Test(t, newApp, func(client *govalintest.Client) {
		response := client.PostResponse("/bodysize", `"aaa"`)
		responseBody := readBody(t, response)
		assert.Equal(
			t,
			`{"title":"Server error","status":500,"type":"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/500"}`,
			responseBody,
			"should trigger error upon max size",
		)
	})

	newApp = govalin.New(func(config *govalin.Config) {
		config.ServerMaxBodyReadSize(4)
	})

	newApp.Post("/bodysize", func(call *govalin.Call) {
		var body string

		err := call.BodyAs(&body)

		if err != nil {
			call.Error(err)
		} else {
			call.JSON(body)
		}
	})

	govalintest.Test(t, newApp, func(client *govalintest.Client) {
		response := client.PostResponse("/bodysize", `"aaaaaaaa"`)
		responseBody := readBody(t, response)
		assert.Equal(
			t,
			`{"title":"Server error","status":500,"type":"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/500"}`,
			responseBody,
			"should trigger error upon more than max size",
		)
	})

	newApp = govalin.New(func(config *govalin.Config) {
		config.ServerMaxBodyReadSize(4)
	})

	newApp.Post("/bodysize", func(call *govalin.Call) {
		var body string

		err := call.BodyAs(&body)

		if err != nil {
			call.Error(err)
		} else {
			call.JSON(body)
		}
	})

	govalintest.Test(t, newApp, func(client *govalintest.Client) {
		response := client.PostResponse("/bodysize", `"aa"`)
		responseBody := readBody(t, response)
		assert.Equal(
			t,
			`"aa"`,
			responseBody,
			"should not trigger error upon max size",
		)
	})
}
