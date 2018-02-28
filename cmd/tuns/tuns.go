/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package main

import (
	"github.com/urfave/cli"
	"os"
	"log"
	"git.cm/naiba/tunnel/manager"
	"git.cm/naiba/tunnel/model"
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
	c.Action = handlerCMD

	if err := c.Run(os.Args); err != nil {
		panic(err)
	}
}

func handlerCMD(ctx *cli.Context) {
	if ctx.Bool("web") {
		log.Println("web")
	}
	go manager.Start()
	select {}
}

func init() {
	model.Migrate()
}
