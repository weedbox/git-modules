package repository_manager

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ModuleName       = "RepositoryManager"
	DefaultReposPath = "./git/repos"
)

type RepositoryManager struct {
	params    Params
	logger    *zap.Logger
	scope     string
	reposPath string
}

type Params struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *zap.Logger
}

func Module(scope string) fx.Option {

	var m *RepositoryManager

	return fx.Module(
		scope,
		fx.Provide(func(p Params) *RepositoryManager {
			rm := &RepositoryManager{
				params: p,
				logger: p.Logger.Named(scope),
				scope:  scope,
			}

			rm.initDefaultConfigs()

			return rm
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

func (m *RepositoryManager) onStart(ctx context.Context) error {
	m.logger.Info("Starting " + ModuleName)
	m.reposPath = viper.GetString(m.getConfigPath("repos_path"))
	return nil
}

func (m *RepositoryManager) onStop(ctx context.Context) error {
	m.logger.Info("Stopped " + ModuleName)
	return nil
}

func (m *RepositoryManager) getConfigPath(key string) string {
	return fmt.Sprintf("%s.%s", m.scope, key)
}

func (m *RepositoryManager) initDefaultConfigs() {
	viper.SetDefault(m.getConfigPath("repos_path"), DefaultReposPath)
}
