/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package model

import (
	"time"
	"github.com/jinzhu/gorm"
)

type Client struct {
	Name       string
	Serial     string   `gorm:"PRIMARY_KEY;UNIQUE_INDEX"`
	Pass       string
	UserID     string
	Online     bool
	Tunnels    []Tunnel `gorm:"foreignkey:ClientSerial"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastActive time.Time
}

func (c *Client) Create(db *gorm.DB) error {
	return db.Create(&c).Error
}

func (c *Client) Get() error {
	return DB().Where(&c).First(&c).Error
}

func (c *Client) Update(db *gorm.DB) error {
	return db.Model(c).Update(c).Error
}
