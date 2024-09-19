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

	test := GetRootAttributes(baseArmResources, HtmlObjects)
	for _, tes2t := range test {
		fmt.Println("YEP, IT WORKS222", tes2t)
	}
	//GetBlocksFromRootAttributes(test)
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
		fmt.Println("\n", uniqueAttribute.Name)
		if !strings.Contains(strings.Join(uniqueAttributeNames, ","), uniqueAttribute.Name) {
			uniqueAttributeNames = append(uniqueAttributeNames, uniqueAttribute.Name)
			uniqueAttributes = append(uniqueAttributes, uniqueAttribute)
		}
	}

	for _, attribute := range AttributeObjects {
		found := false
		for _, uniqueAttribute := range uniqueAttributes {
			if attribute.Name == uniqueAttribute.Name {
				found = true
				break
			}
		}

		// If no match was found, add the attribute to uniqueAttributes
		if !found {
			uniqueAttributes = append(uniqueAttributes, attribute)
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

func GetRootAttributes(armBasicObjects []ArmObject, HtmlObjects []HtmlObject) []RootAttribute {
	var rootAttributesForReturn []RootAttribute
	var htmlObjectCapture HtmlObject
	var masterKey string
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
							masterKey = ""
							for innerInnerAttributeName, innerInnerAttributeValue := range innerPropertyValue.(map[string]interface{}) {
								if innerInnerAttributeName == "name" {
									if len(GetHtmlAttributeMatch(armPropertyName, htmlObjectCapture.Attribute, armPropertyValue)) > 0 {
										masterKey = innerInnerAttributeValue.(string)
										break
									}
								}
							}
							fmt.Println("WE ARE HERE BOIS", armPropertyName)
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
				default:
					{
						htmlAttributeMatch := GetHtmlAttributeMatch(armPropertyName, htmlObjectCapture.Attribute, armPropertyValue)
						fmt.Println("ARM 222", armPropertyName)
						if len(htmlAttributeMatch) > 0 {
							//fmt.Println(htmlAttributeMatch, "HERE WE ARE")
							//rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(armPropertyValue, htmlAttributeMatch[0]))
						}

					}
				}
			} else {
				htmlAttributeMatch := GetHtmlAttributeMatch(armPropertyName, htmlObjectCapture.Attribute, armPropertyValue)
				var blockNames []string
				for _, block := range htmlAttributeMatch {
					if block.Type == "armObject" {
						fmt.Println("MATCHED BLOCK NAME:", block.Name)
						blockNames = append(blockNames, block.Name)
						rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(armPropertyValue, block, "", masterKey))
					}
				}
				for innerAttributeName, innerAttributeValue := range checkForMap {
					fmt.Println("ATT NAME:", innerAttributeName)
					htmlInnerAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObjectCapture.Attribute, innerAttributeValue)
					var blockNamesInner []string
					for _, block := range htmlInnerAttributeMatch {
						if block.Type == "armObject" {
							fmt.Println("BLOCK NAMEW 333", block.Name)
							blockNamesInner = append(blockNamesInner, block.Name)
							rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerAttributeValue, block, "", masterKey))
						}
					}

					if CheckForMap(innerAttributeValue) {
						for innerInnerAttributeName, innerInnerAttributeValue := range innerAttributeValue.(map[string]interface{}) {
							fmt.Println("HERE WE ARE MOTHERFUCJERS,m", innerInnerAttributeName)
							htmlInnerInnerAttributeMatch := GetHtmlAttributeMatch(innerInnerAttributeName, htmlObjectCapture.Attribute, innerInnerAttributeValue)
							fmt.Println("WE MATCHED THIS:", htmlInnerInnerAttributeMatch)
							var blockNamesMinimum []string
							for _, block := range htmlInnerInnerAttributeMatch {
								if block.Type == "string" {
									switch innerInnerAttributeValue.(type) {
									case map[string]interface{}:
										{
											for minimumAttributeName, minimumAttributeValue := range innerInnerAttributeValue.(map[string]interface{}) {
												fmt.Println("MINIMUM ATTRIBUTE NAME:", minimumAttributeName, "VALUE:", minimumAttributeValue, "BLOCK NAMES:", blockNamesInner)
												htmlMinimumAttributeMatch := GetHtmlAttributeMatch(minimumAttributeName, htmlObjectCapture.Attribute, minimumAttributeValue)
												if len(htmlMinimumAttributeMatch) > 0 {
													fmt.Println("MATCHED: 2222", htmlMinimumAttributeMatch)
													rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(minimumAttributeValue, htmlMinimumAttributeMatch[0], fmt.Sprintf("%s/%s", strings.Join(blockNames, "/"), strings.Join(blockNamesInner, "/")), masterKey))
												}
											}
										}
									}
									rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerInnerAttributeValue, block, fmt.Sprintf("%s/%s", strings.Join(blockNames, "/"), strings.Join(blockNamesInner, "/")), masterKey))
								} else {
									blockNamesMinimum = append(blockNamesMinimum, block.Name)
									rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerInnerAttributeValue, block, fmt.Sprintf("%s/%s", strings.Join(blockNamesInner, "/"), strings.Join(blockNames, "/")), masterKey))
								}
							}
							if CheckForMap(innerInnerAttributeValue) {
								for minimumAttributeName, minimumAttributeValue := range innerInnerAttributeValue.(map[string]interface{}) {
									switch minimumAttributeValue.(type) {
									case []interface{}:
										{
											for _, slice := range minimumAttributeValue.([]interface{}) {
												for innerSliceAttributeName, innerSliceAttributeValue := range slice.(map[string]interface{}) {
													htmlInnerSliceAttributeMatch := GetHtmlAttributeMatch(innerSliceAttributeName, htmlObjectCapture.Attribute, innerSliceAttributeValue)
													if len(htmlInnerSliceAttributeMatch) > 0 {
														fmt.Println("HELLO WORLD:22", htmlInnerSliceAttributeMatch[0])
														rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerSliceAttributeValue, htmlInnerSliceAttributeMatch[0], fmt.Sprintf("%s/%s/%s", strings.Join(blockNames, "/"), strings.Join(blockNamesInner, "/"), strings.Join(blockNamesMinimum, "/")), masterKey))
													}
												}
											}
										}
									default:
										{
											htmlMinimumAttributeMatch := GetHtmlAttributeMatch(minimumAttributeName, htmlObjectCapture.Attribute, minimumAttributeValue)
											fmt.Println("MATCHEDED:", htmlMinimumAttributeMatch)
										}
									}
									//fmt.Println("MOST INNER MATCH:", minimumAttributeMatch, "INNER PROB", minimumAttributeName, "OUTER", innerInnerAttributeName, "ROOT", armPropertyName)
								}
							}

							/*
								fmt.Println("Inner Attribute name:", innerInnerAttributeName)
								fmt.Println("htmlAttributeMatch", htmlAttributeMatch)
								htmlAttributeMatchInner := GetHtmlAttributeMatch(innerInnerAttributeName, htmlObjectCapture.Attribute, innerInnerAttributeValue)
								fmt.Println("THIS IS THE MATCH:", htmlAttributeMatchInner)
								var blockNames []string
								for _, block := range htmlInnerAttributeMatch {
									if block.Type == "armObject" {
										fmt.Println("THIS IS THE BLOCK NAME:", block.Name)
										blockNames = append(blockNames, block.Name)
									}
								}
								if len(htmlAttributeMatchInner) > 0 {
									fmt.Println("WE ARE HERE", htmlAttributeMatchInner[0].Name)
									rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerInnerAttributeValue, htmlAttributeMatchInner[0], strings.Join(blockNames, "/")))
								}
							*/

						}

					} else {
						htmlAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObjectCapture.Attribute, innerAttributeValue)
						for _, attribute := range htmlAttributeMatch {
							CheckForMap(attribute)
							switch innerAttributeValue.(type) {
							case []interface{}:
								{
									for _, slice := range innerAttributeValue.([]interface{}) {
										fmt.Println("SLICE:", slice)
										checkForMap, ok := slice.(map[string]interface{})
										if ok {
											for _, innerSliceAttributeValue := range checkForMap {
												fmt.Println("VALUES BABY", innerSliceAttributeValue)
												if CheckForMap(innerSliceAttributeValue) {
													for innerMapAttributeName, innerMapAttributeValue := range innerSliceAttributeValue.(map[string]interface{}) {
														fmt.Println(innerMapAttributeName, innerMapAttributeName, "-------")
														htmlAttributeMatch := GetHtmlAttributeMatch(innerMapAttributeName, htmlObjectCapture.Attribute, innerMapAttributeValue)
														for _, match := range htmlAttributeMatch {
															rootAttributes = append(rootAttributes, ConvertMapToRootAttribute(innerMapAttributeName, innerMapAttributeValue, htmlAttributeMatch, match.Name, masterKey))
														}
													}
												}
											}
										} else {
											htmlAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObjectCapture.Attribute, innerAttributeValue)
											//fmt.Println("innerAttributeName", innerAttributeName)
											if len(htmlAttributeMatch) > 0 {
												rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(slice, htmlAttributeMatch[0], "", masterKey))
											}
										}
									}
								}
							case interface{}:
								{
									fmt.Println("\nINNER ATTRIBUTE NAME:", innerAttributeName, "ARM PROPERTY:", armPropertyName)
									blockNesting := GetHtmlAttributeMatch(armPropertyName, htmlObjectCapture.Attribute, armPropertyValue)
									for _, block := range blockNesting {
										if block.Type == "armProperty" {
											blockNames = append(blockNames, block.Name)
										}

									}
									fmt.Println("WE ARE HERE222", blockNesting, armPropertyName, innerAttributeName)
									innerAttributeMatch := GetHtmlAttributeMatch(innerAttributeName, htmlObjectCapture.Attribute, innerAttributeValue)
									if len(innerAttributeMatch) > 0 {
										fmt.Println("DO WE GET DOWN HERE?=", innerAttributeMatch, innerAttributeName, "BLOCKS", blockNames)
										rootAttributes = append(rootAttributes, ConvertFlatValueToRootAttribute(innerAttributeValue, innerAttributeMatch[0], strings.Join(blockNames, "/"), masterKey))
									}
								}
							}
						}
					}
				}
			}
		}
		GetBlocksFromRootAttributes(rootAttributes)
	}
	return rootAttributesForReturn
}

func GetBlocksFromRootAttributes(rootAttributes []RootAttribute) []BlockAttribute {
	var blocksForReturn []BlockAttribute
	var currentBlockNames []string
	//var uniqueCurrentBlockNames []string
	var seenBlockNames []string
	var uniqueSeenBlockNames []string
	var missingBlockNames []string
	var uniqueMissingBlockNames []string
	var newBlockName string
	//var match bool
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
			fmt.Println("SEEN BLOCK:", name)
			uniqueSeenBlockNames = append(uniqueSeenBlockNames, name)
		}
	}

	for _, currentName := range currentBlockNames {
		blockNamesPart := strings.Split(currentName, "/")
		var newBlockName string
		if len(blockNamesPart) > 0 {
			if strings.Contains(strings.Join(uniqueSeenBlockNames, ","), currentName) {
				if strings.Contains(strings.Join(uniqueSeenBlockNames, ","), "_") && strings.Contains(currentName, "_") || !strings.Contains(strings.Join(uniqueSeenBlockNames, ","), "_") && !strings.Contains(currentName, "_") {
					fmt.Println("WE ARE HERE BABY::", currentName)
					for _, seenName := range seenBlockNames {
						if strings.Contains(currentName, seenName) && currentName != seenName && !strings.Contains(seenName, "os") { //Not the best, but I need to move forward, keep an eye on this
							for _, rootAttribute := range rootAttributes {
								if strings.Contains(rootAttribute.BlockName, seenName) {
									match := false
									blockNames := strings.Split(rootAttribute.BlockName, "/")
									for index, blockName := range blockNames {
										if blockName == seenName {
											newBlockName = blockNames[index-1]
											fmt.Println("NEW BLOCK NAME", newBlockName)
											match = true
											break //We are missing some breaks - It finds subnet 6 times, should only find it 4 (I think)
										}
										if match {
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
	}

	for _, name := range missingBlockNames {
		match := false
		for _, rootAttribute := range rootAttributes {
			if !rootAttribute.IsBlock && rootAttribute.BlockName != "" {
				blockNameParts := strings.Split(rootAttribute.BlockName, "/")
				for _, namePart := range blockNameParts {
					if name == namePart {
						match = true
						break
					}
				}
				if !match {
					fmt.Println("THIS IS STILL MISSING:", name)
					break
				}
			}

		}
	}

	for _, name := range uniqueMissingBlockNames {
		var newBlockNames []string
		for _, rootAttribute := range rootAttributes {
			if strings.Contains(rootAttribute.BlockName, name) {
				fmt.Println("MATCHED 2:", name)
				blockNames := strings.Split(rootAttribute.BlockName, "/")
				counter := 0
				for _, blockName := range blockNames {
					counter++
					if blockName == name {
						newBlockNames = blockNames[:counter-1]
						break
					}
				}
			}
		}
		newBlockName = strings.Join(newBlockNames, "/")
		rootAttribute := RootAttribute{
			Name:      name,
			Value:     nil,
			BlockName: newBlockName,
			IsBlock:   true,
		}
		rootAttributes = append(rootAttributes, rootAttribute)
	}

	patternToMatch := `^address_p(?:refix(?:es)?|aces)?$`
	regexPatternStruct := regexp.MustCompile(patternToMatch)

	//Sanatize data
	for index, rootAttribute := range rootAttributes {
		if !rootAttribute.IsBlock {
			if regexPatternStruct.MatchString(rootAttribute.Name) {
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
		}
	}
	/*
		//Creating each block attribute and gathering all nested root attributes
		for _, newBlock := range rootAttributes {

		}
	*/
	for _, newBlock := range rootAttributes {
		if newBlock.IsBlock {

		}
	}
	/*
		for _, nameCurrentBlock := range uniqueCurrentBlockNames {
			for _, nameSeenBlock := range uniqueSeenBlockNames {
				if
			}
		}
	*/

	fmt.Println("--------------------------------- THIS IS A NEW OBJECT ---------------------------------")

	for _, attribute := range rootAttributes {
		fmt.Println("\nattribute name:", attribute.Name, "||", "block name:", attribute.BlockName, "||", "unique block name:", attribute.UniqueBlockName, "||", "is block:", attribute.IsBlock, "||", "value:", attribute.Value)
	}
	for _, block := range blocksForReturn {
		fmt.Println("\nBLOCK NAME:", block.BlockName, "PARENT:", block.Parent, "--------------------------------------")
		for _, attribute := range block.RootAttribute {
			fmt.Println("ATTRIBUTE NAME:", attribute.Name, "||", "INNER BLOCK:", attribute.BlockName, "||", "TYPE", attribute.IsBlock)
		}
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
		fmt.Println("THIS IS THE BLOCK NAME MATE;", blockName, armPropertyName)
	}
	if test := GetHtmlAttributeMatch(armPropertyName, attributes, armPropertyValue); len(test) > 1 {
		fmt.Println(test, "GATEWAY KEEPER") //GATEKEEPER !
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
										//fmt.Println("BLOCK NAME:", innerBlockNames)
										rootAttributesPart := ConvertMapToRootAttribute(innerSliceAttributeName, innerSliceAttributeValue, attributes, fmt.Sprintf("%s/%s", blockName, strings.Join(innerBlockNames, "/")), masterKey)
										rootAttributes = append(rootAttributes, rootAttributesPart)

									} else if CheckForSlice(innerSliceAttributeValue) {
										for _, innerSlice := range innerSliceAttributeValue.([]interface{}) {
											innerMapCheck, ok := innerSlice.(map[string]interface{})
											if ok {
												for innerInnerSliceAttributeName, innerInnerSliceAttributeValue := range innerMapCheck {
													if CheckForMap(innerInnerSliceAttributeValue) {
														//fmt.Println("WE ARE HERE:", innerInnerSliceAttributeName)
														rootAttributesPart := ConvertMapToRootAttribute(innerInnerSliceAttributeName, innerSliceAttributeValue, attributes, blockName, masterKey)
														rootAttributes = append(rootAttributes, rootAttributesPart)
													} else {
														fmt.Println("MOTHER FUCKERES")
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
					default:
						{
							fmt.Println("HELLO WORLD")
						}
					}
					//rootAttributesPart := ConvertMapToRootAttribute(innerAttributeName, innerAttributeValue, attributes, blockName.Name)
					//rootAttributes = append(rootAttributes, rootAttributesPart)
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

	/*
		if htmlAttribute := GetHtmlAttributeMatch(attributeName, attributes); htmlAttribute != (Attribute{}) {
								rootAttributesPart := ConvertFlatValueToRootAttribute(persistValue, htmlAttribute, blockName.Name)
								rootAttributes = append(rootAttributes, rootAttributesPart)
								break
							}
	*/

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
	armPropertyNameConvert := ConvertArmAttributeName(armPropertyName)
	//fmt.Println("\nARM PROPERTY NAME TO MATCH:", armPropertyNameConvert)
	for _, htmlAttribute := range htmlAttributes { //boot_diagnostic //boot_diagnostics
		if strings.ToLower(armPropertyName) != "id" && strings.ToLower(htmlAttribute.Name) != "location" && strings.ToLower(htmlAttribute.Name) != "locations" {
			if strings.HasPrefix(htmlAttribute.Name, armPropertyNameConvert) {
				//fmt.Println("ARM PROPERTY NAME ABOUT TO GET MATCHED:", armPropertyNameConvert, htmlAttribute.Name)
				if !strings.Contains(armPropertyNameConvert, "os_") && strings.Contains(htmlAttribute.Name, "_") && strings.Contains(armPropertyNameConvert, "_") || !strings.Contains(htmlAttribute.Name, "_") && !strings.Contains(armPropertyNameConvert, "_") {
					//fmt.Println("MATCHED", "ARM:", armPropertyNameConvert, "HTTP:", htmlAttribute.Name)
					htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
				} else if !strings.Contains(armPropertyNameConvert, "os_") && strings.Contains(htmlAttribute.Name, armPropertyNameConvert) && !strings.Contains(htmlAttribute.Name, "ids") { //This negative match will increase in size with experience
					//fmt.Println("MATCHED SPECIAL", htmlAttribute.Name, "ARM:", armPropertyNameConvert)
					htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
				} else if strings.Contains(armPropertyNameConvert, "os_") && strings.Contains(htmlAttribute.Name, armPropertyNameConvert) && armPropertyNameConvert != htmlAttribute.Name {
					checkForMap := CheckForMap(armPropertyValue)
					if checkForMap {
						for attributeName, _ := range armPropertyValue.(map[string]interface{}) {
							armPropertyInnerNameConvert := ConvertArmAttributeName(attributeName)
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
						} else {
							//fmt.Println("STILL MISSING YOU IDIOT,", htmlAttribute.Name)
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
				//fmt.Println("MATCHED2:", htmlAttribute.Name, armPropertyNameConvert)
				htmlAttributeReturn = append(htmlAttributeReturn, htmlAttribute)
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
