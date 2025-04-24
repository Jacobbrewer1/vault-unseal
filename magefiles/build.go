//go:build mage

package magefiles

func buildWithBazel() error {
	// BazelBaseArgs is a slice of arguments to be passed to Bazel commands.
	args := BazelBaseArgs()

	// Add the build target to the arguments.
	args = append(args, "build", "//...")

	// Run the Bazel build command with the specified arguments.
	if err := sh.Run("bazel", args...); err != nil {
		return err
	}
}
