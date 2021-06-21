package relay

import (
	"context"
	"errors"
	"io"
	"log"
	"neko-relay/config"
	"neko-relay/limits"
	. "neko-relay/rules"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	Config config.CONF
)

type Relay struct {
	RID        string
	TCPAddr    *net.TCPAddr
	UDPAddr    *net.UDPAddr
	TCPListen  *net.TCPListener
	UDPConn    *net.UDPConn
	Svr        *http.Server
	TCPTimeout int
	UDPTimeout int
	Laddr      string
	Raddr      string
	REMOTE     string
	RIP        string
	RPORT      int
	Traffic    *TF
	Protocol   string
	StopCh     chan struct{}
}

func NewRelay(rid string, r Rule, tcpTimeout, udpTimeout int, traffic *TF, protocol string) (*Relay, error) {
	laddr := ":" + strconv.Itoa(int(r.Port))
	raddr := r.RIP + ":" + strconv.Itoa(int(r.Rport))
	taddr, err := net.ResolveTCPAddr("tcp", laddr)
	if err != nil {
		return nil, err
	}
	uaddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	if err := limits.Raise(); err != nil {
		log.Println("Try to raise system limits, got", err)
	}
	s := &Relay{
		RID:        rid,
		TCPAddr:    taddr,
		UDPAddr:    uaddr,
		TCPTimeout: tcpTimeout,
		UDPTimeout: udpTimeout,
		Laddr:      laddr,
		Raddr:      raddr,
		RIP:        r.RIP,
		REMOTE:     r.Remote,
		Traffic:    traffic,
		Protocol:   protocol,
	}
	return s, nil
}

// Run server.
func (s *Relay) Serve() error {
	s.StopCh = make(chan struct{}, 16)
	if s.Protocol == "tcp" {
		return s.RunTCPServer()
	} else if s.Protocol == "udp" {
		return s.RunUDPServer()
	} else if s.Protocol == "tcp+udp" {
		if err := s.RunTCPServer(); err != nil {
			return err
		}
		return s.RunUDPServer()

	} else if s.Protocol == "http" {
		return s.RunHttpServer(false)
	} else if s.Protocol == "https" {
		return s.RunHttpServer(true)

	} else if s.Protocol == "ws_tunnel_server_tcp" {
		return s.RunWsTunnelServer(true, false)
	} else if s.Protocol == "ws_tunnel_server_udp" {
		return s.RunWsTunnelServer(false, true)
	} else if s.Protocol == "ws_tunnel_server" {
		return s.RunWsTunnelServer(true, true)

	} else if s.Protocol == "ws_tunnel_client_tcp" {
		return s.RunWsTunnelTcpClient()
	} else if s.Protocol == "ws_tunnel_client_udp" {
		return s.RunWsTunnelUdpClient()
	} else if s.Protocol == "ws_tunnel_client" {
		if err := s.RunWsTunnelTcpClient(); err != nil {
			return err
		}
		return s.RunWsTunnelUdpClient()

	} else if s.Protocol == "wss_tunnel_server_tcp" {
		return s.RunWssTunnelServer(true, false)
	} else if s.Protocol == "wss_tunnel_server_udp" {
		return s.RunWssTunnelServer(false, true)
	} else if s.Protocol == "wss_tunnel_server_tcp+udp" || s.Protocol == "wss_tunnel_server" {
		return s.RunWssTunnelServer(true, true)

	} else if s.Protocol == "wss_tunnel_client_tcp" {
		return s.RunWssTunnelTcpClient()
	} else if s.Protocol == "wss_tunnel_client_udp" {
		return s.RunWssTunnelUdpClient()
	} else if s.Protocol == "wss_tunnel_client_tcp+udp" || s.Protocol == "wss_tunnel_client" {
		if err := s.RunWssTunnelTcpClient(); err != nil {
			return err
		}
		return s.RunWsTunnelUdpClient()

	} else if s.Protocol == "h2_tunnel_server_tcp" {
		return s.RunH2TunnelServer(true, false)
	} else if s.Protocol == "h2_tunnel_server_udp" {
		return s.RunH2TunnelServer(false, true)
	} else if s.Protocol == "h2_tunnel_server_tcp+udp" || s.Protocol == "h2_tunnel_server" {
		return s.RunH2TunnelServer(true, true)

	} else if s.Protocol == "h2_tunnel_client_tcp" {
		return s.RunH2TunnelTcpClient()
	} else if s.Protocol == "h2_tunnel_client_udp" {
		return s.RunH2TunnelUdpClient()
	} else if s.Protocol == "h2_tunnel_client_tcp+udp" || s.Protocol == "h2_tunnel_client" {
		if err := s.RunH2TunnelTcpClient(); err != nil {
			return err
		}
		return s.RunH2TunnelUdpClient()
	}
	return nil
}

func (s *Relay) OK() (bool, error) {
	if strings.Contains(s.Protocol, "tcp") && (s.TCPListen == nil) {
		return false, errors.New("tcp listen is null")
	}
	if strings.Contains(s.Protocol, "udp") && (s.TCPListen == nil) {
		return false, errors.New("tcp listen is null")
	}
	return true, nil
}

// Shutdown server.
func (s *Relay) Close() error {
	close(s.StopCh)
	if s.Svr != nil {
		s.Svr.Shutdown(context.Background())
	}
	time.Sleep(5 * time.Millisecond)
	if s.TCPListen != nil {
		s.TCPListen.Close()
	}
	if s.UDPConn != nil {
		s.UDPConn.Close()
	}
	time.Sleep(5 * time.Millisecond)
	return nil
}

var (
	Pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 16*1024)
		},
	}
)

func Copy(dst, src net.Conn, s *Relay) error {
	defer src.Close()
	defer dst.Close()
	return Copy_io(dst, src, s)
}

func Copy_io(dst io.Writer, src io.Reader, s *Relay) error {
	// n, err := io.Copy(dst, src)
	// if err != nil {
	// 	return nil
	// }
	// if tf != nil {
	// 	tf.Add(uint64(n))
	// }
	// return nil
	buf := Pool.Get().([]byte)
	defer Pool.Put(buf)
	for {
		select {
		case <-s.StopCh:
			return nil
		default:
			n, err := src.Read(buf[:])
			if err != nil {
				return err
			}
			if s == nil {
				return errors.New("s is nil")
			}
			s.Traffic.Add(uint64(n))
			if _, err := dst.Write(buf[0:n]); err != nil {
				return err
			}
		}
	}
}
