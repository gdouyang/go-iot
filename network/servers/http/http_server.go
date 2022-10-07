package httpserver

import (
	"compress/gzip"
	"fmt"
	"go-iot/codec"
	"io"
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

	codec.NewCodec(network)

	go func() {
		err := http.ListenAndServe(addr, nil)

		if err != nil {
			logs.Error(err)
		}
	}()
}

func socketHandler(w http.ResponseWriter, r *http.Request, productId string) {
	r.ParseForm()
	session := newSession(w, r)
	// defer session.Disconnect()

	sc := codec.GetCodec(productId)
	message := getBody(r, 1024)
	sc.OnMessage(&httpContext{
		BaseContext: codec.BaseContext{
			DeviceId:  session.GetDeviceId(),
			ProductId: productId,
			Session:   session,
		},
		Data: message,
		r:    r,
	})
}

func getBody(r *http.Request, MaxMemory int64) []byte {
	if r.Body == nil {
		return []byte{}
	}

	var requestbody []byte
	safe := &io.LimitedReader{R: r.Body, N: MaxMemory}
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(safe)
		if err != nil {
			return nil
		}
		requestbody, _ = io.ReadAll(reader)
	} else {
		requestbody, _ = io.ReadAll(safe)
	}

	r.Body.Close()
	return requestbody
}
