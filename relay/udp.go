package relay

import (
	"fmt"
	"net"
	"time"
)

func (s *Relay) ListenUDP() (err error) {
	s.UDPConn, err = net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		fmt.Println("Listen UDP", s.Laddr, err)
	}
	return
}
func (s *Relay) AcceptAndHandleUDP(handle func(c net.Conn) error) error {
	wait := 1.0
	table := make(map[string]*UDPDistribute)
	buf := make([]byte, 1024*32*2)
	for {
		select {
		case <-s.StopCh:
			return nil
		default:
			// s.acquireConn()
			n, addr, err := s.UDPConn.ReadFrom(buf)
			if err != nil {
				s.releaseConn()
				fmt.Println("Accept", s.Laddr, err)
				if err, ok := err.(net.Error); ok && err.Temporary() {
					continue
				}
				time.Sleep(time.Duration(wait) * time.Second)
				wait *= 1.1
				break
			} else {
				wait = 1.0
			}
			go func() {
				buf = buf[:n]
				if d, ok := table[addr.String()]; ok {
					if d.Connected {
						d.Cache <- buf
						return
					} else {
						delete(table, addr.String())
					}
				}
				c := NewUDPDistribute(s.UDPConn, addr)
				table[addr.String()] = c
				c.Cache <- buf
				handle(c)
			}()
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
	go s.Copy(c, rc)
	s.Copy(rc, c)
	return nil
}
