//go:build mage

package magefiles

var Aliases = map[string]interface{}{
	"fixit": VendorDeps,
}
