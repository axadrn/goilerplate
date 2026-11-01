package model

import (
	"time"
)

const (
	FileTypeAvatar = "avatar"
)

type File struct {
	ID           string    `db:"id"`
	UserID       string    `db:"user_id"`    // Who owns/created this file
	OwnerType    string    `db:"owner_type"` // Entity type, such as "user"
	OwnerID      string    `db:"owner_id"`   // Polymorphic FK
	Type         string    `db:"type"`
	Filename     string    `db:"filename"`
	OriginalName string    `db:"original_name"`
	MimeType     string    `db:"mime_type"`
	Size         int64     `db:"size"`
	StoragePath  string    `db:"storage_path"`
	Public       bool      `db:"public"` // true = public files (7d expiry), false = private files (1h expiry)
	CreatedAt    time.Time `db:"created_at"`
}
