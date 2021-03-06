/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package gin_mod

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/naiba/tunnel"
	"github.com/Unknwon/i18n"
	"github.com/gin-gonic/gin"
)

func TemplateCommonVar(ctx *gin.Context, data gin.H) gin.H {
	//模板解析时间
	ctx.Set("TemplateLoadStart", time.Now().UnixNano()/int64(time.Millisecond))

	var siteVer = tunnel.ServerVersion
	var siteName = tunnel.SiteName
	var siteAnalysis = ""
	siteLang := ctx.MustGet("Lang").(string)
	var siteDesc = tunnel.SiteDesc
	// 站点配置
	data["Site"] = map[string]interface{}{
		"Ver":      siteVer,
		"Name":     siteName,
		"Desc":     siteDesc,
		"Analysis": template.HTML(siteAnalysis),
		"Language": siteLang,
	}
	// 网站标题
	if _, has := data["Title"]; !has {
		data["Title"] = siteName + " - " + siteDesc
	} else {
		data["Title"] = data["Title"].(string) + " - " + siteName
	}
	// 系统
	data["NB"] = map[string]interface{}{
		"PageLoadStart":     ctx.GetInt64("PageLoadStart"),
		"TemplateLoadStart": ctx.GetInt64("TemplateLoadStart"),
	}
	return data
}

var TmplFuncMap = template.FuncMap{
	// 多语言
	"T": i18n.Tr,
	// 本地时间格式化
	"FormatTime": func(t time.Time) string {
		return t.In(tunnel.Loc).Format("2006-01-02 15:04")
	},
	// 乘法
	"Multi": func(a ...float64) string {
		total := 1.0000
		for _, i := range a {
			total = i * total
		}
		return strconv.FormatFloat(total, 'f', 2, 64)
	},
	// 加法
	"Add": func(a ...float64) string {
		var total float64
		for _, i := range a {
			total += i
		}
		return strconv.FormatFloat(total, 'f', 2, 64)
	},
	// 减法
	"Minus": func(a, b int64) string {
		return strconv.FormatInt(a-b, 10)
	},
	// 现在的毫秒数
	"NowMS": func() int64 {
		return time.Now().UnixNano() / int64(time.Millisecond)
	},
	"UCFirst": UCFirst,
	// HTML转义
	"Unescaped": func(x string) interface{} { return template.HTML(x) },
	// 格式化Float
	"FormatFloat": func(f float64) string { return fmt.Sprintf("%.2f", f) },
}

// 首字母大写
func UCFirst(str string) string {
	return strings.ToUpper(str[:1]) + str[1:]
}
