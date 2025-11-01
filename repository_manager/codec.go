package repository_manager

import (
	"time"
)

// Repository represents a Git repository
// @Description Git repository information
type Repository struct {
	Name        string    `json:"name" example:"myorg/myrepo"`
	Description string    `json:"description" example:"My awesome repository"`
	Path        string    `json:"path" example:"/path/to/repos/myorg/myrepo.git"`
	CreatedAt   time.Time `json:"created_at" example:"2025-01-01T00:00:00Z"`
} // @name Repository

// Tag represents a Git tag
// @Description Git tag information (lightweight or annotated)
type Tag struct {
	Name       string    `json:"name" example:"v1.0.0"`
	CommitHash string    `json:"commit_hash" example:"abc123def456789"`
	Message    string    `json:"message,omitempty" example:"Release version 1.0.0"`
	Tagger     string    `json:"tagger,omitempty" example:"John Doe"`
	TaggerDate time.Time `json:"tagger_date,omitempty" example:"2025-01-01T00:00:00Z"`
	Type       string    `json:"type" example:"annotated" enums:"lightweight,annotated"`
} // @name Tag

// Group represents a namespace/organization for repositories
// @Description Group/namespace for organizing repositories
type Group struct {
	Name        string    `json:"name" example:"myorg"`
	Description string    `json:"description" example:"My Organization"`
	Path        string    `json:"path" example:"/path/to/repos/myorg"`
	CreatedAt   time.Time `json:"created_at" example:"2025-01-01T00:00:00Z"`
} // @name Group
