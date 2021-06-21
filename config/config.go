package config

type CONF struct {
	Key   string
	Port  int
	Debug bool

	Tls struct {
		Cert string
		Key  string
	}

	Syncfile string

	Dns struct {
		Nameserver string
		Network    string
	}

	Tsp struct {
		Ws  int
		Wss int
		H2  int
	}

	Fake struct {
		Host    string
		Url     string
		Headers map[string]string
	}
}
