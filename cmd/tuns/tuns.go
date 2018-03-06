/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package main

import (
	"github.com/urfave/cli"
	"os"
	"git.cm/naiba/tunnel/model"
	"git.cm/naiba/tunnel/web"
	"git.cm/naiba/tunnel"
	"log"
	"github.com/xtaci/kcp-go"
	"git.cm/naiba/tunnel/tun"
	"sync"
	"time"
	"encoding/json"
	"fmt"
	"git.cm/naiba/com"
)

var Logger *log.Logger

func init() {
	model.Migrate()
	Logger = log.New(os.Stdout, "[奶爸] ", log.Lmicroseconds)
	tun.OnlineClients = make(map[string]*tun.ClientConnect)
}

func main() {
	c := cli.NewApp()
	c.Name = "奶爸TUN服务端"
	c.Author = "奶爸"
	c.Email = "1@5.nu"

	c.Flags = []cli.Flag{
		cli.BoolFlag{
			Name: "web",
		},
	}
	c.Version = tunnel.ServerVersion
	c.Action = handlerCMD

	if err := c.Run(os.Args); err != nil {
		panic(err)
	}
}

func handlerCMD(ctx *cli.Context) {
	if ctx.Bool("web") {
		go web.RunServer()
	}
	Logger.Println(ctx.App.Name)
	listener, err := kcp.Listen("0.0.0.0:7235")
	if err != nil {
		panic(err)
	}
	// 检测客户端在线
	go checkPingPong()
	// 接入客户端
	for {
		conn, err := listener.Accept()
		if err != nil {
			Logger.Println("[X]客户端接入", err)
			continue
		}
		go func() {
			Logger.Println("[√]客户端接入:", conn.RemoteAddr().String())
			var wg sync.WaitGroup
			cc := &tun.ClientConnect{C: conn, W: &wg, LastActive: time.Now()}
			wg.Add(1)
			// 接收数据
			go tun.ReceivePockets(cc, conn, Logger, handlerReceive)
			// 心跳包
			wg.Wait()
			conn.Close()
			if _, has := tun.OnlineClients[cc.ID]; has {
				// 断开客户端相关连接
				tun.ServerTunnelHotUpdate(cc.ID, true)
				delete(tun.OnlineClients, cc.ID)
				model.DB().Model(&model.Client{Serial: cc.ID}).Update("last_active", time.Now())
			}
			Logger.Println("[X]客户端断开:", cc.ID, conn.RemoteAddr().String())
		}()
	}
}

func handlerReceive(cc *tun.ClientConnect, what byte, data []byte) {

	if len(cc.ID) < 1 && what != tun.CodeRegister && what != tun.CodeLogin && what != tun.CodePingPong {
		tun.SendData(cc.C, tun.CodeLoginNeed, []byte(""), cc.W)
		cc.C.Close()
		return
	}

	// 处理控制协议
	switch what {
	case tun.CodeGetTuns:
		// 更新Tunnels
		var ts []model.Tunnel
		if model.DB().Model(model.Tunnel{}).Where("client_serial = ?", cc.ID).Find(&ts).Error == nil {
			ts, err := json.Marshal(&ts)
			if err == nil {
				tun.SendData(cc.C, tun.CodeGetTuns, ts, cc.W)
			}
		}
		break
	case tun.CodePingPong:
		break
	case tun.CodeRegister:
		// 终端注册
		if len(cc.ID) > 1 {
			tun.SendData(cc.C, tun.CodeError, []byte("已注册，无需再次注册"), cc.W)
			return
		}
		var client model.Client
		client.Serial = com.RandString(25)
		client.Pass = com.RandString(16)
		client.LastActive = time.Now()
		ss := model.DB().Begin()
		if err := client.Create(ss); err != nil {
			tun.SendData(cc.C, tun.CodeRegisterErr, []byte(fmt.Sprintf("数据库错误 %s", err.Error())), cc.W)
			return
		}
		data, err := json.Marshal(&client)
		if err != nil {
			ss.Rollback()
			tun.SendData(cc.C, tun.CodeRegisterErr, []byte(fmt.Sprintf("序列化出错 %s", err.Error())), cc.W)
			return
		}
		if tun.SendData(cc.C, tun.CodeRegister, data, cc.W) != nil {
			ss.Rollback()
		}
		cc.ID = client.Serial
		cc.Tunnels = make(map[uint]*tun.STunnel)
		tun.OnlineClients[cc.ID] = cc
		tun.ServerTunnelHotUpdate(cc.ID, false)
		ss.Commit()
		break
	case tun.CodeLogin:
		// 终端登陆
		if len(cc.ID) > 1 {
			tun.SendData(cc.C, tun.CodeError, []byte("已登录，无需再次登陆"), cc.W)
			return
		}
		var client model.Client
		if err := json.Unmarshal(data, &client); err != nil || len(client.Pass) != 16 {
			tun.SendData(cc.C, tun.CodeError, []byte(fmt.Sprintf("数据反序列化失败 %s", err.Error())), cc.W)
			return
		}
		if err := client.Get(); err != nil {
			tun.SendData(cc.C, tun.CodeError, []byte(fmt.Sprintf("账号或密码错误 %s", err.Error())), cc.W)
			return
		}
		tun.SendData(cc.C, tun.CodeLogin, []byte{}, cc.W)
		cc.ID = client.Serial
		cc.Tunnels = make(map[uint]*tun.STunnel)
		// 登陆成功添加到在线列表
		tun.OnlineClients[cc.ID] = cc
		// 清除旧链接
		tun.ServerTunnelHotUpdate(cc.ID, true)
		// 设置新链接
		tun.ServerTunnelHotUpdate(cc.ID, false)
		break
	default:
		tun.SendData(cc.C, tun.CodeError, []byte("非法协议"), cc.W)
		cc.C.Close()
		return
	}
}

func checkPingPong() {
	for {
		for _, cc := range tun.OnlineClients {
			if cc.LastActive.Add(time.Second * 70).Before(time.Now()) {
				cc.C.Close()
			}
		}
		time.Sleep(time.Second * 10)
	}
}
