package relay

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunH2TunnelServer(tcp, udp bool) error {
	err := s.ListenTCP()
	if err != nil {
		return err
	}
	handler := http.NewServeMux()
	if tcp {
		handler.Handle("/wstcp/", websocket.Handler(s.H2TunnelServerTcpHandle))
	}
	if udp {
		handler.Handle("/wsudp/", websocket.Handler(s.H2TunnelServerUdpHandle))
	}
	handler.Handle("/", NewRP(Config.Fakeurl, Config.Fakehost))

	s.Svr = &http.Server{Handler: handler}
	go s.Svr.Serve(s.TCPListen)
	return nil
}

func (s *Relay) H2TunnelServerTcpHandle(ws *websocket.Conn) {
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

func (s *Relay) H2TunnelServerUdpHandle(ws *websocket.Conn) {
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
