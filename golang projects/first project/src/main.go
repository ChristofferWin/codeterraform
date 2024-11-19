package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/fatih/color"
)

type Attribute struct {
	Type         string          `json:"Type"`
	Name         string          `json:"Name"`
	Parent       string          `json:"Parent"`
	Required     bool            `json:"Required"`
	Descriptor   string          `json:"Descriptor"`
	ShadowCopy   ShadowAttribute `json:"ShadowCopy"`
	TerraformURL string          `json:TerraformURL`
}

type ShadowAttribute struct {
	Type       string `json:"Type"`
	Parent     string `json:"Parent"`
	Required   bool   `json:"Required"`
	Descriptor string `json:"Descriptor"`
}

type ArmResourceType struct {
	Resource_type  string `json:"type"`
	CanBeSeperated bool
	Parent         string
}

type ArmObject struct {
	Name                  string   `json:"name"`
	Resource_id           string   `json:"id"`
	Resource_types        []string `json:"type"`
	Location              string   `json:"location"`
	Resource_group_name   string
	Properties            interface{} `json:"properties"` // Use interface{} for dynamic properties
	Special_resource_type string      `json:"kind"`
	Subscription_id       string
}

// Define the HtmlObject struct with a named attributes field
type HtmlObject struct {
	Resource_type string      `json:"Resource_type"`
	Version       string      `json:"Version"`
	Attribute     []Attribute `json:"Attribute"`
	Not_found     bool
	Last_updated  string `json:LastUpdated`
}

type RootAttribute struct {
	Name            string
	Value           interface{}
	BlockName       string
	IsBlock         bool
	UniqueBlockName string //Master key - To make sure all data can be linked directly
	Descriptor      string
}

type Variable struct {
	Name         string
	Description  string
	DefaultValue string
}

type BlockAttribute struct {
	BlockName       string
	RootAttribute   []RootAttribute
	Parent          string
	UniqueBlockName string
}

type CompileObject struct {
	ResourceDefinitionName string
	Variables              []Variable
	BlockAttributes        []BlockAttribute
	ArmObject              ArmObject
	FilePath               string
	SeperatedResource      bool
	ResourceType           string
	ResourceName           string
	IsRoot                 bool
	HtmlObject             HtmlObject
	AliasProviderName      string
	ProviderName           string
}

type TerraformObject struct {
	ProviderVersion string
	ProviderName    string
	CompileObjects  []CompileObject
}

type TerraformStringConfigObject struct { //This block can be expanded so that in the future its easy to add new type of formatting
	StringConfig string
	FileName     string
}

type StatusObject struct {
	CountFilesCreated              int     `json:"countFilesCreated"`
	CountFilesFailCreate           int     `json:"countFilesFailCreate"`
	CountNotFoundResourceType      int     `json:"countNotFoundResourceType"`
	CountAnalyzedARMTemplates      int     `json:CountAnalyzedARMTemplates`
	CountTerraformResources        int     `json:CountTerraformResources`
	DeploymentFolder               string  `json:"deploymentFolder"`
	TotalPercentageMatch           float64 `json:"totalPercentageMatch"`
	PercentageMatchPerResourceType struct {
		ResourceType    string  `json:"resourceType"`
		PercentageMatch float64 `json:"percentageMatch"`
	} `json:"percentageMatchPerResourceType"`
}

/*
	CODE IS OBSOLETE AND HAS OFFICIALY BEEN MOVED TO ITS OWN REPOSITORY AT: https://github.com/ChristofferWin/TerrafyARM/tree/main
*/

var MapOfResourceTypePerAttributeCount = make(map[string]int)
var ReturnStatusObject = StatusObject{}
var HtmlBaseURL = ""
var htmlObjects = []HtmlObject{}
var AttributeObjects = []Attribute{}
var SystemTerraformDocsFileName string = "./terrafyarm/htmlobjects.json"
var SystemTerraformCompiledObjectsFileName string = "./terrafyarm/terraform-arm-compiled-objects.json"
var currentVersion string = "0.1.0"
var verbose bool
var GlobalHtmlAttributesToMatchAgainst = []Attribute{}

func init() {
	flag.BoolVar(&verbose, "verbose", false, "(Optional & Recommended) Enable verbose output")
}

func main() {
	filePath := flag.String("file-path", "./", "(Optional & Recommended) Path to the ARM json file(s) Can be either a specific filepath or directory path\n\ne.g. 'C:\\ArmTemplates', 'C:\\ArmTemplates\\myarmfile.json' OR '.\\ArmTemplates', '.\\ArmTemplates\\myarmfile.json'\n\nNote: The path symbol might change, e.g. '\\' on Windows and '/' On Unix-based systems like MacOS & Linux\n")
	noCache := flag.Bool("no-cache", false, "(Optional) Switch to determine whether to force 'TerrafyArm' To not use / build a cache (default false)\n")
	clearCache := flag.Bool("clear-cache", false, "(Optional) Switch to determine whether to force 'TerrafyArm' to remove all cache (default false)\n\nUse this switch to reset 'TerrafyArm' everytime new resource types are added to the folder of ARM templates\n\nNote: 'TerrafyArm' will exit after completion (Even used with other arguments, the application exits right after cache removal)\n")
	providerVersion := flag.String("provider-version", "latest", "(Optional) Control the specific Terraform 'AzureRM' provider version to use e.g. '4.9.0'\n")
	seperateSubResources := flag.Bool("seperate-nested-resources", false, "(Optional) Switch to determine whether the decompiler shall seperate nested resources into their own Terraform resource definiton\n\nNote: This switch will not be enabled before a later version\n")
	rootDecompilefolderPath := flag.String("output-file-path", "", "(Required) Path for the new root decompile folder which will be created\n\nParse either a full path e.g. 'C:\\Decompile\\<some folder name>\\my-new-terraform-folder'\n\nOr use a relative path e.g. '.\\my-new-terraform-folder'\n\nNote: The path symbol might change, e.g. '\\' on Windows and '/' On Unix-based systems like MacOS & Linux\n")
	seperateDecompiledResources := flag.Bool("seperate-output-files", false, "(Optional) Switch to determine whether the resources being decompiled shall reside in isolated sub folder\n\nNote: This switch will not be enabled before a later version\n")
	listOfSubscriptionNamedProviders := flag.String("custom-terraform-provider-names", "", "(Optional) Define a list to control the custom alias provider names for all ARM resources in Terraform\n\nThe list must be in format '<a valid Azure Subscription ID>=<alias name>'\n\ne.g. '00000000-0000-0000-0000-000000000000=mycustomnaliasname,11111111-1111-1111-1111-111111111111=mycustomname2'\n\nCustom provider names must not start with a number and only contain symbols '_' & '-'\n\nAny number of terraform-providers can be used. To read more about custom providers, see https://developer.hashicorp.com/terraform/language/providers/configuration\n")
	filePathListOfSubscriptionProviders := flag.String("file-path-custom-providers", "", "(Optional NOT IN USE) Define either a full path or a relative path to the file of custom Azure subscription providers\n\nThe file format must be 'CSV' and should consist of a column named 'subscription_id' & 1 more named 'alias_name'\n\nTo see an example of such file, see: https://github.com/ChristofferWin/TerrafyARM/docs/custom_providers/example.csv\n")
	chromeCustomPath := flag.String("chrome-custom-exe-path", "", "(Optional) Define either the relative or full path to the Google Chrome executeable\n\nNote: This parameter MUST be used if Chrome is not installed on any default path, which depends on the OS\n\nFor more information visit https://github.com/ChristofferWin/TerrafyARM/docs/")
	version := flag.Bool("version", false, "(Optional) Switch to check the current installed local version of 'TerrafyArm'\n")
	flag.Parse()

	logVerbose("Please note that this version is an alpha. Some flags might be disabled and 'TerrafyArm' CAN give invalid results")
	logVerbose("If you see invalid results, it can actually work to simply run 'TerrafyArm' Again on the same ARM files")

	if CheckForTerraformExecuteable() {
		logWarning(strings.TrimSpace(`Terraform was not found on the system
It's ALWAYS recommended to have 'Terraform' installed alongside 'TerrafyArm'
Install Terraform (Windows) visit: https://www.codeterraform.com/post/getting-started-with-using-terraform
Install Terraform (Linux / MacOS) visit: https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli`))
	}

	if *version {
		fmt.Sprintf("The current version of the 'TerrafyArm' Decompiler is '%s'\nFor information about versions, please check the official Github release page at:\nhttps://github.com/ChristofferWin/TerrafyARM/releases", currentVersion)
		return
	}

	if *clearCache {
		err := os.RemoveAll(strings.Split(SystemTerraformDocsFileName, "/")[1])
		if err != nil {
			logFatal(fmt.Sprintf("The following error occured while trying to delete the cache: %s", err))
		}
		return
	}

	if *rootDecompilefolderPath == "" {
		logFatal("a path for flag '-output-file-path' must be provided...\nPlease use command 'terrafyarm -help' For details...")
		return
	}

	if *seperateDecompiledResources {
		logFatal(fmt.Sprintf("The following flag 'seperate-nested-resources' is NOT in use as of version: %s", currentVersion))
		return
	}

	if *seperateSubResources {
		logFatal(fmt.Sprintf("The following flag 'seperate-output-files' is NOT in use as of version: %s", currentVersion))
	}

	if typeValue := reflect.ValueOf(*listOfSubscriptionNamedProviders); typeValue.Kind() != reflect.String {
		logFatal(fmt.Sprintf("The value provided of '%s' for flag '-custom-terraform-provider-names' is not a string...\nUse 'TerrafyArm -help' for more details", listOfSubscriptionNamedProviders))
	}

	if *listOfSubscriptionNamedProviders != "" && *filePathListOfSubscriptionProviders != "" {
		logFatal("Please do not provide values for both flags 'custom-terraform-provider-names' & 'file-path-custom-providers' at the same time\nUse 'TerrafyArm -help' for more details")
	}

	if *filePathListOfSubscriptionProviders != "" {
		logFatal(fmt.Sprintf("Please do not provide a value for flag 'file-path-custom-providers' as its not in use as of 'TerrafyArm' version '%s'", currentVersion))
	}

	if !CheckChromeInstalled(*chromeCustomPath) {
		if *chromeCustomPath != "" {
			logFatal(fmt.Sprintf("Chrome was not found on the given location '%s'\nPlease either provide a custom path to the executeable or make sure Google chrome is installed\nTo install Google chrome visit https://support.google.com/chrome/answer/95346?hl=en&co=GENIE.Platform%3DDesktop", *chromeCustomPath))
		} else {
			logFatal("Chrome was not found on any of the default installation locations on the system\nPlease either provide a custom path to the executeable or make sure Google chrome is installed\nTo install Google chrome visit https://support.google.com/chrome/answer/95346?hl=en&co=GENIE.Platform%3DDesktop")
		}
	}

	fileContent, err := ImportArmFile(filePath)

	if err != nil {
		// Handle the error if it occurs
		logFatal(fmt.Sprintf("Error reading file: %s", err))
		return
	}

	verifiedFiles := VerifyArmFile(fileContent)
	if len(verifiedFiles) == 0 {
		logFatal(fmt.Sprintf("No valid ARM templates found on path: %s", *filePath))
		return
	}

	logOK(fmt.Sprintf("Successfully retrieved %d ARM templates from location %s", len(verifiedFiles), *filePath))

	if *seperateSubResources {
		logFatal("This flag is not implemented in the current version of TerrafyArm")
	}

	baseArmResources := GetArmBaseInformation(verifiedFiles)

	if err != nil {
		logFatal(fmt.Sprintf("Error while trying to retrieve the json ARM content: %s", err))
		return
	}

	var predetermineTypes []string
	for _, armResource := range baseArmResources {
		if armResource.Special_resource_type != "" {
			for x := 0; x <= len(armResource.Resource_types)-1; x++ {
				finalResourceType := ""
				for _, innerType := range armResource.Resource_types {
					if len(strings.Split(innerType, "/")) == 2 {
						finalResourceType = innerType
						break
					}
				}
				predetermineTypes = append(predetermineTypes, ConvertArmToTerraformProvider(finalResourceType, armResource.Special_resource_type))
			}
		}
	}

	var resourceTypesFromArm []string
	var resourceTypesFromDif []string
	var resourceTypesToRetrieve []string

	for _, resourceType := range baseArmResources {
		if resourceType.Special_resource_type == "" {
			resourceTypesFromArm = append(resourceTypesFromArm, resourceType.Resource_types...)
		}
	}

	var htmlObjectsFromCache []HtmlObject

	if !*noCache {
		htmlObjectsFromCache, err = GetCachedSystemFiles()
		if err != nil {
			logWarning("No cache detected, please expect 'TerrafyArm' to consume 5 seconds for each terraform type")
		}
		for _, htmlObjectFromCache := range htmlObjectsFromCache {
			if !htmlObjectFromCache.Not_found {
				resourceTypesFromDif = append(resourceTypesFromDif, htmlObjectFromCache.Resource_type)
			}
		}
	}

	if htmlObjectsFromCache == nil {
		resourceTypesToRetrieve = resourceTypesFromArm
	} else {
		resourceTypesToRetrieve = resourceTypesFromDif
	}

	resourceTypesToRetrieve = append(resourceTypesToRetrieve, UniquifyResourceTypes(predetermineTypes)...)

	mapOfTypes := make(map[string]bool)
	sortedResourceTypes := []string{}

	for _, resourceType := range resourceTypesToRetrieve {
		if !mapOfTypes[resourceType] {
			mapOfTypes[resourceType] = true
			sortedResourceTypes = append(sortedResourceTypes, resourceType)
		}
	}

	cleanHtml := HtmlObject{} //THIS IS USED!!
	logTime := ""
	if htmlObjectsFromCache == nil {
		for index, resourceType := range sortedResourceTypes {
			if index == 0 {
				logVerbose(fmt.Sprintf("Pulling terraform information for %d types\nEstimated time to download %s minutes/seconds...", len(sortedResourceTypes), fmt.Sprintf("%02d:%02d", len(sortedResourceTypes)*3/60, len(sortedResourceTypes)*3%60)))
				logTime = timeFormat(time.Now())
			}
			if resourceType != "" {
				rawHtml, err := GetRawHtml(resourceType, *providerVersion, *chromeCustomPath)

				if err != nil {
					logFatal(fmt.Sprintf("Error while trying to retrieve required documentation: %s\n%s", resourceType, err))
					break
				}
				if rawHtml != "not_found" && rawHtml != "" {
					logOK(fmt.Sprintf("Successfully retrieved terraform information about %s", resourceType))
					cleanHtml = SortRawHtml(rawHtml, resourceType, logTime)
					htmlObjects = append(htmlObjects, cleanHtml)
				} else {
					htmlObject := HtmlObject{
						Resource_type: resourceType,
						Not_found:     true,
					}
					htmlObjects = append(htmlObjects, htmlObject)
				}
			}
		}
		if !*noCache {
			err = NewCachedSystemFiles(htmlObjects, TerraformObject{})
			if err != nil {
				logFatal(fmt.Sprintf("An error occured while running function 'NewCachedSystemFiles'\n%s", err))
			}
		}
	} else {
		htmlObjects = append(htmlObjects, htmlObjectsFromCache...)
		logOK(fmt.Sprintf("Successfully retrieved cache. Cache last updated: %s", htmlObjectsFromCache[0].Last_updated))
	}

	var cleanhtmlObjects []HtmlObject

	for _, htmlObject := range htmlObjects {
		if !htmlObject.Not_found {
			cleanhtmlObjects = append(cleanhtmlObjects, htmlObject)
		}
	}

	compiledObjects := CompileTerraformObjects(baseArmResources, cleanhtmlObjects, *seperateSubResources)

	terraformObject := NewTerraformObject(compiledObjects, *providerVersion)

	err = NewCachedSystemFiles([]HtmlObject{}, terraformObject)
	if err != nil {
		logFatal(fmt.Sprintf("An error occured while running function 'NewCachedSystemFiles' %s", err))
	}

	terraformFileNames := NewCompiledFolderStructure(*seperateDecompiledResources, *rootDecompilefolderPath, terraformObject)

	providerFullPathName := ""
	for _, terraformFileName := range terraformFileNames {
		if strings.Contains(terraformFileName, "providers.tf") {
			providerFullPathName = terraformFileName
			break
		}
	}

	terraformObject = InitializeTerraformFile(providerFullPathName, *providerVersion, terraformObject.ProviderName, terraformObject, *listOfSubscriptionNamedProviders)

	terraformStringConfigTotal := []TerraformStringConfigObject{}

	for _, terraformCompiledObject := range terraformObject.CompileObjects {
		terraformConfig := NewTerraformConfig(terraformCompiledObject, *seperateSubResources)
		if len(terraformConfig) > 0 {
			terraformStringConfigObject := TerraformStringConfigObject{
				StringConfig: NewTerraformConfig(terraformCompiledObject, *seperateSubResources),
				FileName:     fmt.Sprintf("%s/%s", *rootDecompilefolderPath, terraformCompiledObject.FilePath),
			}
			terraformStringConfigTotal = append(terraformStringConfigTotal, terraformStringConfigObject)
		} else {
			logVerbose(fmt.Sprintf("The resource %s of type %s has been skipped due to no matches between ARM and terraform being found...", terraformCompiledObject.ResourceName, terraformCompiledObject.ResourceType))
		}
	}

	for _, terraformConfig := range terraformStringConfigTotal {
		if !strings.HasSuffix(terraformConfig.FileName, "/") {
			WriteTerraformConfigToDisk(terraformConfig.StringConfig, terraformConfig.FileName)
		}
	}

	RunTerraformCommand("fmt", *rootDecompilefolderPath)

	if ReturnStatusObject.CountFilesFailCreate == 0 {
		logOK(fmt.Sprintf(`Successfully decompiled the following ARM to Terraform:
ARM source file(s) location: %s
ARM files analyzed: %d
Terraform file(s) location: %s
Terraform files created: %d
Terraform resources defined: %d
		`, *filePath, len(verifiedFiles), *rootDecompilefolderPath, ReturnStatusObject.CountFilesCreated, ReturnStatusObject.CountTerraformResources))
	}
}

func NewCompiledFolderStructure(seperatedFiles bool, rootFolderPath string, terraformObject TerraformObject) []string {
	var fileNames []string
	var terraformFilePaths []string
	//var folderNames []string Not in use yet

	err := os.Mkdir(rootFolderPath, 0755)
	if err != nil {
		if !os.IsExist(err) {
			logFatal(fmt.Sprintf("an error occured while trying to create the root directory for the decompiled files...\n%s", err))
			return []string{}
		}
	}

	mapOfUniqueResourceTypes := make(map[string]bool)
	for index, compiledObject := range terraformObject.CompileObjects {
		resourceTypeToFileName := compiledObject.FilePath
		if seperatedFiles {
			fileNames = append(fileNames, fmt.Sprintf("%s_%s", compiledObject.ResourceName, resourceTypeToFileName))
		} else {
			if !mapOfUniqueResourceTypes[compiledObject.FilePath] {
				fileNames = append(fileNames, resourceTypeToFileName)
				mapOfUniqueResourceTypes[resourceTypeToFileName] = true
			}
		}
		if index == 0 { //Create provider file
			fileNames = append(fileNames, "providers.tf")
		}
	}

	for _, fileName := range fileNames {
		fullPath := ""
		if !strings.Contains(rootFolderPath, ".") {
			fullPath = filepath.Join(rootFolderPath, fileName) // Get the full path
		} else {
			fullPath = fmt.Sprintf("%s/%s", rootFolderPath, fileName)
		}

		if fileName != "" && fileName != ".tf" {
			logVerbose(fmt.Sprintf("Creating the following file '%s' on location '%s'", fileName, rootFolderPath))
			err := os.WriteFile(fullPath, []byte{}, 0644)
			if err != nil {
				logVerbose(fmt.Sprintf("Error occured while trying to create file '%s'\n%s\nContinuing...", fileName, err))
				ReturnStatusObject.CountFilesFailCreate++
				return []string{}
			}
			ReturnStatusObject.CountFilesCreated++
			terraformFilePaths = append(terraformFilePaths, fullPath)
		}
	}

	return terraformFilePaths
}

func InitializeTerraformFile(terraformFilePath string, providerVersion string, providerName string, terraformCompiledObject TerraformObject, customProviders string) TerraformObject {
	masterSubscription := ""
	mapOfSubCount := make(map[string]int)
	mapOfTerraformProviders := make(map[string]string)
	mapOfUniqueSubscription := make(map[string]bool)
	listOfCustomProviders := strings.Split(customProviders, ",")
	listOfCustomProvidersCleaned := []string{}
	patternAzureSubscriptionGUID := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	patternTerraformAliasName := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	for _, provider := range listOfCustomProviders {
		providerSplit := strings.Split(provider, "=")
		subID := ""
		aliasName := ""
		if len(providerSplit) == 2 {
			subID = providerSplit[0]
			aliasName = providerSplit[1]
			if patternAzureSubscriptionGUID.MatchString(subID) && patternTerraformAliasName.MatchString(aliasName) && !mapOfUniqueSubscription[subID] {
				listOfCustomProvidersCleaned = append(listOfCustomProvidersCleaned, provider)
				mapOfUniqueSubscription[subID] = true
			} else if mapOfUniqueSubscription[subID] {
				logVerbose(fmt.Sprintf("The subscription_id of '%s' is already in use...(Reusing existing provider)\nUse 'TerrafyArm -help' for details", provider))
			} else {
				logVerbose(fmt.Sprintf("The provider of '%s' is not in a valid format...\nUse 'TerrafyArm -help' for details", provider))
			}
		} else {
			logVerbose(fmt.Sprintf("The provider of '%s' is not in a valid format...\nUse 'TerrafyArm -help' for details", provider))
		}
	}

	for _, compiledObject := range terraformCompiledObject.CompileObjects {
		mapOfSubCount[compiledObject.ArmObject.Subscription_id]++
	}

	checkSubnetCount := 0 //Lets keep track of whether all sub counts are the same, if they are, we need to do some logic after this
	biggestCount := 0
	for subID, count := range mapOfSubCount {
		if count > biggestCount && !mapOfUniqueSubscription[subID] {
			checkSubnetCount++
			biggestCount = count
			masterSubscription = subID
		}
	}
	for _, provider := range listOfCustomProvidersCleaned {
		providerSplit := strings.Split(provider, "=")
		subID := providerSplit[0]
		aliasName := providerSplit[1]
		for index, compiledObject := range terraformCompiledObject.CompileObjects {
			if compiledObject.ArmObject.Subscription_id == subID {
				terraformCompiledObject.CompileObjects[index].AliasProviderName = aliasName
			}
		}
	}

	for index, compiledObject := range terraformCompiledObject.CompileObjects {
		if len(compiledObject.AliasProviderName) == 0 && compiledObject.ArmObject.Subscription_id != masterSubscription {
			terraformCompiledObject.CompileObjects[index].AliasProviderName = fmt.Sprintf("auto_provider_%s", strings.Split(compiledObject.ArmObject.Subscription_id, "-")[0])
			mapOfTerraformProviders[terraformCompiledObject.CompileObjects[index].ArmObject.Subscription_id] = terraformCompiledObject.CompileObjects[index].AliasProviderName
		} else if compiledObject.ArmObject.Subscription_id != masterSubscription {
			mapOfTerraformProviders[compiledObject.ArmObject.Subscription_id] = compiledObject.AliasProviderName
		}
	}

	initialCommentBlock := strings.TrimSpace(fmt.Sprintf(`/*
	This Terraform template is created using 'TerrafyArm' and version %s

	Any template CAN have issues - Please report these over on Github at https://github.com/ChristofferWin/TerrafyARM/issues

	As 'TerrafyArm' Progresses in development, please consolidate the releases page at https://github.com/ChristofferWin/TerrafyARM/releases 
	`, currentVersion))

	requiredProvidersBlock := `
	Required boilerplate - The 'version' will only be set when the argument '-provider-version' <some version> has been parsed

	If in any doubt, use command 'TerrafyArm -help' for more information
*/
	`

	requiredContextBlockMaster := strings.TrimSpace(fmt.Sprintf(`/*
	Provider configurations can be controlled by the user. Use the command TerrafyArm -custom-terraform-provider-names <providers>

	For all providers without custom names, TerrafyArm automatically generates names as follows: <first segment of subscription ID>_<resource name>

	The 'default' provider will always be determined by counting the number of resources using the same subscription, if that subscription does not have a custom provider set

	For examples of how to use custom providers, see the help function or visit https://github.com/ChristofferWin/TerrafyARM/docs/
	
*/

	provider "azurerm" {
	  features{}
	  subscription_id = "%s" //Subscription seen the most times in all of the ARM templates provided (count = %d)
	}
	`, masterSubscription, biggestCount))

	//We need to remove all the alias providers using dupplicate subscriptions

	requiredContextBlockAliasSlice := []string{}
	for subID, resourceName := range mapOfTerraformProviders {
		if subID != masterSubscription {
			requiredContextBlockAliasSlice = append(requiredContextBlockAliasSlice, fmt.Sprintf(`
			provider "azurerm" {
			  features{}
			  alias = "%s"
			  subscription_id = "%s"
			}
		`, resourceName, subID))
		}
	}

	requiredContextBlockMaster = fmt.Sprintf("%s\n%s", requiredContextBlockMaster, strings.Join(requiredContextBlockAliasSlice, "\n"))

	initialize_terraform := ""
	if providerVersion == "latest" {
		initialize_terraform = fmt.Sprintf("terraform {\n  required_providers {\n      %s = {\n         source = \"hashicorp/%s\"\n         }\n      }\n}", providerName, providerName)
	} else {
		initialize_terraform = fmt.Sprintf("terraform {\n  required_providers {\n      %s = {\n         source = \"hashicorp/%s\"\n         version = \"%s\"\n         }\n      }\n}", providerName, providerName, providerVersion)
	}

	finalInitializeTerraform := fmt.Sprintf("%s\n%s\n%s\n\n%s", initialCommentBlock, requiredProvidersBlock, initialize_terraform, requiredContextBlockMaster)
	ChangeExistingFile(terraformFilePath, finalInitializeTerraform)

	return terraformCompiledObject
}

func NewTerraformConfig(terraformCompiledObject CompileObject, seperatedResources bool) string {
	rootTerraformConfig := []string{}
	returnRootTerraformConfig := []string{}
	sortedBlockAttributes := []BlockAttribute{}

	rootTerraformDefinition := NewTerraformResourceDefinitionName(terraformCompiledObject.ProviderName, terraformCompiledObject.ResourceType, terraformCompiledObject.ResourceDefinitionName)
	rootTerraformConfig = append(rootTerraformConfig, rootTerraformDefinition)

	if len(terraformCompiledObject.BlockAttributes) > 1 {
		sortedBlockAttributes = SortBlockAttributesForTerraform(terraformCompiledObject.BlockAttributes)
	} else {
		sortedBlockAttributes = terraformCompiledObject.BlockAttributes
	}
	/*
		fmt.Println("------------RESOURCE NAME ----------------")

		if terraformCompiledObject.ProviderName == "ruc-we-velkommen-front-ws01" {
			for _, block := range sortedBlockAttributes {
				fmt.Println("BLOCK:", block.BlockName, block.Parent)
				for _, rootAttribute := range block.RootAttribute {
					fmt.Println("name:", rootAttribute.Name, rootAttribute.BlockName, rootAttribute.IsBlock)
				}
			}
		}
	*/
	parentBlockName := ""
	captureLastNestedBlocks := 0
	countStartNestedBlock := 0
	numberOfSameLevelNestedBlocks := 0
	captureMaxSameLevelBlocks := 0
	placeHolderBlockAttributes := []BlockAttribute{}
	for index, blockAttribute := range sortedBlockAttributes {
		placeholderAttributes := AddPlaceHolderTerraformAttributes(blockAttribute, terraformCompiledObject.HtmlObject, sortedBlockAttributes)
		placeHolderBlockAttributes = AddPlaceHolderTerraformBlocks(blockAttribute.RootAttribute, placeholderAttributes, terraformCompiledObject.HtmlObject)
		if terraformCompiledObject.ProviderName == "AzureAdminlogininprodAzureAD-pleaseinvestigate" {

		}
		if index == 0 {

			//Determine whether to auto retrieve name, resource_group_name and location
			autoCreateName := true
			autoCreateResourceGroupName := true
			autoCreateLocation := true

			for _, terraformRootAttribute := range blockAttribute.RootAttribute {
				if terraformRootAttribute.Name == "name" {
					autoCreateName = false
				} else if terraformRootAttribute.Name == "resource_group_name" {
					autoCreateResourceGroupName = false
				} else if terraformRootAttribute.Name == "location" {
					autoCreateLocation = false
				}
			}

			for _, htmlAttribute := range terraformCompiledObject.HtmlObject.Attribute {
				if htmlAttribute.Name == "name" && htmlAttribute.Parent == "root" && autoCreateName {
					rootAttribute := RootAttribute{
						Name:  "name",
						Value: terraformCompiledObject.ResourceName,
					}
					rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("", rootAttribute, false, true))
				}

				if htmlAttribute.Name == "resource_group_name" && htmlAttribute.Parent == "root" && autoCreateResourceGroupName {
					rootAttribute := RootAttribute{
						Name:  "resource_group_name",
						Value: terraformCompiledObject.ArmObject.Resource_group_name,
					}
					rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("", rootAttribute, false, true))
				}

				if htmlAttribute.Name == "location" && htmlAttribute.Parent == "root" && autoCreateLocation {
					rootAttribute := RootAttribute{
						Name:  "location",
						Value: terraformCompiledObject.ArmObject.Location,
					}
					rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("", rootAttribute, false, true))
				}
			}

			if len(placeholderAttributes) > 0 {
				sortedPlaceHolderAttributes := SortRootAttributesForTerraform(placeholderAttributes)
				for _, placeHolderAttribute := range sortedPlaceHolderAttributes {
					if !placeHolderAttribute.IsBlock {
						rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("", placeHolderAttribute, false, true))
					}
				}
			}
		}
		if blockAttribute.BlockName == "root" {
			parentBlockName = ""
			for _, terraformAttribute := range blockAttribute.RootAttribute {
				if !terraformAttribute.IsBlock {
					rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("", terraformAttribute, false, true))
				}
			}
		} else if blockAttribute.UniqueBlockName == "" && blockAttribute.BlockName != "root" {
			mapOfIndentationCount := make(map[string]int)
			numberOfNestedBlocks := 0

			for _, minimumBlockAttribute := range sortedBlockAttributes {
				if minimumBlockAttribute.Parent == blockAttribute.BlockName && blockAttribute.BlockName != "root" {
					mapOfIndentationCount[blockAttribute.BlockName]++
				}
			}
			if mapOfIndentationCount[blockAttribute.BlockName] == 0 {
				numberOfNestedBlocks = 1
			} else {
				numberOfNestedBlocks = mapOfIndentationCount[blockAttribute.BlockName]
				numberOfSameLevelNestedBlocks = mapOfIndentationCount[blockAttribute.BlockName]
			}

			if numberOfSameLevelNestedBlocks > 1 && captureMaxSameLevelBlocks == 0 {
				captureMaxSameLevelBlocks = numberOfSameLevelNestedBlocks
			}

			for _, minimumBlockAttribute := range sortedBlockAttributes {
				if blockAttribute.Parent == minimumBlockAttribute.BlockName && minimumBlockAttribute.BlockName != "root" {
					parentBlockName = blockAttribute.Parent
				}
			}

			placeholderAttributes := AddPlaceHolderTerraformAttributes(blockAttribute, terraformCompiledObject.HtmlObject, []BlockAttribute{})
			mergedRootAttributes := []RootAttribute{}
			mergedRootAttributes = append(append(mergedRootAttributes, placeholderAttributes...), blockAttribute.RootAttribute...)

			sortedRootAttributes := SortRootAttributesForTerraform(mergedRootAttributes)

			terraformObjectString, nestedBlockCount := AddObjectTerraformAttributeForResourceDefinition(fmt.Sprintf("\n%s {%s", blockAttribute.BlockName, strings.Repeat(" ", numberOfNestedBlocks)), "", sortedRootAttributes, sortedBlockAttributes)
			rootTerraformConfig = append(rootTerraformConfig, terraformObjectString)
			if parentBlockName != "" || nestedBlockCount > 1 && numberOfSameLevelNestedBlocks > 0 {
				if parentBlockName != blockAttribute.Parent {
					captureLastNestedBlocks = nestedBlockCount
				} else if parentBlockName == blockAttribute.Parent {
					if numberOfSameLevelNestedBlocks == 1 {
						rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition(strings.Repeat("\n}", captureLastNestedBlocks), RootAttribute{}, true, false))
					} else {
						rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("\n}", RootAttribute{}, true, false))
					}
					parentBlockName = ""
					captureLastNestedBlocks = 0
					numberOfSameLevelNestedBlocks--
				}
			} else if numberOfSameLevelNestedBlocks == 0 {
				if captureMaxSameLevelBlocks > 0 {
					rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition(strings.Repeat("\n}", captureMaxSameLevelBlocks), RootAttribute{}, true, false))
				} else {
					rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("\n}", RootAttribute{}, true, false))

				}
			}

		} else if blockAttribute.UniqueBlockName != "" && blockAttribute.BlockName != "root" {
			if countStartNestedBlock == 0 {
				countStartNestedBlock = CountBlockNestingForTerraform(sortedBlockAttributes, blockAttribute.RootAttribute)
				if blockAttribute.Parent == "root" && countStartNestedBlock > 1 {
					countStartNestedBlock = countStartNestedBlock + 1
				}
			}

			match := false
			for _, rootAttribute := range blockAttribute.RootAttribute {
				if rootAttribute.Name == "name" {
					match = true
					break
				}
			}
			mergedRootAttributes := []RootAttribute{}
			if !match {
				rootAttribute := RootAttribute{
					Name:            "name",
					Value:           blockAttribute.UniqueBlockName,
					BlockName:       blockAttribute.BlockName,
					UniqueBlockName: blockAttribute.UniqueBlockName,
				}
				mergedRootAttributes = append(mergedRootAttributes, rootAttribute)
			}
			placeholderAttributes := AddPlaceHolderTerraformAttributes(blockAttribute, terraformCompiledObject.HtmlObject, []BlockAttribute{})
			mergedRootAttributes = append(append(mergedRootAttributes, placeholderAttributes...), blockAttribute.RootAttribute...)
			sortedRootAttributes := SortRootAttributesForTerraform(mergedRootAttributes)
			terraformObjectString, nestedBlockCount := AddObjectTerraformAttributeForResourceDefinition(fmt.Sprintf("\n%s {\n", blockAttribute.BlockName), "", sortedRootAttributes, sortedBlockAttributes)
			rootTerraformConfig = append(rootTerraformConfig, terraformObjectString)

			if nestedBlockCount == 1 {
				rootTerraformConfig = append(rootTerraformConfig, strings.Repeat("\n}", countStartNestedBlock))
				countStartNestedBlock = 0
			}
		}
	}

	if len(placeHolderBlockAttributes) > 0 {
		for _, placeHolderBlockAttribute := range placeHolderBlockAttributes {
			sortedPlaceHolderAttributes := SortRootAttributesForTerraform(placeHolderBlockAttribute.RootAttribute)
			terraformString, _ := AddObjectTerraformAttributeForResourceDefinition(fmt.Sprintf("\n%s {", placeHolderBlockAttribute.BlockName), " ", sortedPlaceHolderAttributes, []BlockAttribute{})
			terraformString = terraformString + "\n}"
			rootTerraformConfig = append(rootTerraformConfig, terraformString)
		}
	}

	if terraformCompiledObject.AliasProviderName != "" {

		rootTerraformConfig = append(rootTerraformConfig, fmt.Sprintf(`
	provider = azurerm.%s	
	}
	`, terraformCompiledObject.AliasProviderName))
	} else {
		rootTerraformConfig = append(rootTerraformConfig, "\n}")
	}

	convertTerraformSlice := strings.Split(strings.Join(rootTerraformConfig, "\n"), "\n") //Before this point, the string slice array is built out of "pockets" Of strings - Now we convert them line to line
	for index, line := range convertTerraformSlice {
		if line == "" && index != 0 {
			if strings.Contains(convertTerraformSlice[index+1], "{") || strings.Contains(convertTerraformSlice[index+1], "provider") {
				returnRootTerraformConfig = append(returnRootTerraformConfig, line)
			}
		} else {
			if strings.Contains(line, "PLACE-HOLDER-VALUE\\") {
				retrieveTerraformAttributeURL := strings.Split(line, `\`)
				convertedTerraformLine := retrieveTerraformAttributeURL[0] + fmt.Sprintf("\" # (Link is anchored) %s", retrieveTerraformAttributeURL[1])
				returnRootTerraformConfig = append(returnRootTerraformConfig, convertedTerraformLine)
			} else {
				returnRootTerraformConfig = append(returnRootTerraformConfig, line)
			}
		}
	}

	if len(returnRootTerraformConfig) > 5 {
		ReturnStatusObject.CountTerraformResources++
		return strings.Join(returnRootTerraformConfig, "\n")
	}
	return ""
}

func AddPlaceHolderTerraformBlocks(terraformAttributes []RootAttribute, placeholderAttributes []RootAttribute, htmlObject HtmlObject) []BlockAttribute {
	mapOfRootAttributesForBlocks := make(map[string][]RootAttribute)
	returnPlaceHolderBlocks := []BlockAttribute{}
	terraformAttributesNotFound := []RootAttribute{}

	for _, placeholderAttribute := range placeholderAttributes {
		match := false
		for _, terraformAttribute := range terraformAttributes {
			if placeholderAttribute.Name == terraformAttribute.Name {
				match = true
				break
			}
		}
		if !match {
			terraformAttributesNotFound = append(terraformAttributesNotFound, placeholderAttribute)
		}

	}

	for _, terraformAttribute := range terraformAttributesNotFound {
		for _, attribute := range htmlObject.Attribute {
			if terraformAttribute.Name == attribute.Parent && attribute.Type != "armObject" && attribute.Required {
				rootAttribute := RootAttribute{
					Name:      attribute.Name,
					Value:     fmt.Sprintf("PLACE-HOLDER-VALUE\\%s", attribute.TerraformURL),
					BlockName: attribute.Parent,
					IsBlock:   false,
				}
				mapOfRootAttributesForBlocks[terraformAttribute.Name] = append(mapOfRootAttributesForBlocks[terraformAttribute.Name], rootAttribute)
			}
		}
	}

	for blockName, requiredRootAttributes := range mapOfRootAttributesForBlocks {
		blockAttribute := BlockAttribute{
			BlockName:     blockName,
			RootAttribute: requiredRootAttributes,
			Parent:        "root",
		}
		returnPlaceHolderBlocks = append(returnPlaceHolderBlocks, blockAttribute)
	}
	return returnPlaceHolderBlocks
}

func AddPlaceHolderTerraformAttributes(blockAttribute BlockAttribute, htmlObject HtmlObject, allBlocks []BlockAttribute) []RootAttribute {
	allRequiredAttributes := []Attribute{}
	mapOfMissingAttributes := make(map[string]bool)
	returnRootAttributes := []RootAttribute{}

	for _, attribute := range htmlObject.Attribute {
		if attribute.Required && attribute.Parent == blockAttribute.BlockName && attribute.Name != "name" && attribute.Name != "location" && attribute.Name != "resource_group_name" {
			allRequiredAttributes = append(allRequiredAttributes, attribute)
		}
	}

	for _, requiredAttribute := range allRequiredAttributes {
		for _, terraformAttribute := range blockAttribute.RootAttribute {
			if terraformAttribute.Name == requiredAttribute.Name {
				mapOfMissingAttributes[requiredAttribute.Name] = true
			}

			if !mapOfMissingAttributes[requiredAttribute.Name] {
				mapOfMissingAttributes[requiredAttribute.Name] = false
			}
		}
	}

	for name, isFound := range mapOfMissingAttributes {
		if !isFound {
			for _, requiredAttribute := range allRequiredAttributes {
				if name == requiredAttribute.Name && requiredAttribute.Type != "armObject" {
					rootAttribute := RootAttribute{
						Name:            requiredAttribute.Name,
						Value:           fmt.Sprintf("PLACE-HOLDER-VALUE\\%s", requiredAttribute.TerraformURL),
						BlockName:       requiredAttribute.Parent,
						IsBlock:         false,
						UniqueBlockName: blockAttribute.UniqueBlockName,
					}
					returnRootAttributes = append(returnRootAttributes, rootAttribute)
				}
			}
		}
	}

	mapOfRequiredAttributes := make(map[string]int)
	for _, block := range allBlocks {
		for _, requiredAttribute := range allRequiredAttributes {
			if requiredAttribute.Type == "armObject" {
				mapOfRequiredAttributes[requiredAttribute.Name]++
				if block.BlockName == requiredAttribute.Name && block.UniqueBlockName == "" {
					mapOfRequiredAttributes[requiredAttribute.Name]++
					break
				}
			}
		}
	}

	for blockName, count := range mapOfRequiredAttributes {
		if count == 1 {
			attribute := Attribute{}
			for _, requiredAttribute := range allRequiredAttributes {
				if requiredAttribute.Name == blockName && requiredAttribute.Type == "armObject" {
					attribute = requiredAttribute
				}
			}
			rootAttribute := RootAttribute{
				Name:      attribute.Name,
				Value:     nil,
				BlockName: "root",
				IsBlock:   true,
			}
			returnRootAttributes = append(returnRootAttributes, rootAttribute)
		}
	}

	return returnRootAttributes
}

func SortBlockAttributesForTerraform(blockAttributesToSort []BlockAttribute) []BlockAttribute {
	blocksForReturn := []BlockAttribute{}
	//runWithoutUniqueBlocks := true
	for _, blockAttribute := range blockAttributesToSort {
		if blockAttribute.BlockName == "root" && blockAttribute.Parent == "root" {
			blocksForReturn = append(blocksForReturn, blockAttribute)
			break
		}
	}

	//For all terraform objects, where no unique names are present
	for _, blockAttribute := range blockAttributesToSort {
		if blockAttribute.BlockName != "root" && blockAttribute.Parent == "root" && blockAttribute.UniqueBlockName == "" {
			captureBlockName := ""
			blocksForReturn = append(blocksForReturn, blockAttribute)
			for _, innerBlockAttribute := range blockAttributesToSort {
				if innerBlockAttribute.Parent == blockAttribute.BlockName {
					blocksForReturn = append(blocksForReturn, innerBlockAttribute)
					if innerBlockAttribute.BlockName != "root" {
						captureBlockName = innerBlockAttribute.BlockName
					}
				}
				if captureBlockName == innerBlockAttribute.Parent && innerBlockAttribute.BlockName != "root" {
					blocksForReturn = append(blocksForReturn, innerBlockAttribute)
				}
			}
		}
	}

	//For all terraform objects, where unique names ARE present
	for _, blockAttribute := range blockAttributesToSort {
		if blockAttribute.BlockName != "root" && blockAttribute.Parent == "root" && blockAttribute.UniqueBlockName != "" {
			//runWithoutUniqueBlocks = false
			blocksForReturn = append(blocksForReturn, blockAttribute)
			for _, innerBlockAttribute := range blockAttributesToSort {
				if innerBlockAttribute.UniqueBlockName != "" &&
					innerBlockAttribute.BlockName != "root" &&
					innerBlockAttribute.BlockName != blockAttribute.BlockName &&
					innerBlockAttribute.UniqueBlockName == blockAttribute.UniqueBlockName {
					blocksForReturn = append(blocksForReturn, innerBlockAttribute)
				}
			}
		}
	}

	for index, blockAttribute := range blocksForReturn {
		if index != 0 {
			if blocksForReturn[index].Parent != blocksForReturn[index-1].BlockName && blockAttribute.Parent != "root" {
				if len(blocksForReturn) != index+1 {
					reverseBlockFirst := blocksForReturn[index]
					reverseBlockSecond := blocksForReturn[index+1]
					blocksForReturn[index] = reverseBlockSecond
					blocksForReturn[index+1] = reverseBlockFirst
				}
			}
		}
	}

	removedShadowRootBlocks := []BlockAttribute{}

	//match := false
	for index, blockAttribute := range blocksForReturn {
		if blockAttribute.BlockName != "root" && blocksForReturn[index-1].BlockName == "root" && blockAttribute.UniqueBlockName == "" {
			removedShadowRootBlocks = append(removedShadowRootBlocks, blocksForReturn[index-1:]...)
			break
		} else {
			removedShadowRootBlocks = blocksForReturn
			break
		}
	}
	mapOfSameLevelBlockCount := make(map[string]int)
	for _, blockAttribute := range removedShadowRootBlocks {
		if blockAttribute.UniqueBlockName == "" && blockAttribute.BlockName != "root" {
			for _, innerBlockAttribute := range removedShadowRootBlocks {
				if innerBlockAttribute.UniqueBlockName == "" && innerBlockAttribute.BlockName != "root" && innerBlockAttribute.Parent == blockAttribute.BlockName {
					mapOfSameLevelBlockCount[blockAttribute.BlockName]++
				}
			}
		}
	}

	for _, blockAttribute := range removedShadowRootBlocks {
		if mapOfSameLevelBlockCount[blockAttribute.BlockName] == 0 {
			mapOfSameLevelBlockCount[blockAttribute.BlockName]++
		}
	}
	/*
		removeOrderedBlocks := []BlockAttribute{}
		if runWithoutUniqueBlocks {
			countBack := 0
			parentBlockName := ""
			captureNewOrderBlocks := []BlockAttribute{}
			for _, blockAttribute := range removedShadowRootBlocks {
				if mapOfSameLevelBlockCount[blockAttribute.BlockName] > 1 && blockAttribute.BlockName != "root" {
					captureNewOrderBlocks = append(captureNewOrderBlocks, blockAttribute)
					countBack = mapOfSameLevelBlockCount[blockAttribute.BlockName]
					parentBlockName = blockAttribute.BlockName

				}

				if countBack > 0 && blockAttribute.BlockName != parentBlockName && blockAttribute.Parent == parentBlockName {
					countBack--
					captureNewOrderBlocks = append(captureNewOrderBlocks, blockAttribute)
				}

				if countBack == 0 && parentBlockName != "" {
					parentBlockName = ""
					captureNewOrderBlocks = append(captureNewOrderBlocks, blockAttribute)
				}
			}

			mapOfUniqueBlockAttributes := make(map[string]bool)
			newOrderBlocks := []BlockAttribute{}
			for _, blockAttribute := range captureNewOrderBlocks {
				if !mapOfUniqueBlockAttributes[blockAttribute.BlockName] {
					newOrderBlocks = append(newOrderBlocks, blockAttribute)
					mapOfUniqueBlockAttributes[blockAttribute.BlockName] = true
				}
			}

			removeOrderedBlocks = []BlockAttribute{}
			for _, blockAttribute := range removedShadowRootBlocks {
				if !mapOfUniqueBlockAttributes[blockAttribute.BlockName] {
					removeOrderedBlocks = append(removeOrderedBlocks, blockAttribute)
				}
			}
			rootBlockNamesNewOrder := []string{}
			for _, blockAttribute := range newOrderBlocks {
				rootBlockNamesNewOrder = append(rootBlockNamesNewOrder, blockAttribute.BlockName)
			}
			mapOfOrder := make(map[string]int)
			removeOrderedBlocks = append(removeOrderedBlocks, newOrderBlocks...)
			for index, blockAttribute := range removeOrderedBlocks {
				for _, name := range rootBlockNamesNewOrder {
					if blockAttribute.BlockName == "root" {
						for index2, rootAttribute := range blockAttribute.RootAttribute {
							if rootAttribute.IsBlock {
								mapOfOrder[rootAttribute.Name] = index2
								if name == rootAttribute.Name {
									if len(blockAttribute.RootAttribute) > 2 {
										moveToLastSpot := removeOrderedBlocks[index].RootAttribute[index2]
										moveToCurrentSpot := removeOrderedBlocks[index].RootAttribute[len(blockAttribute.RootAttribute)-1]
										removeOrderedBlocks[index].RootAttribute[len(blockAttribute.RootAttribute)-1] = moveToLastSpot
										removeOrderedBlocks[index].RootAttribute[index2] = moveToCurrentSpot
										break
									}
								}
							}
						}
					}
				}
			}
			for index, block := range removeOrderedBlocks {
				if block.BlockName != "root" && mapOfOrder[block.BlockName] != 0 {
					fmt.Println("Block name 123123123", block.BlockName, index+mapOfOrder[block.BlockName]-, mapOfOrder[block.BlockName])
				}
			}

			for _, block := range removedOrderedBlocks

		} else {
			return removedShadowRootBlocks
		}
	*/
	return removedShadowRootBlocks
}

func SortRootAttributesForTerraform(rootAttributesToSort []RootAttribute) []RootAttribute {
	nameAttributes := []RootAttribute{}
	blockRootAttributes := []RootAttribute{}
	nonRootBlockAttributes := []RootAttribute{}

	for _, rootAttribute := range rootAttributesToSort {
		if rootAttribute.Name == "name" && !rootAttribute.IsBlock {
			nonRootBlockAttributes = append(nonRootBlockAttributes, rootAttribute)
		}
	}

	for _, rootAttribute := range rootAttributesToSort {
		if rootAttribute.Name != "name" && !rootAttribute.IsBlock {
			nonRootBlockAttributes = append(nonRootBlockAttributes, rootAttribute)
		} else if rootAttribute.IsBlock {
			blockRootAttributes = append(blockRootAttributes, rootAttribute)
		}
	}

	// Concatenate slices to form the final sorted slice
	sortedRootAttributes := append(nameAttributes, nonRootBlockAttributes...)
	sortedRootAttributes = append(sortedRootAttributes, blockRootAttributes...)

	return sortedRootAttributes
}

func NewTerraformResourceDefinitionName(terraformResourceName string, terraformResourceType string, terraformProvider string) string {

	return fmt.Sprintf("resource \"%s_%s\" \"%s\" {", terraformProvider, terraformResourceType, terraformResourceName)
}

func AddObjectTerraformAttributeForResourceDefinition(terraformBlock string, indentation string, terraformAttributes []RootAttribute, allTerraformBlocks []BlockAttribute) (string, int) {
	terraformFlatAttributes := []string{}

	for index, terraformAttribute := range terraformAttributes {
		if !terraformAttribute.IsBlock && index == 0 {
			if !terraformAttribute.IsBlock {
				terraformFlatAttributes = append(terraformFlatAttributes, AddFlatTerraformAttributeForResourceDefinition(terraformBlock, terraformAttribute, false, true))
			}

		} else if !terraformAttribute.IsBlock {
			terraformFlatAttributes = append(terraformFlatAttributes, AddFlatTerraformAttributeForResourceDefinition("", terraformAttribute, false, true))
		}
	}

	return strings.Join(terraformFlatAttributes, "\n"), CountBlockNestingForTerraform(allTerraformBlocks, terraformAttributes)
}

func CountBlockNestingForTerraform(terraformBlockAttributes []BlockAttribute, currentTerraformAttributes []RootAttribute) int {
	countOfNestedObjects := 1
	for _, terraformAttribute := range currentTerraformAttributes {
		if terraformAttribute.IsBlock {
			for _, blockAttribute := range terraformBlockAttributes {
				if terraformAttribute.BlockName == blockAttribute.BlockName && terraformAttribute.UniqueBlockName == blockAttribute.UniqueBlockName {
					countOfNestedObjects++
				}
			}
		}
	}

	return countOfNestedObjects
}

func AddFlatTerraformAttributeForResourceDefinition(terraformBlock string, terraformAttribute RootAttribute, endTerraformDefinition bool, noNewLine bool) string {
	valueTypeName := FindTypeByString(terraformAttribute.Value)
	terraformStringValue := ""
	/*
		if CheckForEmptyArmValue(terraformAttribute.Value) {
			fmt.Println("WE ARE HERE!!!", terraformAttribute.Name)
			valueTypeName = "PLACE-HOLDER-VALUE"
		}
	*/
	if valueTypeName == "string" {
		patternEscapeChars := regexp.MustCompile(`[\\"]`)
		terraformStringValue = terraformAttribute.Value.(string)
		terraformStringValue = patternEscapeChars.ReplaceAllString(terraformStringValue, "")

		if strings.HasPrefix(terraformStringValue, "Microsoft.") && terraformAttribute.Name == "name" {
			splitName := strings.Split(terraformStringValue, ".")
			if len(splitName) == 3 {
				terraformStringValue = fmt.Sprintf("%s/%s", strings.Join(splitName[:2], "."), splitName[2])
			}
		}
	}

	if terraformBlock != "" {
		if terraformAttribute != (RootAttribute{}) {
			if valueTypeName == "bool" {
				return fmt.Sprintf("%s\n%s = %t", terraformBlock, terraformAttribute.Name, terraformAttribute.Value)
			} else if valueTypeName == "slice" {
				convertedValue := ConvertGoSliceToTerraformList(terraformAttribute.Value.([]interface{}))
				return fmt.Sprintf("%s\n%s = %v", terraformBlock, terraformAttribute.Name, convertedValue)
			} else if valueTypeName == "int" || valueTypeName == "float" {
				if strings.Contains(terraformAttribute.Descriptor, "number") {
					return fmt.Sprintf("%s\n%s = %f", terraformBlock, terraformAttribute.Name, terraformAttribute.Value)
				}
			} else {
				return fmt.Sprintf("%s\n%s = \"%s\"", terraformBlock, terraformAttribute.Name, terraformStringValue)
			}
		} else {
			if endTerraformDefinition {
				return terraformBlock
			}
		}
	} else {
		if !endTerraformDefinition && valueTypeName != "string" {
			if valueTypeName == "bool" {
				return fmt.Sprintf("%s\n%s = %t", terraformBlock, terraformAttribute.Name, terraformAttribute.Value)
			} else if valueTypeName == "slice" {
				convertedValue := ConvertGoSliceToTerraformList(terraformAttribute.Value.([]interface{}))
				return fmt.Sprintf("\n%s = %v", terraformAttribute.Name, convertedValue)
			} else if valueTypeName == "int" || valueTypeName == "float" {
				if strings.Contains(terraformAttribute.Descriptor, "number") {
					if valueTypeName == "int" {
						return fmt.Sprintf("%s\n%s = %d", terraformBlock, terraformAttribute.Name, terraformAttribute.Value)
					} else {
						return fmt.Sprintf("%s\n%s = %f", terraformBlock, terraformAttribute.Name, terraformAttribute.Value)
					}
				} else {
					return fmt.Sprintf("%s\n%s = %s", terraformBlock, terraformAttribute.Name, fmt.Sprintf("%f", terraformAttribute.Value.(float64)))
				}
			}
		}
	}

	if !CheckForEmptyArmValue(terraformAttribute.Value) {
		if noNewLine {
			return fmt.Sprintf("%s = \"%s\"", terraformAttribute.Name, terraformStringValue)
		} else {
			return fmt.Sprintf("\n%s = \"%s\"", terraformAttribute.Name, terraformStringValue)
		}
	}
	return ""
}

func ConvertGoSliceToTerraformList(terraformAttribute []interface{}) string {
	dataType := ""
	returnValue := ""
	inCaseOfStringType := []string{}
	for _, value := range terraformAttribute {
		switch value.(type) {
		case string:
			{
				dataType = "string"
				for _, value := range terraformAttribute {
					inCaseOfStringType = append(inCaseOfStringType, value.(string))
				}
			}
		case bool:
			{
				dataType = "bool"
			}
		case int:
			{
				dataType = "int"
			}
		case float64:
			{
				dataType = "float64"
			}
		}
	}

	if dataType == "string" {
		returnValue = fmt.Sprintf("[%s]", fmt.Sprintf("\"%s\"", strings.Join(inCaseOfStringType, `", "`)))
	} else {
		if dataType == "bool" {
			returnValue = fmt.Sprintf("[%s]", terraformAttribute...) //Hopefully not reached : -()
		}

	}

	return returnValue
}

func CheckForTerraformExecuteable() bool {
	response := exec.Command("terraform", "-version")
	if response == nil {
		logVerbose("Terraform was not found on the system, continuing...\nTo install Terraform visit: https://www.codeterraform.com/post/getting-started-with-using-terraform\n")
		return false
	}

	return true
}

func RunTerraformCommand(terraformCommand string, terraformFilePath string) {
	if !CheckForTerraformExecuteable() {
		return
	}
	// Run the command
	cmd := exec.Command("terraform", terraformCommand)
	cmd.Dir = terraformFilePath // Set the directory where the command will run

	// Capture the combined output
	_, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
}

func ChangeExistingFile(filePath string, text string) (int, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logFatal(fmt.Sprintf("an error occured while trying to open the file '%s'\n%s", filePath, err))
	}

	defer file.Close() //Make sure to close the file as the last part of the function call

	numberOfLines, err := file.WriteString(text)
	if err != nil {
		logFatal(fmt.Sprintf("an error occured while trying to write to file '%s'\n%s", filePath, err))
	}

	return numberOfLines, nil
}

func ImportArmFile(filePath *string) ([][]byte, error) {

	var fileNames []string
	var files [][]byte
	fileInfo, err := os.Stat(*filePath)
	if err != nil {
		logFatal(fmt.Sprintf("Error while trying to retrieve ARM json files on path:", string(*filePath), "\n%s", err))
	}

	isDir := fileInfo.IsDir()
	flag.Parse()

	if isDir {
		files, err := os.ReadDir(*filePath)

		if err != nil {
			logFatal(fmt.Sprintf("Error while trying to retrieve ARM json files on path:", string(*filePath), "\n%s", err))
		}

		for _, file := range files {
			if strings.Contains(file.Name(), ".json") {
				fullPath := filepath.Join(*filePath, file.Name())
				fileNames = append(fileNames, fullPath)
			}
		}
	} else {
		fileNames = append(fileNames, *filePath)
	}

	for _, fileName := range fileNames {
		file, err := os.ReadFile(fileName)
		if err != nil {
			logFatal(fmt.Sprintf("Error while trying to retrieve ARM json content on file:", string(fileName), "\n%s", err))
		}

		files = append(files, file)
	}

	return files, err
}

func VerifyArmFile(filecontent [][]byte) [][]byte {
	var jsonDump interface{}
	var validJson [][]byte
	var cleanFilecontent [][]byte
	var armMatch bool

	ReturnStatusObject.CountAnalyzedARMTemplates++

	for _, fileContent := range filecontent {
		err := json.Unmarshal(fileContent, &jsonDump)
		if err != nil {
			continue
		} else {
			validJson = append(validJson, fileContent)
		}
	}

	for _, cleanContent := range validJson {
		armMatch = false
		json.Unmarshal(cleanContent, &jsonDump)
		testMap, ok := jsonDump.(map[string]interface{})
		if ok {
			for attributeName := range testMap {
				if attributeName == "properties" {
					armMatch = true
					break
				}
			}
		}
		switch jsonDump.(type) {
		case []interface{}:
			{
				for _, slice := range jsonDump.([]interface{}) {
					for attributeName, _ := range slice.(map[string]interface{}) {
						if attributeName == "properties" {
							armMatch = true
							break
						}
					}
				}
			}
		}
		if armMatch {
			cleanFilecontent = append(cleanFilecontent, cleanContent)
		}
	}

	return cleanFilecontent
}

func GetArmBaseInformation(filecontent [][]byte) []ArmObject {
	var jsonInterface interface{}
	var armBasicObjects []ArmObject
	var jsonMap map[string]interface{}
	var armResourceTypes []string

	for _, bytes := range filecontent {
		err := json.Unmarshal(bytes, &jsonInterface)

		if err != nil {
			logFatal(fmt.Sprintf("Error while transforming file from bytes to json: %s", err))
		}

		switch v := jsonInterface.(type) {
		case map[string]interface{}:
			{
				specialResourceType := ""
				jsonMap = jsonInterface.(map[string]interface{})
				armResourceTypes = GetArmBaseInformationResourceTypes(jsonMap["properties"])

				uniqueArmResourceTypesMap := make(map[string]bool)
				uniqueArmResourceTypes := []string{}

				for _, resourceValue := range armResourceTypes {
					if !uniqueArmResourceTypesMap[resourceValue] {
						uniqueArmResourceTypesMap[resourceValue] = true
						uniqueArmResourceTypes = append(uniqueArmResourceTypes, resourceValue)
					}
				}

				if jsonMap["kind"] != nil && strings.Contains(strings.ToLower(jsonMap["type"].(string)), "web/sites") {
					specialResourceType = GetArmWebAndComputeKind(jsonMap["kind"])
				} else if strings.ToLower(jsonMap["type"].(string)) == "microsoft.compute/virtualmachines" {
					specialResourceType = GetArmWebAndComputeKind(jsonMap["properties"])
				}

				armObject := ArmObject{
					Name:                  jsonMap["name"].(string),
					Resource_id:           jsonMap["id"].(string),
					Resource_types:        append(uniqueArmResourceTypes, jsonMap["type"].(string)),
					Location:              jsonMap["location"].(string),
					Resource_group_name:   strings.Split(jsonMap["id"].(string), "/")[4],
					Properties:            jsonMap["properties"],
					Special_resource_type: specialResourceType,
					Subscription_id:       strings.Split(jsonMap["id"].(string), "/")[2],
				}
				armBasicObjects = append(armBasicObjects, armObject)

			}
		case []interface{}:
			{
				for _, item := range v {
					specialResourceType := ""
					jsonMap = item.(map[string]interface{})
					armResourceTypes = GetArmBaseInformationResourceTypes(jsonMap["properties"])

					uniqueArmResourceTypesMap := make(map[string]bool)
					uniqueArmResourceTypes := []string{}

					for _, resourceValue := range armResourceTypes {
						if !uniqueArmResourceTypesMap[resourceValue] {
							uniqueArmResourceTypesMap[resourceValue] = true
							uniqueArmResourceTypes = append(uniqueArmResourceTypes, resourceValue)
						}
					}

					if jsonMap["kind"] != nil && strings.Contains(strings.ToLower(jsonMap["type"].(string)), "web/sites") {
						specialResourceType = GetArmWebAndComputeKind(jsonMap["kind"])
					} else if strings.ToLower(jsonMap["type"].(string)) == "microsoft.compute/virtualmachines" {
						specialResourceType = GetArmWebAndComputeKind(jsonMap["properties"])
					}

					armObject := ArmObject{
						Name:                  jsonMap["name"].(string),
						Resource_id:           jsonMap["id"].(string),
						Resource_types:        append(uniqueArmResourceTypes, jsonMap["type"].(string)),
						Location:              jsonMap["location"].(string),
						Resource_group_name:   strings.Split(jsonMap["id"].(string), "/")[4],
						Properties:            jsonMap["properties"],
						Special_resource_type: specialResourceType,
						Subscription_id:       strings.Split(jsonMap["id"].(string), "/")[2],
					}
					armBasicObjects = append(armBasicObjects, armObject)
				}
			}
		}
	}

	return armBasicObjects
}

func GetArmBaseInformationResourceTypes(armPropertyValue interface{}) []string {
	var armBaseResourceTypes []string

	patternMicrosoftResourceType := regexp.MustCompile(`(?i)^microsoft\.[^/]+(/[^/]+)+$`)

	if typeOfValue := reflect.ValueOf(armPropertyValue); typeOfValue.Kind() == reflect.Map {
		for _, armAttributeValue := range armPropertyValue.(map[string]interface{}) {
			switch armAttributeValue.(type) {
			case []interface{}:
				{
					for _, innerSliceAttributeValue := range armAttributeValue.([]interface{}) {
						if CheckForMap(innerSliceAttributeValue) {
							for innerAttributeName, innerAttributeValue := range innerSliceAttributeValue.(map[string]interface{}) {
								if CheckForString(innerAttributeValue) {
									if strings.ToLower(innerAttributeName) == "type" && patternMicrosoftResourceType.MatchString(innerAttributeValue.(string)) {
										armBaseResourceTypes = append(armBaseResourceTypes, innerAttributeValue.(string))
									}
								}
							}
						} else if CheckForSlice(innerSliceAttributeValue) {
							for _, miniumumSliceAttributeValue := range innerSliceAttributeValue.([]interface{}) {
								if CheckForMap(miniumumSliceAttributeValue) {
									for minimumAttributeName, minimumAttributeValue := range innerSliceAttributeValue.(map[string]interface{}) {
										if CheckForString(minimumAttributeValue) {
											if strings.ToLower(minimumAttributeName) == "type" && patternMicrosoftResourceType.MatchString(minimumAttributeValue.(string)) {
												armBaseResourceTypes = append(armBaseResourceTypes, minimumAttributeValue.(string))
											}
										}
									}
								}
							}
						}
					}
				}
			case map[string]interface{}:
				{
					for innerAttributeName, innerAttributeValue := range armAttributeValue.(map[string]interface{}) {
						if CheckForString(innerAttributeValue) {
							if strings.ToLower(innerAttributeName) == "type" && patternMicrosoftResourceType.MatchString(innerAttributeValue.(string)) {
								armBaseResourceTypes = append(armBaseResourceTypes, innerAttributeValue.(string))
							}
						}
					}
				}
			}
		}
	}

	return armBaseResourceTypes
}

func GetArmWebAndComputeKind(armPropertyValue interface{}) string {
	var returnArmKind string
	var tempReturnArmKind string

	switch armPropertyValue.(type) {
	case map[string]interface{}:
		{
			for attributeName, attributeValue := range armPropertyValue.(map[string]interface{}) {
				if strings.ToLower(attributeName) == "storageprofile" {
					for innerAttributeName, innerAttributeValue := range attributeValue.(map[string]interface{}) {
						if strings.ToLower(innerAttributeName) == "osdisk" {
							for _, minimumAttributeValue := range innerAttributeValue.(map[string]interface{}) {
								if CheckForString(minimumAttributeValue) {
									if strings.ToLower(minimumAttributeValue.(string)) == "windows" || strings.ToLower(minimumAttributeValue.(string)) == "linux" {
										tempReturnArmKind = fmt.Sprintf("%s_%s", strings.ToLower(minimumAttributeValue.(string)), "virtual_machine")
									}
								}
							}
						}
					}
				}
			}
		}

	case string: //Need to change the name this produces
		{
			attributePartName := strings.Split(armPropertyValue.(string), ",")
			if attributePartName[0] == "app" {
				tempReturnArmKind = fmt.Sprintf("%s_%s", attributePartName[1], "web_app")
			} else if attributePartName[0] == "func" {
				tempReturnArmKind = fmt.Sprintf("%s_%s", attributePartName[1], "funcion_app")
			}
		}
	}
	returnArmKind = tempReturnArmKind

	return returnArmKind
}

func CheckForString(armPropertyValue interface{}) bool {
	var isString bool
	switch armPropertyValue.(type) {
	case string:
		{
			isString = true
		}
	}

	return isString
}

func ConvertArmToTerraformProvider(resourceType string, specialResouceType string) string { //Many issues in this function, many resource types are incorrectly matched
	var convertResourceTypeName string
	var convertResourceTypeTempName string
	var resourceName string

	resourceTypeLower := strings.ToLower(resourceType)
	resourceNamePart := strings.Split(resourceType, "/")

	patternResourceTypeLonger := regexp.MustCompile("([a-z0-9])([A-Z])")
	patternResourceType := regexp.MustCompile(`(?i)^microsoft\.[a-zA-Z0-9]+/[a-zA-Z0-9]+$`)

	checkNameForInsights := regexp.MustCompile(`\binsights\b`)
	checkNameForAlerts := regexp.MustCompile(`(?i)alert`)
	checkNameForNetwork := regexp.MustCompile(`\bnetwork\b`)
	checkNamesForNetworkIP := regexp.MustCompile(`IPAddresses|IPAddress`)
	checkNamesForWeb := regexp.MustCompile(`\bweb\b`)
	checkNamesForCompute := regexp.MustCompile(`\bcompute\b`)
	checkNamesForLogAnalytics := regexp.MustCompile(`\boperationalinsights\b`)
	checkNameForMsSQL := regexp.MustCompile(`\bsql\b`)

	resourceNameBaseConversion := strings.Split(strings.ToLower(patternResourceTypeLonger.ReplaceAllString(resourceType, "${1}_${2}")), "/")
	if checkNameForInsights.MatchString(resourceTypeLower) || checkNameForAlerts.MatchString(resourceTypeLower) {
		resourceName = resourceNameBaseConversion[1]
		if strings.Contains(resourceName, "alert") || !strings.Contains(resourceName, "rule") {
			convertResourceTypeTempName = fmt.Sprintf("%s_%s", "monitor", resourceName)
		} else {
			convertResourceTypeTempName = fmt.Sprintf("%s_%s_%s", "monitor", resourceName, "alert")
		}
	} else if checkNameForNetwork.MatchString(resourceTypeLower) {
		//convert any ARM ipAddress to ip
		if checkNamesForNetworkIP.MatchString(resourceType) {
			convertResourceTypeTempName = strings.Split(checkNamesForNetworkIP.ReplaceAllString(resourceType, "_ip"), "/")[1]
		}

		if strings.Split(resourceType, "/")[1] == "connections" {
			convertResourceTypeTempName = "virtual_network_gateway_connection" //Static conversion because its such an edge case
		}

		if strings.Split(resourceType, "/")[1] == "dnsForwardingRulesets" { //Static conversion because its such an edge case
			convertResourceTypeName = "private_dns_resolver_dns_forwarding_ruleset"
		}

		if strings.Contains(resourceTypeLower, "resolver") || strings.Contains(resourceTypeLower, "forward") {

			if len(resourceNameBaseConversion) == 2 {
				if strings.Contains(resourceTypeLower, "forward") {
					convertResourceTypeTempName = fmt.Sprintf("%s_%s", "private_dns_resolver", resourceNameBaseConversion[1])
				} else {
					convertResourceTypeTempName = fmt.Sprintf("%s_%s", "private", resourceNameBaseConversion[1])
				}
			} else {
				convertResourceTypeTempName = fmt.Sprintf("%s_%s", "private", fmt.Sprintf("%s_%s", strings.TrimSuffix(resourceNameBaseConversion[1], "s"), resourceNameBaseConversion[2]))
			}
		}

		attributePartName := strings.Split(resourceType, "/")

		if len(attributePartName) == 3 && convertResourceTypeTempName == "" {
			if strings.TrimSuffix(resourceNameBaseConversion[1], "s") == "network_watcher" {
				convertResourceTypeTempName = fmt.Sprintf("%s_%s", "network_watcher", resourceNameBaseConversion[2])
			} else {
				convertResourceTypeTempName = resourceNameBaseConversion[2]
			}
		}
	} else if checkNamesForWeb.MatchString(resourceTypeLower) {
		if strings.ToLower(strings.Split(resourceTypeLower, "/")[1]) == "serverfarms" {
			convertResourceTypeTempName = "service_plan" //Static conversion
		} else if specialResouceType != "" {
			attributePartName := strings.Split(specialResouceType, "_")
			if len(attributePartName) > 2 {
				convertResourceTypeTempName = specialResouceType
			}
		}
	} else if checkNamesForCompute.MatchString(resourceTypeLower) {
		if specialResouceType != "" {
			convertResourceTypeTempName = strings.ToLower(specialResouceType)
		}
	} else if checkNamesForLogAnalytics.MatchString(resourceTypeLower) {
		if len(resourceNameBaseConversion) > 2 {

		} else {
			convertResourceTypeTempName = fmt.Sprintf("%s_%s", "log_analytics", resourceNameBaseConversion[1]) //Static base conversion
		}

	} else if checkNameForMsSQL.MatchString(resourceTypeLower) {
		if len(resourceNameBaseConversion) > 2 {
		} else {
			if len(strings.Split(resourceType, "/")) == 2 {
				attributePartName := strings.Split(resourceNameBaseConversion[0], ".")[1]
				convertResourceTypeTempName = fmt.Sprintf("%s%s_%s", "ms", attributePartName, strings.Split(resourceTypeLower, "/")[1])
			} else {
			}
		}
	}

	if patternResourceTypeLonger.MatchString(resourceType) && convertResourceTypeTempName == "" {
		if len(resourceNamePart) <= 3 {
			if len(resourceNamePart) > 1 && len(resourceNamePart) < 3 {
				if matchCapsLock := regexp.MustCompile(`[A-Z]`); !matchCapsLock.MatchString(strings.Split(resourceType, "/")[1]) {
					secondPartOfRootName := strings.Split(resourceNameBaseConversion[0], "_")
					if len(resourceNameBaseConversion) > 1 && len(secondPartOfRootName) > 1 {
						if strings.TrimSuffix(resourceNameBaseConversion[len(resourceNameBaseConversion)-1], "s") == secondPartOfRootName[1] {
							convertResourceTypeTempName = strings.Split(resourceNameBaseConversion[0], ".")[1]
						} else {
							convertResourceTypeTempName = fmt.Sprintf("%s_%s", strings.Split(resourceNameBaseConversion[0], ".")[1], strings.TrimSuffix(resourceNameBaseConversion[len(resourceNameBaseConversion)-1], "s"))
						}
					}
				} else {
					if len(strings.Split(resourceType, "/")) > 2 {
						attributePartName := strings.Split(resourceNameBaseConversion[2], "_")
						if len(attributePartName) > 2 {
							partName := strings.Join(attributePartName[:2], "_")
							if strings.TrimSuffix(resourceNameBaseConversion[1], "s") == partName {
								convertResourceTypeTempName = fmt.Sprintf("%s_%s", partName, attributePartName[len(attributePartName)-1])
							}
						} else {
							convertResourceTypeTempName = fmt.Sprintf("%s_%s", strings.TrimSuffix(resourceNameBaseConversion[1], "s"), resourceNameBaseConversion[2])
						}
					} else {
						convertResourceTypeTempName = strings.Split(strings.ToLower(patternResourceTypeLonger.ReplaceAllString(resourceType, "${1}_${2}")), "/")[1]
					}

					if strings.HasSuffix(convertResourceTypeTempName, "identities") {
						convertResourceTypeTempName = strings.Replace(convertResourceTypeTempName, "identities", "identity", 1)
					}
				}
			} else {
				dotpartAttributeName := strings.Split(resourceNameBaseConversion[0], ".")[1]
				attributePartName := strings.Split(dotpartAttributeName, "_")
				if len(attributePartName) > 1 {
					if attributePartName[1] == strings.TrimSuffix(resourceNameBaseConversion[1], "s") {
						convertResourceTypeTempName = fmt.Sprintf("%s_%s", dotpartAttributeName, resourceNameBaseConversion[2])
					}
				}
			}
		}
	} else if patternResourceType.MatchString(resourceType) && convertResourceTypeTempName == "" {
		convertResourceTypeTempName = fmt.Sprintf("%s_%s", strings.Split(resourceNameBaseConversion[0], ".")[1], resourceNameBaseConversion[1])
	}

	convertResourceTypeName = strings.ToLower(strings.TrimSuffix(convertResourceTypeTempName, "s"))

	return convertResourceTypeName
}

func GetArmResourceTypes(armBaseTypes []string, htmlObjects []HtmlObject) []ArmResourceType {
	var returnArmResourceTypes []ArmResourceType
	var rootResourceType string

	for _, armBaseType := range armBaseTypes {
		if len(strings.Split(armBaseType, "/")) == 2 {
			rootResourceType = armBaseType
		} else if strings.Contains(armBaseType, "_") {
			rootResourceType = armBaseType
		}
	}

	armResourceType := ArmResourceType{ //The root resource type itself
		Resource_type:  rootResourceType,
		CanBeSeperated: false,
		Parent:         "root",
	}

	returnArmResourceTypes = append(returnArmResourceTypes, armResourceType)

	for _, armBaseType := range armBaseTypes {
		if armBaseType == rootResourceType {
			for _, htmlObject := range htmlObjects {
				if htmlObject.Resource_type == armBaseType {
					for _, htmlAttribute := range htmlObject.Attribute {
						for _, armNoneRoot := range armBaseTypes {
							convertArmName := ConvertArmToTerraformProvider(armNoneRoot, "")
							if htmlAttribute.Name == convertArmName {
								armResourceType := ArmResourceType{
									Resource_type:  armNoneRoot,
									CanBeSeperated: true,
									Parent:         armBaseType,
								}
								returnArmResourceTypes = append(returnArmResourceTypes, armResourceType)
							}
						}
					}
				}
			}
		}
	}

	for _, armBaseType := range armBaseTypes {
		if armBaseType != rootResourceType {
			match := false
			for _, armResourceType := range returnArmResourceTypes {
				if armResourceType.Resource_type == armBaseType {
					match = true
				}
			}
			if !match && armBaseType != "" {
				armResourceType := ArmResourceType{
					Resource_type:  armBaseType,
					CanBeSeperated: false,
				}
				returnArmResourceTypes = append(returnArmResourceTypes, armResourceType)
			}
		}
	}

	return returnArmResourceTypes
}

func ConvertArmAttributeName(armPropertyName string, armPropertyValue interface{}) string {
	var returnConvertedName string

	patternLongArmAttributeName := regexp.MustCompile("([a-z0-9])([A-Z])")
	if patternLongArmAttributeName.MatchString(armPropertyName) {
		if strings.HasSuffix(armPropertyName, "s") && !strings.HasSuffix(armPropertyName, "es") {
			returnConvertedName = strings.ToLower(strings.TrimSuffix(patternLongArmAttributeName.ReplaceAllString(armPropertyName, "${1}_${2}"), "s"))
		} else {
			returnConvertedName = strings.ToLower(strings.TrimSuffix(patternLongArmAttributeName.ReplaceAllString(armPropertyName, "${1}_${2}"), "es"))
		}
	} else {
		returnConvertedName = armPropertyName
	}

	returnConvertedName = strings.TrimSuffix(returnConvertedName, "s")

	return returnConvertedName
}

func GetRawHtml(resourceType string, providerVersion string, customChromePath string) (string, error) {
	var HtmlBody string
	var HtmlBodyCompare string
	var convertResourceTypeName string

	if strings.Contains(resourceType, "/") {
		convertResourceTypeName = ConvertArmToTerraformProvider(resourceType, "")
	} else {
		convertResourceTypeName = resourceType
	}

	if convertResourceTypeName == "" {
		return "", nil
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	//FOR LATER USE, VERSION 0.1.0 WILL ONLY SUPPORT GOOGLE CHROME
	/*
			if useEdge {
			// Use Edge (Chromium-based)
			opts = append(chromedp.DefaultExecAllocatorOptions[:],
				chromedp.ExecPath(getEdgeExecPath()), // Automatically finds Edge executable path
				chromedp.Flag("headless", true),      // Set to false to see the browser UI
				chromedp.Flag("disable-gpu", true),
			)
		} else {
			// Use Chrome
			opts = append(chromedp.DefaultExecAllocatorOptions[:],
				chromedp.Flag("headless", true), // Set to false to see the browser UI
				chromedp.Flag("disable-gpu", true),
			)
		}
	*/

	// Create a new Chrome instance
	opts := []func(*chromedp.ExecAllocator){}
	if customChromePath == "" {
		opts = append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true), // Set to false if you want to see the browser UI
			chromedp.Flag("disable-gpu", true),
		)
	} else {
		opts = append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(customChromePath),
			chromedp.Flag("headless", true), // Set to false if you want to see the browser UI
			chromedp.Flag("disable-gpu", true),
		)
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	url := fmt.Sprintf("https://registry.terraform.io/providers/hashicorp/azurerm/%s/docs/resources/%s", providerVersion, convertResourceTypeName)
	logVerbose(fmt.Sprintf("Retrieving Terraform information from: %s", url))
	HtmlBaseURL = url

	for x := 1; x < 25; x++ {
		err := chromedp.Run(taskCtx,
			chromedp.Navigate(string(url)),
			chromedp.WaitReady("body"),                   // Wait for the body to be ready
			chromedp.Sleep(time.Duration(x)*time.Second), // Give some time for JS to execute
			chromedp.InnerHTML("*", &HtmlBody, chromedp.ByQueryAll),
		)

		if err != nil {
			return "", err
		}

		if bytes.Equal([]byte(HtmlBody), []byte(HtmlBodyCompare)) {
			break
		}

		HtmlBodyCompare = HtmlBody

	}

	if strings.Contains(HtmlBodyCompare, "not-found") {
		logWarning(fmt.Sprintf("The resource: %s could not be found at provider version: %s", convertResourceTypeName, providerVersion))
		logVerbose(fmt.Sprintf("The resource type: %s could not be found..\nThis might be due to an invalid translation...\nIf so, please create a github issue at: https://github.com/ChristofferWin/TerrafyARM/issues", convertResourceTypeName))
		ReturnStatusObject.CountNotFoundResourceType++
		return "not_found", nil
	}
	return HtmlBodyCompare, nil
}

func SortRawHtml(rawHtml string, resourceType string, logTime string) HtmlObject { //See the struct type definitions towards the top of the file
	//var uniqueAttributeNames []string
	var uniqueHtmlAttributes []Attribute
	var allHtmlAttributes []Attribute

	//Defining the boundaries of the data we are interested in
	startIndex := regexp.MustCompile(`name="arguments?-references?"`).FindStringIndex(rawHtml)
	endIndex := regexp.MustCompile(`name="attributes-references?"`).FindStringIndex(rawHtml)

	//Isolating only the 'argument references from the HTML dump'
	extractedText := rawHtml[startIndex[1]:endIndex[0]]

	linesHtml := strings.Split(extractedText, "\n")
	patternHtmlBlockName := regexp.MustCompile(`<code>([^<]+)</code>`)
	antiPatternBlockName := regexp.MustCompile(`bblock\b`)
	patternDescriptor := regexp.MustCompile(`</a>\s*-\s*\((Optional|Required)\)\s*(.*)</li>`)

	for index, line := range linesHtml {
		blockNameStringSubMatch := patternHtmlBlockName.FindStringSubmatch(line)
		typeOfHtmlAttribute := ""
		parentName := ""
		descriptorSlice := patternDescriptor.FindStringSubmatch(line)
		descriptor := ""
		if len(descriptorSlice) > 1 {
			descriptor = descriptorSlice[2]
		}
		blockNameStringSubMatchInner := []string{}
		if strings.Contains(line, "block supports") && len(blockNameStringSubMatch) > 1 {
			required := false

			if checkForRequired := regexp.MustCompile(`\b(Required)\b`); checkForRequired.MatchString(line) {
				required = true
			}
			htmlAttribute := Attribute{
				Name:       blockNameStringSubMatch[1],
				Type:       "armObject",
				Parent:     "",
				Descriptor: descriptor,
				Required:   required,
			}
			allHtmlAttributes = append(allHtmlAttributes, htmlAttribute)
		} else if !antiPatternBlockName.MatchString(line) && len(blockNameStringSubMatch) > 1 {
			captureOfLines := linesHtml[:index]
			for reverseIndex := len(captureOfLines) - 1; reverseIndex >= 0; reverseIndex-- {
				if strings.Contains(captureOfLines[reverseIndex], "block supports") {
					blockNameStringSubMatchInner = patternHtmlBlockName.FindStringSubmatch(captureOfLines[reverseIndex])
					if len(blockNameStringSubMatchInner) > 1 {
						parentName = blockNameStringSubMatchInner[1]
					}
					break
				}
			}

			required := false

			if checkForRequired := regexp.MustCompile(`\b(Required)\b`); checkForRequired.MatchString(line) {
				required = true
			}

			if checkForList := regexp.MustCompile(`\blist\b`); checkForList.MatchString(line) {
				typeOfHtmlAttribute = "list"
			} else if checkForBool := regexp.MustCompile(`(?i)Defaults to <code>(true|false)</code>`); checkForBool.MatchString(line) {
				typeOfHtmlAttribute = "bool"
			} else {
				typeOfHtmlAttribute = "string"
			}
			htmlAttiribute := Attribute{
				Name:         blockNameStringSubMatch[1],
				Type:         typeOfHtmlAttribute,
				Parent:       parentName,
				Descriptor:   descriptor,
				Required:     required,
				TerraformURL: fmt.Sprintf("%s#%s-1", HtmlBaseURL, blockNameStringSubMatch[1]),
			}
			allHtmlAttributes = append(allHtmlAttributes, htmlAttiribute)
		}
	}

	mapOfHtmlAttributesToCache := make(map[string]bool)
	mapOfHtmlAttributes := make(map[string]bool)

	patternValidHtmlAttributeName := regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)

	for _, htmlAttribute := range allHtmlAttributes {
		if !mapOfHtmlAttributesToCache[htmlAttribute.Name] && patternValidHtmlAttributeName.MatchString(htmlAttribute.Name) {
			uniqueHtmlAttributes = append(uniqueHtmlAttributes, htmlAttribute)
			if htmlAttribute.Name != "name" {
				mapOfHtmlAttributesToCache[htmlAttribute.Name] = true
			}
		}

		if !mapOfHtmlAttributes[fmt.Sprintf("%s,%s", htmlAttribute.Name, htmlAttribute.Parent)] && patternValidHtmlAttributeName.MatchString(htmlAttribute.Name) {
			if htmlAttribute.Name != "name" {
				mapOfHtmlAttributes[fmt.Sprintf("%s,%s", htmlAttribute.Name, htmlAttribute.Parent)] = true
			}
		}

	}

	for index, attribute := range uniqueHtmlAttributes {
		for name := range mapOfHtmlAttributes {
			nameSplit := strings.Split(name, ",") //[0] == name, [1] == parent
			if len(nameSplit) > 1 {
				if attribute.Name == nameSplit[0] && attribute.Parent != nameSplit[1] {
					captureHtmlAttribute := Attribute{}
					for _, innerAttribute := range allHtmlAttributes {
						if innerAttribute.Name == nameSplit[0] && innerAttribute.Parent == nameSplit[1] {
							captureHtmlAttribute = innerAttribute
							break
						}
					}
					if captureHtmlAttribute.Parent != attribute.Name {
						shadowCopy := ShadowAttribute{
							Type:       captureHtmlAttribute.Type,
							Parent:     captureHtmlAttribute.Parent,
							Required:   captureHtmlAttribute.Required,
							Descriptor: captureHtmlAttribute.Descriptor,
						}
						uniqueHtmlAttributes[index].ShadowCopy = shadowCopy
					}
				}
			}
		}
	}

	for index, htmlAttribute := range uniqueHtmlAttributes {
		match := false
		if htmlAttribute.Type != "armObject" {
			for _, innerHtmlAttribute := range uniqueHtmlAttributes {
				if htmlAttribute.Name == innerHtmlAttribute.Parent {
					match = true
					break
				}
			}
			if match {
				uniqueHtmlAttributes[index].Type = "armObject"
			}
		}
		if htmlAttribute.Parent == "" {
			uniqueHtmlAttributes[index].Parent = "root"
		}
	}
	//Adding all the sorted attributes to the final return armObject
	htmlObject := HtmlObject{
		Resource_type: resourceType,
		Attribute:     uniqueHtmlAttributes,
		Last_updated:  logTime,
	}

	return htmlObject
}

func ConvertFromStringToSlice(stringToSlice string, seperatorChar string) []string {
	arrayOfSlices := strings.Split(strings.TrimSuffix(strings.TrimPrefix(stringToSlice, "{"), "}"), seperatorChar)
	return arrayOfSlices
}

func UniquifyResourceTypes(resourceTypes []string) []string {
	mapOfResourceTypes := make(map[string]bool)
	var sortedResourceTypes []string

	for _, resourceType := range resourceTypes {
		if !mapOfResourceTypes[resourceType] {
			sortedResourceTypes = append(sortedResourceTypes, resourceType)
			mapOfResourceTypes[resourceType] = true
		}
	}

	return sortedResourceTypes
}

func CompileTerraformObjects(armBasicObjects []ArmObject, htmlObjects []HtmlObject, seperateArmResources bool) []CompileObject {
	var htmlObjectCaptureSeperatedResource HtmlObject
	var compiledObjects []CompileObject
	var htmlObjectCaptureRootArmResources HtmlObject
	var htmlObjectCaptureNestedArmResources HtmlObject
	var masterKeyRoot string

	for _, armBasicObject := range armBasicObjects {
		var htmlObjectCaptures []HtmlObject
		blockObjectsNestedResources := []BlockAttribute{}
		blockObjectsRootResources := []BlockAttribute{}
		blockObjectsSeperatedResources := []BlockAttribute{}
		rootAttributesFromNestedResources := []RootAttribute{}
		rootAttributesFromSeperatedResources := []RootAttribute{}
		mapOfRootAttributes := make(map[string][]RootAttribute)

		resourceTypes := []string{}
		resourceTypesModified := []string{}
		rootAttributesForReturn := []RootAttribute{}

		combinedResourceTypes := []string{}
		if armBasicObject.Special_resource_type != "" {
			combinedResourceTypes = append(combinedResourceTypes, armBasicObject.Special_resource_type)
		} else {
			combinedResourceTypes = append(combinedResourceTypes, armBasicObject.Resource_types...)
		}
		allResourceTypes := GetArmResourceTypes(combinedResourceTypes, htmlObjects)

		for _, armResourceType := range allResourceTypes {
			resourceTypes = append(resourceTypes, armResourceType.Resource_type)
			for _, htmlObject := range htmlObjects {
				if htmlObject.Resource_type == armResourceType.Resource_type {
					htmlObjectCaptures = append(htmlObjectCaptures, htmlObject)
				}
			}
		}

		for _, resourceType := range resourceTypes {
			resourceTypeModified := strings.Split(resourceType, "/")
			if len(resourceTypeModified) == 3 {
				resourceTypesModified = append(resourceTypesModified, ConvertArmAttributeName(resourceTypeModified[2], nil))
			}
		}

		for _, armResourceType := range allResourceTypes {
			resourceFileName := ""
			keyForSubType := ""
			masterKey := ""
			if armResourceType.CanBeSeperated {
				captureHtml := HtmlObject{}
				for _, htmlObject := range htmlObjectCaptures {
					if armResourceType.Resource_type == htmlObject.Resource_type {
						captureHtml = htmlObject
					}

					if armResourceType.Parent == htmlObject.Resource_type {
						htmlObjectCaptureNestedArmResources = htmlObject //Must have an if statement for when we enable the use of forcing the seperation of resources
					}
				}

				attributePartName := strings.Split(ConvertArmAttributeName(captureHtml.Resource_type, nil), "/")
				if len(attributePartName) > 2 {
					for attributeName, _ := range armBasicObject.Properties.(map[string]interface{}) {
						if strings.Contains(attributeName, attributePartName[2]) {
							keyForSubType = attributeName
						}
					}
					retrievePartOfArmProperties := armBasicObject.Properties.(map[string]interface{})[keyForSubType]
					for _, slice := range retrievePartOfArmProperties.([]interface{}) {
						masterKey = GetArmMasterKey(slice)
						rootAttributesFromNestedResources = append(rootAttributesFromNestedResources, GetInnerRootAttributes(slice, captureHtml, masterKey, keyForSubType)...)
					}
				}

			} else if !(len(strings.Split(armResourceType.Resource_type, "/")) == 2) && strings.Contains(armResourceType.Resource_type, "/") {
				captureHtml := HtmlObject{}
				for _, htmlObject := range htmlObjectCaptures {
					if armResourceType.Resource_type == htmlObject.Resource_type {
						captureHtml = htmlObject
						htmlObjectCaptureSeperatedResource = htmlObject //Must have an if statement for when we enable the use of forcing the seperation of resources
					}
				}
				attributePartName := strings.Split(ConvertArmAttributeName(captureHtml.Resource_type, nil), "/")
				if len(attributePartName) > 2 {
					for attributeName, _ := range armBasicObject.Properties.(map[string]interface{}) {
						convertAttributeName := ConvertArmAttributeName(attributeName, "")
						if strings.Contains(convertAttributeName, attributePartName[2]) { //Nil exception around here - We have opened for the virtual_network_peering
							keyForSubType = attributeName
						}
					}
					retrievePartOfArmProperties := armBasicObject.Properties.(map[string]interface{})[keyForSubType]
					if CheckForSliceWithMaps(retrievePartOfArmProperties) {
						for _, slice := range retrievePartOfArmProperties.([]interface{}) {
							masterKey = GetArmMasterKey(slice)
							mapOfRootAttributes[masterKey] = append(rootAttributesFromSeperatedResources, GetInnerRootAttributes(slice, captureHtml, masterKey, keyForSubType)...)
						}
					}
				}
			} else {
				masterKeyRoot = armBasicObject.Name
				for attributeName, attributeValue := range armBasicObject.Properties.(map[string]interface{}) {
					captureHtml := HtmlObject{}
					for _, htmlObjectCapture := range htmlObjectCaptures {
						if len(strings.Split(htmlObjectCapture.Resource_type, "/")) == 2 || strings.Contains(htmlObjectCapture.Resource_type, "_") {
							captureHtml = htmlObjectCapture
							htmlObjectCaptureRootArmResources = htmlObjectCapture
						}
					}
					convertAttributeName := ConvertArmAttributeName(attributeName, nil)
					if !strings.Contains(strings.Join(resourceTypesModified, ","), convertAttributeName) {
						htmlAttributeMatch := GetHtmlAttributeMatch(attributeName, captureHtml.Attribute, attributeValue, "")
						if len(htmlAttributeMatch) > 0 && !CheckForEmptyArmValue(attributeValue) {
							for _, match := range htmlAttributeMatch {
								if CheckForMap(attributeValue) {
									rootAttributesForReturn = append(rootAttributesForReturn, ConvertMapToRootAttribute(match.Name, attributeValue, captureHtml.Attribute, match.Parent, ""))
								} else {
									rootAttributesForReturn = append(rootAttributesForReturn, ConvertFlatValueToRootAttribute(attributeValue, match, match.Parent, ""))
								}
							}
						}
						if CheckForMap(attributeValue) {
							for innerAttributeName, innerAttributeValue := range attributeValue.(map[string]interface{}) {
								innerHtmlAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, captureHtml.Attribute, innerAttributeValue, "")
								if len(innerHtmlAttributeMatch) > 0 && !CheckForEmptyArmValue(innerAttributeValue) {
									for _, match := range innerHtmlAttributeMatch {
										if !CheckForMap(innerAttributeValue) && !CheckForSliceWithMaps(innerAttributeValue) && !CheckForEmptyArmValue(innerAttributeValue) {
											rootAttributesForReturn = append(rootAttributesForReturn, ConvertFlatValueToRootAttribute(innerAttributeValue, match, match.Parent, ""))
										}
									}
								}

								if CheckForMap(innerAttributeValue) {
									for minimumAttributeName, minimumAttributeValue := range innerAttributeValue.(map[string]interface{}) {
										minimumHtmlAttributeMatch := GetHtmlAttributeMatch(minimumAttributeName, captureHtml.Attribute, minimumAttributeValue, innerAttributeName)
										if len(minimumHtmlAttributeMatch) > 0 && !CheckForEmptyArmValue(minimumAttributeValue) {
											for _, match := range minimumHtmlAttributeMatch {
												if !CheckForMap(minimumAttributeValue) && !CheckForSliceWithMaps(minimumAttributeValue) && !CheckForEmptyArmValue(minimumAttributeValue) {
													rootAttributesForReturn = append(rootAttributesForReturn, ConvertFlatValueToRootAttribute(minimumAttributeValue, match, match.Parent, ""))
												}
											}
										}
										if CheckForMap(minimumAttributeValue) {
											for deepAttributeName, deepAttributeValue := range minimumAttributeValue.(map[string]interface{}) {
												deepHtmlAttributeMatch := GetHtmlAttributeMatch(deepAttributeName, captureHtml.Attribute, deepAttributeValue, "")
												if len(deepHtmlAttributeMatch) > 0 {
													for _, match := range deepHtmlAttributeMatch {
														if !CheckForMap(deepAttributeValue) && !CheckForSliceWithMaps(deepAttributeValue) && !CheckForEmptyArmValue(deepAttributeValue) {
															rootAttributesForReturn = append(rootAttributesForReturn, ConvertFlatValueToRootAttribute(deepAttributeValue, match, match.Parent, ""))
														}
													}
												}
												if CheckForSliceWithMaps(deepAttributeValue) {
													for _, innerSlice := range deepAttributeValue.([]interface{}) {
														for evenDeeperAttributeName, evenDeeperAttributeValue := range innerSlice.(map[string]interface{}) {
															evenDeeperHtmlAttributeMatch := GetHtmlAttributeMatch(evenDeeperAttributeName, captureHtml.Attribute, evenDeeperAttributeValue, deepAttributeName)
															if len(evenDeeperHtmlAttributeMatch) > 0 {
																for _, match := range evenDeeperHtmlAttributeMatch {
																	if !CheckForMap(evenDeeperAttributeValue) && !CheckForSliceWithMaps(evenDeeperAttributeValue) && !CheckForEmptyArmValue(evenDeeperAttributeValue) {
																		rootAttributesForReturn = append(rootAttributesForReturn, ConvertFlatValueToRootAttribute(evenDeeperAttributeValue, match, match.Parent, ""))
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}

			if seperateArmResources {
				//By default, the decompiler will aim to merge as many resource types and nested types as possible
				//This switch is not supported yet
				return nil
			} else {
				keyPartName := ConvertArmAttributeName(keyForSubType, "")
				if armResourceType.Parent == "root" || armResourceType.CanBeSeperated {
					if armBasicObject.Special_resource_type != "" {
						resourceFileName = fmt.Sprintf("%s.tf", armBasicObject.Special_resource_type)
					} else {
						for _, resourceType := range armBasicObject.Resource_types {
							if len(strings.Split(resourceType, "/")) == 2 {
								resourceFileName = fmt.Sprintf("%s.tf", ConvertArmToTerraformProvider(resourceType, ""))
							}
						}
					}
				} else {
					resourceFileName = fmt.Sprintf("%s.tf", keyPartName)
				}

				if resourceFileName == ".tf" {
					resourceFileName = ""
				}

				if armResourceType.Parent == "root" {
					for _, innerArmResourceType := range allResourceTypes {
						if strings.Contains(innerArmResourceType.Resource_type, armResourceType.Resource_type) && armResourceType.Resource_type != innerArmResourceType.Resource_type && innerArmResourceType.CanBeSeperated {
							blockObjectsRootResources = append(blockObjectsRootResources, GetBlocksFromRootAttributes(rootAttributesForReturn, htmlObjectCaptureRootArmResources, true)...)
							break
						}
					}

					if len(blockObjectsRootResources) == 0 {
						blockObjectsRootResources = append(blockObjectsRootResources, GetBlocksFromRootAttributes(rootAttributesForReturn, htmlObjectCaptureRootArmResources, false)...)
					}

					if len(blockObjectsRootResources) > 0 {
						compileObjectRootResources := CompileObject{
							ResourceDefinitionName: "azurerm", //Must be changed from static the moment the decompiler supports more than just azurerm
							ArmObject:              armBasicObject,
							Variables:              nil,
							BlockAttributes:        blockObjectsRootResources,
							SeperatedResource:      false,
							FilePath:               resourceFileName,
							ResourceType:           strings.Replace(resourceFileName, ".tf", "", 1),
							ResourceName:           masterKeyRoot,
							IsRoot:                 true,
							HtmlObject:             htmlObjectCaptureRootArmResources,
							ProviderName:           strings.Replace(strings.Replace(strings.Replace(masterKeyRoot, `"`, "", -1), ".", "-", -1), " ", "", -1),
						}
						compiledObjects = append(compiledObjects, compileObjectRootResources)
					}

				} else if !armResourceType.CanBeSeperated {
					for resourceName, rootAttributes := range mapOfRootAttributes {
						blockObjectsSeperatedResources = []BlockAttribute{}
						blockObjectsSeperatedResources = append(blockObjectsSeperatedResources, GetBlocksFromRootAttributes(rootAttributes, htmlObjectCaptureSeperatedResource, false)...)
						if len(blockObjectsSeperatedResources) > 0 {
							compileObjectSeperatedResources := CompileObject{
								ResourceDefinitionName: "azurerm", //Must be changed from static the moment the decompiler supports more than just azurerm
								ArmObject:              armBasicObject,
								Variables:              nil,
								BlockAttributes:        blockObjectsSeperatedResources,
								SeperatedResource:      true,
								FilePath:               resourceFileName,
								ResourceType:           strings.Replace(resourceFileName, ".tf", "", 1),
								ResourceName:           resourceName,
								HtmlObject:             htmlObjectCaptureSeperatedResource,
								ProviderName:           strings.Replace(strings.Replace(strings.Replace(resourceName, `"`, "", -1), ".", "-", -1), " ", "", -1),
							}
							compiledObjects = append(compiledObjects, compileObjectSeperatedResources)
						}
					}
				}
			}
		}

		blockObjectsNestedResources = append(blockObjectsNestedResources, GetBlocksFromRootAttributes(rootAttributesFromNestedResources, htmlObjectCaptureNestedArmResources, false)...)
		for index, compiledObject := range compiledObjects {
			if compiledObject.ResourceName == armBasicObject.Name && len(blockObjectsNestedResources) > 0 {
				compiledObjects[index].BlockAttributes = append(compiledObjects[index].BlockAttributes, blockObjectsNestedResources...)
				break
			}
		}
	}

	for index, compiledObject := range compiledObjects {
		for index2, blockAttribute := range compiledObject.BlockAttributes {
			mapOfRootAttributes := make(map[string]bool)
			uniqueRootAttributes := []RootAttribute{}
			for _, rootAttribute := range blockAttribute.RootAttribute {
				if !mapOfRootAttributes[fmt.Sprintf("%s,%s", rootAttribute.Name, rootAttribute.UniqueBlockName)] && rootAttribute.Name != "" {
					uniqueRootAttributes = append(uniqueRootAttributes, rootAttribute)
					mapOfRootAttributes[fmt.Sprintf("%s,%s", rootAttribute.Name, rootAttribute.UniqueBlockName)] = true
				}
			}
			compiledObjects[index].BlockAttributes[index2].RootAttribute = uniqueRootAttributes
		}
	}

	for index, compiledObject := range compiledObjects {
		if len(compiledObject.BlockAttributes) > 1 {
			match := false
			for _, blockAttribute := range compiledObject.BlockAttributes {
				if blockAttribute.BlockName != "root" && blockAttribute.Parent != "" {
					match = true
					break
				}
			}
			if !match {
				summarizedRootAttributes := []RootAttribute{}
				for _, blockAttribute := range compiledObject.BlockAttributes {
					for _, rootAttribute := range blockAttribute.RootAttribute {
						summarizedRootAttributes = append(summarizedRootAttributes, rootAttribute)
					}
				}
				rootBlockAttributeSlice := []BlockAttribute{}
				blockAttribute := BlockAttribute{
					BlockName:     "root",
					RootAttribute: summarizedRootAttributes,
					Parent:        "",
				}
				rootBlockAttributeSlice = append(rootBlockAttributeSlice, blockAttribute)
				compiledObjects[index].BlockAttributes = rootBlockAttributeSlice
			}
		}
	}

	for index, compiledObject := range compiledObjects {
		//fmt.Println("NEW COMPILED OBJECT", compiledObject.ResourceName, "BLOCKS", compiledObject.BlockAttributes)
		blockAttributesForRoot := []BlockAttribute{}
		blockAttributesForRest := []BlockAttribute{}
		for index2, blockAttribute := range compiledObject.BlockAttributes {
			if blockAttribute.Parent == "" {
				compiledObjects[index].BlockAttributes[index2].Parent = "root"
			}
			blockNameFixed := ConvertArmAttributeName(blockAttribute.BlockName, "")
			if blockNameFixed == strings.Replace(compiledObject.FilePath, ".tf", "", 1) || blockAttribute.BlockName == "root" {
				blockAttributesForRoot = append(blockAttributesForRoot, blockAttribute)
			} else if blockAttribute.BlockName != "root" && blockAttribute.BlockName != "" {
				blockAttributesForRest = append(blockAttributesForRest, blockAttribute)
			}
		}
		blockAttributesForRoot = append(blockAttributesForRoot, blockAttributesForRest...)
		compiledObjects[index].BlockAttributes = blockAttributesForRoot
	}

	for index, compiledObject := range compiledObjects {
		rootAttributesForRootBlock := []RootAttribute{}
		blockAttributesNoneRoot := []BlockAttribute{}
		if !compiledObject.SeperatedResource {
			for _, blockAttribute := range compiledObject.BlockAttributes {
				if blockAttribute.Parent == "root" && blockAttribute.BlockName == "root" {
					rootAttributesForRootBlock = append(rootAttributesForRootBlock, blockAttribute.RootAttribute...)
				} else {
					blockAttributesNoneRoot = append(blockAttributesNoneRoot, blockAttribute)
				}
			}
			blockRootAttribute := BlockAttribute{
				BlockName:     "root",
				RootAttribute: rootAttributesForRootBlock,
				Parent:        "root",
			}
			blockAttributesNoneRoot = append(blockAttributesNoneRoot, blockRootAttribute)
			compiledObjects[index].BlockAttributes = blockAttributesNoneRoot
		}
	}

	for index, compiledObject := range compiledObjects {
		blockAttributesNotEmpty := []BlockAttribute{}
		for index2, blockAttribute := range compiledObject.BlockAttributes {
			if len(blockAttribute.RootAttribute) > 0 {
				if blockAttribute.BlockName == "root" && blockAttribute.Parent == "" {
					compiledObject.BlockAttributes[index2].Parent = "root"
				}
				blockAttributesNotEmpty = append(blockAttributesNotEmpty, compiledObject.BlockAttributes[index2])
			}
		}
		compiledObjects[index].BlockAttributes = blockAttributesNotEmpty
	}
	/*
		fmt.Println("\n---------------------RESOURCE NAME ----------------------------")
		for _, compiledObject := range compiledObjects {
			if compiledObject.ResourceType == "linux_web_app" && compiledObject.ProviderName == "ruc-we-velkommen-test-api-ws01" {
				fmt.Println("\nCOMPILED OBJECT --------------", compiledObject.ResourceName)
				for _, block := range compiledObject.BlockAttributes {
					fmt.Println(fmt.Sprintf("\n-------------BLOCK----------------%s-%s", block.BlockName, block.Parent))
					for index2, rootAttribute := range block.RootAttribute {
						fmt.Println(index2, "NAME", rootAttribute.Name, "BLOCK", rootAttribute.BlockName, "IS", rootAttribute.IsBlock, rootAttribute.Value)
					}
				}
			}
		}
	*/
	//fmt.Println("LEN OF COMPILED OBJECTS:", len(compiledObjects))

	return compiledObjects
}

func FindTypeByString(value interface{}) string {
	reflectValue := reflect.ValueOf(value)

	switch reflectValue.Kind() {
	case reflect.String:
		return "string"
	case reflect.Slice:
		return "slice"
	case reflect.Bool:
		return "bool"
	case reflect.Int:
		return "int" // Using int as the base type for int's
	case reflect.Float64:
		// Check if it's effectively an integer
		if reflectValue.Float() == float64(int(reflectValue.Float())) {
			return "int" // It's an integer-like float
		}
		return "float" // Otherwise, treat as float64
	}

	return ""
}

func GetInnerRootAttributes(armProperties interface{}, htmlObject HtmlObject, masterKey string, blockName string) []RootAttribute {
	var returnRootAttributes []RootAttribute

	for attributeName, attributeValue := range armProperties.(map[string]interface{}) {
		if CheckForMap(attributeValue) {
			for innerAttributeName, innerAttributeValue := range attributeValue.(map[string]interface{}) {
				htmlAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObject.Attribute, innerAttributeValue, attributeName)
				if len(htmlAttributeMatch) > 0 && !(CheckForEmptyArmValue(innerAttributeValue)) {
					for _, match := range htmlAttributeMatch {
						returnRootAttributes = append(returnRootAttributes, ConvertFlatValueToRootAttribute(innerAttributeValue, match, blockName, masterKey))
					}
				}

				if CheckForSliceWithMaps(innerAttributeValue) {
					for _, slice := range innerAttributeValue.([]interface{}) {
						if CheckForMap(slice) {
							for minimumAttributeName, minimumAttributeValue := range slice.(map[string]interface{}) {
								minimumHtmlAttributeMatch := GetHtmlAttributeMatch(minimumAttributeName, htmlObject.Attribute, minimumAttributeValue, innerAttributeName)
								if len(minimumHtmlAttributeMatch) > 0 && !(CheckForEmptyArmValue(minimumAttributeValue)) {
									for _, match := range minimumHtmlAttributeMatch {
										returnRootAttributes = append(returnRootAttributes, ConvertFlatValueToRootAttribute(minimumAttributeValue, match, match.Parent, masterKey))
									}
								}
								if CheckForSliceWithMaps(minimumAttributeValue) {

								} else if CheckForMap(minimumAttributeValue) {
									for deepAttributeName, deepAttributeValue := range minimumAttributeValue.(map[string]interface{}) {
										deepHtmlAttributeMatch := GetHtmlAttributeMatch(deepAttributeName, htmlObject.Attribute, deepAttributeValue, minimumAttributeName)
										if len(deepHtmlAttributeMatch) > 0 && !(CheckForEmptyArmValue(deepAttributeValue)) {
											for _, match := range deepHtmlAttributeMatch {
												returnRootAttributes = append(returnRootAttributes, ConvertFlatValueToRootAttribute(deepAttributeValue, match, match.Parent, masterKey))
											}
										}
									}
								}
							}
						}
					}
				}
			}
		} else if CheckForSliceWithMaps(attributeValue) {
			for _, slice := range attributeValue.([]interface{}) {
				if slice.(map[string]interface{})["properties"] != nil {
					masterKey := GetArmMasterKey(slice)
					for innerAttributeName, innerAttributeValue := range slice.(map[string]interface{})["properties"].(map[string]interface{}) {
						htmlAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObject.Attribute, innerAttributeValue, attributeName)
						if len(htmlAttributeMatch) > 0 && !CheckForEmptyArmValue(innerAttributeValue) {
							for _, match := range htmlAttributeMatch {
								returnRootAttributes = append(returnRootAttributes, ConvertFlatValueToRootAttribute(innerAttributeValue, match, match.Parent, masterKey))
							}
						}
						if CheckForSliceWithMaps(innerAttributeValue) {
							for _, slice := range innerAttributeValue.([]interface{}) {
								if CheckForMap(slice) {
									for minimumAttributeName, minimumAttributeValue := range slice.(map[string]interface{}) {

										htmlAttributeMatch := GetHtmlAttributeMatch(minimumAttributeName, htmlObject.Attribute, nil, innerAttributeName)
										if len(htmlAttributeMatch) > 0 && !CheckForEmptyArmValue(minimumAttributeValue) {
											for _, match := range htmlAttributeMatch {
												returnRootAttributes = append(returnRootAttributes, ConvertFlatValueToRootAttribute(minimumAttributeValue, match, match.Parent, masterKey))
											}
										}
									}
								}
							}
						}
						if CheckForMap(innerAttributeValue) {
							masterKey := GetArmMasterKey(innerAttributeValue)
							for minimumAttributeName, minimumAttributeValue := range innerAttributeValue.(map[string]interface{}) {
								htmlAttributeMatch := GetHtmlAttributeMatch(minimumAttributeName, htmlObject.Attribute, minimumAttributeValue, innerAttributeName)
								if len(htmlAttributeMatch) > 0 && !CheckForEmptyArmValue(minimumAttributeValue) {
									for _, match := range htmlAttributeMatch {
										returnRootAttributes = append(returnRootAttributes, ConvertFlatValueToRootAttribute(minimumAttributeValue, match, match.Parent, masterKey))
									}
								}
							}
						}
					}
				} else {
					for innerAttributeName, innerAttributeValue := range slice.(map[string]interface{}) {
						innerHtmlMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObject.Attribute, innerAttributeValue, innerAttributeName)
						if len(innerHtmlMatch) > 0 && !CheckForEmptyArmValue(innerAttributeValue) {
							for _, match := range innerHtmlMatch {
								returnRootAttributes = append(returnRootAttributes, ConvertFlatValueToRootAttribute(innerAttributeValue, match, "", ""))
							}
						}
					}
				}
			}
		} else {
			htmlAttributeMatchRoot := GetHtmlAttributeMatch(attributeName, htmlObject.Attribute, nil, "")
			if len(htmlAttributeMatchRoot) > 0 && !CheckForEmptyArmValue(attributeValue) {
				for _, match := range htmlAttributeMatchRoot {
					returnRootAttributes = append(returnRootAttributes, ConvertFlatValueToRootAttribute(attributeValue, match, match.Parent, masterKey))
				}
			}
		}
	}

	return returnRootAttributes
}

func CheckForEmptyArmValue(armPropertyValue interface{}) bool {
	// If the value is nil, it's considered empty
	if armPropertyValue == nil {
		return true
	}

	// Use reflection to check the underlying type and value
	reflectValue := reflect.ValueOf(armPropertyValue)

	switch reflectValue.Kind() {
	case reflect.String:
		// Empty string check
		return reflectValue.Len() == 0
	case reflect.Slice, reflect.Map, reflect.Array:
		// Empty slice, map, or array check
		return reflectValue.Len() == 0
	case reflect.Ptr, reflect.Interface:
		// If it's a pointer or interface, check if it's nil or if its underlying value is empty
		return reflectValue.IsNil() || CheckForEmptyArmValue(reflectValue.Elem().Interface())
	default:
		// For other types (like ints, structs, etc.), consider them not empty
		// Feel free to add more checks here for other types like struct zero values, etc.
		return false
	}
}

func GetSubArmPartOfProperties(armPropertyValue interface{}, resourceType string) []map[string]interface{} {
	var armPropertyNameConvert string
	var innerArmPropertyNameConvert string
	var returnArmPropertyValue []map[string]interface{}
	armPropertyNameConvertSlice := strings.Split(ConvertArmAttributeName(resourceType, nil), "/")
	if len(armPropertyNameConvertSlice) == 3 {
		armPropertyNameConvert = armPropertyNameConvertSlice[2]
	}

	for attributeName, attributeValue := range armPropertyValue.(map[string]interface{}) {
		innerArmPropertyNameConvertSlice := strings.Split(ConvertArmAttributeName(attributeName, nil), "/")
		if len(innerArmPropertyNameConvertSlice) == 3 {
			innerArmPropertyNameConvert = innerArmPropertyNameConvertSlice[2]
		}
		if strings.TrimSuffix(innerArmPropertyNameConvert, "s") == armPropertyNameConvert {
			switch attributeValue.(type) {
			case []interface{}:
				{
					for _, slice := range attributeValue.([]interface{}) {
						returnArmPropertyValue = append(returnArmPropertyValue, slice.(map[string]interface{})["properties"].(map[string]interface{}))
					}
				}

			case map[string]interface{}:
				{
					returnArmPropertyValue = append(returnArmPropertyValue, attributeValue.(map[string]interface{})["properties"].(map[string]interface{}))
				}
			}
		}
	}
	return returnArmPropertyValue
}

func GetBlocksFromRootAttributes(rootAttributes []RootAttribute, htmlObject HtmlObject, isSeperated bool) []BlockAttribute {
	var blocksForReturn []BlockAttribute
	var summarizedRootAttributes []RootAttribute
	var removeDuplicateValues []string
	var furtherSummarizeRootAttributes []RootAttribute
	var removeShadowRootAttributes []RootAttribute

	for index, rootAttribute := range rootAttributes {
		if isSeperated {
			rootAttributes[index].BlockName = "root"
		} else if rootAttribute.Name != "name" && rootAttribute.Name != "id" {
			for _, attribute := range htmlObject.Attribute {
				if rootAttribute.Name == attribute.Name {
					rootAttributes[index].BlockName = attribute.Parent
				}
			}
		}
	}

	for index, rootAttribute := range rootAttributes {
		if !strings.Contains(rootAttribute.Name, "name") && !strings.Contains(rootAttribute.Name, "id") {
			matches := []Attribute{}
			for _, htmlObject := range htmlObjects {
				for _, attribute := range htmlObject.Attribute {
					if rootAttribute.Name == attribute.Name {
						matches = append(matches, attribute)
					}
				}
			}
			if len(matches) > 1 {
				maxRootAttributeMatchLength := 0
				correctParent := ""
				for _, attribute := range matches {
					rootAttributesToMatch := []RootAttribute{}
					for _, innerRootAttribute := range rootAttributes {
						if innerRootAttribute.BlockName == attribute.Parent {
							rootAttributesToMatch = append(rootAttributesToMatch, innerRootAttribute)
						}
					}
					if len(rootAttributesToMatch) > maxRootAttributeMatchLength {
						correctParent = attribute.Parent
					}
				}
				rootAttributes[index].BlockName = correctParent
			}
		}
	}

	patternToMatchAddress := `^address_p(?:refix(?:es)?|aces)?$`
	regexPatternStructAddress := regexp.MustCompile(patternToMatchAddress)
	patternToMatchProfile := `(true|false|enabled|disabled|policies|(\d+)|(\d{1,3}(\.\d{1,3}){3}(\/\d{1,2})?))$`
	regexPatternStructProfile := regexp.MustCompile(patternToMatchProfile)

	//Sanatize data
	for index, rootAttribute := range rootAttributes {
		if !rootAttribute.IsBlock {
			if regexPatternStructAddress.MatchString(rootAttribute.Name) {
				if rootAttribute.BlockName == "" || rootAttribute.BlockName == "root" {
					if strings.Contains(rootAttribute.Name, "prefixes") {
						rootAttributes[index].Name = strings.Replace(rootAttribute.Name, "prefixes", "space", 1)
					} else if strings.Contains(rootAttribute.Name, "prefix") {
						rootAttributes[index].Name = strings.Replace(rootAttribute.Name, "prefix", "space", 1)
					}
				}
				switch rootAttribute.Value.(type) {
				case string:
					{
						var newRootValue []interface{}
						newRootValue = append(newRootValue, rootAttribute.Value)
						rootAttributes[index].Value = newRootValue
					}
				}
			}
			if checkForMap, ok := rootAttribute.Value.(map[string]interface{}); ok {
				lastPartOfAttributeName := ""
				lastPartOfAttributeNameSlice := strings.Split(rootAttribute.Name, "_")
				if len(lastPartOfAttributeNameSlice) > 1 {
					lastPartOfAttributeName = lastPartOfAttributeNameSlice[len(lastPartOfAttributeNameSlice)-1]
				} else {
					lastPartOfAttributeName = lastPartOfAttributeNameSlice[0]
				}
				for attributeName, attributeValue := range checkForMap {
					if strings.HasSuffix(strings.ToLower(attributeName), lastPartOfAttributeName) {
						rootAttributes[index].Value = attributeValue
					}
				}
			}
		}
	}
	//Summarize root attributes whereever possible
	for _, rootAttribute := range rootAttributes {
		var fromSliceValues []string
		var fromFlatValues []string
		var persistRootAttribute RootAttribute
		match := false
		if !rootAttribute.IsBlock && rootAttribute.UniqueBlockName != "" {
			{
				for _, innerRootAttribute := range rootAttributes {
					if rootAttribute.BlockName == innerRootAttribute.BlockName && rootAttribute.UniqueBlockName == innerRootAttribute.UniqueBlockName && !regexPatternStructAddress.MatchString(innerRootAttribute.Name) && !regexPatternStructAddress.MatchString(rootAttribute.Name) && !regexPatternStructProfile.MatchString(innerRootAttribute.Name) && !regexPatternStructProfile.MatchString(rootAttribute.Name) && innerRootAttribute.Name != "name" && rootAttribute.Name != "name" {
						switch innerRootAttribute.Value.(type) {
						case []interface{}:
							{
								for _, value := range innerRootAttribute.Value.([]interface{}) {
									if !CheckForMap(value) {
										fromSliceValues = append(fromSliceValues, value.(string), rootAttribute.UniqueBlockName, rootAttribute.Name)
									}
								}
							}
						case interface{}:
							{
								if _, ok := innerRootAttribute.Value.(bool); !ok {
									if _, mapOk := innerRootAttribute.Value.(map[string]interface{}); !mapOk {
										if !regexPatternStructAddress.MatchString(innerRootAttribute.Name) && !regexPatternStructProfile.MatchString(innerRootAttribute.Value.(string)) && !strings.Contains(innerRootAttribute.Name, "id") && strings.HasSuffix(innerRootAttribute.Name, "s") {
											fromFlatValues = append(fromFlatValues, innerRootAttribute.Value.(string), rootAttribute.UniqueBlockName, rootAttribute.Name)
										}
									}
								}
							}
						}
						persistRootAttribute = rootAttribute
						match = true
					}
				}
			}
		}

		if match {
			matchInner := false
			result := []string{} //Remove tainted elements AFTER its checked whether its a duplicate or not
			valueForBlock := append(fromSliceValues, fromFlatValues...)
			if len(removeDuplicateValues) == len(valueForBlock) {
				for index := range removeDuplicateValues {
					if removeDuplicateValues[index] != valueForBlock[index] {
						matchInner = true
					}
				}
			} else if len(removeDuplicateValues) == 0 {
				matchInner = true
			}
			if matchInner {

				//Removing the tained values from valueForBlock and placing it in result
				if len(valueForBlock) > 0 {
					result = append(result, valueForBlock[0])
				}

				for i := 3; i < len(valueForBlock); i += 3 {
					result = append(result, valueForBlock[i])
				}

				rootAttribute := RootAttribute{
					Name:            persistRootAttribute.Name,
					Value:           result,
					IsBlock:         false,
					BlockName:       persistRootAttribute.BlockName,
					UniqueBlockName: persistRootAttribute.UniqueBlockName,
				}
				summarizedRootAttributes = append(summarizedRootAttributes, rootAttribute)
				removeDuplicateValues = append(fromSliceValues, fromFlatValues...)
			}
		}
	}

	for index, rootAttribute := range summarizedRootAttributes {
		if !CheckForMap(rootAttribute.Value) && !CheckForSliceWithMaps(rootAttribute.Value) && !CheckForEmptyArmValue(rootAttribute.Value) && !CheckForSlice(rootAttribute.Value) {
			furtherSummarizeRootAttributes = append(furtherSummarizeRootAttributes, rootAttribute)
		}

		if CheckForSlice(rootAttribute.Value) {
			var summarizedValues []interface{}
			mapOfValues := make(map[interface{}]bool)
			var loopValue []interface{}
			switch value := rootAttribute.Value.(type) {
			case []interface{}:
				{
					loopValue = value
				}
			default:
				// Convert []string to []interface{}
				loopValue = make([]interface{}, len(value.([]string)))
				for i, str := range value.([]string) {
					loopValue[i] = str
				}
			}
			for _, value := range loopValue {
				if CheckForString(value) && len(loopValue) > 1 {
					if !mapOfValues[value] {
						mapOfValues[value] = true
						summarizedValues = append(summarizedValues, value)
					}
				} else {
					summarizedValues = append(summarizedValues, value)
				}
			}
			summarizedRootAttributes[index].Value = summarizedValues
			furtherSummarizeRootAttributes = append(furtherSummarizeRootAttributes, summarizedRootAttributes[index])
		}
	}

	rootAttributes = append(rootAttributes, furtherSummarizeRootAttributes...)

	for _, rootAttribute := range rootAttributes {
		if CheckForString(rootAttribute.Value) {
			if strings.TrimSpace(rootAttribute.UniqueBlockName) != strings.TrimSpace(rootAttribute.Value.(string)) {
				removeShadowRootAttributes = append(removeShadowRootAttributes, rootAttribute)
			}
		} else {
			if !CheckForMap(rootAttribute.Value) && !CheckForSliceWithMaps(rootAttribute.Value) {
				removeShadowRootAttributes = append(removeShadowRootAttributes, rootAttribute)
			}
		}
	}

	mapOfRootAttributes := make(map[string]bool)
	almostSummarizeRootAttributes := []RootAttribute{}
	for _, rootAttribute := range removeShadowRootAttributes {
		if !mapOfRootAttributes[fmt.Sprintf("%s/%s/%s", rootAttribute.Name, rootAttribute.UniqueBlockName, rootAttribute.BlockName)] && rootAttribute.Name != "" {
			almostSummarizeRootAttributes = append(almostSummarizeRootAttributes, rootAttribute)
			mapOfRootAttributes[fmt.Sprintf("%s/%s/%s", rootAttribute.Name, rootAttribute.UniqueBlockName, rootAttribute.BlockName)] = true
		}
	}

	//Transform strings to slices in case the HTML attribute ís an actual list
	for index, rootAttribute := range almostSummarizeRootAttributes {
		switch rootAttribute.Value.(type) {
		case string:
			{
				for _, attribute := range htmlObject.Attribute {
					if rootAttribute.Name == attribute.Name && attribute.Type == "list" {
						var convertStringToSlice []interface{}
						convertStringToSlice = append(convertStringToSlice, rootAttribute.Value.(string))
						almostSummarizeRootAttributes[index].Value = convertStringToSlice
					} else if rootAttribute.Name == attribute.Name && attribute.Type == "bool" {
						if strings.Contains(strings.ToLower(rootAttribute.Name), "enable") {
							almostSummarizeRootAttributes[index].Value = true
						} else {
							almostSummarizeRootAttributes[index].Value = false
						}
					}
				}
			}
		}
	}

	attributeForSubResourceBlock := Attribute{}
	for index, rootAttribute := range almostSummarizeRootAttributes {
		for _, attribute := range htmlObject.Attribute {
			if rootAttribute.BlockName == attribute.Name && attribute.Parent == "root" {
				almostSummarizeRootAttributes[index].Descriptor = attribute.Descriptor //Inject Descriptor so that we dont need to use the HTML object attributes when we create terraform - This MUST be changed in later verisons (Optimize each struct type)
				attributeForSubResourceBlock = attribute
				break
			}
		}
		if attributeForSubResourceBlock != (Attribute{}) {
			break
		}
	}

	//Add the nested object root attribute (The below has no effect on root attribute types without a unique value, hence all these block attributes are never created)
	uniqueBlockNames := []string{}
	mapOfUniqueBlockNames := make(map[string]bool)
	for _, rootAttribute := range almostSummarizeRootAttributes {
		if !mapOfUniqueBlockNames[rootAttribute.UniqueBlockName] && rootAttribute.UniqueBlockName != "" {
			uniqueBlockNames = append(uniqueBlockNames, rootAttribute.UniqueBlockName)
			mapOfUniqueBlockNames[rootAttribute.UniqueBlockName] = true
		}
	}

	for index := range len(uniqueBlockNames) {
		rootAttribute := RootAttribute{
			Name:            attributeForSubResourceBlock.Name,
			Value:           nil,
			IsBlock:         true,
			BlockName:       "root",
			UniqueBlockName: uniqueBlockNames[index],
		}
		almostSummarizeRootAttributes = append(almostSummarizeRootAttributes, rootAttribute)
	}

	mapOfUniqueBlockNames = make(map[string]bool)
	for _, rootAttribute := range almostSummarizeRootAttributes {
		if rootAttribute.UniqueBlockName == "" && !mapOfUniqueBlockNames[rootAttribute.BlockName] {
			mapOfUniqueBlockNames[rootAttribute.BlockName] = true
		}
	}

	for name, _ := range mapOfUniqueBlockNames {
		rootAttribute := RootAttribute{
			Name:            name,
			Value:           nil,
			IsBlock:         true,
			BlockName:       "root",
			UniqueBlockName: "",
		}
		almostSummarizeRootAttributes = append(almostSummarizeRootAttributes, rootAttribute)
	}

	for index, rootAttribute := range almostSummarizeRootAttributes {
		for _, attribute := range htmlObject.Attribute {
			if rootAttribute.Name == attribute.Name && rootAttribute.Name != "name" && rootAttribute.Name != "id" {
				almostSummarizeRootAttributes[index].BlockName = attribute.Parent
			}
		}
	}

	//rootAttributesForBlockAttributes := []RootAttribute{}

	for index, rootAttribute := range almostSummarizeRootAttributes {
		//match := false
		newParentName := ""
		mapOfRootAttributes := make(map[string]bool)
		capturesimilarRootAttributesOldParent := []RootAttribute{}
		capturesimilarRootAttributesNewParent := []RootAttribute{}
		for _, attribute := range htmlObject.Attribute {
			if rootAttribute.Name == attribute.Name && attribute.ShadowCopy.Parent != attribute.Parent && attribute.ShadowCopy.Parent != "" {
				for _, innerRootAttribute := range almostSummarizeRootAttributes {
					if innerRootAttribute.BlockName == attribute.Parent && !mapOfRootAttributes[fmt.Sprintf("%s,%s", innerRootAttribute.Name, innerRootAttribute.UniqueBlockName)] && innerRootAttribute.BlockName != "root" {
						capturesimilarRootAttributesOldParent = append(capturesimilarRootAttributesOldParent, innerRootAttribute)
						mapOfRootAttributes[fmt.Sprintf("%s,%s", innerRootAttribute.Name, innerRootAttribute.UniqueBlockName)] = true
					} else if innerRootAttribute.BlockName == attribute.ShadowCopy.Parent && !mapOfRootAttributes[fmt.Sprintf("%s,%s", innerRootAttribute.Name, innerRootAttribute.UniqueBlockName)] && innerRootAttribute.BlockName != "root" {
						newParentName = attribute.ShadowCopy.Parent
						capturesimilarRootAttributesNewParent = append(capturesimilarRootAttributesNewParent, innerRootAttribute)
						mapOfRootAttributes[fmt.Sprintf("%s,%s", innerRootAttribute.Name, innerRootAttribute.UniqueBlockName)] = true
					}
				}
				if newParentName != "root" && newParentName != "" && len(capturesimilarRootAttributesNewParent) > len(capturesimilarRootAttributesOldParent) {
					almostSummarizeRootAttributes[index].BlockName = newParentName
				}
			}
		}
	}

	blockRootAttributes := []RootAttribute{}
	for _, rootAttribute := range almostSummarizeRootAttributes {
		if rootAttribute.IsBlock && rootAttribute.UniqueBlockName != "" {
			blockRootAttributes = append(blockRootAttributes, rootAttribute)
		}
	}

	mapOfUniqueMissingBlockRootAttributes := make(map[string]bool)
	for _, rootAttribute := range almostSummarizeRootAttributes {
		if rootAttribute.UniqueBlockName != "" {
			checkForBlock := []RootAttribute{}
			for _, blockRootAttribute := range blockRootAttributes {
				if rootAttribute.BlockName == blockRootAttribute.Name && rootAttribute.UniqueBlockName == blockRootAttribute.UniqueBlockName && rootAttribute.Name != blockRootAttribute.Name {
					checkForBlock = append(checkForBlock, blockRootAttribute)
				}
			}
			if len(checkForBlock) == 0 {
				if !mapOfUniqueMissingBlockRootAttributes[rootAttribute.UniqueBlockName] && rootAttribute.BlockName != "root" {
					blockName := ""
					for _, blockAttribute := range blockRootAttributes {
						if strings.Contains(rootAttribute.BlockName, blockAttribute.Name) && rootAttribute.BlockName != blockAttribute.Name {
							blockName = blockAttribute.Name
						}
					}
					rootAttribute := RootAttribute{
						Name:            rootAttribute.BlockName,
						Value:           nil,
						BlockName:       blockName,
						IsBlock:         true,
						UniqueBlockName: rootAttribute.UniqueBlockName,
					}
					almostSummarizeRootAttributes = append(almostSummarizeRootAttributes, rootAttribute)
					mapOfUniqueMissingBlockRootAttributes[rootAttribute.UniqueBlockName] = true
				}
			}
		}
	}

	blockRootAttributes = []RootAttribute{}
	//rootName := ""
	/*
		for _, rootAttribute := range almostSummarizeRootAttributes {
			if rootAttribute.Name == "name" && !rootAttribute.IsBlock && rootAttribute.BlockName == "root" {
				if CheckForEmptyArmValue(rootAttribute.Value) {
					rootName =
				}
				rootName = rootAttribute.Value.(string)
			}
		}
	*/
	/*
		for index, rootAttribute := range almostSummarizeRootAttributes {
			if !rootAttribute.IsBlock && rootAttribute.BlockName != "root" && rootAttribute.Name == "name" {

			}
		}
	*/
	for _, rootAttribute := range almostSummarizeRootAttributes {
		if rootAttribute.IsBlock {
			blockRootAttributes = append(blockRootAttributes, rootAttribute)
		}
	}

	rootAttributesForBlockSlice := [][]RootAttribute{}
	for _, rootAttribute := range blockRootAttributes {
		rootAttributesForBlock := []RootAttribute{}
		for _, innerRootAttribute := range almostSummarizeRootAttributes {
			if rootAttribute.Name != innerRootAttribute.Name && rootAttribute.Name == innerRootAttribute.BlockName && rootAttribute.UniqueBlockName == innerRootAttribute.UniqueBlockName {
				rootAttributesForBlock = append(rootAttributesForBlock, innerRootAttribute)
			}
		}
		rootAttributesForBlockSlice = append(rootAttributesForBlockSlice, rootAttributesForBlock)
	}

	for _, rootAttribute := range blockRootAttributes {
		rootAttributesForBlock := []RootAttribute{}
		for _, capturedRootAttributeGroup := range rootAttributesForBlockSlice {
			for _, innerRootAtttribute := range capturedRootAttributeGroup {
				if innerRootAtttribute.BlockName == rootAttribute.Name && rootAttribute.UniqueBlockName == innerRootAtttribute.UniqueBlockName {
					rootAttributesForBlock = append(rootAttributesForBlock, innerRootAtttribute)
				}
			}
		}
		blockAttribute := BlockAttribute{
			BlockName:     rootAttribute.Name,
			RootAttribute: rootAttributesForBlock,
			Parent:        rootAttribute.BlockName,
		}
		blocksForReturn = append(blocksForReturn, blockAttribute)
	}

	rootAttributesForRootBlock := []RootAttribute{}
	for _, rootAttribute := range almostSummarizeRootAttributes {
		if rootAttribute.BlockName == "root" {
			rootAttributesForRootBlock = append(rootAttributesForRootBlock, rootAttribute)
		}
	}

	blockAttribute := BlockAttribute{
		BlockName:     "root",
		RootAttribute: rootAttributesForRootBlock,
		Parent:        "root",
	}

	//Inject unique block name from root attributes of each block
	for index, blockAttribute := range blocksForReturn {
		uniqueBlockName := ""
		for _, rootAttribute := range blockAttribute.RootAttribute {
			if rootAttribute.UniqueBlockName != "" {
				uniqueBlockName = rootAttribute.UniqueBlockName
			}
		}
		blocksForReturn[index].UniqueBlockName = uniqueBlockName
	}

	blocksForReturn = append(blocksForReturn, blockAttribute)

	//Remove any root attribute of which has the name "resource_group_name"
	for index, blockAttribute := range blocksForReturn {
		rootAttributes := []RootAttribute{}
		for _, rootAttribute := range blockAttribute.RootAttribute {
			if rootAttribute.Name != "resource_group_name" {
				rootAttributes = append(rootAttributes, rootAttribute)
			}
		}

		blocksForReturn[index].RootAttribute = rootAttributes
	}

	for index, blockAttriubte := range blocksForReturn {
		if blockAttriubte.BlockName == "identity" {
			match := false
			for _, rootAttribute := range blockAttribute.RootAttribute {
				if rootAttribute.Name == "type" && rootAttribute.Value == "SystemAssigned" {
					match = true
				} else if rootAttribute.Name == "identity_ids" && CheckForSlice(rootAttribute.Value) {
					if len(rootAttribute.Value.([]interface{})) > 0 {
						match = true
					}
				}
			}
			if !match {
				blocksForReturn[index].RootAttribute = []RootAttribute{}
			}
		} else {
			for _, attribute := range htmlObject.Attribute {
				if blockAttriubte.BlockName == attribute.Name && attribute.ShadowCopy.Parent != "" {
					rootAttributesFromOldBlock := []RootAttribute{}
					rootAttributesFromNewBlock := []RootAttribute{}
					for _, innerBlockAttribute := range blocksForReturn {
						if innerBlockAttribute.BlockName == blockAttribute.BlockName {
							rootAttributesFromOldBlock = append(rootAttributesFromOldBlock, innerBlockAttribute.RootAttribute...)
						} else if innerBlockAttribute.BlockName == attribute.ShadowCopy.Parent {
							rootAttributesFromNewBlock = append(rootAttributesFromNewBlock, innerBlockAttribute.RootAttribute...)
						}
					}
					if len(rootAttributesFromOldBlock) > len(rootAttributesFromNewBlock) {
						blocksForReturn[index].Parent = attribute.ShadowCopy.Parent
					}
				}
			}
		}
	}

	mapOfRootBlocks := make(map[string]bool)
	blockAttributesForReturn := []BlockAttribute{}

	for _, blockAttribute := range blocksForReturn {
		if len(blockAttribute.RootAttribute) > 0 {
			rootAttributesCleaned := []RootAttribute{}

			for _, rootAttribute := range blockAttribute.RootAttribute {
				if rootAttribute.Name != "root" && rootAttribute.Name != "" {
					rootAttributesCleaned = append(rootAttributesCleaned, rootAttribute)
				}
			}

			newBlockAttribute := blockAttribute
			newBlockAttribute.RootAttribute = rootAttributesCleaned

			if blockAttribute.BlockName == "root" && blockAttribute.Parent == "root" {
				if !mapOfRootBlocks[fmt.Sprintf("%s,%s", blockAttribute.BlockName, blockAttribute.Parent)] {
					mapOfRootBlocks[fmt.Sprintf("%s,%s", blockAttribute.BlockName, blockAttribute.Parent)] = true
					blockAttributesForReturn = append(blockAttributesForReturn, newBlockAttribute)
				}
			} else {
				blockAttributesForReturn = append(blockAttributesForReturn, newBlockAttribute)
			}
		}
	}

	mapOfValueCount := make(map[string]int)
	for _, blockAttribute := range blockAttributesForReturn {
		for _, rootAttribute := range blockAttribute.RootAttribute {
			if CheckForString(rootAttribute.Value) && rootAttribute.Name != "name" && rootAttribute.Name != "id" && CheckForEmptyArmValue(rootAttribute.UniqueBlockName) {
				mapOfValueCount[rootAttribute.Value.(string)]++
			}
		}
	}
	rootAttributeNames := []string{}
	for value, _ := range mapOfValueCount {
		if mapOfValueCount[value] > 1 && mapOfValueCount[value] <= 3 {
			for _, blockAttribute := range blockAttributesForReturn {
				for _, rootAttribute := range blockAttribute.RootAttribute {
					if rootAttribute.Value == value {
						rootAttributeNames = append(rootAttributeNames, rootAttribute.Name)
					}
				}
			}
		}
	}

	poisonedRootAttributeNames := []string{}
	for _, name := range rootAttributeNames {
		nameSlice := strings.Split(name, "_")
		captureNames := []string{}
		if len(nameSlice) == 1 {
			for _, innerName := range rootAttributeNames {
				if strings.Contains(innerName, name) && name != innerName {
					captureNames = append(captureNames, innerName)
				}
			}
			for _, capturedName := range captureNames {
				capturedNameSlice := strings.Split(capturedName, "_")
				if len(capturedNameSlice) > 1 {
					if nameSlice[0] == capturedNameSlice[0] { //Poison found
						poisonedRootAttributeNames = append(poisonedRootAttributeNames, capturedName)
					}
				}
			}
		}
	}

	blocksWithPoisonRemoved := []BlockAttribute{}
	for _, blockAttribute := range blockAttributesForReturn {
		match := false
		for _, rootAttribute := range blockAttribute.RootAttribute {
			for _, poisonedName := range poisonedRootAttributeNames {
				if rootAttribute.Name == poisonedName {
					match = true
				}
			}
		}
		if !match {
			blocksWithPoisonRemoved = append(blocksWithPoisonRemoved, blockAttribute)
		}
	}

	for index, blockAttribute := range blocksWithPoisonRemoved {
		if blockAttribute.BlockName == "root" && len(blocksWithPoisonRemoved) > 1 {
			rootAttributesToKeep := []RootAttribute{}
			for _, rootAttribute := range blockAttribute.RootAttribute {
				match := false
				for _, innerBlockAttribute := range blocksWithPoisonRemoved {
					if rootAttribute.Name == innerBlockAttribute.BlockName {
						match = true
						break
					}
				}
				if match || !rootAttribute.IsBlock {
					rootAttributesToKeep = append(rootAttributesToKeep, rootAttribute)
				}
			}

			blocksWithPoisonRemoved[index].RootAttribute = rootAttributesToKeep
		}
	}
	/*
		mapOfRootBlockAttributes := make(map[string]bool)
		nameOfRootBlocks := []string{}
		for _, block := range blocksWithPoisonRemoved {
			if block.BlockName == "root" && block.Parent == "root" {
				for _, rootAttribute := range block.RootAttribute {
					if rootAttribute.IsBlock {
						nameOfRootBlocks = append(nameOfRootBlocks, rootAttribute.Name)
					}
				}
			}
		}

		for _, blockName := range nameOfRootBlocks {
			for _, block := range blocksWithPoisonRemoved {
				if blockName == block.BlockName {
					mapOfRootBlockAttributes[block.BlockName] = true
					break
				}
			}
			if !mapOfRootBlockAttributes[blockName] {
				mapOfRootBlockAttributes[blockName] = true
				fmt.Println("WE ARE MISSING THE FOLLOING BLOCK:", blockName)
			}
		}
	*/

	for index, block := range blocksWithPoisonRemoved {
		for index2, rootAttribute := range block.RootAttribute {
			if CheckForEmptyArmValue(rootAttribute.Value) && !rootAttribute.IsBlock {
				for _, attribute := range htmlObject.Attribute {
					if rootAttribute.Name == attribute.Name {
						blocksWithPoisonRemoved[index].RootAttribute[index2].Value = fmt.Sprintf("PLACE-HOLDER-VALUE/%s", attribute.TerraformURL)
						break
					}
				}
			}
		}
	}

	return blocksWithPoisonRemoved
}

func CheckForSliceWithMaps(armPropertyValue interface{}) bool {
	var isMap bool
	switch armPropertyValue.(type) {
	case []interface{}:
		{
			for _, value := range armPropertyValue.([]interface{}) {
				if CheckForMap(value) {
					isMap = true
				} else {
					return false
				}
			}
		}
	} //Fix issue with the fact that we now see type: []map[string]interface{}

	return isMap
}

func GetArmMasterKey(armPropertyValue interface{}) string {
	var masterKey string
	if CheckForSlice(armPropertyValue) {
		for _, slice := range armPropertyValue.([]interface{}) {
			for name, value := range slice.(map[string]interface{}) {
				if CheckForString(value) {
					if CompareStrings(name, "name") {
						masterKey = value.(string)
					}
				}
			}
		}
	} else if CheckForMap(armPropertyValue) {
		for name, value := range armPropertyValue.(map[string]interface{}) {
			if CheckForString(value) {
				if CompareStrings(name, "name") {
					masterKey = value.(string)
				}
			}
		}
	}
	return masterKey
}

func CompareStrings(armPropertyName1, armPropertyName2 string) bool {
	return strings.ToLower(armPropertyName1) == strings.ToLower(armPropertyName2)
}

func ConvertMapToRootAttribute(armPropertyName string, armPropertyValue interface{}, attributes []Attribute, blockName string, masterKey string) RootAttribute {
	for attributeName, attributeValue := range armPropertyValue.(map[string]interface{}) {
		htmlAttribute := GetHtmlAttributeMatch(attributeName, attributes, attributeValue, blockName)

		for _, attribute := range htmlAttribute {
			if attribute != (Attribute{}) {
				if attribute.Type == "armObject" {
					rootAttribute := RootAttribute{
						Name:            attribute.Name,
						Value:           attributeValue,
						BlockName:       blockName,
						IsBlock:         true,
						UniqueBlockName: masterKey,
					}
					return rootAttribute
				} else {
					rootAttribute := RootAttribute{
						Name:            attribute.Name,
						Value:           attributeValue,
						BlockName:       blockName,
						IsBlock:         false,
						UniqueBlockName: masterKey,
					}
					return rootAttribute
				}

			}
		}
	}
	return (RootAttribute{})
}

func ConvertFlatValueToRootAttribute(armPropertyValue interface{}, attribute Attribute, blockName string, masterKey string) RootAttribute {

	if attribute.Type == "armObject" {
		rootAttribute := RootAttribute{
			Name:            attribute.Name,
			Value:           nil,
			BlockName:       blockName,
			IsBlock:         true,
			UniqueBlockName: masterKey,
		}
		return rootAttribute
	}

	rootAttribute := RootAttribute{
		Name:            attribute.Name,
		Value:           armPropertyValue,
		BlockName:       blockName,
		IsBlock:         false,
		UniqueBlockName: masterKey,
	}

	return rootAttribute
}

func GetHtmlAttributeMatch(armPropertyName string, htmlAttributes []Attribute, armPropertyValue interface{}, blockName string) []Attribute {
	var htmlAttributeReturn []Attribute
	armPropertyNameConvert := ConvertArmAttributeName(armPropertyName, "")
	for _, htmlAttribute := range htmlAttributes {
		if strings.ToLower(armPropertyName) != "id" && strings.ToLower(htmlAttribute.Name) != "location" && strings.ToLower(htmlAttribute.Name) != "locations" && !(htmlAttribute.Name == "name" && htmlAttribute.Parent == "root") {
			if htmlAttribute.Name == armPropertyNameConvert && strings.HasPrefix(htmlAttribute.Parent, blockName) {
				htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
			} else if strings.HasPrefix(htmlAttribute.Name, armPropertyNameConvert) {
				if !strings.Contains(armPropertyNameConvert, "os_") && strings.Contains(htmlAttribute.Name, "_") && strings.Contains(armPropertyNameConvert, "_") || !strings.Contains(htmlAttribute.Name, "_") && !strings.Contains(armPropertyNameConvert, "_") && htmlAttribute.Name != "resource_group_name" && htmlAttribute.Name != "location" && len(htmlAttributeReturn) == 0 {
					blockNameConverted := ConvertArmAttributeName(blockName, nil)
					if strings.HasPrefix(htmlAttribute.Parent, blockNameConverted) {
						htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
					} else if len(htmlAttributeReturn) == 0 {
						htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
					}

				} else if !strings.Contains(armPropertyNameConvert, "os_") && strings.Contains(htmlAttribute.Name, armPropertyNameConvert) && !strings.Contains(htmlAttribute.Name, "ids") { //This negative match will increase in size with experience
					htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
				} else if strings.Contains(armPropertyNameConvert, "os_") && strings.Contains(htmlAttribute.Name, armPropertyNameConvert) && armPropertyNameConvert != htmlAttribute.Name {
					checkForMap := CheckForMap(armPropertyValue)
					if checkForMap {
						for attributeName, _ := range armPropertyValue.(map[string]interface{}) {
							armPropertyInnerNameConvert := ConvertArmAttributeName(attributeName, "")
							if strings.Contains(armPropertyInnerNameConvert, "windows") && strings.Contains(htmlAttribute.Name, "windows") {
								htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
								break
							} else if strings.Contains(armPropertyInnerNameConvert, "linux") && strings.Contains(htmlAttribute.Name, "linux") {
								htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
								break
							} else {
								if strings.HasSuffix(htmlAttribute.Name, armPropertyInnerNameConvert) || htmlAttribute.Name == armPropertyInnerNameConvert && len(htmlAttributeReturn) == 0 {
									htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
									break
								}
							}
						}
					} else {
						if htmlAttribute.Name == armPropertyNameConvert && len(htmlAttributeReturn) == 0 {
							htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
						}
					}
				}

				if htmlAttribute.Name == armPropertyNameConvert {
					if htmlAttribute.Type == "armObject" && len(htmlAttributeReturn) == 0 {
						htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
					}
				}

				for _, attribute := range htmlAttributes {
					if strings.HasSuffix(attribute.Name, htmlAttribute.Name) && attribute.Type == "armObject" && attribute.Name != htmlAttribute.Name && len(htmlAttributeReturn) == 0 {
						htmlAttributeReturn = append(htmlAttributeReturn, attribute)
					}
				}
			} else if strings.HasSuffix(htmlAttribute.Name, armPropertyNameConvert) && (strings.Contains(htmlAttribute.Name, "_") && strings.Contains(armPropertyNameConvert, "_") || !strings.Contains(htmlAttribute.Name, "_") && !strings.Contains(armPropertyNameConvert, "_")) && len(htmlAttributeReturn) == 0 {
				htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
			}
		}
	}

	if armPropertyNameConvert == "key_data" {
		for _, htmlAttribute := range htmlAttributes {
			if CheckForString(armPropertyValue) {
				partOfValue := strings.Split(armPropertyValue.(string), " ")
				if len(partOfValue) > 1 {
					if strings.Contains(htmlAttribute.Descriptor, partOfValue[0]) {
						htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
						break
					}
				}
			}
		}
	}

	if len(htmlAttributeReturn) == 0 {
		armPropertyNamePart := strings.Split(armPropertyNameConvert, "_")
		for _, htmlAttribute := range htmlAttributes {
			if len(armPropertyNamePart) == 2 {
				if htmlAttribute.Name == armPropertyNamePart[1] {
					htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
					break
				}
			}
		}
	}

	cleanHtmlAttributeReturn := []Attribute{}
	for _, htmlAttribute := range htmlAttributeReturn {
		if strings.Contains(htmlAttribute.Name, "name") {
			convertBlockName := ConvertArmAttributeName(blockName, "")
			if strings.Contains(htmlAttribute.Parent, convertBlockName) {
				cleanHtmlAttributeReturn = append(cleanHtmlAttributeReturn, htmlAttribute)
			}
		} else {
			cleanHtmlAttributeReturn = append(cleanHtmlAttributeReturn, htmlAttribute)
		}
	}

	return cleanHtmlAttributeReturn

}

func CheckForMap(mapToCheck interface{}) bool {
	if _, ok := mapToCheck.(map[string]interface{}); ok {
		return ok //ok can only be true
	}
	return false
}

func CheckForSlice(sliceToCheck interface{}) bool {
	reflectValue := reflect.ValueOf(sliceToCheck)
	if reflectValue.Kind() == reflect.Slice {
		return true
	}
	return false
}

func GetInnerMapFlatValue(mapToCheck interface{}, armPropertyName string) interface{} {
	checkMap, ok := mapToCheck.(map[string]interface{})
	regex := regexp.MustCompile(`[a-z]+|[A-Z][a-z]*`)
	if ok {
		for attributeName, attributeValue := range checkMap {
			startOfAttributeName := regex.FindAllString(armPropertyName, -1)[0]
			if strings.Contains(attributeName, startOfAttributeName) {
				return attributeValue
			}
		}
	}
	return mapToCheck
}

func NewCachedSystemFiles(htmlObjects []HtmlObject, terraformObjects TerraformObject) error {
	var jsonData []byte
	var err error
	if len(htmlObjects) > 0 {
		jsonData, err = json.MarshalIndent(htmlObjects, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal htmlObjects to JSON: %w", err)
		}
	} else {
		jsonData, err = json.MarshalIndent(terraformObjects, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal terraformObjects to JSON: %w", err)
		}
	}

	// Replace escaped HTML entities with actual characters
	prettyJson := bytes.ReplaceAll(jsonData, []byte(`\u003c`), []byte("<"))
	prettyJson = bytes.ReplaceAll(prettyJson, []byte(`\u003e`), []byte(">"))
	prettyJson = bytes.ReplaceAll(prettyJson, []byte(`\u0026`), []byte("&"))

	// Check if directory exists, and create it if necessary
	dir := filepath.Dir(SystemTerraformDocsFileName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if len(htmlObjects) <= 0 {
		if err := os.WriteFile(SystemTerraformCompiledObjectsFileName, prettyJson, 0644); err != nil {
			return fmt.Errorf("failed to write JSON to file %s: %w", SystemTerraformCompiledObjectsFileName, err)
		}
	} else {
		// Write the modified pretty-printed JSON directly to the file
		if err := os.WriteFile(SystemTerraformDocsFileName, prettyJson, 0644); err != nil {
			return fmt.Errorf("failed to write JSON to file %s: %w", SystemTerraformDocsFileName, err)
		}
	}

	return nil
}

func GetCachedSystemFiles() ([]HtmlObject, error) {
	var htmlObjects []HtmlObject
	file, err := os.ReadFile(SystemTerraformDocsFileName)
	if err != nil {
		return nil, err
	}
	extractJson := json.Unmarshal(file, &htmlObjects)
	if extractJson != nil {
		return nil, err
	}

	return htmlObjects, nil
}

func NewTerraformObject(terraformCompiledObjects []CompileObject, providerVersion string) TerraformObject {

	terraformObject := TerraformObject{
		ProviderVersion: providerVersion,
		ProviderName:    terraformCompiledObjects[0].ResourceDefinitionName,
		CompileObjects:  terraformCompiledObjects,
	}
	return terraformObject
}

func WriteTerraformConfigToDisk(terraformConfig string, fileName string) {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	// Check if the file is empty
	stat, _ := file.Stat()

	// Prepend newline if the file is not empty
	if stat.Size() > 0 {
		terraformConfig = "\n\n" + terraformConfig
	}

	if err != nil {
		logWarning(fmt.Sprintf("the following error occured while trying to create file name:%s\n%s", fileName, err))
	}
	defer file.Close()
	file.WriteString(terraformConfig)
}

func logFatal(message string) {
	red := color.New(color.FgHiRed).SprintFunc()
	log.Fatal(red(message + "\n"))
}

func logVerbose(message string) {
	if verbose {
		currentTime := timeFormat(time.Now())
		fmt.Printf("VERBOSE: " + currentTime + " " + message + "\n")
	}
}

func logOK(message string) {
	green := color.New(color.FgHiGreen).SprintFunc()
	fmt.Printf(green(message + "\n"))
}

func logWarning(message string) {
	currentTime := timeFormat(time.Now())
	yellow := color.New(color.FgHiYellow).SprintFunc()
	fmt.Printf(yellow("WARNING: " + currentTime + " " + message + "\n"))
}

func timeFormat(time time.Time) string {
	time.Format("2000-01-01 00:00:00")
	return strings.Split(time.Local().String(), ".")[0]
}

func CheckChromeInstalled(customChromePath string) bool {
	var chromePaths []string

	switch runtime.GOOS {
	case "windows":
		// Common paths for Chrome on Windows
		chromePaths = []string{
			"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
		}
	case "darwin":
		// Common path for Chrome on macOS
		chromePaths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
	case "linux":
		// Common paths for Chrome on Linux
		chromePaths = []string{
			"/usr/bin/google-chrome",
			"/usr/local/bin/google-chrome",
			"/snap/bin/chromium", // For Chromium as an alternative
		}
	default:
		logFatal(fmt.Sprintf("Unsupported operating system %s", runtime.GOOS))
		return false
	}

	// Check if any of the paths exist
	for _, path := range chromePaths {
		if customChromePath != "" {
			if _, err := os.Stat(customChromePath); err == nil {
				return true
			}
		}
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}
