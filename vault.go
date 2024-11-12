package main

import (
	"fmt"
	"log/slog"

	"github.com/hashicorp/vault/api"
	core "k8s.io/api/core/v1"
)

func newVaultClient(addr string) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("error creating vault client: %w", err)
	}

	return client, nil
}

func isVaultSealed(client *api.Client) (bool, error) {
	sealed, err := client.Sys().SealStatus()
	if err != nil {
		return false, fmt.Errorf("error checking vault seal status: %w", err)
	}

	return sealed.Sealed, nil
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

func (a *app) unsealVault(vc *api.Client) error {
	for _, key := range a.unsealKeys {
		resp, err := vc.Sys().Unseal(key)
		if err != nil {
			return fmt.Errorf("error unsealing vault: %w", err)
		}

		slog.Info(
			"Unsealing vault",
			slog.String("key", key),
			slog.Bool("sealed", resp.Sealed),
			slog.String("progress", fmt.Sprintf("%d/%d", resp.Progress, resp.T)),
		)

		if !resp.Sealed {
			slog.Info("Vault unsealed")
			break
		}
	}

	return nil
}
