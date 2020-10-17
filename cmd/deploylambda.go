package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/spf13/cobra"
)

var (
	lambdaZipFile  string
	lambdaBucket   string
	lambdaFunction string
	lambdaHandler  string
	lambdaARN      string
	lambdaRuntime  string
)

var deployLambdaCmd = &cobra.Command{
	Use:     "lambda",
	Short:   "Deploy your Lambda code",
	Long:    `Deploy your Lambda code from the current folder.`,
	Example: `pandacli deploy lambda`,
	ValidArgs: []string{
		"zipfile",
		"bucket",
		"function",
		"handler",
		"resource",
		"runtime",
	},
	Args: cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Lambda called")
		//	fmt.Println(args)

		if lambdaBucket == "" || lambdaARN == "" {
			fmt.Println("You must supply a zip file name, bucket name, function name, handler, ARN, and runtime.")
			os.Exit(0)
		}

		// Initialize a session that the SDK will use to load
		// credentials from the shared credentials file ~/.aws/credentials.
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		svc := lambda.New(sess)

		contents, err := ioutil.ReadFile(lambdaZipFile + ".zip")
		CheckError(err)

		lambdaCode := &lambda.FunctionCode{
			S3Bucket:        &lambdaBucket,
			S3Key:           &lambdaZipFile,
			S3ObjectVersion: aws.String(""),
			ZipFile:         contents,
		}

		lambdaArgs := &lambda.CreateFunctionInput{
			Code:         lambdaCode,
			FunctionName: &lambdaFunction,
			Handler:      &lambdaHandler,
			Role:         &lambdaARN,
			Runtime:      &lambdaRuntime,
		}

		result, err := svc.CreateFunction(lambdaArgs)
		if err != nil {
			CheckError(err)
		} else {
			fmt.Println(result)
		}
	},
}

func init() {

	deployLambdaCmd.PersistentFlags().StringVar(&lambdaZipFile, "zipfile", "", "zipfile description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdaBucket, "bucket", "", "bucket description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdaFunction, "function", "", "function description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdaHandler, "handler", "", "handler description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdaARN, "arn", "", "resource description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdaRuntime, "runtime", "", "runtime description.")

	deployLambdaCmd.MarkPersistentFlagRequired("zipfile")
	deployLambdaCmd.MarkPersistentFlagRequired("function")
	deployLambdaCmd.MarkPersistentFlagRequired("handler")
	deployLambdaCmd.MarkPersistentFlagRequired("runtime")
}
