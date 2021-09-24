package http_helper

import (
	"github.com/gorilla/websocket"
	"net/http"
)

var (
	wsUpgrade = websocket.Upgrader{
		// 允许所有CORS跨域请求
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func UpgradeWebsocket(resp http.ResponseWriter, req *http.Request) (conn *websocket.Conn, err error) {
	// WebSocket握手
	return wsUpgrade.Upgrade(resp, req, nil)
}
