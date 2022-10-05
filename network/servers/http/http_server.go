package httpserver

import (
	"fmt"
	"go-iot/codec"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
)

func ServerStart(network codec.Network) {
	spec := &HttpServerSpec{}
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
	fmt.Fprintf(w, "hello world "+productId)
}
