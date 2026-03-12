package gormhelper

import (
	"time"

	"gorm.io/gorm"
)

// TimeToGormDeletedAt converts a *time.Time to gorm.DeletedAt.
func TimeToGormDeletedAt(t *time.Time) gorm.DeletedAt {
	if t != nil {
		return gorm.DeletedAt{Time: *t, Valid: true}
	}
	return gorm.DeletedAt{}
}

// GormDeletedAtToTime converts a gorm.DeletedAt to *time.Time.
func GormDeletedAtToTime(g *gorm.DeletedAt) *time.Time {
	if g.Valid {
		return &g.Time
	}
	return nil
}
