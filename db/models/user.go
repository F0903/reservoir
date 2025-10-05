package models

import (
	"reservoir/utils/phc"
	"time"
)

type User struct {
	ID                     int64
	Username               string    `db:"username"`
	PasswordHash           phc.PHC   `db:"password_hash"`
	PasswordChangeRequired bool      `db:"password_change_required"`
	CreatedAt              time.Time `db:"created_at"`
	UpdatedAt              time.Time `db:"updated_at"`
}
