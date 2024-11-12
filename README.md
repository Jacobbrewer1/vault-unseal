# Vault Unseal

## ❔ Why

HashiCorp Vault provides a few options for auto-unsealing clusters:

    Cloud KMS (AWS, Azure, GCP, and others) (cloud only)
    Hardware Security Modules with PKCS11 (enterprise only)
    Transit Engine via Vault (requires another vault cluster)
    Potentially others

However, depending on your deployment conditions and use-cases of Vault, some of the above may not be feasible (cost,
network connectivity, complexity). This may lead you to want to roll your own unseal functionality, however, it's not
easy to do in a relatively secure manner.

So, what do we need to solve? We want to auto-unseal a vault cluster, by providing the necessary unseal tokens when we
find vault is sealed. We also want to make sure we're sending notifications when this happens, so if vault was unsealed
unintentionally (not patching, upgrades, etc), possibly related to crashing or malicious intent, a human can investigate
at a later time (not 3am in the morning).

## ✔️ Solution

This repository provides a simple solution to the above problem. It is a small GO application that runs in the same K8s
cluster as Vault. It watches the Vault status and if it finds that Vault is sealed, it will attempt to unseal it using
the provided unseal keys.

## 💻 Installation

We recommend using the provided helm chart to install the application. The helm chart can be found in the `helm` folder
of this repository.

## 📝 Configuration

The application will take a configuration file as input. The configuration file should be in JSON format and should
contain the following fields:

```json
{
  "unseal_keys": [
    "key1",
    "key2",
    "key3"
  ]
}
```

## ⚠️ Security

We are bringing encryption to the configuration file in the future.
