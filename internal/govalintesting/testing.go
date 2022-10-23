package govalintesting

import (
	"fmt"
	"net"
	"time"

	"github.com/ddliu/go-httpclient"
	"github.com/pkkummermo/govalin/pkg/govalin"
)

const startupInMS = 1

type TestFunc func(app *govalin.App) *govalin.App
type ExecFunc func(http GovalinHTTP)

var httpClient *httpclient.HttpClient = httpclient.Defaults(httpclient.Map{
	httpclient.OPT_USERAGENT: "govalin-testing",
})

// GovalinHTTP is a simple wrapper with utility methods to simplify testing.
type GovalinHTTP struct {
	http httpclient.HttpClient
	Host string
}

func (govalinHttp *GovalinHTTP) Get(path string, params ...interface{}) string {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Get(url, params...)
	if err != nil {
		log.Fatalf("HTTP: Failed to GET %s. %v", url, err)
	}

	data, err := response.ToString()
	if err != nil {
		log.Fatalf("HTTP: Failed decode GET response as string for %s. %v", url, err)
	}

	return data
}

func (govalinHttp *GovalinHTTP) GetResponse(path string, params ...interface{}) *httpclient.Response {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Get(url, params...)
	if err != nil {
		log.Fatalf("HTTP: Failed to GET %s. %v", url, err)
	}

	return response
}

func (govalinHttp *GovalinHTTP) Post(path string, postData any) string {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Post(url, postData)
	if err != nil {
		log.Fatalf("HTTP: Failed to POST %s. %v", url, err)
	}

	data, err := response.ToString()
	if err != nil {
		log.Fatalf("HTTP: Failed decode POST response as string for %s. %v", url, err)
	}

	return data
}

func (govalinHttp *GovalinHTTP) PostResponse(path string, postData interface{}) *httpclient.Response {
	url := govalinHttp.Host + path
	response, err := govalinHttp.http.Post(url, postData)
	if err != nil {
		log.Fatalf("HTTP: Failed to GET %s. %v", url, err)
	}

	return response
}

func (govalinHttp *GovalinHTTP) Raw() *httpclient.HttpClient {
	return &govalinHttp.http
}

func HTTPTestUtil(serverF TestFunc, testFunc ExecFunc) {
	port, err := freePort()
	if err != nil {
		log.Fatalf("Could not find free port. %v", err)
	}
	testInstance, err := govalin.New()
	if err != nil {
		log.Fatalf("Failed to create test server. %v", err)
	}
	server := serverF(testInstance)

	go func() {
		err = server.Start(port)
		if err != nil {
			log.Errorf("Failed to start test server. %v", err)
		}
	}()

	time.Sleep(time.Millisecond * startupInMS)

	testFunc(GovalinHTTP{http: *httpClient, Host: fmt.Sprintf("http://localhost:%d", port)})

	err = server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to shutdown test server. %v", err)
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
	return uint16(l.Addr().(*net.TCPAddr).Port), nil
}
