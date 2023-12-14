package govalin_test

import (
	"testing"

	"github.com/gorilla/websocket"
	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/stretchr/testify/assert"
)

func TestWebsocketOpen(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Ws("/ws", func(wsConfig *govalin.WsConfig) {
			wsConfig.OnOpen = func(wsConnection *govalin.WsConnection) {
				err := wsConnection.SendText("Hello open")
				assert.Nil(t, err, "Should not return error when sending text")
			}
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		ws := http.Websocket("/ws")
		defer ws.Close()

		_, message, err := ws.ReadMessage()
		assert.Nil(t, err, "Should not return error when reading message")

		assert.Equal(t, "Hello open", string(message), "Should receive message from server")
	})
}

func TestWebsocketOnMessage(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Ws("/ws", func(wsConfig *govalin.WsConfig) {
			wsConfig.OnMessage = func(wsMessage *govalin.WsMessage) {
				err := wsMessage.ReplyText(wsMessage.AsText())
				assert.Nil(t, err, "Should not return error when replying text")
			}
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		ws := http.Websocket("/ws")
		defer ws.Close()

		err := ws.WriteMessage(websocket.TextMessage, []byte("Hello server"))
		assert.Nil(t, err, "Should not return error when sending message")

		_, message, err := ws.ReadMessage()
		assert.Nil(t, err, "Should not return error when reading message")

		assert.Equal(t, "Hello server", string(message), "Should receive echo from server")
	})
}

func TestWebsocketOnClose(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Ws("/ws", func(wsConfig *govalin.WsConfig) {
			wsConfig.OnClose = func(closeCode int, closeReason string) {
				assert.Equal(
					t,
					websocket.CloseAbnormalClosure,
					closeCode, "Should receive 'abnormal' (actually quite normal) closure code",
				)
			}
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		ws := http.Websocket("/ws")
		ws.Close()
	})
}

func TestWebsocket(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Ws("/ws", func(wsConfig *govalin.WsConfig) {
			wsConfig.OnClose = func(closeCode int, closeReason string) {
				assert.Equal(
					t,
					websocket.CloseNormalClosure,
					closeCode, "Should receive 'normal' closure code",
				)
			}
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		ws := http.Websocket("/ws")
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Test normal closure")
		assert.NoError(t, ws.WriteMessage(websocket.CloseMessage, closeMessage), "Should not return error when closing")
		ws.Close()
	})
}
