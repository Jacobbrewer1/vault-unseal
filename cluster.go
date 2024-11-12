package main

import (
	"context"
	"log/slog"
	"os"
	"time"

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
	watcher, err := a.client.CoreV1().Pods(a.namespace).Watch(a.ctx, metav1.ListOptions{
		LabelSelector: labels.Set{
			labelNameAppName: a.targetService,
		}.AsSelector().String(),
	})
	if err != nil {
		slog.Error("Error watching pods", slog.String(loggingKeyError, err.Error()))
		return
	}

	wp := workerpool.NewWorkerPool(
		workerpool.WithDelayedStart(),
	)

	for event := range watcher.ResultChan() {
		wp.MustSchedule(newEventTask(a, event))
	}
}

type eventTask struct {
	a     *app
	event watch.Event
}

func newEventTask(a *app, event watch.Event) *eventTask {
	return &eventTask{
		a:     a,
		event: event,
	}
}

func (t *eventTask) Run() {
	l := slog.With(
		slog.String("type", string(t.event.Type)),
	)

	pod, ok := t.event.Object.(*core.Pod)
	if !ok {
		// Object is not a pod
		return
	}

	switch t.event.Type {
	case watch.Added:
		// Wait until the pod is in a running state
		//
		// Do this over the course of 10 seconds
		// with a 1-second delay between each check
		if pod.Status.Phase != core.PodRunning {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			found := false
			for {
				select {
				case <-ctx.Done():
					l.Warn("Pod did not enter running state in time")
					return
				default:
					var err error
					pod, err = t.a.client.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{
						ResourceVersion: pod.ResourceVersion,
					})
					if err != nil {
						l.Error("Error getting pod", slog.String(loggingKeyError, err.Error()))
						return
					}

					if pod.Status.Phase == core.PodRunning {
						// Update the pod object to get the latest information
						l.Debug("Pod is running", slog.String("pod", pod.Name))
						found = true
						break
					}
					time.Sleep(1 * time.Second)
				}

				if found {
					break
				}
			}
		}

		vc, err := newVaultClient(generateVaultAddress(pod.Spec.Containers[0].Ports, pod.Status.PodIP))
		if err != nil {
			l.Error("Error creating vault client", slog.String(loggingKeyError, err.Error()))
			return
		}

		sealed, err := isVaultSealed(vc)
		if err != nil {
			l.Error("Error checking vault seal status", slog.String(loggingKeyError, err.Error()))
			return
		} else if !sealed {
			// No need to do anything if vault is not sealed
			return
		}

		l.Info(
			"Pod is running, attempting to unseal vault",
			slog.String("pod", pod.Name),
			slog.String("addr", pod.Status.PodIP),
		)

		// Unseal the vault
		if err := t.a.unsealVault(vc); err != nil {
			l.Error("Error unsealing vault", slog.String(loggingKeyError, err.Error()))
			return
		}
	case watch.Modified:
		// Do something
	case watch.Deleted:
		// Do something
	case watch.Error:
		// Do something
	default:
		l.Warn("Unknown event type", slog.String("type", string(t.event.Type)))
	}
}

func getDeployedNamespace() string {
	// Get the namespace that the app is running in
	appNamespace, err := os.ReadFile(defaultNamespaceFile)
	if err != nil {
		slog.Error("Error reading app namespace", slog.String(loggingKeyError, err.Error()))
		return "default"
	}

	return string(appNamespace)
}
