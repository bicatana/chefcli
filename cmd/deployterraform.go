package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var terraformCmd = &cobra.Command{
	Use:     "terraform",
	Short:   "Deploy your Terraform code",
	Long:    `Deploy your Terraform code from the current folder.`,
	Example: `pandacli deploy terraform`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return errors.New("requires a name argument")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		command := exec.Command("/usr/local/bin/terraform", "apply -auto-approve")

		var out bytes.Buffer
		var stderr bytes.Buffer

		command.Stdout = &out
		command.Stderr = &stderr

		err := command.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			return
		}
		fmt.Println("Result: " + out.String())

	},
}

func init() {

}
