package main

const (
	appName = "vault-unseal"

	loggingKeyAppName   = "app"
	loggingKeyError     = "err"
	loggingKeyPod       = "pod"
	loggingKeyAddr      = "addr"
	loggingKeyPhase     = "phase"
	loggingKeyTaskID    = "task_id"
	loggingKeyEventType = "event_type"
	loggingKeyType      = "type"
	loggingKeySignal    = "signal"
	loggingKeySealed    = "sealed"
	loggingKeyProgress  = "progress"

	defaultKubeConfigLocation = "$HOME/.kube/config"
	defaultTargetNamespace    = "vault"
	defaultTargetService      = "vault"
	defaultNamespaceFile      = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	envKeyKubeConfigLocation = "KUBE_CONFIG_LOCATION"
	envKeyVaultNamespace     = "VAULT_NAMESPACE"
	envKeyTargetService      = "TARGET_SERVICE"

	labelNameAppName = "app.kubernetes.io/name"
)
