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
	lambdazipfile  string
	lambdabucket   string
	lambdafunction string
	lambdahandler  string
	lambdaresource string
	lambdaruntime  string
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

		if lambdabucket == "" || lambdaresource == "" {
			fmt.Println("You must supply a zip file name, bucket name, function name, handler, ARN, and runtime.")
			os.Exit(0)
		}

		// Initialize a session that the SDK will use to load
		// credentials from the shared credentials file ~/.aws/credentials.
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		svc := lambda.New(sess)

		contents, err := ioutil.ReadFile(lambdazipfile + ".zip")
		if err != nil {
			fmt.Println("Could not read " + lambdazipfile + ".zip")
			os.Exit(0)
		}

		lambdacode := &lambda.FunctionCode{
			S3Bucket:        &lambdabucket,
			S3Key:           &lambdazipfile,
			S3ObjectVersion: aws.String(""),
			ZipFile:         contents,
		}

		lambdaargs := &lambda.CreateFunctionInput{
			Code:         lambdacode,
			FunctionName: &lambdafunction,
			Handler:      &lambdahandler,
			Role:         &lambdaresource,
			Runtime:      &lambdaruntime,
		}

		result, err := svc.CreateFunction(lambdaargs)
		if err != nil {
			fmt.Println("Cannot create function: " + err.Error())
		} else {
			fmt.Println(result)
		}
	},
}

func init() {

	deployLambdaCmd.PersistentFlags().StringVar(&lambdazipfile, "zipfile", "", "zipfile description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdabucket, "bucket", "", "bucket description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdafunction, "function", "", "function description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdahandler, "handler", "", "handler description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdaresource, "resource", "", "resource description.")
	deployLambdaCmd.PersistentFlags().StringVar(&lambdaruntime, "runtime", "", "runtime description.")

	deployLambdaCmd.MarkPersistentFlagRequired("zipfile")
	deployLambdaCmd.MarkPersistentFlagRequired("function")
	deployLambdaCmd.MarkPersistentFlagRequired("handler")
	deployLambdaCmd.MarkPersistentFlagRequired("runtime")
}
