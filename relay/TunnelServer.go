package relay

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
)

type TunnelServer struct {
	mu       *sync.RWMutex
	Handlers map[string]http.Handler
	RP       *RP
}

func (S *TunnelServer) AddHandler(pattern string, handler http.Handler) error {
	S.mu.Lock()
	defer S.mu.Unlock()
	if pattern == "" {
		return errors.New("invalid pattern")
	}
	if handler == nil {
		return errors.New("handler is nil")
	}
	S.Handlers[pattern] = handler
	return nil
}
func (S *TunnelServer) DelHandler(pattern string) error {
	S.mu.Lock()
	defer S.mu.Unlock()
	if pattern == "" {
		return errors.New("invalid pattern")
	}
	delete(S.Handlers, pattern)
	return nil
}

func setHeader(w http.ResponseWriter) {
	for key, val := range Config.Fake.Headers {
		w.Header().Set(key, val)
	}
}

func (S *TunnelServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, has := S.Handlers[r.URL.Path]
	if has {
		setHeader(w)
		handler.ServeHTTP(w, r)
	} else if Config.Fake.Host != "" {
		S.RP.ServeHTTP(w, r)
	}
}
func (S *TunnelServer) ListenAndServe(port int) error {
	go http.ListenAndServe(":"+strconv.Itoa(port), S)
	return nil
}
