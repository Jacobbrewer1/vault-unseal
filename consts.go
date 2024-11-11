package main

const (
	appName = "vault-unseal"

	loggingKeyAppName = "app"

	defaultKubeConfigLocation = "$HOME/.kube/config"
	defaultTargetNamespace    = "vault"
	defaultTargetService      = "vault"
	defaultNamespaceFile      = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	envKeyKubeConfigLocation = "KUBE_CONFIG_LOCATION"
	envKeyVaultNamespace     = "VAULT_NAMESPACE"
	envKeyTargetService      = "TARGET_SERVICE"

	labelNameAppName = "app.kubernetes.io/name"
	//labelNameInstance = "app.kubernetes.io/instance"

	cryptoKeySecretName = "vault-unseal-crypto-key"
	cryptoKeySecretKey  = "crypto-key"
)
