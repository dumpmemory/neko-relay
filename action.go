package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"neko-relay/relay"
	. "neko-relay/rules"
	"neko-relay/stat"
	"strconv"

	"github.com/gin-gonic/gin"
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
	Port, _ := strconv.Atoi(c.PostForm("port"))
	remote := c.PostForm("remote")
	Rport, _ := strconv.Atoi(c.PostForm("rport"))
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
func Ping(c *gin.Context) {
	c.Writer.WriteString("pong")
	c.Done()
}
func PostTraffic(c *gin.Context) {
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
}
func PostAdd(c *gin.Context) {
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
}
func PostEdit(c *gin.Context) {
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
}
func PostRestart(c *gin.Context) {
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
}
func PostDel(c *gin.Context) {
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
}
func PostSync(c *gin.Context) {
	newRules := make(map[string]Rule)
	data := []byte(c.PostForm("rules"))
	json.Unmarshal(data, &newRules)
	sync(newRules)
	resp(c, true, Rules, 200)
}
func Stat(c *gin.Context) {
	res, err := stat.GetStat()
	if err == nil {
		resp(c, true, res, 200)
	} else {
		resp(c, false, err, 500)
	}
}
func GetData(c *gin.Context) {
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
		"tsp": gin.H{
			"ws": gin.H{
				"port":   Config.Tsp.Ws,
				"status": relay.WsMuxTunnelServer.TCPListen != nil,
			},
			"wss": gin.H{
				"port":   Config.Tsp.Wss,
				"status": relay.WssMuxTunnelServer.TCPListen != nil,
			},
		},
		"errors":  errs,
		"working": working,
	})
}
func CheckKey(c *gin.Context) {
	if c.Request.Header.Get("key") != Config.Key {
		resp(c, false, "Api key Incorrect", 500)
		c.Abort()
		return
	}
	c.Next()
}
