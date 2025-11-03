package git_http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// handleGitProtocolOrAPI handles Git HTTP protocol requests
// Supports multi-level repository paths.
// Routes:
// - /repos/username/repo.git/* -> Git protocol (handleGitProtocol)
func (m *GitHTTP) handleGitProtocolOrAPI(c *gin.Context) {
	// Extract full path after URL prefix
	// c.Param("path") returns path with leading slash, e.g., "/username/repo.git/info/refs" or "/hello.git/info/refs"
	fullPath := strings.TrimPrefix(c.Param("path"), "/")

	// Check if this is a Git protocol request (contains .git)
	if strings.Contains(fullPath, ".git/") || strings.HasSuffix(fullPath, ".git") {
		m.handleGitProtocol(c, fullPath)
		return
	}

	// Not a valid Git protocol path
	m.logger.Warn("Invalid git protocol path",
		zap.String("fullPath", fullPath),
		zap.String("expected", "path should contain .git (e.g., hello.git/info/refs)"),
	)
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid git protocol path, expected path with .git"})
}

// handleGitProtocol handles all Git HTTP protocol requests by delegating to gitkit
// Supports multi-level paths like "username/repo.git/info/refs"
func (m *GitHTTP) handleGitProtocol(c *gin.Context, fullPath string) {
	// Extract repository name from path (remove .git suffix and everything after)
	// Examples:
	// - "username/repo.git/info/refs" -> "username/repo"
	// - "hello.git/info/refs" -> "hello"
	// - "org/team/project.git" -> "org/team/project"
	var repoName string
	var gitPath string

	if idx := strings.Index(fullPath, ".git/"); idx != -1 {
		repoName = fullPath[:idx]
		gitPath = fullPath[idx+4:] // Everything after ".git/"
	} else if strings.HasSuffix(fullPath, ".git") {
		repoName = strings.TrimSuffix(fullPath, ".git")
		gitPath = ""
	} else {
		m.logger.Error("Invalid git protocol path format",
			zap.String("fullPath", fullPath),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid git protocol path"})
		return
	}

	// Verify repository exists
	_, err := m.params.RepositoryManager.GetRepository(repoName)
	if err != nil {
		m.logger.Warn("Repository not found for git operation",
			zap.String("repoName", repoName),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   fmt.Sprintf("repository not found: %s", repoName),
			"hint":    fmt.Sprintf("Create repository first: POST /apis/v1/repos with {\"name\": \"%s\"}", repoName),
			"repoAPI": "/apis/v1/repos",
		})
		return
	}

	// Use http.StripPrefix to remove url_prefix, then delegate to gitkit
	// This ensures gitkit sees clean paths like /hello.git/info/refs
	//
	// Example:
	//   - url_prefix = "/git/repos"
	//   - Original request: /git/repos/hello.git/info/refs
	//   - After StripPrefix: /hello.git/info/refs
	//   - gitkit finds repo at: data/git-repos/hello.git ✓

	originalPath := c.Request.URL.Path

	m.logger.Info("Delegating to gitkit",
		zap.String("repo", repoName),
		zap.String("method", c.Request.Method),
		zap.String("urlPrefix", m.urlPrefix),
		zap.String("originalPath", originalPath),
		zap.String("gitPath", gitPath),
		zap.String("service", c.Query("service")),
		zap.String("queryString", c.Request.URL.RawQuery),
	)

	// Use StripPrefix to remove url_prefix before passing to gitkit
	handler := http.StripPrefix(m.urlPrefix, m.gitService)
	handler.ServeHTTP(c.Writer, c.Request)
}
