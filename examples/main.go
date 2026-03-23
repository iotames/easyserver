package main

import (
	"log"
	"os"
	"strings"

	e "github.com/iotames/easyserver"
	"github.com/iotames/easyserver/custom"
	"github.com/iotames/easyserver/hotswap"
)

var ss *e.Server

func main() {
	ss = e.NewServer(":1212")
	args := os.Args
	log.Println("args=", strings.Join(args, "|"))
	if len(args) == 1 {
		setApi(ss)
		return
	}
	switch args[1] {
	case "tcp":
		runTcpSvr(333)
	case "api":
		setApi(ss)
	case "middle":
		setMiddle(ss)
	default:
		log.Println("args[1]=", args[1])
	}
	ss.ListenAndServe()
	if err := ss.ListenAndServe(); err != nil {
		panic(err)
	}
}

func init() {
	initScript()
}

func initScript() {
	hotswap.GetScriptDir(hotswap.NewScriptDir(custom.GetFs(), "runtime"))
}
