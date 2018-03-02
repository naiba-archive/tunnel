/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package main

import (
	"git.cm/naiba/tunnel/manager"
	"time"
	"log"
	"git.cm/naiba/tunnel"
)

func main() {
	log.Println("NB Tun v" + tunnel.ClientVersion)
	// 连接服务器
	s := manager.NewClientController()
	go s.Connect()
	s.LoadConf()
	for {
		if s.Conn == nil {
			time.Sleep(time.Second * 1)
			continue
		} else {
			if len(s.Conf.Section("").Key("serial").String()) != 25 {
				// 云端注册
				if s.Encoder.Encode(manager.Pocket{manager.CodeReg, ""}) != nil {
					s.Conn.Close()
				}
			} else {
				// 登陆
				s.Login()
			}
			break
		}
	}
	select {}
}
