package govalin

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/gorilla/websocket"
)

// WsConfig contains configuration for a websocket handler.
type WsConfig struct {
	// OnUpgrade is called when a websocket connection is upgraded.
	OnUpgrade WsOnUpgradeFunc
	// OnOpen is called when a websocket connection is opened.
	OnOpen WsOnOpenFunc
	// OnMessage is called when a message is received from the client.
	OnMessage WsOnMessageFunc
	// OnClose is called when a websocket connection is closed.
	OnClose WsOnCloseFunc
	// OnError is called when an error occurs.
	OnError WsOnErrorFunc
}

func newWsConfig() *WsConfig {
	return &WsConfig{
		OnUpgrade: defaultOnUpgrade,
		OnOpen:    func(wsCall *WsConnection) {},
		OnMessage: func(wsMessage *WsMessage) {},
		OnClose:   func(closeCode int, closeReason string) {},
		OnError: func(err error) {
			slog.Error("Error occured on websocket", "err", err)
		},
	}
}

type WsHandlerFunc func(wsConfig *WsConfig)

type WsOnUpgradeFunc func(call *Call) (*WsConnection, error)
type WsOnOpenFunc func(wsConnection *WsConnection)
type WsOnCloseFunc func(closeCode int, closeReason string)
type WsOnErrorFunc func(err error)
type WsOnMessageFunc func(wsMessage *WsMessage)

const defaultWSReadBufferSize = 1024
const defaultWSWriteBufferSize = 1024

var upgrader = websocket.Upgrader{
	ReadBufferSize:  defaultWSReadBufferSize,
	WriteBufferSize: defaultWSWriteBufferSize,
}

// WsConnection is a connection to a websocket.
type WsConnection struct {
	call *Call
	conn *websocket.Conn
}

// WsMessage is a message received from a websocket.
type WsMessage struct {
	wsCall *WsConnection
	data   []byte
}

// AsBytes returns the message as a byte array.
func (message *WsMessage) AsBytes() []byte {
	return message.data
}

// AsText returns the message as a string.
func (message *WsMessage) AsText() string {
	return string(message.data)
}

// As unmarshals the message into the given jsonStruct.
func (message *WsMessage) As(jsonStruct interface{}) error {
	if reflect.ValueOf(jsonStruct).Type().Kind() != reflect.Pointer {
		return newErrorFromType(serverError, fmt.Errorf("must provide a pointer to correctly unmarshal body"))
	}

	unmarshallErr := json.Unmarshal(message.data, jsonStruct)
	if unmarshallErr != nil {
		return newErrorFromType(userError, unmarshallErr)
	}

	return nil
}

// Reply sends a JSON reply to the client of the message.
func (message *WsMessage) Reply(json interface{}) error {
	return message.wsCall.Send(json)
}

// ReplyText sends a reply to the client of the message.
func (message *WsMessage) ReplyText(text string) error {
	return message.wsCall.SendText(text)
}

// ReplyBytes sends a reply to the client of the message.
func (message *WsMessage) ReplyBytes(bytes []byte) error {
	return message.wsCall.conn.WriteMessage(websocket.BinaryMessage, bytes)
}

var defaultOnUpgrade = func(call *Call) (*WsConnection, error) {
	conn, err := upgrader.Upgrade(*call.Raw.W, call.Raw.Req, nil)
	if err != nil {
		return nil, err
	}

	return &WsConnection{
		call: call,
		conn: conn,
	}, nil
}

// Close closes the websocket connection.
func (wsConnection *WsConnection) Close() error {
	return wsConnection.conn.Close()
}

// Send sends a JSON message to the client.
func (wsConnection *WsConnection) Send(json interface{}) error {
	return wsConnection.conn.WriteJSON(json)
}

// SendText sends a text message to the client.
func (wsConnection *WsConnection) SendText(text string) error {
	return wsConnection.conn.WriteMessage(websocket.TextMessage, []byte(text))
}

// SendBytes sends a binary message to the client.
func (wsConnection *WsConnection) SendBytes(bytes []byte) error {
	return wsConnection.conn.WriteMessage(websocket.BinaryMessage, bytes)
}

// readWebsocketFunc is a helper function for reading messages from a websocket.
// It will read messages until the websocket is closed, and call the appropriate
// callbacks on the wsConfig.
func readWebsocketFunc(wsCall *WsConnection, wsConfig *WsConfig) {
	for {
		_, message, readMessageErr := wsCall.conn.ReadMessage()

		var closeError *websocket.CloseError

		if errors.As(readMessageErr, &closeError) {
			wsConfig.OnClose(closeError.Code, closeError.Text)
			closeErr := wsCall.Close()
			if closeErr != nil {
				wsConfig.OnError(closeErr)
			}
			return
		}

		if readMessageErr != nil {
			wsConfig.OnError(readMessageErr)
			wsCall.Close()
			return
		}

		wsConfig.OnMessage(&WsMessage{
			wsCall: wsCall,
			data:   message,
		})
	}
}

// Ws registers a websocket handler for the given path.
func (server *App) Ws(path string, handlerFunc WsHandlerFunc) *App {
	wsConfig := newWsConfig()
	handlerFunc(wsConfig)

	server.addMethod(http.MethodGet, server.currentFragment+path, func(call *Call) {
		call.Status(http.StatusSwitchingProtocols)
		wsCall, upgradeErr := wsConfig.OnUpgrade(call)

		if upgradeErr != nil {
			wsConfig.OnError(upgradeErr)
			return
		}

		wsConfig.OnOpen(wsCall)
		go readWebsocketFunc(wsCall, wsConfig)
	})

	return server
}
