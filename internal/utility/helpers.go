// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package utility

// IMPORTS {{{
import (
	// "fmt"
	// "os"
	//
	// "github.com/spf13/cobra"
	// "github.com/spf13/viper"

	"github.com/charlievieth/fastwalk"
) // }}}

// ResolvePath takes a path of an unknown type and converts it into an absolute path
//
// determine path type based on prefix, can use `strings.Prefix` or `strings.HasPrefix` for this along with `filepath.Abs` and `os.UserHomeDir`
// if the path is an absolute path (starts with /) leave as is
// if the path is a home folder path (starts with ~ or ~/) expand to absolute path
// if the path is a relative path (starts with ./ or similar) error out
	// Make sure to include . paths though, given the next stipulation
// if the path is "implied" (starts with a word) assume its from the home directory
func ResolvePath() string {
	return ""
}
