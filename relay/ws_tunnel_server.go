package relay

import (
	"fmt"
	"net"
	"net/http"
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
	handler.Handle("/", NewRP(Config.Fake.Url, Config.Fake.Host))

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

var WsMuxTunnelServer = &TunnelServer{
	mu:       new(sync.RWMutex),
	Handlers: make(map[string]http.Handler),
	RP:       NewRP(Config.Fake.Url, Config.Fake.Host),
}
