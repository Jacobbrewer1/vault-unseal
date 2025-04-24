package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v10"
	hashiVault "github.com/hashicorp/vault/api"

	"github.com/jacobbrewer1/web"
	"github.com/jacobbrewer1/web/logging"
)

type (
	AppConfig struct {
		VaultNamespace string `env:"VAULT_NAMESPACE" envDefault:"vault"`
		TargetService  string `env:"TARGET_SERVICE" envDefault:"vault"`

		unsealKeys []string
	}

	App struct {
		config *AppConfig
		base   *web.App

		vaultClient *hashiVault.Client
	}
)

func NewApp(l *slog.Logger) (*App, error) {
	base, err := web.NewApp(l)
	if err != nil {
		return nil, fmt.Errorf("failed to create web app: %w", err)
	}

	config := new(AppConfig)
	if err := env.Parse(config); err != nil {
		return nil, fmt.Errorf("failed to parse environment: %w", err)
	}

	return &App{
		config: config,
		base:   base,
	}, nil
}

func (a *App) Start() error {
	if err := a.base.Start(
		web.WithViperConfig(),
		web.WithConfigWatchers(func() {
			a.base.Shutdown() // Reboot the app if the config changes
		}),
		web.WithInClusterKubeClient(),
		web.WithKubernetesPodInformer(),
		web.WithServiceEndpointHashBucket(appName),
		web.WithDependencyBootstrap(func(ctx context.Context) error {
			vip := a.base.Viper()
			a.config.unsealKeys = vip.GetStringSlice("unseal_keys")
			switch len(a.config.unsealKeys) {
			case 0:
				return fmt.Errorf("no unseal keys provided")
			case 1, 2:
				return fmt.Errorf("not enough unseal keys provided")
			case 3, 4, 5:
				// Valid range, do nothing
			default:
				return fmt.Errorf("too many unseal keys provided")
			}
			return nil
		}),
		web.WithDependencyBootstrap(func(ctx context.Context) error {
			vaultClient, err := hashiVault.NewClient(hashiVault.DefaultConfig())
			if err != nil {
				return fmt.Errorf("failed to create vault client: %w", err)
			}
			a.vaultClient = vaultClient
			return nil
		}),
		web.WithIndefiniteAsyncTask("unseal-vault", a.watchVaultPods(
			logging.LoggerWithComponent(a.base.Logger(), "watch-new-pods"),
		)),
	); err != nil {
		return fmt.Errorf("failed to start web app: %w", err)
	}

	return nil
}

func (a *App) WaitForEnd() {
	a.base.WaitForEnd(a.base.Shutdown)
}

func main() {
	l := logging.NewLogger(
		logging.WithAppName(appName),
	)

	app, err := NewApp(l)
	if err != nil {
		l.Error("failed to create app", slog.String(logging.KeyError, err.Error()))
		os.Exit(1)
	}

	if err := app.Start(); err != nil {
		l.Error("failed to start app", slog.String(logging.KeyError, err.Error()))
		os.Exit(1)
	}

	app.WaitForEnd()
}
