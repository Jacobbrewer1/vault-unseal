package main

import (
	"log/slog"
	"os"

	"github.com/google/uuid"
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
		wp.MustSchedule(newEventTask(a, wp, event))
	}

	slog.Debug("Watcher channel closed")
	wp.Stop()
}

type eventTask struct {
	id    string
	a     *app
	wp    *workerpool.WorkerPool
	event watch.Event
}

func newEventTask(a *app, wp *workerpool.WorkerPool, event watch.Event) *eventTask {
	return &eventTask{
		id:    uuid.New().String(),
		a:     a,
		wp:    wp,
		event: event,
	}
}

func (t *eventTask) Run() {
	l := slog.With(
		slog.String("task_id", t.id),
		slog.String("event_type", string(t.event.Type)),
	)

	pod, ok := t.event.Object.(*core.Pod)
	if !ok {
		// Object is not a pod
		return
	}

	switch t.event.Type {
	case watch.Added, watch.Error:
		// Is the pod still in the cluster? This is to prevent retry attempts from getting stuck
		if pod.ObjectMeta.DeletionTimestamp != nil {
			l.Info("Pod is being deleted, aborting")
			return
		}

		// If the pod is not running, re-schedule the task
		if pod.Status.Phase != core.PodRunning {
			t.wp.MustSchedule(t)
			return
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
	case watch.Modified, watch.Deleted, watch.Bookmark:
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
