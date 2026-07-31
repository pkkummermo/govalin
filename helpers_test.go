package govalin_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/http/contenttypes"
	"github.com/pkkummermo/govalin/internal/http/headers"
)

// bodyTooLargeResponse is the response govalin writes when a request body
// exceeds the configured max body read size.
const bodyTooLargeResponse = `{"title":"Payload too large","status":413,` +
	`"type":"https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/413",` +
	`"details":[{"field":"body","reason":"Request body exceeds the maximum allowed size"}]}`

// newTestApp creates an app with logs disabled to keep test output clean,
// optionally applying further config on top.
func newTestApp(configFuncs ...govalin.ConfigFunc) *govalin.App {
	return govalin.New(func(config *govalin.Config) {
		config.EnableAccessLog(false)
		config.EnableStartupLog(false)

		for _, configFunc := range configFuncs {
			configFunc(config)
		}
	})
}

// readBody reads and closes a response body, failing the test on error.
func readBody(t *testing.T, response *http.Response) string {
	t.Helper()

	defer func() {
		_ = response.Body.Close()
	}()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return string(data)
}

// urlEncodedRequest builds a POST carrying an application/x-www-form-urlencoded body.
func urlEncodedRequest(t *testing.T, path string, values url.Values) *http.Request {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	request.Header.Set(headers.ContentType, contenttypes.ApplicationFormURLEncoded)

	return request
}

// multipartRequest builds a POST carrying a single uploaded file.
func multipartRequest(t *testing.T, path string, field string, content string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	filePart, err := writer.CreateFormFile(field, "upload.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err = filePart.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}
	if err = writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, path, &body)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	request.Header.Set(headers.ContentType, writer.FormDataContentType())

	return request
}
