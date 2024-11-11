package main

import (
	"fmt"

	"github.com/hashicorp/vault/api"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type vaultInitResponse struct {
	unsealKeys []string
	rootToken  string
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
			return fmt.Sprintf("%s://%s:%d", port.Protocol, ip, port.ContainerPort)
		}
	}

	return fmt.Sprintf("%s://%s:8200", targetScheme, ip) // Default to 8200 on http.
}

func (a *app) keyVault() error {
	// Get the vault pods.
	pods, err := a.client.CoreV1().Pods(a.namespace).List(a.ctx, metav1.ListOptions{
		LabelSelector: labels.Set{
			labelNameAppName: a.targetService,
		}.AsSelector().String(),
	})
	if err != nil {
		return fmt.Errorf("error getting vault pods: %w", err)
	}

	// Get the vault pod.
	if len(pods.Items) == 0 {
		return fmt.Errorf("no vault pods found")
	}

	resp, err := a.vaultInit(generateVaultAddress(pods.Items[0].Spec.Containers[0].Ports, pods.Items[0].Status.PodIP))
	if err != nil {
		return fmt.Errorf("error initializing vault: %w", err)
	}
}

func (a *app) vaultInit(addr string) (*vaultInitResponse, error) {
	res := &api.InitRequest{
		SecretThreshold: 3,
		SecretShares:    2,
	}
	unsealKeys := make([]string, 0)

	vc, err := newVaultClient(addr)
	if err != nil {
		return nil, fmt.Errorf("error creating vault client: %w", err)
	}

	key, err := vc.Sys().Init(res)
	if err != nil {
		return nil, fmt.Errorf("error initializing vault: %w", err)
	}

	for _, key := range key.Keys {
		unsealKeys = append(unsealKeys, key)
	}

	return &vaultInitResponse{
		unsealKeys: unsealKeys,
		rootToken:  key.RootToken,
	}, nil
}
