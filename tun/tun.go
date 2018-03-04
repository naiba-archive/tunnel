/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package tun

import (
	"log"
	"sync"
	"strconv"
	"net"
	"fmt"
	"time"
	"io"
	"git.cm/naiba/tunnel/model"
	"strings"
)

type ClientConnect struct {
	ID         string
	C          net.Conn
	W          *sync.WaitGroup
	LastActive time.Time
}
type STunnel struct {
	Tunnel model.Tunnel
	LL     net.Listener
	RL     net.Listener
}

var STunnels map[uint]*STunnel              // 穿透中的端口
var OnlineClients map[string]*ClientConnect // 在线的客户端
var CodePingPong byte = 0                   //心跳包
var CodeRegister byte = 1                   //注册设备
var CodeRegisterErr byte = 2                //注册设备失败
var CodeLogin byte = 3                      //登录
var CodeLoginNeed byte = 4                  //需要登录
var CodeGetTuns byte = 5                    //请求tunnels
var CodeError byte = 100                    //注册设备
func init() {
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("[奶爸]")
}
func ReceivePockets(cc *ClientConnect, server net.Conn, logger *log.Logger, callback func(c *ClientConnect, what byte, data []byte)) {
	for {
		cc.LastActive = time.Now()
		logReceivePocket := func(msg string, err error) {
			errorMsg := ""
			if err != nil {
				errorMsg = err.Error()
			}
			logger.Println("[Error]",
				fmt.Sprintf("%s %s <-- %s : %s",
					msg,
					server.RemoteAddr().String(),
					server.LocalAddr().String(),
					errorMsg))
			cc.W.Done()
		}
		// 取第一包
		pocket := make([]byte, 1800)
		num, err := server.Read(pocket)
		if err != nil || num < 11 {
			logReceivePocket("接收数据", err)
			return
		}
		what := pocket[0]
		size, err := strconv.Atoi(string(pocket[1:11]))
		data := pocket[11:num]
		if err != nil || size < num-11 {
			logReceivePocket("校验数据", err)
			return
		}
		// 装包
		for all := num - 11; all < size; {
			pocket := make([]byte, 1800)
			num, err := server.Read(pocket)
			if err != nil {
				logReceivePocket("接收数据", err)
				return
			}
			data = append(data, pocket[:num]...)
			all += num
		}
		// 回调
		go callback(cc, what, data)
	}
}

func SendData(conn net.Conn, what byte, data []byte, wg *sync.WaitGroup) error {
	wg.Add(1)
	size := len(data)
	var send []byte
	send = []byte{what}
	send = append(send, []byte(fmt.Sprintf("%010d", size))...)
	send = append(send, data...)
	_, err := conn.Write(send)
	wg.Done()
	return err
}

func IOCopyWithWaitGroup(remote, local net.Conn, close bool, wg *sync.WaitGroup) {
	io.Copy(remote, local)
	if close {
		remote.Close()
	}
	wg.Done()
}
func UpdateSTunnels() {
	var ts []model.Tunnel
	if err := model.DB().Find(&ts).Error; err != nil {
		return
	}
	//清除取消的转发
DEL:
	for id, st := range STunnels {
		for _, t := range ts {
			if t.ID == st.Tunnel.ID {
				continue DEL
			}
		}
		if st.LL != nil {
			st.LL.Close()
		}
		if st.RL != nil {
			st.RL.Close()
		}
		delete(STunnels, id)
	}
	for _, t := range ts {
		if st, has := STunnels[t.ID]; has {
			if st.Tunnel.IsEqual(t) {
				continue
			} else {
				st.RL.Close()
				st.LL.Close()
				STunnels[t.ID].Tunnel = t
			}
		} else {
			var st STunnel
			st.Tunnel = t
			STunnels[t.ID] = &st
		}
		go Listener2Listener(STunnels[t.ID])
	}
}

func Listener2Listener(st *STunnel) {
	addr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(st.Tunnel.OpenAddr))
	if err != nil {
		log.Println("监听失败,tunnelID:", st.Tunnel.ID, err)
		return
	}
	rListener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Println("监听失败,tunnelID:", st.Tunnel.ID, err)
		return
	}
	lAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(st.Tunnel.Port))
	lListener, err := net.ListenTCP("tcp", lAddr)
	if err != nil {
		log.Println("监听失败,tunnelID:", st.Tunnel.ID, err)
		return
	}
	st.LL = lListener
	st.RL = rListener
	log.Println("[OpenTunnel]", st.Tunnel.ID, st.Tunnel.OpenAddr, st.Tunnel.LocalAddr)
	for {

		t, has := STunnels[st.Tunnel.ID]
		if !has || !t.Tunnel.IsEqual(st.Tunnel) {
			log.Println("[CloseTunnel]", st.Tunnel.ID, st.Tunnel.OpenAddr, st.Tunnel.LocalAddr)
			rListener.Close()
			lListener.Close()
			return
		}

		rConn, err := rListener.Accept()
		if err != nil {
			log.Println("链接失败,tunnelID:", st.Tunnel.ID, err)
			return
		} else {
			log.Println("[RConnect]", "0.0.0.0:"+strconv.Itoa(st.Tunnel.OpenAddr), rConn.RemoteAddr().String())
		}

		lConn, err := lListener.Accept()
		if err != nil {
			log.Println("链接失败,tunnelID:", st.Tunnel.ID, err)
			return
		} else {
			x1 := strings.Split(lConn.RemoteAddr().String(), ":")
			x2 := strings.Split(OnlineClients[st.Tunnel.ClientSerial].C.RemoteAddr().String(), ":")
			if x1[0] != x2[0] {
				log.Println("客户端认证失败,tunnelID:", st.Tunnel.ID, x1[0], x2[0])
				return
			}
			log.Println("[LConnect]", "0.0.0.0:"+strconv.Itoa(st.Tunnel.Port), lConn.RemoteAddr().String())
		}

		go handlerConn(st, rConn, lConn)
	}
}

func handlerConn(st *STunnel, rConn net.Conn, lConn net.Conn) {
	log.Println("[OpenConn]", st.Tunnel.OpenAddr, rConn.RemoteAddr().String(), lConn.RemoteAddr().String())
	var wg sync.WaitGroup
	wg.Add(2)
	go IOCopyWithWaitGroup(rConn, lConn, true, &wg)
	go IOCopyWithWaitGroup(lConn, rConn, true, &wg)
	wg.Wait()
	log.Println("[CloseConn]", st.Tunnel.OpenAddr, rConn.RemoteAddr().String())
}
