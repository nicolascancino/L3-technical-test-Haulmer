package tables

import "time"

type Card struct {
	ID             string    `gorm:"column:id;type:uuid;primaryKey"`
	CardHolderName string    `gorm:"column:card_holder_name;not null"`
	Brand          string    `gorm:"column:brand;not null"`
	PANHash        string    `gorm:"column:pan_hash;not null;uniqueIndex"`
	Last4          string    `gorm:"column:last4;type:char(4);not null"`
	ExpMonth       int       `gorm:"column:exp_month;not null"`
	ExpYear        int       `gorm:"column:exp_year;not null"`
	IdempotencyKey string    `gorm:"column:idempotency_key"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (Card) TableName() string { return "cards" }
