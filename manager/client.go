/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package manager

import (
	"net"
	"github.com/xtaci/kcp-go"
	"encoding/gob"
	"log"
	"gopkg.in/ini.v1"
	"os"
	"strings"
)

type Server struct {
	Conn    net.Conn
	Decoder *gob.Decoder
	Encoder *gob.Encoder
	Addr    string
	Conf    *ini.File
}

const ClientConf = "nb.ini"

func DefaultServer() *Server {
	return &Server{Addr: "localhost:4040"}
}

func (s *Server) Connect() {
	conn, err := kcp.Dial(s.Addr)
	if err != nil {
		panic(err)
	}
	s.Conn = conn
	s.Decoder = gob.NewDecoder(conn)
	s.Encoder = gob.NewEncoder(conn)
	s.handlerConn()
}

func (s *Server) Login() {
	s.Encoder.Encode(Pocket{CodeLogin, s.Conf.Section("").Key("serial").String() + "#" + s.Conf.Section("").Key("pass").String()})
}

func (s *Server) ForwardTable() {
	s.Encoder.Encode(Pocket{CodeUpdateForwardCTable, ""})
}

func (s *Server) handlerConn() {
	for {
		var data Pocket
		if err := s.Decoder.Decode(&data); err != nil {
			panic(err)
		} else {
			go s.handlerReceive(data)
		}
	}
}

func (s *Server) handlerReceive(rec Pocket) {
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
		s.Conf.SaveTo(ClientConf)
		s.Login()
		break
	case CodeLoginSuccess:
		// 登陆成功
		s.ForwardTable()
		break
	case CodeNeedLogin:
		// 需要登录
		s.Login()
		break
	case CodeUpdateForwardCTable:
		// 更新转发表
		log.Println(rec)
		break
	}
}

func (s *Server) LoadConf() {
	var err error
	if _, err = os.Stat(ClientConf); err != nil {
		s.Conf = ini.Empty()
	} else {
		s.Conf, err = ini.Load(ClientConf)
		if err != nil {
			panic(err)
		}
	}
}
