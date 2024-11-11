package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jacobbrewer1/workerpool"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

func getVaultNamespace() string {
	vaultNamespace := os.Getenv(envKeyVaultNamespace)
	if vaultNamespace == "" {
		vaultNamespace = defaultTargetNamespace
	}
	return vaultNamespace
}

func getTargetService() string {
	targetService := os.Getenv(envKeyTargetService)
	if targetService == "" {
		targetService = defaultTargetService
	}
	return targetService
}

func (a *app) watchNewPods() {
	watcher, err := a.client.CoreV1().Pods(a.namespace).Watch(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set{
			labelNameAppName: a.targetService,
		}.AsSelector().String(),
	})
	if err != nil {
		slog.Error("Error watching pods", slog.String("error", err.Error()))
		return
	}

	wp := workerpool.NewWorkerPool(
		workerpool.WithDelayedStart(),
	)

	for event := range watcher.ResultChan() {
		wp.MustSchedule(newEventTask(event))
	}
}

type eventTask struct {
	a     *app
	event watch.Event
}

func newEventTask(event watch.Event) *eventTask {
	return &eventTask{
		event: event,
	}
}

func (t *eventTask) Run() {
	pod, ok := t.event.Object.(*core.Pod)
	if !ok {
		// Object is not a pod
		return
	}

	vc, err := newVaultClient(generateVaultAddress(pod.Spec.Containers[0].Ports, pod.Status.PodIP))
	if err != nil {
		slog.Error("Error creating vault client", slog.String("error", err.Error()))
		return
	}

	switch t.event.Type {
	case watch.Added:
		sealed, err := isVaultSealed(vc)
		if err != nil {
			slog.Error("Error checking vault seal status", slog.String("error", err.Error()))
			return
		} else if !sealed {
			// No need to do anything if vault is not sealed
			return
		}

		// Wait until the pod is in a running state
		if pod.Status.Phase != core.PodRunning {
			return
		}
	case watch.Modified:
		// Do something
	case watch.Deleted:
		// Do something
	case watch.Error:
		// Do something
	default:
		slog.Warn("Unknown event type", slog.String("type", string(t.event.Type)))
	}
}

func (a *app) createCryptoKeySecret() error {
	// Create the secret
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cryptoKeySecretName,
			Namespace: a.deployedNamespace,
		},
		Data: map[string][]byte{
			cryptoKeySecretKey: []byte(a.cryptoKey),
		},
	}

	_, err := a.client.CoreV1().Secrets(a.deployedNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating secret: %w", err)
	}

	return nil
}

func (a *app) getCryptoKey() (string, error) {
	secret, err := a.client.CoreV1().Secrets(a.deployedNamespace).Get(context.TODO(), cryptoKeySecretName, metav1.GetOptions{})
	if err != nil {
		slog.Error("Error getting secret", slog.String("error", err.Error()))
		return "", fmt.Errorf("error getting secret: %w", err)
	}

	return string(secret.Data[cryptoKeySecretKey]), nil
}

func getDeployedNamespace() string {
	// Get the namespace that the app is running in
	appNamespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		slog.Error("Error reading app namespace", slog.String("error", err.Error()))
		return "default"
	}

	return string(appNamespace)
}
