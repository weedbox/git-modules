package repository_manager

import (
	"errors"
	"fmt"
)

// Common errors
var (
	// ErrEmptyName indicates that the name parameter is empty
	ErrEmptyName = errors.New("name cannot be empty")

	// ErrInvalidName indicates that the name contains invalid characters
	ErrInvalidName = errors.New("invalid name: must contain only alphanumeric characters, dashes, underscores, and dots")
)

// Repository errors
var (
	// ErrRepositoryNameEmpty indicates repository name is empty
	ErrRepositoryNameEmpty = errors.New("repository name cannot be empty")

	// ErrRepositoryInvalidName indicates repository name is invalid
	ErrRepositoryInvalidName = errors.New("invalid repository name: must contain only alphanumeric characters, dashes, underscores, and dots")
)

// Tag errors
var (
	// ErrTagNameEmpty indicates tag name is empty
	ErrTagNameEmpty = errors.New("tag name cannot be empty")
)

// Group errors
var (
	// ErrGroupInvalidName indicates group name is invalid
	ErrGroupInvalidName = errors.New("invalid group name: must contain only alphanumeric characters, dashes, underscores, and dots")
)

// Error types for dynamic errors with context

// AlreadyExistsError represents a resource that already exists
type AlreadyExistsError struct {
	ResourceType string // "repository", "group", etc.
	Name         string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists: %s", e.ResourceType, e.Name)
}

// NotFoundError represents a resource that was not found
type NotFoundError struct {
	ResourceType string // "repository", "group", "tag", etc.
	Name         string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.ResourceType, e.Name)
}

// NotEmptyError represents a resource that is not empty when it should be
type NotEmptyError struct {
	ResourceType string // "group", etc.
	Name         string
}

func (e *NotEmptyError) Error() string {
	return fmt.Sprintf("%s is not empty: %s", e.ResourceType, e.Name)
}

// InvalidTypeError represents a resource that has the wrong type
type InvalidTypeError struct {
	Expected string
	Actual   string
	Name     string
}

func (e *InvalidTypeError) Error() string {
	return fmt.Sprintf("not a %s: %s", e.Expected, e.Name)
}

// ConflictError represents a conflict between resources
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

// Operation errors - wrapping underlying errors

// OperationError represents an error during an operation
type OperationError struct {
	Op  string // Operation name: "create", "delete", "open", etc.
	Err error  // Underlying error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("failed to %s: %w", e.Op, e.Err)
}

func (e *OperationError) Unwrap() error {
	return e.Err
}

// Helper functions to create common errors

// NewRepositoryNotFoundError creates a repository not found error
func NewRepositoryNotFoundError(name string) error {
	return &NotFoundError{
		ResourceType: "repository",
		Name:         name,
	}
}

// NewRepositoryAlreadyExistsError creates a repository already exists error
func NewRepositoryAlreadyExistsError(name string) error {
	return &AlreadyExistsError{
		ResourceType: "repository",
		Name:         name,
	}
}

// NewGroupNotFoundError creates a group not found error
func NewGroupNotFoundError(name string) error {
	return &NotFoundError{
		ResourceType: "group",
		Name:         name,
	}
}

// NewGroupAlreadyExistsError creates a group already exists error
func NewGroupAlreadyExistsError(name string) error {
	return &AlreadyExistsError{
		ResourceType: "group",
		Name:         name,
	}
}

// NewGroupNotEmptyError creates a group not empty error
func NewGroupNotEmptyError(name string) error {
	return &NotEmptyError{
		ResourceType: "group",
		Name:         name,
	}
}

// NewTagNotFoundError creates a tag not found error
func NewTagNotFoundError(name string) error {
	return &NotFoundError{
		ResourceType: "tag",
		Name:         name,
	}
}

// NewNotAGroupError creates a not a group error
func NewNotAGroupError(name string) error {
	return &InvalidTypeError{
		Expected: "group",
		Name:     name,
	}
}

// NewRepositoryWithNameExistsError creates an error when repository conflicts with group name
func NewRepositoryWithNameExistsError(name string) error {
	return &ConflictError{
		Message: fmt.Sprintf("a repository with this name already exists: %s", name),
	}
}

// NewGroupIsRepositoryError creates an error when a group path is actually a repository
func NewGroupIsRepositoryError(name string) error {
	return &InvalidTypeError{
		Expected: "group",
		Actual:   "repository",
		Name:     name,
	}
}

// Operation error helpers

// WrapCreateParentDirsError wraps an error when creating parent directories
func WrapCreateParentDirsError(err error) error {
	return &OperationError{Op: "create parent directories", Err: err}
}

// WrapInitGitRepoError wraps an error when initializing git repository
func WrapInitGitRepoError(err error) error {
	return &OperationError{Op: "initialize git repository", Err: err}
}

// WrapOpenRepoError wraps an error when opening repository
func WrapOpenRepoError(err error) error {
	return &OperationError{Op: "open repository", Err: err}
}

// WrapGetRepoConfigError wraps an error when getting repository config
func WrapGetRepoConfigError(err error) error {
	return &OperationError{Op: "get repository config", Err: err}
}

// WrapStatRepoError wraps an error when stating repository
func WrapStatRepoError(err error) error {
	return &OperationError{Op: "stat repository", Err: err}
}

// WrapDeleteRepoDirError wraps an error when deleting repository directory
func WrapDeleteRepoDirError(err error) error {
	return &OperationError{Op: "delete repository directory", Err: err}
}

// WrapWalkReposDirError wraps an error when walking repositories directory
func WrapWalkReposDirError(err error) error {
	return &OperationError{Op: "walk repositories directory", Err: err}
}

// WrapGetHEADError wraps an error when getting HEAD
func WrapGetHEADError(err error) error {
	return &OperationError{Op: "get HEAD", Err: err}
}

// WrapCommitNotFoundError wraps an error when commit is not found
func WrapCommitNotFoundError(err error) error {
	return &OperationError{Op: "find commit", Err: err}
}

// WrapEncodeTagError wraps an error when encoding tag object
func WrapEncodeTagError(err error) error {
	return &OperationError{Op: "encode tag object", Err: err}
}

// WrapStoreTagError wraps an error when storing tag object
func WrapStoreTagError(err error) error {
	return &OperationError{Op: "store tag object", Err: err}
}

// WrapSetTagRefError wraps an error when setting tag reference
func WrapSetTagRefError(err error) error {
	return &OperationError{Op: "set tag reference", Err: err}
}

// WrapDeleteTagError wraps an error when deleting tag
func WrapDeleteTagError(err error) error {
	return &OperationError{Op: "delete tag", Err: err}
}

// WrapTagNotFoundError wraps an error when tag is not found
func WrapTagNotFoundError(err error) error {
	return &OperationError{Op: "find tag", Err: err}
}

// WrapGetTagsError wraps an error when getting tags
func WrapGetTagsError(err error) error {
	return &OperationError{Op: "get tags", Err: err}
}

// WrapIterateTagsError wraps an error when iterating tags
func WrapIterateTagsError(err error) error {
	return &OperationError{Op: "iterate tags", Err: err}
}

// WrapCreateGroupDirError wraps an error when creating group directory
func WrapCreateGroupDirError(err error) error {
	return &OperationError{Op: "create group directory", Err: err}
}

// WrapStatGroupError wraps an error when stating group
func WrapStatGroupError(err error) error {
	return &OperationError{Op: "stat group", Err: err}
}

// WrapWalkGroupsDirError wraps an error when walking groups directory
func WrapWalkGroupsDirError(err error) error {
	return &OperationError{Op: "walk groups directory", Err: err}
}

// WrapReadGroupDirError wraps an error when reading group directory
func WrapReadGroupDirError(err error) error {
	return &OperationError{Op: "read group directory", Err: err}
}

// WrapDeleteGroupDirError wraps an error when deleting group directory
func WrapDeleteGroupDirError(err error) error {
	return &OperationError{Op: "delete group directory", Err: err}
}
