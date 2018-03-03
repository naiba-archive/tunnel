/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package main

import (
	"log"
	"os"
	"github.com/xtaci/kcp-go"
	"time"
	"sync"
	"git.cm/naiba/tunnel/tun"
	"io/ioutil"
	"net"
	"git.cm/naiba/tunnel/model"
	"encoding/json"
	"git.cm/naiba/tunnel"
	"strconv"
)

var Registered bool
var logger *log.Logger
var loginData []byte

type CTunnel struct {
	Tunnel model.Tunnel
	LC     net.Conn
	RC     net.Conn
}

var CTunnels map[uint]*CTunnel

func init() {
	logger = log.New(os.Stdout, "[奶爸] ", log.Lmicroseconds)
	logger.Println("=== 奶爸 Tunnel Client ===")
	CTunnels = make(map[uint]*CTunnel)
}

func main() {
	logger.Println(" |- 加载配置文件...")
	if _, err := os.Stat("NB"); err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		} else {
			logger.Println(" |- 初次使用注册...")
		}
	} else {
		loginData, err = ioutil.ReadFile("NB")
		if err != nil {
			panic(err)
		}
		var cl model.Client
		if json.Unmarshal(loginData, &cl) != nil {
			logger.Println("[致命错误] 账户信息解析失败")
			panic(err)
		}
		logger.Println(" |========================================")
		logger.Println(" |- 序列号：" + cl.Serial)
		logger.Println(" |- 密码：" + cl.Pass)
		logger.Println(" |========================================")
		logger.Println(" |- 正在接入云端...")
		Registered = true

	}

	// 服务器通信
	for {
		server, err := kcp.Dial(tunnel.SiteDomain + ":7235")
		if err != nil {
			logger.Println("服务器通信失败，5 秒后重新连接", err)
			time.Sleep(time.Second * 5)
			continue
		}
		logger.Println("[√]连接服务器:", server.RemoteAddr().String())
		var wg sync.WaitGroup
		wg.Add(1)
		// 接收数据
		go tun.ReceivePockets(&tun.ClientConnect{W: &wg, C: server}, server, logger, handlerReceive)
		// 建立连接
		go pingPong(server, &wg)
		// 注册或登录
		if Registered {
			go login(server, &wg)
		} else {
			go tun.SendData(server, tun.CodeRegister, []byte{}, &wg)
		}
		wg.Wait()
		server.Close()
		logger.Println("[X]服务器断开:", server.RemoteAddr().String())
		time.Sleep(time.Second * 3)
	}
}

func handlerReceive(cc *tun.ClientConnect, what byte, data []byte) {
	switch what {
	case tun.CodePingPong:
		logger.Println("来自服务器的问候")
		break
	case tun.CodeError:
		logger.Println("服务器返回错误：", string(data))
		break
	case tun.CodeRegisterErr:
		logger.Println("注册失败：", string(data))
		tun.SendData(cc.C, tun.CodeRegister, []byte{}, cc.W)
		break
	case tun.CodeRegister:
		logger.Println("注册成功", string(data))
		err := ioutil.WriteFile("NB", data, os.ModePerm)
		if err != nil {
			panic(err)
		}
		tun.SendData(cc.C, tun.CodeGetTuns, []byte{}, cc.W)
		break
	case tun.CodeLogin:
		logger.Println("登陆成功")
		tun.SendData(cc.C, tun.CodeGetTuns, []byte{}, cc.W)
		break
	case tun.CodeGetTuns:
		// 更新转发链接
		var ts []model.Tunnel
		if json.Unmarshal(data, &ts) != nil {
			log.Println(string(data))
			tun.SendData(cc.C, tun.CodeGetTuns, []byte{}, cc.W)
		}
		updateCTunnelConnect(ts)
		break
	case tun.CodeLoginNeed:
		logger.Println("需要登录")
		cc.C.Close()
		break
	default:
		logger.Println("未知协议", what, string(data))
	}
}

func login(conn net.Conn, wg *sync.WaitGroup) {
	tun.SendData(conn, tun.CodeLogin, loginData, wg)
}

func pingPong(conn net.Conn, wg *sync.WaitGroup) {
	for {
		if tun.SendData(conn, tun.CodePingPong, []byte{}, wg) != nil {
			conn.Close()
			return
		}
		time.Sleep(time.Second * 50)
	}
}

func updateCTunnelConnect(ts []model.Tunnel) {
	// 关闭已删除转发
DEL:
	for id, ct := range CTunnels {
		for _, t := range ts {
			if t.ID == id {
				continue DEL
			}
		}
		if ct.RC != nil {
			ct.RC.Close()
		}
		if ct.LC != nil {
			ct.LC.Close()
		}
		delete(CTunnels, id)
	}
	// 更新转发列表
	for _, t := range ts {
		// 检查是否有更改
		if ct, has := CTunnels[t.ID]; has {
			if ct.Tunnel.IsEqual(t) {
				continue
			}
			ct.LC.Close()
			ct.RC.Close()
			ct.Tunnel = t
		} else {
			var ct CTunnel
			ct.Tunnel = t
			CTunnels[t.ID] = &ct
		}

		// 转发
		switch t.Protocol {
		default:
			go TCPHost2Host(CTunnels[t.ID])
			break
		}
	}
}

type Num struct {
	sync.RWMutex
	num int
}

func TCPHost2Host(t *CTunnel) {
	var n Num
	n.RWMutex.Lock()
	// 可支持100人同时在线
	n.num = 100
	n.Unlock()
	for {

		n.RWMutex.Lock()
		if n.num < 0 {
			n.Unlock()
			time.Sleep(time.Second)
			continue
		} else {
			n.num -= 1
			n.Unlock()
		}

		nt, has := CTunnels[t.Tunnel.ID]
		if !has || !nt.Tunnel.IsEqual(t.Tunnel) {
			log.Println("[ClostTunnel]", t.Tunnel.ID, t.Tunnel.LocalAddr)
			return
		}
		var rConn, lConn net.Conn
		var err error
		rAddr := tunnel.SiteDomain + ":" + strconv.Itoa(t.Tunnel.Port)
		for {
			rConn, err = net.Dial("tcp", rAddr)
			if err != nil {
				logger.Println("[Connectfail]", rAddr, err)
				continue
			} else {
				logger.Println("[RConnect]", rConn.RemoteAddr().String(), rConn.LocalAddr().String())
			}
			break
		}
		for {
			lConn, err = net.Dial("tcp", t.Tunnel.LocalAddr)
			if err != nil {
				logger.Println("[Connectfail]", rConn.LocalAddr().String(), err)
				if lConn != nil {
					lConn.Close()
				}
				return
			} else {
				logger.Println("[LConnect]", lConn.RemoteAddr().String(), lConn.LocalAddr().String())
			}
			break
		}
		go handlerConn(rConn, lConn, &n)
	}
}
func handlerConn(rConn net.Conn, lConn net.Conn, num *Num) {
	var wg sync.WaitGroup
	logger.Println("[Connected]", rConn.RemoteAddr().String(), lConn.LocalAddr().String())
	wg.Add(2)
	go tun.IOCopyWithWaitGroup(rConn, lConn, true, &wg)
	go tun.IOCopyWithWaitGroup(lConn, rConn, true, &wg)
	wg.Wait()
	num.Lock()
	num.num += 1
	num.Unlock()
	logger.Println("[Disconnected]", rConn.RemoteAddr().String(), lConn.LocalAddr().String())
}
