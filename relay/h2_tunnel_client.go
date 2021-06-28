package relay

import (
	"fmt"
	"net"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunH2TunnelTcpClient() error {
	err := s.ListenTCP()
	if err != nil {
		return err
	}
	go s.AcceptAndHandleTCP(s.H2TunnelClientTcpHandle)
	return nil
}

func (s *Relay) H2TunnelClientTcpHandle(c *net.TCPConn) error {
	defer c.Close()
	ws_config, err := websocket.NewConfig("ws://"+s.Raddr+"/wstcp/", "http://"+s.Raddr+"/wstcp/")
	if err != nil {
		fmt.Println("WS Config", s.Raddr, err)
		return err
	}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", s.RIP)
	ws_config.Header.Set("X-Forward-Host", Config.Fake.Host)
	ws_config.Header.Set("X-Forward-Protocol", c.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", c.RemoteAddr().String())

	rc, err := websocket.DialConfig(ws_config)
	if err != nil {
		fmt.Println("Dial WS", s.Raddr, err)
		return err
	}
	defer rc.Close()
	rc.PayloadType = websocket.BinaryFrame

	go s.Copy(rc, c)
	s.Copy(c, rc)
	return nil
}

func (s *Relay) RunH2TunnelUdpClient() error {
	err := s.ListenUDP()
	if err != nil {
		return err
	}
	go s.AcceptAndHandleUDP(s.H2TunnelClientUdpHandle)
	return nil
}

func (s *Relay) H2TunnelClientUdpHandle(c net.Conn) error {
	defer c.Close()
	ws_config, err := websocket.NewConfig("ws://"+s.Raddr+"/wsudp/", "http://"+s.Raddr+"/wsudp/")
	if err != nil {
		fmt.Println("WS Config", s.Raddr, err)
		return err
	}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4240.198 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", s.RIP)
	ws_config.Header.Set("X-Forward-Host", Config.Fake.Host)
	ws_config.Header.Set("X-Forward-Protocol", c.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", c.RemoteAddr().String())

	rc, err := websocket.DialConfig(ws_config)
	if err != nil {
		fmt.Println("Dial WS", s.Raddr, err)
		return err
	}
	defer rc.Close()
	rc.PayloadType = websocket.BinaryFrame

	go s.Copy(c, rc)
	s.Copy(rc, c)
	return nil
}
