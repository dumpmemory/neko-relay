package relay

import (
	"errors"
	"net"
	"time"
)

type UDPDistribute struct {
	Connected bool
	Conn      *(net.UDPConn)
	Cache     chan []byte
	RAddr     net.Addr
	LAddr     net.Addr
}

func NewUDPDistribute(conn *(net.UDPConn), addr net.Addr) *UDPDistribute {
	return &UDPDistribute{
		Connected: true,
		Conn:      conn,
		Cache:     make(chan []byte, 16),
		RAddr:     addr,
		LAddr:     conn.LocalAddr(),
	}
}

func (th *UDPDistribute) Close() error {
	th.Connected = false
	return th.Conn.Close()
	// return nil
}

func (th *UDPDistribute) Read(b []byte) (n int, err error) {
	if !th.Connected {
		return 0, errors.New("udp conn has been closed")
	}
	select {
	case <-time.After(16 * time.Second):
		return 0, errors.New("i/o read timeout")
	case data := <-th.Cache:
		n := len(data)
		copy(b, data)
		return n, nil
	}
}

func (th *UDPDistribute) Write(b []byte) (int, error) {
	if !th.Connected {
		return 0, errors.New("udp conn has been closed")
	}
	return th.Conn.WriteTo(b, th.RAddr)
}

func (th *UDPDistribute) RemoteAddr() net.Addr {
	return th.RAddr
}
func (th *UDPDistribute) LocalAddr() net.Addr {
	return th.LAddr
}
func (th *UDPDistribute) SetDeadline(t time.Time) error {
	return th.Conn.SetDeadline(t)
}
func (th *UDPDistribute) SetReadDeadline(t time.Time) error {
	return th.Conn.SetReadDeadline(t)
}
func (th *UDPDistribute) SetWriteDeadline(t time.Time) error {
	return th.Conn.SetWriteDeadline(t)
}
