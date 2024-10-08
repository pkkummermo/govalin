package govalintesting

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/ddliu/go-httpclient"
	"github.com/gorilla/websocket"
	"github.com/pkkummermo/govalin"
)

const generalStartupTimeoutInMS = 1

type (
	TestFunc func(app *govalin.App) *govalin.App
	ExecFunc func(http GovalinHTTP)
)

// GovalinHTTP is a simple wrapper with utility methods to simplify testing.
type GovalinHTTP struct {
	http httpclient.HttpClient
	Host string
}

func (govalinHttp *GovalinHTTP) Head(path string) string {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Head(url)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to HEAD %s. %v", url, err))
		os.Exit(1)
	}

	data, err := response.ToString()
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed decode HEAD response as string for %s. %v", url, err))
		os.Exit(1)
	}

	return data
}

func (govalinHttp *GovalinHTTP) HeadResponse(path string) *httpclient.Response {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Head(url)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to HEAD %s. %v", url, err))
		os.Exit(1)
	}

	return response
}

func (govalinHttp *GovalinHTTP) Get(path string, params ...any) string {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Get(url, params...)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to GET %s. %v", url, err))
		os.Exit(1)
	}

	data, err := response.ToString()
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed decode GET response as string for %s. %v", url, err))
		os.Exit(1)
	}

	return data
}

func (govalinHttp *GovalinHTTP) GetResponse(path string, params ...any) *httpclient.Response {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Get(url, params...)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to GET %s. %v", url, err))
		os.Exit(1)
	}

	return response
}

func (govalinHttp *GovalinHTTP) Post(path string, postData any) string {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Post(url, postData)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to POST %s. %v", url, err))
		os.Exit(1)
	}

	data, err := response.ToString()
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed decode POST response as string for %s. %v", url, err))
		os.Exit(1)
	}

	return data
}

func (govalinHttp *GovalinHTTP) PostResponse(path string, postData any) *httpclient.Response {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Post(url, postData)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to POST %s. %v", url, err))
		os.Exit(1)
	}

	return response
}

func (govalinHttp *GovalinHTTP) Put(path string, putData io.Reader) string {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Put(url, putData)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to PUT %s. %v", url, err))
		os.Exit(1)
	}

	data, err := response.ToString()
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed decode PUT response as string for %s. %v", url, err))
		os.Exit(1)
	}

	return data
}

func (govalinHttp *GovalinHTTP) PutResponse(path string, putData io.Reader) *httpclient.Response {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Put(url, putData)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to PUT %s. %v", url, err))
		os.Exit(1)
	}

	return response
}

func (govalinHttp *GovalinHTTP) Patch(path string, patchData map[string]string) string {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Patch(url, patchData)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to PATCH %s. %v", url, err))
		os.Exit(1)
	}

	data, err := response.ToString()
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed decode PATCH response as string for %s. %v", url, err))
		os.Exit(1)
	}

	return data
}

func (govalinHttp *GovalinHTTP) PatchResponse(path string, patchData map[string]string) *httpclient.Response {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Patch(url, patchData)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to PATCH %s. %v", url, err))
		os.Exit(1)
	}

	return response
}

func (govalinHttp *GovalinHTTP) Options(path string, optionsData ...map[string]string) string {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Options(url, optionsData...)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to OPTIONS %s. %v", url, err))
		os.Exit(1)
	}

	data, err := response.ToString()
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed decode OPTIONS response as string for %s. %v", url, err))
		os.Exit(1)
	}

	return data
}

func (govalinHttp *GovalinHTTP) OptionResponse(path string, optionsData ...map[string]string) *httpclient.Response {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Options(url, optionsData...)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to OPTIONS %s. %v", url, err))
		os.Exit(1)
	}

	return response
}

func (govalinHttp *GovalinHTTP) Delete(path string, deleteData ...any) string {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Delete(url, deleteData...)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to DELETE %s. %v", url, err))
		os.Exit(1)
	}

	data, err := response.ToString()
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed decode DELETE response as string for %s. %v", url, err))
		os.Exit(1)
	}

	return data
}

func (govalinHttp *GovalinHTTP) DeleteResponse(path string, deleteData ...any) *httpclient.Response {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Delete(url, deleteData...)
	if err != nil {
		slog.Error(fmt.Sprintf("HTTP: Failed to GET %s. %v", url, err))
		os.Exit(1)
	}

	return response
}

func (govalinHttp *GovalinHTTP) Websocket(path string) *websocket.Conn {
	url := strings.ReplaceAll(govalinHttp.Host, "http", "ws") + path
	websocket, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		slog.Error(fmt.Sprintf("WS: Failed connect to websocket on url '%s'. %v", url, err))
		os.Exit(1)
	}

	return websocket
}

func (govalinHttp *GovalinHTTP) Raw() *httpclient.HttpClient {
	return &govalinHttp.http
}

func HTTPTestUtil(serverF TestFunc, testFunc ExecFunc) {
	port, err := freePort()
	if err != nil {
		slog.Error(fmt.Sprintf("Could not find free port. %v", err))
		os.Exit(1)
	}
	startupChann := make(chan (bool))
	testInstance := govalin.New(func(config *govalin.Config) {
		config.EnableAccessLog(false)
		config.EnableStartupLog(false)
		config.Events(func(serverEvents *govalin.ServerEvents) {
			serverEvents.AddOnServerStartup(func() {
				startupChann <- true
			})
		})
	})

	server := serverF(testInstance)

	go func() {
		err = server.Start(port)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to start test server. %v", err))
			os.Exit(1)
		}
	}()

	// If the server is the test instance, we wait for the configured startup event.
	if server == testInstance {
		select {
		case <-startupChann:
		case <-time.After(generalStartupTimeoutInMS * time.Millisecond):
			slog.Error(fmt.Sprintf("Server startup timed out after %d ms", generalStartupTimeoutInMS))
			os.Exit(1)
		}
	} else {
		// If the server is not the test instance, we wait for the default startup timeout.
		time.Sleep(generalStartupTimeoutInMS * time.Millisecond)
	}

	testFunc(GovalinHTTP{http: *httpclient.Defaults(
		httpclient.Map{
			httpclient.OPT_USERAGENT: "govalin-testing",
		},
	), Host: fmt.Sprintf("http://localhost:%d", port)})

	err = server.Shutdown()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to shutdown test server. %v", err))
		os.Exit(1)
	}
}

// Get free port to be used for testing purposes.
func freePort() (uint16, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint16(l.Addr().(*net.TCPAddr).Port), nil //nolint: gosec // Ports dont go higher than 65535
}
