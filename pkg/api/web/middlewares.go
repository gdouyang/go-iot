package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"go-iot/pkg/core/common"
	"go-iot/pkg/logger"

	"github.com/go-chi/chi/v5/middleware"
)

func (m *dynamicMux) newAPILogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		t1 := time.Now()
		defer func() {
			logger.Debugf("%s %s %s %v rx:%dB tx:%dB start:%v process:%v", r.Method, r.RemoteAddr, r.URL.Path, ww.Status(),
				r.ContentLength, int64(ww.BytesWritten()),
				t1, time.Since(t1))
		}()
		next.ServeHTTP(w, r)
	})
}

func (m *dynamicMux) newRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				if e, ok := rvr.(common.Err); ok {
					logger.Errorf("recover from %s, err: %v\n", r.URL.Path, rvr)
					HandleAPIError(w, r, e.Code, errors.New(e.Message))
				} else {
					logger.Errorf("recover from %s, err: %v, stack trace:\n%s\n",
						r.URL.Path, rvr, debug.Stack())
					HandleAPIError(w, r, http.StatusInternalServerError, fmt.Errorf("%v", rvr))
				}

			}
		}()
		next.ServeHTTP(w, r)
	})
}

// HandleAPIError handles api error.
func HandleAPIError(w http.ResponseWriter, r *http.Request, code int, err error) {
	w.WriteHeader(code)
	buff, err := json.Marshal(common.Err{
		Code:    code,
		Message: err.Error(),
	})
	if err != nil {
		panic(err)
	}
	w.Write(buff)
}
