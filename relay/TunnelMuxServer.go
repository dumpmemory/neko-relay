package relay

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
)

type TunnelServer struct {
	mu        *sync.RWMutex
	Handlers  map[string]http.Handler
	RP        *RP
	TCPListen *net.TCPListener
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
func (S *TunnelServer) Serve(port int) error {
	taddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Start TunnelMuxServer:", err)
		return err
	}
	S.TCPListen, err = net.ListenTCP("tcp", taddr)
	if err != nil {
		fmt.Println("Start TunnelMuxServer:", err)
		return err
	}
	go http.Serve(S.TCPListen, S)
	return nil
}
