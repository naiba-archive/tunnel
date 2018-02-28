/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package model

import "github.com/jinzhu/gorm"

type Tunnel struct {
	gorm.Model
	Client       Client `gorm:"foreignkey:ClientSerial"`
	ClientSerial string `gorm:"INDEX"`
	Protocol     int
	LocalAddr    string
	Port         int
	Traffic      int64
}
