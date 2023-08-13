package web

const (
	// APIPrefix is the prefix of api.
	APIPrefix = "/api"
)

var (
	apis = []*Entry{}
)

func init() {

}

// registers global admin APIs.
func RegisterAPIs(api []*Entry) {
	apis = append(apis, api...)
}

func RegisterAPI(path string, method string, c ControllerInterface, handler string) {
	apis = append(apis, &Entry{Path: path, Method: method, Controller: c, Handler: handler})
}

// func (s *Server) listAPIEntries() []*Entry {
// 	return []*server.Entry{
// 		{
// 			Path:    "",
// 			Method:  "GET",
// 			Handler: s.listAPIs,
// 		},
// 	}
// }

// func (s *Server) listAPIs(w http.ResponseWriter, r *http.Request) {
// 	buff, err := yaml.Marshal(apis)
// 	if err != nil {
// 		panic(fmt.Errorf("marshal %#v to yaml failed: %v", apis, err))
// 	}
// 	w.Header().Set("Content-Type", "text/vnd.yaml")
// 	w.Write(buff)
// }
