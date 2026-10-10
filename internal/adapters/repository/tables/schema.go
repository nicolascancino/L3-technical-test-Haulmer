package tables

import (
	_ "embed"

	"gorm.io/gorm"
)

//go:embed schema.sql
var schema string

func Migrate(db *gorm.DB) error {
	return db.Exec(schema).Error
}
