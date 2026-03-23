package main

import (
	"fmt"

	"github.com/iotames/easyserver/tcpsvr"
)

func runTcpSvr(port int) {
	ts := tcpsvr.NewServer(fmt.Sprintf("0.0.0.0:%d", port), 20)
	if err := ts.Run(); err != nil {
		panic(err)
	}
}
