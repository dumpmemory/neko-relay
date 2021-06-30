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

	"github.com/juju/ratelimit"
	cmap "github.com/orcaman/concurrent-map"
)

var (
	Config config.CONF
	D      = net.Dialer{Timeout: 30 * time.Second}
)

type Relay struct {
	RID          string
	Laddr        string
	TCPAddr      *net.TCPAddr
	UDPAddr      *net.UDPAddr
	TCPListen    *net.TCPListener
	UDPConn      *net.UDPConn
	UDPExchanges cmap.ConcurrentMap
	UDPSrc       cmap.ConcurrentMap
	Svr          *http.Server
	TCPTimeout   int
	UDPTimeout   int
	Raddr        string
	RTCPAddr     *net.TCPAddr
	RUDPAddr     *net.UDPAddr
	REMOTE       string
	RIP          string
	RPORT        int
	Traffic      *TF
	Protocol     string
	StopCh       chan struct{}
	Limit        struct {
		Speed       int
		Connections int
	}
	Bucket      *ratelimit.Bucket
	ConnLimiter chan struct{}
}

func NewRelay(rid string, r Rule, tcpTimeout, udpTimeout int, traffic *TF, protocol string) (*Relay, error) {
	laddr := ":" + strconv.Itoa(r.Port)
	raddr := r.RIP + ":" + strconv.Itoa(r.Rport)
	taddr, err := net.ResolveTCPAddr("tcp", laddr)
	if err != nil {
		return nil, err
	}
	uaddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	rtaddr, err := net.ResolveTCPAddr("tcp", raddr)
	if err != nil {
		return nil, err
	}
	ruaddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		return nil, err
	}
	if err := limits.Raise(); err != nil {
		log.Println("Try to raise system limits, got", err)
	}
	s := &Relay{
		RID:        rid,
		Laddr:      laddr,
		TCPAddr:    taddr,
		UDPAddr:    uaddr,
		TCPTimeout: tcpTimeout,
		UDPTimeout: udpTimeout,
		Raddr:      raddr,
		RTCPAddr:   rtaddr,
		RUDPAddr:   ruaddr,
		RIP:        r.RIP,
		REMOTE:     r.Remote,
		Traffic:    traffic,
		Protocol:   protocol,
		Limit:      r.Limit,
	}
	if r.Limit.Speed > 0 {
		s.Bucket = ratelimit.NewBucketWithRate(
			float64(r.Limit.Speed*128*1024),
			int64(r.Limit.Speed*128*1024),
		)
	}
	if r.Limit.Connections > 0 {
		s.ConnLimiter = make(chan struct{}, r.Limit.Connections)
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
		return s.RunHttpsServer()

	} else if s.Protocol == "ws_tunnel_server_tcp" {
		return s.RunWsTunnelServer(true, false)
	} else if s.Protocol == "ws_tunnel_server_udp" {
		return s.RunWsTunnelServer(false, true)
	} else if s.Protocol == "ws_tunnel_server_tcp+udp" || s.Protocol == "ws_tunnel_server" {
		return s.RunWsTunnelServer(true, true)

	} else if s.Protocol == "ws_tunnel_client_tcp" {
		return s.RunWsTunnelTcpClient()
	} else if s.Protocol == "ws_tunnel_client_udp" {
		return s.RunWsTunnelUdpClient()
	} else if s.Protocol == "ws_tunnel_client_tcp+udp" || s.Protocol == "ws_tunnel_client" {
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
	if Config.Tsp.Ws > 0 && strings.Contains(s.Protocol, "ws_tunnel_server") {
		return true, nil
	}
	if Config.Tsp.Wss > 0 && strings.Contains(s.Protocol, "wss_tunnel_server") {
		return true, nil
	}

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
	if Config.Tsp.Ws > 0 && strings.Contains(s.Protocol, "ws_tunnel_server") {
		WsMuxTunnelServer.DelHandler("/ws/tcp/" + s.RID + "/")
		WsMuxTunnelServer.DelHandler("/ws/udp/" + s.RID + "/")
	}
	if Config.Tsp.Wss > 0 && strings.Contains(s.Protocol, "wss_tunnel_server") {
		WssMuxTunnelServer.DelHandler("/wss/tcp/" + s.RID + "/")
		WssMuxTunnelServer.DelHandler("/wss/udp/" + s.RID + "/")
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
	LPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024)
		},
	}
)

func (s *Relay) Copy(dst, src net.Conn) error {
	defer src.Close()
	defer dst.Close()
	return s.Copy_io(dst, src, false)
}

func (s *Relay) Copy_io(dst io.Writer, src io.Reader, large_buf bool) error {
	if s.Limit.Speed > 0 {
		src = ratelimit.Reader(src, s.Bucket)
	}
	// n, err := io.Copy(dst, src)
	// if err != nil {
	// 	return nil
	// }
	// s.Traffic.Add(uint64(n))
	// return nil
	var b []byte
	if large_buf {
		b = LPool.Get().([]byte)
		defer LPool.Put(b)
	} else {
		b = Pool.Get().([]byte)
		defer Pool.Put(b)
	}
	for {
		select {
		case <-s.StopCh:
			return nil
		default:
			n, err := src.Read(b[:])
			if err != nil {
				return err
			}
			s.Traffic.Add(uint64(n))
			if _, err := dst.Write(b[0:n]); err != nil {
				return err
			}
		}
	}
}

func (s *Relay) Copy_udp(dst *net.UDPConn, src io.Reader, ClientAddr *net.UDPAddr) error {
	if s.Limit.Speed > 0 {
		src = ratelimit.Reader(src, s.Bucket)
	}
	b := LPool.Get().([]byte)
	defer LPool.Put(b)
	for {
		select {
		case <-s.StopCh:
			return nil
		default:
			n, err := src.Read(b[:])
			if err != nil {
				return err
			}
			s.Traffic.Add(uint64(n))
			if _, err := dst.WriteToUDP(b[0:n], ClientAddr); err != nil {
				return err
			}
		}
	}
}

func (s *Relay) acquireConn() {
	if s.Limit.Connections > 0 {
		s.ConnLimiter <- struct{}{}
	}
}
func (s *Relay) releaseConn() {
	// if s.Limit.Connections > 0 {
	<-s.ConnLimiter
	// }
}
