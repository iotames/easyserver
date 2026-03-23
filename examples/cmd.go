package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/iotames/easyserver/httpsvr"
	"github.com/iotames/easyserver/response"
	"github.com/iotames/easyserver/tcpsvr"
)

var wsvr *tcpsvr.Server

var cmdLastExeAt time.Time

// /user/cmd?do=sync&token=xxxxxxxx
func execmd(ctx httpsvr.Context) {
	var err error
	// postdata := map[string]string{}
	// err = ctx.GetPostJson(postdata)
	// 解析JSON失败json.Unmarshal error: json: Unmarshal(non-pointer map[string]string)
	var postdata map[string]string
	err = ctx.GetPostJson(&postdata)
	if err != nil {
		ctx.Writer.Write(response.NewApiDataFail(err.Error(), 500).Bytes())
		return
	}
	optname, ok := postdata["optname"]
	if ok {
		// 5分钟内不要重复提交
		if optname == "userlist" {
			if time.Since(cmdLastExeAt) < time.Minute {
				// http状态码设置为400也可以工作 ctx.Json(map[string]any{"status": http.StatusBadRequest, "msg": "请求已提交, 5分钟后再试"}, 400)
				ctx.Json(map[string]any{"status": http.StatusBadRequest, "msg": "请求已提交, 5分钟后再试"}, 200)
				return
			}
			cmdLastExeAt = time.Now()
		}
		err = execByName(optname)
		if err != nil {
			log.Printf("---execmd--postdata(%+v)-error(%+v)----\n", postdata, err)
			ctx.Json(map[string]any{"code": 500, "msg": "获取命令列表错误：" + err.Error()}, 200)
			// ctx.Writer.Write(response.NewApiDataServerError(err.Error()).Bytes())
			return
		}
	} else {
		ctx.Writer.Write(response.NewApiDataFail("参数错误", 400).Bytes())
		return
	}
	if optname == "userlist" {
		ctx.Writer.Write(response.NewApiDataOk("点击“提交”后3-5分钟执行完成，请耐心等待！").Bytes())
		return
	}
	ctx.Writer.Write(response.NewApiDataOk("执行成功").Bytes())
}

func execByName(optname string) error {
	var cmd *exec.Cmd
	var cmdinfos []CmdInfo
	var err error
	var connWriters []io.Writer
	// wsvr := tcpserver.GetServer()
	if wsvr != nil {
		connWriters = wsvr.GetOutputWriters()
	}

	cmdinfos, err = GetCmds()
	if err != nil {
		return err
	}
	for _, cmdinfo := range cmdinfos {
		if cmdinfo.Name == optname {
			cmd = exec.Command(cmdinfo.Cmd, cmdinfo.Args...)
			if cmdinfo.Type == TYPE_CMD_SHELL {
				// ping 二进制帧 echo 文本帧
				switch runtime.GOOS {
				case "windows":
					cmd = exec.Command("cmd", "/c", cmdinfo.Cmd)
				case "linux":
					cmd = exec.Command("/bin/bash", "-c", cmdinfo.Cmd)
				default:
					cmd = exec.Command("/bin/bash", "-c", cmdinfo.Cmd)
				}
			}
		}
	}

	if cmd == nil {
		return fmt.Errorf("not found cmdname: %s", optname)
	}
	timeText := time.Now().Format("2006-01-02 15:04:05")
	cmd.Env = append(os.Environ(), "NOW_TIME="+timeText)

	// 标准输出：同时写入本地 stdout 和所有连接
	stdoutWriters := append([]io.Writer{os.Stdout}, connWriters...)
	cmd.Stdout = io.MultiWriter(stdoutWriters...)

	// 标准错误：同时写入本地 stderr 和所有连接
	stderrWriters := append([]io.Writer{os.Stderr}, connWriters...)
	cmd.Stderr = io.MultiWriter(stderrWriters...)

	// 启动命令
	err = cmd.Start()
	if err != nil {
		return err
	}

	// 关键修复：启动一个协程来等待命令结束，以回收系统资源，避免僵尸进程
	go func() {
		waitErr := cmd.Wait()
		if waitErr != nil {
			// 此处可以记录日志，但不应影响主流程返回
			log.Printf("---execByName--命令等待结束时报错(%v)---\n", waitErr)
		}
	}()

	return nil // 主函数立即返回，不阻塞HTTP请求
}
