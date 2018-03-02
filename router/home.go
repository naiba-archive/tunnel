/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package router

import (
	"github.com/gin-gonic/gin"
	"git.cm/naiba/tunnel/pkg/gin-mod"
	"git.cm/naiba/tunnel/model"
	"time"
	"strconv"
	"git.cm/naiba/com"
)

func Home(ctx *gin.Context) {
	ctx.HTML(200, "home", gin_mod.TemplateCommonVar(ctx, gin.H{}))
}

func LoginHandler(ctx *gin.Context) {
	type LoginForm struct {
		Serial   string `form:"serial" binding:"required,alphanum,len=25"`
		Password string `form:"password" binding:"required,alphanum,len=16"`
	}
	var lf LoginForm
	if err := ctx.ShouldBind(&lf); err != nil {
		gin_mod.JSAlertRedirect("序列号或密码填写有误", "/", ctx)
		return
	}
	var cl model.Client
	cl.Serial = lf.Serial
	cl.Pass = lf.Password
	if cl.Get() != nil {
		gin_mod.JSAlertRedirect("序列号或密码错误", "", ctx)
		return
	}
	gin_mod.SetCookie(ctx, "type", "client")
	gin_mod.SetCookie(ctx, "serial", cl.Serial)
	gin_mod.SetCookie(ctx, "token", com.MD5(cl.Serial+strconv.Itoa(time.Now().Year())+cl.Pass))
	gin_mod.JSAlertRedirect("登陆成功", "/dashboard/"+cl.Serial, ctx)
}
