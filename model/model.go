/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package model

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"time"
)

var dbInstance *gorm.DB
var Debug = true
var Loc time.Location

func DB() *gorm.DB {
	if dbInstance == nil {
		var err error
		dbInstance, err = gorm.Open("sqlite3", "NB.db?_loc="+Loc.String())
		if err != nil {
			panic(err)
		}
		dbInstance.LogMode(Debug)
	}
	return dbInstance
}

func Migrate() {
	DB().AutoMigrate(Client{}, Tunnel{})
}
