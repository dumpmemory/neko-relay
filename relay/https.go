package relay

import (
	"net"
	"strings"
)

func (s *Relay) RunHttpsServer() error {
	err := s.ListenTCP()
	if err != nil {
		return err
	}
	go s.AcceptAndHandleTCP(s.HttpsHandle)
	return nil
}

func (s *Relay) HttpsHandle(conn *net.TCPConn) (err error) {
	firstByte := make([]byte, 1)
	_, err = conn.Read(firstByte)
	if err != nil {
		conn.Close()
		return
	}

	if firstByte[0] != 0x16 {
		conn.Close()
		return
	}

	versionBytes := make([]byte, 2)
	_, err = conn.Read(versionBytes)
	if err != nil {
		conn.Close()
		return
	}
	if versionBytes[0] < 3 || (versionBytes[0] == 3 && versionBytes[1] < 1) {
		conn.Close()
		return
	}

	restLengthBytes := make([]byte, 2)
	_, err = conn.Read(restLengthBytes)
	if err != nil {
		conn.Close()
		return
	}
	restLength := (int(restLengthBytes[0]) << 8) + int(restLengthBytes[1])

	rest := make([]byte, restLength)
	_, err = conn.Read(rest)
	if err != nil {
		conn.Close()
		return
	}

	current := 0
	if len(rest) == 0 {
		conn.Close()
		return
	}
	handshakeType := rest[0]
	current += 1
	if handshakeType != 0x1 {
		conn.Close()
		return
	}

	current += 3
	current += 2
	current += 4 + 28
	sessionIDLength := int(rest[current])
	current += 1
	current += sessionIDLength

	cipherSuiteLength := (int(rest[current]) << 8) + int(rest[current+1])
	current += 2
	current += cipherSuiteLength

	compressionMethodLength := int(rest[current])
	current += 1
	current += compressionMethodLength

	if current > restLength {
		conn.Close()
		return
	}

	current += 2

	hostname := ""
	for current < restLength && hostname == "" {
		extensionType := (int(rest[current]) << 8) + int(rest[current+1])
		current += 2

		extensionDataLength := (int(rest[current]) << 8) + int(rest[current+1])
		current += 2

		if extensionType == 0 {
			current += 2

			nameType := rest[current]
			current += 1
			if nameType != 0 {
				conn.Close()
				return
			}
			nameLen := (int(rest[current]) << 8) + int(rest[current+1])
			current += 2
			hostname = strings.ToLower(string(rest[current : current+nameLen]))
		}

		current += extensionDataLength
	}

	if hostname == "" {
		conn.Close()
		return
	}

	// value, _ := Shared.HTTPS.Get(hostname)
	// i, ok := value.(string)
	// if !ok {
	// 	conn.Close()
	// 	return
	// }

	proxy, error := net.Dial("tcp", s.Raddr)
	if error != nil {
		conn.Close()
		return
	}

	proxy.Write(firstByte)
	proxy.Write(versionBytes)
	proxy.Write(restLengthBytes)
	proxy.Write(rest)

	go s.Copy(conn, proxy)
	s.Copy(proxy, conn)
	return
}
