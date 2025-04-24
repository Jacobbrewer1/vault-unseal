//go:build mage

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	gcpStorageHost = "https://storage.googleapis.com"
)

const (
	envGithubActions                 = "GITHUB_ACTIONS"
	envRepositoryName                = "GITHUB_REPOSITORY"
	envRepositoryOwner               = "GITHUB_REPOSITORY_OWNER"
	envGCPServiceAccountJsonLocation = "GOOGLE_APPLICATION_CREDENTIALS_JSON_PATH"
)

// IsGithubRunner checks if the code is running on a GitHub Actions runner.
var IsGithubRunner = sync.OnceValue(func() bool {
	got := os.Getenv(envGithubActions)
	if got == "" {
		return false
	}

	isAction, err := strconv.ParseBool(got)
	if err != nil {
		return false
	}
	return isAction
})

// RepositoryName returns the name of the repository.
var RepositoryName = sync.OnceValue(func() string {
	name := os.Getenv(envRepositoryName)
	if name == "" {
		return ""
	}
	return name
})

// RepositoryOwner returns the owner of the repository.
var RepositoryOwner = sync.OnceValue(func() string {
	owner := os.Getenv(envRepositoryOwner)
	if owner == "" {
		return ""
	}
	return owner
})

// RepositoryNameOnly returns the name of the repository without the owner.
var RepositoryNameOnly = sync.OnceValue(func() string {
	name := RepositoryName()
	if name == "" {
		return ""
	}

	owner := RepositoryOwner()
	if owner == "" {
		return ""
	}

	name = strings.TrimPrefix(name, owner+"/")
	return name
})

// GCPServiceAccountJsonLocation returns the location of the GCP service account JSON file.
var GCPServiceAccountJsonLocation = sync.OnceValue(func() string {
	location := os.Getenv(envGCPServiceAccountJsonLocation)
	if location == "" {
		return ""
	}
	return location
})

// BazelBaseArgs returns the base arguments for Bazel commands.
var BazelBaseArgs = sync.OnceValue(func() []string {
	args := make([]string, 0)

	if IsGithubRunner() && RepositoryNameOnly() != "" && GCPServiceAccountJsonLocation() != "" {
		args = append(args, fmt.Sprintf("--remote_cache=%s/%s", gcpStorageHost, RepositoryNameOnly()))
		args = append(args, fmt.Sprintf("--google_credentials=%s", GCPServiceAccountJsonLocation()))
	}

	return args
})
