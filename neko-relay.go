package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"neko-relay/config"
	"neko-relay/relay"
	. "neko-relay/rules"
	"neko-relay/stat"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

var (
	Config config.CONF
)

func resp(c *gin.Context, success bool, data interface{}, code int) {
	c.JSON(code, gin.H{
		"success": success,
		"data":    data,
	})
}
func check(r Rule) error {
	if r.Port > 65535 || r.Rport > 65535 {
		return errors.New("port is not in range")
	}
	return nil
}
func ParseRule(c *gin.Context) (rid string, r Rule, err error) {
	rid = c.PostForm("rid")
	port, _ := strconv.Atoi(c.PostForm("port"))
	Port := uint(port)
	remote := c.PostForm("remote")
	rport, _ := strconv.Atoi(c.PostForm("rport"))
	Rport := uint(rport)
	typ := c.PostForm("type")
	var RIP string
	RIP, err = getIP(remote)
	if err != nil {
		return
	}
	r = Rule{Port: Port, Remote: remote, RIP: RIP, Rport: Rport, Type: typ}
	err = check(r)
	return
}
func main() {
	var confpath string
	var show_version bool
	Debug := false
	flag.StringVar(&confpath, "c", "", "config")
	flag.StringVar(&Config.Key, "key", "", "api key")
	flag.IntVar(&Config.Port, "port", 8080, "api port")

	flag.StringVar(&Config.Tls.Cert, "tls_cert", "public.pem", "cert file")
	flag.StringVar(&Config.Tls.Key, "tls_key", "private.key", "key file")

	flag.StringVar(&Config.Dns.Nameserver, "dns_nameserver", "", "dns nameserver")
	flag.StringVar(&Config.Dns.Network, "dns_network", "", "dns network (udp/tcp)")

	flag.StringVar(&Config.Fake.Host, "fake_host", "", "fake host")
	flag.StringVar(&Config.Fake.Host, "fake_url", "", "fake url")

	flag.BoolVar(&Debug, "debug", false, "enable Config.Debug")
	flag.BoolVar(&show_version, "v", false, "show version")
	flag.Parse()
	if confpath != "" {
		data, err := ioutil.ReadFile(confpath)
		if err != nil {
			log.Panic(err)
		}
		err = yaml.Unmarshal([]byte(data), &Config)
		if err != nil {
			panic(err)
		}
		fmt.Println(Config.Tsp)
	}
	Config.Debug = Debug
	if show_version {
		fmt.Println("neko-relay v1.4.3")
		fmt.Println("TCP & UDP & WS TUNNEL & WSS TUNNEL & HTTP & HTTPS & STAT")
		return
	}
	if !Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	relay.Config = Config
	relay.GetCert()
	r := gin.New()
	datapath := "/data"
	if Config.Key != "" {
		datapath = "/data/" + Config.Key
	}
	r.GET(datapath, getData)
	if Config.Debug && Config.Key != "" {
		r.Use(checkKey)
	}
	r.POST("/traffic", func(c *gin.Context) {
		reset, _ := strconv.ParseBool(c.DefaultPostForm("reset", "false"))
		y := gin.H{}
		for item := range Traffic.IterBuffered() {
			rid, tf := item.Key, item.Val.(*relay.TF)
			y[rid] = tf.Total()
			if reset {
				tf.Reset()
			}
		}
		resp(c, true, y, 200)
	})
	r.POST("/add", func(c *gin.Context) {
		rid, r, err := ParseRule(c)
		if err == nil {
			Rules.Set(rid, r)
			err = start(rid, r)
			if err == nil {
				resp(c, true, nil, 200)
			} else {
				resp(c, false, err.Error(), 500)
			}
		} else {
			resp(c, false, err.Error(), 500)
			return
		}
	})
	r.POST("/edit", func(c *gin.Context) {
		rid, r, err := ParseRule(c)
		fmt.Println("edit", rid, r, err)
		if err == nil {
			stop(rid, r)
			Rules.Set(rid, r)
			err = start(rid, r)
			if err == nil {
				resp(c, true, nil, 200)
			} else {
				resp(c, false, err.Error(), 500)
			}
		} else {
			resp(c, false, err.Error(), 500)
			return
		}
	})
	r.POST("/restart", func(c *gin.Context) {
		rid := c.PostForm("rid")
		r, has := Rules.Get(rid)
		if has {
			err := restart(rid, r.(Rule))
			if err == nil {
				resp(c, true, nil, 200)
			} else {
				resp(c, false, err.Error(), 500)
			}
		} else {
			resp(c, false, "rid doesn't exit", 500)
		}
	})
	r.POST("/del", func(c *gin.Context) {
		rid := c.PostForm("rid")
		rule, has := Rules.Get(rid)
		if !has {
			resp(c, false, gin.H{
				"rule":    nil,
				"traffic": 0,
			}, 200)
			return
		}
		r := rule.(Rule)
		traffic, _ := Traffic.Get(rid)
		stop(rid, r)
		Rules.Remove(rid)
		Traffic.Remove(rid)
		resp(c, true, gin.H{
			"rule":    rule,
			"traffic": traffic,
		}, 200)
	})
	r.POST("/sync", func(c *gin.Context) {
		newRules := make(map[string]Rule)
		data := []byte(c.PostForm("rules"))
		json.Unmarshal(data, &newRules)
		if Config.Syncfile != "" {
			err := ioutil.WriteFile(Config.Syncfile, data, 0644)
			if err != nil {
				log.Println(err)
			}
		}
		sync(newRules)
		resp(c, true, Rules, 200)
	})
	r.GET("/stat", func(c *gin.Context) {
		res, err := stat.GetStat()
		if err == nil {
			resp(c, true, res, 200)
		} else {
			resp(c, false, err, 500)
		}
	})
	go Init()
	fmt.Println("Api port:", Config.Port)
	fmt.Println("Api key:", Config.Key)
	r.Run(":" + strconv.Itoa(Config.Port))
}
func checkKey(c *gin.Context) {
	if c.Request.Header.Get("key") != Config.Key {
		resp(c, false, "Api key Incorrect", 500)
		c.Abort()
		return
	}
	c.Next()
}
func getData(c *gin.Context) {
	working := Rules.Items()
	errs := make(map[string]Rule)
	for t := range Svrs.IterBuffered() {
		ok, _ := t.Val.(*relay.Relay).OK()
		if !ok {
			errs[t.Key] = working[t.Key].(Rule)
			delete(working, t.Key)
		}
	}
	c.JSON(200, gin.H{
		"syncing": syncing,
		"errors":  errs,
		"working": working,
	})
}
