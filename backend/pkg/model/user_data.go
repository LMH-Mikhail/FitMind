package model

import "time"

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	PasswordHash string     `json:"passwordHash"`
	Nickname     string     `json:"nickname"`
	AvatarURL    string     `json:"avatarUrl"`
	Gender       string     `json:"gender"`
	Status       string     `json:"status"`
	LastLoginAt  *time.Time `json:"lastLoginAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}
