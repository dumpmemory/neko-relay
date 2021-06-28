package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"neko-relay/relay"
	. "neko-relay/rules"
	"net"
	"strconv"
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
	svr, err := relay.NewRelay(rid, r, 30, 10, getTF(rid), r.Type)
	if err != nil {
		return
	}
	Svrs.Set(rid, svr)
	err = svr.Serve()
	if err != nil {
		return
	}
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
		// return
		syncing = false
		time.Sleep(time.Duration(Config.Dns.Timeout+10) * time.Millisecond)
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
		if r.RIP == "" {
			rip, err := getIP(r.Remote)
			if err != nil {
				continue
			}
			r.RIP = rip
		}
		err := check(r)
		if err == nil {
			Rules.Set(rid, r)
			start(rid, r)
		}
	}
	syncing = false
	if Config.Syncfile != "" {
		data, err := Rules.MarshalJSON()
		if err == nil {
			err = ioutil.WriteFile(Config.Syncfile, data, 0644)
		}
		if err != nil {
			log.Println(err)
		}
	}
}

var Rsvr = net.DefaultResolver

func getIP(host string) (string, error) {
	ips, err := Rsvr.LookupHost(context.Background(), host)
	// ips, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	return ips[0], nil
}

func ddns() {
	for {
		time.Sleep(time.Second * 30)
		for syncing {
			time.Sleep(100 * time.Millisecond)
		}
		for item := range Rules.IterBuffered() {
			rid, r := item.Key, item.Val.(Rule)
			RIP, err := getIP(r.Remote)
			if err == nil && RIP != r.RIP {
				r.RIP = RIP
				svr, _ := Svrs.Get(rid)
				Svr := svr.(*relay.Relay)
				Svr.Raddr = r.RIP + ":" + strconv.Itoa(r.Rport)
				Rules.Set(rid, r)
				// Svrs.Set(rid, Svr)
				// restart(rid, r)
			}
		}
	}
}

func Init() {
	if Config.Dns.Nameserver != "" {
		Rsvr = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(Config.Dns.Timeout) * time.Millisecond,
				}
				return d.DialContext(ctx, Config.Dns.Network, Config.Dns.Nameserver+":53")
			},
		}
	}
	if Config.Tsp.Ws > 0 {
		relay.WsMuxTunnelServer.Serve(Config.Tsp.Ws)
	}
	if Config.Tsp.Wss > 0 {
		relay.WssMuxTunnelServer.Serve(Config.Tsp.Wss)
	}
	if Config.Syncfile != "" {
		data, err := ioutil.ReadFile(Config.Syncfile)
		if err == nil {
			newRules := make(map[string]Rule)
			json.Unmarshal(data, &newRules)
			sync(newRules)
		} else {
			log.Println(err)
		}
	}
	go ddns()
}
