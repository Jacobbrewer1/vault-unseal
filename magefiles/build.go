//go:build mage

package main

import "github.com/magefile/mage/sh"

type Build mg.Namespace

func (b Build) All() error {
	if err := buildWithBazel(); err != nil {
		return err
	}
	return nil
}

func buildWithBazel() error {
	// BazelBaseArgs is a slice of arguments to be passed to Bazel commands.
	args := BazelBaseArgs()

	// Add the build target to the arguments.
	args = append(args, "build", "//...")

	// Run the Bazel build command with the specified arguments.
	if err := sh.Run("bazel", args...); err != nil {
		return err
	}
	return nil
}
