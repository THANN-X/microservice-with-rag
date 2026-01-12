package gormhelper

import (
	"time"

	"gorm.io/gorm"
)

func TimeToGormDeletedAt(t *time.Time) gorm.DeletedAt {
	if t != nil {
		return gorm.DeletedAt{Time: *t, Valid: true}
	}
	return gorm.DeletedAt{}
}

func GormDeletedAtToTime(g *gorm.DeletedAt) *time.Time {
	if g.Valid {
		return &g.Time
	}
	return nil
}
