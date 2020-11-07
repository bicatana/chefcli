package cmd

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"
)

type Cookbook struct {
	Recipe      []byte
	Awscreds    string ""
	Function    string `yaml:"function"`
	Zipfile     string `yaml:"zipfile"`
	Handler     string `yaml:"handler"`
	ARN         string `yaml:"arn"`
	Runtime     string `yaml:"runtime"`
	Layer       string `yaml:"layer"`
	Description string `yaml:"description"`
	Tfplan      string ""
	//	Bucket   string `yaml:"bucket"`
}

var (
	Now    bool
	Update bool
)

var version string = "1.0.0"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "chefcli",
	Short: "A deployment CLI for Terraform and Serverless functions",
	Long: `

	ChefCLI
		
	A deployment CLI for Terraform and Lambda functions. The CLI can be run from the root folder that contains your Terraform and/or Lambda functions.
	
	Example: 
	chefcli cook terraform`,
	ValidArgs: []string{
		"cook",
		"create",
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

// Function to check and output errors.
func CheckError(err error) {
	if err != nil {
		log.Fatalf("Error: %v", err)
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// Function to check if a file exists
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Zip crawler to add files from a directory and sub-dirs
func AddFiles(w *zip.Writer, basePath, baseInZip string) {
	// check if basePath is not a Dir
	ing, err := ioutil.ReadFile(basePath)
	if err == nil {
		f, err := w.Create(basePath)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write(ing)
	} else {
		// Else, open the Directory
		files, err := ioutil.ReadDir(basePath)
		if err != nil {
			fmt.Println(err)
		}

		for _, file := range files {
			fmt.Println(basePath + file.Name())
			if !file.IsDir() {
				dat, err := ioutil.ReadFile(basePath + file.Name())
				if err != nil {
					fmt.Println(err)
				}

				// Add some files to the archive.
				f, err := w.Create(baseInZip + file.Name())
				if err != nil {
					fmt.Println(err)
				}
				_, err = f.Write(dat)
				if err != nil {
					fmt.Println(err)
				}
			} else if file.IsDir() {

				// Recurse
				newBase := basePath + "/" + file.Name() + "/"
				fmt.Println("Recursing and Adding SubDir: " + file.Name())
				fmt.Println("Recursing and Adding SubDir: " + newBase)

				AddFiles(w, newBase, baseInZip+file.Name()+"/")
			}
		}
	}
}

func init() {
	rootCmd.AddCommand(cookCmd)
	rootCmd.AddCommand(createCmd)
}
