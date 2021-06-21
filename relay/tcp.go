package relay

import (
	"fmt"
	"net"
	"time"
)

func (s *Relay) ListenTCP() (err error) {
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		fmt.Println("Listen TCP", s.Laddr, err)
	}
	return
}

func (s *Relay) AcceptAndHandleTCP(handle func(c *net.TCPConn) error) error {
	wait := 1.0
	for {
		select {
		case <-s.StopCh:
			return nil
		default:
			c, err := s.TCPListen.AcceptTCP()
			if err == nil {
				go handle(c)
				wait = 1.0
			} else {
				fmt.Println("Accept", s.Laddr, err)
				if err, ok := err.(net.Error); ok && err.Temporary() {
					continue
				}
				time.Sleep(time.Duration(wait) * time.Second)
				wait *= 1.1
			}
		}
	}
	return nil
}

func (s *Relay) RunTCPServer() error {
	err := s.ListenTCP()
	if err != nil {
		return err
	}
	go s.AcceptAndHandleTCP(s.TCPHandle)
	return nil
}

func (s *Relay) TCPHandle(c *net.TCPConn) error {
	defer c.Close()
	rc, err := net.DialTimeout("tcp", s.Raddr, time.Duration(s.TCPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial TCP", s.Laddr, "<=>", s.Raddr, err)
		return err
	}
	defer rc.Close()
	go Copy(c, rc, s)
	Copy(rc, c, s)

	return nil
}
