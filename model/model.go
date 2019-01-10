/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package model

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/naiba/tunnel"
)

var dbInstance *gorm.DB

func DB() *gorm.DB {
	if dbInstance == nil {
		var err error
		dbInstance, err = gorm.Open("sqlite3", tunnel.ServerDBPath+"?_loc="+tunnel.Loc.String())
		if err != nil {
			panic(err)
		}
		dbInstance.LogMode(tunnel.Debug)
	}
	return dbInstance
}

func Migrate() {
	DB().AutoMigrate(Client{}, Tunnel{})
}
