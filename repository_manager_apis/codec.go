package repository_manager_apis

// CreateRepositoryRequest represents the request body for creating a repository or group
// @Description Request body for creating a repository or group
type CreateRepositoryRequest struct {
	Name        string `json:"name" binding:"required" example:"myorg/myrepo"`
	Description string `json:"description" example:"My awesome repository"`
	IsPrivate   bool   `json:"is_private" example:"false"`
	Type        string `json:"type" example:"repository" enums:"repository,group"`
} // @name CreateRepositoryRequest

// CreateTagRequest represents the request body for creating a tag
// @Description Request body for creating a Git tag
type CreateTagRequest struct {
	TagName    string `json:"tag_name" binding:"required" example:"v1.0.0"`
	CommitHash string `json:"commit_hash" example:"abc123def456"`
	Message    string `json:"message" example:"Release version 1.0.0"`
	Tagger     string `json:"tagger" example:"John Doe"`
} // @name CreateTagRequest

// ErrorResponse represents an error response
// @Description Error response body
type ErrorResponse struct {
	Error string `json:"error" example:"repository not found"`
} // @name ErrorResponse

// MessageResponse represents a success message response
// @Description Success message response
type MessageResponse struct {
	Message string `json:"message" example:"Repository deleted successfully"`
} // @name MessageResponse
