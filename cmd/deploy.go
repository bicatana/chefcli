package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy Terraform and Lambda function ",
	Long:  `This deploys your Terraform code and Lambda function.`,
	ValidArgs: []string{
		"terraform",
		"lambda",
	},
	Args:                  cobra.OnlyValidArgs,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("deploy initialized")
	},
}

func init() {
	deployCmd.AddCommand(deployLambdaCmd)
	deployCmd.AddCommand(terraformDeployCmd)
}
