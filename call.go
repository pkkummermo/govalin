package govalin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"log/slog"

	"github.com/google/uuid"
	"github.com/pkkummermo/govalin/internal/http/charsets"
	"github.com/pkkummermo/govalin/internal/http/contenttypes"
	"github.com/pkkummermo/govalin/internal/http/headers"
	"github.com/pkkummermo/govalin/internal/session"
	"github.com/pkkummermo/govalin/internal/validation"
	"golang.org/x/exp/maps"
)

const sessionCookieName = "govalin-session"

type raw struct {
	W   *http.ResponseWriter
	Req *http.Request
}

// Call is used to interact with the request and response object
//
// It exposes several convenience methods for handling both path, query as well
// as body for the request. It follows simple conventions for getting and setting
// values and uses the same method for getting values from the request and setting
// values on the response by having optional values.
type Call struct {
	id            string
	config        *Config
	status        int
	statusWritten bool
	w             http.ResponseWriter
	req           *http.Request
	pathParams    map[string]string
	bodyBytes     []byte
	charset       string
	session       session.Session
	Raw           raw // Raw contains the raw request and response
}

func newCallFromRequest(w http.ResponseWriter, req *http.Request, config *Config, pathParams map[string]string) Call {
	govalinIDHeader := req.Header[http.CanonicalHeaderKey("x-govalin-id")]

	var uniqueID string
	if govalinIDHeader == nil {
		uniqueID = uuid.New().String()
	} else {
		uniqueID = govalinIDHeader[0]
	}

	call := Call{
		id:         uniqueID,
		config:     config,
		w:          w,
		req:        req,
		status:     0,
		pathParams: pathParams,
		charset:    charsets.UTF8,
		Raw: raw{
			W:   &w,
			Req: req,
		},
	}

	if config.server.sessionsEnabled {
		initiateSessionFromCall(&call)
	}

	return call
}

// initiateSessionFromCall tries to get the session from the request. If the session
// doesn't exist, it will create a new session and set the session cookie on the response.
func initiateSessionFromCall(call *Call) {
	sessionCookie, getSessionErr := call.Cookie(sessionCookieName)

	// Create the session if it doesn't exist
	if errors.Is(http.ErrNoCookie, getSessionErr) {
		addNewSessionErr := addNewSessionToCall(call)
		if addNewSessionErr != nil {
			slog.Error("Failed to add new session to call", addNewSessionErr)
		}
		return
	}

	session, getSessionErr := call.config.server.sessionStore.GetSession(sessionCookie.Value, 0)
	// The session might be expired, so we need to create a new one
	if getSessionErr != nil {
		slog.Debug("Failed to get session from session store, adding new session", getSessionErr)
		addNewSessionErr := addNewSessionToCall(call)
		if addNewSessionErr != nil {
			slog.Error("Failed to add new session to call", addNewSessionErr)
		}
		return
	}

	call.session = session
}

func addNewSessionToCall(call *Call) error {
	sessionID, createSessionErr := call.config.server.sessionStore.
		CreateSession(time.Now().Add(call.config.server.sessionExpireTime).UnixNano())

	if createSessionErr != nil {
		slog.Error("Failed to create session", createSessionErr)
		return createSessionErr
	}

	session, getNewSessionErr := call.config.server.sessionStore.
		GetSession(sessionID, 0)
	if getNewSessionErr != nil {
		slog.Error("Failed to get session from session store", getNewSessionErr)
		return getNewSessionErr
	}

	_, cookieErr := call.Cookie(sessionCookieName, &http.Cookie{
		Value:    sessionID,
		Expires:  time.Now().Add(call.config.server.sessionExpireTime),
		HttpOnly: true,
	})
	if cookieErr != nil {
		slog.Error("Failed to set session cookie", cookieErr)
		return cookieErr
	}

	call.session = session
	return nil
}

// readBody reads the body as bytes and caches the value on call.
func (call *Call) readBody() ([]byte, error) {
	if call.bodyBytes != nil {
		return call.bodyBytes, nil
	}

	limitedReader := io.LimitReader(call.req.Body, call.config.server.maxBodyReadSize)

	bytes, err := io.ReadAll(limitedReader)
	if err != nil {
		call.bodyBytes = []byte{}
		return []byte{}, fmt.Errorf("failed to read request body. %w", err)
	}

	// If the size of bytes read and max body read size is the same, we could have a too big of a body.
	// Try to read a single byte to see if the body still has any data
	if len(bytes) == int(call.config.server.maxBodyReadSize) {
		numBytes, readError := call.req.Body.Read(make([]byte, 1))

		if (readError == nil || errors.Is(readError, io.EOF)) && numBytes == 1 {
			call.bodyBytes = []byte{}
			return []byte{}, fmt.Errorf("request body was too big, could not read full body")
		}
	}

	call.bodyBytes = bytes

	return call.bodyBytes, nil
}

// parseForm parses the internal request form based on Content-Type. If the Content-Type
// is not recognized, it returns a validation error.
func (call *Call) parseForm() error {
	contentType := call.Header(headers.ContentType)

	switch {
	case strings.Contains(contentType, contenttypes.ApplicationFormURLEncoded):
		err := call.req.ParseForm()
		if err != nil {
			slog.Error("Failed to parse form data", err)
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(
					"formData",
					"Invalid form data",
				),
			))
		}
		return nil
	case strings.Contains(contentType, contenttypes.MultipartFormData):
		err := call.req.ParseMultipartForm(0)
		if err != nil {
			slog.Error("Failed to parse form data", err)
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(
					"formData",
					"Invalid form data",
				),
			))
		}

		return nil
	default:
		slog.Warn("POST request is missing the correct content-type to parse form param")
		return validation.NewError(validation.NewErrorResponse(
			http.StatusBadRequest,
			validation.NewParameterErrorDetail(
				headers.ContentType,
				"Missing or invalid '"+headers.ContentType+"' header. "+
					"Must be '"+contenttypes.MultipartFormData+"' or "+
					"'"+contenttypes.ApplicationFormURLEncoded+"'",
			),
		))
	}
}

func (call *Call) sendStatusOrDefault() {
	if call.statusWritten {
		return
	}

	if call.status == 0 {
		call.status = http.StatusOK
	}

	call.w.WriteHeader(call.status)
	call.statusWritten = true
}

// ID gives an UUIDv4 string that's unique to the call.
func (call *Call) ID() string {
	return call.id
}

// Method returns the method for the current request.
func (call *Call) Method() string {
	return call.req.Method
}

// Authorization returns the Authorization header if available.
func (call *Call) Authorization() string {
	return call.Header(headers.Authorization)
}

// Host returns the host of the current request.
func (call *Call) Host() string {
	return call.Header(headers.Host)
}

// Referer returns the requests referer header if available. Also handles the edge
// case if the name of the header names spelling is correct (referrer).
func (call *Call) Referer() string {
	return call.Header(headers.Referer)
}

// UserAgent returns the request UserAgent if available.
func (call *Call) UserAgent() string {
	return call.Header(headers.UserAgent)
}

// URL returns the requested URI.
func (call *Call) URL() *url.URL {
	return call.req.URL
}

// Get or set a Cookie by name and value
//
// Get a Cookie based on given Cookie name request
// or set a Cookie on the response by providing a value.
func (call *Call) Cookie(name string, cookies ...*http.Cookie) (*http.Cookie, error) {
	if len(cookies) > 0 {
		cookies[0].Name = name
		http.SetCookie(call.w, cookies[0])
		return cookies[0], nil
	}

	return call.req.Cookie(name)
}

// Get the value of given form param key
//
// Parses the body as a www-form-urlencoded body. If the content type is not correct
// a warning is given and an empty string is returned.
func (call *Call) FormParam(key string) (string, error) {
	err := call.parseForm()
	if err != nil {
		return "", err
	}

	return call.req.Form.Get(key), nil
}

// FormParams returns all form parameters available in the request body.
func (call *Call) FormParams() (url.Values, error) {
	err := call.parseForm()
	if err != nil {
		return make(url.Values), err
	}

	return call.req.Form, nil
}

// File returns a FileHeader for given file name in the request body.
func (call *Call) File(key string) (*multipart.FileHeader, error) {
	err := call.parseForm()
	if err != nil {
		return nil, err
	}

	if file, ok := call.req.MultipartForm.File[key]; ok {
		return file[0], nil
	}

	return nil, validation.NewError(validation.NewErrorResponse(
		http.StatusBadRequest,
		validation.NewParameterErrorDetail(
			key,
			fmt.Sprintf("Missing file with name '%s'", key),
		),
	))
}

// Files returns an array for the given file name in the request body.
func (call *Call) Files(key string) ([]*multipart.FileHeader, error) {
	err := call.parseForm()
	if err != nil {
		return nil, err
	}

	if file, ok := call.req.MultipartForm.File[key]; ok {
		return file, nil
	}

	return nil, validation.NewError(validation.NewErrorResponse(
		http.StatusBadRequest,
		validation.NewParameterErrorDetail(
			key,
			fmt.Sprintf("Missing files with name '%s'", key),
		),
	))
}

// Get form param value by key, if empty, use default
//
// Get a form param value based on given key from the request,
// or use the given default value if the value is an empty string.
func (call *Call) FormParamOrDefault(key string, def string) string {
	formParam, err := call.FormParam(key)

	if formParam == "" || err != nil {
		return def
	}

	return formParam
}

// Get query param for given key
//
// Returns the query param value as string.
func (call *Call) QueryParam(key string) string {
	return call.req.URL.Query().Get(key)
}

// Get query param by key, if empty, use default
//
// Returns the query param value as string or use the given
// default value if the value is an empty string.
func (call *Call) QueryParamOrDefault(key string, def string) string {
	queryParam := call.QueryParam(key)

	if queryParam == "" {
		return def
	}

	return queryParam
}

// Get path param based on key.
func (call *Call) PathParam(key string) string {
	if _, ok := call.pathParams[key]; !ok {
		slog.Error(
			fmt.Sprintf(
				"Tried to access non-existing path param '%s'."+
					"This is most likely an error and should be fixed. Available values are: %v",
				key,
				maps.Keys(call.pathParams),
			),
		)
	}

	return call.pathParams[key]
}

// Get all path params as a map
//
// Returns a map populated with the values based on the
// configuration of the path URL as a map[string]string.
func (call *Call) PathParams() map[string]string {
	return call.pathParams
}

// Get or set header by given key and value
//
// Get a header value based on given header key from the request
// or set header value on the response by providing a value.
func (call *Call) Header(key string, value ...string) string {
	key = http.CanonicalHeaderKey(key)

	if len(value) > 0 {
		call.w.Header().Add(key, value[0])
		return value[0]
	}

	if key == headers.Host {
		return call.req.Host
	}

	if call.req.Header[key] != nil {
		return call.req.Header[key][0]
	}

	return ""
}

// Get header value by key, if empty, use default
//
// Get a header value based on given header key from the request,
// or use the given default value if the value is an empty string.
func (call *Call) HeaderOrDefault(key string, def string) string {
	value := call.Header(key)

	if value == "" {
		return def
	}

	return value
}

// Set HTTP status that will be used on JSON/Text/HTML calls
//
// If the status has already been set, a warning will be printed. The status will not be
// written to the response until a JSON/Text/HTML-call is made.
func (call *Call) Status(statusCode ...int) int {
	if len(statusCode) > 0 {
		call.status = statusCode[0]
	}

	return call.status
}

// Send text as pure text to response
//
// Text will set the content-type of the response as text/plain and write it to the response.
// If no other status has been given the response, it will write a 200 OK to the response.
func (call *Call) Text(text string) {
	call.w.Header().Add(headers.ContentType, headers.ContentTypeHeader(contenttypes.TextPlain, call.charset))
	call.sendStatusOrDefault()

	_, err := call.w.Write([]byte(text))
	if err != nil {
		slog.Error(fmt.Sprintf("Error when trying write to response, %v", err))
	}
}

// Send text as HTML to response
//
// HTML will set the content-type of the response as text/html and write it to the response.
// If no other status has been given the response, it will write a 200 OK to the response.
func (call *Call) HTML(text string) {
	call.w.Header().Add(headers.ContentType, headers.ContentTypeHeader(contenttypes.TextHTML, call.charset))
	call.sendStatusOrDefault()

	_, err := call.w.Write([]byte(text))
	if err != nil {
		slog.Error(fmt.Sprintf("Error when trying write to response, %v", err))
	}
}

// Send obj as JSON to response
//
// JSON will set the content-type of the response as application/json and serializes the given
// object as JSON, and writes it to the response. If no other status has been given the response,
// it will write a 200 OK to the response.
func (call *Call) JSON(obj interface{}) {
	call.w.Header().Add(headers.ContentType, headers.ContentTypeHeader(contenttypes.ApplicationJSON, charsets.UTF8))
	jsonBytes, err := json.Marshal(obj)

	if err != nil {
		slog.Error(fmt.Sprintf("error when trying to JSON marshall object, %v", err))
	}

	call.sendStatusOrDefault()

	_, err = call.w.Write(jsonBytes)

	if err != nil {
		slog.Error(fmt.Sprintf("error when trying write to response, %v", err))
	}
}

// Redirect redirects the request to the given URL
//
// Redirect will set the status code to 302 or 301 (if permenant) and
// set the location header to the given URL.
func (call *Call) Redirect(url string, permanent ...bool) {
	if len(permanent) > 0 && permanent[0] {
		call.Status(http.StatusMovedPermanently)
	} else {
		call.Status(http.StatusFound)
	}
	call.Header(headers.Location, url)
	call.sendStatusOrDefault()
}

// Get body as given struct
//
// BodyAs takes a pointer as input and tries to deserialize the body into the object
// expecting the body to be JSON. Returns an error on failed unmarshalling or non-pointer.
func (call *Call) BodyAs(obj any) error {
	bodyBytes, err := call.readBody()

	if err != nil {
		return err
	}

	if reflect.ValueOf(obj).Type().Kind() != reflect.Pointer {
		return newErrorFromType(serverError, fmt.Errorf("must provide a pointer to correctly unmarshal body"))
	}

	err = json.Unmarshal(bodyBytes, obj)
	if err != nil {
		return newErrorFromType(userError, err)
	}

	return nil
}

// Get or set a session attribute by key and value
//
// Get or set a session attribute based on given key from the request. The
// session attribute will be stored in the session store and will be available
// for the next request. If no value is given, it will return the value of the
// session attribute. If the session attribute is not found, an error will be returned.
func (call *Call) SessionAttr(key string, value ...any) (any, error) {
	if !call.config.server.sessionsEnabled {
		slog.Warn(`Tried to access session attributes when sessions were not enabled.
To enable, either enable sessions on the app config object or use the session plugin when creating the app`)
		return nil, errors.New("session handling is not enabled")
	}

	if len(value) > 0 {
		call.session.Data[key] = value[0]
		return value[0], call.config.server.sessionStore.SetSessionData(call.session.ID, call.session.Data)
	}

	if call.session.Data[key] == nil {
		return nil, errors.New("session attribute not found")
	}

	return call.session.Data[key], nil
}

// Get a session attribute by key or default value
//
// Get a session attribute based on given key from the request. The
// session attribute is stored in the session store and will be available
// for the next request. If the session attribute is not found, the default
// value will be returned.
func (call *Call) SessionAttrOrDefault(key string, def any) any {
	if !call.config.server.sessionsEnabled {
		slog.Warn("Tried to access session attributes when sessions were not enabled")
		return def
	}

	if call.session.Data[key] == nil {
		return def
	}

	return call.session.Data[key]
}

// Handle an error
//
// Write a response based on given error. If the error is recognized as a
// govalin error the error is handled specific according to the error.
func (call *Call) Error(err error) {
	var govalinErr *govalinError
	if errors.As(err, &govalinErr) {
		if govalinErr.errorType == userError {
			call.Status(http.StatusBadRequest)
		} else if govalinErr.errorType == serverError {
			call.Status(http.StatusInternalServerError)
		}

		var unmarshalErr *json.UnmarshalTypeError
		if errors.As(govalinErr.originalError, &unmarshalErr) {
			call.JSON(validation.GetUnmarshalError(unmarshalErr).ErrorResponse)
			return
		}

		var jsonSyntaxErr *json.SyntaxError
		if errors.As(govalinErr.originalError, &jsonSyntaxErr) {
			call.JSON(validation.NewError(
				validation.NewErrorResponse(
					http.StatusBadRequest,
					validation.NewParameterErrorDetail("jsonBody", "Invalid JSON found in body"),
				),
			).ErrorResponse)
			return
		}

		slog.Warn(
			fmt.Sprintf(
				"Unknown govalin error %v. Original err: %v. Error not handled", govalinErr, govalinErr.originalError,
			),
		)

		return
	}

	var validationErr *validation.Error
	if errors.As(err, &validationErr) {
		call.Status(http.StatusBadRequest)
		call.JSON(validationErr.ErrorResponse)
		return
	}

	slog.Error(fmt.Sprintf("Unknown error '%v'. Error not handled", err))
	call.JSON(validation.NewError(
		validation.NewErrorResponse(
			http.StatusInternalServerError,
		),
	).ErrorResponse)
}
