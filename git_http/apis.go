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
	// Extract full path after /repos/
	// c.Param("path") returns path with leading slash, e.g., "/username/repo.git/info/refs"
	fullPath := strings.TrimPrefix(c.Param("path"), "/")

	// Check if this is a Git protocol request (contains .git)
	if strings.Contains(fullPath, ".git/") || strings.HasSuffix(fullPath, ".git") {
		m.handleGitProtocol(c, fullPath)
		return
	}

	// Not a valid Git protocol path
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid git protocol path, expected path with .git"})
}

// handleGitProtocol handles all Git HTTP protocol requests by delegating to gitkit
// Supports multi-level paths like "username/repo.git/info/refs"
func (m *GitHTTP) handleGitProtocol(c *gin.Context, fullPath string) {
	// Extract repository name from path (remove .git suffix and everything after)
	// Examples:
	// - "username/repo.git/info/refs" -> "username/repo"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid git protocol path"})
		return
	}

	// Verify repository exists
	_, err := m.params.RepositoryManager.GetRepository(repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("repository not found: %s", repoName)})
		return
	}

	// Rewrite the path for gitkit (gitkit expects paths like: /repo.git/info/refs)
	// Original: /repos/username/repo.git/info/refs
	// Gitkit expects: /username/repo.git/info/refs
	if gitPath != "" {
		c.Request.URL.Path = "/" + repoName + ".git/" + gitPath
	} else {
		c.Request.URL.Path = "/" + repoName + ".git"
	}

	// Log the Git operation
	m.logger.Info("Git protocol request",
		zap.String("repo", repoName),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
	)

	// Delegate to gitkit handler (it's just an http.Handler)
	m.serveGitPath(c.Writer, c.Request)
}

// ServeGitHTTP delegates Git HTTP protocol handling to gitkit
func (m *GitHTTP) serveGitPath(w http.ResponseWriter, r *http.Request) {
	// gitkit.Server implements http.Handler interface
	// It handles all Git Smart HTTP protocol operations:
	// - GET  /repo.git/info/refs?service=git-upload-pack (clone discovery)
	// - POST /repo.git/git-upload-pack (clone/fetch)
	// - GET  /repo.git/info/refs?service=git-receive-pack (push discovery)
	// - POST /repo.git/git-receive-pack (push)
	m.gitService.ServeHTTP(w, r)
}
