package govalin

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

// StaticConfig contains configuration for a static handler.
type StaticConfig struct {
	hostPath   string
	spaMode    bool
	staticPath string
	fsContent  fs.FS
	validators *sync.Map
}

func newStaticConfig(validators *sync.Map) *StaticConfig {
	return &StaticConfig{
		hostPath:   "/",
		spaMode:    false,
		staticPath: "static",
		fsContent:  nil,
		validators: validators,
	}
}

// indexFileName is the file served for the mount root and for the SPA fallback.
const indexFileName = "index.html"

// serveIndex serves the mount's index.html, for the mount root and for the SPA
// fallback. A mount without one is a misconfiguration, so it is reported as
// such rather than as a missing file.
func (config *StaticConfig) serveIndex(call *Call, hostedFileSystem fs.FS) {
	failStatic(call, config.serveFile(call, hostedFileSystem, indexFileName), fmt.Sprintf(
		`Failed to serve %s for the static mount on '%s'.
This might be due to a misconfigured static path or embedded bundle, or simply
that the %s file doesn't exist.`, indexFileName, config.hostPath, indexFileName))
}

// failStatic answers a static file that could not be served. A response already
// on the wire cannot be turned into an error, so a failure part-way through a
// body is only logged.
func failStatic(call *Call, err error, message string) {
	if err == nil {
		return
	}

	slog.Error(message, "err", err)

	if !call.committed() {
		call.Error(err)
	}
}

// validatorFor derives the validator for a static file: from when the file changed if the
// file system knows, from what it contains if it does not, as an embedded one does not.
// Both forms are strong tags — net/http needs a strong match to honour If-Range.
func (config *StaticConfig) validatorFor(hostedFileSystem fs.FS, name string, fileInfo fs.FileInfo) string {
	if !fileInfo.ModTime().IsZero() {
		return fmt.Sprintf("%x-%x", fileInfo.ModTime().UnixNano(), fileInfo.Size())
	}

	if cached, isCached := config.validators.Load(name); isCached {
		return cached.(string)
	}

	etag, hashErr := hashFile(hostedFileSystem, name)
	if hashErr != nil {
		// A validator we could not derive only means the client revalidates next time.
		slog.Debug("Failed to hash a static file for its validator", "name", name, "err", hashErr)

		return ""
	}

	config.validators.Store(name, etag)

	return etag
}

// hashFile derives a validator from a file's content, on its own handle so the
// one being served keeps its position.
func hashFile(hostedFileSystem fs.FS, name string) (string, error) {
	file, openErr := hostedFileSystem.Open(name)
	if openErr != nil {
		return "", openErr
	}
	defer func() { _ = file.Close() }()

	// Not a security boundary: this names a version, it does not attest to one.
	hash := fnv.New64a()
	if _, copyErr := io.Copy(hash, file); copyErr != nil {
		return "", copyErr
	}

	return fmt.Sprintf("%x", hash.Sum64()), nil
}

// serveFile sends a file from the mounted file system through Call, so the
// response stays inside the govalin lifecycle and Range requests are answered.
func (config *StaticConfig) serveFile(call *Call, hostedFileSystem fs.FS, name string) error {
	file, openErr := hostedFileSystem.Open(name)
	if openErr != nil {
		return openErr
	}
	defer func() { _ = file.Close() }()

	fileInfo, statErr := file.Stat()
	if statErr != nil {
		return statErr
	}

	// Both sinks revalidate the same way; http.ServeContent covers only the seekable one.
	if etag := config.validatorFor(hostedFileSystem, name, fileInfo); etag != "" && call.NotModified(etag) {
		return nil
	}

	// Only a seekable file can serve ranges; os.Root and embed give one, a custom fs.FS may not.
	seekableFile, isSeekable := file.(io.ReadSeeker)
	if !isSeekable {
		return call.Stream(mime.TypeByExtension(path.Ext(name)), file)
	}

	call.ServeContent(name, fileInfo.ModTime(), seekableFile)

	return nil
}

// serveWithFileServer hands the request to net/http for the two jobs govalin
// leaves it: generating a directory listing, and redirecting a URL whose
// trailing slash disagrees with what the name it addresses holds.
func (config *StaticConfig) serveWithFileServer(call *Call, hostedFileSystem fs.FS) {
	http.StripPrefix(
		config.hostPath,
		http.FileServer(http.FS(hostedFileSystem)),
	).ServeHTTP(*call.Raw.W, call.Raw.Req)
}

func (config *StaticConfig) handle(call *Call) {
	isFS := config.fsContent != nil

	mountPath := strings.TrimPrefix(call.URL().Path, config.hostPath)

	// The file server is handed the URL with the mount stripped off, so this is the one
	// trailing-slash redirect it cannot make. The query addresses the resource, not its spelling.
	if mountPath == "" {
		canonicalURL := config.hostPath + "/"
		if rawQuery := call.URL().RawQuery; rawQuery != "" {
			canonicalURL += "?" + rawQuery
		}

		call.Redirect(canonicalURL, true)

		return
	}

	hostedFileSystem := config.fsContent

	if !isFS {
		// os.Root has the OS refuse an escaping symlink or traversal, not a path compare (ADR 0006).
		root, rootErr := os.OpenRoot(config.staticPath)
		if rootErr != nil {
			slog.Error(fmt.Sprintf(`Failed to open the configured static file folder.
Are you sure it exists and is readable on the given path: '%s'`, config.staticPath))
			call.Status(http.StatusInternalServerError)
			call.Text("500 internal server error")

			return
		}
		defer func() { _ = root.Close() }()

		hostedFileSystem = root.FS()
	}

	// io/fs refuses a name carrying a trailing slash, so the URL's claim is kept apart from it.
	namedAsDirectory := strings.HasSuffix(mountPath, "/")

	// An fs.FS names its own root "." rather than the empty string.
	name := strings.TrimRight(strings.TrimPrefix(mountPath, "/"), "/")
	if name == "" {
		name = "."
	}

	fileInfo, statErr := fs.Stat(hostedFileSystem, name)

	var pathErr *fs.PathError

	// A '..' segment or an escaping symlink must not answer differently from a missing file.
	isNotFoundError := errors.Is(statErr, fs.ErrNotExist) || errors.As(statErr, &pathErr)

	// A name that resolves to nothing is a client-side route on a SPA mount.
	if config.spaMode && isNotFoundError {
		config.serveIndex(call, hostedFileSystem)
		return
	}

	switch {
	case isNotFoundError:
		// The file server answers an escaping name with 500, which confirms something is there.
		call.Status(http.StatusNotFound)
		call.Text("404 page not found")

		return
	case statErr != nil:
		call.Status(http.StatusInternalServerError)
		call.Error(statErr)
		return
	case fileInfo.IsDir() != namedAsDirectory:
		// The URL and the file system disagree, so the file server redirects to the
		// spelling that addresses what is there, and neither is served at both.
		config.serveWithFileServer(call, hostedFileSystem)

		return
	case fileInfo.IsDir():
		// A directory with an index.html is serving a file, so it goes through Call for a validator.
		indexName := path.Join(name, indexFileName)
		if _, indexErr := fs.Stat(hostedFileSystem, indexName); indexErr != nil {
			// Except on a SPA mount, whose bundle is nobody's to enumerate.
			if config.spaMode {
				config.serveIndex(call, hostedFileSystem)
			} else {
				config.serveWithFileServer(call, hostedFileSystem)
			}

			return
		}

		name = indexName
	}

	failStatic(
		call,
		config.serveFile(call, hostedFileSystem, name),
		fmt.Sprintf("Failed to serve the static file '%s'", name),
	)
}

// HostPath sets the host path for the static handler. This is trimmed from the
// URL before serving the static files.
func (config *StaticConfig) HostPath(hostPath string) *StaticConfig {
	config.hostPath = hostPath

	return config
}

// WithStaticPath sets the path to a directory containing static files.
//
// The directory will be served at the given path, relative to where the server
// is started.
func (config *StaticConfig) WithStaticPath(staticPath string) *StaticConfig {
	config.staticPath = staticPath

	return config
}

// WithFS sets the bundled FS to serve static files from.
func (config *StaticConfig) WithFS(fsContent fs.FS) *StaticConfig {
	config.fsContent = fsContent

	return config
}

// EnableSPAMode enables SPA mode for the static handler.
//
// SPA mode will serve the index.html file for all requests that doesn't match
// a static file.
func (config *StaticConfig) EnableSPAMode(spaMode bool) *StaticConfig {
	config.spaMode = spaMode

	return config
}

// Add a Static endpoint
//
// Add a static endpoint which will serve static files from the given path or bundled FS.
func (server *App) Static(path string, staticHandlerFunc StaticHandlerFunc) *App {
	// TODO: this doesn't feel right to override the path like this
	normalizedPath := strings.TrimRight(path, "/*")
	wildcardPath := normalizedPath + "/*"

	// Outlives the per-request config, and is scoped to one mount, so the file name is key enough.
	validators := &sync.Map{}

	staticGetHandler := func(call *Call) {
		internalConfig := newStaticConfig(validators)
		internalConfig.HostPath(normalizedPath)

		staticHandlerFunc(call, internalConfig)

		internalConfig.handle(call)
	}

	// TODO: this should be handled by a single handler, not two
	server.addMethod(http.MethodGet, server.currentFragment+normalizedPath+"/", staticGetHandler)
	server.addMethod(http.MethodGet, server.currentFragment+wildcardPath, staticGetHandler)

	return server
}
