package repository_manager

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// Helper function to create a test RepositoryManager
func setupTestManager(t *testing.T) (*RepositoryManager, string) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "repo_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	logger, _ := zap.NewDevelopment()

	manager := &RepositoryManager{
		logger:    logger,
		reposPath: tmpDir,
	}

	return manager, tmpDir
}

// Helper function to clean up test resources
func teardownTestManager(tmpDir string) {
	os.RemoveAll(tmpDir)
}

// Test isValidRepoName function with various inputs
func TestIsValidRepoName(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		want     bool
	}{
		// Valid single-level names
		{"simple name", "myrepo", true},
		{"with dash", "my-repo", true},
		{"with underscore", "my_repo", true},
		{"with dot", "my.repo", true},
		{"with numbers", "repo123", true},

		// Valid multi-level names
		{"two levels", "username/repo", true},
		{"three levels", "group/project/repo", true},
		{"complex path", "org/team/project/repo", true},

		// Invalid names - empty or special
		{"empty", "", false},
		{"dot only", ".", false},
		{"dotdot only", "..", false},

		// Invalid names - path traversal
		{"path traversal", "../repo", false},
		{"path traversal mid", "user/../repo", false},
		{"dotdot segment", "user/..", false},
		{"dot segment", "user/./repo", false},

		// Invalid names - slashes
		{"leading slash", "/repo", false},
		{"trailing slash", "repo/", false},
		{"double slash", "user//repo", false},
		{"backslash", "user\\repo", false},

		// Invalid names - special characters
		{"with space", "my repo", false},
		{"with @", "user@repo", false},
		{"with #", "repo#1", false},
		{"with &", "repo&test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRepoName(tt.repoName)
			if got != tt.want {
				t.Errorf("isValidRepoName(%q) = %v, want %v", tt.repoName, got, tt.want)
			}
		})
	}
}

// Test creating repositories with single-level path
func TestCreateRepository_SingleLevel(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer teardownTestManager(tmpDir)

	repo, err := manager.CreateRepository("test-repo", "Test repository", false)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	if repo.Name != "test-repo" {
		t.Errorf("Expected repo name 'test-repo', got '%s'", repo.Name)
	}

	// Verify repository directory exists
	expectedPath := filepath.Join(tmpDir, "test-repo.git")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Repository directory does not exist at %s", expectedPath)
	}
}

// Test creating repositories with multi-level path
func TestCreateRepository_MultiLevel(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer teardownTestManager(tmpDir)

	testCases := []struct {
		name        string
		description string
	}{
		{"user1/repo1", "User repository"},
		{"org/team/project", "Team project repository"},
		{"company/dept/team/app", "Deeply nested repository"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo, err := manager.CreateRepository(tc.name, tc.description, false)
			if err != nil {
				t.Fatalf("Failed to create repository '%s': %v", tc.name, err)
			}

			if repo.Name != tc.name {
				t.Errorf("Expected repo name '%s', got '%s'", tc.name, repo.Name)
			}

			// Verify repository directory exists
			expectedPath := filepath.Join(tmpDir, tc.name+".git")
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				t.Errorf("Repository directory does not exist at %s", expectedPath)
			}

			// Verify parent directories were created
			parentDir := filepath.Dir(expectedPath)
			if _, err := os.Stat(parentDir); os.IsNotExist(err) {
				t.Errorf("Parent directory does not exist at %s", parentDir)
			}
		})
	}
}

// Test listing repositories with multi-level structure
func TestListRepositories_MultiLevel(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer teardownTestManager(tmpDir)

	// Create multiple repositories at different levels
	repoNames := []string{
		"simple",
		"user1/repo1",
		"user1/repo2",
		"user2/repo1",
		"org/team1/project1",
		"org/team2/project2",
	}

	for _, name := range repoNames {
		_, err := manager.CreateRepository(name, "Test repo: "+name, false)
		if err != nil {
			t.Fatalf("Failed to create repository '%s': %v", name, err)
		}
	}

	// List all repositories
	repos, err := manager.ListRepositories()
	if err != nil {
		t.Fatalf("Failed to list repositories: %v", err)
	}

	if len(repos) != len(repoNames) {
		t.Errorf("Expected %d repositories, got %d", len(repoNames), len(repos))
	}

	// Verify all created repositories are in the list
	repoMap := make(map[string]bool)
	for _, repo := range repos {
		repoMap[repo.Name] = true
	}

	for _, name := range repoNames {
		if !repoMap[name] {
			t.Errorf("Repository '%s' not found in list", name)
		}
	}
}

// Test getting a specific repository
func TestGetRepository_MultiLevel(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer teardownTestManager(tmpDir)

	name := "user/team/project"
	description := "Multi-level test repository"

	// Create repository
	_, err := manager.CreateRepository(name, description, false)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Get repository
	repo, err := manager.GetRepository(name)
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if repo.Name != name {
		t.Errorf("Expected repo name '%s', got '%s'", name, repo.Name)
	}

	if repo.Description != description {
		t.Errorf("Expected description '%s', got '%s'", description, repo.Description)
	}
}

// Test deleting a repository with multi-level path
func TestDeleteRepository_MultiLevel(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer teardownTestManager(tmpDir)

	name := "org/project/repo"

	// Create repository
	_, err := manager.CreateRepository(name, "Test repo", false)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Verify it exists
	repoPath := filepath.Join(tmpDir, name+".git")
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		t.Fatalf("Repository was not created at %s", repoPath)
	}

	// Delete repository
	err = manager.DeleteRepository(name)
	if err != nil {
		t.Fatalf("Failed to delete repository: %v", err)
	}

	// Verify it's deleted
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Errorf("Repository still exists at %s after deletion", repoPath)
	}
}

// Test creating duplicate repository
func TestCreateRepository_Duplicate(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer teardownTestManager(tmpDir)

	name := "user/repo"

	// Create first repository
	_, err := manager.CreateRepository(name, "First", false)
	if err != nil {
		t.Fatalf("Failed to create first repository: %v", err)
	}

	// Try to create duplicate
	_, err = manager.CreateRepository(name, "Second", false)
	if err == nil {
		t.Error("Expected error when creating duplicate repository, got nil")
	}
}

// Test creating repository with invalid name
func TestCreateRepository_InvalidName(t *testing.T) {
	manager, tmpDir := setupTestManager(t)
	defer teardownTestManager(tmpDir)

	invalidNames := []string{
		"",
		"../etc/passwd",
		"user//repo",
		"/absolute/path",
		"trailing/slash/",
		"with\\backslash",
		"with space",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			_, err := manager.CreateRepository(name, "Test", false)
			if err == nil {
				t.Errorf("Expected error for invalid name '%s', got nil", name)
			}
		})
	}
}
