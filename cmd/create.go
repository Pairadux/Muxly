// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"

	"github.com/Pairadux/tms/internal/forms"

	"github.com/spf13/cobra"
) // }}}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a session",
	Long: `Create a session

An interactive prompt for creating a session.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: implement this command
		fmt.Println("create called")
		var burger string
		var name string
		var instructions string
		var toppings []string
		var sauceLevel int
		var discount bool

		forms.CreateForm(&burger, &name, &instructions, &toppings, &sauceLevel, &discount).Run()

		fmt.Printf("burger: %s, name %s, instructions %s, toppings %s, sauceLevel %d, discount %v\n", burger, name, instructions, toppings, sauceLevel, discount)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
