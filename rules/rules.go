package rules

type Rule struct {
	Port   int
	Remote string
	RIP    string
	Rport  int
	Type   string
	Limit  struct {
		Speed       int
		Connections int
	}
}
