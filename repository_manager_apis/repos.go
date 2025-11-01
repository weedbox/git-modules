package repository_manager_apis

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// handleCreateRepository handles POST /apis/v1/repos
// @Summary Create a repository or group
// @Description Create a new Git repository or group/namespace
// @Tags Repositories
// @Accept json
// @Produce json
// @Param body body CreateRepositoryRequest true "Repository or Group creation request"
// @Success 201 {object} repository_manager.Repository "Repository created successfully"
// @Success 201 {object} repository_manager.Group "Group created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 500 {object} ErrorResponse "Failed to create repository or group"
// @Router /apis/v1/repos [post]
func (m *RepositoryManagerAPIs) handleCreateRepository(c *gin.Context) {
	var req CreateRepositoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Default to repository if type is not specified
	if req.Type == "" {
		req.Type = "repository"
	}

	// Create group or repository based on type
	if req.Type == "group" {
		group, err := m.params.RepositoryManager.CreateGroup(req.Name, req.Description)
		if err != nil {
			m.logger.Error("Failed to create group", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusCreated, group)
		return
	}

	// Create repository (default behavior)
	repo, err := m.params.RepositoryManager.CreateRepository(req.Name, req.Description, req.IsPrivate)
	if err != nil {
		m.logger.Error("Failed to create repository", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, repo)
}

// handleListRepositories handles GET /apis/v1/repos
// @Summary List all repositories
// @Description Get a list of all Git repositories
// @Tags Repositories
// @Produce json
// @Success 200 {array} repository_manager.Repository "List of repositories"
// @Failure 500 {object} ErrorResponse "Failed to list repositories"
// @Router /apis/v1/repos [get]
func (m *RepositoryManagerAPIs) handleListRepositories(c *gin.Context) {
	repos, err := m.params.RepositoryManager.ListRepositories()
	if err != nil {
		m.logger.Error("Failed to list repositories", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, repos)
}

// handleGetRepository handles GET /apis/v1/repos/*name
// @Summary Get repository information
// @Description Get detailed information about a specific repository. Supports multi-level paths like "username/repo" or "org/team/project"
// @Tags Repositories
// @Produce json
// @Param name path string true "Repository name (supports multi-level paths)" example:"myorg/myrepo"
// @Success 200 {object} repository_manager.Repository "Repository information"
// @Failure 404 {object} ErrorResponse "Repository not found"
// @Router /apis/v1/repos/{name} [get]
func (m *RepositoryManagerAPIs) handleGetRepository(c *gin.Context) {
	// Extract repository name from path parameter
	// c.Param("name") returns path with leading slash, e.g., "/username/repo"
	name := strings.TrimPrefix(c.Param("name"), "/")

	repo, err := m.params.RepositoryManager.GetRepository(name)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, repo)
}

// handleDeleteRepository handles DELETE /apis/v1/repos/*name
// @Summary Delete a repository
// @Description Delete a repository by name. Supports multi-level paths like "username/repo" or "org/team/project"
// @Tags Repositories
// @Produce json
// @Param name path string true "Repository name (supports multi-level paths)" example:"myorg/myrepo"
// @Success 200 {object} MessageResponse "Repository deleted successfully"
// @Failure 500 {object} ErrorResponse "Failed to delete repository"
// @Router /apis/v1/repos/{name} [delete]
func (m *RepositoryManagerAPIs) handleDeleteRepository(c *gin.Context) {
	// Extract repository name from path parameter
	// c.Param("name") returns path with leading slash, e.g., "/username/repo"
	name := strings.TrimPrefix(c.Param("name"), "/")

	if err := m.params.RepositoryManager.DeleteRepository(name); err != nil {
		m.logger.Error("Failed to delete repository", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Repository deleted successfully"})
}
