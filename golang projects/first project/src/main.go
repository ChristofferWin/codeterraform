package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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
	Name                string `json:"name"`
	Resource_id         string `json:"id"`
	Resource_type       string `json:"type"`
	Location            string `json:"location"`
	Resource_group_name string
	Properties          interface{} `json:"properties"` // Use interface{} for dynamic properties
}

// Define the HtmlObject struct with a named attributes field
type HtmlObject struct {
	Resource_type string      `json:"Resource_type"`
	Version       string      `json:"Version"`
	Attribute     []Attribute `json:"Attribute"`
}

type RootAttribute struct {
	Name      string
	Value     interface{}
	BlockName string //Must be used with format <root name>/<level 1 object>/<level 2 object>
	IsBlock   bool
}

type Variable struct {
	Name         string
	Description  string
	DefaultValue string
}

type BlockAttribute struct {
	BlockName      string
	BlockAttribute interface{}
	RootAttribute  []RootAttribute
	BlockNumber    int //Determine the amount of blocks, e.g if an arm definition has a list of subnets with 2 subnets = 2 for the 'BlockNumber'
}

type CompileObject struct {
	ResourceDefinitionName string
	RootAttributes         []RootAttribute
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

	var resourceTypesFromArm []string
	var resourceTypesFromDif []string
	var resourceTypesToRetrieve []string

	for _, resourceType := range baseArmResources {
		resourceTypesFromArm = append(resourceTypesFromArm, resourceType.Resource_type)
	}

	var htmlObjectsFromCache []HtmlObject

	if !*noCache {
		htmlObjectsFromCache, err = GetCachedSystemFiles()
		if err != nil {
			fmt.Println("No cache detected, retrieving all required information...")
		}
		for _, htmlObjectFromCache := range htmlObjectsFromCache {
			resourceTypesFromDif = append(resourceTypesFromDif, htmlObjectFromCache.Resource_type)
		}

		for _, resourceTypeFromArm := range resourceTypesFromArm {
			if !strings.Contains(strings.Join(resourceTypesFromDif, ","), resourceTypeFromArm) {
				resourceTypesToRetrieve = append(resourceTypesToRetrieve, resourceTypeFromArm)
			}
		}

		resourceTypesToRetrieve = UniquifyResourceTypes(resourceTypesToRetrieve)
	} else {
		resourceTypesToRetrieve = UniquifyResourceTypes(resourceTypesFromArm)
	}

	for _, resourceType := range resourceTypesToRetrieve {
		rawHtml, err := GetRawHtml(resourceType, *providerVersion)

		if err != nil {
			fmt.Println("Error while trying to retrieve required documentation", err, resourceType)
			break
		}
		cleanHtml := SortRawHtml(rawHtml, resourceType)
		HtmlObjects = append(HtmlObjects, cleanHtml)
	}
	HtmlObjects = append(HtmlObjects, htmlObjectsFromCache...)
	if !*noCache {
		err = NewCachedSystemFiles(HtmlObjects)
		if err != nil {
			fmt.Println("An error occured while running function 'NewCachedSystemFiles'", err)
		}
	}

	test := GetRootAttributesToCompiledObjects(baseArmResources, HtmlObjects)
	for _, y := range test {
		fmt.Println("{\nNAME:", y.ResourceDefinitionName)
		for _, x := range y.RootAttributes {
			fmt.Println("ROOT NAME:", x.Name, "ROOT VALUE:", x.Value, "BLOCK?", x.IsBlock, "BLOCK NAME:", x.BlockName)
		}
	}
	GetBlocksToCompiledObjects(test)
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
				fileNames = append(fileNames, file.Name())
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
	var armCheck bool
	for _, fileContent := range filecontent {
		err := json.Unmarshal(fileContent, &jsonDump)
		if err != nil {
			continue
		} else {
			validJson = append(validJson, fileContent)
		}
	}

	for _, cleanContent := range validJson {
		json.Unmarshal(cleanContent, &jsonDump)
		testMap, ok := jsonDump.(map[string]interface{})
		if ok {
			for attributeName := range testMap {
				if attributeName == "properties" {
					armCheck = true
				}
			}
			if armCheck {
				cleanFilecontent = append(cleanFilecontent, cleanContent)
			}
		}
	}
	return cleanFilecontent
}

func GetArmBaseInformation(filecontent [][]byte) []ArmObject {
	var jsonInterface interface{}
	var armBasicObjects []ArmObject
	var jsonMap map[string]interface{}

	for _, bytes := range filecontent {
		err := json.Unmarshal(bytes, &jsonInterface)

		if err != nil {
			fmt.Println("Error while transforming file from bytes to json:", err)
		}

		switch v := jsonInterface.(type) {
		case map[string]interface{}:
			{
				jsonMap = jsonInterface.(map[string]interface{})
				armObject := ArmObject{
					Name:                jsonMap["name"].(string),
					Resource_id:         jsonMap["id"].(string),
					Resource_type:       jsonMap["type"].(string),
					Location:            jsonMap["location"].(string),
					Resource_group_name: strings.Split(jsonMap["id"].(string), "/")[4],
					Properties:          jsonMap["properties"],
				}
				armBasicObjects = append(armBasicObjects, armObject)

			}
		case []interface{}:
			{
				for _, item := range v {
					jsonMap = item.(map[string]interface{})
					armObject := ArmObject{
						Name:                jsonMap["name"].(string),
						Resource_id:         jsonMap["id"].(string),
						Resource_type:       jsonMap["type"].(string),
						Location:            jsonMap["location"].(string),
						Resource_group_name: strings.Split(jsonMap["id"].(string), "/")[4],
						Properties:          jsonMap["properties"],
					}
					armBasicObjects = append(armBasicObjects, armObject)
				}
			}
		}
	}

	return armBasicObjects
}

func ConvertArmAttributeName(resourceType string) string {
	var resourceTypeRegex *regexp.Regexp
	var convertResourceTypeName string
	// Use a regex to find places where we need to insert underscores
	regexToMatchAttributeNames := regexp.MustCompile("([A-Z])")

	//Define regex so that we can differentiate between resource types 'Something/Something' And Something/Something/Somthing
	if len(strings.Split(resourceType, "/")) == 2 {
		resourceTypeRegex = regexp.MustCompile("([a-z0-9])([A-Z])")
		convertResourceTypeName = resourceTypeRegex.ReplaceAllString(resourceType, "${1}_${2}")
		convertResourceTypeName = strings.ToLower(strings.Split(convertResourceTypeName, "/")[1])
	} else if regexToMatchAttributeNames.MatchString(resourceType) && len(strings.Split(resourceType, "/")) < 2 {
		convertResourceTypeName = regexToMatchAttributeNames.ReplaceAllString(resourceType, "_$1")
	} else {
		sliceArray := strings.Split(resourceType, "/")
		convertResourceTypeName = strings.ToLower(sliceArray[len(sliceArray)-1])
	}
	convertResourceTypeName = strings.ToLower(strings.TrimSuffix(convertResourceTypeName, "s"))
	return convertResourceTypeName
}

func GetRawHtml(resourceType string, providerVersion string) (string, error) {
	var HtmlBody string
	var HtmlBodyCompare string
	convertResourceTypeName := ConvertArmAttributeName(resourceType)

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

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

		//os.WriteFile("ByQuery2", []byte(HtmlBodyCompare), 0644)
	}
	return HtmlBodyCompare, nil
}

func SortRawHtml(rawHtml string, resourceType string) HtmlObject { //See the struct type definitions towards the top of the file
	var allAttributes []string
	var flatAttributes []string //Either bool, string, int or string array, must be determined by the ARM values
	var blockAttribute []string //Object
	var uniqueAttributeNames []string
	var uniqueAttributes []Attribute

	//Defining the regex matching ALL attributes, which we will sort through first, then seperate on type - armObject vs string
	allAttributesRegex := regexp.MustCompile(`\(Required|Optional\)`)
	//Defining the regex pattern to only match block definitions - The right side finds edgecases, e.g there might be more ways for Hashicorp to define blocks
	oneBlockRegex := regexp.MustCompile(`(?:An|A|One or more|) <code>([^<]+)</code> block|(Can be specified multiple times.)*?<code>([^<]+)</code> block`)
	//Defining the regex pattern which will be used to retrieve all the 'flat' level arguments
	isolateAttributesRegex := regexp.MustCompile(`href="#([^"]+)"`)

	//Defining the boundaries of the data we are interested in
	startIndex := regexp.MustCompile(`name="argument-reference"`).FindStringIndex(rawHtml)
	endIndex := regexp.MustCompile(`name="attributes-reference"`).FindStringIndex(rawHtml)

	//Isolating only the 'argument references from the HTML dump'
	extractedText := rawHtml[startIndex[1]:endIndex[0]]

	linesHtml := strings.Split(extractedText, "\n")

	//Filter lines containing (Required) or (Optional), retrieving ALL arguments regardless of type
	for _, line := range linesHtml {
		if allAttributesRegex.MatchString(line) {
			allAttributes = append(allAttributes, line)
		}
	}
	//We now seperate each of the 2 types into seperate slices
	for _, line := range allAttributes {
		if oneBlockRegex.MatchString(line) {
			blockAttribute = append(blockAttribute, line)
		} else {
			flatAttributes = append(flatAttributes, line)
		}
	}
	//For all the type 'block' Add them to the overall attribute slices
	for _, line := range blockAttribute {
		{
			attribute := Attribute{
				Type: "armObject",
				Name: oneBlockRegex.FindStringSubmatch(line)[1],
			}
			AttributeObjects = append(AttributeObjects, attribute)
		}
	}
	//For all the type 'string' Add them to the overall attribute slices
	for _, line := range flatAttributes {
		attribute := Attribute{
			Type: "string",
			Name: isolateAttributesRegex.FindStringSubmatch(line)[1],
		}
		AttributeObjects = append(AttributeObjects, attribute)
	}

	for _, uniqueAttribute := range AttributeObjects {
		if !strings.Contains(strings.Join(uniqueAttributeNames, ","), uniqueAttribute.Name) {
			uniqueAttributeNames = append(uniqueAttributeNames, uniqueAttribute.Name)
			uniqueAttributes = append(uniqueAttributes, uniqueAttribute)
		}
	}
	//Adding all the sorted attributes to the final return armObject
	htmlObject := HtmlObject{
		Resource_type: resourceType,
		Attribute:     uniqueAttributes,
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

func GetRootAttributesToCompiledObjects(armBasicObjects []ArmObject, HtmlObjects []HtmlObject) []CompileObject {
	var compiledObjects []CompileObject
	var htmlObjectCapture HtmlObject
	for _, armBasicObject := range armBasicObjects {
		var rootAttributes []RootAttribute
		var rootAttributesFromReturn []RootAttribute
		for _, htmlObject := range HtmlObjects {
			if htmlObject.Resource_type == armBasicObject.Resource_type {
				htmlObjectCapture = htmlObject
				break
			}
		}

		for armPropertyName, armPropertyValue := range armBasicObject.Properties.(map[string]interface{}) {
			checkForMap, ok := armPropertyValue.(map[string]interface{})

			if !ok {
				switch armPropertyValue.(type) {
				case []interface{}:
					{
						for _, innerPropertyValue := range armPropertyValue.([]interface{}) {
							_, ok := innerPropertyValue.(map[string]interface{})
							if ok {
								rootAttributesFromReturn = GetRootAttributes(armPropertyName, innerPropertyValue, htmlObjectCapture.Attribute)
								rootAttributes = append(rootAttributes, rootAttributesFromReturn...)
							}
						}

					}
				}
			} else {
				for innerAttributeName, innerAttributeValue := range checkForMap {
					if CheckForMap(innerAttributeValue) {
						rootAttributesFromReturn = GetRootAttributes(innerAttributeName, innerAttributeValue, htmlObjectCapture.Attribute)
						rootAttributes = append(rootAttributes, rootAttributesFromReturn...)
					} else {
						htmlAttributeMatch := GeHtmlAttributeMatch(innerAttributeName, htmlObjectCapture.Attribute)
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
														htmlAttributeMatch := GeHtmlAttributeMatch(innerMapAttributeName, htmlObjectCapture.Attribute)
														for _, match := range htmlAttributeMatch {
															rootAttributes = append(rootAttributes, ConvertMapToRootAttribute(innerMapAttributeName, innerMapAttributeValue, htmlAttributeMatch, match.Name))
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
		compiledObject := CompileObject{
			ResourceDefinitionName: fmt.Sprintf("azurerm_%s", ConvertArmAttributeName(htmlObjectCapture.Resource_type)),
			RootAttributes:         rootAttributes,
			Variables:              nil,
			BlockAttributes:        nil,
		}
		compiledObjects = append(compiledObjects, compiledObject)
	}
	return compiledObjects
}

func GetBlocksToCompiledObjects(compiledObjects []CompileObject) []CompileObject {
	var compiledObjectsForReturn []CompileObject
	for _, object := range compiledObjects {
		for _, block := range object.RootAttributes {
			split := strings.Split(block.BlockName, "/")
			fmt.Println(split)
		}
	}
	return compiledObjectsForReturn
}

func GetRootAttributes(armPropertyName string, armPropertyValue interface{}, attributes []Attribute) []RootAttribute {
	var rootAttributes []RootAttribute
	var blockName string
	blockNames := GeHtmlAttributeMatch(armPropertyName, attributes)
	if blockNames != nil {
		if blockNames[0].Name != "properties" {
			blockName = blockNames[0].Name
		}
	}
	if test := GeHtmlAttributeMatch(armPropertyName, attributes); len(test) > 1 {
		fmt.Println(test, "GATEWAY KEEPER") //GATEKEEPER !
	}

	for _, attributeValue := range armPropertyValue.(map[string]interface{}) {

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
										innerBlockAttribute := GeHtmlAttributeMatch(innerAttributeName, attributes)
										var innerBlockNames []string
										for _, block := range innerBlockAttribute {
											innerBlockNames = append(innerBlockNames, block.Name)
										}
										//fmt.Println(innerBlockNames)
										rootAttributesPart := ConvertMapToRootAttribute(innerSliceAttributeName, innerSliceAttributeValue, attributes, fmt.Sprintf("%s/%s", blockName, strings.Join(innerBlockNames, "/")))
										rootAttributes = append(rootAttributes, rootAttributesPart)

									} else if CheckForSlice(innerSliceAttributeValue) {
										for _, innerSlice := range innerSliceAttributeValue.([]interface{}) {
											innerMapCheck, ok := innerSlice.(map[string]interface{})
											if ok {
												for innerInnerSliceAttributeName, innerInnerSliceAttributeValue := range innerMapCheck {
													if CheckForMap(innerInnerSliceAttributeValue) {
														//fmt.Println("WE ARE HERE:", innerInnerSliceAttributeName)
														rootAttributesPart := ConvertMapToRootAttribute(innerInnerSliceAttributeName, innerSliceAttributeValue, attributes, blockName)
														rootAttributes = append(rootAttributes, rootAttributesPart)
													}
												}
											}

										}
									} else {
										//fmt.Println("ATTRIBUTE NAME:", innerSliceAttributeName)
										htmlAttributeMatch := GeHtmlAttributeMatch(innerSliceAttributeName, attributes)
										var blockNames []string
										var attributesSorted []Attribute
										for _, attribute := range htmlAttributeMatch {
											if attribute.Type == "armObject" {
												attributesSorted = append(attributesSorted, attribute)
											}
										}
										for _, attribute := range attributesSorted {
											blockNames = append(blockNames, attribute.Name)
										}

										for _, attribute := range attributesSorted {
											rootAttribute := RootAttribute{
												Name:      attribute.Name,
												Value:     innerSliceAttributeValue,
												BlockName: fmt.Sprintf("%s/%s", blockName, strings.Join(blockNames, "/")),
												IsBlock:   true,
											}
											rootAttributes = append(rootAttributes, rootAttribute)
										}
										var innerBlockNames []string
										innerBlocks := GeHtmlAttributeMatch(innerAttributeName, attributes)
										for _, block := range innerBlocks {
											innerBlockNames = append(innerBlockNames, block.Name)
										}

										for _, attribute := range htmlAttributeMatch {
											if attribute.Type == "string" {
												rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerSliceAttributeValue, attribute, fmt.Sprintf("%s/%s", blockName, strings.Join(innerBlockNames, "/"))))
											}
										}
									}
								}
							}
						}
					case interface{}:
						{
							htmlAttributeMatch := GeHtmlAttributeMatch(innerAttributeName, attributes)
							for _, attribute := range htmlAttributeMatch {
								rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerAttributeValue, attribute, blockName))
							}
						}
					}
					//rootAttributesPart := ConvertMapToRootAttribute(innerAttributeName, innerAttributeValue, attributes, blockName.Name)
					//rootAttributes = append(rootAttributes, rootAttributesPart)
					persistValue = checkForMap //Makes sure that nesting continues
				}
				break

				// Update persistValue to go deeper into the map
			}
			break
		}
	}

	/*
		if htmlAttribute := GeHtmlAttributeMatch(attributeName, attributes); htmlAttribute != (Attribute{}) {
								rootAttributesPart := ConvertFlatValueToRootAttribute(persistValue, htmlAttribute, blockName.Name)
								rootAttributes = append(rootAttributes, rootAttributesPart)
								break
							}
	*/

	return rootAttributes
}

func ConvertMapToRootAttribute(armPropertyName string, armPropertyValue interface{}, attributes []Attribute, blockName string) RootAttribute {
	for attributeName, attributeValue := range armPropertyValue.(map[string]interface{}) {
		htmlAttribute := GeHtmlAttributeMatch(attributeName, attributes)

		for _, attribute := range htmlAttribute {
			if attribute != (Attribute{}) {
				if attribute.Type == "armObject" {
					rootAttribute := RootAttribute{
						Name:      attribute.Name,
						Value:     attributeValue,
						BlockName: blockName,
						IsBlock:   true,
					}
					return rootAttribute
				} else {
					rootAttribute := RootAttribute{
						Name:      attribute.Name,
						Value:     attributeValue,
						BlockName: blockName,
						IsBlock:   false,
					}
					return rootAttribute
				}

			}
		}
	}
	return (RootAttribute{})
}

func ConvertFlatValueToRootAttribute(armPropertyValue interface{}, attribute Attribute, blockName string) RootAttribute {
	/*	if armPropertyValue == "Disabled" {
			rootAttribute := RootAttribute{
				Name:      attribute.Name,
				Value:     false,
				BlockName: blockName,
				IsBlock:   false,
			}
			return rootAttribute
		} else if armPropertyValue == "Enabled" {
			rootAttribute := RootAttribute{
				Name:      attribute.Name,
				Value:     true,
				BlockName: blockName,
				IsBlock:   false,
			}
			return rootAttribute
		}
	*/
	rootAttribute := RootAttribute{
		Name:      attribute.Name,
		Value:     armPropertyValue,
		BlockName: blockName,
		IsBlock:   false,
	}
	return rootAttribute
}

func GeHtmlAttributeMatch(armPropertyName string, htmlAttributes []Attribute) []Attribute {
	var htmlAttributeReturn []Attribute
	armPropertyNameConvert := ConvertArmAttributeName(armPropertyName)
	for _, htmlAttribute := range htmlAttributes {
		if strings.ToLower(htmlAttribute.Name) != "id" {
			if strings.Contains(armPropertyNameConvert, "windows") || strings.Contains(armPropertyNameConvert, "linux") {
				if strings.Contains(armPropertyNameConvert, "windows") {
					htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
					return htmlAttributeReturn
				} else if strings.Contains(armPropertyNameConvert, "linux") {
					htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
					return htmlAttributeReturn
				}
			}
			if strings.HasPrefix(htmlAttribute.Name, armPropertyNameConvert) {
				if strings.Contains(htmlAttribute.Name, "_") && strings.Contains(armPropertyNameConvert, "_") || !strings.Contains(htmlAttribute.Name, "_") && !strings.Contains(armPropertyNameConvert, "_") {
					htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
				}
				for _, attribute := range htmlAttributes {
					if strings.HasSuffix(attribute.Name, htmlAttribute.Name) && attribute.Type == "armObject" && attribute.Name != htmlAttribute.Name {
						//fmt.Println("MATCHED:", attribute.Name, htmlAttribute.Name)
						htmlAttributeReturn = append(htmlAttributeReturn, attribute)
					}
				}
			} else if strings.Contains(htmlAttribute.Name, armPropertyNameConvert) {

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
	jsonData, err := json.Marshal(HtmlObjects)
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
	var HtmlObjects []HtmlObject

	file, err := os.ReadFile(SystemTerraformDocsFileName)
	if err != nil {
		return nil, err
	}
	extractJson := json.Unmarshal(file, &HtmlObjects)
	if extractJson != nil {
		return nil, err
	}

	return HtmlObjects, nil
}

//func SortResourceTypes()
