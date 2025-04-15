package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hashicorp/vault/api"
	core "k8s.io/api/core/v1"
)

func unsealNewVaultPod(ctx context.Context, l *slog.Logger, target string, keys []string) error {
	vc, err := newVaultClient(target)
	if err != nil {
		return fmt.Errorf("error creating vault client: %w", err)
	}

	for _, key := range keys {
		resp, err := vc.Sys().UnsealWithContext(ctx, key)
		if err != nil {
			return fmt.Errorf("error unsealing vault: %w", err)
		}

		l.Debug(
			"Unsealing vault",
			slog.Bool(loggingKeySealed, resp.Sealed),
			slog.String(loggingKeyProgress, fmt.Sprintf("%d/%d", resp.Progress, resp.T)),
		)

		if resp.Sealed {
			continue
		}

		l.Debug("Vault unsealed")
		break
	}

	return nil
}

func newVaultClient(addr string) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("error creating vault client: %w", err)
	}

	return client, nil
}

func generateVaultAddress(ports []core.ContainerPort, ip string) string {
	const targetScheme = "http"
	for _, port := range ports {
		if port.Name == targetScheme {
			return fmt.Sprintf("%s://%s:%d", port.Name, ip, port.ContainerPort)
		}
	}
	return fmt.Sprintf("%s://%s:8200", targetScheme, ip) // Default to 8200 on http.
}
