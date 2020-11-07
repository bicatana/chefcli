package cmd

import (
	"archive/zip"
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
		"now",
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
				os.Exit(0)
			}

			// Check Handler name
			if cb.Handler == "" {
				fmt.Println("There is no Handler name. Plese supply a handler name in your Recipe.")
				os.Exit(0)
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
					os.Exit(0)
				}
			}

			// set up the variables with the paths to zip
			lambdaVenvFolder := cb.Function + "/lib/python3.8/site-packages"
			functionFile := cb.Function + ".py"

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
			AddFiles(writer, lambdaVenvFolder, "")

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
				os.Exit(0)
			}

			// Initialize a session that the SDK will use to load
			// credentials from the shared credentials file ~/.aws/credentials.
			sess := session.Must(session.NewSessionWithOptions(session.Options{
				SharedConfigState: session.SharedConfigEnable,
			}))

			svc := lambda.New(sess)

			contents, err := ioutil.ReadFile(cb.Zipfile + ".zip")
			CheckError(err)

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
	},
}

func init() {

	cookLambdaCmd.PersistentFlags().BoolVar(&Now, "now", false, "function description.")
	cookLambdaCmd.PersistentFlags().BoolVar(&Update, "update", false, "function description.")
	// cookLambdaCmd.PersistentFlags().StringVar(&lambdaHandler, "handler", "h", "handler description.")
	// cookLambdaCmd.PersistentFlags().StringVar(&lambdaARN, "arn", "", "resource description.")
	// cookLambdaCmd.PersistentFlags().StringVar(&lambdaRuntime, "runtime", "r", "runtime description.")

	// deployLambdaCmd.MarkPersistentFlagRequired("zipfile")
	// deployLambdaCmd.MarkPersistentFlagRequired("function")
	// deployLambdaCmd.MarkPersistentFlagRequired("handler")
	// deployLambdaCmd.MarkPersistentFlagRequired("runtime")
}
