/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package dashboard

import (
	"github.com/gin-gonic/gin"
	"github.com/naiba/tunnel/pkg/gin-mod"
	"github.com/naiba/tunnel/model"
	"math/rand"
	"encoding/json"
	"errors"
	"github.com/naiba/tunnel/tun"
	"sync"
	"time"
)

func Serial(ctx *gin.Context) {
	serial := ctx.Param("serial")
	lt := ctx.GetString("loginType")
	var cl *model.Client
	switch lt {
	case "client":
		cl = ctx.MustGet("loginClient").(*model.Client)
		if cl.Serial != serial {
			gin_mod.JSAlertRedirect("您没有权限管理此终端", "/", ctx)
			return
		}
		break
	default:
		gin_mod.JSAlertRedirect("非法登陆", "/", ctx)
		return
	}

	oc, has := tun.OnlineClients[cl.Serial]
	if !has {
		gin_mod.JSAlertRedirect("设备不在线，无法管理", "/", ctx)
		return
	}

	type tunnelManage struct {
		ID        uint   `form:"id"`
		LocalAddr string `form:"local"`
		Act       string `form:"act"`
	}

	var tl tunnelManage
	if ctx.BindQuery(&tl) == nil && tl.Act != "" {
		switch tl.Act {
		case "add":
			if len(tl.LocalAddr) < 4 {
				gin_mod.JSAlertRedirect("内网地址不正确", ctx.GetHeader("Referer"), ctx)
				return
			}
			var t model.Tunnel
			t.LocalAddr = tl.LocalAddr
			t.ClientSerial = cl.Serial
			rand.Seed(time.Now().Unix())
			t.Port = rand.Intn(10000) + 20000
			t.OpenAddr = rand.Intn(10000) + 10000
			if t.Create(model.DB()) != nil {
				gin_mod.JSAlertRedirect("数据库错误", ctx.GetHeader("Referer"), ctx)
				return
			}
			break
		case "update":
			if tl.ID < 1 {
				gin_mod.JSAlertRedirect("ID错误", ctx.GetHeader("Referer"), ctx)
				return
			}
			t, err := checkIdBelongsToClient(tl.ID, cl)
			if err != nil {
				gin_mod.JSAlertRedirect("ID错误", ctx.GetHeader("Referer"), ctx)
				return
			}
			t.LocalAddr = tl.LocalAddr
			if t.Update(model.DB()) != nil {
				gin_mod.JSAlertRedirect("数据库错误", ctx.GetHeader("Referer"), ctx)
				return
			}
			break
		case "delete":
			if tl.ID < 1 {
				gin_mod.JSAlertRedirect("ID错误", ctx.GetHeader("Referer"), ctx)
				return
			}
			t, err := checkIdBelongsToClient(tl.ID, cl)
			if err != nil {
				gin_mod.JSAlertRedirect("ID错误", ctx.GetHeader("Referer"), ctx)
				return
			}

			if model.DB().Delete(&t).Error != nil {
				gin_mod.JSAlertRedirect("数据库错误", ctx.GetHeader("Referer"), ctx)
				return
			}
			break
		default:
			gin_mod.JSAlertRedirect("未知操作", ctx.GetHeader("Referer"), ctx)
			return
		}
		model.DB().Model(cl).Related(&cl.Tunnels, "client_serial")
		data, _ := json.Marshal(cl.Tunnels)
		// 客户端更新
		var wg sync.WaitGroup
		tun.SendData(oc.C, tun.CodeGetTuns, data, &wg)
		// 服务端更新
		tun.ServerTunnelHotUpdate(cl.Serial, false)
		gin_mod.JSAlertRedirect("操作成功", ctx.GetHeader("Referer"), ctx)
		return
	} else {
		model.DB().Model(cl).Related(&cl.Tunnels, "client_serial")
		ctx.HTML(200, "dashboard/serial", gin_mod.TemplateCommonVar(ctx, gin.H{
			"Title":  "终端管理",
			"Client": cl,
		}))
	}
}

func checkIdBelongsToClient(id uint, cl *model.Client) (model.Tunnel, error) {
	var t model.Tunnel
	t.ID = id
	t.ClientSerial = cl.Serial
	if t.Get() != nil {
		return t, errors.New("记录不存在")
	}
	return t, nil
}
