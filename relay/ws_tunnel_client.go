package relay

import (
	"fmt"
	"net"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelTcpClient() error {
	err := s.ListenTCP()
	if err != nil {
		return err
	}
	go s.AcceptAndHandleTCP(s.WsTunnelClientTcpHandle)
	return nil
}

func (s *Relay) WsTunnelClientTcpHandle(c *net.TCPConn) error {
	defer c.Close()
	ws_config, err := websocket.NewConfig("ws://"+s.Raddr+"/ws/tcp/"+s.RID+"/", "http://"+s.Raddr+"/ws/tcp/"+s.RID+"/")
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

func (s *Relay) RunWsTunnelUdpClient() error {
	err := s.ListenUDP()
	if err != nil {
		return err
	}
	go s.AcceptAndHandleUDP(s.WsTunnelClientUdpHandle)
	return nil
}

func (s *Relay) WsTunnelClientUdpHandle(c net.Conn) error {
	defer func() {
		c.Close()
		s.releaseConn()
	}()
	ws_config, err := websocket.NewConfig("ws://"+s.Raddr+"/ws/udp/"+s.RID+"/", "http://"+s.Raddr+"/ws/udp/"+s.RID+"/")
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

	go s.Copy_io(c, rc, true)
	s.Copy_io(rc, c, true)
	return nil
}
