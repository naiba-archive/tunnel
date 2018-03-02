/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package manager

import (
	"git.cm/naiba/tunnel/model"
	"net"
	"fmt"
	"log"
	"io"
	"strings"
	"time"
	"sync"
	"os"
)

const timeout = 5

type Service struct {
	Tunnels   map[uint]model.Tunnel
	Conns     map[uint]net.Conn
	Listeners map[uint]net.Listener
	Tuns      map[uint]net.Listener
}

func NewService() *Service {
	return &Service{make(map[uint]model.Tunnel), make(map[uint]net.Conn), make(map[uint]net.Listener), make(map[uint]net.Listener)}
}

func (s *Service) Update(tunnels []model.Tunnel, callback func(t model.Tunnel)) {
	for _, t := range tunnels {
		if st, has := s.Tunnels[t.ID]; has {
			if t.IsEqual(st) {
				continue
			}
		}
		go callback(t)
	}
}

func (s *Service) ServeLocalAddr(t model.Tunnel) {
	if c, has := s.Conns[t.ID]; has {
		c.Close()
		delete(s.Conns, t.ID)
	}

	for {
		log.Println("[+]", "try to connect host:["+fmt.Sprintf("localhost:%d", t.Port)+"] and ["+t.LocalAddr+"]")
		var host1, host2 net.Conn
		var err error
		for {
			host1, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", t.Port))
			if err == nil {
				log.Println("[→]", "connect ["+fmt.Sprintf("localhost:%d", t.Port)+"] success.")
				break
			} else {
				log.Println("[x]", "connect target address ["+fmt.Sprintf("localhost:%d", t.Port)+"] faild. retry in ", timeout, " seconds. ")
				time.Sleep(timeout * time.Second)
			}
		}
		for {
			host2, err = net.Dial("tcp", t.LocalAddr)
			if err == nil {
				log.Println("[→]", "connect ["+t.LocalAddr+"] success.")
				break
			} else {
				log.Println("[x]", "connect target address ["+t.LocalAddr+"] faild. retry in ", timeout, " seconds. ")
				time.Sleep(timeout * time.Second)
			}
		}
		forward(host1, host2)
	}
}

func (s *Service) ServeOpenAddr(t model.Tunnel) {
	listen1 := start_server(fmt.Sprintf("0.0.0.0:%d", t.Port))
	listen2 := start_server(fmt.Sprintf("0.0.0.0:%d", t.OpenAddr))
	log.Println("[√]", "listen port:", fmt.Sprintf("%d", t.Port), "and", fmt.Sprintf("%d", t.OpenAddr), "success. waiting for client...")
	for {
		conn1 := accept(listen1, t.ClientSerial, true)
		conn2 := accept(listen2, t.ClientSerial, false)
		if conn1 == nil || conn2 == nil {
			log.Println("[x]", "accept client faild. retry in ", timeout, " seconds. ")
			time.Sleep(timeout * time.Second)
			continue
		}
		forward(conn1, conn2)
	}
}

func (s *Service) ServeTun(tun net.Listener, t *model.Tunnel, local bool) {
	for {
		conn, err := tun.Accept()
		if err != nil {
			panic(err)
			continue
		}

		log.Println("建立连接", t.ClientSerial, conn.RemoteAddr().Network(), conn.RemoteAddr().String(), conn.LocalAddr().String())

		if local {
			// 如果是穿透连接
			clc, has := SC().Conns[t.ClientSerial]
			rip := strings.Split(conn.RemoteAddr().String(), ":")
			cip := strings.Split(clc.RemoteAddr().String(), ":")
			if !has || rip[0] != cip[0] {
				log.Println("非客户端连接", t.ClientSerial, cip, rip)
				conn.Close()
				continue
			}
			log.Println("建立连接", t.ClientSerial)
			s.Conns[t.ID] = conn
		} else {
			// 如果是公开连接
			c, has := s.Conns[t.ID]
			if has {
				log.Println("建立连接", conn.LocalAddr().String())
				go transferTun(conn, c)
			} else {
				log.Println("客户端不在线", t.ClientSerial, conn.RemoteAddr().String())
				conn.Close()
				continue
			}
		}
	}
}

func transferTun(src, dist net.Conn) {
	go func() {
		log.Println(io.Copy(src, dist))
	}()
	log.Println(io.Copy(dist, src))
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	log.Printf("[+] start transmit. [%s],[%s] <-> [%s],[%s] \n", conn1.LocalAddr().String(), conn1.RemoteAddr().String(), conn2.LocalAddr().String(), conn2.RemoteAddr().String())
	var wg sync.WaitGroup
	// wait tow goroutines
	wg.Add(2)
	go connCopy(conn1, conn2, &wg)
	go connCopy(conn2, conn1, &wg)
	//blocking when the wg is locked
	wg.Wait()
}

func connCopy(conn1 net.Conn, conn2 net.Conn, wg *sync.WaitGroup) {
	//TODO:log, record the data from conn1 and conn2.
	logFile := openLog(conn1.LocalAddr().String(), conn1.RemoteAddr().String(), conn2.LocalAddr().String(), conn2.RemoteAddr().String())
	if logFile != nil {
		w := io.MultiWriter(conn1, logFile)
		io.Copy(w, conn2)
	} else {
		io.Copy(conn1, conn2)
	}
	conn1.Close()
	log.Println("[←]", "close the connect at local:["+conn1.LocalAddr().String()+"] and remote:["+conn1.RemoteAddr().String()+"]")
	//conn2.Close()
	//log.Println("[←]", "close the connect at local:["+conn2.LocalAddr().String()+"] and remote:["+conn2.RemoteAddr().String()+"]")
	wg.Done()
}

func start_server(address string) net.Listener {
	log.Println("[+]", "try to start server on:["+address+"]")
	server, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalln("[x]", "listen address ["+address+"] faild.")
	}
	log.Println("[√]", "start listen at address:["+address+"]")
	return server
}

func accept(listener net.Listener, serial string, isTun bool) net.Conn {
	conn, err := listener.Accept()
	if err != nil {
		log.Println("[x]", "accept connect ["+conn.RemoteAddr().String()+"] faild.", err.Error())
		return nil
	}
	log.Println("[√]", "accept a new client. remote address:["+conn.RemoteAddr().String()+"], local address:["+conn.LocalAddr().String()+"]")
	if isTun {
		rip := strings.Split(SC().Conns[serial].RemoteAddr().String(), ":")
		rip2 := strings.Split(conn.RemoteAddr().String(), ":")
		if rip[0] != rip2[0] {
			log.Println("[x]", "ban fake client. remote address:["+conn.RemoteAddr().String()+"], local address:["+conn.LocalAddr().String()+"]")
			return nil
		}
	}
	return conn
}

func openLog(address1, address2, address3, address4 string) *os.File {
	args := os.Args
	argc := len(os.Args)
	var logFileError error
	var logFile *os.File
	if argc > 5 && args[4] == "-log" {
		address1 = strings.Replace(address1, ":", "_", -1)
		address2 = strings.Replace(address2, ":", "_", -1)
		address3 = strings.Replace(address3, ":", "_", -1)
		address4 = strings.Replace(address4, ":", "_", -1)
		timeStr := time.Now().Format("2006_01_02_15_04_05") // "2006-01-02 15:04:05"
		logPath := args[5] + "/" + timeStr + args[1] + "-" + address1 + "_" + address2 + "-" + address3 + "_" + address4 + ".log"
		logPath = strings.Replace(logPath, `\`, "/", -1)
		logPath = strings.Replace(logPath, "//", "/", -1)
		logFile, logFileError = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE, 0666)
		if logFileError != nil {
			log.Fatalln("[x]", "log file path error.", logFileError.Error())
		}
		log.Println("[√]", "open test log file success. path:", logPath)
	}
	return logFile
}
