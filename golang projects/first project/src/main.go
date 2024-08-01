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
	filepath := flag.String("filepath", "", "Path to the file")
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

	baseArmResources, err := RetrieveArmBaseInformation(fileContent)

	if err != nil {
		fmt.Println("Error while trying to retrieve the json ARM content", err)
	}

	var resourceTypes []string

	for _, resourceType := range baseArmResources {
		resourceTypes = append(resourceTypes, resourceType.resource_type)
	}

	resourceTypesUnique := UniquifyResourceTypes(resourceTypes)

	for x := 0; x < len(resourceTypesUnique); x++ {
		rawHtml, err := RetrieveRawHtml(baseArmResources[x].resource_type)

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

	//fmt.Println(HtmlObjects)
	/*
				for x := 0; x < len(HtmlObjects); x++ {
					fmt.Println("\n", "THIS IS FOR RESOURCE TYPE:", HtmlObjects[x].name, "-------------------")
					fmt.Println("\n", HtmlObjects[x].attributes.required)
					fmt.Println("\n", HtmlObjects[x].attributes.optional)
				}


		//fmt.Println(HtmlObjects[2].attributes.required)

		//fmt.Println(HtmlObjects[0].attributes.required)
		/*
			for x := 0; x < len(HtmlObjects[0].attributes.optional); x++ {
				fmt.Println("\nLINE", HtmlObjects[0].attributes.optional[x])
			}
	*/
	//fmt.Println("-------------------------------------------------------", HtmlObjects[0].attributes.required)
	/*
		pattern := regexp.MustCompile(regexp.QuoteMeta("<code>") + "(.Block*)" + regexp.QuoteMeta("</code>"))
		for x := 0; x < len(HtmlObjects); x++ {
			for y := 0; y < len(HtmlObjects[x].attributes.required); y++ {
				matches := pattern.FindAllStringSubmatch(HtmlObjects[x].attributes.optional[y], 3)
				for a := 0; a < len(matches); a++ {
					fmt.Println("THIS IS THE ATTRIBUTE FOUND:", matches[a])
					//HtmlObjects[x].attributes.required[y] = matches[1]
				}
			}
		}
	*/
	/*
		for x := 0; x < len(baseArmResources); x++ {
			for y := 0; y < len(baseArmResources[x].properties); y++ {
				fmt.Println(baseArmResources[x].properties[y])
			}
		}
	*/
}

func ImportArmFile(filepath *string) ([]byte, error) {
	file, err := os.ReadFile(*filepath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func VerifyArmFile(filecontent []byte) error {
	err := fastjson.ValidateBytes(filecontent)
	return err
}

func RetrieveArmBaseInformation(filecontent []byte) ([]ArmBasicObject, error) {
	var parserObject fastjson.Parser
	rawJsonBytes, err := parserObject.ParseBytes(filecontent) //Error overwritten by error 2

	var arrayJsonObjects []*fastjson.Value

	// Check if the parsed JSON is an array or an object
	if rawJsonBytes.Type() == fastjson.TypeArray {
		// If it's an array, get the array elements
		arrayJsonObjects = rawJsonBytes.GetArray()
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
	} else if rawJsonBytes.Type() == fastjson.TypeObject {
		// If it's an object, convert the object to an array of values
		object_pre := rawJsonBytes.GetObject()
		object := ArmBasicObject{
			name:          string(object_pre.Get("name").GetStringBytes()),
			resource_type: string(object_pre.Get("type").GetStringBytes()),
			resource_id:   string(object_pre.Get("id").GetStringBytes()),
			location:      string(object_pre.Get("location").GetStringBytes()),
			properties:    ConvertFromStringToSlice(object_pre.Get("properties").String(), ","),
		}

		armBasicObjects = append(armBasicObjects, object)
	} else {
		return nil, fmt.Errorf("unexpected JSON type: %s", rawJsonBytes.Type())
	}

	return armBasicObjects, err
}

func RetrieveRawHtml(resourceType string) (string, error) {
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
	sortingString := strings.Join(resourceTypes, ",")
	var sortingSlice []string
	fmt.Println(resourceTypes)
	fmt.Println(sortingString)

	for _, resourceType := range resourceTypes {
		if strings.Count(sortingString, resourceType) == 1 {
			sortingSlice = append(sortingSlice, resourceType)
			fmt.Println("In", strings.Join(sortingSlice, ","), resourceType)
		} else {
			if !(strings.Contains(strings.Join(sortingSlice, ","), resourceType)) {
				sortingSlice = append(sortingSlice, resourceType)
				fmt.Println("In else", strings.Join(sortingSlice, ","), resourceType)
			}
		}
	}

	return sortingSlice
}
