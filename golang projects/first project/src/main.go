package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/valyala/fastjson"
)

type ArmBasicObject struct {
	name          string
	resource_type string
	resource_id   string
	location      string
	properties    []string
}

type AzureRm struct {
}

type Attribute struct {
	type_ string
	name  string
}

// Define the HtmlObject struct with a named attributes field
type HtmlObject struct {
	resource_type string
	attribute     []Attribute
}

var HtmlObjects = []HtmlObject{}
var AttributeObjects = []Attribute{}
var armBasicObjects = []ArmBasicObject{}

func main() {

	// Define and parse the filepath flag
	filepath := flag.String("filepath", "./", "Path to the file")
	flag.Parse()

	// Call the ImportArmFile function with the filepath argument
	fileContent, err := ImportArmFile(filepath)

	if err != nil {
		// Handle the error if it occurs
		fmt.Println("Error reading file:", err)
		return
	}

	if err := VerifyArmFile(fileContent); err != nil {
		fmt.Println("Error validating json file:", err)
		return
	}

	baseArmResources := GetArmBaseInformation(fileContent)

	if err != nil {
		fmt.Println("Error while trying to retrieve the json ARM content", err)
	}

	SortArmBasicObject(armBasicObjects)

	var resourceTypes []string

	for _, resourceType := range baseArmResources {
		resourceTypes = append(resourceTypes, resourceType.resource_type)
	}

	resourceTypesUnique := UniquifyResourceTypes(resourceTypes)

	for x := 0; x < len(resourceTypesUnique); x++ {
		rawHtml, err := GetRawHtml(baseArmResources[x].resource_type)

		if err != nil {
			fmt.Println("Error while trying to retrieve required documentation", err, baseArmResources[x].resource_type)
			break
		}
		cleanHtml := SortRawHtml(rawHtml, baseArmResources[x].resource_type)
		HtmlObjects = append(HtmlObjects, cleanHtml)
	}

	for _, object := range HtmlObjects {
		fmt.Println("\n", "-------------------------", object.resource_type, "-------------------------")
		for x, attribute := range object.attribute {
			fmt.Println(x+1, "Attribute name:", attribute.name, "||", "Type:", attribute.type_)
		}
	}
}

func ImportArmFile(filepath *string) ([][]byte, error) {

	var fileNames []string
	var files [][]byte
	fileInfo, err := os.Stat(*filepath)
	if err != nil {
		fmt.Println("Error while trying to retrieve ARM json files on path:", string(*filepath), "\nStracktrace:", err)
	}

	isDir := fileInfo.IsDir()
	flag.Parse()

	if isDir {
		files, err := os.ReadDir(*filepath)

		if err != nil {
			fmt.Println("Error while trying to retrieve ARM json files on path:", string(*filepath), "\nStracktrace:", err)
		}

		for _, file := range files {
			if strings.Contains(file.Name(), ".json") {
				fileNames = append(fileNames, file.Name())
			}
		}
	} else {
		fileNames = append(fileNames, *filepath)
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

func VerifyArmFile(filecontent [][]byte) error {
	//err := fastjson.ValidateBytes(filecontent)
	return nil //err
}

func GetArmBaseInformation(filecontent [][]byte) []ArmBasicObject {
	var parserObject fastjson.Parser
	var arrayJsonObjects []*fastjson.Value

	for _, bytes := range filecontent {
		parserObject, err := parserObject.ParseBytes(bytes)

		if err != nil {
			fmt.Println("Error while transforming file from bytes to json:", err)
		}

		// Check if the parsed JSON is an array or an object
		if parserObject.Type() == fastjson.TypeArray {
			// If it's an array, get the array elements
			arrayJsonObjects = parserObject.GetArray()
			for _, x := range arrayJsonObjects {
				object := ArmBasicObject{
					name:          string(x.GetStringBytes("name")),
					resource_type: string(x.GetStringBytes("type")),
					resource_id:   string(x.GetStringBytes("id")),
					location:      string(x.GetStringBytes("location")),
					properties:    ConvertFromStringToSlice((x.GetObject("properties").String()), ","),
				}

				armBasicObjects = append(armBasicObjects, object)
			}
		} else if parserObject.Type() == fastjson.TypeObject {
			// If it's an object, convert the object to an array of values
			object_pre := parserObject.GetObject()
			object := ArmBasicObject{
				name:          string(object_pre.Get("name").GetStringBytes()),
				resource_type: string(object_pre.Get("type").GetStringBytes()),
				resource_id:   string(object_pre.Get("id").GetStringBytes()),
				location:      string(object_pre.Get("location").GetStringBytes()),
				properties:    ConvertFromStringToSlice(object_pre.Get("properties").String(), ","),
			}

			armBasicObjects = append(armBasicObjects, object)
		}
	}

	return armBasicObjects
}

func GetRawHtml(resourceType string) (string, error) {
	var HtmlBody string
	var HtmlBodyCompare string
	var resourceTypeRegex *regexp.Regexp
	var convertResourceTypeName string
	// Use a regex to find places where we need to insert underscores

	//Define regex so that we can differentiate between resource types 'Something/Something' And Something/Something/Somthing
	if len(strings.Split(resourceType, "/")) == 2 {
		convertResourceTypeName = func() string {
			resourceTypeRegex = regexp.MustCompile("([a-z0-9])([A-Z])")
			convertResourceTypeName = resourceTypeRegex.ReplaceAllString(resourceType, "${1}_${2}")
			return strings.Split(convertResourceTypeName, "/")[1]
		}()
	} else {
		convertResourceTypeName = func() string {
			sliceArray := strings.Split(resourceType, "/")
			return sliceArray[len(sliceArray)-1]
		}()
	}

	// Remove the trailing 's' if it exists
	convertResourceTypeName = strings.TrimSuffix(convertResourceTypeName, "s")

	// Convert the entire string to lower case
	convertResourceTypeName = strings.ToLower(convertResourceTypeName)

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

	url := fmt.Sprintf("https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/%s", convertResourceTypeName)

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

	//Defining the regex matching ALL attributes, which we will sort through first, then seperate on type - object vs string
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
				type_: "object",
				name:  oneBlockRegex.FindStringSubmatch(line)[1],
			}
			AttributeObjects = append(AttributeObjects, attribute)
		}
	}
	//For all the type 'string' Add them to the overall attribute slices
	for _, line := range flatAttributes {
		attribute := Attribute{
			type_: "string",
			name:  isolateAttributesRegex.FindStringSubmatch(line)[1],
		}
		AttributeObjects = append(AttributeObjects, attribute)
	}
	//Adding all the sorted attributes to the final return object
	HtmlObject := HtmlObject{
		resource_type: resourceType,
		attribute:     AttributeObjects,
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

func SortArmBasicObject(armBasicObjects []ArmBasicObject) []ArmBasicObject {
	var indexOfStartObject int
	var isInsideTupple bool
	var properties []string
	regexStartOfListOfObject := regexp.MustCompile(`\[\{`)
	regexEndOfListOfObject := regexp.MustCompile(`\}\]`)
	regexFlatAttribute := regexp.MustCompile(`\s*\"[^\"]+\":\"[^\"]+\"`)

	for x, armObject := range armBasicObjects {
		for y, property := range armObject.properties {
			//fmt.Println("NEW LINE:", y, ":", property)
			if regexStartOfListOfObject.MatchString(property) && !regexEndOfListOfObject.MatchString(property) {
				indexOfStartObject = y
				isInsideTupple = true
			} else if isInsideTupple && strings.Contains(armObject.properties[y+2], "}]") {
				property := strings.Join(armObject.properties[indexOfStartObject:len(armObject.properties)-indexOfStartObject+1], "%")
				properties = append(properties, property)
				isInsideTupple = false
			} else if strings.Contains(property, ":{") && strings.Contains(property, "}") && !isInsideTupple {
				properties = append(properties, property)
			} else if regexFlatAttribute.MatchString(property) && !isInsideTupple {
				properties = append(properties, property)
			}

		}
		armBasicObjects[x].properties = properties
	}

	for _, object := range armBasicObjects {
		fmt.Println("THIS IS OBJECT:", object.name)
		for _, line := range object.properties {
			fmt.Println("\n------------ATTRIBUTE----------------")
			for y, line := range strings.Split(line, "%") {
				fmt.Println("Object line:", y, line)
			}
		}
	}

	os.Exit(0)
	return armBasicObjects
}

type JSONData struct {
	Name       string      `json:"name"`
	ID         string      `json:"id"`
	Etag       string      `json:"etag"`
	Type       string      `json:"type"`
	Location   string      `json:"location"`
	Properties interface{} `json:"properties"` // Use interface{} for dynamic properties
}

jsonData := `[
	{
		"name": "test-1-file-1-vnet",
		"id": "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/test/providers/Microsoft.Network/virtualNetworks/test-vnet",
		"etag": "W/\"73685588-a0ca-4796-8a47-6aa2369f9bd3\"",
		"type": "Microsoft.Network/virtualNetworks/subnets",
		"location": "eastus",
		"properties": {
			"provisioningState": "Succeeded",
			"resourceGuid": "64c57928-3194-4d8c-80ab-389177d79cd7",
			"addressSpace": {
				"addressPrefixes": [
					"10.0.0.0/16"
				]
			},
			"networkProfile": {
				"networkInterfaces": [
					{
						"id": "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/test/providers/Microsoft.Network/networkInterfaces/test686",
						"properties": {
							"deleteOption": "Detach"
						},
						"resourceGroup": "test"
					}
				]
			},
			"subnets": [
				{
					"name": "default",
					"id": "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/test/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/default",
					"etag": "W/\"73685588-a0ca-4796-8a47-6aa2369f9bd3\"",
					"properties": {
						"provisioningState": "Succeeded",
						"addressPrefix": "10.0.0.0/24",
						"ipConfigurations": [
							{
								"id": "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/TEST/providers/Microsoft.Network/networkInterfaces/TEST686/ipConfigurations/IPCONFIG1"
							}
						],
						"delegations": [],
						"privateEndpointNetworkPolicies": "Disabled",
						"privateLinkServiceNetworkPolicies": "Enabled"
					},
					"type": "Microsoft.Network/networkWatchers"
				}
			],
			"virtualNetworkPeerings": [],
			"enableDdosProtection": false
		}
	}
]`

var data []JSONData

// Unmarshal the JSON into the Go struct
err := json.Unmarshal([]byte(jsonData), &data)
if err != nil {
	fmt.Println("Error:", err)
	return
}

// Print the parsed data
for _, item := range data {
	fmt.Printf("Name: %s\n", item.Name)
	fmt.Printf("ID: %s\n", item.ID)
	fmt.Printf("Location: %s\n", item.Location)
	fmt.Printf("Properties: %v\n", item.Properties)
}
}