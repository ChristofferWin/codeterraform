package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type Attribute struct {
	Type   string `json:"Type"`
	Name   string `json:"Name"`
	Parent string
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
}

type TerraformObject struct {
	ProviderVersion string
	CompileObjects  []CompileObject
}

var TerraformCompiledObject TerraformObject
var HtmlObjects = []HtmlObject{}
var AttributeObjects = []Attribute{}
var SystemTerraformDocsFileName string = "./terraformdecompile/terraformdocsresourcedefinitions.json"

func main() {
	// Define and parse the filePath flag
	filePath := flag.String("file-path", "./", "Path to the file ARM json file(s) Can be either a specific file path or directory path")
	noCache := flag.Bool("no-cache", false, "Switch to determine whether to use the cache if any is present")
	clearCache := flag.Bool("clear-cache", false, "Switch to determine whether the application shall remove all cache")
	providerVersion := flag.String("provider-version", "latest", "Use a custom version for the terraform decompiler - Useful in cases where ARM templates are old and where an older provider version might give better results, '<major, minor, patch>', eg. '3.11.0' ")
	flag.Parse()

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

	cleanHtml := HtmlObject{}

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
					HtmlObjects = append(HtmlObjects, cleanHtml)
				} else {
					htmlObject := HtmlObject{
						Resource_type: resourceType,
						Not_found:     true, //We need to add more information to this struct, e.g. even though we cant retrieve its HTML, it still has a type, etc.
					}
					HtmlObjects = append(HtmlObjects, htmlObject)
				}
			}
		}
		if !*noCache {
			err = NewCachedSystemFiles(HtmlObjects)
			if err != nil {
				fmt.Println("An error occured while running function 'NewCachedSystemFiles'", err)
			}
		}
	} else {
		HtmlObjects = append(HtmlObjects, htmlObjectsFromCache...)
	}

	var cleanHtmlObjects []HtmlObject

	for _, htmlObject := range HtmlObjects {
		if !htmlObject.Not_found {
			cleanHtmlObjects = append(cleanHtmlObjects, htmlObject)
		}
	}

	GetRootAttributes(baseArmResources, cleanHtmlObjects)

}

func ImportArmFile(filePath *string) ([][]byte, error) {

	var fileNames []string
	var files [][]byte
	fileInfo, err := os.Stat(*filePath)
	if err != nil {
		fmt.Println("Error while trying to retrieve ARM json files on path:", string(*filePath), "\nStracktrace:", err)
	}

	isDir := fileInfo.IsDir()
	flag.Parse()

	if isDir {
		files, err := os.ReadDir(*filePath)

		if err != nil {
			fmt.Println("Error while trying to retrieve ARM json files on path:", string(*filePath), "\nStracktrace:", err)
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

	case string:
		{
			attributePartName := strings.Split(armPropertyValue.(string), ",")
			if attributePartName[0] == "app" {
				tempReturnArmKind = fmt.Sprintf("%s/%s", "app", attributePartName[1])
			} else if attributePartName[0] == "func" {
				tempReturnArmKind = fmt.Sprintf("%s/%s", "func", attributePartName[1])
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

func ConvertArmAttributeName(armPropertyName string, armPropertyValue interface{}) string {
	var returnConvertedName string

	patternLongArmAttributeName := regexp.MustCompile("([a-z0-9])([A-Z])")
	if patternLongArmAttributeName.MatchString(armPropertyName) {
		returnConvertedName = strings.ToLower(strings.TrimSuffix(patternLongArmAttributeName.ReplaceAllString(armPropertyName, "${1}_${2}"), "s"))
		//fmt.Println(returnConvertedName)
	}

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

	for index, line := range linesHtml {
		blockNameStringSubMatch := patternHtmlBlockName.FindStringSubmatch(line)
		typeOfHtmlAttribute := ""
		parentName := ""
		blockNameStringSubMatchInner := []string{}
		if strings.Contains(line, "block supports") && len(blockNameStringSubMatch) > 1 {
			htmlAttribute := Attribute{
				Name:   blockNameStringSubMatch[1],
				Type:   "armObject",
				Parent: "",
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
				Name:   blockNameStringSubMatch[1],
				Type:   typeOfHtmlAttribute,
				Parent: parentName,
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
	fmt.Println("THIS IS THE RESOURCE TYPE:", resourceType)
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

func GetRootAttributes(armBasicObjects []ArmObject, HtmlObjects []HtmlObject) []RootAttribute {
	var rootAttributesForReturn []RootAttribute
	var htmlObjectCapture HtmlObject
	var masterKey string

	//var matchingHtmlObjects []HtmlObject

	for _, armBasicObject := range armBasicObjects {
		allResourceTypes := []string{}
		allResourceTypes = append(append(allResourceTypes, armBasicObject.Resource_types...), armBasicObject.Special_resource_type) //append(append(newBlockNames, blockPart[1]), blockPart[2])
		var rootAttributes []RootAttribute
		var rootAttributesFromReturn []RootAttribute
		for _, armResourceType := range allResourceTypes {
			fmt.Println("THIS IS THE RESOURCE TYPE:", armResourceType)
			for _, htmlObject := range HtmlObjects {
				if armResourceType == htmlObject.Resource_type {
					fmt.Println("MATCHED:", htmlObject.Resource_type)
					break
				}
			}
		}

		for armPropertyName, armPropertyValue := range armBasicObject.Properties.(map[string]interface{}) {

			checkForMap, ok := armPropertyValue.(map[string]interface{})

			if !ok {
				switch armPropertyValue.(type) {
				case []interface{}:
					{
						for _, innerPropertyValue := range armPropertyValue.([]interface{}) {
							masterKey = ""
							if CheckForMap(innerPropertyValue) {
								for innerInnerAttributeName, innerInnerAttributeValue := range innerPropertyValue.(map[string]interface{}) {
									if innerInnerAttributeName == "name" {
										if len(GetHtmlAttributeMatch(armPropertyName, htmlObjectCapture.Attribute, armPropertyValue)) > 0 {
											masterKey = innerInnerAttributeValue.(string)
											break
										}
									}
								}
							}

							htmlAttributeMatch := GetHtmlAttributeMatch(armPropertyName, htmlObjectCapture.Attribute, armPropertyValue)
							if len(htmlAttributeMatch) > 0 {
								rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(nil, htmlAttributeMatch[0], "", masterKey))
								_, ok := innerPropertyValue.(map[string]interface{})
								if ok {
									rootAttributesFromReturn = GetInnerRootAttributes(armPropertyName, innerPropertyValue, htmlObjectCapture.Attribute, masterKey)
									rootAttributes = append(rootAttributes, rootAttributesFromReturn...)
								}
							}

						}
					}
				case map[string]interface{}:
					{
						fmt.Println("YEP, WE ARE HERE", armPropertyName)
					}
				case interface{}:
					{
						htmlAttributeMatch := GetHtmlAttributeMatch(armPropertyName, htmlObjectCapture.Attribute, armPropertyValue)
						if len(htmlAttributeMatch) > 0 {
							rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(armPropertyValue, htmlAttributeMatch[0], "", masterKey))
						}
					}

				}
			} else {
				htmlAttributeMatch := GetHtmlAttributeMatch(armPropertyName, htmlObjectCapture.Attribute, armPropertyValue)
				//fmt.Println("ARM ATTRIBUTE TO MATCH", armPropertyName)
				var blockNames []string
				for _, block := range htmlAttributeMatch {
					if block.Type == "armObject" {
						blockNames = append(blockNames, block.Name)
						rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(armPropertyValue, block, "", masterKey))
					}
				}
				for innerAttributeName, innerAttributeValue := range checkForMap {
					htmlInnerAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObjectCapture.Attribute, innerAttributeValue)
					var blockNamesInner []string
					for _, block := range htmlInnerAttributeMatch {
						if block.Type == "armObject" {
							blockNamesInner = append(blockNamesInner, block.Name)
							rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerAttributeValue, block, "", masterKey))
						}
					}

					if CheckForMap(innerAttributeValue) {
						for innerInnerAttributeName, innerInnerAttributeValue := range innerAttributeValue.(map[string]interface{}) {
							//fmt.Println("where the fuck are we", innerInnerAttributeName)
							htmlInnerInnerAttributeMatch := GetHtmlAttributeMatch(innerInnerAttributeName, htmlObjectCapture.Attribute, innerInnerAttributeValue)
							//fmt.Println("where the fuck are we", innerInnerAttributeName, htmlInnerInnerAttributeMatch)
							var blockNamesMinimum []string
							for _, block := range htmlInnerInnerAttributeMatch {
								if block.Type == "string" {
									switch innerInnerAttributeValue.(type) {
									case map[string]interface{}:
										{
											for minimumAttributeName, minimumAttributeValue := range innerInnerAttributeValue.(map[string]interface{}) {
												fmt.Println("THIS IS THE ATTRIBUTE:", minimumAttributeName)
												htmlMinimumAttributeMatch := GetHtmlAttributeMatch(minimumAttributeName, htmlObjectCapture.Attribute, minimumAttributeValue)
												if len(htmlMinimumAttributeMatch) > 0 {
													rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(minimumAttributeValue, htmlMinimumAttributeMatch[0], fmt.Sprintf("%s/%s", strings.Join(blockNames, "/"), strings.Join(blockNamesInner, "/")), masterKey))
												}
											}
										}
									case interface{}:
										{
											fmt.Println("I think we are missing some stuff here", innerInnerAttributeName)
										}
									}
									rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerInnerAttributeValue, block, fmt.Sprintf("%s/%s", strings.Join(blockNames, "/"), strings.Join(blockNamesInner, "/")), masterKey))
								} else {
									blockNamesMinimum = append(blockNamesMinimum, block.Name)
									rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerInnerAttributeValue, block, fmt.Sprintf("%s/%s", strings.Join(blockNamesInner, "/"), strings.Join(blockNames, "/")), masterKey))
								}
							}
							if CheckForMap(innerInnerAttributeValue) {
								for _, minimumAttributeValue := range innerInnerAttributeValue.(map[string]interface{}) {
									switch minimumAttributeValue.(type) {
									case []interface{}:
										{
											for _, slice := range minimumAttributeValue.([]interface{}) {
												for innerSliceAttributeName, innerSliceAttributeValue := range slice.(map[string]interface{}) {
													htmlInnerSliceAttributeMatch := GetHtmlAttributeMatch(innerSliceAttributeName, htmlObjectCapture.Attribute, innerSliceAttributeValue)
													if len(htmlInnerSliceAttributeMatch) > 0 {
														rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerSliceAttributeValue, htmlInnerSliceAttributeMatch[0], fmt.Sprintf("%s/%s/%s", strings.Join(blockNames, "/"), strings.Join(blockNamesInner, "/"), strings.Join(blockNamesMinimum, "/")), masterKey))
													}
												}
											}
										}
									}
								}
							}
						}

					} else {
						htmlAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObjectCapture.Attribute, innerAttributeValue)
						for _, attribute := range htmlAttributeMatch {
							CheckForMap(attribute)
							switch innerAttributeValue.(type) {
							case []interface{}:
								{
									for _, slice := range innerAttributeValue.([]interface{}) {
										checkForMap, ok := slice.(map[string]interface{})
										if ok {
											for _, innerSliceAttributeValue := range checkForMap {
												if CheckForMap(innerSliceAttributeValue) {
													for innerMapAttributeName, innerMapAttributeValue := range innerSliceAttributeValue.(map[string]interface{}) {
														htmlAttributeMatch := GetHtmlAttributeMatch(innerMapAttributeName, htmlObjectCapture.Attribute, innerMapAttributeValue)
														for _, match := range htmlAttributeMatch {
															rootAttributes = append(rootAttributes, ConvertMapToRootAttribute(innerMapAttributeName, innerMapAttributeValue, htmlAttributeMatch, match.Name, masterKey))
														}
													}
												}
											}
										} else {
											htmlAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObjectCapture.Attribute, innerAttributeValue)
											if len(htmlAttributeMatch) > 0 {
												rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(slice, htmlAttributeMatch[0], "", masterKey))
											}
										}
									}
								}
							case interface{}:
								{
									blockNesting := GetHtmlAttributeMatch(armPropertyName, htmlObjectCapture.Attribute, armPropertyValue)
									for _, block := range blockNesting {
										if block.Type == "armProperty" {
											blockNames = append(blockNames, block.Name)
										}

									}
									innerAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObjectCapture.Attribute, innerAttributeValue)
									if len(innerAttributeMatch) > 0 {
										rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerAttributeValue, innerAttributeMatch[0], strings.Join(blockNames, "/"), masterKey))
									}
								}
							}
						}
					}
				}
			}
		}
		GetBlocksFromRootAttributes(rootAttributes, htmlObjectCapture)
	}
	return rootAttributesForReturn
}

func GetBlocksFromRootAttributes(rootAttributes []RootAttribute, htmlObject HtmlObject) []BlockAttribute {
	var rootBlockAttribute BlockAttribute
	var forNestedRootAttributeBlock []RootAttribute
	var forRootAttributeBlock []RootAttribute
	var blocksForReturn []BlockAttribute
	var currentBlockNames []string
	var seenBlockNames []string
	var uniqueSeenBlockNames []string
	var summarizedRootAttributes []RootAttribute
	var removeDuplicateValues []string

	//fmt.Println("WE ARE HERE:", len(rootAttributes))
	/*
		for index, rootAttribute := range rootAttributes {
			fmt.Println(index, "ATTRIBUTE NAME:3333", rootAttribute.Name, "||", "BLOCK NAME", rootAttribute.BlockName, "||", "IS BLOCK", rootAttribute.IsBlock, "||", "UNIQUE NAME:", rootAttribute.UniqueBlockName, "VALUE:", rootAttribute.Value)
		}
	*/
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

	for _, block := range rootAttributes {
		if !strings.Contains(strings.Join(seenBlockNames, ","), block.BlockName) {
			partBlockNames := strings.Split(block.BlockName, "/")
			seenBlockNames = append(seenBlockNames, partBlockNames...)
		}

		if block.IsBlock {
			currentBlockNames = append(currentBlockNames, block.Name)
		}
	}

	for _, name := range seenBlockNames {
		if !strings.Contains(strings.Join(uniqueSeenBlockNames, ","), name) {
			uniqueSeenBlockNames = append(uniqueSeenBlockNames, name)
		}
	}

	/*
		for _, currentName := range currentBlockNames {
			blockNamesPart := strings.Split(currentName, "/")
			var newBlockName string
			var persistSeenName string
			var persistRootAttribute []string
			if len(blockNamesPart) > 0 {
				if strings.Contains(strings.Join(uniqueSeenBlockNames, ","), currentName) {
					if strings.Contains(strings.Join(uniqueSeenBlockNames, ","), "_") && strings.Contains(currentName, "_") || !strings.Contains(strings.Join(uniqueSeenBlockNames, ","), "_") && !strings.Contains(currentName, "_") {
						for _, seenName := range seenBlockNames {
							if strings.Contains(currentName, seenName) && currentName != seenName && !strings.Contains(seenName, "os") { //Not the best, but I need to move forward, keep an eye on this
								for _, rootAttribute := range rootAttributes {
									if strings.Contains(rootAttribute.BlockName, seenName) {
										blockNames := strings.Split(rootAttribute.BlockName, "/")
										for index, blockName := range blockNames {
											if blockName == seenName {
												newBlockName = blockNames[index-1]
												persistSeenName = seenName
												if !strings.Contains(strings.Join(persistRootAttribute, ","), rootAttribute.UniqueBlockName) {
													persistRootAttribute = append(persistRootAttribute, rootAttribute.UniqueBlockName)
												}
												break
											}
										}
									}
								}
								break
							}
						}
					}
				}
			}
			if newBlockName != "" {
				for _, uniqueBlockName := range persistRootAttribute {
					rootAttribute := RootAttribute{
						Name:            persistSeenName,
						Value:           nil,
						IsBlock:         true,
						BlockName:       newBlockName,
						UniqueBlockName: uniqueBlockName,
					}
					rootAttributes = append(rootAttributes, rootAttribute)
				}
			}
		}
	*/
	patternToMatchAddress := `^address_p(?:refix(?:es)?|aces)?$`
	regexPatternStructAddress := regexp.MustCompile(patternToMatchAddress)
	patternToMatchProfile := `(true|false|enabled|disabled|policies|(\d+)|(\d{1,3}(\.\d{1,3}){3}(\/\d{1,2})?))$`
	regexPatternStructProfile := regexp.MustCompile(patternToMatchProfile)

	//Sanatize data
	for index, rootAttribute := range rootAttributes {
		if !rootAttribute.IsBlock {
			if regexPatternStructAddress.MatchString(rootAttribute.Name) {
				if rootAttribute.BlockName == "" {
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
									fromSliceValues = append(fromSliceValues, value.(string), rootAttribute.UniqueBlockName, rootAttribute.Name)
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

	mapOfSummarizedRootAttributes := make(map[string]bool)

	for _, rootAttribute := range summarizedRootAttributes {
		mapOfSummarizedRootAttributes[fmt.Sprintf("%s/%s/%s", rootAttribute.Name, rootAttribute.UniqueBlockName, rootAttribute.BlockName)] = true
	}

	for _, rootAttribute := range rootAttributes {
		if found := mapOfSummarizedRootAttributes[fmt.Sprintf("%s/%s/%s", rootAttribute.Name, rootAttribute.UniqueBlockName, rootAttribute.BlockName)]; !found {
			summarizedRootAttributes = append(summarizedRootAttributes, rootAttribute)
			mapOfSummarizedRootAttributes[fmt.Sprintf("%s/%s/%s", rootAttribute.Name, rootAttribute.UniqueBlockName, rootAttribute.BlockName)] = true
		}
	}

	//Fix broken block names in case new root attributes has been created
	for _, rootAttribute := range summarizedRootAttributes {
		if rootAttribute.IsBlock {
			for _, attribute := range summarizedRootAttributes {
				if strings.Contains(attribute.BlockName, rootAttribute.Name) && !attribute.IsBlock {
					blockPartNames := strings.Split(attribute.BlockName, "/")
					if len(blockPartNames) > 2 {
						for index, innerAttribute := range summarizedRootAttributes {
							if blockPartNames[len(blockPartNames)-1] == innerAttribute.Name && innerAttribute.IsBlock {
								if !strings.Contains(innerAttribute.BlockName, blockPartNames[len(blockPartNames)-2]) {
									summarizedRootAttributes[index].BlockName = blockPartNames[len(blockPartNames)-2]
									if summarizedRootAttributes[index].UniqueBlockName == "" {

									}
									break
								}
							}
						}
					}
				}
				break
			}
		}
	}

	//Create root block Attributes
	for _, newBlockAttribute := range summarizedRootAttributes {
		if newBlockAttribute.IsBlock {
			for _, rootAttribute := range summarizedRootAttributes {
				if newBlockAttribute.BlockName == "" && rootAttribute.BlockName == "" {
					forRootAttributeBlock = append(forRootAttributeBlock, rootAttribute)
				}
			}
			if len(forRootAttributeBlock) > 0 {
				rootBlockAttribute = BlockAttribute{
					BlockName:     "root",
					RootAttribute: forRootAttributeBlock,
				}
				break
			}
		}
	}

	for _, newBlockAttribute := range rootBlockAttribute.RootAttribute {
		forRootAttributeBlock = []RootAttribute{}
		forNestedRootAttributeBlock = []RootAttribute{}
		var newBlockNames []string
		var blockPart []string
		for _, rootAttribute := range summarizedRootAttributes {
			if newBlockAttribute.Name != rootAttribute.Name {
				if newBlockAttribute.UniqueBlockName == "" {
					blockPart = strings.Split(rootAttribute.BlockName, "/")
					if len(blockPart) == 1 {
						if newBlockAttribute.Name == blockPart[0] {
							forRootAttributeBlock = append(forRootAttributeBlock, rootAttribute)
						}
					} else if len(blockPart) == 2 {
						if newBlockAttribute.Name == blockPart[0] {
							newBlockNames = blockPart
							forNestedRootAttributeBlock = append(forNestedRootAttributeBlock, rootAttribute)
						}
					}
				} else {
					if newBlockAttribute.Name != rootAttribute.Name && newBlockAttribute.UniqueBlockName == rootAttribute.UniqueBlockName {
						blockPart = strings.Split(rootAttribute.BlockName, "/")
						if len(blockPart) == 1 {
							if newBlockAttribute.Name == blockPart[0] {
								forRootAttributeBlock = append(forRootAttributeBlock, rootAttribute)
							}
						} else if len(blockPart) == 3 {
							if newBlockAttribute.Name == blockPart[0] {
								newBlockNames = append(append(newBlockNames, blockPart[1]), blockPart[2])
								forNestedRootAttributeBlock = append(forNestedRootAttributeBlock, rootAttribute)
							}
						}
					}
				}
			}
		}
		if len(forRootAttributeBlock) > 0 && len(blockPart) == 1 {

			blockAttributeNested := BlockAttribute{
				BlockName:     newBlockAttribute.Name,
				RootAttribute: forRootAttributeBlock,
				Parent:        "root",
			}
			blocksForReturn = append(blocksForReturn, blockAttributeNested)
		}

		if len(forNestedRootAttributeBlock) > 0 {
			blockAttributeNested := BlockAttribute{
				BlockName:     newBlockNames[1],
				RootAttribute: forNestedRootAttributeBlock,
				Parent:        newBlockNames[0],
			}
			blocksForReturn = append(blocksForReturn, blockAttributeNested)
		}
	}

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

	for index, rootAttribute := range summarizedRootAttributes {
		fmt.Println(index, "ATTRIBUTE NAME:", rootAttribute.Name, "||", "BLOCK NAME", rootAttribute.BlockName, "||", "IS BLOCK", rootAttribute.IsBlock, "||", "UNIQUE NAME:", rootAttribute.UniqueBlockName, "VALUE:", rootAttribute.Value)
	}
	return nil //compiledObjectsForReturn
}

func GetInnerRootAttributes(armPropertyName string, armPropertyValue interface{}, attributes []Attribute, masterKey string) []RootAttribute {
	var rootAttributes []RootAttribute
	var blockName string
	blockNames := GetHtmlAttributeMatch(armPropertyName, attributes, armPropertyValue)
	if blockNames != nil {
		if blockNames[0].Name != "properties" {
			blockName = blockNames[0].Name
		}
	}

	for attributeName, attributeValue := range armPropertyValue.(map[string]interface{}) {

		// Initialize persistValue with the current attributeValue
		persistValue := attributeValue
		for {
			// Try to cast persistValue to a map
			checkForMap, ok := persistValue.(map[string]interface{})
			if ok {
				for innerAttributeName, innerAttributeValue := range checkForMap {
					switch innerAttributeValue.(type) {
					case []interface{}:
						{
							for _, slice := range innerAttributeValue.([]interface{}) {
								for innerSliceAttributeName, innerSliceAttributeValue := range slice.(map[string]interface{}) {
									if CheckForMap(innerSliceAttributeValue) {
										innerBlockAttribute := GetHtmlAttributeMatch(innerAttributeName, attributes, innerAttributeValue)
										var innerBlockNames []string
										for _, block := range innerBlockAttribute {
											innerBlockNames = append(innerBlockNames, block.Name)
										}
										fmt.Println("TJOPS TIS THE:", innerSliceAttributeName, masterKey)
										rootAttributesPart := ConvertMapToRootAttribute(innerSliceAttributeName, innerSliceAttributeValue, attributes, fmt.Sprintf("%s/%s", blockName, strings.Join(innerBlockNames, "/")), masterKey)
										rootAttributes = append(rootAttributes, rootAttributesPart)

									} else if CheckForSlice(innerSliceAttributeValue) {
										for _, innerSlice := range innerSliceAttributeValue.([]interface{}) {
											innerMapCheck, ok := innerSlice.(map[string]interface{})
											if ok {
												for innerInnerSliceAttributeName, innerInnerSliceAttributeValue := range innerMapCheck {
													if CheckForMap(innerInnerSliceAttributeValue) {
														rootAttributesPart := ConvertMapToRootAttribute(innerInnerSliceAttributeName, innerSliceAttributeValue, attributes, blockName, masterKey)
														rootAttributes = append(rootAttributes, rootAttributesPart)
													}
												}
											} else {
												htmlAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, attributes, innerAttributeValue)
												var blockNames []string
												for _, block := range htmlAttributeMatch {
													if block.Type == "armObject" {
														blockNames = append(blockNames, block.Name)
													}
												}
												innerHtmlAttributeMatch := GetHtmlAttributeMatch(innerSliceAttributeName, attributes, innerSliceAttributeValue)
												if len(innerHtmlAttributeMatch) > 0 {
													rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerSliceAttributeValue, innerHtmlAttributeMatch[0], fmt.Sprintf("%s/%s", blockName, strings.Join(blockNames, "/")), masterKey))
												}
											}

										}
									} else {
										htmlAttributeMatch := GetHtmlAttributeMatch(innerSliceAttributeName, attributes, innerSliceAttributeValue)
										var attributesSorted []Attribute
										for _, attribute := range htmlAttributeMatch {
											if attribute.Type == "armObject" {
												attributesSorted = append(attributesSorted, attribute)
											}
										}

										innerBlockAttribute := GetHtmlAttributeMatch(armPropertyName, attributes, innerAttributeValue)
										var rootBlockNames []string

										for _, attribute := range innerBlockAttribute {
											if attribute.Type == "armObject" {
												rootBlockNames = append(rootBlockNames, attribute.Name)
											}
										}

										for _, attribute := range attributesSorted {
											rootAttribute := RootAttribute{
												Name:      attribute.Name,
												Value:     nil,
												BlockName: strings.Join(rootBlockNames, "/"),
												IsBlock:   true,
											}
											rootAttributes = append(rootAttributes, rootAttribute)
										}
										var innerBlockNames []string
										innerBlocks := GetHtmlAttributeMatch(innerAttributeName, attributes, innerAttributeValue)
										for _, block := range innerBlocks {
											if block.Type == "armObject" {
												innerBlockNames = append(innerBlockNames, block.Name)
											}
										}

										for _, attribute := range htmlAttributeMatch {
											if attribute.Type == "string" {
												rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerSliceAttributeValue, attribute, fmt.Sprintf("%s/%s", blockName, strings.Join(innerBlockNames, "/")), masterKey))
											}
										}
									}
								}
							}
						}
					case interface{}:
						{
							htmlAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, attributes, innerAttributeValue)
							for _, attribute := range htmlAttributeMatch {
								if blockName != "" {
									rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerAttributeValue, attribute, blockName, masterKey))
								}
							}
						}
					}
					persistValue = checkForMap //Makes sure that nesting continues
				}
				break

				// Update persistValue to go deeper into the map
			} else {
				htmlInnerAttributeMatch := GetHtmlAttributeMatch(attributeName, attributes, attributeValue)
				if len(htmlInnerAttributeMatch) > 0 {
					for _, rootAttribute := range htmlInnerAttributeMatch {
						rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(attributeValue, rootAttribute, blockName, masterKey))
					}
				}
			}
			break
		}
	}

	return rootAttributes
}

func ConvertMapToRootAttribute(armPropertyName string, armPropertyValue interface{}, attributes []Attribute, blockName string, masterKey string) RootAttribute {
	for attributeName, attributeValue := range armPropertyValue.(map[string]interface{}) {
		htmlAttribute := GetHtmlAttributeMatch(attributeName, attributes, attributeValue)

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

func GetHtmlAttributeMatch(armPropertyName string, htmlAttributes []Attribute, armPropertyValue interface{}) []Attribute {
	var htmlAttributeReturn []Attribute
	armPropertyNameConvert := ConvertArmAttributeName(armPropertyName, "")
	//fmt.Println("THIS IS THE ATTRIBUTE BEING MATCHED", "BEFORE:", armPropertyName, "AFTER:", armPropertyNameConvert)
	for _, htmlAttribute := range htmlAttributes {
		if strings.ToLower(armPropertyName) != "id" && strings.ToLower(htmlAttribute.Name) != "location" && strings.ToLower(htmlAttribute.Name) != "locations" {
			if strings.HasPrefix(htmlAttribute.Name, armPropertyNameConvert) {
				//fmt.Println("YEP WE ARE HERE:", armPropertyNameConvert, htmlAttribute.Name)
				if !strings.Contains(armPropertyNameConvert, "os_") && strings.Contains(htmlAttribute.Name, "_") && strings.Contains(armPropertyNameConvert, "_") || !strings.Contains(htmlAttribute.Name, "_") && !strings.Contains(armPropertyNameConvert, "_") && htmlAttribute.Name != "resource_group_name" && htmlAttribute.Name != "location" {
					htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
				} else if !strings.Contains(armPropertyNameConvert, "os_") && strings.Contains(htmlAttribute.Name, armPropertyNameConvert) && !strings.Contains(htmlAttribute.Name, "ids") { //This negative match will increase in size with experience
					//fmt.Println("WHAT DID WE JUST MATCH?", htmlAttribute.Name, armPropertyNameConvert)
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
								if strings.HasSuffix(htmlAttribute.Name, armPropertyInnerNameConvert) || htmlAttribute.Name == armPropertyInnerNameConvert {
									htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
									break
								}
							}
						}
					} else {
						if htmlAttribute.Name == armPropertyNameConvert {
							htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
						}
					}
				}

				if htmlAttribute.Name == armPropertyNameConvert {
					if htmlAttribute.Type == "armObject" {
						htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
					}
				}

				for _, attribute := range htmlAttributes {
					if strings.HasSuffix(attribute.Name, htmlAttribute.Name) && attribute.Type == "armObject" && attribute.Name != htmlAttribute.Name {
						htmlAttributeReturn = append(htmlAttributeReturn, attribute)
					}
				}
			} else if strings.HasSuffix(htmlAttribute.Name, armPropertyNameConvert) && (strings.Contains(htmlAttribute.Name, "_") && strings.Contains(armPropertyNameConvert, "_") || !strings.Contains(htmlAttribute.Name, "_") && !strings.Contains(armPropertyNameConvert, "_")) {
				htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
			}
		}
	}

	if len(htmlAttributeReturn) == 0 {
		blockPartName := strings.Split(armPropertyNameConvert, "_")
		for _, htmlAttribute := range htmlAttributes {
			if strings.Contains(htmlAttribute.Name, strings.Join(blockPartName[1:len(blockPartName)], "_")) {
				htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
				break
			}
		}
	}

	return htmlAttributeReturn
}

func CheckForMap(mapToCheck interface{}) bool {
	if _, ok := mapToCheck.(map[string]interface{}); ok {
		return ok //ok can only be true
	}
	return false
}

func CheckForSlice(sliceToCheck interface{}) bool {
	var ifError bool
	switch sliceToCheck.(type) {
	case []interface{}:
		{
			for _, test := range sliceToCheck.([]interface{}) {
				if test != nil {
					return true
				}
			}
		}
	case map[string]interface{}:
		{
			fmt.Errorf("The value provided is a map, please only provide slices or flat values\n", sliceToCheck)
		}
	}

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	return ifError
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

func NewCachedSystemFiles(HtmlObjects []HtmlObject) error {
	jsonData, err := json.MarshalIndent(HtmlObjects, "", "  ")
	if err != nil {
		fmt.Println("The following error occured when trying to convert htmlobjects to json:", err)
	}
	if _, err := os.Stat(strings.Split(SystemTerraformDocsFileName, "/")[1]); os.IsNotExist(err) {
		err = os.Mkdir(strings.Split(SystemTerraformDocsFileName, "/")[1], 0755)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}

	err = os.WriteFile(SystemTerraformDocsFileName, jsonData, 0644)
	if err != nil {
		return err
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
