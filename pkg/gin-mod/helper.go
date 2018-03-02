/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package gin_mod

import (
	"github.com/gin-gonic/gin"
	"html"
)

func JSAlertRedirect(msg, url string, ctx *gin.Context) {
	ctx.Writer.WriteString(`
<script>
alert('` + html.EscapeString(msg) + `');
window.location.href='` + url + `';
</script>
`)
}
