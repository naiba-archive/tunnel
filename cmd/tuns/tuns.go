/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package main

import (
	"github.com/urfave/cli"
	"os"
	"git.cm/naiba/tunnel/manager"
	"git.cm/naiba/tunnel/model"
	"git.cm/naiba/tunnel"
	"git.cm/naiba/tunnel/cmd/web"
)

func main() {
	c := cli.NewApp()
	c.Name = "奶爸TUN服务端"
	c.Author = "奶爸"
	c.Email = "1@5.nu"

	c.Flags = []cli.Flag{
		cli.BoolFlag{
			Name: "web",
		},
	}
	c.Version = tunnel.ServerVersion
	c.Action = handlerCMD

	if err := c.Run(os.Args); err != nil {
		panic(err)
	}
}

func handlerCMD(ctx *cli.Context) {
	if ctx.Bool("web") {
		go web.RunServer()
	}
	go manager.SC().Serve()
	service := manager.NewService()
	var ts []model.Tunnel
	model.DB().Model(model.Tunnel{}).Find(&ts)
	service.Update(ts, service.ServeOpenAddr)
	select {}
}

func init() {
	model.Migrate()
}
