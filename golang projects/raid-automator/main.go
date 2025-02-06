package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/chromedp/chromedp"
)

type keyvaultToken struct {
	Name      string `json:"name"`
	VersionID string `json:"version"`
}

type keyvault struct {
	URI    string
	Tokens []keyvaultToken `json:"keyvaultToken"`
}

type messageTemplate struct {
	Name   string
	Fields []*discordgo.MessageEmbedField
}

type emojies struct {
	ID        string `json:"ID"`
	Wrapper   string
	ShortName string `json:"emojiShortName"`
	Name      string `json:"emojiName"`
	TypeInt   int    `json:"emojiType"`     //0 = race, 1 = class, 2 = spec, 3 = fun
	NickName  string `json:"emojiNickName"` //something like "pala" And get all emojies associated with that
}

type classSpecs struct {
	ClassSpec      string `json:"classSpec"`
	MemeSpec       bool   `json:"memeSpec"`
	ClassGuideLink string `json:"classGuideLink"`
}

type classesInternal struct {
	Name          string       `json:"name"`
	ClassSpecs    []classSpecs `json:"listOfSpecs"`
	PossibleRaces []string     `json:"possibleRaces"`
}

type class struct {
	Name               string `json:"name"`
	NameEmojiID        string `json:"nameEmojiID"`
	IngameRace         string `json:"ingameRace"`
	IngameRaceEmojiID  string `json:"ingameRaceEmojiID"`
	IngameClass        string `json:"ingameClass"`
	IngameClassEmohiID string `json:"ingameClassEmojiID"`
	SpecEmojiID        string `json:"specEmojiID"`
	HasDouseEmojiID    string `json:"HasDouseEmojiID"`
}

type raiderProfile struct {
	Name         string
	ID           string
	DiscordRoles []string
	ChannelID    string
	classInfo    class
	Raiding      raidMember
}

type raidMember struct {
	HasReserved bool
	MemberRole  string //Will either be trial, raider or puggie
}

type schedule struct {
	HourMinute string
	Weekday    time.Weekday
}

type raidType struct {
	RaidName string
	Timer    int
	RaidSize int
}

type raidCooldown struct {
	Date      string
	EventType raidType
}

var (
	raiderProfiles  = []raiderProfile{}
	classesImport   = []classesInternal{}
	emojiesImport   = []emojies{}
	raidHelperToken = ""
	botToken        = ""
	keyvaultConfig  = keyvault{}

	messageTemplates = map[string]messageTemplate{
		"New_user": {
			Name: "New_user",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Initial screening of new player",
					Value:  fmt.Sprintf("Please react 1 time to each question being posted by the bot\n\n"),
					Inline: false,
				}, /*
					{
						Name:  "GETTING HELP",
						Value: "If your in doubt about your spec, let the bot help you provide spec information:\n\nType a random spec like: wowuser rogue sdaasd human puggie yes\n\nThe bogus adsds set for spec will force the bot to give spec info!",
					},
				*/
			},
		},
		"Player_Welcome": {
			Name: "Player_Welcome",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Welcome!",
					Value:  "Welcome to the server! We're glad to have you.",
					Inline: true,
				},
				{
					Name:   "Rules",
					Value:  "Please review the rules before participating.",
					Inline: false,
				},
			},
		},
	}
)

const (
	serverID = "630793944632131594"

	channelInfo      = "1308521695564402899"
	channelGeneral   = "1308521052036530291"
	channelVoting    = "1316379489906855936"
	channelSignUp    = "1308521842407116830"
	channelSignUpPug = "1334949433208606791"
	channelWelcome   = "1309312094822203402"
	channelBot       = "1336098468615426189"

	categoryBot = "1336097759073140920"

	roleTemp        = "1335442062459535371"
	rolePuggie      = "1335028472770461847"
	roleTrial       = "1335374757847109768"
	roleGuildMember = "651796644739940369"
	roleRaider      = "1322681249294323814"
	roleWarrior     = "1314531430122131550"
	rolePaladin     = "1314531854031913000"
	roleRogue       = "1314532022945054751"
	roleHunter      = "1314532209759359056"
	roleMage        = "1314532563796361227"
	roleWarlock     = "1314532784529866843"
	roleDruid       = "1314533133688897577"
	rolePriest      = "1314533393614241832"

	raidHelperEventBaseURL = "https://raid-helper.dev/api/v2/events/"
	raidHelperId           = "579155972115660803"
	classesPath            = "./classes.json"
	keyvaultPath           = "./keyvault.json"
	cachePath              = "./cache.json"
	emojiesPath            = "./emojies.json"

	permissionViewChannel    = int64(1 << 0)  // 1
	permissionSendMessages   = int64(1 << 10) // 1024
	permissionReadMessages   = int64(1 << 11) // 2048
	permissionManageMessages = int64(1 << 13) // 8192
)

func init() {
	/*
		ImportKeyvaultConfig()
		azCred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			log.Fatal("An error occured while trying to retrieve the default system assigned managed identity:", err)
		}
		azKeyvaultClient, err := azsecrets.NewClient(keyvaultConfig.URI, azCred, nil)
		azContext := context.Background()
		secret, err := azKeyvaultClient.GetSecret(azContext, keyvaultConfig.Tokens[0].Name, keyvaultConfig.Tokens[0].VersionID, nil)
		botToken = *secret.Value
		secret, err = azKeyvaultClient.GetSecret(azContext, keyvaultConfig.Tokens[1].Name, keyvaultConfig.Tokens[1].VersionID, nil)
		raidHelperToken = *secret.Value
	*/
	ImportEmojies()
	ImportClasses()

}

func GetOnlyPresentRaids(raidCooldowns []raidCooldown) []raidCooldown {
	timeNow := time.Now()
	presentRaids := []raidCooldown{}
	timeLayout := "January 2, 2006"
	for x := len(raidCooldowns) - 1; x >= 0; x-- {
		convertStringToTime, err := time.Parse(timeLayout, raidCooldowns[x].Date)
		if err != nil {
			log.Fatal("An error occured while trying to convert the string to a time object:", err)
		}
		if convertStringToTime.After(timeNow) {
			presentRaids = append(presentRaids, raidCooldowns[x])
		}
	}
	return presentRaids
}

func GetRaidReset() []raidCooldown {
	allRaids := []raidCooldown{}
	commingRaids := []raidCooldown{}
	raid := raidCooldown{}
	opts := []func(*chromedp.ExecAllocator){}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	htmlBody := ""
	//var datesArray = []string{}
	defer cancel()
	opts = append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true), // Set to false if you want to see the browser UI
		chromedp.Flag("disable-gpu", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	url := "https://classicraidreset.com/EU/Classic"

	err := chromedp.Run(taskCtx,
		chromedp.Navigate(string(url)),
		chromedp.WaitReady("body"),                   // Wait for the body to be ready
		chromedp.Sleep(time.Duration(5)*time.Second), // Give some time for JS to execute
		chromedp.InnerHTML("*", &htmlBody, chromedp.ByQueryAll),
	)

	if err != nil {
		log.Fatal("An error occured while trying to retrieve information about raid resets:", err)
	}

	//Defining the boundaries of the data we are interested in
	startIndex := regexp.MustCompile(`<div\s+x-ref="calendar"\s+wire:ignore=""\s+id="calendar-element"`).FindStringIndex(htmlBody)
	endIndex := regexp.MustCompile(`©\s+Qirel\s+Development\s+-\s+2025`).FindStringIndex(htmlBody)

	//Isolating only the 'argument references from the HTML dump'
	extractedText := htmlBody[startIndex[1]:endIndex[0]]

	// Regex to capture the date and event information
	eventRegex := regexp.MustCompile(`<a\s+aria-label="([A-Za-z]+\s\d{1,2},\s\d{4})"[^>]*>.*?(\d{2}:\d{2} - [^<]+)`)

	// Find the match
	matches := eventRegex.FindAllStringSubmatch(extractedText, -1)

	if matches != nil {
		for _, match := range matches {
			if strings.Contains(match[2], "AQ20") {
				raid = raidCooldown{
					Date: match[1],
					EventType: raidType{
						RaidName: "AQ20",
						Timer:    5,
						RaidSize: 20,
					},
				}
			} else if strings.Contains(match[2], "AQ40") {
				raid = raidCooldown{
					Date: match[1],
					EventType: raidType{
						RaidName: "AQ40",
						Timer:    7,
						RaidSize: 40,
					},
				}
			} else if strings.Contains(match[2], "Onyxia's Lair") {
				raid = raidCooldown{
					Date: match[1],
					EventType: raidType{
						RaidName: "Ony",
						Timer:    5,
						RaidSize: 20,
					},
				}
			}

			allRaids = append(allRaids, raid)
		}
	}
	commingRaids = GetOnlyPresentRaids(allRaids)
	return commingRaids
}

func UpdateCache(raiderProfiles []raiderProfile) {
	mapOfRaiderProfiles := make(map[string]bool)
	uniqueRaiderProfiles := []raiderProfile{}
	for _, raider := range raiderProfiles {
		if !mapOfRaiderProfiles[raider.ID] {
			uniqueRaiderProfiles = append(uniqueRaiderProfiles, raider)
		}
	}

	jsonRaiderProfiles, err := json.MarshalIndent(uniqueRaiderProfiles, "", "    ") // Indent with 4 spaces
	file, err := os.OpenFile("data.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("An error occured while trying to open the file:", cachePath, err)
		return
	}
	file.Write(jsonRaiderProfiles)
	defer file.Close()
}

func GetEmojies(typeEmoji int, nickNames []string) []emojies {
	allEmojies := []emojies{}
	returnEmojies := []emojies{}
	mapOfUniqueEmojies := make(map[string]bool)
	for _, emoji := range emojiesImport {
		if typeEmoji == emoji.TypeInt {
			for _, nickName := range nickNames {
				nickNameLower := strings.ToLower(nickName)
				if strings.Contains(emoji.NickName, nickNameLower) {
					allEmojies = append(allEmojies, emoji)
				}
			}
		}
	}

	for _, emoji := range allEmojies {
		if !mapOfUniqueEmojies[emoji.ID] {
			returnEmojies = append(returnEmojies, emoji)
			mapOfUniqueEmojies[emoji.ID] = true
		}
	}

	for x, emoji := range returnEmojies {
		returnEmojies[x].Wrapper = fmt.Sprintf("<:%s:%s>", emoji.Name, emoji.ID)
	}
	return returnEmojies
}

func NewDiscordSession(debug bool) *discordgo.Session {
	botSession, err := discordgo.New("Bot " + botToken)
	if debug {
		botSession.LogLevel = discordgo.LogDebug
	}
	botSession.Identify.Intents = discordgo.IntentGuildMessages | discordgo.IntentGuildMessageReactions | discordgo.IntentDirectMessageReactions | discordgo.IntentsGuildMembers | discordgo.IntentsDirectMessages
	if err != nil {
		log.Fatal("An error occured while trying to ")
	}
	err = botSession.Open()
	if err != nil {
		log.Fatal("An error occured while trying to open a new discord bot connection", err)
	}
	return botSession
}

func NewPlayerJoin(botSession *discordgo.Session) {
	createdChannels := make(map[string]string) // raidProfile.ID -> channelID
	mapOfUsedConnections := make(map[string]bool)
	mapOfNewUsers := make(map[string]*raiderProfile)
	stage0 := "Welcome to the server"
	stage2 := "Please answer all the following questions:"
	stage3 := "What is your class? (Use reactions)"

	// Handler for when a new user joins
	botSession.AddHandler(func(session *discordgo.Session, eventOuter *discordgo.GuildMemberAdd) {
		raidProfile := raiderProfile{
			Name: eventOuter.User.Username,
			ID:   eventOuter.User.ID,
		}
		mapOfNewUsers[raidProfile.ID] = &raidProfile

		fmt.Println("USER IS PUT INTO THE MAP", raidProfile.ID, raidProfile.ChannelID)
		botSession.GuildMemberRoleAdd(serverID, raidProfile.ID, roleTemp)

		// Check if user already has a setup channel
		if _, exists := createdChannels[raidProfile.ID]; exists {
			return
		}
		botSession.ChannelMessageSend(channelBot, fmt.Sprintf("%s <@%s> <:cracked:1312847304725893190>", stage0, raidProfile.ID))
		// Create a new channel for the user
		channelName := fmt.Sprintf("automatic-%s", raidProfile.ID)
		newChannelTemplate := discordgo.GuildChannelCreateData{
			Name:     channelName,
			Type:     discordgo.ChannelTypeGuildText,
			Topic:    "Set roles / raid status for user",
			ParentID: categoryBot,
			PermissionOverwrites: []*discordgo.PermissionOverwrite{
				{
					ID:    botSession.State.User.ID,
					Type:  discordgo.PermissionOverwriteTypeMember,
					Allow: permissionViewChannel | permissionManageMessages | permissionSendMessages | permissionReadMessages,
				},
				{
					ID:    roleTemp, // The role you want to add
					Type:  discordgo.PermissionOverwriteTypeRole,
					Allow: permissionViewChannel | permissionReadMessages, // Grant role view & send permissions
				},
			},
		}

		newChannelWithUser, err := botSession.GuildChannelCreateComplex(serverID, newChannelTemplate)
		mapOfNewUsers[raidProfile.ID].ChannelID = newChannelWithUser.ID
		if err != nil {
			log.Printf("Error creating channel %s: %v", newChannelTemplate.Name, err)
			return
		}

		createdChannels[raidProfile.ID] = newChannelWithUser.ID

		tagUser := discordgo.MessageEmbed{
			Fields: messageTemplates["New_user"].Fields,
		}
		botSession.ChannelMessageSendEmbed(newChannelWithUser.ID, &tagUser)
		botSession.ChannelMessageSend(newChannelWithUser.ID, fmt.Sprintf("<@%s> %s", raidProfile.ID, stage2))

		// Notify in bot channel if user setup is required
		if !mapOfUsedConnections[newChannelWithUser.ID] {
			botSession.ChannelMessageSend(channelBot, fmt.Sprintf("<@%s> <:cracked:1312847304725893190> VISIT <#%s> TO GET SETUP", raidProfile.ID, newChannelWithUser.ID))
			mapOfUsedConnections[newChannelWithUser.ID] = true
		}
	})

	// Separate Message Handler (Fixes multiple registrations)
	botSession.AddHandler(func(session *discordgo.Session, event *discordgo.MessageCreate) {
		if event.Content != "" {
			if strings.Contains(event.Content, " VISIT ") && session.State.User.ID == event.Author.ID {
				// Get emojis for class selection
				emojies := GetEmojies(1, []string{"class"})
				var emojieStringSlice []string
				for _, emoji := range emojies {
					emojieStringSlice = append(emojieStringSlice, fmt.Sprintf("\n%s %s", emoji.Wrapper, emoji.ShortName))
				}
				emojieString := fmt.Sprintf("%s\n%s", stage3, strings.Join(emojieStringSlice, "\n"))
				patternUserID := regexp.MustCompile(`<@(\d+)>`)
				patternChannelID := regexp.MustCompile(`<#(\d+)>`)
				userID := patternUserID.FindStringSubmatch(event.Content)
				channelID := patternChannelID.FindStringSubmatch(event.Content)
				newUser := userID[1]
				newChannel := channelID[1]
				if newUser != "" {
					botSession.ChannelMessageSend(newChannel, fmt.Sprintf("\n\n%s", emojieString))
				} else {
					fmt.Println("WAS NOT ABLE TO FIND THE ID")
				}
			}
		}
	})
	//playerClassName := ""
	//playerRace := ""
	//playerSpec := ""
	//PlayerRole := ""
	//playerDouse := ""
	hasRun := false
	botSession.AddHandler(func(session *discordgo.Session, event *discordgo.MessageReactionAdd) {
		playerClassName := ""
		playerClassID := ""
		classSpecific := class{}
		if raidProfileInMem, excist := mapOfNewUsers[event.UserID]; excist && mapOfNewUsers[event.UserID].classInfo.IngameClassEmohiID != "" {
			fmt.Println("THE KEY EXISTS!!", raidProfileInMem)
			hasRun = true
			playerID := event.Member.User.ID
			specEmojies := GetEmojies(2, []string{"spec"})
			specEmojieStringSlice := []string{}
			for _, specEmojie := range specEmojies {
				class := strings.Split(specEmojie.Name, "_")[0]
				if class == mapOfNewUsers[playerID].classInfo.Name {
					specEmojieStringSlice = append(specEmojieStringSlice, fmt.Sprintf("%s %s", specEmojie.Wrapper, specEmojie.ShortName))
				}
			}

			fmt.Println("THIS IS THE MAP SO FAR FROM THE USER:", mapOfNewUsers[playerID])
			//playerRace := event.Emoji.ID
			//botSession.ChannelMessageSend(event.ChannelID)

		} else if emojie := DetermineEmoji(event.Emoji.Name); emojie.TypeInt == 0 && emojie != (emojies{}) && excist {

			fmt.Println("WE ARE HERE NOIW:", emojie.ID, emojie.Name)

			//botSession.ChannelMessageSend(userProfile.ChannelID, fmt.Sprintf("You reacted with <:%s:%s> %s\n\nWhat is your in-game spec?\n\n%s", userProfile.classInfo.Name, userProfile.classInfo.NameEmojiID, userProfile.classInfo.Name, strings.Join(specEmojieStringSlice, "\n\n")))
		}

		if DetermineEmoji(event.Emoji.Name).TypeInt == 1 && hasRun {
			playerClassID = event.Emoji.ID
			playerClassName = event.Emoji.Name
			playerUserID := event.Member.User.ID
			possibleRacesEmojies := []emojies{}
			nextReactionMessage := []string{}
			emojiNameConvert := ""
			fmt.Println("ARE H")
			if strings.Contains(event.Emoji.Name, "_"); len(strings.Split(event.Emoji.Name, "_")) == 2 {
				emojiNameConvert = strings.Split(event.Emoji.Name, "_")[0]
				fmt.Println("THIS IS IT", emojiNameConvert)
			} else {
				emojiNameConvert = event.Emoji.Name
			}

			for _, classInternal := range classesImport {
				if strings.ToLower(classInternal.Name) == strings.ToLower(emojiNameConvert) {
					for _, race := range classInternal.PossibleRaces {
						possibleRacesEmojies = append(possibleRacesEmojies, DetermineEmoji(race))
						fmt.Println("FOUND RACE:", DetermineEmoji(race))
					}
					classSpecific = class{
						Name:        playerClassName,
						NameEmojiID: playerClassID,
					}
					// Check if the user exists, if not, initialize it
					if mapOfNewUsers[playerUserID] == nil {
						fmt.Println("WE HAVE NO DATA!!!!", playerUserID)
						mapOfNewUsers[playerUserID] = &raiderProfile{}
					}
					mapOfNewUsers[playerUserID].classInfo = classSpecific
				}
			}
			for _, emoji := range possibleRacesEmojies {
				nextReactionMessage = append(nextReactionMessage, fmt.Sprintf("%s %s", emoji.Wrapper, emoji.ShortName))
			}
			hasRun = false
			botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("You reacted with <:%s:%s> %s\n\nWhat is your in-game race?:\n\n%s", event.Emoji.Name, playerClassID, event.Emoji.Name, strings.Join(nextReactionMessage, "\n\n")))
		}
	})

	UpdateCache(raiderProfiles)
}

func DetermineEmoji(nickName string) emojies {
	patternConvertString := regexp.MustCompile(`^([a-zA-Z]+)_\d+$`)
	convertNickNameString := ""
	if convertNickName := patternConvertString.FindSubmatch([]byte(nickName)); len(convertNickName) > 1 {
		convertNickNameString = strings.ToLower(string(convertNickName[1]))
	} else {
		convertNickNameString = strings.ToLower(nickName)
	}

	fmt.Println("FOUND NICK NAMES:", convertNickNameString)
	for x, emoji := range emojiesImport {
		if strings.ToLower(emoji.ShortName) == convertNickNameString || strings.Contains(convertNickNameString, strings.ToLower(emoji.ShortName)) {
			fmt.Println("WE DONT REACH HERER????", emoji)
			emojiesImport[x].Wrapper = fmt.Sprintf("<:%s:%s>", emoji.Name, emoji.ID)
			return emojiesImport[x]
		}
	}
	return emojies{}
}

func RunAtSpecificTime(taskToRun func(), schedule schedule) {
	for {
		timeNow := time.Now()

		// Parse hour and minute from the schedule
		parts := strings.Split(schedule.HourMinute, ":")
		if len(parts) != 2 {
			log.Fatal("Invalid time format. Please use HH:MM (e.g., 19:30)")
		}
		hour, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Fatal("Invalid hour in schedule:", err)
		}
		minute, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Fatal("Invalid minute in schedule:", err)
		}

		// Find the next occurrence of the scheduled day
		daysUntilNextOccurrence := (int(schedule.Weekday) - int(timeNow.Weekday()) + 7) % 7
		if daysUntilNextOccurrence == 0 && (timeNow.Hour() > hour || (timeNow.Hour() == hour && timeNow.Minute() >= minute)) {
			// If today is the scheduled day but the time has already passed, schedule for next week
			daysUntilNextOccurrence = 7
		}

		// Compute target time
		targetTime := timeNow.AddDate(0, 0, daysUntilNextOccurrence)
		targetTime = time.Date(targetTime.Year(), targetTime.Month(), targetTime.Day(), hour, minute, 0, 0, targetTime.Location())

		// Wait until the target time
		timeLeft := time.Until(targetTime)
		fmt.Printf("Task scheduled for: %s\n", targetTime.Format(time.RFC1123))
		time.Sleep(timeLeft)

		// Execute the task
		taskToRun()
	}
}

func main() {
	newBotSession := NewDiscordSession(true)
	var err error
	//NewMainRaid()
	if err != nil {
		log.Fatal("Error getting guild info:", err)
	}
	defer newBotSession.Close()

	newBotSession.AddHandler(func(session *discordgo.Session, event *discordgo.MessageUpdate) {
		fmt.Printf("Message updated! ID: %s, Channel: %s\n", event.ID, event.ChannelID)
		if event.Author.ID == raidHelperId {
			if event.Embeds[0].Fields != nil { //Must avoid a panic incase no fields are present
				URLString := ""
				raidID := ""
				convetFieldsToByteArray, _ := json.Marshal(event.Embeds[0].Fields)
				patternFindRaidURL := regexp.MustCompile(`https://raid-helper\.dev/event/(\d+)`)
				URLArray := patternFindRaidURL.FindAllString(string(convetFieldsToByteArray), -1) //WE get the URL FINALLY!!!

				if len(URLArray) > 0 {
					URLString = URLArray[0]
				}

				if strings.Count(URLString, "/") == 4 {
					raidID = strings.Split(URLString, "/")[len(strings.Split(URLString, "/"))-1]
				} else {
					return
				}
				fmt.Println("RAID ID::_", raidID)
				RetriveEvent(raidID)
			}
		}
	})

	NewPlayerJoin(newBotSession)
	log.Println("Bot is now running. Press CTRL+C to exit.")
	//botSession.ChannelMessageSend(channelID, "Hi @everyone - Meet your new personal assistent! I cant wait to get started making sure @Toxico does not troll hihi")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}

func RetriveEvent(raidID string) string {
	newRaidURL := raidHelperEventBaseURL + raidID
	getSignupData, _ := http.NewRequest("GET", newRaidURL, nil)
	getSignupData.Header = http.Header{
		"Authorization": {raidHelperToken},
		"Content-Type":  {"application/json"},
	}
	client := &http.Client{}
	data, _ := client.Do(getSignupData)
	// Read and print the response body
	body, err := io.ReadAll(data.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}
	test, _ := json.MarshalIndent(body, "", " ")
	os.WriteFile("tester.json", test, 0644)
	return ""
}

func ImportClasses() {
	classesImportBytes, err := os.ReadFile(classesPath)
	if err != nil {
		log.Fatal("the following error occured while trying to load the classes config file:\n", err)
	}
	if err := json.Unmarshal(classesImportBytes, &classesImport); err != nil {
		log.Fatal("the following error occured while trying to convert bytes to maps:\n", err)
	}
}

func ImportEmojies() {
	emojiesImportBytes, err := os.ReadFile(emojiesPath)
	if err != nil {
		log.Fatal("the following error occured while trying to load the classes config file:\n", err)
	}
	if err := json.Unmarshal(emojiesImportBytes, &emojiesImport); err != nil {
		log.Fatal("the following error occured while trying to convert bytes to maps:\n", err)
	}
}

func ImportKeyvaultConfig() {
	keyvaultImportBytes, err := os.ReadFile(keyvaultPath)
	if err != nil {
		log.Fatal("An error occured while trying to load the keyvault config:", err)
	}
	json.Unmarshal(keyvaultImportBytes, &keyvaultConfig)

	if len(keyvaultConfig.Tokens) < 2 {
		log.Fatal("An error occured while counting the amount of secrets available: Less than 2 token definitions are present")
	}

	keyvaultConfig = keyvault{
		URI:    fmt.Sprintf("%s.vault.azure.net", keyvaultConfig.URI),
		Tokens: keyvaultConfig.Tokens,
	}
	return
}
