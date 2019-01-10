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
	"github.com/naiba/tunnel/model"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

type ClientConnect struct {
	ID         string
	C          net.Conn
	W          *sync.WaitGroup
	LastActive time.Time
	Tunnels    map[uint]*STunnel
}
type STunnel struct {
	Tunnel model.Tunnel
	LL     net.Listener
	RL     net.Listener
}

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

func IOCopyWithWaitGroup(remote, local net.Conn, wg *sync.WaitGroup) {
	io.Copy(remote, local)
	remote.Close()
	wg.Done()
}

func ServerTunnelHotUpdate(serial string, del bool) {
	var ts []model.Tunnel
	if err := model.DB().Model(&model.Client{Serial: serial}).Related(&ts, "client_serial").Error; err != nil {
		return
	}
	//清除取消的转发
DEL:
	for id, st := range OnlineClients[serial].Tunnels {
		for _, t := range ts {
			if t.ID == st.Tunnel.ID {
				if !del && t.IsEqual(st.Tunnel) {
					continue DEL
				}
			}
		}
		if st.LL != nil {
			st.LL.Close()
			st.LL = nil
		}
		if st.RL != nil {
			st.RL.Close()
			st.RL = nil
		}
		log.Println("[Pending CloseTunnel]", st.Tunnel.ID, st.Tunnel.LocalAddr)
		delete(OnlineClients[serial].Tunnels, id)
	}
	if !del {
		for _, t := range ts {
			if _, has := OnlineClients[serial].Tunnels[t.ID]; !has {
				var st STunnel
				st.Tunnel = t
				OnlineClients[serial].Tunnels[t.ID] = &st
				log.Println("[Pending CreateTunnel]", st.Tunnel.ID, st.Tunnel.LocalAddr)
				go Listener2Listener(OnlineClients[serial].Tunnels[t.ID])
			}
		}
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
	lListener, err := kcp.Listen("0.0.0.0:" + strconv.Itoa(st.Tunnel.Port))
	if err != nil {
		log.Println("监听失败,tunnelID:", st.Tunnel.ID, err)
		return
	}
	st.LL = lListener
	st.RL = rListener
	log.Println("[OK CreateTunnel]", st.Tunnel.ID, st.Tunnel.OpenAddr, st.Tunnel.LocalAddr)
	for {

		if st.LL == nil || st.RL == nil {
			log.Println("[OK CloseTunnel]", st.Tunnel.ID, st.Tunnel.OpenAddr, st.Tunnel.LocalAddr)
			return
		}

		lConn, err := lListener.Accept()
		if err != nil {
			log.Println("Error CreateTunnel:", st.Tunnel.ID, err)
			continue
		}
		sRConn, err2 := smux.Client(lConn, smux.DefaultConfig())
		if err2 != nil {
			log.Println("Error CreateTunnel:", st.Tunnel.ID, err2)
			continue
		}
		errOpenStreamRetry := 0
		for {
			if st.LL == nil || st.RL == nil {
				sRConn.Close()
				lConn.Close()
				log.Println("[OK CloseTunnel]", st.Tunnel.ID, st.Tunnel.OpenAddr, st.Tunnel.LocalAddr)
				return
			}
			// 定义OpenStream的错误次数
			if errOpenStreamRetry > 3 {
				sRConn.Close()
				log.Println("[Error ConnectTunnel]", st.Tunnel.ID, st.Tunnel.OpenAddr, st.Tunnel.LocalAddr)
				break
			}
			rConn, err := rListener.Accept()
			if err != nil {
				log.Println("[Error ConnectTunnel]", err)
				time.Sleep(time.Second * 1)
				continue
			}
			log.Println("[OK Vistor connected]", rConn.RemoteAddr().String())
			go func() {
				sc, err := sRConn.OpenStream()
				if err != nil {
					errOpenStreamRetry++
					return
				}
				log.Println("[OK Tunnel connected]", sc.RemoteAddr().String())
				errOpenStreamRetry = 0
				var wg sync.WaitGroup
				wg.Add(2)
				go IOCopyWithWaitGroup(sc, rConn, &wg)
				go IOCopyWithWaitGroup(rConn, sc, &wg)
				wg.Wait()
				log.Println("[OK Tunnel closed]", sc.RemoteAddr().String())
				log.Println("[OK Vistor closed]", rConn.RemoteAddr().String())
			}()
		}
	}
}
