package repository_manager

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"go.uber.org/zap"
)

// CreateRepository creates a new Git repository
func (m *RepositoryManager) CreateRepository(name, description string) (*Repository, error) {
	// Validate repository name
	if name == "" {
		return nil, ErrRepositoryNameEmpty
	}

	// Sanitize name (allow only alphanumeric, dash, underscore, dot)
	if !isValidRepoName(name) {
		return nil, ErrRepositoryInvalidName
	}

	// Create repository path
	repoPath := filepath.Join(m.reposPath, name+".git")

	// Check if repository already exists
	if _, err := os.Stat(repoPath); err == nil {
		return nil, NewRepositoryAlreadyExistsError(name)
	}

	// Create parent directories if they don't exist
	// For multi-level paths like "username/repo", this ensures "username" directory exists
	parentDir := filepath.Dir(repoPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return nil, WrapCreateParentDirsError(err)
	}

	// Initialize bare repository using go-git
	_, err := git.PlainInit(repoPath, true)
	if err != nil {
		return nil, WrapInitGitRepoError(err)
	}

	// Open the repository to configure it
	fs := osfs.New(repoPath)
	storer := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	repo, err := git.Open(storer, fs)
	if err != nil {
		return nil, WrapOpenRepoError(err)
	}

	// Get repository config
	cfg, err := repo.Config()
	if err != nil {
		return nil, WrapGetRepoConfigError(err)
	}

	// Enable receive-pack for HTTP push
	cfg.Raw.Section("http").SetOption("receivepack", "true")

	// Store description
	if description != "" {
		cfg.Raw.Section("repository").SetOption("description", description)
	}

	// Save config
	if err := repo.SetConfig(cfg); err != nil {
		m.logger.Warn("Failed to save config", zap.Error(err))
	}

	// Get creation time
	info, err := os.Stat(repoPath)
	if err != nil {
		return nil, WrapStatRepoError(err)
	}

	repository := &Repository{
		Name:        name,
		Description: description,
		Path:        repoPath,
		CreatedAt:   info.ModTime(),
	}

	m.logger.Info("Repository created", zap.String("name", name), zap.String("path", repoPath))

	return repository, nil
}

// GetReposPath returns the base path where repositories are stored
func (m *RepositoryManager) GetReposPath() string {
	return m.reposPath
}

// DeleteRepository deletes a Git repository
func (m *RepositoryManager) DeleteRepository(name string) error {
	// Validate repository name to prevent path traversal attacks
	if !isValidRepoName(name) {
		return ErrRepositoryInvalidName
	}

	repoPath := filepath.Join(m.reposPath, name+".git")

	// Check if repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return NewRepositoryNotFoundError(name)
	}

	// Delete filesystem directory
	if err := os.RemoveAll(repoPath); err != nil {
		m.logger.Error("Failed to delete repository directory", zap.String("path", repoPath), zap.Error(err))
		return WrapDeleteRepoDirError(err)
	}

	m.logger.Info("Repository deleted", zap.String("name", name), zap.String("path", repoPath))
	return nil
}

// GetRepository retrieves a repository by name
func (m *RepositoryManager) GetRepository(name string) (*Repository, error) {
	// Validate repository name to prevent path traversal attacks
	if !isValidRepoName(name) {
		return nil, ErrRepositoryInvalidName
	}

	repoPath := filepath.Join(m.reposPath, name+".git")

	// Check if repository exists
	info, err := os.Stat(repoPath)
	if os.IsNotExist(err) {
		return nil, NewRepositoryNotFoundError(name)
	}
	if err != nil {
		return nil, WrapStatRepoError(err)
	}

	// Open repository to read config
	fs := osfs.New(repoPath)
	storer := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	repo, err := git.Open(storer, fs)
	if err != nil {
		return nil, WrapOpenRepoError(err)
	}

	// Read description from git config
	description := ""
	cfg, err := repo.Config()
	if err == nil {
		if cfg.Raw.HasSection("repository") {
			description = cfg.Raw.Section("repository").Option("description")
		}
	}

	repository := &Repository{
		Name:        name,
		Description: description,
		Path:        repoPath,
		CreatedAt:   info.ModTime(),
	}

	return repository, nil
}

// ListRepositories returns all repositories
// Supports multi-level directory structures, recursively scanning for .git repositories
func (m *RepositoryManager) ListRepositories() ([]Repository, error) {
	repos := make([]Repository, 0)

	// Walk through all directories recursively
	err := filepath.WalkDir(m.reposPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !d.IsDir() {
			return nil
		}

		// Check if this is a .git repository
		if !strings.HasSuffix(d.Name(), ".git") {
			return nil
		}

		// Get relative path from repos root
		relPath, err := filepath.Rel(m.reposPath, path)
		if err != nil {
			m.logger.Warn("Failed to get relative path", zap.String("path", path), zap.Error(err))
			return nil
		}

		// Remove .git suffix to get repository name
		repoName := strings.TrimSuffix(relPath, ".git")

		// Get directory info
		info, err := d.Info()
		if err != nil {
			m.logger.Warn("Failed to get info for repository", zap.String("name", repoName), zap.Error(err))
			return nil
		}

		// Read description from git config using go-git
		description := ""
		fs := osfs.New(path)
		storer := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
		repo, err := git.Open(storer, fs)
		if err == nil {
			cfg, err := repo.Config()
			if err == nil && cfg.Raw.HasSection("repository") {
				description = cfg.Raw.Section("repository").Option("description")
			}
		}

		repos = append(repos, Repository{
			Name:        repoName,
			Description: description,
			Path:        path,
			CreatedAt:   info.ModTime(),
		})

		// Skip descending into .git directory
		return filepath.SkipDir
	})

	if err != nil {
		return nil, WrapWalkReposDirError(err)
	}

	return repos, nil
}

// CreateTag creates a Git tag
func (m *RepositoryManager) CreateTag(repoName, tagName, commitHash, message, tagger string) (*Tag, error) {
	// Validate repository name to prevent path traversal attacks
	if !isValidRepoName(repoName) {
		return nil, ErrRepositoryInvalidName
	}

	if tagName == "" {
		return nil, ErrTagNameEmpty
	}

	repoPath := filepath.Join(m.reposPath, repoName+".git")

	// Open repository
	fs := osfs.New(repoPath)
	storer := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	repo, err := git.Open(storer, fs)
	if err != nil {
		return nil, WrapOpenRepoError(err)
	}

	// Get commit hash
	var hash plumbing.Hash
	if commitHash == "" {
		// Use HEAD if no commit specified
		ref, err := repo.Head()
		if err != nil {
			return nil, WrapGetHEADError(err)
		}
		hash = ref.Hash()
	} else {
		hash = plumbing.NewHash(commitHash)
	}

	// Check if commit exists
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, WrapCommitNotFoundError(err)
	}

	tag := &Tag{
		Name:       tagName,
		CommitHash: hash.String(),
	}

	// Create tag
	if message != "" {
		// Annotated tag
		tagObj := &object.Tag{
			Name:       tagName,
			Message:    message,
			Target:     hash,
			TargetType: plumbing.CommitObject,
		}

		if tagger != "" {
			// Parse tagger (format: "Name <email>")
			tagObj.Tagger = object.Signature{
				Name:  tagger,
				Email: "",
				When:  commit.Committer.When,
			}
		} else {
			tagObj.Tagger = commit.Committer
		}

		// Encode and store tag object
		obj := repo.Storer.NewEncodedObject()
		if err := tagObj.Encode(obj); err != nil {
			return nil, WrapEncodeTagError(err)
		}
		tagHash, err := repo.Storer.SetEncodedObject(obj)
		if err != nil {
			return nil, WrapStoreTagError(err)
		}

		// Create reference
		tagRef := plumbing.NewHashReference(
			plumbing.NewTagReferenceName(tagName),
			tagHash,
		)
		if err := repo.Storer.SetReference(tagRef); err != nil {
			return nil, WrapSetTagRefError(err)
		}

		tag.Type = "annotated"
		tag.Message = message
		tag.Tagger = tagObj.Tagger.Name
		tag.TaggerDate = tagObj.Tagger.When
	} else {
		// Lightweight tag - just a reference to a commit
		tagRef := plumbing.NewHashReference(
			plumbing.NewTagReferenceName(tagName),
			hash,
		)
		if err := repo.Storer.SetReference(tagRef); err != nil {
			return nil, WrapSetTagRefError(err)
		}

		tag.Type = "lightweight"
	}

	m.logger.Info("Tag created", zap.String("repo", repoName), zap.String("tag", tagName), zap.String("commit", hash.String()))
	return tag, nil
}

// DeleteTag deletes a Git tag
func (m *RepositoryManager) DeleteTag(repoName, tagName string) error {
	// Validate repository name to prevent path traversal attacks
	if !isValidRepoName(repoName) {
		return ErrRepositoryInvalidName
	}

	if tagName == "" {
		return ErrTagNameEmpty
	}

	repoPath := filepath.Join(m.reposPath, repoName+".git")

	// Open repository
	fs := osfs.New(repoPath)
	storer := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	repo, err := git.Open(storer, fs)
	if err != nil {
		return WrapOpenRepoError(err)
	}

	// Delete tag reference
	tagRef := plumbing.NewTagReferenceName(tagName)
	if err := repo.Storer.RemoveReference(tagRef); err != nil {
		return WrapDeleteTagError(err)
	}

	m.logger.Info("Tag deleted", zap.String("repo", repoName), zap.String("tag", tagName))
	return nil
}

// GetTag retrieves a specific tag
func (m *RepositoryManager) GetTag(repoName, tagName string) (*Tag, error) {
	// Validate repository name to prevent path traversal attacks
	if !isValidRepoName(repoName) {
		return nil, ErrRepositoryInvalidName
	}

	if tagName == "" {
		return nil, ErrTagNameEmpty
	}

	repoPath := filepath.Join(m.reposPath, repoName+".git")

	// Open repository
	fs := osfs.New(repoPath)
	storer := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	repo, err := git.Open(storer, fs)
	if err != nil {
		return nil, WrapOpenRepoError(err)
	}

	// Get tag reference
	tagRef, err := repo.Reference(plumbing.NewTagReferenceName(tagName), true)
	if err != nil {
		return nil, WrapTagNotFoundError(err)
	}

	tag := &Tag{
		Name:       tagName,
		CommitHash: tagRef.Hash().String(),
	}

	// Try to get tag object (annotated tag)
	tagObj, err := repo.TagObject(tagRef.Hash())
	if err == nil {
		// It's an annotated tag
		tag.Type = "annotated"
		tag.Message = tagObj.Message
		tag.Tagger = tagObj.Tagger.Name
		tag.TaggerDate = tagObj.Tagger.When
		tag.CommitHash = tagObj.Target.String()
	} else {
		// It's a lightweight tag
		tag.Type = "lightweight"
	}

	return tag, nil
}

// ListTags lists all tags in a repository
func (m *RepositoryManager) ListTags(repoName string) ([]Tag, error) {
	// Validate repository name to prevent path traversal attacks
	if !isValidRepoName(repoName) {
		return nil, ErrRepositoryInvalidName
	}

	repoPath := filepath.Join(m.reposPath, repoName+".git")

	// Open repository
	fs := osfs.New(repoPath)
	storer := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	repo, err := git.Open(storer, fs)
	if err != nil {
		return nil, WrapOpenRepoError(err)
	}

	// Get all tag references
	tagRefs, err := repo.Tags()
	if err != nil {
		return nil, WrapGetTagsError(err)
	}

	tags := make([]Tag, 0)
	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()
		tag := Tag{
			Name:       tagName,
			CommitHash: ref.Hash().String(),
		}

		// Try to get tag object (annotated tag)
		tagObj, err := repo.TagObject(ref.Hash())
		if err == nil {
			// It's an annotated tag
			tag.Type = "annotated"
			tag.Message = tagObj.Message
			tag.Tagger = tagObj.Tagger.Name
			tag.TaggerDate = tagObj.Tagger.When
			tag.CommitHash = tagObj.Target.String()
		} else {
			// It's a lightweight tag
			tag.Type = "lightweight"
		}

		tags = append(tags, tag)
		return nil
	})

	if err != nil {
		return nil, WrapIterateTagsError(err)
	}

	return tags, nil
}

// isValidRepoName checks if the repository name is valid
// Supports multi-level paths like "username/repo" or "group/project/repo"
func isValidRepoName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}

	// Disallow backslashes (Windows path separator)
	if strings.Contains(name, "\\") {
		return false
	}

	// Disallow leading or trailing slashes
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return false
	}

	// Split by forward slash to validate each path segment
	segments := strings.Split(name, "/")

	for _, segment := range segments {
		// Each segment must not be empty (no consecutive slashes)
		if segment == "" {
			return false
		}

		// Check for relative path components
		if segment == "." || segment == ".." {
			return false
		}

		// Check each character in the segment
		for _, c := range segment {
			if !((c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') ||
				c == '-' || c == '_' || c == '.') {
				return false
			}
		}
	}

	return true
}

// CreateGroup creates a new group (namespace/organization)
func (m *RepositoryManager) CreateGroup(name, description string) (*Group, error) {
	// Validate group name using the same validation as repository names
	if !isValidRepoName(name) {
		return nil, ErrGroupInvalidName
	}

	// Create group path (without .git suffix)
	groupPath := filepath.Join(m.reposPath, name)

	// Check if group already exists
	if _, err := os.Stat(groupPath); err == nil {
		return nil, NewGroupAlreadyExistsError(name)
	}

	// Check if a repository with this name exists
	repoPath := groupPath + ".git"
	if _, err := os.Stat(repoPath); err == nil {
		return nil, NewRepositoryWithNameExistsError(name)
	}

	// Create the group directory
	if err := os.MkdirAll(groupPath, 0755); err != nil {
		return nil, WrapCreateGroupDirError(err)
	}

	// Create a .groupinfo file to store metadata
	if description != "" {
		infoPath := filepath.Join(groupPath, ".groupinfo")
		if err := os.WriteFile(infoPath, []byte(description), 0644); err != nil {
			m.logger.Warn("Failed to write group info", zap.Error(err))
		}
	}

	// Get creation time
	info, err := os.Stat(groupPath)
	if err != nil {
		return nil, WrapStatGroupError(err)
	}

	group := &Group{
		Name:        name,
		Description: description,
		Path:        groupPath,
		CreatedAt:   info.ModTime(),
	}

	m.logger.Info("Group created", zap.String("name", name), zap.String("path", groupPath))
	return group, nil
}

// GetGroup retrieves a group by name
func (m *RepositoryManager) GetGroup(name string) (*Group, error) {
	// Validate group name to prevent path traversal attacks
	if !isValidRepoName(name) {
		return nil, ErrGroupInvalidName
	}

	groupPath := filepath.Join(m.reposPath, name)

	// Check if group exists and is a directory
	info, err := os.Stat(groupPath)
	if os.IsNotExist(err) {
		return nil, NewGroupNotFoundError(name)
	}
	if err != nil {
		return nil, WrapStatGroupError(err)
	}
	if !info.IsDir() {
		return nil, NewNotAGroupError(name)
	}

	// Check if it's actually a repository (has .git in the name)
	if strings.HasSuffix(groupPath, ".git") {
		return nil, NewGroupIsRepositoryError(name)
	}

	// Read description from .groupinfo file
	description := ""
	infoPath := filepath.Join(groupPath, ".groupinfo")
	if data, err := os.ReadFile(infoPath); err == nil {
		description = string(data)
	}

	group := &Group{
		Name:        name,
		Description: description,
		Path:        groupPath,
		CreatedAt:   info.ModTime(),
	}

	return group, nil
}

// ListGroups returns all groups (directories without .git suffix)
func (m *RepositoryManager) ListGroups() ([]Group, error) {
	groups := make([]Group, 0)

	// Walk through directories
	err := filepath.WalkDir(m.reposPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip root directory
		if path == m.reposPath {
			return nil
		}

		// Only process directories
		if !d.IsDir() {
			return nil
		}

		// Skip .git directories (repositories)
		if strings.HasSuffix(d.Name(), ".git") {
			return filepath.SkipDir
		}

		// Get relative path from repos root
		relPath, err := filepath.Rel(m.reposPath, path)
		if err != nil {
			m.logger.Warn("Failed to get relative path", zap.String("path", path), zap.Error(err))
			return nil
		}

		// Get directory info
		info, err := d.Info()
		if err != nil {
			m.logger.Warn("Failed to get info for group", zap.String("name", relPath), zap.Error(err))
			return nil
		}

		// Read description
		description := ""
		infoPath := filepath.Join(path, ".groupinfo")
		if data, err := os.ReadFile(infoPath); err == nil {
			description = string(data)
		}

		groups = append(groups, Group{
			Name:        relPath,
			Description: description,
			Path:        path,
			CreatedAt:   info.ModTime(),
		})

		return nil
	})

	if err != nil {
		return nil, WrapWalkGroupsDirError(err)
	}

	return groups, nil
}

// DeleteGroup deletes a group
func (m *RepositoryManager) DeleteGroup(name string) error {
	// Validate group name to prevent path traversal attacks
	if !isValidRepoName(name) {
		return ErrGroupInvalidName
	}

	groupPath := filepath.Join(m.reposPath, name)

	// Check if group exists
	info, err := os.Stat(groupPath)
	if os.IsNotExist(err) {
		return NewGroupNotFoundError(name)
	}
	if err != nil {
		return WrapStatGroupError(err)
	}
	if !info.IsDir() {
		return NewNotAGroupError(name)
	}

	// Check if group is empty (contains only .groupinfo file or is completely empty)
	entries, err := os.ReadDir(groupPath)
	if err != nil {
		return WrapReadGroupDirError(err)
	}

	for _, entry := range entries {
		if entry.Name() != ".groupinfo" {
			return NewGroupNotEmptyError(name)
		}
	}

	// Delete the directory
	if err := os.RemoveAll(groupPath); err != nil {
		m.logger.Error("Failed to delete group directory", zap.String("path", groupPath), zap.Error(err))
		return WrapDeleteGroupDirError(err)
	}

	m.logger.Info("Group deleted", zap.String("name", name), zap.String("path", groupPath))
	return nil
}

// IsGroup checks if the given name is a group (not a repository)
func (m *RepositoryManager) IsGroup(name string) bool {
	// Validate group name to prevent path traversal attacks
	if !isValidRepoName(name) {
		return false
	}

	groupPath := filepath.Join(m.reposPath, name)

	// Check if it exists and is a directory
	info, err := os.Stat(groupPath)
	if err != nil || !info.IsDir() {
		return false
	}

	// It's a group if it doesn't end with .git
	return !strings.HasSuffix(name, ".git")
}

// IsRepository checks if the given name is a repository
func (m *RepositoryManager) IsRepository(name string) bool {
	// Validate repository name to prevent path traversal attacks
	if !isValidRepoName(name) {
		return false
	}

	repoPath := filepath.Join(m.reposPath, name+".git")

	// Check if it exists and is a directory
	info, err := os.Stat(repoPath)
	if err != nil || !info.IsDir() {
		return false
	}

	return true
}
