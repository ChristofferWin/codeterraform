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

// Define the HtmlObject struct with a named attributes field
type HtmlObject struct {
	name       string
	attributes struct {
		optional []string
		required []string
	}
}

var HtmlObjects = []HtmlObject{}
var armBasicObjects = []ArmBasicObject{}
var rawHtmlArray []string

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

	for x := 0; x < len(baseArmResources); x++ {
		rawHtml, err := RetrieveRawHtml(baseArmResources[x].resource_type)

		if err != nil {
			fmt.Println("Error while trying to retrieve required documentation", err, baseArmResources[x].resource_type)
			break
		}
		cleanHtml := SortRawHtml(rawHtml, baseArmResources[x].resource_type)
		HtmlObjects = append(HtmlObjects, cleanHtml)
	}
	/*
		for x := 0; x < len(HtmlObjects); x++ {
			fmt.Println("\n", "THIS IS FOR RESOURCE TYPE:", HtmlObjects[x].name, "-------------------")
			fmt.Println("\n", HtmlObjects[x].attributes.required)
			fmt.Println("\n", HtmlObjects[x].attributes.optional)
		}
	*/

	fmt.Println(HtmlObjects[2].attributes.required)

	pattern := regexp.MustCompile(regexp.QuoteMeta("<code>") + "(.*?)" + regexp.QuoteMeta("</code>"))
	for x := 0; x < len(HtmlObjects); x++ {
		for y := 0; y < len(HtmlObjects[x].attributes.required); y++ {
			matches := pattern.FindStringSubmatch(HtmlObjects[x].attributes.required[y])
			for a := 0; a < len(matches); a++ {
				HtmlObjects[x].attributes.required[y] = matches[1]
			}
		}
	}

	fmt.Println(HtmlObjects[2].attributes.required)

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
		for x := 0; x < len(arrayJsonObjects); x++ {
			object := ArmBasicObject{
				name:          string(arrayJsonObjects[x].GetStringBytes("name")),
				resource_type: string(arrayJsonObjects[x].GetStringBytes("type")),
				resource_id:   string(arrayJsonObjects[x].GetStringBytes("id")),
				location:      string(arrayJsonObjects[x].GetStringBytes("location")),
				properties:    ConvertFromStringToSlice((arrayJsonObjects[x].GetObject("properties").String()), ","),
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

	// Use a regex to find places where we need to insert underscores
	regex := regexp.MustCompile("([a-z0-9])([A-Z])")
	convertResourceTypeName := regex.ReplaceAllString(resourceType, "${1}_${2}")

	// Remove the trailing 's' if it exists
	convertResourceTypeName = strings.TrimSuffix(convertResourceTypeName, "s")

	//Only use right side part
	convertResourceTypeName = strings.Split(convertResourceTypeName, "/")[1]

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

	fmt.Println("This is the url:", url)

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
	return HtmlBodyCompare, nil
}

func SortRawHtml(rawHtml string, resourceType string) HtmlObject { //See the struct type definitions towards the top of the file
	var requiredArguments []string
	var optionalArguments []string

	// Define the regular expressions for (Required) and (Optional)  //It seems that this regex destroys all blocks which is not ideal
	//Need to change the regex
	requiredRegex := regexp.MustCompile(`\(Required\)`)
	optionalRegex := regexp.MustCompile(`\(Optional\)`)

	// Filter lines containing (Required) or (Optional)
	linesHtml := strings.Split(rawHtml, "\n")
	for _, line := range linesHtml {
		if requiredRegex.MatchString(line) {
			requiredArguments = append(requiredArguments, line)
		} else if optionalRegex.MatchString(line) {
			optionalArguments = append(optionalArguments, line)
		}
	}

	object := HtmlObject{
		name: resourceType,

		attributes: struct {
			optional []string
			required []string
		}{optionalArguments, requiredArguments},
	}

	return object
}

func ConvertFromStringToSlice(stringToSlice string, seperatorChar string) []string {
	arrayOfSlices := strings.Split(strings.TrimSuffix(strings.TrimPrefix(stringToSlice, "{"), "}"), seperatorChar)
	return arrayOfSlices
}
