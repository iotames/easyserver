package main

import (
	"github.com/iotames/easyserver/hotswap"
)

// {
//     "time": "2026-03-06 10:35:00",
//     "list": [
//         {
//             "type": "shell",
//             "name": "userlist",
//             "title": "人员同步",
//             "cmd": "/home/santic/kettle_hour.sh",
//             "args": []
//         },
//         {
//             "type": "shell",
//             "name": "debug",
//             "title": "调试",
//             "cmd": "echo Hello Santic ${NOW_TIME}",
//             "args": []
//         }
//     ]
// }

const TYPE_CMD_SHELL = "shell"

type CmdInfo struct {
	Type  string   `json:"type"`
	Name  string   `json:"name"`
	Title string   `json:"title"`
	Cmd   string   `json:"cmd"`
	Args  []string `json:"args"`
}

type CmdsInfo struct {
	Type string    `json:"type"`
	List []CmdInfo `json:"list"`
}

func GetCmds() ([]CmdInfo, error) {
	sp := hotswap.GetScriptDir(nil)
	cmdsinfo := CmdsInfo{}
	err := sp.DecodeJson("cmdlist.json", &cmdsinfo)
	return cmdsinfo.List, err
}
