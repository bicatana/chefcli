package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	tfout    bytes.Buffer
	tfstderr bytes.Buffer
)

var cookTerraformCmd = &cobra.Command{
	Use:     "terraform",
	Short:   "Deploy your Terraform code",
	Long:    `Deploy your Terraform code from the current folder.`,
	Example: `cookcli deploy terraform`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return errors.New("does not require extra arguments")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		command := exec.Command("/usr/local/bin/terraform", "apply -auto-approve")

		command.Stdout = &tfout
		command.Stderr = &tfstderr

		err := command.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + tfstderr.String())
			return
		}
		fmt.Println("Result: " + out.String())

	},
}

func init() {

}
