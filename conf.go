/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package tunnel

import "time"

const ServerDBPath = "NB.db"
const ServerVersion = "0.2.0beta"
const SiteName = "Tun"
const SiteDesc = "内网穿透服务"
const SiteLanguage = "en-US"

var SiteDomain = "xn--g1ao.com"

var Debug = false
var Loc *time.Location

func init() {
	Loc, _ = time.LoadLocation("Asia/Shanghai")
	if Debug {
		SiteDomain = "localhost"
	}
}
