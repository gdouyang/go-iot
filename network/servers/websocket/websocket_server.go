package websocketsocker

import (
	"fmt"
	"go-iot/codec"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

func ServerStart(network codec.Network) {
	spec := &WebsocketServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port

	http.HandleFunc("/socket", func(w http.ResponseWriter, r *http.Request) {
		socketHandler(w, r, network.ProductId)
	})
	http.HandleFunc("/", home)
	addr := spec.Host + ":" + fmt.Sprint(spec.Port)

	err := http.ListenAndServe(addr, nil)

	if err != nil {
		log.Fatal(err)
	}
}

func socketHandler(w http.ResponseWriter, r *http.Request, productId string) {
	// Upgrade our raw HTTP connection to a websocket based one
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return
	}
	defer conn.Close()

	session := newSession(conn)
	defer session.DisConnect()

	sc := codec.GetCodec(productId)

	context := &websocketContext{productId: productId, session: session}

	sc.OnConnect(context)

	// The event loop
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error during message reading:", err)
			break
		}
		log.Printf("Received: %s", message)

		context.Data = message
		sc.Decode(context)
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Index Page")
}
