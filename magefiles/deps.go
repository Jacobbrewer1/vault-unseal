//go:build mage

package main

import (
	"github.com/magefile/mage/sh"
)

// VendorDeps manages vendoring of Golang dependencies.
func VendorDeps() error {
	if err := sh.Run("go", "mod", "tidy"); err != nil {
		return err
	}

	if err := sh.Run("go", "mod", "verify"); err != nil {
		return err
	}

	if err := sh.Run("bazel", "mod", "tidy"); err != nil {
		return err
	}

	if err := sh.Run("bazel", "run", "//:gazelle"); err != nil {
		return err
	}

	return nil
}
