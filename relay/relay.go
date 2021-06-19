package relay

import (
	"context"
	"io"
	"log"
	"neko-relay/config"
	"neko-relay/limits"
	. "neko-relay/rules"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	Config config.CONF
)

type Relay struct {
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

func NewRelay(r Rule, tcpTimeout, udpTimeout int, traffic *TF, protocol string) (*Relay, error) {
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
func (s *Relay) Serve() (err error) {
	s.StopCh = make(chan struct{}, 16)
	if s.Protocol == "tcp" || s.Protocol == "tcp+udp" {
		if err = s.RunTCPServer(); err != nil {
			return
		}
	}
	if s.Protocol == "udp" || s.Protocol == "tcp+udp" {
		if err = s.RunUDPServer(); err != nil {
			return
		}
	}
	if s.Protocol == "http" {
		return s.RunHttpServer(false)
	}
	if s.Protocol == "https" {
		return s.RunHttpServer(true)
	}
	if s.Protocol == "ws_tunnel_server" {
		return s.RunWsTunnelServer(true, true)
	}
	if s.Protocol == "ws_tunnel_client" {
		if err = s.RunWsTunnelTcpClient(); err != nil {
			return
		}
		return s.RunWsTunnelUdpClient()
	}

	if s.Protocol == "wss_tunnel_server" {
		return s.RunWssTunnelServer(true, true)
	}
	if s.Protocol == "wss_tunnel_client" {
		if err = s.RunWssTunnelTcpClient(); err != nil {
			return
		}
		return s.RunWssTunnelUdpClient()
	}
	return nil
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
		s.TCPListen = nil
	}
	if s.UDPConn != nil {
		s.UDPConn.Close()
		s.UDPConn = nil
	}
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
	for dst != nil && src != nil && s.Traffic != nil {
		select {
		case <-s.StopCh:
			return nil
		default:
			n, err := src.Read(buf[:])
			if err != nil {
				return err
			}
			s.Traffic.Add(uint64(n))
			if _, err := dst.Write(buf[0:n]); err != nil {
				return err
			}
		}
	}
	return nil
}
