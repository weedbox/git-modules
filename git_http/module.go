package git_http

import (
	"context"
	"fmt"

	"github.com/sosedoff/gitkit"
	"github.com/spf13/viper"
	"github.com/weedbox/common-modules/http_server"
	"github.com/weedbox/git-modules/repository_manager"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ModuleName       = "GitHTTP"
	DefaultURLPrefix = "/"
)

type GitHTTP struct {
	params     Params
	logger     *zap.Logger
	scope      string
	gitService *gitkit.Server
	urlPrefix  string
}

type Params struct {
	fx.In

	Lifecycle         fx.Lifecycle
	Logger            *zap.Logger
	RepositoryManager *repository_manager.RepositoryManager
	HTTPServer        *http_server.HTTPServer
}

func Module(scope string) fx.Option {

	var m *GitHTTP

	return fx.Module(
		scope,
		fx.Provide(func(p Params) *GitHTTP {
			g := &GitHTTP{
				params: p,
				logger: p.Logger.Named(scope),
				scope:  scope,
			}

			g.initDefaultConfigs()

			return g
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

func (m *GitHTTP) onStart(ctx context.Context) error {
	m.logger.Info("Starting " + ModuleName)

	// Get and save URL prefix
	m.urlPrefix = viper.GetString(m.getConfigPath("url_prefix"))

	reposPath := m.params.RepositoryManager.GetReposPath()
	m.logger.Info("Initializing Git HTTP service",
		zap.String("reposPath", reposPath),
		zap.String("urlPrefix", m.urlPrefix),
	)

	// Initializing Git service for HTTP protocol
	m.gitService = gitkit.New(gitkit.Config{
		Dir:        reposPath,
		AutoCreate: false,
		AutoHooks:  false,
		Auth:       false, // Disable authentication for now
	})

	// Register routes on main router with urlPrefix
	router := m.params.HTTPServer.GetRouter()

	// Git HTTP protocol routes - must be registered before other routes to match .git paths
	// Supports multi-level paths like: /repos/username/repo.git/info/refs
	router.Any(m.urlPrefix+"/*path", m.handleGitProtocolOrAPI)

	return nil
}

func (m *GitHTTP) onStop(ctx context.Context) error {
	m.logger.Info("Stopped " + ModuleName)
	return nil
}

func (m *GitHTTP) getConfigPath(key string) string {
	return fmt.Sprintf("%s.%s", m.scope, key)
}

func (m *GitHTTP) initDefaultConfigs() {
	viper.SetDefault(m.getConfigPath("url_prefix"), DefaultURLPrefix)
}

func (m *GitHTTP) GetRepoPrefix() string {
	return m.urlPrefix
}
