package govalin_test

import (
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/stretchr/testify/assert"
)

func TestServerMaxBodyReadSizeConfig(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
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

		return newApp
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Post(http.Host+"/bodysize", `"aaa"`)
		responseBody, _ := response.ToString()
		assert.Equal(
			t,
			`{"title":"Server error","status":500,"type":"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/500"}`,
			responseBody,
			"should trigger error upon max size",
		)
	})

	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
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

		return newApp
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Post(http.Host+"/bodysize", `"aaaaaaaa"`)
		responseBody, _ := response.ToString()
		assert.Equal(
			t,
			`{"title":"Server error","status":500,"type":"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/500"}`,
			responseBody,
			"should trigger error upon more than max size",
		)
	})

	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
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

		return newApp
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Post(http.Host+"/bodysize", `"aa"`)
		responseBody, _ := response.ToString()
		assert.Equal(
			t,
			`"aa"`,
			responseBody,
			"should not trigger error upon max size",
		)
	})
}
