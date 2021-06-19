package main

import (
	"context"
	"fmt"
	"neko-relay/relay"
	. "neko-relay/rules"
	"net"
	"time"

	cmap "github.com/orcaman/concurrent-map"
)

var (
	Rules   = cmap.New()
	Traffic = cmap.New()
	Svrs    = cmap.New()
	syncing = false
)

func getTF(rid string) (tf *relay.TF) {
	Tf, has := Traffic.Get(rid)
	if has {
		tf = Tf.(*relay.TF)
	}
	if !has {
		tf = relay.NewTF()
		Traffic.Set(rid, tf)
	}
	return
}

func start(rid string, r Rule) (err error) {
	svr, err := relay.NewRelay(r, 30, 10, getTF(rid), r.Type)
	if err != nil {
		return
	}
	Svrs.Set(rid, svr)
	svr.Serve()
	time.Sleep(5 * time.Millisecond)
	return
}
func stop(rid string, r Rule) {
	Svr, has := Svrs.Get(rid)
	if has {
		Svr.(*relay.Relay).Close()
		Svrs.Remove(rid)
	}
}
func restart(rid string, r Rule) error {
	Svr, has := Svrs.Get(rid)
	if has {
		Svr.(*relay.Relay).Close()
		return Svr.(*relay.Relay).Serve()
	} else {
		return start(rid, r)
	}
}
func cmp(x, y Rule) bool {
	return x.Port == y.Port && x.Remote == y.Remote && x.Rport == y.Rport && x.Type == y.Type
}

func sync(newRules map[string]Rule) {
	if syncing {
		return
		// syncing = false
		// time.Sleep(1000 * time.Millisecond)
	}
	syncing = true
	if Config.Debug {
		fmt.Println(newRules)
	}
	for item := range Rules.IterBuffered() {
		rid := item.Key
		rule, has := newRules[rid]
		if has && cmp(rule, item.Val.(Rule)) {
			delete(newRules, rid)
		} else {
			stop(rid, rule)
			Rules.Remove(rid)
			Traffic.Remove(rid)
		}
	}
	for rid, r := range newRules {
		if !syncing {
			return
		}
		if Config.Debug {
			fmt.Println(r)
		}
		rip, err := getIP(r.Remote)
		if err != nil {
			continue
		}
		r.RIP = rip
		err = check(r)
		if err == nil {
			Rules.Set(rid, r)
			start(rid, r)
		}
	}
	syncing = false
}

var (
	rsvr = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Duration(10000) * time.Millisecond,
			}
			return d.DialContext(ctx, Config.Dns.Network, Config.Dns.Server+":53")
		},
	}
)

func getIP(host string) (string, error) {
	ips, err := rsvr.LookupHost(context.Background(), host)
	// ips, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	return ips[0], nil
}

func ddns() {
	for {
		time.Sleep(time.Second * 60)
		for syncing {
			time.Sleep(100 * time.Millisecond)
		}
		for item := range Rules.IterBuffered() {
			rid, r := item.Key, item.Val.(Rule)
			RIP, err := getIP(r.Remote)
			if err == nil && RIP != r.RIP {
				r.RIP = RIP
				stop(rid, r)
				start(rid, r)
			}
		}
	}
}
