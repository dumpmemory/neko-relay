package config

type CONF struct {
	Key   string
	Port  int
	Debug bool

	Certfile string
	Keyfile  string

	Syncfile string

	Fakehost string
	Fakeurl  string

	Dns struct {
		Server  string
		Network string
	}

	Tsp struct {
		Ws  int
		Wss int
		H2  int
	}
}
