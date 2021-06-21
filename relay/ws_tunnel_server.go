package relay

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelServer(tcp, udp bool) error {
	if Config.Tsp.Ws > 0 {
		if tcp {
			WsMuxTunnelServer.AddHandler("/ws/tcp/"+s.RID+"/", websocket.Handler(s.WsTunnelServerTcpHandle))
		}
		if udp {
			WsMuxTunnelServer.AddHandler("/ws/udp/"+s.RID+"/", websocket.Handler(s.WsTunnelServerUdpHandle))
		}
		return nil
	}
	err := s.ListenTCP()
	if err != nil {
		return err
	}
	handler := http.NewServeMux()
	if tcp {
		handler.Handle("/ws/tcp/"+s.RID+"/", websocket.Handler(s.WsTunnelServerTcpHandle))
	}
	if udp {
		handler.Handle("/ws/udp/"+s.RID+"/", websocket.Handler(s.WsTunnelServerUdpHandle))
	}
	handler.Handle("/", NewRP(Config.Fakeurl, Config.Fakehost))

	s.Svr = &http.Server{Handler: handler}
	go s.Svr.Serve(s.TCPListen)
	return nil
}

func (s *Relay) WsTunnelServerTcpHandle(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	defer ws.Close()

	rc, err := net.DialTimeout("tcp", s.Raddr, time.Duration(s.TCPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial TCP", s.Laddr, "<=>", s.Raddr, err)
		return
	}
	defer rc.Close()
	go Copy(rc, ws, s)
	Copy(ws, rc, s)
}

func (s *Relay) WsTunnelServerUdpHandle(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	defer ws.Close()

	rc, err := net.DialTimeout("udp", s.Raddr, time.Duration(s.UDPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial UDP", s.Laddr, "<=>", s.Raddr, err)
		return
	}
	defer rc.Close()
	go Copy(rc, ws, s)
	Copy(ws, rc, s)
}

type TunnelServer struct {
	mu       *sync.RWMutex
	Handlers map[string]http.Handler
	RP       *RP
}

var WsMuxTunnelServer = &TunnelServer{
	mu:       new(sync.RWMutex),
	Handlers: make(map[string]http.Handler),
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

func (S *TunnelServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, has := S.Handlers[r.URL.Path]
	if has {
		handler.ServeHTTP(w, r)
	} else {
		S.RP.ServeHTTP(w, r)
	}
}
func (S *TunnelServer) ListenAndServe(port int) error {
	go http.ListenAndServe(":"+strconv.Itoa(port), S)
	return nil
}
