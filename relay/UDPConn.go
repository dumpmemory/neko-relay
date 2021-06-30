package relay

import (
	"errors"
	"net"
	"time"
)

type UDPConn struct {
	Connected bool
	Conn      *(net.UDPConn)
	Cache     chan []byte
	RAddr     net.Addr
	LAddr     net.Addr
}

func NewUDPConn(conn *(net.UDPConn), addr net.Addr) *UDPConn {
	return &UDPConn{
		Connected: true,
		Conn:      conn,
		Cache:     make(chan []byte, 16),
		RAddr:     addr,
		LAddr:     conn.LocalAddr(),
	}
}

func (uc *UDPConn) Close() error {
	uc.Connected = false
	return nil
}

func (uc *UDPConn) Read(b []byte) (n int, err error) {
	if !uc.Connected {
		return 0, errors.New("udp conn has closed")
	}

	select {
	case <-time.After(16 * time.Second):
		return 0, errors.New("i/o read timeout")
	case data := <-uc.Cache:
		n := len(data)
		copy(b, data)
		return n, nil
	}
}

func (uc *UDPConn) Write(b []byte) (int, error) {
	if !uc.Connected {
		return 0, errors.New("udp conn has closed")
	}
	return uc.Conn.WriteTo(b, uc.RAddr)
}

func (uc *UDPConn) RemoteAddr() net.Addr {
	return uc.RAddr
}

func (uc *UDPConn) LocalAddr() net.Addr {
	return uc.LAddr
}

func (uc *UDPConn) SetDeadline(t time.Time) error {
	return uc.Conn.SetDeadline(t)
}

func (uc *UDPConn) SetReadDeadline(t time.Time) error {
	return uc.Conn.SetReadDeadline(t)
}

func (uc *UDPConn) SetWriteDeadline(t time.Time) error {
	return uc.Conn.SetWriteDeadline(t)
}
