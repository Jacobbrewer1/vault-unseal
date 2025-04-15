package main

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/jacobbrewer1/web"
	"github.com/jacobbrewer1/web/cache"
	"github.com/jacobbrewer1/web/logging"
	core "k8s.io/api/core/v1"
	kubeCache "k8s.io/client-go/tools/cache"
)

// watchVaultPods watches for new pods and distributes them to the correct handler that will determine if it is a
// Vault pod. If it is, it will attempt to unseal the vault using the unseal keys provided.
func watchVaultPods(
	l *slog.Logger,
	podInformer kubeCache.SharedIndexInformer,
	hashBucket cache.HashBucket,
	unsealKeys []string,
) web.AsyncTaskFunc {
	return func(ctx context.Context) {
		if _, err := podInformer.AddEventHandler(kubeCache.ResourceEventHandlerFuncs{
			AddFunc: newPodHandler(
				ctx,
				logging.LoggerWithComponent(l, "new-pod-handler"),
				hashBucket,
				unsealKeys,
			),
			UpdateFunc: updatePodHandler(
				ctx,
				logging.LoggerWithComponent(l, "update-pod-handler"),
				hashBucket,
				unsealKeys,
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
	}
}

// newPodHandler is the handler for new pods. It will check if the pod is a Vault pod and if it is sealed. If it is, it
// will attempt to unseal the vault using the unseal keys provided.
func newPodHandler(ctx context.Context, l *slog.Logger, hashBucket cache.HashBucket, unsealKeys []string) func(any) {
	return func(podObj any) {
		pod, ok := podObj.(*core.Pod)
		if !ok {
			return
		}

		l = l.With(
			slog.String(loggingKeyPod, pod.Name),
		)

		if pod.GetNamespace() != targetNamespace {
			return
		}

		if !hashBucket.InBucket(pod.Name) {
			return
		}

		if !isVaultPod(pod) {
			return
		}
		if !isVaultPodSealed(pod) {
			return
		}

		l.Info("Updated Vault pod detected, attempting to unseal vault", slog.String(loggingKeyPod, pod.Name))

		if err := unsealNewVaultPod( // nolint:revive // Traditional error handling
			ctx,
			l,
			generateVaultAddress(pod.Spec.Containers[0].Ports, pod.Status.PodIP),
			unsealKeys,
		); err != nil {
			l.Error("Error unsealing vault", slog.String(loggingKeyError, err.Error()))
			return
		}
	}
}

// updatePodHandler is the handler for updated pods. It will check if the pod is a Vault pod and if it is sealed. If it
// is, it will attempt to unseal the vault using the unseal keys provided.
func updatePodHandler(ctx context.Context, l *slog.Logger, hashBucket cache.HashBucket, unsealKeys []string) func(any, any) {
	return func(_, newObj any) {
		pod, ok := newObj.(*core.Pod)
		if !ok {
			return
		}

		l = l.With(
			slog.String(loggingKeyPod, pod.Name),
		)

		if pod.GetNamespace() != targetNamespace {
			return
		}

		if !hashBucket.InBucket(pod.Name) {
			return
		}

		if !isVaultPod(pod) {
			return
		}
		if !isVaultPodSealed(pod) {
			return
		}

		l.Info("Updated Vault pod detected, attempting to unseal vault", slog.String(loggingKeyPod, pod.Name))

		if err := unsealNewVaultPod( // nolint:revive // Traditional error handling
			ctx,
			l,
			generateVaultAddress(pod.Spec.Containers[0].Ports, pod.Status.PodIP),
			unsealKeys,
		); err != nil {
			l.Error("Error unsealing vault", slog.String(loggingKeyError, err.Error()))
			return
		}
	}
}

// isVaultPod checks if the pod is a Vault pod by checking the labels.
func isVaultPod(pod *core.Pod) bool {
	return pod.Labels["app.kubernetes.io/name"] == "vault"
}

// isVaultPodSealed checks if the pod is a Vault pod and if it is sealed by checking the labels.
func isVaultPodSealed(pod *core.Pod) bool {
	sealed, ok := pod.Labels["vault-sealed"]
	if !ok {
		return false
	}

	isSealed, err := strconv.ParseBool(sealed)
	if err != nil {
		return false
	}

	return isSealed
}
