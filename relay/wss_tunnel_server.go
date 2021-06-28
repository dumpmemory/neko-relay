package relay

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWssTunnelServer(tcp, udp bool) error {
	if Config.Tsp.Wss > 0 {
		if tcp {
			WssMuxTunnelServer.AddHandler("/wss/tcp/"+s.RID+"/", websocket.Handler(s.WsTunnelServerTcpHandle))
		}
		if udp {
			WssMuxTunnelServer.AddHandler("/wss/udp/"+s.RID+"/", websocket.Handler(s.WsTunnelServerUdpHandle))
		}
		return nil
	}
	err := s.ListenTCP()
	if err != nil {
		return err
	}
	handler := http.NewServeMux()
	if tcp {
		handler.Handle("/wss/tcp/"+s.RID+"/", websocket.Handler(s.WssTunnelServerTcpHandle))
	}
	if udp {
		handler.Handle("/wss/udp/"+s.RID+"/", websocket.Handler(s.WssTunnelServerUdpHandle))
	}
	handler.Handle("/", NewRP(Config.Fake.Url, Config.Fake.Host))
	s.Svr = &http.Server{Handler: handler}
	go s.Svr.ServeTLS(s.TCPListen, Config.Tls.Cert, Config.Tls.Key)
	return nil
}
func (s *Relay) WssTunnelServerTcpHandle(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	defer ws.Close()

	tmp, err := net.DialTimeout("tcp", s.Raddr, time.Duration(s.TCPTimeout)*time.Second)
	if err != nil {
		return
	}
	rc := tmp.(*net.TCPConn)
	defer rc.Close()
	go s.Copy(rc, ws)
	s.Copy(ws, rc)
}

func (s *Relay) WssTunnelServerUdpHandle(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	defer ws.Close()

	rc, err := net.DialTimeout("udp", s.Raddr, time.Duration(s.UDPTimeout)*time.Second)
	if err != nil {
		return
	}
	defer rc.Close()

	go s.Copy(rc, ws)
	s.Copy(ws, rc)
}

var WssMuxTunnelServer = &TunnelServer{
	mu:       new(sync.RWMutex),
	Handlers: make(map[string]http.Handler),
	RP:       NewRP(Config.Fake.Url, Config.Fake.Host),
}
