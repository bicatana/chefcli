package cmd

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	layerCookbook string
	layerFolder   string
	layerout      bytes.Buffer
	layerstderr   bytes.Buffer
)

var cookLayerCmd = &cobra.Command{
	Use:     "layer",
	Short:   "Cook your Lambda Layers",
	Long:    "Cook your Lambda Layers from the current folder.",
	Example: "chefcli cook layer",
	ValidArgs: []string{
		"now",
	},
	Args: cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {

		// Make sure we call Cook Lambda.
		fmt.Println("Cooking Lambda Layers.")

		// cb is going to be our Cookbook
		cb := Cookbook{}

		// Check if recipe.yml exists and ensure the Cookbook is picked up
		if !FileExists("recipe.yml") && !FileExists("recipe.yaml") {
			fmt.Println("There is no recipe.yml or recipe.yaml file present. Please set one up.")
			os.Exit(0)
		} else {
			if FileExists("recipe.yml") {
				layerCookbook, err := ioutil.ReadFile("recipe.yml")
				err = yaml.Unmarshal(layerCookbook, &cb)
				if err != nil {
					log.Fatalf("Unmarshal: %v", err)
				}
				cb.Recipe = layerCookbook
			} else {
				layerCookbook, err := ioutil.ReadFile("recipe.yaml")
				err = yaml.Unmarshal(layerCookbook, &cb)
				if err != nil {
					log.Fatalf("Unmarshal: %v", err)
				}
				cb.Recipe = layerCookbook
			}

			// Check Layer name
			if cb.Layer == "" {
				fmt.Println("There is no Layer name. Plese supply a layer name in your Recipe.")
				os.Exit(0)
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

			// Check For ARN, Function and Handler names
			if cb.ARN == "" {
				fmt.Println("You must supply an ARN.")
				os.Exit(0)
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

			// ensure description is blank if it is empty
			if cb.Description == "" {
				cb.Description = ""
			}

			// set up the variables with the paths to zip
			// Check the folder named python for layer building
			layerFolder = "python"
			_, err := os.Stat(layerFolder)
			if os.IsExist(err) {
				fmt.Println("A folder named python already exists. ChefCLI uses that folder for handling the layer cooking, please move it or delete it so that ChefCLI can use it.")
				os.Exit(0)
			}
			errDir := os.MkdirAll("python/lib/"+cb.Runtime+"/site-packages", 0755)
			CheckError(errDir)

			// check that a requirements.txt file exists
			if !FileExists("requirements.txt") {
				fmt.Println("There is no requirements.txt file present. Please set one up with the required packages for your packages.")
				os.Exit(0)
			}

			// Old script, keeping this for reference
			//			script := []byte("#!/bin/bash\nexport PKG_DIR='python'\nrm -rf ${PKG_DIR} && mkdir -p ${PKG_DIR}\ndocker run --rm -v $(pwd):/layers -w /layers lambci/lambda:build-python3.8 \\ \n pip3 install -r requirements.txt --no-deps -t ${PKG_DIR}")

			// Script to leverage Docker to build layer libraries, see https://aws.amazon.com/premiumsupport/knowledge-center/lambda-layer-simulated-docker/ for more information
			script := []byte("#!/bin/bash\ndocker run -v \"$PWD\":/var/task \"lambci/lambda:build-python3.8\" /bin/sh -c \"pip3 install -r requirements.txt -t python/lib/python3.8/site-packages/ --no-deps\"")

			// write script
			err = ioutil.WriteFile("python/get_layer_packages.sh", script, 0644)
			CheckError(err)

			// execute command
			command := exec.Command("/bin/sh", "python/get_layer_packages.sh")
			command.Stdout = &layerout
			command.Stderr = &layerstderr

			// run execution
			err = command.Run()
			if err != nil {
				fmt.Println(fmt.Sprint(err) + ": " + layerstderr.String())
				return
			}
			fmt.Println("Result: " + out.String())

			// removing singular file as opposed to using os.RemoveAll
			err = os.Remove("python/get_layer_packages.sh")
			CheckError(err)

			// Get a Buffer to Write To
			outFile, err := os.Create(cb.Function + "_layers.zip")
			CheckError(err)
			defer outFile.Close()
			// Create a new zip archive.
			writer := zip.NewWriter(outFile)
			// Add some files to the archive.
			AddFiles(writer, layerFolder, "python/")

			// Make sure to check the error on Close.
			err = writer.Close()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("ZIP archive is ready. The name of the archive is " + cb.Function + "_layers.zip")
			}

			// check --now flag to publish layer
			if Now {
				// Publish layer leveraging shared config to pickup region configuration
				sess := session.Must(session.NewSessionWithOptions(session.Options{
					SharedConfigState: session.SharedConfigEnable,
				}))

				svc := lambda.New(sess)

				contents, err := ioutil.ReadFile(cb.Function + "_layers.zip")
				CheckError(err)

				input := &lambda.PublishLayerVersionInput{
					CompatibleRuntimes: []*string{
						aws.String("python3.8"),
					},
					Content: &lambda.LayerVersionContentInput{
						ZipFile: contents,
					},
					Description: &cb.Description,
					LayerName:   &cb.Layer,
					//				LicenseInfo: aws.String("MIT"),
				}
				// getting the result of the publish layer operation
				result, err := svc.PublishLayerVersion(input)
				if CheckAWSError(err) {
					os.Exit(1)
				}
				//fmt.Println(result)
				// we want to check if we are going to add the layer to our Lambda function
				fmt.Println("Layer built. Do you want to add it to your Lambda function, " + cb.Function + "? [yes/no]")
				userInput := bufio.NewScanner(os.Stdin)
				userInput.Scan()
				//				fmt.Printf("%v", input.Text())
				if userInput.Text() != "yes" && userInput.Text() != "no" {
					fmt.Println("Please type either 'yes' or 'no'.")
					os.Exit(1)
				}
				if userInput.Text() == "no" {
					fmt.Println("Understood. The new layer version is not added to the Lambda function.")
					os.Exit(1)
				} else {
					fmt.Println("Understood. Adding new layer...")
					// retrieving the LayerVersionArn to use when adding the layer to our Lambda function
					layerVersionARN := *result.LayerVersionArn
					layerVersionARNSlice := []string{layerVersionARN}
					fmt.Println("=========")
					fmt.Println(layerVersionARN)

					input := &lambda.UpdateFunctionConfigurationInput{
						FunctionName: &cb.Function,
						Layers:       aws.StringSlice(layerVersionARNSlice),
					}

					result, err := svc.UpdateFunctionConfiguration(input)
					if CheckAWSError(err) {
						os.Exit(1)
					}

					fmt.Println(result)
				}
			}
		}
	},
}

func init() {
	cookLayerCmd.PersistentFlags().BoolVar(&Now, "now", false, "Cook and deliver layer.")
}
