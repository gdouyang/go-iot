package web

import (
	"go-iot/pkg/logger"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type (
	dynamicMux struct {
		server *Server
		router atomic.Value
	}
)

func newDynamicMux(server *Server) *dynamicMux {
	m := &dynamicMux{
		server: server,
	}

	m.router.Store(chi.NewRouter())

	m.reloadAPIs()

	return m
}

func (m *dynamicMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.router.Load().(*chi.Mux).ServeHTTP(w, r)
}

func (m *dynamicMux) reloadAPIs() {
	router := chi.NewMux()

	router.Use(middleware.StripSlashes)
	router.Use(m.newAPILogger)
	router.Use(m.newRecoverer)

	for _, api := range apis {
		path := APIPrefix + api.Path
		handler := func(e *Entry) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				reflectVal := reflect.ValueOf(e.Controller)
				t := reflect.Indirect(reflectVal).Type()
				ctl := reflect.New(t)
				execController, ok := ctl.Interface().(ControllerInterface)
				if !ok {
					panic("controller is not ControllerInterface")
				}
				execController.Init(w, r)
				execController.Prepare()

				vc := reflect.ValueOf(execController)
				method := vc.MethodByName(e.Handler)
				method.Call(nil)
			}
		}(api)
		switch api.Method {
		case "GET":
			router.Get(path, handler)
		case "HEAD":
			router.Head(path, handler)
		case "PUT":
			router.Put(path, handler)
		case "POST":
			router.Post(path, handler)
		case "PATCH":
			router.Patch(path, handler)
		case "DELETE":
			router.Delete(path, handler)
		case "CONNECT":
			router.Connect(path, handler)
		case "OPTIONS":
			router.Options(path, handler)
		case "TRACE":
			router.Trace(path, handler)
		default:
			logger.Errorf("BUG: unsupported method: %s",
				api.Method)
		}
	}
	// Create a route along /static that will serve contents from
	// the ./static/ folder.
	workDir, _ := os.Getwd()
	m.FileServer(router, "/", http.Dir(filepath.Join(workDir, "views")))
	m.FileServer(router, "/static", http.Dir(filepath.Join(workDir, "static")))
	m.FileServer(router, "/api/file", http.Dir(filepath.Join(workDir, "files")))

	m.router.Store(router)
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func (m *dynamicMux) FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
