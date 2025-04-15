package main

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/google/uuid"
	"github.com/jacobbrewer1/web"
	"github.com/jacobbrewer1/web/cache"
	"github.com/jacobbrewer1/web/logging"
	core "k8s.io/api/core/v1"
	kubeCache "k8s.io/client-go/tools/cache"
)

func (a *App) watchNewPods(l *slog.Logger) web.AsyncTaskFunc {
	return func(ctx context.Context) {
		podInformer := a.base.PodInformer()

		if _, err := podInformer.AddEventHandler(kubeCache.ResourceEventHandlerFuncs{
			AddFunc: newPodHandler(
				ctx,
				logging.LoggerWithComponent(l, "new-pod-handler"),
				a.base.ServiceEndpointHashBucket(),
				a.config.unsealKeys,
			),
		}); err != nil {
			l.Error("Error adding event handler", slog.String(loggingKeyError, err.Error()))
			return
		}

		if err := podInformer.SetWatchErrorHandler(func(r *kubeCache.Reflector, err error) {
			l.Error("Error watching pods", slog.String(loggingKeyError, err.Error()))
		}); err != nil {
			l.Error("Error setting watch error handler", slog.String(loggingKeyError, err.Error()))
		}

		podInformer.Run(ctx.Done())

		<-ctx.Done()
	}
}

func newPodHandler(ctx context.Context, l *slog.Logger, hashBucket cache.HashBucket, unsealKeys []string) func(any) {
	return func(podObj any) {
		l = l.With(
			slog.String(loggingKeyTaskID, uuid.New().String()),
			slog.String(loggingKeyEventType, "pod"),
		)

		pod, ok := podObj.(*core.Pod)
		if !ok {
			return
		}

		if pod.GetNamespace() != targetNamespace {
			return
		}

		if !hashBucket.InBucket(pod.Name) {
			return
		}

		// Is this a vault pod?
		if pod.Labels["app.kubernetes.io/name"] != "vault" {
			l.Debug("Pod is not a vault pod, ignoring", slog.String(loggingKeyPod, pod.Name))
			return
		}

		if isSealed, _ := strconv.ParseBool(pod.Labels["vault-sealed"]); !isSealed {
			l.Debug("Pod is not sealed, ignoring", slog.String(loggingKeyPod, pod.Name))
			return
		}

		l.Info("New pod added, attempting to unseal vault", slog.String(loggingKeyPod, pod.Name))

		if err := unsealNewVaultPod( // nolint:revive // Traditional error handling
			ctx,
			l,
			generateVaultAddress(pod.Spec.Containers[0].Ports, pod.Status.HostIP),
			unsealKeys,
		); err != nil {
			l.Error("Error unsealing vault", slog.String(loggingKeyError, err.Error()))
			return
		}
	}
}
