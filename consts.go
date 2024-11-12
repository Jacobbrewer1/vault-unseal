package main

const (
	appName = "vault-unseal"

	loggingKeyAppName = "app"
	loggingKeyError   = "err"

	defaultKubeConfigLocation = "$HOME/.kube/config"
	defaultTargetNamespace    = "vault"
	defaultTargetService      = "vault"
	defaultNamespaceFile      = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	envKeyKubeConfigLocation = "KUBE_CONFIG_LOCATION"
	envKeyVaultNamespace     = "VAULT_NAMESPACE"
	envKeyTargetService      = "TARGET_SERVICE"

	labelNameAppName = "app.kubernetes.io/name"
)
