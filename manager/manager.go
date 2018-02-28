/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package manager

import (
	"github.com/xtaci/kcp-go"
	"log"
	"encoding/gob"
)

const CodeReg = 1001
const CodeRegSuccess = 1002
const CodeLogin = 1003
const CodeLoginSuccess = 1004
const CodeNeedLogin = 1005
const CodeUpdateForwardCTable = 2001
const CodeError = 5000

type Pocket struct {
	Code int
	Data string
}

func Start() {
	sl, err := kcp.Listen(":4040")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := sl.Accept()
		if err != nil {
			log.Println("连接失败", err)
			continue
		}
		// 建立客户端
		var c ConClient
		c.Conn = conn
		c.Encoder = gob.NewEncoder(conn)
		c.Decoder = gob.NewDecoder(conn)
		go c.handlerConn()
	}
}
