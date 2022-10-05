package websocketsocker

import (
	"fmt"
	"go-iot/codec"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

func ServerStart(network codec.Network) {
	spec := &WebsocketServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port

	if len(spec.Paths) == 0 {
		spec.Paths = append(spec.Paths, "/")
	}

	for _, path := range spec.Paths {
		http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			socketHandler(w, r, network.ProductId)
		})
	}
	addr := spec.Host + ":" + fmt.Sprint(spec.Port)

	err := http.ListenAndServe(addr, nil)

	if err != nil {
		logs.Error(err)
	}
}

func socketHandler(w http.ResponseWriter, r *http.Request, productId string) {
	// Upgrade our raw HTTP connection to a websocket based one
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logs.Error("Error during connection upgradation:", err)
		return
	}

	session := newSession(conn)
	defer session.Disconnect()

	sc := codec.GetCodec(productId)

	sc.OnConnect(&websocketContext{
		BaseContext: codec.BaseContext{ProductId: productId,
			Session: session,
		},
	})

	// The event loop
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			logs.Error("Error during message reading:", err)
			break
		}
		// logs.Info("Received: %s", message)

		sc.OnMessage(&websocketContext{
			BaseContext: codec.BaseContext{ProductId: productId,
				Session: session,
			},
			Data:    message,
			msgType: messageType,
		})
	}
}
