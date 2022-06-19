// Package entity ...
package entity

import (
	"fmt"
	"time"
)

// Version информация о версии
type Version struct {
	Major    int `json:"major,omitempty"`
	Minor    int `json:"minor,omitempty"`
	Patch    int `json:"patch,omitempty"`
	Revision int `json:"revision,omitempty"`
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Revision)
}

// Состояние файла при выдаче дифа
const (
	FileCreated  = "new"      // новый файл
	FileRemoved  = "removed"  // удаленный файл
	FileModified = "modified" // измененный файл
)

// FileInfo информация о файле
type FileInfo struct {
	Name     string `json:"name,omitempty"`
	Checksum string `json:"checksum,omitempty"`
	Status   string `json:"status,omitempty"`
	Data     []byte `json:"-"`
	DataID   uint32 `json:"oid,omitempty"`
}

// UpdateInfo информация об обновлении
type UpdateInfo struct {
	ID         uint64     `json:"id,omitempty"`
	CreateTime time.Time  `json:"createTime,omitempty"`
	BuildTime  time.Time  `json:"buildTime,omitempty"`
	Channel    string     `json:"channel,omitempty"`
	Version    Version    `json:"version,omitempty"`
	Info       string     `json:"info,omitempty"`
	Enabled    bool       `json:"enabled,omitempty"`
	Files      []FileInfo `json:"files,omitempty"`
}
