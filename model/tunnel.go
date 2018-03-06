/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package model

import (
	"github.com/jinzhu/gorm"
)

const ProtocolHTTP = 1
const ProtocolTCP = 2
const ProtocolUDP = 3

type Tunnel struct {
	gorm.Model
	Client       Client `gorm:"foreignkey:ClientSerial"`
	ClientSerial string `gorm:"INDEX"`
	Protocol     int
	LocalAddr    string
	OpenAddr     int    `gorm:"unique"`
	Port         int    `gorm:"unique"`
	Traffic      int64
}

func (t *Tunnel) IsEqual(t1 Tunnel) bool {
	return t.ID == t1.ID &&
		t.ClientSerial == t1.ClientSerial &&
		t.LocalAddr == t1.LocalAddr &&
		t.Port == t1.Port &&
		t.Protocol == t1.Protocol &&
		t.OpenAddr == t1.OpenAddr
}

func (t *Tunnel) Create(db *gorm.DB) error {
	return db.Create(t).Error
}

func (t *Tunnel) Get() error {
	return DB().Where(t).First(t).Error
}

func (t *Tunnel) Update(db *gorm.DB) error {
	return db.Model(t).Update(t).Error
}
