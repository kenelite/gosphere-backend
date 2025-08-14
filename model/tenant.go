package model

import "time"

type Tenant struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;uniqueIndex" json:"name"`
	Namespace   string    `gorm:"size:128;uniqueIndex" json:"namespace"`
	Description string    `gorm:"size:512" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
