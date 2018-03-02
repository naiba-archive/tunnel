/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package manager

import (
	"net"
	"encoding/gob"
	"log"
	"gopkg.in/ini.v1"
	"os"
	"strings"
	"git.cm/naiba/tunnel/model"
	"encoding/json"
	"git.cm/naiba/tunnel"
)

type ClientController struct {
	Service *Service
	Conn    net.Conn
	Decoder *gob.Decoder
	Encoder *gob.Encoder
	Addr    string
	Conf    *ini.File
}

func NewClientController() *ClientController {
	return &ClientController{Addr: "localhost:4040"}
}

func (s *ClientController) Connect() {
	conn, err := net.Dial("tcp", s.Addr)
	if err != nil {
		panic(err)
	}
	s.Conn = conn
	s.Decoder = gob.NewDecoder(conn)
	s.Encoder = gob.NewEncoder(conn)
	s.Service = NewService()
	s.handlerConn()
}

func (s *ClientController) Login() {
	s.Encoder.Encode(Pocket{CodeLogin, s.Conf.Section("").Key("serial").String() + "#" + s.Conf.Section("").Key("pass").String()})
}

func (s *ClientController) ReqForwardTable() {
	s.Encoder.Encode(Pocket{CodeUpdateForwardCTable, ""})
}

func (s *ClientController) handlerConn() {
	for {
		var data Pocket
		if err := s.Decoder.Decode(&data); err != nil {
			log.Println("解码失败", err)
		} else {
			go s.handlerReceive(data)
		}
	}
}

func (s *ClientController) handlerReceive(rec Pocket) {
	log.Println(rec)
	switch rec.Code {
	case CodeError:
		// 错误
		log.Fatal(rec.Data)
		break
	case CodeRegSuccess:
		// 注册成功，登陆
		data := strings.Split(rec.Data, "#")
		s.Conf.Section("").Key("serial").SetValue(data[0])
		s.Conf.Section("").Key("pass").SetValue(data[1])
		s.Conf.SaveTo(tunnel.ClientConfPath)
		s.Login()
		break
	case CodeLoginSuccess:
		// 登陆成功
		s.ReqForwardTable()
		break
	case CodeNeedLogin:
		// 需要登录
		s.Login()
		break
	case CodeUpdateForwardCTable:
		// 动态更新转发表
		var lists []model.Tunnel
		if err := json.Unmarshal([]byte(rec.Data), &lists); err != nil {
			log.Println("错误", "转发表解析", err)
			s.ReqForwardTable()
		}
		s.Service.Update(lists, s.Service.ServeLocalAddr)
		break
	}
}

func (s *ClientController) LoadConf() {
	var err error
	if _, err = os.Stat(tunnel.ClientConfPath); err != nil {
		s.Conf = ini.Empty()
	} else {
		s.Conf, err = ini.Load(tunnel.ClientConfPath)
		if err != nil {
			panic(err)
		}
	}
}
