package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
)

type App interface {
	Start()
}

type app struct {
	ctx               context.Context
	client            *kubernetes.Clientset
	config            *viper.Viper
	deployedNamespace string
	namespace         string
	targetService     string
}

func newApp(
	ctx context.Context,
	client *kubernetes.Clientset,
	config *viper.Viper,
) App {
	return &app{
		ctx:               ctx,
		client:            client,
		config:            config,
		deployedNamespace: getDeployedNamespace(),
		namespace:         getVaultNamespace(),
		targetService:     getTargetService(),
	}
}

func (a *app) Start() {
	a.watchNewPods()
}

func init() {
	flag.Parse()
	initializeLogger()
}

func main() {
	a, err := InitializeApp()
	if err != nil {
		slog.Error("Error initializing app", slog.String(loggingKeyError, err.Error()))
		os.Exit(1)
	} else if a == nil {
		slog.Error("App is nil")
		os.Exit(1)
	}
	a.Start()
}
