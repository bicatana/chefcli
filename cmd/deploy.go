package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/spf13/cobra"
)

var (
	profileFlag string
	yesFlag     bool
	deleteFlag  bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy Terraform and Lambda function ",
	Long:  `This deploys your Terraform code and Lambda function.`,
	ValidArgs: []string{
		"profile",
		"terraform",
		"lambda",
	},
	Args:                  cobra.OnlyValidArgs,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("deploy initialized")

		//Kudos to Stefan Sundin for the AWS Key Rotate piece. See https://github.com/stefansundin/aws-rotate-key
		// Get credentials
		credentialsPath := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
		if len(credentialsPath) == 0 {
			usr, err := user.Current()
			if err != nil {
				fmt.Println("Error: Could not locate your home directory. Please set the AWS_SHARED_CREDENTIALS_FILE environment variable.")
				os.Exit(1)
			}
			credentialsPath = fmt.Sprintf("%s/.aws/credentials", usr.HomeDir)
		}

		credentialsProvider := credentials.NewSharedCredentials(credentialsPath, profileFlag)
		creds, err := credentialsProvider.Get()
		CheckError(err)
		fmt.Printf("Using access key %s from profile \"%s\".\n", creds.AccessKeyID, profileFlag)

		// Read credentials file
		bytes, err := ioutil.ReadFile(credentialsPath)
		CheckError(err)
		credentialsText := string(bytes)
		// Check if we can find the credentials in the file
		// It's better to detect a malformed file now than after we have created the new key
		re_aws_access_key_id := regexp.MustCompile(fmt.Sprintf(`(?m)^aws_access_key_id *= *%s`, regexp.QuoteMeta(creds.AccessKeyID)))
		re_aws_secret_access_key := regexp.MustCompile(fmt.Sprintf(`(?m)^aws_secret_access_key *= *%s`, regexp.QuoteMeta(creds.SecretAccessKey)))
		if !re_aws_access_key_id.MatchString(credentialsText) || !re_aws_secret_access_key.MatchString(credentialsText) {
			fmt.Println()
			fmt.Printf("Unable to find your credentials in %s.\n", credentialsPath)
			fmt.Println("Please make sure your file is formatted like the following:")
			fmt.Println()
			fmt.Printf("aws_access_key_id=%s\n", creds.AccessKeyID)
			fmt.Println("aws_secret_access_key=...")
			fmt.Println()
			os.Exit(1)
		}

		// Create session
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Profile:           profileFlag,
		}))

		// sts get-caller-identity
		stsClient := sts.New(sess)
		respGetCallerIdentity, err := stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err != nil {
			fmt.Println("Error getting caller identity. Is the key disabled?")
			fmt.Println()
			CheckError(err)
		}
		fmt.Printf("Your user ARN is: %s\n", *respGetCallerIdentity.Arn)

		// iam list-access-keys
		// If the UserName field is not specified, the UserName is determined implicitly based on the AWS access key ID used to sign the request.
		iamClient := iam.New(sess)
		respListAccessKeys, err := iamClient.ListAccessKeys(&iam.ListAccessKeysInput{})
		CheckError(err)

		// Print key information
		fmt.Printf("You have %d access key%v associated with your user:\n", len(respListAccessKeys.AccessKeyMetadata), pluralize(len(respListAccessKeys.AccessKeyMetadata)))
		for _, key := range respListAccessKeys.AccessKeyMetadata {
			respAccessKeyLastUsed, err2 := iamClient.GetAccessKeyLastUsed(&iam.GetAccessKeyLastUsedInput{
				AccessKeyId: key.AccessKeyId,
			})
			if err2 != nil {
				fmt.Printf("- %s (%s, created %s)\n", *key.AccessKeyId, *key.Status, key.CreateDate)
			} else if respAccessKeyLastUsed.AccessKeyLastUsed.LastUsedDate == nil {
				fmt.Printf("- %s (%s, created %s, never used)\n", *key.AccessKeyId, *key.Status, key.CreateDate)
			} else {
				fmt.Printf("- %s (%s, created %s, last used %s for service %s in %s)\n", *key.AccessKeyId, *key.Status, key.CreateDate, respAccessKeyLastUsed.AccessKeyLastUsed.LastUsedDate, *respAccessKeyLastUsed.AccessKeyLastUsed.ServiceName, *respAccessKeyLastUsed.AccessKeyLastUsed.Region)
			}
		}
		fmt.Println()

		if len(respListAccessKeys.AccessKeyMetadata) == 2 {
			keyIndex := 0
			if *respListAccessKeys.AccessKeyMetadata[0].AccessKeyId == creds.AccessKeyID {
				keyIndex = 1
			}

			if yesFlag == false {
				fmt.Println("You have two access keys, which is the max number of access keys.")
				fmt.Printf("Do you want to delete %s and create a new key? [yN] ", *respListAccessKeys.AccessKeyMetadata[keyIndex].AccessKeyId)
				if *respListAccessKeys.AccessKeyMetadata[keyIndex].Status == "Active" {
					fmt.Printf("\nWARNING: This key is currently Active! ")
				}
				reader := bufio.NewReader(os.Stdin)
				yn, err2 := reader.ReadString('\n')
				CheckError(err2)
				if yn[0] != 'y' && yn[0] != 'Y' {
					os.Exit(1)
				}
			}

			_, err2 := iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				AccessKeyId: respListAccessKeys.AccessKeyMetadata[keyIndex].AccessKeyId,
			})
			CheckError(err2)
			fmt.Printf("Deleted access key %s.\n", *respListAccessKeys.AccessKeyMetadata[keyIndex].AccessKeyId)
		} else if yesFlag == false {
			cleanupAction := "deactivate"
			if deleteFlag {
				cleanupAction = "delete"
			}
			fmt.Printf("Do you want to create a new key and %s %s? [yN] ", cleanupAction, *respListAccessKeys.AccessKeyMetadata[0].AccessKeyId)
			reader := bufio.NewReader(os.Stdin)
			yn, err2 := reader.ReadString('\n')
			CheckError(err2)
			if yn[0] != 'y' && yn[0] != 'Y' {
				os.Exit(1)
			}
		}

		// Create the new access key
		// If you do not specify a user name, IAM determines the user name implicitly based on the AWS access key ID signing the request.
		respCreateAccessKey, err := iamClient.CreateAccessKey(&iam.CreateAccessKeyInput{})
		CheckError(err)
		fmt.Printf("Created access key %s.\n", *respCreateAccessKey.AccessKey.AccessKeyId)

		// Replace key pair in credentials file
		// This search & replace does not limit itself to the specified profile, which may be useful if the user is using the same key in multiple profiles
		credentialsText = re_aws_access_key_id.ReplaceAllString(credentialsText, `aws_access_key_id=`+*respCreateAccessKey.AccessKey.AccessKeyId)
		credentialsText = re_aws_secret_access_key.ReplaceAllString(credentialsText, `aws_secret_access_key=`+*respCreateAccessKey.AccessKey.SecretAccessKey)

		// Verify that the regexp actually replaced something
		if !strings.Contains(credentialsText, *respCreateAccessKey.AccessKey.AccessKeyId) || !strings.Contains(credentialsText, *respCreateAccessKey.AccessKey.SecretAccessKey) {
			fmt.Println("Failed to replace old access key. Aborting.")
			fmt.Printf("Please verify that the file %s is formatted correctly.\n", credentialsPath)
			// Delete the key we created
			_, err2 := iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				AccessKeyId: respCreateAccessKey.AccessKey.AccessKeyId,
			})
			CheckError(err2)
			fmt.Printf("Deleted access key %s.\n", *respCreateAccessKey.AccessKey.AccessKeyId)
			os.Exit(1)
		}

		// Write new file
		err = ioutil.WriteFile(credentialsPath, []byte(credentialsText), 0600)
		CheckError(err)
		fmt.Printf("Wrote new key pair to %s\n", credentialsPath)

		// Delete the old key if flag is set, otherwise deactivate it
		if deleteFlag {
			_, err := iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				AccessKeyId: &creds.AccessKeyID,
			})
			CheckError(err)
			fmt.Printf("Deleted old access key %s.\n", creds.AccessKeyID)
		} else {
			_, err = iamClient.UpdateAccessKey(&iam.UpdateAccessKeyInput{
				AccessKeyId: &creds.AccessKeyID,
				Status:      aws.String("Inactive"),
			})
			CheckError(err)
			fmt.Printf("Deactivated old access key %s.\n", creds.AccessKeyID)
			fmt.Println("Please make sure this key is not used elsewhere.")
		}
		fmt.Println("Please note that it may take a minute for your new access key to propagate in the AWS control plane.")

	},
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func init() {

	deployCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "AWS Profile Description.")
	deployCmd.PersistentFlags().BoolVar(&yesFlag, "yes", false, "Yes Flag Description.")
	deployCmd.PersistentFlags().BoolVar(&deleteFlag, "delete", false, "Delete Flag Description.")

	deployCmd.AddCommand(deployLambdaCmd)
	deployCmd.AddCommand(terraformDeployCmd)
}
