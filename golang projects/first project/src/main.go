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
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type Attribute struct {
	Type       string `json:"Type"`
	Name       string `json:"Name"`
	Parent     string
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
}

// Define the HtmlObject struct with a named attributes field
type HtmlObject struct {
	Resource_type string      `json:"Resource_type"`
	Version       string      `json:"Version"`
	Attribute     []Attribute `json:"Attribute"`
	Not_found     bool
}

type RootAttribute struct {
	Name            string
	Value           interface{}
	BlockName       string //Must be used with format <root name>/<level 1 object>/<level 2 object>
	IsBlock         bool
	UniqueBlockName string //Master key - To make sure all data can be linked directly
}

type Variable struct {
	Name         string
	Description  string
	DefaultValue string
}

type BlockAttribute struct {
	BlockName     string
	RootAttribute []RootAttribute
	Parent        string
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
}

type TerraformObject struct {
	ProviderVersion string
	ProviderName    string
	CompileObjects  []CompileObject
}

var htmlObjects = []HtmlObject{}
var AttributeObjects = []Attribute{}
var SystemTerraformDocsFileName string = "./terrafyarm/htmlobjects.json"
var SystemTerraformCompiledObjectsFileName string = "./terrafyarm/terraform-arm-compiled-objects.json"
var currentVersion string = "0.1.0"
var verbose bool

func init() {
	// Define the verbose flag
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
}

func main() {
	// Define and parse the filePath flag
	filePath := flag.String("file-path", "./", "Path to the file ARM json file(s) Can be either a specific file path or directory path")
	noCache := flag.Bool("no-cache", false, "Switch to determine whether to use the cache if any is present")
	clearCache := flag.Bool("clear-cache", false, "Switch to determine whether the application shall remove all cache")
	providerVersion := flag.String("provider-version", "latest", "Use a custom version for the terraform decompiler - Useful in cases where ARM templates are old and where an older provider version might give better results, '<major, minor, patch>', eg. '3.11.0' ")
	seperateSubResources := flag.Bool("seperate-nested-resources", false, "Switch to determine whether the decompiler shall seperate nested resources into their own Terraform resource definiton, OBS. This switch will not be enabled before version 0.2.0.\nFollow the resource notes for more details")
	rootDecompilefolderPath := flag.String("output-file-path", "", `Path for the new root decompile folder which will be created - Parse either a full path, e.g. C:\Decompile\<some folder name>\<end folder name, ultimately the name of the comming folder, or use a relative path e.g. './<some folder name>/<end folder name, ultimately the name of the comming folder'`)
	seperateDecompiledResources := flag.Bool("seperate-output-files", false, "Switch to determine whether the resources being decompiled shall reside in isolated sub folders")
	version := flag.Bool("version", false, "Switch to check the current version of 'TerrafyArm'")
	customProviderContext := flag.String("custom-provider-context", "", "(NOT IN USE AS OF VERSION 0.1.0 - Parse custom provider contexts in the format of '<resource_id>,<custom provider name>'\nOnly the resource_id part is mandatory)")

	flag.Parse()

	if *version {
		fmt.Printf("The current version of the 'TerrafyArm' Decompiler is '%s'\nFor information about versions, please check the official Github release page at:\nhttps://github.com/ChristofferWin/TerrafyARM/releases", currentVersion)
		return
	}

	if *customProviderContext != "" {
		fmt.Printf("The argument of 'customProviderContext' Is not activated as of this current version...")
		return
	}

	if *clearCache {
		err := os.RemoveAll(strings.Split(SystemTerraformDocsFileName, "/")[1])
		if err != nil {
			fmt.Println("The following error occured while trying to delete the cache:", err)
		}
		return
	}

	// Call the ImportArmFile function with the filePath argument
	fileContent, err := ImportArmFile(filePath)

	if err != nil {
		// Handle the error if it occurs
		fmt.Println("Error reading file:", err)
		return
	}

	verifiedFiles := VerifyArmFile(fileContent)
	if len(verifiedFiles) == 0 {
		fmt.Println("No valid ARM templates found on path:", *filePath)
		return
	}

	if *seperateSubResources {
		fmt.Println("This flag is not implemented in the current version of TerrafyArm")
	}

	baseArmResources := GetArmBaseInformation(verifiedFiles)

	if err != nil {
		fmt.Println("Error while trying to retrieve the json ARM content", err)
	}

	var predetermineTypes []string
	for _, armResource := range baseArmResources {
		if armResource.Special_resource_type != "" {
			for _, resourceType := range armResource.Resource_types {
				predetermineTypes = append(predetermineTypes, ConvertArmToTerraformProvider(resourceType, armResource.Special_resource_type))
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
			fmt.Println("No cache detected, retrieving all required information...")
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

	if htmlObjectsFromCache == nil {
		for _, resourceType := range sortedResourceTypes {
			if resourceType != "" {
				rawHtml, err := GetRawHtml(resourceType, *providerVersion)

				if err != nil {
					fmt.Println("Error while trying to retrieve required documentation", err, resourceType)
					break
				}
				if rawHtml != "not_found" && rawHtml != "" {
					cleanHtml = SortRawHtml(rawHtml, resourceType)
					htmlObjects = append(htmlObjects, cleanHtml)
				} else {
					htmlObject := HtmlObject{
						Resource_type: resourceType,
						Not_found:     true, //We need to add more information to this struct, e.g. even though we cant retrieve its HTML, it still has a type, etc.
					}
					htmlObjects = append(htmlObjects, htmlObject)
				}
			}
		}
		if !*noCache {
			err = NewCachedSystemFiles(htmlObjects, TerraformObject{})
			if err != nil {
				fmt.Println("An error occured while running function 'NewCachedSystemFiles'", err)
			}
		}
	} else {
		htmlObjects = append(htmlObjects, htmlObjectsFromCache...)
	}

	var cleanhtmlObjects []HtmlObject

	for _, htmlObject := range htmlObjects {
		if !htmlObject.Not_found {
			cleanhtmlObjects = append(cleanhtmlObjects, htmlObject)
		}
	}

	compiledObjects := CompileTerraformObjects(baseArmResources, cleanhtmlObjects, *seperateSubResources)
	for _, object := range compiledObjects {
		fmt.Println("FILE NAME FOUND:", object.FilePath, "NAME", object.SeperatedResource)
	}
	terraformObject := NewTerraformObject(compiledObjects, *providerVersion)

	err = NewCachedSystemFiles([]HtmlObject{}, terraformObject)
	if err != nil {
		fmt.Println("An error occured while running function 'NewCachedSystemFiles'", err)
	}

	terraformFileNames := NewCompiledFolderStructure(*seperateDecompiledResources, *rootDecompilefolderPath, terraformObject)

	providerFullPathName := ""
	for _, terraformFileName := range terraformFileNames {
		if strings.Contains(terraformFileName, "providers.tf") {
			providerFullPathName = terraformFileName
			break
		}
	}

	InitializeTerraformFile(providerFullPathName, *providerVersion, terraformObject.ProviderName)

	for _, terraformCompiledObject := range terraformObject.CompileObjects {
		NewTerraformConfig(terraformCompiledObject, *seperateDecompiledResources)
	}
}

func NewCompiledFolderStructure(seperatedFiles bool, rootFolderPath string, terraformObject TerraformObject) []string {
	var fileNames []string
	var terraformFilePaths []string
	//var folderNames []string Not in use yet

	err := os.Mkdir(rootFolderPath, 0755)
	if err != nil {
		if !os.IsExist(err) {
			fmt.Println("an error occured while trying to create the root directory for the decompiled files...\n", err)
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

	for _, name := range fileNames {
		fmt.Println("COMMING NAME:", name)
	}

	for _, fileName := range fileNames {
		fullPath := ""
		if !strings.Contains(rootFolderPath, ".") {
			fullPath = filepath.Join(rootFolderPath, fileName) // Get the full path
		} else {
			fullPath = fmt.Sprintf("%s/%s", rootFolderPath, fileName)
		}

		logVerbose(fmt.Sprintf("Creating the following file '%s' on location '%s'", fileName, rootFolderPath))
		err := os.WriteFile(fullPath, []byte{}, 0644)
		if err != nil {
			fmt.Println("an error occured while trying to create file '%s'\n%s", fileName, err)
			return []string{}
		}
		terraformFilePaths = append(terraformFilePaths, fullPath)
	}

	return terraformFilePaths
}

func InitializeTerraformFile(terraformFilePath string, providerVersion string, providerName string) {
	initialCommentBlock := `/*
	This Terraform template is created using 'TerrafyArm'
	Template CAN have issues - Please report these as issues over on github at https://github.com/ChristofferWin/TerrafyARM/issues
	As 'TerrafyArm' Progresses in development, please consolidate the releases page at https://github.com/ChristofferWin/TerrafyARM/releases 
	`

	requiredProvidersBlock := `
	Required boilerplate - The 'version' Will only be set when the argument '-provider-version' <some version> has been parsed
	*/
	`

	requiredContextBlock := `/*
	Required context block - By not defining any specific subscription_id, terraform will use in-line context
	To read more about "in-lnline" Terraform Azure CLI context, please consolidate the following blog post: https://www.codeterraform.com/post/terraform-weekly-tips-1
	In future versions it will be possible to parse custom provider context´s of which the user wants defined.
	*/

	provider "azurerm" {
	  features{}
	}
	`

	initialize_terraform := ""
	if providerVersion == "latest" {
		initialize_terraform = fmt.Sprintf("terraform {\n  required_providers {\n      %s = {\n         source = \"hashicorp/%s\"\n         }\n      }\n}", providerName, providerName)
	} else {
		initialize_terraform = fmt.Sprintf("terraform {\n  required_providers {\n      %s = {\n         source = \"hashicorp/%s\"\n         version = \"%s\"\n         }\n      }\n}", providerName, providerName, providerVersion)
	}

	finalInitializeTerraform := fmt.Sprintf("%s\n%s\n%s\n\n%s", initialCommentBlock, requiredProvidersBlock, initialize_terraform, requiredContextBlock)
	ChangeExistingFile(terraformFilePath, finalInitializeTerraform)
	if CheckForTerraformExecuteable() {
		RunTerraformCommand("fmt", terraformFilePath)
	}
}

func NewTerraformConfig(terraformCompiledObject CompileObject, seperatedResources bool) {
	rootTerraformConfig := []string{}
	//resourceName := ""
	//terraformProvider := ""
	//terraformResourceType := ""
	fmt.Println("HERE:32", terraformCompiledObject.ResourceName, len(terraformCompiledObject.BlockAttributes))
	for _, block := range terraformCompiledObject.BlockAttributes {
		fmt.Println("\n------------BLOCK BLOCK-------------")
		fmt.Println("NAME:", block.BlockName, "PARENT:", block.Parent)
		for index, rootAttribute := range block.RootAttribute {
			fmt.Println(index, "ROOT NAME:", rootAttribute.Name, "IS BLOCK", rootAttribute.IsBlock, "PARENT", rootAttribute.BlockName, "UNIQUE", rootAttribute.UniqueBlockName, "VALUE", rootAttribute.Value)
		}
	}
	rootTerraformDefinition := NewTerraformResourceDefinitionName(terraformCompiledObject.ResourceName, terraformCompiledObject.ResourceType, terraformCompiledObject.ResourceDefinitionName)
	rootTerraformConfig = append(rootTerraformConfig, rootTerraformDefinition)
	firstTime := true
	for _, blockAttribute := range terraformCompiledObject.BlockAttributes {
		blockNameFixed := ConvertArmAttributeName(blockAttribute.BlockName, "")
		if blockAttribute.Parent == "root" || blockAttribute.Parent == "" {
			if blockNameFixed != "" && blockNameFixed == "root" {
				for _, terraformAttribute := range blockAttribute.RootAttribute {
					if !terraformAttribute.IsBlock && (terraformAttribute.BlockName == "root" || terraformAttribute.BlockName == "") {
						if firstTime {
							firstTime = false
							for _, htmlAttribute := range terraformCompiledObject.HtmlObject.Attribute {
								if htmlAttribute.Name == "name" && htmlAttribute.Parent == "root" {
									rootAttribute := RootAttribute{
										Name:  "name",
										Value: terraformCompiledObject.ResourceName,
									}
									rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("", rootAttribute, false))
								}

								if htmlAttribute.Name == "resource_group_name" && htmlAttribute.Parent == "root" {
									rootAttribute := RootAttribute{
										Name:  "resource_group_name",
										Value: terraformCompiledObject.ArmObject.Resource_group_name,
									}
									rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("", rootAttribute, false))
								}

								if htmlAttribute.Name == "location" && htmlAttribute.Parent == "root" {
									rootAttribute := RootAttribute{
										Name:  "location",
										Value: terraformCompiledObject.ArmObject.Location,
									}
									rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("", rootAttribute, false))
								}
							}
							firstTime = false
						} else {
							rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("", terraformAttribute, false))
						}
					} else {
						fmt.Println("BLOCK NAME:3333", blockNameFixed, blockAttribute.RootAttribute)
					}
				}
			}
		}
	}
	rootTerraformConfig = append(rootTerraformConfig, AddFlatTerraformAttributeForResourceDefinition("\n}", RootAttribute{}, true))
	fmt.Println("CONFIG:", rootTerraformConfig)
}

func NewTerraformResourceDefinitionName(terraformResourceName string, terraformResourceType string, terraformProvider string) string {

	return fmt.Sprintf("resource \"%s_%s\" \"%s\" {", terraformProvider, terraformResourceType, terraformResourceName)
}

func AddFlatTerraformAttributeForResourceDefinition(terraformBlock string, terraformAttribute RootAttribute, endTerraformDefinition bool) string {
	valueTypeName := FindTypeByString(terraformAttribute.Value)
	if terraformBlock != "" {
		if terraformAttribute != (RootAttribute{}) {
			if valueTypeName == "bool" {
				return fmt.Sprintf("%s\n%s = %t", terraformBlock, terraformAttribute.Name, terraformAttribute.Value)
			} else if valueTypeName == "slice" {
				return fmt.Sprintf("%s\n%s = %v", terraformBlock, terraformAttribute.Name, terraformAttribute.Value)
			} else {
				return fmt.Sprintf("%s\n%s = \"%s\"", terraformBlock, terraformAttribute.Name, terraformAttribute.Value)
			}
		} else {
			if endTerraformDefinition {
				return terraformBlock
			}
		}
	} else {
		if !endTerraformDefinition {
			if valueTypeName == "bool" {
				return fmt.Sprintf("%s\n%s = %t", terraformBlock, terraformAttribute.Name, terraformAttribute.Value)
			} else if valueTypeName == "slice" {
				return fmt.Sprintf("\n%s = %v", terraformAttribute.Name, terraformAttribute.Value)
			}
		}
	}

	return fmt.Sprintf("\n%s = \"%s\"", terraformAttribute.Name, terraformAttribute.Value)
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
	splitPath := strings.Split(terraformFilePath, "/")
	finalPath := strings.Join(splitPath[:len(splitPath)-1], "/")
	finalCommand := terraformCommand
	command := exec.Command("terraform", finalCommand)
	command.Dir = finalPath
	_, err := command.CombinedOutput()
	if err != nil {
		logVerbose(fmt.Sprintf("an error occured while trying to run command '%s'", terraformCommand))
	}
}

func ChangeExistingFile(filePath string, text string) (int, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("an error occured while trying to open the file '%s'\n%s", filePath, err)
	}

	defer file.Close() //Make sure to close the file as the last part of the function call

	numberOfLines, err := file.WriteString(text)
	if err != nil {
		log.Fatalf("an error occured while trying to write to file '%s'\n%s", filePath, err)
	}

	return numberOfLines, nil
}

func ImportArmFile(filePath *string) ([][]byte, error) {

	var fileNames []string
	var files [][]byte
	fileInfo, err := os.Stat(*filePath)
	if err != nil {
		fmt.Println("Error while trying to retrieve ARM json files on path:", string(*filePath), "\nStacktrace:", err)
	}

	isDir := fileInfo.IsDir()
	flag.Parse()

	if isDir {
		files, err := os.ReadDir(*filePath)

		if err != nil {
			fmt.Println("Error while trying to retrieve ARM json files on path:", string(*filePath), "\nStacktrace:", err)
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
			fmt.Println("Error while trying to retrieve ARM json content on file:", string(fileName), "\nStracktrace:", err)
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
			fmt.Println("Error while transforming file from bytes to json:", err)
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
			attributePartName := strings.Split(specialResouceType, "/")
			if attributePartName[0] == "app" && attributePartName[1] == "linux" {
				convertResourceTypeTempName = fmt.Sprintf("%s_%s_%s", attributePartName[1], "web", attributePartName[0])
			} else if attributePartName[0] == "app" && attributePartName[1] == "windows" {
				convertResourceTypeTempName = fmt.Sprintf("%s_%s_%s", attributePartName[1], "web", attributePartName[0])
			} else if attributePartName[0] == "func" && attributePartName[1] == "linux" {
				convertResourceTypeTempName = fmt.Sprintf("%s_%s_%s", attributePartName[1], "function", attributePartName[0])
			} else if attributePartName[0] == "func" && attributePartName[1] == "windows" {
				convertResourceTypeTempName = fmt.Sprintf("%s_%s_%s", attributePartName[1], "function", attributePartName[0])
			}
		}
	} else if checkNamesForCompute.MatchString(resourceTypeLower) {
		if specialResouceType != "" {
			convertResourceTypeTempName = strings.ToLower(specialResouceType)
			//convertResourceTypeTempName = fmt.Sprintf("%s_%s", attributePartName[1], resourceNameBaseConversion[1])
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
				} else if convertResourceTypeTempName == "" {
				}
			}
		}
	} else if patternResourceType.MatchString(resourceType) && convertResourceTypeTempName == "" {
		convertResourceTypeTempName = fmt.Sprintf("%s_%s", strings.Split(resourceNameBaseConversion[0], ".")[1], resourceNameBaseConversion[1])
	} else {

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

func GetRawHtml(resourceType string, providerVersion string) (string, error) {
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

	fmt.Println("THIS IS IT:", convertResourceTypeName)

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
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true), // Set to false if you want to see the browser UI
		chromedp.Flag("disable-gpu", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	url := fmt.Sprintf("https://registry.terraform.io/providers/hashicorp/azurerm/%s/docs/resources/%s", providerVersion, convertResourceTypeName)
	fmt.Println(url)

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
		fmt.Println("Warning: The resource:", convertResourceTypeName, "could not be found at provider version:", providerVersion)
		fmt.Println("Verbose: The resource type:", resourceType, "could not be found..\nThis might be due to an invalid translation...\nIf so, please create a github issue at: https://github.com/ChristofferWin/TerrafyARM/issues")
		return "not_found", nil
	}
	return HtmlBodyCompare, nil
}

func SortRawHtml(rawHtml string, resourceType string) HtmlObject { //See the struct type definitions towards the top of the file
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
			htmlAttribute := Attribute{
				Name:       blockNameStringSubMatch[1],
				Type:       "armObject",
				Parent:     "",
				Descriptor: descriptor,
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

			if checkForList := regexp.MustCompile(`\blist\b`); checkForList.MatchString(line) {
				typeOfHtmlAttribute = "list"
			} else {
				typeOfHtmlAttribute = "string"
			}
			htmlAttiribute := Attribute{
				Name:       blockNameStringSubMatch[1],
				Type:       typeOfHtmlAttribute,
				Parent:     parentName,
				Descriptor: descriptor,
			}
			allHtmlAttributes = append(allHtmlAttributes, htmlAttiribute)
		}
	}

	mapOfHtmlAttributes := make(map[string]bool)
	patternValidHtmlAttributeName := regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)

	for _, htmlAttribute := range allHtmlAttributes {
		if !mapOfHtmlAttributes[htmlAttribute.Name] && patternValidHtmlAttributeName.MatchString(htmlAttribute.Name) {
			uniqueHtmlAttributes = append(uniqueHtmlAttributes, htmlAttribute)
			if htmlAttribute.Name != "name" {
				mapOfHtmlAttributes[htmlAttribute.Name] = true
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
	//var notPartOfRoot bool

	for _, armBasicObject := range armBasicObjects {
		var htmlObjectCaptures []HtmlObject
		blockObjectsNestedResources := []BlockAttribute{}
		blockObjectsRootResources := []BlockAttribute{}
		blockObjectsSeperatedResources := []BlockAttribute{}
		rootAttributesFromNestedResources := []RootAttribute{}
		rootAttributesFromSeperatedResources := []RootAttribute{}

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
			fmt.Println("FOUND THE FOLLOWING RESOURCE TYPE:", armResourceType)
			resourceFileName := ""
			keyForSubType := ""
			masterKey := ""
			if armResourceType.CanBeSeperated {
				captureHtml := HtmlObject{}
				for _, htmlObject := range htmlObjectCaptures {
					if armResourceType.Resource_type == htmlObject.Resource_type {
						captureHtml = htmlObject
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
				//rootAttributesForReturn = GetInnerRootAttributes(armBasicObject, captureHtml.Attribute) //We need to specifically capture the properties of the resource types that are matched by the if statement
			} else if !(len(strings.Split(armResourceType.Resource_type, "/")) == 2) && strings.Contains(armResourceType.Resource_type, "/") {
				//notPartOfRoot = true
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
					for _, slice := range retrievePartOfArmProperties.([]interface{}) {
						masterKey = GetArmMasterKey(slice)
						rootAttributesFromSeperatedResources = append(rootAttributesFromSeperatedResources, GetInnerRootAttributes(slice, captureHtml, masterKey, keyForSubType)...)
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
									rootAttributesForReturn = append(rootAttributesForReturn, ConvertMapToRootAttribute(attributeName, attributeValue, captureHtml.Attribute, match.Parent, ""))
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
										} else if CheckForSliceWithMaps(minimumAttributeValue) {
											//fmt.Println("NOW WE HERE::", minimumAttributeValue)
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
				resourceName := ""
				if masterKey == "" || armResourceType.CanBeSeperated {
					resourceName = armBasicObject.Name
				} else {
					resourceName = masterKey
				}
				keyPartName := ConvertArmAttributeName(keyForSubType, "")
				if armResourceType.Parent == "root" || armResourceType.CanBeSeperated {
					if armBasicObject.Special_resource_type != "" {
						resourceFileName = fmt.Sprintf("%s.tf", armBasicObject.Special_resource_type)
					} else {
						for _, resourceType := range armBasicObject.Resource_types {
							if len(strings.Split(resourceType, "/")) == 2 {
								resourceFileName = fmt.Sprintf("%s.tf", ConvertArmAttributeName(strings.Split(resourceType, "/")[1], ""))
							}
						}
					}
				} else {
					resourceFileName = fmt.Sprintf("%s.tf", keyPartName)
				}

				if armResourceType.Parent == "root" {
					blockObjectsRootResources = append(blockObjectsRootResources, GetBlocksFromRootAttributes(rootAttributesForReturn, htmlObjectCaptureRootArmResources)...)
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
					}
					compiledObjects = append(compiledObjects, compileObjectRootResources)

				} else if !armResourceType.CanBeSeperated {
					blockObjectsSeperatedResources = append(blockObjectsSeperatedResources, GetBlocksFromRootAttributes(rootAttributesFromSeperatedResources, htmlObjectCaptureSeperatedResource)...)
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
					}
					compiledObjects = append(compiledObjects, compileObjectSeperatedResources)
				}
			}
		}
		blockObjectsNestedResources = append(blockObjectsNestedResources, GetBlocksFromRootAttributes(rootAttributesFromNestedResources, htmlObjectCaptureNestedArmResources)...)
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
		//fmt.Println("NEW COMPILED OBJECT", compiledObject.ResourceName, "BLOCKS", compiledObject.BlockAttributes)
		blockAttributesForRoot := []BlockAttribute{}
		blockAttributesForRest := []BlockAttribute{}
		for index2, blockAttribute := range compiledObject.BlockAttributes {
			if blockAttribute.Parent == "" {
				compiledObjects[index].BlockAttributes[index2].Parent = "root"
			}
			blockNameFixed := ConvertArmAttributeName(blockAttribute.BlockName, "")
			if blockNameFixed == strings.Replace(compiledObject.FilePath, ".tf", "", 1) || blockAttribute.BlockName == "root" {
				fmt.Println("WE ARE HERE BOIS", blockNameFixed)
				blockAttributesForRoot = append(blockAttributesForRoot, blockAttribute)
			} else if blockAttribute.BlockName != "root" && blockAttribute.BlockName != "" {
				blockAttributesForRest = append(blockAttributesForRest, blockAttribute)
			}
		}
		blockAttributesForRoot = append(blockAttributesForRoot, blockAttributesForRest...)
		compiledObjects[index].BlockAttributes = blockAttributesForRoot
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
	/*
		for _, compiledObject := range compiledObjects {
			for _, blockAttribute := range compiledObject.BlockAttributes {
				cleanUpFalseRootAttributes := []RootAttribute{}
				for _, rootAttributes := range blockAttribute.RootAttribute {
					if
				}
			}
		}
	*/
	return compiledObjects
}

func FindTypeByString(value interface{}) string {
	reflectValue := reflect.ValueOf(value)
	if reflectValue.Kind() == reflect.String {
		return "string"
	} else if reflectValue.Kind() == reflect.Slice {
		return "slice"
	} else if reflectValue.Kind() == reflect.Bool {
		return "bool"
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
				} else if CheckForEmptyArmValue(innerAttributeValue) {
					//fmt.Println("We are missing out on:", innerAttributeName, innerAttributeValue)
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
									for _, innerSlice := range minimumAttributeValue.([]interface{}) {
										fmt.Println("NOW WE HERE_:", innerSlice)
									}
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
				} else if CheckForMap(innerAttributeValue) {

				} else {
					//fmt.Println("FINALLY FOUND IT:", innerAttributeName, innerAttributeValue)
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
						if CheckForMap(innerAttributeValue) {
							for minimumAttributeName, _ := range innerAttributeValue.(map[string]interface{}) {
								fmt.Println("ARE WE HERE NOW?", minimumAttributeName)
							}
						} else if CheckForSliceWithMaps(innerAttributeValue) {
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
								fmt.Println("ARE WE HERE NOW????", minimumAttributeName)
								htmlAttributeMatch := GetHtmlAttributeMatch(minimumAttributeName, htmlObject.Attribute, minimumAttributeValue, innerAttributeName)
								if len(htmlAttributeMatch) > 0 && !CheckForEmptyArmValue(minimumAttributeValue) {
									for _, match := range htmlAttributeMatch {
										returnRootAttributes = append(returnRootAttributes, ConvertFlatValueToRootAttribute(minimumAttributeValue, match, match.Parent, masterKey))
									}
								}
							}
						} else {
							fmt.Println("WE missing something=?=", innerAttributeName)
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

func GetBlocksFromRootAttributes(rootAttributes []RootAttribute, htmlObject HtmlObject) []BlockAttribute {
	var blocksForReturn []BlockAttribute
	var summarizedRootAttributes []RootAttribute
	var removeDuplicateValues []string
	var furtherSummarizeRootAttributes []RootAttribute
	var removeShadowRootAttributes []RootAttribute

	for index, rootAttribute := range rootAttributes {
		if !strings.Contains(rootAttribute.Name, "name") && !strings.Contains(rootAttribute.Name, "id") {
			for _, htmlObject := range htmlObjects {
				for _, attribute := range htmlObject.Attribute {
					if rootAttribute.Name == attribute.Name {
						if rootAttribute.BlockName != attribute.Parent {
							rootAttributes[index].BlockName = attribute.Parent
						}
					}
				}
			}
		}
	}

	for index, rootAttribute := range rootAttributes {
		if strings.HasPrefix(rootAttribute.BlockName, "/") || strings.HasSuffix(rootAttribute.BlockName, "/") {
			rootAttributes[index].BlockName = strings.Trim(rootAttribute.BlockName, "/")
		}

		if strings.Contains(rootAttribute.BlockName, "//") {
			rootAttributes[index].BlockName = strings.Replace(rootAttribute.BlockName, "//", "/", 1)
		}
	}
	for index, rootAttribute := range rootAttributes {
		partOfBlockNames := strings.Split(rootAttribute.BlockName, "/")
		if len(partOfBlockNames) > 2 {
			if strings.HasPrefix(rootAttribute.BlockName, partOfBlockNames[1]) {
				blockNamePart := partOfBlockNames[0]
				rootAttributes[index].BlockName = fmt.Sprintf("%s/%s", blockNamePart, strings.Join(partOfBlockNames[2:], "/"))
			} else {
				var uniqueBlockNames []string
				for _, name := range partOfBlockNames {
					if !strings.Contains(strings.Join(uniqueBlockNames, ","), name) {
						uniqueBlockNames = append(uniqueBlockNames, name)
					}
				}
				rootAttributes[index].BlockName = strings.Join(uniqueBlockNames, "/")
			}
		} else if len(partOfBlockNames) == 2 {
			if partOfBlockNames[0] == partOfBlockNames[1] {
				rootAttributes[index].BlockName = partOfBlockNames[0]
			} else if strings.HasPrefix(rootAttribute.BlockName, partOfBlockNames[1]) {
				if strings.Contains(rootAttribute.BlockName, "os") {
					if !strings.Contains(rootAttribute.Name, "name") {
						rootAttributes[index].BlockName = partOfBlockNames[0]
					} else {
						rootAttributes[index].BlockName = partOfBlockNames[1]
					}
				}
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
			} else {

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

	attributeForSubResourceBlock := Attribute{}
	for _, rootAttribute := range almostSummarizeRootAttributes {
		for _, attribute := range htmlObject.Attribute {
			if rootAttribute.BlockName == attribute.Name && attribute.Parent == "root" {
				attributeForSubResourceBlock = attribute
				break
			}
		}
		if attributeForSubResourceBlock != (Attribute{}) {
			break
		}
	}

	//Add the nested object root attribute
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

	blocksForReturn = append(blocksForReturn, blockAttribute)

	for _, block := range blocksForReturn {
		fmt.Println("\n------------------BLOCK NAME:", block.BlockName, "PARENT", block.Parent, "-----------------------")
		for _, attribute := range block.RootAttribute {
			if attribute.UniqueBlockName == "" {
				fmt.Println("\nRoot attribute name:", attribute.Name, "VALUE:", attribute.Value)
			} else {
				fmt.Println("\nRoot attribute name:", attribute.Name, "UNIQUE NAME", attribute.UniqueBlockName, "VALUE:", attribute.Value)
			}
		}
	}

	for index, rootAttribute := range almostSummarizeRootAttributes {
		fmt.Println(index, "ATTRIBUTE NAME", rootAttribute.Name, "||", "BLOCK NAME", rootAttribute.BlockName, "||", "IS BLOCK", rootAttribute.IsBlock, "||", "UNIQUE NAME:", rootAttribute.UniqueBlockName, "VALUE:", rootAttribute.Value)
	}

	return blocksForReturn
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
	//fmt.Println("THIS IS THE ATTRIBUTE BEING MATCHED", "BEFORE:", armPropertyName, "AFTER:", armPropertyNameConvert)
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

	if len(htmlObjects) > 0 {
		// Write the modified pretty-printed JSON directly to the file
		if err := os.WriteFile(SystemTerraformDocsFileName, prettyJson, 0644); err != nil {
			return fmt.Errorf("failed to write JSON to file %s: %w", SystemTerraformDocsFileName, err)
		}
	} else {
		if err := os.WriteFile(SystemTerraformCompiledObjectsFileName, prettyJson, 0644); err != nil {
			return fmt.Errorf("failed to write JSON to file %s: %w", SystemTerraformCompiledObjectsFileName, err)
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

func logVerbose(message string) {
	if verbose {
		fmt.Println("VERBOSE: " + message)
	}
}
