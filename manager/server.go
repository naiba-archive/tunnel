/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package manager

import (
	"net"
	"log"
	"encoding/gob"
	"time"
	"git.cm/naiba/com"
	"git.cm/naiba/tunnel/model"
	"strings"
	"encoding/json"
)

type ConnectEntry struct {
	Conn   net.Conn
	Serial string
}

type ServerController struct {
	Service *Service
	Conns   map[string]net.Conn
	Client  map[string]*model.Client
}

var sc *ServerController

func SC() *ServerController {
	if sc == nil {
		sc = &ServerController{NewService(), make(map[string]net.Conn), make(map[string]*model.Client)}
	}
	return sc
}

func (sc *ServerController) Serve() {
	x, _ := net.ResolveTCPAddr("tcp", ":4040")
	sl, err := net.ListenTCP("tcp", x)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := sl.Accept()
		if err != nil {
			log.Println("连接失败", err)
			continue
		}

		var c ConnectEntry
		c.Conn = conn
		go c.handlerConn(conn)
	}
}

func (c *ConnectEntry) handlerConn(conn net.Conn) {
	defer c.Close()
	d := gob.NewDecoder(conn)
	for {
		var data Pocket
		if err := d.Decode(&data); err != nil {
			log.Println("解码失败", err)
			return
		} else {

			go c.handlerReceive(data)
		}
	}
}

func (c *ConnectEntry) handlerReceive(rec Pocket) {
	log.Println(rec)
	if len(c.Serial) != 25 {
		// 未授权先授权
		switch rec.Code {
		case CodeReg:
			// 注册Client
			serial := com.RandString(25)
			pass := com.RandString(16)
			var ct model.Client
			ct.Serial = serial
			ct.Pass = pass
			ct.CreatedAt = time.Now()
			ct.LastActive = time.Now()
			ct.Online = true
			session := model.DB().Begin()
			if err := ct.Create(session); err != nil {
				session.Rollback()
				c.SendData(Pocket{CodeError, err.Error()})
				c.Close()
			} else {
				if err := c.SendData(Pocket{CodeRegSuccess, serial + "#" + pass}); err != nil {
					session.Rollback()
					panic(err)
					c.Close()
				}
				session.Commit()
			}
			break
		case CodeLogin:
			// 登陆
			data := strings.Split(rec.Data, "#")
			var ct model.Client
			ct.Serial = data[0]
			ct.Pass = data[1]
			if ct.Get() != nil {
				c.SendData(Pocket{CodeError, "Error serial or pass"})
				return
			}
			ct.Online = true
			ct.LastActive = time.Now()
			ct.Update(model.DB())
			// 更新连接状态
			SC().Conns[ct.Serial] = c.Conn
			SC().Client[ct.Serial] = &ct
			c.Serial = ct.Serial
			c.SendData(Pocket{CodeLoginSuccess, "Login success"})
			break
		default:
			// 需要登录
			c.SendData(Pocket{CodeNeedLogin, ""})
		}
	} else {
		// 授权后操作
		switch rec.Code {
		case CodeUpdateForwardCTable:
			// 更新转发表
			client, _ := SC().Client[c.Serial]
			if err := model.DB().Model(client).Related(&client.Tunnels, "client_serial").Error; err != nil {
				log.Println("获取转发表", err)
				return
			}
			data, err := json.Marshal(client.Tunnels)
			if err != nil {
				log.Println("序列化", err)
				return
			}
			c.SendData(Pocket{CodeUpdateForwardCTable, string(data)})
			break
		default:
			c.SendData(Pocket{CodeNeedLogin, "Error code"})
		}
	}
}

func (c *ConnectEntry) SendData(data Pocket) error {
	return gob.NewEncoder(c.Conn).Encode(data)
}

func (c *ConnectEntry) Close() {
	log.Println("断开链接", c.Serial, c.Conn.RemoteAddr())
	delete(SC().Client, c.Serial)
	delete(SC().Conns, c.Serial)
	c.Conn.Close()
}
