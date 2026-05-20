package provider

import "time"

// ModelType enumerates the supported model categories.
type ModelType string

const (
	ModelTypeVideo ModelType = "video"
	ModelTypeText  ModelType = "text"
	ModelTypeAudio ModelType = "audio"
	ModelTypeImage ModelType = "image"
)

var validModelTypes = map[ModelType]bool{
	ModelTypeVideo: true,
	ModelTypeText:  true,
	ModelTypeAudio: true,
	ModelTypeImage: true,
}

func IsValidModelType(t string) bool {
	return validModelTypes[ModelType(t)]
}

type Provider struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type Model struct {
	ID                  string     `json:"id"`
	ProviderID          string     `json:"provider_id"`
	Name                string     `json:"name"`
	ModelType           string     `json:"model_type"`
	APIKey              string     `json:"api_key"`
	URL                 string     `json:"url"`
	Endpoint            string     `json:"endpoint"`
	AccessKeyID         string     `json:"access_key_id"`
	SecretAccessKey     string     `json:"secret_access_key"`
	DefaultAssetGroupID string     `json:"default_asset_group_id"`
	ProjectName         string     `json:"project_name"`
	ProjectNumber       string     `json:"project_number"`
	Active              bool       `json:"active"`
	Favorite            bool       `json:"favorite"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at"`
}

type ModelWithProvider struct {
	Model
	ProviderName string `json:"provider_name"`
}

type ProviderWithModels struct {
	Provider Provider `json:"provider"`
	Models   []Model  `json:"models"`
}

type CreateProviderRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateProviderRequest struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}

type CreateModelRequest struct {
	ProviderID          string `json:"provider_id" binding:"required"`
	Name                string `json:"name" binding:"required"`
	ModelType           string `json:"model_type"`
	APIKey              string `json:"api_key" binding:"required"`
	URL                 string `json:"url" binding:"required"`
	Endpoint            string `json:"endpoint" binding:"required"`
	AccessKeyID         string `json:"access_key_id"`
	SecretAccessKey     string `json:"secret_access_key"`
	DefaultAssetGroupID string `json:"default_asset_group_id"`
	ProjectName         string `json:"project_name"`
	ProjectNumber       string `json:"project_number"`
	Active              *bool  `json:"active"`
}

type UpdateModelRequest struct {
	Name                *string `json:"name"`
	ModelType           *string `json:"model_type"`
	APIKey              *string `json:"api_key"`
	URL                 *string `json:"url"`
	Endpoint            *string `json:"endpoint"`
	AccessKeyID         *string `json:"access_key_id"`
	SecretAccessKey     *string `json:"secret_access_key"`
	DefaultAssetGroupID *string `json:"default_asset_group_id"`
	ProjectName         *string `json:"project_name"`
	ProjectNumber       *string `json:"project_number"`
	Active              *bool   `json:"active"`
}
