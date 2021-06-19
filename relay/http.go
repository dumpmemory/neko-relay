package relay

import (
	"net/http"
	"net/url"
)

func (s *Relay) RunHttpServer(tls bool) error {
	err := s.ListenTCP()
	if err != nil {
		return err
	}

	handler := http.NewServeMux()
	target := "http://" + s.Raddr
	if tls {
		target = "https://" + s.Raddr
	}
	u, err := url.Parse(target)
	if err != nil {
		return err
	}
	handler.Handle("/", NewSingleHostReverseProxy(u, s))
	s.Svr = &http.Server{Handler: handler}
	if tls {
		go s.Svr.ServeTLS(s.TCPListen, Config.Certfile, Config.Keyfile)
	} else {
		go s.Svr.Serve(s.TCPListen)
	}
	return nil
}
