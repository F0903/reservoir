package models

import "time"

type UserInfo struct {
	ID                     int64     `json:"id"`
	Username               string    `json:"username"`
	PasswordChangeRequired bool      `json:"password_change_required"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}
