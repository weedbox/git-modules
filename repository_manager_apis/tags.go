package repository_manager_apis

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// handleCreateTag handles POST /apis/v1/repos/*name/tags
// @Summary Create a tag
// @Description Create a new Git tag (lightweight or annotated) for a repository. Supports multi-level repository paths like "username/repo/tags"
// @Tags Tags
// @Accept json
// @Produce json
// @Param name path string true "Repository name (supports multi-level paths)" example:"myorg/myrepo"
// @Param body body CreateTagRequest true "Tag creation request"
// @Success 201 {object} repository_manager.Tag "Tag created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 500 {object} ErrorResponse "Failed to create tag"
// @Router /apis/v1/repos/{name}/tags [post]
func (m *RepositoryManagerAPIs) handleCreateTag(c *gin.Context) {
	// Extract repository name from path parameter
	// c.Param("name") returns path with leading slash, e.g., "/username/repo"
	repoName := strings.TrimPrefix(c.Param("name"), "/")

	var req CreateTagRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	tag, err := m.params.RepositoryManager.CreateTag(repoName, req.TagName, req.CommitHash, req.Message, req.Tagger)
	if err != nil {
		m.logger.Error("Failed to create tag", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tag)
}

// handleListTags handles GET /apis/v1/repos/*name/tags
// @Summary List all tags
// @Description Get a list of all tags in a repository. Supports multi-level repository paths like "username/repo/tags"
// @Tags Tags
// @Produce json
// @Param name path string true "Repository name (supports multi-level paths)" example:"myorg/myrepo"
// @Success 200 {array} repository_manager.Tag "List of tags"
// @Failure 500 {object} ErrorResponse "Failed to list tags"
// @Router /apis/v1/repos/{name}/tags [get]
func (m *RepositoryManagerAPIs) handleListTags(c *gin.Context) {
	// Extract repository name from path parameter
	// c.Param("name") returns path with leading slash, e.g., "/username/repo"
	repoName := strings.TrimPrefix(c.Param("name"), "/")

	tags, err := m.params.RepositoryManager.ListTags(repoName)
	if err != nil {
		m.logger.Error("Failed to list tags", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// handleGetTag handles GET /apis/v1/repos/*name/tags/*tag
// @Summary Get tag information
// @Description Get detailed information about a specific tag. Supports multi-level repository paths and tag names like "username/repo/tags/release/v1.0.0"
// @Tags Tags
// @Produce json
// @Param name path string true "Repository name (supports multi-level paths)" example:"myorg/myrepo"
// @Param tag path string true "Tag name (supports multi-level paths)" example:"release/v1.0.0"
// @Success 200 {object} repository_manager.Tag "Tag information"
// @Failure 404 {object} ErrorResponse "Tag not found"
// @Router /apis/v1/repos/{name}/tags/{tag} [get]
func (m *RepositoryManagerAPIs) handleGetTag(c *gin.Context) {
	// Extract repository name from path parameter
	// c.Param("name") returns path with leading slash, e.g., "/username/repo"
	repoName := strings.TrimPrefix(c.Param("name"), "/")

	// Extract tag name from path parameter
	// c.Param("tag") returns path with leading slash, e.g., "/v1.0.0"
	tagName := strings.TrimPrefix(c.Param("tag"), "/")

	tag, err := m.params.RepositoryManager.GetTag(repoName, tagName)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, tag)
}

// handleDeleteTag handles DELETE /apis/v1/repos/*name/tags/*tag
// @Summary Delete a tag
// @Description Delete a tag from a repository. Supports multi-level repository paths and tag names like "username/repo/tags/release/v1.0.0"
// @Tags Tags
// @Produce json
// @Param name path string true "Repository name (supports multi-level paths)" example:"myorg/myrepo"
// @Param tag path string true "Tag name (supports multi-level paths)" example:"release/v1.0.0"
// @Success 200 {object} MessageResponse "Tag deleted successfully"
// @Failure 500 {object} ErrorResponse "Failed to delete tag"
// @Router /apis/v1/repos/{name}/tags/{tag} [delete]
func (m *RepositoryManagerAPIs) handleDeleteTag(c *gin.Context) {
	// Extract repository name from path parameter
	// c.Param("name") returns path with leading slash, e.g., "/username/repo"
	repoName := strings.TrimPrefix(c.Param("name"), "/")

	// Extract tag name from path parameter
	// c.Param("tag") returns path with leading slash, e.g., "/v1.0.0"
	tagName := strings.TrimPrefix(c.Param("tag"), "/")

	if err := m.params.RepositoryManager.DeleteTag(repoName, tagName); err != nil {
		m.logger.Error("Failed to delete tag", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Tag deleted successfully"})
}
