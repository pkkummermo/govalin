package govalin_test

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
)

func TestServerMaxBodyReadSizeConfig(t *testing.T) {
	newApp := newTestApp(func(config *govalin.Config) {
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
		assert.Equal(t, http.StatusRequestEntityTooLarge, response.StatusCode)
		assert.Equal(
			t,
			bodyTooLargeResponse,
			responseBody,
			"should trigger error upon max size",
		)
	})

	newApp = newTestApp(func(config *govalin.Config) {
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
		assert.Equal(t, http.StatusRequestEntityTooLarge, response.StatusCode)
		assert.Equal(
			t,
			bodyTooLargeResponse,
			responseBody,
			"should trigger error upon more than max size",
		)
	})

	newApp = newTestApp(func(config *govalin.Config) {
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

func TestServerMaxBodyReadSizeErrorIsReplayedOnReread(t *testing.T) {
	newApp := newTestApp(func(config *govalin.Config) {
		config.ServerMaxBodyReadSize(4)
	})

	newApp.Post("/bodysize", func(call *govalin.Call) {
		var body string

		assert.Error(t, call.BodyAs(&body), "first read should report the oversized body")

		// The body is consumed, so a second read must replay the failure rather
		// than hand back the empty cache as a success.
		err := call.BodyAs(&body)

		if err != nil {
			call.Error(err)
		} else {
			call.JSON(body)
		}
	})

	govalintest.Test(t, newApp, func(client *govalintest.Client) {
		response := client.PostResponse("/bodysize", `"aaaaaaaa"`)
		assert.Equal(t, http.StatusRequestEntityTooLarge, response.StatusCode)
		assert.Equal(t, bodyTooLargeResponse, readBody(t, response))
	})
}

func TestServerMaxBodyReadSizeWithoutContentLength(t *testing.T) {
	var observedLength atomic.Int64

	newApp := newTestApp(func(config *govalin.Config) {
		config.ServerMaxBodyReadSize(4)
		// Without a declared length there is nothing to pre-check, so the read
		// trips MaxBytesReader and net/http lingers 500ms on the connection,
		// outlasting the default 200ms shutdown timeout.
		config.ServerShutdownTimeout(2000)
	})

	newApp.Post("/bodysize", func(call *govalin.Call) {
		var body string

		observedLength.Store(call.Raw.Req.ContentLength)

		err := call.BodyAs(&body)

		if err != nil {
			call.Error(err)
		} else {
			call.JSON(body)
		}
	})

	govalintest.Test(t, newApp, func(client *govalintest.Client) {
		chunkedRequest := func(body string) *http.Request {
			req, err := http.NewRequest(http.MethodPost, "/bodysize", strings.NewReader(body))
			assert.NoError(t, err)
			req.ContentLength = -1 // Force chunked encoding.
			return req
		}

		response := client.Do(chunkedRequest(`"aa"`))
		// Guards the premise: an unknown length is what routes readBody down its
		// growing-buffer branch.
		assert.Equal(t, int64(-1), observedLength.Load(), "server should see an unknown body length")
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, `"aa"`, readBody(t, response), "chunked body within the limit should be read")

		response = client.Do(chunkedRequest(`"aaaaaaaa"`))
		assert.Equal(t, int64(-1), observedLength.Load(), "server should see an unknown body length")
		assert.Equal(t, http.StatusRequestEntityTooLarge, response.StatusCode)
		assert.Equal(
			t,
			bodyTooLargeResponse,
			readBody(t, response),
			"chunked body over the limit should be rejected",
		)
	})
}
