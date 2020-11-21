package cmd

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	lambdaCookbook   string
	functionFile     string
	lambdaVenvFolder string
)

var cookLambdaCmd = &cobra.Command{
	Use:     "lambda",
	Short:   "Cook your Lambda code",
	Long:    "Cook your Lambda code from the current folder.",
	Example: "chefcli cook lambda",
	ValidArgs: []string{
		"venv",
		"new",
		"update",
	},
	Args: cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {

		// Make sure we call Cook Lambda.
		fmt.Println("Cooking Lambda function.")

		// cb is going to be our Cookbook
		cb := Cookbook{}

		// Check if recipe.yml exists and ensure the Cookbook is picked up
		if !FileExists("recipe.yml") && !FileExists("recipe.yaml") {
			fmt.Println("There is no recipe.yml or recipe.yaml file present. Please set one up.")
			os.Exit(0)
		} else {
			if FileExists("recipe.yml") {
				lambdaCookbook, err := ioutil.ReadFile("recipe.yml")
				err = yaml.Unmarshal(lambdaCookbook, &cb)
				if err != nil {
					log.Fatalf("Unmarshal: %v", err)
				}
				cb.Recipe = lambdaCookbook
			} else {
				lambdaCookbook, err := ioutil.ReadFile("recipe.yaml")
				err = yaml.Unmarshal(lambdaCookbook, &cb)
				if err != nil {
					log.Fatalf("Unmarshal: %v", err)
				}
				cb.Recipe = lambdaCookbook
			}

			// Check Function name
			if cb.Function == "" {
				fmt.Println("There is no Function name. Plese supply a function name in your Recipe.")
				os.Exit(1)
			}

			// Check Handler name
			if cb.Handler == "" {
				fmt.Println("There is no Handler name. Plese supply a handler name in your Recipe.")
				os.Exit(1)
			} else {
				cb.Handler = cb.Function + "." + cb.Handler
			}

			// Check the Runtime
			if cb.Runtime == "" {
				fmt.Println("There is no Runtime specified.")
				fmt.Println("Defaulting to Python 3.8 .")
				cb.Runtime = "python3.8"
			}

			// Check for the Zip File Name
			if cb.Zipfile == "" {
				if FileExists(cb.Function + ".py") {
					cb.Zipfile = cb.Function
					fmt.Println("No Zip file found. Function file found. Checking for Virtual Env.")
				}
				// Check for the dependencies folder
				if _, err := os.Stat(cb.Function + "/lib/python3.8/site-packages"); os.IsNotExist(err) {
					fmt.Println("No Virtual Env found. Exiting.")
					CheckError(err)
					os.Exit(1)
				}
			}

			//Ensuring that the user wants to leverage the Virtual Environment in the ZIP
			if Venv {
				fmt.Println("Virtual environments are for local development. Are you sure you want to include them in your Lambda ZIP package? [yes/no]")
				input := bufio.NewScanner(os.Stdin)
				input.Scan()
				//				fmt.Printf("%v", input.Text())
				if input.Text() != "yes" && input.Text() != "no" {
					fmt.Println("Please type either 'yes' or 'no'.")
					os.Exit(1)
				}
				if input.Text() == "no" {
					fmt.Println("Understood. Use 'chefcli cook layer' to build the relevant layers for your Lambda.")
					os.Exit(1)
				} else {
					fmt.Println("Understood. Proceeding with the virtual environment...")
					// set up the Venv variable with the path to the site-packages
					lambdaVenvFolder = cb.Function + "/lib/python3.8/site-packages"
				}
			}
			//setting up the file name for the Python Lambda
			functionFile = cb.Function + ".py"

			// Get a Buffer to Write To
			outFile, err := os.Create(cb.Function + ".zip")
			if err != nil {
				fmt.Println(err)
			}
			defer outFile.Close()
			// Create a new zip archive.
			writer := zip.NewWriter(outFile)
			// Add some files to the archive.
			AddFiles(writer, functionFile, "/")
			if Venv {
				AddFiles(writer, lambdaVenvFolder, "")
			}

			if err != nil {
				fmt.Println(err)
			}
			// Make sure to check the error on Close.
			err = writer.Close()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("ZIP archive is ready. The name of the archive is " + cb.Function + ".zip")
			}

			// Check For ARN, Function and Handler names
			if cb.ARN == "" {
				fmt.Println("You must supply an ARN.")
				os.Exit(1)
			}

			contents, err := ioutil.ReadFile(cb.Zipfile + ".zip")
			CheckError(err)
			// Only deploy if the Now flag is set
			if New {
				// Initialize a session that the SDK will use to load
				// credentials from the shared credentials file ~/.aws/credentials.
				sess := session.Must(session.NewSessionWithOptions(session.Options{
					SharedConfigState: session.SharedConfigEnable,
				}))

				svc := lambda.New(sess)

				lambdaCode := &lambda.FunctionCode{
					//	S3Bucket:        &cb.Bucket,
					//	S3Key:           &cb.Zipfile,
					//	S3ObjectVersion: aws.String("1"),
					ZipFile: contents,
				}

				lambdaArgs := &lambda.CreateFunctionInput{
					Code:         lambdaCode,
					FunctionName: &cb.Function,
					Handler:      &cb.Handler,
					Role:         &cb.ARN,
					Runtime:      &cb.Runtime,
				}

				result, err := svc.CreateFunction(lambdaArgs)
				if err != nil {
					CheckError(err)
				} else {
					fmt.Println(result)
				}
			}

			if Update {
				// Initialize a session that the SDK will use to load
				// credentials from the shared credentials file ~/.aws/credentials.
				sess := session.Must(session.NewSessionWithOptions(session.Options{
					SharedConfigState: session.SharedConfigEnable,
				}))

				svc := lambda.New(sess)

				input := &lambda.UpdateFunctionCodeInput{
					FunctionName: &cb.Function,
					ZipFile:      contents,
				}

				result, err := svc.UpdateFunctionCode(input)
				if CheckAWSError(err) {
					os.Exit(1)
				}
				fmt.Println(result)
			}
		}
	},
}

func init() {

	cookLambdaCmd.PersistentFlags().BoolVar(&New, "new", false, "cook new Lambda function.")
	cookLambdaCmd.PersistentFlags().BoolVar(&Update, "update", false, "cook update for a Lambda function.")
	cookLambdaCmd.PersistentFlags().BoolVar(&Venv, "venv", false, "add virtual environment packages to the ZIP archive.")

	// keeping for reference
	// deployLambdaCmd.MarkPersistentFlagRequired("")
}
