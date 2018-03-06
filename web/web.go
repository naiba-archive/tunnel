/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package web

import (
	"github.com/gin-gonic/gin"
	"github.com/Unknwon/i18n"
	"git.cm/naiba/tunnel/router"
	"git.cm/naiba/tunnel/pkg/gin-mod"
	"net/http"
	"git.cm/naiba/tunnel"
	"git.cm/naiba/tunnel/router/dashboard"
	"git.cm/naiba/tunnel/model"
)

func RunServer() {

	if tunnel.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)

	}

	webEngine := gin.New()
	webEngine.Use(gin.Recovery())
	webEngine.SetFuncMap(gin_mod.TmplFuncMap)
	webEngine.Static("/static", "bin_data/static")
	webEngine.LoadHTMLGlob("bin_data/templates/**/*")
	webEngine.Use(gin_mod.CommonSettings)
	webEngine.Use(gin_mod.AuthMiddleware)

	needGuest := webEngine.Group("/")
	needGuest.Use(authRequired(true, false))
	{
		needGuest.GET("/", router.Home)
		needGuest.POST("/login", router.LoginHandler)
	}

	needLogin := webEngine.Group("/dashboard")
	needLogin.Use(authRequired(false, true))
	{
		needLogin.GET("/:serial", dashboard.Serial)
	}

	webEngine.GET("/language/:lang", func(ctx *gin.Context) {
		lang := ctx.Param("lang")
		availableLang := map[string]int{
			"zh-CN": 1,
			"en-US": 1,
		}
		if _, has := availableLang[lang]; has {
			gin_mod.SetCookie(ctx, "lang", lang)
		}
		ctx.Redirect(http.StatusMovedPermanently, ctx.GetHeader("Referer"))
	})

	webEngine.Run("127.0.0.1:8032")
}

func authRequired(guest, login bool) gin.HandlerFunc {
	return gin_mod.ToggleAuthOption(&gin_mod.AutoOptions{NeedGuest: guest, NeedLogin: login}, func(ctx *gin.Context) {
		gin_mod.JSAlertRedirect(i18n.Tr(ctx.MustGet("Lang").(string), "already_login"), "/dashboard/"+ctx.MustGet("loginClient").(*model.Client).Serial, ctx)
	}, func(ctx *gin.Context) {
		gin_mod.JSAlertRedirect(i18n.Tr(ctx.MustGet("Lang").(string), "need_login"), "/", ctx)
	})
}

func init() {
	// 初始化多语言
	i18n.SetMessage("en-US", "bin_data/locale/locale_en-US.ini")
	i18n.SetMessage("zh-CN", "bin_data/locale/locale_zh-CN.ini")
}
