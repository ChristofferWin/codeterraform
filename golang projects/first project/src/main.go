package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/valyala/fastjson"
)

type Attribute struct {
	Type string `json:"Type"`
	Name string `json:"Name"`
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
	Name  string
	Value interface{}
}

type Variable struct {
	Name         string
	Description  string
	DefaultValue string
}

type NestedAttribute struct {
	BlockName       string
	NestedAttribute interface{}
	BlockNumber     int //Determine the amount of blocks, e.g if an arm definition has a list of subnets with 2 subnets = 2 for the 'BlockNumber'
}

type CompileObject struct {
	ResourceDefinitionName string
	RootAttributes         []RootAttribute
	Variables              []Variable
	NestedAttributes       []NestedAttribute
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
	/*
		if err := VerifyArmFile(fileContent); err != nil {
			fmt.Println("Error validating json file:", err)
			return
		}
	*/
	baseArmResources := GetArmBaseInformation(fileContent)

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

	SortArmObject(baseArmResources, HtmlObjects)

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

func VerifyArmFile(filecontent [][]byte, fileName string) error {
	for _, content := range filecontent {
		err := fastjson.ValidateBytes(content)
		return err
	}
	return nil
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

		os.WriteFile("ByQuery2", []byte(HtmlBodyCompare), 0644)
	}
	return HtmlBodyCompare, nil
}

func SortRawHtml(rawHtml string, resourceType string) HtmlObject { //See the struct type definitions towards the top of the file
	var allAttributes []string
	var flatAttributes []string //Either bool, string, int or string array, must be determined by the ARM values
	var blockAttribute []string //Object

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
	//Adding all the sorted attributes to the final return armObject
	HtmlObject := HtmlObject{
		Resource_type: resourceType,
		Attribute:     AttributeObjects,
	}

	return HtmlObject
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

func SortArmObject(armBasicObjects []ArmObject, HtmlObjects []HtmlObject) []CompileObject {
	//var rootAttributes []RootAttribute
	var match bool
	var CompiledReturn []CompileObject
	var htmlObject HtmlObject
	//var attributeName string
	var mapValue interface{}
	var isMap bool
	//var recursiveValue interface{}
	var recursiveValues []interface{}
	//var keepGoing interface{}
	for _, armBasicObject := range armBasicObjects {
		//terraformObjectName := strings.Split(strings.Split(ConvertArmAttributeName(armBasicObject.Resource_type), "")[0], "/")
		resourceType := ConvertArmAttributeName(armBasicObject.Resource_type)
		resourceType = fmt.Sprintf(`"azurerm_%s" "%s_object" `, resourceType, resourceType)
		fmt.Println("TF CODE:", resourceType)

		for _, object := range HtmlObjects {
			if object.Resource_type == armBasicObject.Resource_type {
				htmlObject = object
				break
			}
		}

		for armPropertyName, armPropertyValue := range armBasicObject.Properties.(map[string]interface{}) {
			armPropertyNameConvert := ConvertArmAttributeName(armPropertyName)
			recursiveValue := armPropertyValue
			for _, object := range htmlObject.Attribute {
				if strings.Contains(object.Name, armPropertyNameConvert) {
					match = true
					//attributeName = object.Name
				}

				if match && object.Type == "string" {
					reflectValue := reflect.ValueOf(recursiveValue)
					if reflectValue.Kind() == reflect.Map {
						for {
							isMap, mapValue = RecursiveMapLookUp(recursiveValue)
							if !isMap {
								recursiveValues = append(recursiveValues, mapValue)
								fmt.Println("WE ARE HERE:", mapValue)
								break
							}
							recursiveValue = mapValue
						}

					}

				}
			}
		}

		for x, y := range recursiveValues {
			fmt.Println("\nATTRIBUTENAME:", x, "ATTRIBUTE VALUE:", y)
		}

		object := CompileObject{
			ResourceDefinitionName: resourceType,
		}
		CompiledReturn = append(CompiledReturn, object)
	}
	/*
		for v, x := range armBasicObjects[0].Properties.(map[string]interface{}) {
			fmt.Println("\nKEY", v, "VALUE", x)
		}
	*/
	//var test HtmlObject

	//Attribute names in Arm use CammelCase - We need to conver it to lowercase + _ seperator and we need to remove any trailing 's'
	//In addtion we want to do the above BUT also allow attributes between HTML and armobjects to be matched in case 's' Is simply there anyways
	/*
		for _, object := range armBasicObjects {
			fmt.Println("\n", "-------------------------", object.Resource_type, "-------------------------")
			properties := object.Properties.(map[string]interface{})
			for v, value := range properties {
				fmt.Println("\nARM ATTRIBUTE NAME:", v, "ARM VALUE", value)
			}
		}
	*/

	for _, armObject := range HtmlObjects {
		fmt.Println("\n", "-------------------------", armObject.Resource_type, "-------------------------")
		for x, attribute := range armObject.Attribute {
			fmt.Println(x+1, "Attribute name:", attribute.Name, "||", "Type:", attribute.Type)
		}
	}
	os.Exit(0)
	//fmt.Println(armBasicObjects[0].Properties)

	return nil
}

func RecursiveMapLookUp(mapToCheck interface{}) (bool, interface{}) {
	mapConvert := reflect.ValueOf(mapToCheck)
	var isMap bool
	var mapReturn interface{}
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	for _, mapName := range mapConvert.MapKeys() {
		mapReturn = mapConvert.MapIndex(mapName)
		isMap = true

	}
	return isMap, mapReturn
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
