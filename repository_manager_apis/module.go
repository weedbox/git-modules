// Package repository_manager_apis provides REST API for Git repository management
//
// @title Git Repository Manager API
// @version 1.0
// @description REST API for managing Git repositories, tags, and groups/namespaces
// @description
// @description This API provides comprehensive Git repository management capabilities including:
// @description - Repository CRUD operations with multi-level path support
// @description - Git tag management (lightweight and annotated tags)
// @description - Group/namespace management for organizing repositories
// @description
// @description All repository and group paths support multi-level hierarchies like "org/team/project"
//
// @contact.name API Support
// @contact.url https://github.com/weedbox/git-modules
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @BasePath /apis/v1/repos
//
// @schemes http https
// @produce json
// @consumes json
package repository_manager_apis

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/weedbox/common-modules/http_server"
	"github.com/weedbox/git-modules/repository_manager"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ModuleName       = "RepositoryManagerAPIs"
	DefaultURLPrefix = "/apis/v1/repos"
)

type RepositoryManagerAPIs struct {
	params           Params
	logger           *zap.Logger
	scope            string
	middlewareConfig MiddlewareConfig
}

type Params struct {
	fx.In

	Lifecycle         fx.Lifecycle
	Logger            *zap.Logger
	RepositoryManager *repository_manager.RepositoryManager
	HTTPServer        *http_server.HTTPServer
}

func Module(scope string) fx.Option {

	var m *RepositoryManagerAPIs

	return fx.Module(
		scope,
		fx.Provide(func(p Params) *RepositoryManagerAPIs {
			apis := &RepositoryManagerAPIs{
				params: p,
				logger: p.Logger.Named(scope),
				scope:  scope,
			}

			apis.initDefaultConfigs()

			return apis
		}),
		fx.Populate(&m),
		fx.Invoke(func(p Params) {

			p.Lifecycle.Append(
				fx.Hook{
					OnStart: m.onStart,
					OnStop:  m.onStop,
				},
			)
		}),
	)

}

func (m *RepositoryManagerAPIs) onStart(ctx context.Context) error {
	m.logger.Info("Starting " + ModuleName)

	// Register routes
	urlPrefix := viper.GetString(m.getConfigPath("url_prefix"))
	router := m.params.HTTPServer.GetRouter().Group(urlPrefix)

	// Repository management routes
	router.POST("", append(m.middlewareConfig.CreateRepository, m.handleCreateRepository)...)
	router.GET("", append(m.middlewareConfig.ListRepositories, m.handleListRepositories)...)

	// Repository and tag operations with tags middleware
	// The middleware will check if path contains /tags/ and if repo exists
	router.GET("/*name", m.tagsMiddleware(), m.dispatchGet())
	router.POST("/*name", m.tagsMiddleware(), m.dispatchPost())
	router.DELETE("/*name", m.tagsMiddleware(), m.dispatchDelete())

	return nil
}

func (m *RepositoryManagerAPIs) onStop(ctx context.Context) error {
	m.logger.Info("Stopped " + ModuleName)
	return nil
}

func (m *RepositoryManagerAPIs) getConfigPath(key string) string {
	return fmt.Sprintf("%s.%s", m.scope, key)
}

func (m *RepositoryManagerAPIs) initDefaultConfigs() {
	viper.SetDefault(m.getConfigPath("url_prefix"), DefaultURLPrefix)

	// Default empty middleware config
	mwcfg := NewMiddlewareConfig()
	m.SetupMiddleware(mwcfg)
}

func (m *RepositoryManagerAPIs) SetupMiddleware(cfg MiddlewareConfig) {
	m.middlewareConfig = cfg
	m.middlewareConfig.CreateRepository = append([]gin.HandlerFunc{}, cfg.CreateRepository...)
	m.middlewareConfig.ListRepositories = append([]gin.HandlerFunc{}, cfg.ListRepositories...)
	m.middlewareConfig.GetRepository = append([]gin.HandlerFunc{}, cfg.GetRepository...)
	m.middlewareConfig.DeleteRepository = append([]gin.HandlerFunc{}, cfg.DeleteRepository...)
	m.middlewareConfig.CreateTag = append([]gin.HandlerFunc{}, cfg.CreateTag...)
	m.middlewareConfig.ListTags = append([]gin.HandlerFunc{}, cfg.ListTags...)
	m.middlewareConfig.GetTag = append([]gin.HandlerFunc{}, cfg.GetTag...)
	m.middlewareConfig.DeleteTag = append([]gin.HandlerFunc{}, cfg.DeleteTag...)
	m.middlewareConfig.CreateGroup = append([]gin.HandlerFunc{}, cfg.CreateGroup...)
	m.middlewareConfig.ListGroups = append([]gin.HandlerFunc{}, cfg.ListGroups...)
	m.middlewareConfig.GetGroup = append([]gin.HandlerFunc{}, cfg.GetGroup...)
	m.middlewareConfig.DeleteGroup = append([]gin.HandlerFunc{}, cfg.DeleteGroup...)
}

type pathKind int

const (
	pathKindRepository pathKind = iota
	pathKindGroup
	pathKindTagsRoot
	pathKindTagItem
)

const (
	contextKeyPathKind = "path_kind"
	contextKeyRepoName = "repo_name"
	contextKeyTagName  = "tag_name"
)

// tagsMiddleware checks if the path is a tags operation and validates repository existence
// Also differentiates between repositories and groups
func (m *RepositoryManagerAPIs) tagsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Param("name"), "/")
		if path == "" {
			c.Next()
			return
		}

		// Check if path contains /tags/
		idx := indexOfTagsSegment(path)
		if idx < 0 {
			// Not a tags path, check if it's a repository or group
			isRepo := m.params.RepositoryManager.IsRepository(path)
			isGroup := m.params.RepositoryManager.IsGroup(path)

			if isRepo {
				c.Set(contextKeyPathKind, pathKindRepository)
				c.Set(contextKeyRepoName, path)
			} else if isGroup {
				c.Set(contextKeyPathKind, pathKindGroup)
				c.Set(contextKeyRepoName, path)
			} else {
				// Neither exists, return 404
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			c.Next()
			return
		}

		// Extract repo name and check if it exists
		repoName := strings.TrimSuffix(path[:idx], "/")
		if repoName == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// Verify repository exists (tags only work on repositories)
		_, err := m.params.RepositoryManager.GetRepository(repoName)
		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// Parse tags path
		rest := path[idx+len("/tags"):]
		if rest == "" {
			// Path is /repo/tags (list tags)
			c.Set(contextKeyPathKind, pathKindTagsRoot)
			c.Set(contextKeyRepoName, repoName)
			c.Next()
			return
		}

		if !strings.HasPrefix(rest, "/") {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		tagName := strings.TrimPrefix(rest, "/")
		if tagName == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// Path is /repo/tags/tagname (tag operation)
		c.Set(contextKeyPathKind, pathKindTagItem)
		c.Set(contextKeyRepoName, repoName)
		c.Set(contextKeyTagName, tagName)
		c.Next()
	}
}

func (m *RepositoryManagerAPIs) dispatchGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		kind, ok := c.Get(contextKeyPathKind)
		if !ok {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		repoName, _ := c.Get(contextKeyRepoName)
		tagName, _ := c.Get(contextKeyTagName)

		setParam(c, "name", "/"+repoName.(string))

		switch kind.(pathKind) {
		case pathKindRepository:
			m.invokeHandlers(c, m.middlewareConfig.GetRepository, m.handleGetRepository)
		case pathKindGroup:
			m.invokeHandlers(c, m.middlewareConfig.GetGroup, m.handleGetGroup)
		case pathKindTagsRoot:
			m.invokeHandlers(c, m.middlewareConfig.ListTags, m.handleListTags)
		case pathKindTagItem:
			setParam(c, "tag", "/"+tagName.(string))
			m.invokeHandlers(c, m.middlewareConfig.GetTag, m.handleGetTag)
		default:
			c.AbortWithStatus(http.StatusNotFound)
		}
	}
}

func (m *RepositoryManagerAPIs) dispatchPost() gin.HandlerFunc {
	return func(c *gin.Context) {
		kind, ok := c.Get(contextKeyPathKind)
		if !ok {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		repoName, _ := c.Get(contextKeyRepoName)
		setParam(c, "name", "/"+repoName.(string))

		switch kind.(pathKind) {
		case pathKindTagsRoot:
			m.invokeHandlers(c, m.middlewareConfig.CreateTag, m.handleCreateTag)
		default:
			c.AbortWithStatus(http.StatusNotFound)
		}
	}
}

func (m *RepositoryManagerAPIs) dispatchDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		kind, ok := c.Get(contextKeyPathKind)
		if !ok {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		repoName, _ := c.Get(contextKeyRepoName)
		tagName, _ := c.Get(contextKeyTagName)

		setParam(c, "name", "/"+repoName.(string))

		switch kind.(pathKind) {
		case pathKindRepository:
			m.invokeHandlers(c, m.middlewareConfig.DeleteRepository, m.handleDeleteRepository)
		case pathKindGroup:
			m.invokeHandlers(c, m.middlewareConfig.DeleteGroup, m.handleDeleteGroup)
		case pathKindTagItem:
			setParam(c, "tag", "/"+tagName.(string))
			m.invokeHandlers(c, m.middlewareConfig.DeleteTag, m.handleDeleteTag)
		default:
			c.AbortWithStatus(http.StatusNotFound)
		}
	}
}

func (m *RepositoryManagerAPIs) invokeHandlers(c *gin.Context, middlewares []gin.HandlerFunc, handler gin.HandlerFunc) {
	chain := make(gin.HandlersChain, 0, len(middlewares)+1)
	chain = append(chain, middlewares...)
	chain = append(chain, handler)
	runHandlerChain(c, chain)
}

func runHandlerChain(c *gin.Context, chain gin.HandlersChain) {
	if len(chain) == 0 {
		return
	}

	setGinHandlers(c, chain)
	c.Next()
}

func setGinHandlers(c *gin.Context, chain gin.HandlersChain) {
	ctxValue := reflect.ValueOf(c).Elem()

	handlersField := ctxValue.FieldByName("handlers")
	indexField := ctxValue.FieldByName("index")

	setUnexportedField(handlersField, reflect.ValueOf(chain))
	setUnexportedField(indexField, reflect.ValueOf(int8(-1)))
}

func setUnexportedField(field, value reflect.Value) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(value)
}

func setParam(c *gin.Context, key, value string) {
	for i := range c.Params {
		if c.Params[i].Key == key {
			c.Params[i].Value = value
			return
		}
	}
	c.Params = append(c.Params, gin.Param{Key: key, Value: value})
}

func indexOfTagsSegment(path string) int {
	const segment = "/tags"

	for idx := strings.Index(path, segment); idx >= 0; {
		end := idx + len(segment)
		if end == len(path) || path[end] == '/' {
			return idx
		}

		next := strings.Index(path[end:], segment)
		if next < 0 {
			return -1
		}
		idx = end + next
	}

	return -1
}
