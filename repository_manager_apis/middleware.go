package repository_manager_apis

import "github.com/gin-gonic/gin"

type MiddlewareConfig struct {
	// Repository middlewares
	CreateRepository []gin.HandlerFunc
	ListRepositories []gin.HandlerFunc
	GetRepository    []gin.HandlerFunc
	DeleteRepository []gin.HandlerFunc

	// Tag middlewares
	CreateTag []gin.HandlerFunc
	ListTags  []gin.HandlerFunc
	GetTag    []gin.HandlerFunc
	DeleteTag []gin.HandlerFunc

	// Group middlewares
	CreateGroup []gin.HandlerFunc
	ListGroups  []gin.HandlerFunc
	GetGroup    []gin.HandlerFunc
	DeleteGroup []gin.HandlerFunc
}

func NewMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		CreateRepository: []gin.HandlerFunc{},
		ListRepositories: []gin.HandlerFunc{},
		GetRepository:    []gin.HandlerFunc{},
		DeleteRepository: []gin.HandlerFunc{},
		CreateTag:        []gin.HandlerFunc{},
		ListTags:         []gin.HandlerFunc{},
		GetTag:           []gin.HandlerFunc{},
		DeleteTag:        []gin.HandlerFunc{},
		CreateGroup:      []gin.HandlerFunc{},
		ListGroups:       []gin.HandlerFunc{},
		GetGroup:         []gin.HandlerFunc{},
		DeleteGroup:      []gin.HandlerFunc{},
	}
}

func (mc *MiddlewareConfig) Use(fn gin.HandlerFunc) {

	// Append to all repository middleware slices
	mc.CreateRepository = append(mc.CreateRepository, fn)
	mc.ListRepositories = append(mc.ListRepositories, fn)
	mc.GetRepository = append(mc.GetRepository, fn)
	mc.DeleteRepository = append(mc.DeleteRepository, fn)

	// Append to all tag middleware slices
	mc.CreateTag = append(mc.CreateTag, fn)
	mc.ListTags = append(mc.ListTags, fn)
	mc.GetTag = append(mc.GetTag, fn)
	mc.DeleteTag = append(mc.DeleteTag, fn)

	// Append to all group middleware slices
	mc.CreateGroup = append(mc.CreateGroup, fn)
	mc.ListGroups = append(mc.ListGroups, fn)
	mc.GetGroup = append(mc.GetGroup, fn)
	mc.DeleteGroup = append(mc.DeleteGroup, fn)
}
