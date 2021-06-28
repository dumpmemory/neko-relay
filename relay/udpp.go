package relay

import (
	"fmt"
	"log"
	"net"
	"strings"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/txthinking/socks5"
)

func (s *Relay) ListenUdp() (err error) {
	s.UDPConn, err = net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		fmt.Println("Listen UDP", s.Laddr, err)
	}
	return
}
func (s *Relay) AcceptAndHandleUdp(handle func(addr *net.UDPAddr, b []byte) error) error {
	s.UDPExchanges = cmap.New()
	s.UDPSrc = cmap.New()
	for {
		select {
		case <-s.StopCh:
			return nil
		default:
			b := make([]byte, 65507)
			n, addr, err := s.UDPConn.ReadFromUDP(b)
			if err != nil {
				return err
			}
			go func(addr *net.UDPAddr, b []byte) {
				if err := handle(addr, b); err != nil {
					log.Println(err)
					return
				}
			}(addr, b[:n])
		}
	}
}

type uconn struct {
	UDPConn *net.UDPConn
	ue      *socks5.UDPExchange
}

func (u *uconn) Write(b []byte) (int, error) {
	return u.UDPConn.WriteToUDP(b, u.ue.ClientAddr)
}

func (s *Relay) UdpHandle(addr *net.UDPAddr, b []byte) error {
	src := addr.String()
	send := func(ue *socks5.UDPExchange, data []byte) error {
		_, err := ue.RemoteConn.Write(data)
		if err != nil {
			return err
		}
		return nil
	}

	dst := s.Raddr
	var ue *socks5.UDPExchange
	iue, ok := s.UDPExchanges.Get(src + dst)
	if ok {
		ue = iue.(*socks5.UDPExchange)
		return send(ue, b)
	}

	var laddr *net.UDPAddr
	any, ok := s.UDPSrc.Get(src + dst)
	if ok {
		laddr = any.(*net.UDPAddr)
	}
	rc, err := net.DialUDP("udp", laddr, s.RUDPAddr)
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			// we dont choose lock, so ignore this error
			return nil
		}
		return err
	}
	if laddr == nil {
		s.UDPSrc.Set(src+dst, rc.LocalAddr().(*net.UDPAddr))
	}
	ue = &socks5.UDPExchange{
		ClientAddr: addr,
		RemoteConn: rc,
	}
	if err := send(ue, b); err != nil {
		ue.RemoteConn.Close()
		return err
	}
	s.UDPExchanges.Set(src+dst, ue)
	go func(ue *socks5.UDPExchange, dst string) {
		defer func() {
			ue.RemoteConn.Close()
			s.UDPExchanges.Remove(s.Raddr + dst)
		}()
		s.Copy_udp(s.UDPConn, ue.RemoteConn, ue.ClientAddr)
	}(ue, dst)
	return nil
}
func (s *Relay) RunUdpServer() error {
	err := s.ListenUdp()
	if err != nil {
		return err
	}
	go s.AcceptAndHandleUdp(s.UdpHandle)
	return nil
}
