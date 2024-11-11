package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jacobbrewer1/vault-unseal/encryption"
	"k8s.io/client-go/kubernetes"
)

type App interface {
	Start()
}

type app struct {
	ctx               context.Context
	client            *kubernetes.Clientset
	deployedNamespace string
	namespace         string
	targetService     string
	cryptoKey         string
}

func newApp(
	ctx context.Context,
	client *kubernetes.Clientset,
) App {
	return &app{
		ctx:               ctx,
		client:            client,
		deployedNamespace: getDeployedNamespace(),
		namespace:         getVaultNamespace(),
		targetService:     getTargetService(),
		cryptoKey:         encryption.GetCryptoKey(),
	}
}

func (a *app) Start() {
	a.watchNewPods()
}

func init() {
	initializeLogger()
}

func main() {
	a, err := InitializeApp()
	if err != nil {
		slog.Error("Error initializing app", slog.String("error", err.Error()))
		os.Exit(1)
	} else if a == nil {
		slog.Error("App is nil")
		os.Exit(1)
	}
	a.Start()
}
