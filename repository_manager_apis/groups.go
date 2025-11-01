package repository_manager_apis

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// handleCreateGroup handles POST /apis/v1/repos (for groups)
// @Summary Create a group (deprecated - use type field in POST /repos instead)
// @Description Create a new group/namespace. This handler is called internally when type="group" in the create repository endpoint
// @Tags Groups
// @Accept json
// @Produce json
// @Param body body CreateRepositoryRequest true "Group creation request"
// @Success 201 {object} repository_manager.Group "Group created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 500 {object} ErrorResponse "Failed to create group"
// @Router /apis/v1/repos [post]
func (m *RepositoryManagerAPIs) handleCreateGroup(c *gin.Context) {
	var req CreateRepositoryRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	group, err := m.params.RepositoryManager.CreateGroup(req.Name, req.Description)
	if err != nil {
		m.logger.Error("Failed to create group", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

// handleListGroups handles GET /apis/v1/repos/groups
// @Summary List all groups
// @Description Get a list of all groups/namespaces
// @Tags Groups
// @Produce json
// @Success 200 {array} repository_manager.Group "List of groups"
// @Failure 500 {object} ErrorResponse "Failed to list groups"
// @Router /apis/v1/repos/groups [get]
func (m *RepositoryManagerAPIs) handleListGroups(c *gin.Context) {
	groups, err := m.params.RepositoryManager.ListGroups()
	if err != nil {
		m.logger.Error("Failed to list groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// handleGetGroup handles GET /apis/v1/repos/*name (when it's a group)
// @Summary Get group information
// @Description Get detailed information about a specific group. Supports multi-level paths like "org/team"
// @Tags Groups
// @Produce json
// @Param name path string true "Group name (supports multi-level paths)" example:"myorg"
// @Success 200 {object} repository_manager.Group "Group information"
// @Failure 404 {object} ErrorResponse "Group not found"
// @Router /apis/v1/repos/{name} [get]
func (m *RepositoryManagerAPIs) handleGetGroup(c *gin.Context) {
	// Extract group name from path parameter
	name := strings.TrimPrefix(c.Param("name"), "/")

	group, err := m.params.RepositoryManager.GetGroup(name)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

// handleDeleteGroup handles DELETE /apis/v1/repos/*name (when it's a group)
// @Summary Delete a group
// @Description Delete an empty group. The group must be empty (contain no repositories or subgroups) to be deleted. Supports multi-level paths like "org/team"
// @Tags Groups
// @Produce json
// @Param name path string true "Group name (supports multi-level paths)" example:"myorg"
// @Success 200 {object} MessageResponse "Group deleted successfully"
// @Failure 500 {object} ErrorResponse "Failed to delete group (may not be empty)"
// @Router /apis/v1/repos/{name} [delete]
func (m *RepositoryManagerAPIs) handleDeleteGroup(c *gin.Context) {
	// Extract group name from path parameter
	name := strings.TrimPrefix(c.Param("name"), "/")

	if err := m.params.RepositoryManager.DeleteGroup(name); err != nil {
		m.logger.Error("Failed to delete group", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Group deleted successfully"})
}
