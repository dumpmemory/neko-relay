package relay

import (
	"fmt"
	"net"
	"time"

	cmap "github.com/orcaman/concurrent-map"
)

func (s *Relay) ListenUDP() (err error) {
	s.UDPConn, err = net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		fmt.Println("Listen UDP", s.Laddr, err)
	}
	return
}
func (s *Relay) AcceptAndHandleUDP(handle func(c net.Conn) error) error {
	table := cmap.New()
	for {
		select {
		case <-s.StopCh:
			return nil
		default:
			buf := make([]byte, 1024*32)
			s.acquireConn()
			n, addr, err := s.UDPConn.ReadFrom(buf)
			if err != nil {
				s.releaseConn()
				fmt.Println("Accept", s.Laddr, err)
				if err, ok := err.(net.Error); ok && err.Temporary() {
					continue
				}
				return err
			}
			b := buf[:n]
			if d, ok := table.Get(addr.String()); ok {
				if d.(*UDPConn).Connected {
					d.(*UDPConn).Cache <- buf
					continue
				} else {
					table.Remove(addr.String())
				}
			}
			c := NewUDPConn(s.UDPConn, addr)
			table.Set(addr.String(), c)
			c.Cache <- b
			go handle(c)
		}
	}
}
func (s *Relay) RunUDPServer() error {
	err := s.ListenUDP()
	if err != nil {
		return err
	}
	go s.AcceptAndHandleUDP(s.UDPHandle)
	return nil
}

func (s *Relay) UDPHandle(c net.Conn) error {
	defer c.Close()
	defer s.releaseConn()
	rc, err := net.DialTimeout("udp", s.Raddr, time.Duration(s.UDPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial UDP", s.Laddr, "<=>", s.Raddr, err)
		return err
	}
	defer rc.Close()
	go s.Copy_io(c, rc, true)
	s.Copy_io(rc, c, true)
	return nil
}
