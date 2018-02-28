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

type ConClient struct {
	Encoder  *gob.Encoder
	Decoder  *gob.Decoder
	Conn     net.Conn
	Client   *model.Client
	Accepted bool
}

func (c *ConClient) handlerConn() {
	defer c.Close()
	d := gob.NewDecoder(c.Conn)
	for {
		log.Println("解码")
		var data Pocket
		if err := d.Decode(&data); err != nil {
			log.Println("解码失败", err)
			return
		} else {
			go c.handlerReceive(data)
		}
	}
}

func (c *ConClient) handlerReceive(rec Pocket) {
	log.Println(rec)
	if !c.Accepted {
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
				c.Encoder.Encode(Pocket{CodeError, err.Error()})
				c.Close()
			} else {
				if err := c.Encoder.Encode(Pocket{CodeRegSuccess, serial + "#" + pass}); err != nil {
					session.Rollback()
					log.Println(err)
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
				c.Encoder.Encode(Pocket{CodeError, "Error serial or pass"})
				return
			}
			ct.Online = true
			c.Accepted = true
			ct.LastActive = time.Now()
			ct.Update(model.DB())
			c.Client = &ct
			c.Encoder.Encode(Pocket{CodeLoginSuccess, "Login success"})
			break
		default:
			// 需要登录
			c.Encoder.Encode(Pocket{CodeNeedLogin, ""})
		}
	} else {
		// 授权后操作
		switch rec.Code {
		case CodeUpdateForwardCTable:
			// 更新转发表
			var tunnels []model.Tunnel
			if model.DB().Model(c.Client).Related(&tunnels, "client_serial").Error != nil {
				return
			}
			data, err := json.Marshal(tunnels)
			if err != nil {
				return
			}
			c.Encoder.Encode(Pocket{CodeUpdateForwardCTable, string(data)})
			break
		default:
			c.Encoder.Encode(Pocket{CodeNeedLogin, "Error code"})
		}
	}
}

func (c *ConClient) Close() {
	log.Println("断开链接", c.Client, c.Conn.RemoteAddr())
	c.Accepted = false
	c.Client.LastActive = time.Now()
	c.Client.Online = false
	c.Client.Update(model.DB())
	c.Conn.Close()
}
