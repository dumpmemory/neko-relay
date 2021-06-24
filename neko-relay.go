package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"neko-relay/config"
	"neko-relay/relay"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

var (
	Config config.CONF
)

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
	flag.IntVar(&Config.Dns.Timeout, "dns_timeout", 2000, "dns timeout (ms)")

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
		str, _ := json.MarshalIndent(Config, "", "    ")
		fmt.Println(string(str))
	}
	Config.Debug = Debug
	if show_version {
		fmt.Println("neko-relay v1.4.4")
		fmt.Println("TCP & UDP & WS TUNNEL & WSS TUNNEL & Tunnel Mux & HTTP & HTTPS & STAT")
		return
	}
	if !Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	relay.Config = Config
	relay.GetCert()
	r := gin.New()
	r.POST("/ping", Ping)
	r.GET("/ping", Ping)
	datapath := "/data"
	if Config.Key != "" {
		datapath = "/data/" + Config.Key
	}
	r.GET(datapath, GetData)
	if Config.Debug && Config.Key != "" {
		r.Use(CheckKey)
	}
	r.POST("/traffic", PostTraffic)
	r.POST("/add", PostAdd)
	r.POST("/edit", PostEdit)
	r.POST("/restart", PostRestart)
	r.POST("/del", PostDel)
	r.POST("/sync", PostSync)
	r.GET("/stat", Stat)
	go Init()
	fmt.Println("Api port:", Config.Port)
	fmt.Println("Api key:", Config.Key)
	r.Run(":" + strconv.Itoa(Config.Port))
}
