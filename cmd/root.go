package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pandacli",
	Short: "A deployment CLI for Terraform and Serverless functions",
	Long: `
	:::::::::     :::     ::::    ::: :::::::::      :::      ::::::::  :::        ::::::::::: 
	:+:    :+:  :+: :+:   :+:+:   :+: :+:    :+:   :+: :+:   :+:    :+: :+:            :+:     
	+:+    +:+ +:+   +:+  :+:+:+  +:+ +:+    +:+  +:+   +:+  +:+        +:+            +:+     
	+#++:++#+ +#++:++#++: +#+ +:+ +#+ +#+    +:+ +#++:++#++: +#+        +#+            +#+     
	+#+       +#+     +#+ +#+  +#+#+# +#+    +#+ +#+     +#+ +#+        +#+            +#+     
	#+#       #+#     #+# #+#   #+#+# #+#    #+# #+#     #+# #+#    #+# #+#            #+#     
	###       ###     ### ###    #### #########  ###     ###  ########  ########## ########### 
		
	A deployment CLI for Terraform and Serverless functions. The CLI can be run from the root folder that contains your Terraform and/or Serverless functions.
	
	Example: 
	pandacli deploy terraform`,
	ValidArgs: []string{
		"terraform",
		"lambda",
	},
	Args:    cobra.OnlyValidArgs,
	Version: version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(deployCmd)
}