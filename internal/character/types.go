package character

import "time"

type Character struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Metadata    string     `json:"metadata"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type FileRef struct {
	FileID    string    `json:"file_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type CharacterWithFiles struct {
	Character Character `json:"character"`
	Files     []FileRef `json:"files"`
}

type CreateCharacterRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
}

type UpdateCharacterRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Metadata    *string `json:"metadata"`
}

type AddFileRequest struct {
	FileID string `json:"file_id" binding:"required"`
	Role   string `json:"role"`
}
