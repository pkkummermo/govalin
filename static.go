package govalin

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type StaticConfig struct {
	hostPath   string
	spaMode    bool
	staticPath string
	fsContent  fs.FS
}

func newStaticConfig() *StaticConfig {
	return &StaticConfig{
		hostPath:   "/",
		spaMode:    false,
		staticPath: "static",
		fsContent:  nil,
	}
}

func (config *StaticConfig) serveIndexFS(call *Call) {
	// file does not exist, serve index.html
	file, openIndexErr := config.fsContent.Open("index.html")
	if openIndexErr != nil {
		slog.Error(`Failed to open index.html in bundled embedded files.
This might be due to a misconfigured embedded bundle or simply 
that the index.html files doesn't exist.`)
		call.Error(openIndexErr)
		return
	}

	index, readIndexErr := io.ReadAll(file)
	if readIndexErr != nil {
		slog.Error(`Failed to read index.html in bundled embedded files.
This might be due to a misconfigured embedded bundle or simply 
that the index.html files doesn't exist.`)
		call.Error(openIndexErr)
		return
	}

	call.Status(http.StatusOK)
	call.HTML(string(index))
}

func (config *StaticConfig) serveIndexStatic(call *Call) {
	indexPath := filepath.Join(config.staticPath, "index.html")
	if _, indexFileDoesNotExist := os.Stat(
		indexPath,
	); os.IsNotExist(indexFileDoesNotExist) {
		slog.Error(fmt.Sprintf(`Failed to read the index.html file in the given static file folder. 
Are you sure it exists on the given path: '%s'`, indexPath))
		call.Error(indexFileDoesNotExist)
		return
	}

	// file does not exist, serve index.html
	call.Status(http.StatusOK)
	http.ServeFile(*call.Raw.W, call.Raw.Req, filepath.Join(config.staticPath, "index.html"))
}

func (config *StaticConfig) handle(call *Call) {
	isFS := config.fsContent != nil
	isStatic := config.fsContent == nil

	path := call.URL().Path

	// remove host path
	path = strings.TrimPrefix(path, config.hostPath)

	if isStatic {
		// prepend the path with the path to the static directory
		path = filepath.Join(config.staticPath, path)
	}

	// check whether a file exists at the given path
	var statErr error

	if isFS {
		_, statErr = fs.Stat(config.fsContent, path)
	} else {
		_, statErr = os.Stat(path)
	}

	var pathErr *fs.PathError

	isNotFoundError := errors.Is(statErr, fs.ErrNotExist) || errors.As(statErr, &pathErr)

	// Serve index if:
	// 1. If path is empty (slash root)
	// 2. if SPA mode is enabled, and if the file doesn't exist
	if path == "" || (config.spaMode && isNotFoundError) {
		if isFS {
			config.serveIndexFS(call)
		} else {
			config.serveIndexStatic(call)
		}
		return
	}

	switch {
	case isNotFoundError:
		call.Status(http.StatusNotFound)
	case statErr != nil:
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		call.Status(http.StatusInternalServerError)
		call.Error(statErr)
		return
	default:
		call.Status(http.StatusOK)
	}

	// otherwise, use http.FileServer to serve the file system
	var hostedFileSystem http.FileSystem
	if isFS {
		hostedFileSystem = http.FS(config.fsContent)
	} else {
		hostedFileSystem = http.Dir(config.staticPath)
	}

	http.StripPrefix(
		config.hostPath,
		http.FileServer(hostedFileSystem),
	).ServeHTTP(*call.Raw.W, call.Raw.Req)
}

func (config *StaticConfig) HostPath(hostPath string) *StaticConfig {
	config.hostPath = hostPath

	return config
}

func (config *StaticConfig) WithStaticPath(staticPath string) *StaticConfig {
	config.staticPath = staticPath

	return config
}

func (config *StaticConfig) WithFS(fsContent fs.FS) *StaticConfig {
	config.fsContent = fsContent

	return config
}

func (config *StaticConfig) EnableSPAMode(spaMode bool) *StaticConfig {
	config.spaMode = spaMode

	return config
}
