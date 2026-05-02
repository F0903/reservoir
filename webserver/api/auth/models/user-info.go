package models

import (
	dbmodels "reservoir/db/models"
	"time"
)

type UserInfo struct {
	ID                     int64     `json:"id"`
	Username               string    `json:"username"`
	IsAdmin                bool      `json:"is_admin"`
	PasswordChangeRequired bool      `json:"password_change_required"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func FromUser(user *dbmodels.User) UserInfo {
	return UserInfo{
		ID:                     user.ID,
		Username:               user.Username,
		IsAdmin:                true,
		PasswordChangeRequired: user.PasswordChangeRequired,
		CreatedAt:              user.CreatedAt,
		UpdatedAt:              user.UpdatedAt,
	}
}
