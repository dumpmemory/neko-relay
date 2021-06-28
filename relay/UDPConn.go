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

func (this *UDPConn) Close() error {
	this.Connected = false
	return nil
}

func (this *UDPConn) Read(b []byte) (n int, err error) {
	if !this.Connected {
		return 0, errors.New("udp conn has closed")
	}

	select {
	case <-time.After(16 * time.Second):
		return 0, errors.New("i/o read timeout")
	case data := <-this.Cache:
		n := len(data)
		copy(b, data)
		return n, nil
	}
}

func (this *UDPConn) Write(b []byte) (int, error) {
	if !this.Connected {
		return 0, errors.New("udp conn has closed")
	}
	return this.Conn.WriteTo(b, this.RAddr)
}

func (this *UDPConn) RemoteAddr() net.Addr {
	return this.RAddr
}

func (this *UDPConn) LocalAddr() net.Addr {
	return this.LAddr
}

func (this *UDPConn) SetDeadline(t time.Time) error {
	return this.Conn.SetDeadline(t)
}

func (this *UDPConn) SetReadDeadline(t time.Time) error {
	return this.Conn.SetReadDeadline(t)
}

func (this *UDPConn) SetWriteDeadline(t time.Time) error {
	return this.Conn.SetWriteDeadline(t)
}
