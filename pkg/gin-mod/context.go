/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */
package gin_mod

import (
	"strconv"
	"time"

	"github.com/naiba/tunnel"
	"github.com/naiba/tunnel/model"
	"github.com/gin-gonic/gin"
	"github.com/naiba/com"
)

type AutoOptions struct {
	NeedGuest bool
	NeedLogin bool
}

func CommonSettings(ctx *gin.Context) {
	// 页面执行时间
	ctx.Set("PageLoadStart", time.Now().UnixNano()/int64(time.Millisecond))
	// 界面语言
	siteLang, err := ctx.Cookie("lang")
	if err != nil {
		siteLang = tunnel.SiteLanguage
	}
	ctx.Set("Lang", siteLang)
	// 设置Header
	ctx.Header("X-Owner", "NB(1@5.nu)")
	// Next
	ctx.Next()
}

func AuthMiddleware(ctx *gin.Context) {
	t, err := ctx.Cookie("type")
	if err == nil {
		switch t {
		case "client":
			serial, err1 := ctx.Cookie("serial")
			token, err2 := ctx.Cookie("token")
			if err1 != nil || err2 != nil {
				return
			}
			var cl model.Client
			cl.Serial = serial
			if cl.Get() == nil {
				if com.MD5(cl.Serial+strconv.Itoa(time.Now().Year())+cl.Pass) == token {
					ctx.Set("loginType", "client")
					ctx.Set("loginClient", &cl)
				}
			}
			break
		case "user":
			break
		}
	}
}

func ToggleAuthOption(opt *AutoOptions, pass, deny func(ctx *gin.Context)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if opt.NeedGuest && ctx.GetString("loginType") != "" {
			// 登陆成功回调
			pass(ctx)
			ctx.Abort()
		}
		if opt.NeedLogin && ctx.GetString("loginType") == "" {
			// 登陆失败回调
			deny(ctx)
			ctx.Abort()
		}
	}
}

func SetCookie(ctx *gin.Context, key string, val string) {
	ctx.SetCookie(key, val, 60*60*24*365*1.5, "/", tunnel.SiteDomain, false, true)
}
