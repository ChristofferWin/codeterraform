package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/errors"

	"github.com/bwmarrin/discordgo"
)

type informationLog struct {
	Action    string `json:"action"`
	Message   string `json:"message"`
	TimeStamp string `json:"time_stamp"`
}

type errorLog struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	TimeStamp string `json:"time_stamp"`
}

type config struct {
	ServerID            string           `json:"serverID"`
	ChannelID           string           `json:"channelID"`
	WarcraftLogsGuildID string           `json:"warcraftLogsGuildID"`
	WarcraftLogsAppID   string           `json:"warcraftLogsAppID"`
	DiscordAppID        string           `json:"discordAppID"`
	Announce            map[string]topic `json:"announce"` //Define map of thread-name of slash commands you want to announce
}

type topic struct {
	Name             string `json:"name"`
	Order            int    `json:"order"` //Prio will be from lowest > highest, so 1 is the first in order of topic
	Description      string `json:"description"`
	ShortDescription string `json:"shortDescription"`
	DateString       string `json:"dateString"`
	GIFLocalPath     string `json:"gifLocalPath"` //File path relative on the local system
	MainThread       bool   `json:"mainThread"`   //If this is true - the topic will be used on the front page of the bot channel and NOT as part of its own thread
	ToC              bool   `json:"toc"`          //ToC = Is this topic supposed to link all the threads together, forming a table of contents (ToC) in the main-thread
}

type trackPost struct {
	MessageID string `json:"messageID"`
	Active bool `json:"active"`
	ChannelID string `json:"channelID"`
	LinkedChannelID string `json:"linkedChannelID"`
	LinkedChannelName string `json:"linkedChannelName"`
}

type keyvaultToken struct {
	Name      string `json:"name"`
	VersionID string `json:"version"`
}

type keyvault struct {
	Name   string
	Tokens []keyvaultToken `json:"keyvaultToken"`
}

type messageTemplate struct {
	Name            string
	Fields          []*discordgo.MessageEmbedField
	EmojiesCaptured []emojies
	EmojieGroupType int
	EmojieGroup     []string
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
	ClassNickName  string `json:"specNickName"`
}

type classesInternal struct {
	Name          string       `json:"name"`
	ClassSpecs    []classSpecs `json:"listOfSpecs"`
	PossibleRaces []string     `json:"possibleRaces"`
}

type class struct {
	Name               string `json:"name"`
	IngameRace         string `json:"ingameRace"`
	IngameRaceEmojiID  string `json:"ingameRaceEmojiID"`
	IngameClass        string `json:"ingameClass"`
	IngameClassEmojiID string `json:"ingameClassEmojiID"`
	SpecEmoji          string `json:"specEmoji"`
	SpecEmojiID        string `json:"specEmojiID"`
	HasDouseEmoji      string `json:"HasDouseEmoji"`
	HasDouseEmojiID    string `json:"HasDouseEmojiID"`
	ClassType          string `json:"classType"`
}

type raiderProfile struct {
	Username              string                `json:"username"`
	MainCharName          string                `json:"mainCharName"`
	ID                    string                `json:"id"`
	IsOfficer             bool                  `json:"isOfficer"`
	GuildRole             discordRole           `json:"guildRole"`
	GuildRoleEmojieID     string                `json:"guildRoleEmojieID"`
	DiscordRoles          []string              `json:"discordRoles"`
	ChannelID             string                `json:"channelId"`
	ClassInfo             class                 `json:"classInfo"`
	AttendanceInfo        map[string]attendance `json:"attendance"`
	LastTimeChangedString string                `json:"lastTimeChangedRaider"`
	DateJoinedGuild       string                `json:"date_joined_guild"`
	RaidData              logsRaider            `json:"raidData"`
	MainSwitch            map[string]bool       `json:"mainSwitch"`
	BenchInfo             map[string][]bench    `json:"benchInfo"`
	TotalMainRaidsJoined  int                   `json:"totalMainRaidsJoined"`
	TotalRaidsJoined      int                   `json:"totalRaidsJoined"`
}

type raiderProfiles struct {
	GuildName             string          `json:"guildName"`
	CountOfLogs           int             `json:"countOfLogs"`
	LastTimeChangedString string          `json:"lastTimeChanged"`
	Raiders               []raiderProfile `json:"raiderProfiles"`
}

type discordRole struct {
	RoleID   string
	RoleName string
}

type consumable struct {
	Name        string
	Elixir      bool
	Flask       bool
	Other       bool
	UsuageCount int
	Uptime      int
}

type lootLog struct {
	RaidID       string //FOREIGN KEY, from
	ItemName     string
	ItemURL      string
	BISIndicator int //3 = BIS many phases, 2 = BIS 1 phase, 1 = MS upgrade
}

type trackRaid struct { //Helper struct for scanning raid-helper events efficiently on discord``
	PlayersAlreadyTracked map[string]bench `json:"playersAlreadyTracked"` //We will only track weekly benches using this field - As the automatic part that tracks benches, only tracks for current main raid (this week)
	RaidDiscordTitle      string           `json:"raidDiscordTitle"`
	DiscordMessageID      string           `json:"discordMessageID"`
	ChannelID             string           `json:"channelID"`
}

type bench struct {
	RaidLeaderName      string   //Get from raid-helper edit message of signup
	RaidLeaderDiscordID string   //Get interaction of benchreason of officer / raidleader
	Reason              string   //Submit modolar on discordgo
	DateString          string   //Get from warcraftlog
	RaidTitle           string   //Get from raid-helper edit message of signup
	RaidNames           []string //Get from warcraftlog
}

type attendance struct {
	RaidCount         int
	RaidProcent       float64
	MainRaid          bool
	RaidsMissed       []string
	LateNoticeProcent float64 //Out of the 100% of the time where a raider is ABSCENT, how many % of that time is the notice late
}

type logsRaiderBoss struct {
	Name            string
	KillCount       int
	KillTime        string //In mm:ss
	MaxTotalDamage  float64
	DPS             float64
	HPS             float64
	MaxTotalHealing float64
}

type logsRaiderParse struct {
	RaidTier      string //40 man raiding - 25 man raiding - 10 man raiding
	RankWorld     float64
	RankRegion    float64
	RankServer    float64
	GameVersion   string
	RelativeToTop float64 //% calculated from points difference in % of whats calculated in func CalculateRaidPerformance - This number will always be negative OR 0 if your top 1
	Deviation     float64 //Calculated as RelativeToTop but the value can only be POSTIVE OR NEGATIVE AND NOT 0 - as its relative to yourself from the week before
	Parse         map[string]float64
	Points        int
	Top1          bool
	Top2          bool
	Top3          bool
	Top5          bool
	BestBoss      logsRaiderBoss
	BestBossDiff  logsRaiderBoss
	WorstBoss     logsRaiderBoss
	WorstBossDiff logsRaiderBoss
	SpecName      string
}

type logsRaider struct {
	TimeOfData                  string               `json:"timeOfData"`
	CountOfRaidersInCalculation int                  `json:"countOfRaidersInCalculation"`
	URL                         string               `json:"url"`
	WorldBuffs                  map[int]logWorldBuff `json:"worldBuffs"`
	Consumes                    bool                 `json:"consumes"`
	Parses                      logsRaiderParse      `json:"parses"`
	LastRaid                    logPlayer            `json:"lastRaidStats"`
	AverageRaid                 map[string]logPlayer `json:"averageRaidStats"` //Given a period of three months \ two months \ 1 month
}

type logAllData struct { //Raw data
	RaidAverageItemLevel float64
	UniqueID             string
	Players              []logPlayer
	PlayersCount         int
	RaidTime             time.Duration
	RaidTimeString       string
	RaidStartUnixTime    int64
	RaidStartTimeString  string
	TotalDeaths          int
	MetaData             logsBase
	RaidTitle            string
	RaidNames            []string
}

type logPlayerPresent struct { //Calculate this on-demand by users / Do not cache, (customized strings NOT raw data)
	Name             string
	WarcraftLogsURL  string
	WorldBuffStatus  string
	CPMStatus        string
	ConsumesStatus   string
	RoleStatus       string
	DeathStatus      string
	AttendanceStatus string
}

type logPlayer struct {
	Name             string
	DiscordID        string
	InternalLogID    int
	Specs            []logPlayerSpec
	ClassName        string
	WarcraftLogsGUID int64
	DamageTaken      int64
	DamageDone       int64
	HealingDone      int64
	ItemLevel        int
	WorldBuffs       []logWorldBuff
	WorldBuffSummary string
	//Enchants         []logPlayerEnchant
	Deaths          []logPlayerDeath
	DeathSummary    string
	Abilities       []logPlayerAbility
	AbillitySummary string
	MinuteAPM       float64
	ActiveTimeMS    int64
	Consumables     map[string]consumable
}

type logPlayerSpec struct {
	Name     string
	TypeRole string
	MainSpec bool
}

type logDataLoss struct {
	logCodeWithErrors  string
	queryName          string
	insideFunctionName string
}

type logPlayerDeath struct {
	KilledBy               string
	TimeToDie              float64
	DamageTakenSecond      float64
	PercentageRaidComplete float64
	PartOfWipe             bool
	FirstDeath             bool
	InstaKilled            bool
	MeleeHit               bool
	FallDeath              bool
	LastBoss               bool
}

type logWorldBuff struct {
	Name               string
	WowheadID          int64
	InGame             bool
	MeleeOnly          bool
	CasterOnly         bool
	PercentUsedInRaids int
}

type logPlayerEnchant struct {
	ItemSlot     int
	Name         string
	TempoaryName string
}

type logPlayerAbility struct {
	Name       string
	Type       int
	TotalCasts int
}

type logsBase struct {
	LoggerName string `json:"loggerName"`
	Code       string `json:"code"`
	startTime  time.Time
	endTime    time.Time
}

type applicationResponse struct {
	Response *discordgo.InteractionResponse
}

type applicationCommand struct {
	Name               string
	Template           *discordgo.ApplicationCommand
	Responses          map[string]applicationResponse
	Messages           map[string]*discordgo.MessageCreate
	RequiresPriviledge bool
	Interaction        *discordgo.Interaction
}

type schedule struct {
	Name       string `json:"name"`
	HourMinute string `json:"hour_minute"`
	Weekday    time.Weekday
	Interval   int `json:"interval"`
	WeekdayInt int `json:"weekday_int"`
}

type raidHelper struct {
	ID         string
	DateString string
	LeaderID   string
	Note       string
}

// Must follow the API structure of the official documentation https://raid-helper.dev/documentation/api
type raidHelperEvent struct {
	LeaderID    string `json:"leaderId"`
	Time        string `json:"time"`
	Date        string `json:"date"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Must follow the API structure of the official documentation https://raid-helper.dev/documentation/api
type raidHelperSoftres struct {
	Instance             string `json:"instance"`
	Faction              string `json:"faction"`
	ResLimitPerCharacter int    `json:"resLimitPerCharacter"`
	ResLimitPerItem      int    `json:"resLimitPerItem"`
	DiscordProtection    bool   `json:"discordProtection"`
	HideReserves         bool   `json:"hideReserves"`
	BotBehaviour         string `json:"botBehaviour"`
}

type commingRaid struct {
	Name        string       `json:"name"`
	NextReset   string       `json:"next_reset"`
	CurrentRaid bool         `json:"current_raid"` //Is this a main raid as in the most current content out
	ResetLength int          `json:"reset_length"` //The reset timer in days, will be 3, 5 or 7
	Logger      []raidLogger `json:"raid_logger"`
}

type raidLogger struct {
	UserID string `json:"user_id"`
}

type WarcraftLogTokenCurrent struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type SyncGroupScema struct {
	ColorOfRole *int
	GroupNames  []string
}

var (
	classesImport  = []classesInternal{}
	emojiesImport  = []emojies{}
	KeyvaultConfig = keyvault{}
	configCurrent  = config{
		ServerID:            serverID,
		WarcraftLogsGuildID: strconv.Itoa(warcraftLogsGuildID),
		WarcraftLogsAppID:   warcraftLogsAppID,
		DiscordAppID:        crackedAppID,
	}

	automaticAnnounceDiscordChannel = &discordgo.Channel{}
	trackCacheChanged = make(chan struct{}, 1) //Channel will be between func AutoTrackPosts() & UseSlashCommand / Commands from the discord server

	//Used by function CalculateRaiderPerformance()
	mapOfPointScale = map[string]int{
		"pointScaleAPM/APM":                2,
		"pointScaleDeath/Death rate":       4,
		"pointScaleParse/Parse low & high": 8,
		"pointScaleStat/DPS & HPS":         8,
	}

	mapOfBossesToSkip = map[string]int{ //Should finish this - Will depend on if their are bosses in TBC that is also not regarded in warcraftlogs
		"Gothik the Harvester": 123,
	}

	mapOfPointScaleProcent = make(map[string]float64)

	messageTemplates = map[string]messageTemplate{
		"New_user": {
			Name: "New_user",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Initial screening of new player",
					Value:  "Please react 1 time to each question being posted by the bot\n\n",
					Inline: false,
				}, /*
					{
						Name:  "GETTING HELP",
						Value: "If your in doubt about your spec, let the bot help you provide spec information:\n\nType a random spec like: wowuser rogue sdaasd human puggie yes\n\nThe bogus adsds set for spec will force the bot to give spec info!",
					},
				*/
			},
		},
		"Finalize_screening_user": {
			Name: "Finalize_screening_user",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Screening complete - Thank you",
					Value:  "Please react 1 time to each question being posted by the bot\n\n",
					Inline: false,
				}, /*
					{
						Name:  "GETTING HELP",
						Value: "If your in doubt about your spec, let the bot help you provide spec information:\n\nType a random spec like: wowuser rogue sdaasd human puggie yes\n\nThe bogus adsds set for spec will force the bot to give spec info!",
					},
				*/
			},
		},
		"Ask_raider_direct_question_bwl_attunement": {
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   fmt.Sprintf("Are you **attuned** to **BWL**?%s", crackedBuiltin),
					Value:  fmt.Sprintf("\nBWL FIRST RAID FRIDAY 21-03-2025\nYou are recieving this message because your an active raider of <Hardened> %s\n", crackedBuiltin),
					Inline: false,
				},
				{
					Name:   "Please type EITHER",
					Value:  "yes OR no",
					Inline: false,
				},
				{
					Name:   "Message recieved!",
					Value:  "",
					Inline: false,
				},
			},
			EmojieGroupType: 3,
			EmojieGroup:     []string{"fun", "yes", "no"},
			EmojiesCaptured: []emojies{},
		},
	}

	slashCommandSubOptionSmallRaids = &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionBoolean,
		Name:        "includesmallraids",
		Description: "Include any Onyxia and Zul'gurub raids.",
		Required:    false,
	}

	slashCommandGeneralResponses = map[string]applicationResponse{
		"verboseMessage": {
			Response: &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
					Title: "This is an verbose message from the bot regarding:",
				},
			},
		},
		"errorMessage": {
			Response: &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
					Title: "An error occured regarding last command:",
				},
			},
		},
		"successMessage": {
			Response: &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
					Title: fmt.Sprintf("Your last command has finished successfully: %s", crackedBuiltin),
				},
			},
		},
		"buttonMessage": {
			Response: &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
					Title: "Please select a button below",
				},
			},
		},
	}

	slashCommandAllUsers = map[string]applicationCommand{
		"aboutme": {
			Template: &discordgo.ApplicationCommand{
				Name:        "aboutme",
				Description: fmt.Sprintf("See information about your main char in <Hardened> %s", crackedBuiltin),
				Version:     "1",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Name:        "logs",
						Description: "Use the 'logs' command to open a sub-menu related to your personal log data",
						Required:    false,
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionBoolean,
								Name:        "newest",
								Description: "See stats about your last main-raid's performance",
								Required:    false,
							},
							{
								Type:        discordgo.ApplicationCommandOptionInteger,
								Name:        "count",
								Description: "Specify the number of raids to include in the performance calculation",
								Required:    false,
							},
							{
								Type:        discordgo.ApplicationCommandOptionBoolean,
								Name:        "allstars",
								Description: "See information about your warcraft-logs allstar's for your main",
								Required:    false,
							},
							{
								Type:        discordgo.ApplicationCommandOptionString,
								Name:        "date",
								Description: "See stats from a specific raid, parse date e.g. 17-04-2025",
								Required:    false,
							},
						},
					},
					/*
						{
							Name:        "playerinfo",
							Description: "Use the 'playerinfo' command to see a sub-menu of options related to you",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionBoolean,
									Name:        "summary",
									Description: "See overall information about you",
									Required:    false,
								},
								{
									Type:        discordgo.ApplicationCommandOptionBoolean,
									Name:        "worldbuffs",
									Description: "See information related to your use of world-buffs in main-raids",
									Required:    false,
								},
							},
						},*/
				},
			},
			Responses: map[string]applicationResponse{
				"overallinformation": {},
			},
		},
		"howto": {
			Template: &discordgo.ApplicationCommand{
				Name:        "howto",
				Description: "Get help about how to use the bot",
			},
		},
		"myattendance": {
			Template: &discordgo.ApplicationCommand{
				Name:        "myattendance",
				Description: "View all your raid attendance details in <Hardened>",
			},
		},
		"mymissedraids": {
			Template: &discordgo.ApplicationCommand{
				Name:        "mymissedraids",
				Description: "View which specific raids you have missed",
			},
		},
		"myreminder": {
			Template: &discordgo.ApplicationCommand{
				Name:        "myreminder",
				Description: "Create an alert - The bot will notify you, once the time is up!",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "title",
						Required:    true,
						Description: "What would you like to be reminded of?",
						Type:        discordgo.ApplicationCommandOptionString,
					},
					{
						Name:        "time",
						Required:    true,
						Description: "When would you like to be alerted? In format: `1h30m15s` (Countdown) or `23:59:59` (Servertime)",
						Type:        discordgo.ApplicationCommandOptionString,
					},
				},
			},
			Responses: map[string]applicationResponse{
				"examples": {
					Response: &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Flags: discordgo.MessageFlagsEphemeral,
							Title: "Examples of how to specify time",
							Embeds: []*discordgo.MessageEmbed{
								{
									Title: fmt.Sprintf("Time format examples %s", crackedBuiltin),
									Color: blueColor,
									Fields: []*discordgo.MessageEmbedField{
										{
											Name:  "If you want a clock format (Remember it will be server time)",
											Value: "`11:00:00`\n`13:55:27`\n`12:01:00`\n(You **must** add hours, minutes & seconds)",
										},
										{
											Name:  "If you want countdown format",
											Value: "`1h30m15s`\n`1h30m`\n`1h`\n`30m10s`\n`15m`\n`50s`",
										},
									},
								},
							},
						},
					},
				},
			},
		},

		/*
			"mynewmain": {
				Template: &discordgo.ApplicationCommand{
					Name:        "mynewmain",
					Description: "Define your new main, this command must be accepted by an officer",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "oldmainname",
							Description: "Type name with same symbols, e.g. Wyzzlò",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		*/
		"myraiderperformance": {
			Template: &discordgo.ApplicationCommand{
				Name:        "myraiderperformance",
				Description: fmt.Sprintf("See general information about your mains raid-performance in %s", guildName),
			},
		},
		"hi": {
			Template: &discordgo.ApplicationCommand{
				Name:        "hi",
				Description: "Say hi to the bot from any channel to can type in!",
			},
		},
		"joke": {
			Template: &discordgo.ApplicationCommand{
				Name:        "joke",
				Description: "Make the bot do a joke in your current channel",
			},
		},
		"feedback": {
			Template: &discordgo.ApplicationCommand{
				Name:        "feedback",
				Description: "Send feedback directly to the officer team. This is GREATLY appriciated",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "subject",
						Description: "What is the subject of the matter?",
						Type:        discordgo.ApplicationCommandOptionString,
						Required:    true,
						Choices:     DefineFeedbackOptionsForTemplate(), // Depends on feedbackSubjectsSlice
					},
					{
						Name:        "anonymous",
						Description: "Do you want this feedback to be anonymous? Set to true",
						Type:        discordgo.ApplicationCommandOptionBoolean,
						Required:    true,
					},
				},
			},
			Responses: map[string]applicationResponse{
				"description": {
					Response: &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseModal,
						Data: &discordgo.InteractionResponseData{
							CustomID: "feedback_modal",
							Title:    "Submit Feedback",
							Components: []discordgo.MessageComponent{
								discordgo.ActionsRow{
									Components: []discordgo.MessageComponent{
										&discordgo.TextInput{
											CustomID:    "feedback_description",
											Label:       "Describe your feedback",
											Style:       discordgo.TextInputParagraph, // multi-line
											Placeholder: "What do you want to tell the officer team?",
											Required:    true,
											MinLength:   10,
											MaxLength:   4000, // enough for 300+ words
										},
									},
								},
							},
						},
					},
				},
			},
		}}

	slashCommandAdminUserOptions = map[string]*discordgo.ApplicationCommandOption{
		"playername": {
			Required:    false,
			Name:        "playername",
			Description: "Use @<playername> to see specific raider attendance about user",
			Type:        discordgo.ApplicationCommandOptionString,
		},
	}

	slashCommandAdminCenter = map[string]applicationCommand{
		"announcebot": {
			Template: &discordgo.ApplicationCommand{
				Name:        "announcebot",
				Description: "Run the raidautomator announcement program",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "title",
						Description: "Define a title for your announcement",
						Type:        discordgo.ApplicationCommandOptionString,
						Required: 	 true,
					},
					{
						Name:        "description",
						Description: "To add roles / players to tag, use format @Role OR @Player - Add as many tags as you want",
						Type:        discordgo.ApplicationCommandOptionString,
						Required: 	 true,
					},
					{
						Name:        "channel",
						Description: "If you want the bot to track a channel with an announcement. Use when channels gets recreated",
						Type:        discordgo.ApplicationCommandOptionString,
						Required: 	 false,
					},
				},
			},
		},
		"deletebotchannel": {
			Template: &discordgo.ApplicationCommand{
				Name:        "deletebotchannel",
				Description: "Force the bot to refresh its own bot-channel",
			},
		},
		"benchreason": {
			Template: &discordgo.ApplicationCommand{
				Name:        "benchreason",
				Description: "All benched players are automatically added but with this command, you can add a `reason`",
			},
			Responses: map[string]applicationResponse{
				"reason": {
					Response: &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseModal,
						Data: &discordgo.InteractionResponseData{
							CustomID: "bench_modal",
							Title:    "Define specific bench reasons",
							Components: []discordgo.MessageComponent{
								discordgo.ActionsRow{
									Components: []discordgo.MessageComponent{
										&discordgo.TextInput{
											CustomID:    "general_reason",
											Label:       "General reason for benching this week",
											Style:       discordgo.TextInputShort,
											Placeholder: "General bench reason",
											Required:    true,
											MaxLength:   100,
											MinLength:   10,
										},
									},
								},
								discordgo.ActionsRow{
									Components: []discordgo.MessageComponent{
										&discordgo.TextInput{
											CustomID:    "specific_reason",
											Label:       "Specific reasons (one per line)",
											Style:       discordgo.TextInputParagraph,
											Placeholder: "name=reason (one per line)",
											Required:    false,
											MaxLength:   4000,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"resetraidcache": {
			Template: &discordgo.ApplicationCommand{
				Name:        "resetraidcache",
				Description: "Force the bot to recreate the raids cache - Can take multiple minutes...",
				Version:     "1",
			},
			Responses: map[string]applicationResponse{
				"result": {
					Response: &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Title:  fmt.Sprintf("Raid-information from command 'resetRaidCache' %s", crackedBuiltin),
							Flags:  discordgo.MessageFlagsEphemeral,
							Embeds: []*discordgo.MessageEmbed{},
						},
					},
				},
			},
			RequiresPriviledge: true,
			Interaction:        &discordgo.Interaction{},
		},
		"raidsummary": {
			Template: &discordgo.ApplicationCommand{
				Name:        "raidsummary",
				Description: "Retrieve raid-summary data over an x period",
				Version:     "1",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "lastraid",
						Description: "Raid-summary data about last reset",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandOption{
							slashCommandSubOptionSmallRaids,
						},
					},
					{
						Name:        "month",
						Description: "Raid-summary data about the last 4 resets",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandOption{
							slashCommandSubOptionSmallRaids,
						},
					},
					{
						Name:        "alltime",
						Description: "Meta data about every single raid",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandOption{
							slashCommandSubOptionSmallRaids,
						},
					},
					{
						Name:        "daysorweeks",
						Description: "define in either format <number of days>d e.g. 5d OR <number of weeks> e.g. 5w\nDefaults to 30d",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandOption{
							{
								Required:    false,
								Description: "time string, eg. 5d, 20d, 30w",
								Name:        "timestring",
								Type:        discordgo.ApplicationCommandOptionString,
							},
							{
								Required:    false,
								Description: "include ony and zg",
								Name:        "includesmallraids",
								Type:        discordgo.ApplicationCommandOptionBoolean,
							},
						},
					},
				},
			},
			Responses: map[string]applicationResponse{
				"result": {
					Response: &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Title: "Raid summary information using options:",
							Flags: discordgo.MessageFlagsEphemeral,
							Embeds: []*discordgo.MessageEmbed{
								{
									Title: fmt.Sprintf("Command successfully ran %s", crackedBuiltin),
									Color: greenColor,
								},
							},
						},
					},
				},
				"resultraidsmerged": {
					Response: &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Title: "Raid summary merged raids:",
							Flags: discordgo.MessageFlagsEphemeral,
							Embeds: []*discordgo.MessageEmbed{
								{
									Title: fmt.Sprintf("Logs has been split into seperate outputs %s", crackedBuiltin),
									Color: greenColor,
								},
							},
						},
					},
				},
			},
		},
		"simplemessage": {
			Template: &discordgo.ApplicationCommand{
				Name:        "simplemessage",
				Description: "Parse a string value - If you want to include emojies and or tags, simply add them to the text",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Required:    true,
						Name:        "value",
						Type:        discordgo.ApplicationCommandOptionString,
						Description: "value of the string to sent to the channel",
					},
				},
			},

			Responses: map[string]applicationResponse{
				"messagetouser": {
					Response: &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{
								{
									Title: fmt.Sprintf("Message from the bot %s", crackedBuiltin),
									Color: blueColor,
								},
							},
						},
					},
				},
			},
		},
		"deletechannelcontent": {
			Template: &discordgo.ApplicationCommand{
				Name:        "deletechannelcontent",
				Description: "Delete all the content in a given channel",
			},
		},
		"seeraiderattendance": {
			Template: &discordgo.ApplicationCommand{
				Name:        "seeraiderattendance",
				Description: "Get a full overview over all raiders attendance",
				Options:     slashCommandAdminUserOptions["playername"].Options,
			},
		},
		"seeraidermissedraids": {
			Template: &discordgo.ApplicationCommand{
				Name:        "seeraidermissedraids",
				Description: "See a specific raiders missed raids the last 3 months",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "playername",
						Required:    true,
						Description: "Use @<playername> to see specific raider attendance about user",
						Type:        discordgo.ApplicationCommandOptionString,
					},
				},
			},
		},
		"seebench": {
			Template: &discordgo.ApplicationCommand{
				Name:        "seebench",
				Description: "See an overview of all stats related to raiders being benched from raids the last week",

				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "period",
						Required:    true,
						Description: "Choose the period from the list",
						Type:        discordgo.ApplicationCommandOptionString,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{
								Name:  "Last week",
								Value: "lastWeek",
							},
							{
								Name:  "Last 4 weeks",
								Value: "oneMonth",
							},
							{
								Name:  "Last 3 months",
								Value: "threeMonth",
							},
							{
								Name:  "Since guild startet",
								Value: "start",
							},
						},
					},
					slashCommandAdminUserOptions["playername"],
				},
			},
		},
		"updateweeklyattendance": {
			Template: &discordgo.ApplicationCommand{
				Name:        "updateweeklyattendance",
				Description: "Manually update the weekly attendance for raiders, cracked will automatically do it fridays at 12:00",
			},
		},
		"promotetrial": {
			Template: &discordgo.ApplicationCommand{
				Name:        "promotetrial",
				Description: "Promote a specific trial to raider - It should ONLY be the GM or recruit leader that does this...",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "trialname",
						Required:    true,
						Type:        discordgo.ApplicationCommandOptionString,
						Description: "Use @<playername> to see specific raider attendance about user",
					},
				},
			},
		},
		"syncdiscordroles": {
			Template: &discordgo.ApplicationCommand{
				Name:        "syncdiscordroles",
				Description: "Sync all raiders into specific category roles takes several minutes",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "fullsync",
						Required:    true,
						Type:        discordgo.ApplicationCommandOptionBoolean,
						Description: "Perform a full sync of all raiders - will take several minutes to complete...",
					},
				},
			},
		},
	}
	/*
		slashCommandTemplates = map[string]applicationCommand{
			"playerinfo": {
				Template: &discordgo.ApplicationCommand{
					GuildID:     serverID,
					Name:        "warcraftlogs",
					Description: "Get information about you from Warcraftlogs",
					Version:     "1",
					//DefaultMemberPermissions: GetIntPointer(10),
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "general-information",
							Description: "Get general information about your parse",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
					},
				},
				Responses: map[string]applicationResponse{
					"general-information": {
						Response: &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Flags: discordgo.MessageFlagsEphemeral,
								Embeds: []*discordgo.MessageEmbed{ //
									{
										Title:       "General info from Warcraftlogs",
										Description: "This information is ONLY calculated in the perspective of your main",
										URL:         "https://www.fresh.warcraftlogs.com/character/eu/thunderstrike/", //We need to add the char-name when the template is executed
										Color:       0x00ff00,                                                         // Green color
										Fields: []*discordgo.MessageEmbedField{
											{
												Name:  "Average parse",
												Value: "",
											},
											{
												Name:  `Average % performance`,
												Value: "",
											},
											{
												Name:  "Last time died besides a wipe",
												Value: "",
											},
											{
												Name:  "World buffs stats",
												Value: "",
											},
											{
												Name:  "Attendance % of main-raids since joining the guild",
												Value: "",
											},
										},
									},
								}},
						},
					},
				},
			},
		}
	*/
	mapOfWarcaftLogsQueries = map[string]map[string]any{
		"playerRankings": {
			"query": `
				query GetCharacter($name: String!, $serverSlug: String!, $serverRegion: String!) {
					characterData {
						character(name: $name, serverSlug: $serverSlug, serverRegion: $serverRegion) {
							id
							name
							classID
							faction {
								name
							}
							level
							guilds {
								name
							}
							zoneRankings
							encounterRankings
						}
					}
				}`,
			"variables": map[string]string{
				"name":         "",
				"serverSlug":   "thunderstrike",
				"serverRegion": "eu",
			},
		},
		"guildLogsRaidIDs": {
			"query": `query GuildLogs($guildID: Int!, $page: Int!) {
				reportData {
					reports(guildID: $guildID, page: $page) {
						data {
							code
							title
							startTime
							endTime
							owner {
								name
							}
						}
					}
				}
			}`,
			"variables": map[string]any{
				"guildID": warcraftLogsGuildID, // Replace with the actual Guild ID from Step 1
				"page":    1,
			},
		},
		"allFightIDsForRaid": {
			"query": `query GetFights($code: String!) {
				reportData {
					report(code: $code) {
						fights {
							id
						}
					}
				}
			}`,
			"variables": map[string]interface{}{
				"code": "", // Replace with the actual log code
			},
		},
		"logsByOwnerAndCode": {
			"query": `query GetLogByCode($code: String!, $fightIDs: [Int!]!) {
				reportData {
					report(code: $code) {
					code
					title
					startTime
					zone {
						id
						name
					}
					fights {
						id
						encounterID
						startTime
						endTime
						kill
					}
					masterData {
						actors {
						id
						name
						type
						subType
						}
					}
					owner {
						id
						name
					}
					players: table(dataType: Summary, fightIDs: $fightIDs) 
					combatantInfo: events(dataType: CombatantInfo, startTime: 0, endTime: 999999999) {
						data
					}
					deaths: events(dataType: Deaths, startTime: 0, endTime: 999999999) {
						data
					}
					buffs: events(dataType: Buffs, startTime: 0, endTime: 999999999) {
						data
					}
					damageDone: events(dataType: DamageDone, startTime: 0, endTime: 999999999) {
						data
					}
					healingDone: events(dataType: Healing, startTime: 0, endTime: 999999999) {
						data
					}
					resources: events(dataType: Resources, startTime: 0, endTime: 999999999) {
						data
					}
					buffUptimes: table(dataType: Buffs, fightIDs: $fightIDs)
					castsSummary: table(dataType: Casts, fightIDs: $fightIDs)
					deathSummary: table(dataType: Deaths, fightIDs: $fightIDs)
					}
				}
			}`,
			"variables": map[string]interface{}{
				"code":     "", // Replace with the actual log code
				"fightIDs": []int64{},
			},
		},
		"reportActorBuffs": {
			"query": `query GetActorBuffs($code: String!, $fightIDs: [Int!]!, $actorID: Int!) {
				reportData {
					report(code: $code) {
						reportActor(fightIDs: $fightIDs, id: $actorID) {
							buffs(startTime: 0, endTime: 999999999) {
								ability {
									guid
									name
								}
								totalUses
								totalUptime
								bands {
									startTime
									endTime
								}
							}
						}
					}
				}
			}`,
			"variables": map[string]interface{}{
				"code":     "",        // Log code (string)
				"fightIDs": []int64{}, // Fights you want to target
				"actorID":  0,         // Loop over this per actor
			},
		},
		"logsByEncounterID": {
			"query": `query GetEncounterInfo($encounterID: Int!) {
				worldData {
					encounter(id: $encounterID) {
						id
						name
						zone {
							id
							name
						}
					}
				}
			}`,
			"variables": map[string]any{
				"encounterID": 0,
			},
		},
	}

	//Define the time where the Wow-guild startet to log
	timeGuildStarted = "November 15, 2024 18:00:00"

	// Mapping constant names to values
	mapOfConstantRoles = map[string]string{
		"roleDruid":   roleDruid,
		"roleHunter":  roleHunter,
		"roleMage":    roleMage,
		"rolePaladin": rolePaladin,
		"rolePriest":  rolePriest,
		"roleRogue":   roleRogue,
		"roleWarlock": roleWarlock,
		"roleWarrior": roleWarrior,
	}

	mapOfConstantClasses = map[string]string{
		"channelDruid":   channelDruid,
		"channelHunter":  channelHunter,
		"channelMage":    channelMage,
		"channelPaladin": channelPaladin,
		"channelPriest":  channelPriest,
		"channelRogue":   channelRogue,
		"channelWarlock": channelWarlock,
		"channelWarrior": channelWarrior,
	}

	mapOfConstantOfficers = map[string]string{
		"officerGMArlissa": officerGMArlissa,
		"officerPriest":    officerPriest,
		"officerRogue":     officerRogue,
		"officerWarrior":   officerWarrior,
		"officerMage":      officerMage,
		"officerDruid":     officerDruid,
	}

	mapOfLoggers = map[string]string{
		"Throyn1986": SplitOfficerName(officerPriest)["ID"],
		"Zyrtec":     officialLogger1,
		"Shufflez26": SplitOfficerName(officerGMArlissa)["ID"],
	}

	feedbackSubjectsSlice = []string{
		"Loot system",
		"Raids",
		"Raid-leading",
		"Officers",
		"General",
		"Our discord bot",
		"Motivation",
		"Issues between guildies",
		"Whistleblower",
	}

	skippedBosses = []string{"gothik the harvester"}

	classesPath             = baseCachePath + "classes.json"
	keyvaultPath            = baseCachePath + "keyvault.json"
	emojiesPath             = baseCachePath + "emojies.json"
	cacheTrackedPostsCache =  baseCachePath + "cache_tracked_posts.json"
	configPath              = baseCachePath + "config.json"
	raidHelperEventsPath    = baseCachePath + "raid_helper_events.json"
	belowRaidersCachePath   = baseCachePath + "cache_trials_pugs.json"
	raidersCachePath        = baseCachePath + "cache_raiders.json"
	raiderProfilesCachePath = baseCachePath + "cache_raider_profiles.json"
	raidHelperCachePath     = baseCachePath + "cache_raid_helper.json"
	raidCachePath           = baseCachePath + "cache_raids.json" // Will be the largest file due to warcraftlogs info
	raidAllDataPath         = baseCachePath + "cache_raid_all_data.json"
	informationLogPath      = baseCachePath + "information_log.json" // Will grow over time
	//errorLogPathWarcraftLogs = baseCachePath + "warcraft_logs_query_errors.json" // Will grow over time
	errorLogPath       = baseCachePath + "error_log.json" // Will grow over time
	customSchedulePath = baseCachePath + "custom_schedules.json"

	ScheduledEvents = []schedule{ //NIL
		{
			Name:       "updateweeklyattendance",
			HourMinute: "12:00",
			Weekday:    time.Friday,
			Interval:   7,
		},
		/*
			{
				Name:       "sign1",
				HourMinute: "20:00",
				Weekday:    time.Friday,
				Interval:   7,
			},
			{
				Name:       "sign2",
				HourMinute: "15:00",
				Weekday:    time.Sunday,
				Interval:   7,
			},
			{
				Name:       "cleanup",
				HourMinute: "19:45",
				Weekday:    time.Thursday,
				Interval:   7,
			},
		*/
	}

	mapOfTokens = map[string]string{
		"Bot":           "",
		"Raid_helper":   "",
		"Warcraft_logs": "",
	}

	knownWorldBuffs = map[int]logWorldBuff{
		355363: {
			Name:      "Rallying Cry of the Dragonslayer",
			WowheadID: 355363,
			InGame:    true,
		},
		15366: {
			Name:      "Songflower Serenade",
			WowheadID: 15366,
			InGame:    true,
		},
		22820: {
			Name:       "Slip'kik's Savvy",
			WowheadID:  22820,
			InGame:     true,
			CasterOnly: true,
		},
		22817: {
			Name:      "Fengus' Ferocity",
			WowheadID: 22817,
			InGame:    true,
			MeleeOnly: true,
		},
		22818: {
			Name:      "Mol'dar's Moxie",
			WowheadID: 22818,
			InGame:    true,
		},
		24425: {
			Name:      "Spirit of Zandalar",
			WowheadID: 24425,
			InGame:    false,
		},
		355366: {
			Name:      "Warchief's Blessing",
			WowheadID: 355366,
			InGame:    true,
			MeleeOnly: true,
		},
		23768: {
			Name:      "Sayge's Dark Fortune of Damage",
			WowheadID: 23768,
			InGame:    true,
		},
	}

	knownConsumables = map[int]string{
		17626: "Flask of the Titans",
	}

	mapOfMergedGroups = map[string]SyncGroupScema{
		"Melee": {
			ColorOfRole: GetIntPointer(0x553A46),
			GroupNames: []string{
				"Warrior",
				"Rogue",
				"Druid",
				"Shaman",
				"Paladin",
			},
		},
		"Ranged": {
			ColorOfRole: GetIntPointer(0x3498db),
			GroupNames: []string{
				"Mage",
				"Warlock",
				"Druid",
				"Shaman",
				"Hunter",
			},
		},
		"Healer": {
			ColorOfRole: GetIntPointer(0x979c9f),
			GroupNames: []string{
				"Druid",
				"Priest",
				"Paladin",
				"Shaman",
			},
		},
		"Tank": {
			ColorOfRole: GetIntPointer(0x422f04),
			GroupNames: []string{
				"Druid",
				"Warrior",
				"Paladin",
			},
		},
	}

	tankAbillities = []string{
		"Taunt",
		"Growl",
		"Righteous Defense",
		"Revenge",
	}

	mapOfRoles = make(map[string]string)

	BotSessionMain = &discordgo.Session{}
	

	greenColor  = 0x00FF00 // Pure Green
	yellowColor = 0xFFFF00 // Pure Yellow
	redColor    = 0xFF0000 // Pure Red
	blueColor   = 0x0000FF // Pure Blue

	raiderCacheMutex       sync.Mutex
	raidHelperCascheMutex  sync.Mutex
	postTrackMutex         sync.Mutex
	errorLogMutex          sync.Mutex
	configCacheMutex       sync.Mutex
	MapOfUserDefinedAlerts sync.Map

	GuildStartTime time.Time
)

const (
	serverID            = "630793944632131594"
	warcraftLogsGuildID = 773986
	botName             = "raid-automater"
	guildName           = "Hardened"
	channelInfo         = "1308521695564402899"
	channelFeedback     = "1441245331214958625"
	channelLog          = "1318700380900823103"
	channelGeneral      = "1308521052036530291"
	channelGearCheck    = "1396200186208063608"
	channelVoting       = "1316379489906855936"
	channelSignUp       = "1346922479951675483"
	channelSignUpPug    = "1334949433208606791"
	channelSignUpNaxx   = "1418598263782899832"
	channelWelcome      = "1309312094822203402"
	channelBot          = "1336098468615426189"
	channelServerRules  = "1312791528267186216"
	channelOfficer      = "1308522605065539714"

	channelNameAnnouncement = "bot-assistance-🤖"

	googleSheetBaseURL = "https://docs.google.com/spreadsheets/d/1wlRwuKusSL01MReBgpbFXyat13LMZ6dtlgk5aN4Ruq0"

	channelDruid   = "667826282678976515"
	channelHunter  = "667826319333130260"
	channelMage    = "1308522161916481667"
	channelPaladin = "667826420956921856"
	channelPriest  = "667826515601391646"
	channelRogue   = "667826598678102023"
	channelWarlock = "1308522389176324139"
	channelWarrior = "1308522446500008006"

	crackedBuiltin     = "<:cracked:1312847304725893190>"
	antiCrackedBuiltin = "<:anticracked:1344269727928680508>"
	crackedAppID       = "1331000695305801858"
	thumbsUp           = "👍"
	thumbsDown         = "👎"
	ony                = "<:ony:1355867899071565835>"
	mc                 = "<:mc:1355865300951892008>"
	bwl                = "<:bwl:1355867897431593000>"

	categoryBot        = "1336097759073140920"
	categoryAssistance = "1465470731893866526"

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

	roleOfficer    = "1309512897612746752"
	roleRaidLeader = "1308525656031625317"

	officerGMArlissa = "346353264461217795/Arlissa"
	officerPriest    = "812709542554370098/Throyn"
	officerRogue     = "655113437327917065/Akasuna"
	officerWarrior   = "626123398681985054/Joebaldo"
	officerMage      = "232480016854679553/Dumblydore"
	officerDruid     = "231066682842415105/Sleepybear"

	officialLogger1 = "276387587155820544" //Zyrtek

	raidHelperEventBaseURL = "https://raid-helper.dev/api/v2/events/"
	raidHelperId           = "579155972115660803"

	baseCachePath = "./"

	permissionViewChannel    = int64(1 << 0)  // 1
	permissionSendMessages   = int64(1 << 10) // 1024
	permissionReadMessages   = int64(1 << 11) // 2048
	permissionManageMessages = int64(1 << 13) // 8192

	timeLayout      = "January 2, 2006 15:04:05"
	timeLayoutLogs  = "02-01-2006 15:04:05"
	timeLayOutShort = "02-01-2006"

	warcraftLogsAppID      = "a09d4ea0-ff4f-4d30-8d8b-f25f30303c8d"
	warcraftLogsServerSlug = "thunderstrike"
	warcraftLogsRegion     = "EU"
	warcraftLogsNativeID   = "1335683225263018024"

	azureStorageURI = "raiderbuild.blob.core.windows.net"

	logWarningsName    = "information_log.json"
	logErrorName       = "error_log.json"
	azureContainerName = "logs"
)

func init() {
	fmt.Println("THIS IS VERSION 1.0.0")
	CheckRuntime()
}

func CheckRuntime() {
	err := os.WriteFile(logWarningsName, []byte{}, 0644)
	if err != nil {
		log.Fatalf("An error occured while trying to create file %s Please make sure the program has write access to the folder, error is: %s", logWarningsName, err)
	}
	err = os.WriteFile(logErrorName, []byte{}, 0644)
	if err != nil {
		log.Fatalf("An error occured while trying to create file %s Please make sure the program has write access to the folder, error is: %s", logErrorName, err)
	}
	time.Sleep(5 * time.Second)

	ImportKeyvaultConfig()
	WriteInformationLog("Keyvault config successfully imported during start-up", "Import Keyvault config")
	ImportEmojies()
	WriteInformationLog("Emojie config successfully imported during start-up", "Import Emojie config")
	ImportClasses()
	WriteInformationLog("Class config successfully imported during start-up", "Import Class config")

	azCred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		WriteErrorLog("An error occured while trying to retrieve the default system assigned managed identity: Inside function CheckRuntime", err.Error())
	} else {
		WriteInformationLog("Default system assigned identity successfully retrieved during start-up", "Import Identity")
	}

	azKeyvaultClient, err := azsecrets.NewClient(KeyvaultConfig.Name, azCred, nil)

	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to start a new key vault client to: %s", KeyvaultConfig.Name), err.Error())
	} else {
		WriteInformationLog("Keyvault client successfully established during start-up", "Import Keyvault client")
	}

	for _, tokenConfig := range KeyvaultConfig.Tokens {
		secret, err := azKeyvaultClient.GetSecret(context.TODO(), tokenConfig.Name, "", nil)
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve the specific secret for %s", tokenConfig.Name), err.Error())
		} else {
			mapOfTokens[tokenConfig.Name] = *secret.Value
		}
	}

	marshal := GetHttpResponseData("POST", "", "https://www.warcraftlogs.com/oauth/token", nil, true)
	if mapOfToken, ok := marshal.(map[string]any); ok {
		for attributeName, value := range mapOfToken {
			if attributeName == "access_token" {
				mapOfTokens["warcraftLogsRefreshToken"] = value.(string)
			}
		}
	}

	if _, ok := mapOfTokens["warcraftLogsRefreshToken"]; !ok {
		log.Fatalf("The warcraftlogs token could not be obtained and therefor the application must stop. See the error log at %s during startup", errorLogPath)
	}
	customSchedules := RetrieveCustomSchedules()
	if len(customSchedules) > 0 {
		if !(len(customSchedules) == 1 && customSchedules[0].Name == "empty") {
			removeEmpty := []schedule{}
			for x, schedule := range customSchedules {
				if schedule.Name != "empty" {
					customSchedules[x].Weekday = time.Weekday(schedule.WeekdayInt)
					removeEmpty = append(removeEmpty, customSchedules[x])
				}
			}
			ScheduledEvents = append(ScheduledEvents, removeEmpty...)
		}
	} else {
		WriteInformationLog("No custom schedules found, continuing...", "No custom schedules")
	}

	//err = StorageAccountAppendBlob(logWarningsName, azureContainerName, logWarningsName, StorageAccountClient(azureStorageURI), azContext)
	if err != nil {
		//log.Fatalf("An error occured while trying to handle the storage setup for the application %s", err.Error())
	}

	/*
		Test different required connections for bot:
		Discord server itself
		Raid-helper API
		WarcraftLogs API
	*/
	WriteInformationLog(fmt.Sprintf("Bot %s successfully established a connection with server id %s", botName, serverID), "Connect to Discord")
	//RetriveRaidHelperEvent(BotSessionMain, true)
	WriteInformationLog(fmt.Sprintf("Bot %s successfully established a connection with the raid-helper API", botName), "Connect to Raid-helper")
	WriteInformationLog("The system is OK to start - Running main() in 5 seconds...", "System-startup OK")
	time.Sleep(5 * time.Second)
}

func main() {
	BotSessionMain = NewDiscordSession(false)
	defer BotSessionMain.Close()
	var err error
	GuildStartTime, err = time.Parse(timeLayout, timeGuildStarted)
	if err != nil {
		WriteErrorLog("An error occured while trying to parse the guilds start-time as time.Time type, during main(), the program will stop...", err.Error())
		log.Fatalf("The guild-start-time of '%s' Is not valid, please set the constant 'timeGuildStarted' In format '%s'", timeGuildStarted, timeLayout)
	}
	/*
		SIGNALS BELOW
	*/
	ImportEmojies()
	ImportClasses()
	//NotifyPlayerRaidQuestion((PrepareTemplateWithEmojie(messageTemplates["Ask_raider_direct_question_douse"])), BotSessionMain)
	NewPlayerJoin(BotSessionMain)
	//AutoTrackRaidEvents(BotSessionMain)
	NewSlashCommand(BotSessionMain)
	UseSlashCommand(BotSessionMain) //Contains go-routines
	DeleteOldSlashCommand(BotSessionMain)
	CalculateRaidWeightsProcent()
	go AutoTrackPosts()
	go AutoAnnounceTracker(5 * time.Second, BotSessionMain) //Contains go-routines

	AutoUpdateRaidLogCache(BotSessionMain, []string{})
	go DeleteOldBotChannels(1, 30, BotSessionMain)

	if profiles := ReadWriteRaiderProfiles(nil, true); len(profiles) == 0 {
		InitializeDiscordProfiles(InitializeRaiderProfiles(), BotSessionMain, true) //Retrieve ALL raiders from ANY time since the guild startet logging
	}
	existing := CheckForExistingCache(raidAllDataPath)
	logs := []logAllData{}
	json.Unmarshal(existing, &logs)
	for x, raid := range logs {
		fmt.Println(x, "raid name:", raid.RaidTitle)
	}
	customSchedules := RetrieveCustomSchedules()
	if len(customSchedules) > 0 {
		if !(len(customSchedules) == 1 && customSchedules[0].Name == "empty") {
			removeEmpty := []schedule{}
			for x, schedule := range customSchedules {
				if schedule.Name != "empty" {
					customSchedules[x].Weekday = time.Weekday(schedule.WeekdayInt)
					removeEmpty = append(removeEmpty, customSchedules[x])
				}
			}
			ScheduledEvents = append(ScheduledEvents, removeEmpty...)
		}
	} else {
		WriteInformationLog("No custom schedules found, continuing...", "No custom schedules")
	}

	//NotifyPlayerRaidPlan(BotSessionMain)
	for x, taskSchedule := range ScheduledEvents {
		fmt.Println(x, " -- ", "Task:", taskSchedule, "Time", taskSchedule.HourMinute, "Day", taskSchedule.Weekday.String())
		switch taskSchedule.Name {
		case "updateweeklyattendance":
			{
				RunAtSpecificTime(func() {
					WriteInformationLog(AddWeeklyRaiderAttendance(), "Updating weekly attendance")
				}, taskSchedule, false)
			}
		}
	}
	//fmt.Println(len(GetAllWarcraftLogsRaidData(false, true)))
	//Since we are running inside a PaaS service, we will never stop unless forced
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}

func CalculateRaidWeightsProcent() {
	totalWeights := 0
	for _, weightInt := range mapOfPointScale {
		totalWeights += weightInt
	}
	if totalWeights == 0 {
		WriteErrorLog("Either the total weights number %d or the count of weights %d is 0, which means the bot cannot calculate raider performance - Please make sure all weights are present in runtime + none of them are 0 or less, during the function CalculateRaidWeightsProcent()", "Raid-performance weights 0")
		return
	}
	for weightName, weightInt := range mapOfPointScale {
		mapOfPointScaleProcent[weightName] = float64(weightInt) / float64(totalWeights) * 100
	}
}

func SplitOfficerName(officerName string) map[string]string {
	mapOfNames := make(map[string]string)
	splitNames := strings.Split(officerName, "/")
	if len(splitNames) != 2 {
		WriteErrorLog(fmt.Sprintf("This function will only accept values of type string in format name/id but got %s, during the function SplitOfficerName()", officerName), "Wrong format of string")
		mapOfNames[officerName] = officerName
		return mapOfNames
	}
	mapOfNames["ID"] = splitNames[0]
	mapOfNames["Name"] = splitNames[1]
	return mapOfNames
}

func DefineFeedbackOptionsForTemplate() []*discordgo.ApplicationCommandOptionChoice {
	returnCommandOptionChoice := []*discordgo.ApplicationCommandOptionChoice{}
	for _, subject := range feedbackSubjectsSlice {
		choice := &discordgo.ApplicationCommandOptionChoice{
			Name:  subject,
			Value: subject,
		}
		returnCommandOptionChoice = append(returnCommandOptionChoice, choice)
	}
	return returnCommandOptionChoice
}

func ManageMergedGroups(session *discordgo.Session, syncType string) []string {
	mapOfPlayers := make(map[string]bool)
	mapOfPlayerStatus := make(map[string]bool)
	mapOfDeletedRoles := make(map[string]bool)
	lookbackDuration := time.Hour * 24 * 14 //Default is 2 weeks for Delta sync
	count := 0
	countMerged := 0
	countNotMerged := 0

	rolesAll, err := session.GuildRoles(serverID)
	if err != nil {
		WriteErrorLog("An error occured while trying to retrive all guild roles, during the function ResolveGroupName()", err.Error())
		return []string{fmt.Sprintf("An internal error occured - Please contact %s", SplitOfficerName(officerGMArlissa)["Name"])}
	}

	for _, role := range rolesAll {
		mapOfRoles[role.Name] = role.ID
	}

	rootRoleNames := []string{}

	for name := range mapOfMergedGroups {
		rootRoleNames = append(rootRoleNames, name)
	}

	if syncType == "full" {
		lookbackDuration = 30 * 24 * time.Hour
		for _, mergedGroupName := range rootRoleNames {
			roleID := mapOfRoles[mergedGroupName]
			if roleID != "" && !mapOfDeletedRoles[mergedGroupName] {
				err := session.GuildRoleDelete(serverID, roleID)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to delete channel %s, during the function ManageMergedGroups()", mergedGroupName), err.Error())
					return []string{fmt.Sprintf("An internal error occured - Please contact %s")}
				}
				mentionable := true
				newRole, err := session.GuildRoleCreate(serverID, &discordgo.RoleParams{
					Name:        mergedGroupName,
					Mentionable: &mentionable,
					Color:       mapOfMergedGroups[mergedGroupName].ColorOfRole,
				})
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to create channel %s, during the function ManageMergedGroups()", mergedGroupName), err.Error())
					return []string{fmt.Sprintf("An internal error occured - Please contact %s", SplitOfficerName(officerGMArlissa)["Name"])}
				}
				WriteInformationLog(fmt.Sprintf("The channel %s has been successfully created, during the function ManageMergedGroups()", mergedGroupName), "Successfully creating discord channel")
				mapOfDeletedRoles[mergedGroupName] = true
				mapOfRoles[mergedGroupName] = newRole.ID
			}
		}
	}

	raidMembers := RetrieveUsersInRole([]string{roleRaider, roleTrial}, session) //SLICE OF IDS

	raidCache, err := ReadRaidDataCache((time.Now().Add(-lookbackDuration)), true)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to read the cache %s, during the function ManageMergedGroups()", raidAllDataPath), err.Error())
		return []string{fmt.Sprintf("An internal error occured - Please contact %s", SplitOfficerName(officerGMArlissa)["Name"])}
	}
	for _, log := range raidCache {
		for _, player := range log.Players {
			if !mapOfPlayers[player.Name] {
				raider := false
				raiderDiscordID := ResolvePlayerName(player.Name, session)
				if slices.Contains(raidMembers, raiderDiscordID) {
					raider = true
					count++
				}

				if !raider {
					WriteInformationLog(fmt.Sprintf("Player skipped: %s due to the user not being a raider / trial", player.Name), "Player not added to group")
					mapOfPlayerStatus[player.Name] = raider
					continue
				}

				for _, spec := range player.Specs {
					if spec.TypeRole != "dps" && raider {
						mapOfPlayerStatus[player.Name] = raider
						err := session.GuildMemberRoleAdd(serverID, ResolvePlayerName(player.Name, session), mapOfRoles[spec.TypeRole])
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to add player: %s to group %s, during the function ManageMergedGroups()", player.Name, spec.TypeRole), err.Error())
							continue
						}
						WriteInformationLog(fmt.Sprintf("Player: %s has been successfully added to merged group %s", player.Name, spec.TypeRole), "Successfully added player to group")
					}
				}
				mapOfPlayers[player.Name] = true
			}
		}
	}

	var returnString strings.Builder
	returnString.WriteString("```\n")
	returnString.WriteString("Player           | Raider |\n")
	returnString.WriteString("-----------------|--------|\n")
	for playerName, isRaider := range mapOfPlayerStatus {
		if !isRaider {
			fmt.Fprintf(&returnString, "%-16s | %-6s\n", playerName, "No")
			countNotMerged++
		}
	}

	for playerName, isRaider := range mapOfPlayerStatus {
		if isRaider {
			fmt.Fprintf(&returnString, "%-16s | %-6s\n", playerName, "Yes")
			countMerged++
		}
	}
	returnString.WriteString("```\n")
	returnString.WriteString(fmt.Sprintf("\nTotal number of raiders merged: **%d**", count))

	if len([]rune(returnString.String())) > 1999 {
		fmt.Println("NUMBER OF RUNES:", len([]rune(returnString.String())))
		return []string{fmt.Sprintf("\nTotal number of raiders merged: **%d**", count)}
	}

	return []string{returnString.String(), "Completed successfully"}
}

func DeleteOldBotChannels(timeInMinutes int, maxAwaitInMinutes int, session *discordgo.Session) {
	timeInterval := time.Duration(timeInMinutes) * time.Minute
	timeTicker := time.NewTicker(timeInterval)
	defer timeTicker.Stop()
	for range timeTicker.C {
		playersInTempRole := RetrieveUsersInRole([]string{roleTemp}, session)
		botChannels, err := session.GuildChannels(serverID)
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to retrive all guild channels from discord server %s, during the function DeleteOldBotChannels()", serverID), err.Error())
		}
		timeNow := time.Now()
		for _, channel := range botChannels {
			if strings.Contains(channel.Name, "automatic-") {
				allChannelMessages, _ := session.ChannelMessages(channel.ID, 100, "", "", "")
				timeLastMessage := allChannelMessages[0].Timestamp
				timeDuration := timeNow.Sub(timeLastMessage)
				if timeDuration > time.Duration(maxAwaitInMinutes)*time.Minute {
					_, err := session.ChannelDelete(channel.ID)
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to delete channel %s %s, during the function DeleteOldBotChannels()", channel.ID, channel.Name), err.Error())
						continue
					}
					WriteInformationLog(fmt.Sprintf("The bot channel was successfully deleted - %s %s, during the function DeleteOldBotChannels()", channel.ID, channel.Name), "Successfully deleted channel")
					regexID := regexp.MustCompile(`<@(\d+)>`)
					messageContent := allChannelMessages[len(allChannelMessages)-2].Content //The last element in the array is empty - Second last is the actual message
					playerIDSlice := regexID.FindStringSubmatch(messageContent)
					if len(playerIDSlice) < 1 {
						WriteErrorLog(fmt.Sprintf("No player ID was found in the first message from the bot, so the user cannot be deleted - Message %s, during the function DeleteOldBotChannels()", messageContent), err.Error())
						continue
					}
					playerName := ResolvePlayerID(playerIDSlice[1], session)
					for _, userID := range playersInTempRole {
						if userID == playerIDSlice[1] {
							err = session.GuildMemberDelete(serverID, userID)
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to delete the user %s, during the function DeleteOldBotChannels()", playerName), err.Error())
								break
							}
							WriteInformationLog(fmt.Sprintf("Successfully kicked user: %s", playerName), "Successfully deleted user")
							break
						}
					}
				}
			}
		}
	}
}

func NewInteractionResponseToSpecificCommand(logType int, data string, discordType ...discordgo.InteractionResponseType) discordgo.InteractionResponse {
	messageSlice := strings.Split(data, "|")
	templateCopy := discordgo.InteractionResponse{}
	if len(messageSlice) <= 1 {
		WriteInformationLog("The data provided for the function NewInteractionResponseToSpecificCommand() is missing parts. Please use format commandName/Output", "Create Slash Command response")
		if data == "" {
			WriteInformationLog("No data provided for the function NewInteractionResponseToSpecificCommand() - Data is crucial for this function, so it will return early...", "Return early")
		}
		return discordgo.InteractionResponse{}
	}

	if len(discordType) == 0 {
		templateCopy.Type = discordgo.InteractionResponseChannelMessageWithSource
	} else {
		templateCopy.Type = discordType[0]
		if templateCopy.Data == nil {
			templateCopy.Data = &discordgo.InteractionResponseData{}
		}
		templateCopy.Data.Flags |= discordgo.MessageFlagsEphemeral
		return templateCopy
	}

	switch logType {
	case 0:
		{
			templateCopy = *slashCommandGeneralResponses["errorMessage"].Response
			messageErrorResponse := &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Embeds: []*discordgo.MessageEmbed{
					{
						Color:       redColor,
						Title:       fmt.Sprintf("Error for command call: %s", messageSlice[0]),
						Description: fmt.Sprintf("About the error: %s", messageSlice[1]),
					},
				},
			}

			templateCopy.Data = messageErrorResponse
		}
	case 1:
		{
			templateCopy = *slashCommandGeneralResponses["verboseMessage"].Response
			messageVerboseResponse := &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Embeds: []*discordgo.MessageEmbed{
					{
						Color:       yellowColor,
						Title:       fmt.Sprintf("Progress on your command: %s", messageSlice[0]),
						Description: "This is general information about the progress of your last command:",
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:  "Progress information:",
								Value: fmt.Sprintf("Message: %s", messageSlice[1]),
							},
						},
					},
				},
			}
			templateCopy.Data = messageVerboseResponse
		}
	case 2:
		{
			templateCopy = *slashCommandGeneralResponses["successMessage"].Response
			messageSuccessResponse := &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Embeds: []*discordgo.MessageEmbed{
					{
						Color:       greenColor,
						Title:       fmt.Sprintf("Your command has completed: %s", messageSlice[0]),
						Description: fmt.Sprintf("This might be part of a larger message %s", crackedBuiltin),
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:  "Success information:",
								Value: fmt.Sprintf("**%s**", messageSlice[1]),
							},
						},
					},
				},
			}
			templateCopy.Data = messageSuccessResponse
		}
	case 3:
		{
			templateCopy = *slashCommandGeneralResponses["buttonMessage"].Response
			messageButtonRespond := &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Embeds: []*discordgo.MessageEmbed{
					{
						Color:       blueColor,
						Title:       messageSlice[0],
						Description: messageSlice[1],
					},
				},
			}
			templateCopy.Data = messageButtonRespond
		}
	}
	return templateCopy
}

func GetAllWarcraftLogsRaidData(inMem bool, newestOne bool, logCode string, botInfo ...any) []logAllData {
	//time.Sleep(30 * time.Second)
	quriesToRun := []map[string]any{}
	WriteInformationLog("Retrieving warcraftlogs data for query with name: 'guildLogsRaidIDs' during function GetAllWarcraftLogsRaidData()", "Getting Warcraft logs data")
	allLogsBase := []logsBase{}
	//time.Sleep(5 * time.Second)
	innerSession := &discordgo.Session{}
	event := &discordgo.Interaction{}
	doStatusCount := 0
	doStatus := false
	if len(botInfo) > 1 {
		if session, ok := botInfo[0].(*discordgo.Session); ok {
			innerSession = session
			doStatusCount++
		}

		if interaction, ok := botInfo[1].(*discordgo.Interaction); ok {
			event = interaction
			doStatusCount++
		}
		if doStatusCount == 2 {
			doStatus = true
			fmt.Println("WE WILL DO STATUS")
		}
	}
	x := 1
	for {
		pageQuerie := mapOfWarcaftLogsQueries["guildLogsRaidIDs"]
		if x != 1 || newestOne {
			pageQuerie = SetWarcraftLogQueryVariables(mapOfWarcaftLogsQueries["guildLogsRaidIDs"], x)[0]
		}
		if logs, ok := GetWarcraftLogsData(pageQuerie)["logs"].([]logsBase); ok {
			if len(logs) == 0 {
				break
			}
			allLogsBase = append(allLogsBase, logs...)
		}
		x++
	}
	mapOfQueries := SetWarcraftLogQueryVariables(mapOfWarcaftLogsQueries["logsByOwnerAndCode"], allLogsBase)
	if logCode != "" {
		for _, mapOfQuery := range mapOfQueries {
			for key, value := range mapOfQuery {
				if key == "variables" {
					for key, value := range value.(map[string]any) {
						if key == "code" {
							if value == logCode {
								quriesToRun = append(quriesToRun, mapOfQuery)
								break
							}
						}
					}
				}
			}
		}
	} else {
		quriesToRun = mapOfQueries
	}
	logsOfAllRaids := []logAllData{}
	for x, query := range quriesToRun {
		if doStatus {
			interactionResponse := NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("Progess on job|**Completed %.1f%% so far**", float64(x)/float64(len(quriesToRun))*100))
			_, err := innerSession.InteractionResponseEdit(event, &discordgo.WebhookEdit{
				Embeds: &interactionResponse.Data.Embeds,
			})
			if err != nil {
				WriteErrorLog("An error occured while trying to sent status message to user during the function GetAllWarcraftLogsRaidData()", err.Error())
			}
		}

		index := x
		time.Sleep(1 * time.Second)
		newQuery := SetWarcraftLogQueryVariables(mapOfWarcaftLogsQueries["allFightIDsForRaid"], []logsBase{allLogsBase[index]})
		WriteInformationLog("Retrieving warcraftlogs data for query with name: 'allFightIDsForRaid' during function GetAllWarcraftLogsRaidData()", "Getting Warcraft logs data")
		if len(newQuery) == 0 {
			continue
		}
		fightIDs := GetWarcraftLogsData(newQuery[0])
		newQuery = SetWarcraftLogQueryVariables(quriesToRun[index], fightIDs)
		WriteInformationLog("Retrieving warcraftlogs data for query with name: 'logsByOwnerAndCode' during function GetAllWarcraftLogsRaidData()", "Getting Warcraft logs data")

		log := GetWarcraftLogsData(newQuery[0])
		if VerifyWarcraftLogData(log) {
			allDataLogs, _ := UnwrapFullWarcraftLogRaid(log, query)
			logsOfAllRaids = append(logsOfAllRaids, allDataLogs)
			//logsOfAllRaids = RetriveRaiderSpecificData(logsOfAllRaids, actorIDs) //Adding data to the existing data
		}
		if newestOne {
			WriteInformationLog("The flag of 'newestLog' Was used, therefor only 1 log is returned... During function GetAllWarcraftLogsData()", "Warcraft logs return data")
			break
		}
	}

	return WriteRaidCache(UpdateClassSpec(logsOfAllRaids))
}

/*
	func RetriveRaiderSpecificData(allLogs []logAllData, actors []int, fights map[string]any, query map[string]any) []logAllData {
		SetWarcraftLogQueryVariables
		return nil
	}
*/
func WriteRaidCache(logDataSlice []logAllData) []logAllData {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", " ")
	existingLogData := []logAllData{}
	returnLogAllData := []logAllData{}
	if len(logDataSlice) == 1 {
		if logDataBytes := CheckForExistingCache(raidAllDataPath); logDataBytes != nil {
			json.Unmarshal(logDataBytes, &existingLogData)
		} else {
			WriteInformationLog("The slice of logAllData has a length of %d, inside of function WriteRaidCache()", "Writing raid cache")
		}
	}
	existingLogData = append(logDataSlice, existingLogData...)
	sort.Slice(existingLogData, func(i, j int) bool {
		timeI, _ := time.Parse(timeLayout, existingLogData[i].RaidStartTimeString)
		timeJ, _ := time.Parse(timeLayout, existingLogData[j].RaidStartTimeString)
		return timeI.After(timeJ)
	})
	err := encoder.Encode(existingLogData)
	if err != nil {
		WriteErrorLog("An error occured while trying to json-encode all the raids", err.Error())
	}
	err = os.WriteFile(raidAllDataPath, buf.Bytes(), 0644)
	json.Unmarshal(buf.Bytes(), &returnLogAllData)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to write the json data to path %s during function GetAllWarcraftLogsRaidData()", raidAllDataPath), err.Error())
	}
	WriteInformationLog(fmt.Sprintf("The following %d of type []logAllData has been found on path %s", len(returnLogAllData), raidAllDataPath), "Reading cache")
	return returnLogAllData
}

/*
	func WriteWarcraftLogsQueryErrorLogs(failedQueryResult map[string]any, failedQueryRequest map[string]any) {
		errorLogWarcraftLogsMutex.Lock()
		defer errorLogWarcraftLogsMutex.Unlock()
		cachedErrors := []logDataLoss{}

		dataLoss := logDataLoss{}
		mapOfSumAndCount := make(map[int]int)
		if mapOfError, ok := errorSlice.(map[string]any); ok {
			if variablesExist, ok := mapOfError["variables"].(map[string]any); ok {

			}
		}

		if sliceOfFailedQueries, ok := failedQueryRequest["errors"].([]any); ok {
			for x, errorSlice := range sliceOfFailedQueries {

			}
		}

		errStruct := logDataLoss{}
		if currentCache := CheckForExistingCache(errorLogPath); len(currentCache) > 0 {
			err := json.Unmarshal(currentCache, &cachedErrors)
			if err != nil {
				log.Fatal("An error occured while trying to unmarshal json from information log cache: inside function WriteErrorLog()", err)
			}
		}
		cachedErrors = append(cachedErrors, errStruct)

		errJson, err := json.MarshalIndent(cachedErrors, "", " ")
		if err != nil {
			log.Fatal("An error occured while trying to marshal information log to json: inside function WriteErrorLog()", err)
		}

		cacheFile, err := os.OpenFile(errorLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(fmt.Sprintf("Error opening file: Inside function WriteErrorLog() %s", errorLogPath), err.Error())
		}
		defer cacheFile.Close()
		cacheFile.Write(errJson)
	}
*/

func RetrieveSpecificEncounterLog(encounterID map[string]int64) []map[string]any {
	mapOfEncounters := []map[string]any{}
	mapOfEncounterQueries := SetWarcraftLogQueryVariables(mapOfWarcaftLogsQueries["logsByEncounterID"], encounterID)
	for x, mapOfEncounterQuery := range mapOfEncounterQueries {
		data := GetWarcraftLogsData(mapOfEncounterQuery)
		if data != nil && data["errors"] == nil {
			mapOfEncounters = append(mapOfEncounters, data)
		} else {
			fmt.Printf("The following map %d had errors \n%s", x, data)
		}
	}
	return mapOfEncounters
}

func UnwrapFullWarcraftLogRaid(mapToUnwrap map[string]any, warcraftLogsQuery map[string]any) (logAllData, []int) {
	playerLogs := []logPlayer{}
	mapSemiUnwrapped := map[string]any{} //mapToUnwrap["logs"].(map[string]any)["data"].(map[string]any)["reportData"].(map[string]any)["report"]
	returnBeforeCleaning := logAllData{}
	if mapOfLogs, ok := mapToUnwrap["logs"].(map[string]any); ok {
		if _, ok := mapOfLogs["data"].(map[string]any); ok {
			mapSemiUnwrapped = mapToUnwrap["logs"].(map[string]any)["data"].(map[string]any)["reportData"].(map[string]any)["report"].(map[string]any)
		} else {
			fmt.Println("AN ERROR OCCURED", mapToUnwrap)
			return logAllData{}, nil
		}
	} else {
		WriteErrorLog("This log is invalid as it lacks the fundamental structure of map[logs], during the function UnwrapFullWarcraftLogRaid()", "Wrong format")
	}
	mapOfPlayers := map[string]any{}
	if mapOfPlayersNested, ok := mapSemiUnwrapped["players"]; ok {
		if _, ok := mapOfPlayersNested.(map[string]any); ok {
			mapOfPlayers = mapOfPlayersNested.(map[string]any)["data"].(map[string]any)
		} else {
			WriteErrorLog(fmt.Sprintf("The warcraft log with code: %s is lagging map of data inside of the players - This is crucial for the function, therefore this returns...", mapSemiUnwrapped["code"]), "During function UnwrapFullWarcraftLog()")
			return logAllData{}, nil
		}
	}
	combatEvents := mapSemiUnwrapped["combatantInfo"].(map[string]any)["data"].([]any)
	if len(combatEvents) == 0 {
		WriteErrorLog("The warcraftlog provided does not contain any combat events which are crucial for analysis, returning...", "During function UnwrapFullWarcraftLogRaid()")
		return logAllData{}, nil
	}
	var totalRaidTime float64
	var totalFightTime float64
	encounterIDs := []int64{}
	if sliceOfFights, ok := mapSemiUnwrapped["fights"].([]any); ok {
		for _, sliceOfFight := range sliceOfFights {
			if mapOfFight, ok := sliceOfFight.(map[string]any); ok {
				if encounterID, ok := mapOfFight["encounterID"].(float64); ok {
					if encounterID != 0 { //Dont need to see trash
						encounterIDs = append(encounterIDs, int64(encounterID))
					}
				}
				totalFightTime += mapOfFight["endTime"].(float64) - mapOfFight["startTime"].(float64)
			}
		}
		totalRaidTime = sliceOfFights[len(sliceOfFights)-1].(map[string]any)["endTime"].(float64)
	}
	if raidComposition, ok := mapOfPlayers["composition"]; ok {
		for _, slice := range raidComposition.([]any) {
			if playerInfo, ok := slice.(map[string]any); ok {
				playerSpecs := []logPlayerSpec{}
				if specs, ok := playerInfo["specs"].([]any); ok {
					if len(specs) == 1 {
						expandInnerMap := specs[0].(map[string]any)
						playerSpec := logPlayerSpec{
							Name:     expandInnerMap["spec"].(string),
							TypeRole: expandInnerMap["role"].(string),
							MainSpec: true,
						}
						playerSpecs = append(playerSpecs, playerSpec)
					} else {
						for _, spec := range specs {
							expandInnerMap := spec.(map[string]any)
							isTank := false
							if expandInnerMap["role"] == "tank" { //We cant see name of person, of damage taken table, therefor we need to calculate the tank roles first here
								isTank = true
							}
							playerSpec := logPlayerSpec{
								Name:     expandInnerMap["spec"].(string),
								TypeRole: expandInnerMap["role"].(string),
								MainSpec: isTank,
							}
							playerSpecs = append(playerSpecs, playerSpec)
						}
					}
				}

				if !strings.Contains(playerInfo["name"].(string), " ") {
					playerLog := logPlayer{
						Name: playerInfo["name"].(string),
						//DiscordID: ResolvePlayerID(),
						WarcraftLogsGUID: int64(playerInfo["guid"].(float64)),
						Specs:            playerSpecs,
						ClassName:        playerInfo["type"].(string),
					}
					playerLogs = append(playerLogs, playerLog)
				} //Make sure no chickens and other trinket stuff gets a playerObject
			}

		}
	}
	for x, player := range playerLogs {
		if len(player.Specs) == 1 {
			continue
		}

		fixTank := false
		newPlayerLogsSlice := []logPlayerSpec{}
		for _, spec := range player.Specs {
			if spec.TypeRole == "Tank" {
				fixTank = true
				newPlayerLogsSlice = append(newPlayerLogsSlice, spec)
			}
		}
		if fixTank {
			playerLogs[x].Specs = newPlayerLogsSlice
		}
	}
	deathCounter := 0
	actorIDs := []int{}
	for x, playerLog := range playerLogs {
		if logInternalRaiderIDsSlice, ok := mapSemiUnwrapped["masterData"].(map[string]any)["actors"].([]any); ok {
			for _, internalSlice := range logInternalRaiderIDsSlice {
				if mapOfInternal, ok := internalSlice.(map[string]any); ok {
					if mapOfInternal["name"].(string) == playerLog.Name {
						playerLogs[x].InternalLogID = int(mapOfInternal["id"].(float64))
					}
				}
			}
		}

		actorIDs = append(actorIDs, playerLogs[x].InternalLogID)
		if damageDoneSlice, ok := mapOfPlayers["damageDone"].([]any); ok {
			for _, damageDone := range damageDoneSlice {
				if mapOfDamageDone, ok := damageDone.(map[string]any); ok {
					if mapOfDamageDone["name"] == playerLog.Name {
						playerLogs[x].DamageDone = int64(mapOfDamageDone["total"].(float64))
						break
					}
				}
			}
		}

		if healingDoneSlice, ok := mapOfPlayers["healingDone"].([]any); ok {
			for _, healingDone := range healingDoneSlice {
				if mapOfHealingDone, ok := healingDone.(map[string]any); ok {
					if mapOfHealingDone["name"] == playerLog.Name {
						if healingDone, ok := mapOfHealingDone["total"].(float64); ok {
							playerLogs[x].HealingDone = int64(healingDone)
							break
						}
					}
				}
			}
		}

		if deathsSummarySlice, ok := mapSemiUnwrapped["deathSummary"].(map[string]any)["data"].(map[string]any)["entries"].([]any); ok {
			for _, deathSlice := range deathsSummarySlice {
				if deathMap, ok := deathSlice.(map[string]any); ok {
					mapOfDead := make(map[string]bool)
					if deathMap["name"].(string) == playerLog.Name {
						if deathTimers, ok := deathMap["events"].([]any); ok {
							var totalTimeToDie time.Duration
							deathCounter++
							for x := len(deathTimers) - 1; x >= 0; x-- {
								if x >= 1 && time.Duration(int64(deathTimers[x].(map[string]any)["timestamp"].(float64))) > 0 && time.Duration(int64(deathTimers[x-1].(map[string]any)["timestamp"].(float64))) > 0 {
									timeCurrentIndex := time.Duration(int64(deathTimers[x].(map[string]any)["timestamp"].(float64))) * time.Millisecond
									timeNextIndex := time.Duration(int64(deathTimers[x-1].(map[string]any)["timestamp"].(float64))) * time.Millisecond
									duration := timeNextIndex - timeCurrentIndex
									totalTimeToDie += duration
								}
							}
							totalTimeToDieRounded := math.Round(float64(totalTimeToDie.Seconds())*100) / 100
							if sourceOfDeath, ok := deathMap["damage"].(map[string]any); ok {
								playerDeath := logPlayerDeath{}
								if math.Round(sourceOfDeath["total"].(float64)) > 0 && math.Round(deathMap["timestamp"].(float64)) > 0 && totalRaidTime > 0 && totalTimeToDie > 0 {
									playerDeath.TimeToDie = totalTimeToDieRounded
									playerDeath.DamageTakenSecond = math.Round(sourceOfDeath["total"].(float64) / float64(totalTimeToDie.Seconds()))
									playerDeath.PercentageRaidComplete = math.Round(deathMap["timestamp"].(float64) / totalRaidTime * 100)
								}

								if playerDeath.PercentageRaidComplete == 100 {
									playerDeath.LastBoss = true
								}

								if !mapOfDead[deathMap["name"].(string)] {
									playerDeath.FirstDeath = true
									mapOfDead[deathMap["name"].(string)] = true
								}

								if playerDeath.TimeToDie < 2 { //If you die within 2 seconds, u insta died
									playerDeath.InstaKilled = true
								}

								if abilityDamageTakenMap, ok := sourceOfDeath["abilities"].([]any); ok {
									for _, abilitySlice := range abilityDamageTakenMap {
										var maxDmgSource float64
										for attributeName, attributeValue := range abilitySlice.(map[string]any) {
											if attributeName == "total" {
												if maxDmgSource < attributeValue.(float64) {
													playerDeath.KilledBy = abilitySlice.(map[string]any)["name"].(string)
													break
												}
											}
										}
									}

									if playerDeath.KilledBy == "Melee" {
										playerDeath.MeleeHit = true
									}
								}
								playerLogs[x].Deaths = append(playerLogs[x].Deaths, playerDeath)
								break
							}
						}
					}
				}
			}
		}

		if sliceCastSummary, ok := mapSemiUnwrapped["castsSummary"].(map[string]any)["data"].(map[string]any)["entries"].([]any); ok {
			for _, sliceSummary := range sliceCastSummary {
				if mapOfSummary, ok := sliceSummary.(map[string]any); ok {
					if int64(mapOfSummary["guid"].(float64)) == playerLog.WarcraftLogsGUID {
						playerLogs[x].ActiveTimeMS = int64(mapOfSummary["activeTime"].(float64))
						for _, sliceOfAbility := range mapOfSummary["abilities"].([]any) {
							if mapOfAbility, ok := sliceOfAbility.(map[string]any); ok {
								playerAbility := logPlayerAbility{
									Name:       mapOfAbility["name"].(string),
									Type:       int(mapOfAbility["type"].(float64)),
									TotalCasts: int(mapOfAbility["total"].(float64)),
								}
								playerLogs[x].Abilities = append(playerLogs[x].Abilities, playerAbility)
							}
						}
					}
				}
			}
		}

		if totalSliceOfDetails, ok := mapOfPlayers["playerDetails"].(map[string]any)["dps"]; ok {
			if sliceOfDetails, ok := totalSliceOfDetails.([]any); ok {
				for _, slice := range sliceOfDetails {
					if mapOfDetails, ok := slice.(map[string]any); ok {
						if mapOfDetails["name"].(string) == playerLog.Name {
							if itemLevel, ok := mapOfDetails["maxItemLevel"].(float64); ok {
								playerLogs[x].ItemLevel = int(itemLevel)
							}
						}
					}
				}
			}
		}

		//Calculate mainspec IF its not tanks
		for y, spec := range playerLog.Specs {
			if len(playerLog.Specs) > 1 && spec.TypeRole != "tank" {
				if playerLogs[x].DamageDone > playerLogs[x].HealingDone {
					if spec.TypeRole == "dps" {
						playerLogs[x].Specs[y].MainSpec = true
					}
				} else if playerLogs[x].DamageDone < playerLogs[x].HealingDone {
					if spec.TypeRole == "healer" {
						playerLogs[x].Specs[y].MainSpec = true
					}
				}
			}
		}

		//Calculate CPM
		castSumPlayer := 0
		for _, playerAbility := range playerLogs[x].Abilities {
			castSumPlayer += playerAbility.TotalCasts
		}

		playerLogs[x].MinuteAPM = math.Round(float64(castSumPlayer)/(time.Duration(totalFightTime*float64(time.Millisecond))).Minutes()*100) / 100
		//Calculate worldbuffs
		mapOfRequiredWorldBuffs := make(map[int]bool)
		for _, combatSlice := range combatEvents {
			if mapOfCombat, ok := combatSlice.(map[string]any); ok {
				if mapOfCombat["sourceID"] == float64(playerLogs[x].InternalLogID) {
					if sliceOfAuras, ok := mapOfCombat["auras"].([]any); ok {
						for _, attribute := range sliceOfAuras {
							if mapOfAura, ok := attribute.(map[string]any); ok {
								abillityID := int(mapOfAura["ability"].(float64))
								if knownWorldBuffs[abillityID] != (logWorldBuff{}) && !mapOfRequiredWorldBuffs[abillityID] {
									playerLogs[x].WorldBuffs = append(playerLogs[x].WorldBuffs, knownWorldBuffs[abillityID])
									mapOfRequiredWorldBuffs[abillityID] = true
								}
							}
						}
					}
				}
			}
		}
	}
	mapOfZoneNames := make(map[string]bool)
	for _, encounter := range encounterIDs {
		mapOfEncounter := map[string]any{
			"encounterID": int64(encounter),
		}
		newQuery := SetWarcraftLogQueryVariables(mapOfWarcaftLogsQueries["logsByEncounterID"], mapOfEncounter)
		if len(newQuery) > 0 {
			encounterData := GetWarcraftLogsData(newQuery[0])["encounter"].(map[string]any)
			if VerifyWarcraftLogData(encounterData) {
				if mapOfData, ok := encounterData["data"].(map[string]any); ok {
					if mapOfWorldData, ok := mapOfData["worldData"].(map[string]any); ok {
						if mapOfEncounter, ok := mapOfWorldData["encounter"].(map[string]any); ok {
							if mapOfZone, ok := mapOfEncounter["zone"].(map[string]any); ok {
								mapOfZoneNames[mapOfZone["name"].(string)] = true
							}
						}
					}
				}
			}
		}
	}
	raidNames := []string{}
	raidTitleNamesSlice := []string{}
	if raidTitle, ok := mapSemiUnwrapped["title"].(string); ok {
		raidTitleNamesSlice = strings.Split(strings.Split(strings.ToLower(strings.TrimSpace(raidTitle)), "-")[0], "+")
	} else {
		fmt.Println("DO WE EVER REACH HERE?", mapSemiUnwrapped)
	}
	for _, raidShortName := range raidTitleNamesSlice {
		raidNames = append(raidNames, RaidNameLongHandConversion(raidShortName))
	}

	raidTimeHrs := int(time.Duration(totalRaidTime * float64(time.Millisecond)).Hours())
	raidTimeMinutes := int((time.Duration(totalRaidTime * float64(time.Millisecond)).Minutes())) % 60
	raidTimeSeconds := int((time.Duration(totalRaidTime * float64(time.Millisecond)).Seconds())) % 60
	unixTime := int64(mapSemiUnwrapped["startTime"].(float64))
	returnBeforeCleaning = logAllData{
		RaidAverageItemLevel: mapOfPlayers["itemLevel"].(float64),
		Players:              playerLogs,
		PlayersCount:         len(playerLogs),
		RaidTime:             time.Duration(totalRaidTime),
		RaidTimeString:       fmt.Sprintf("%02d:%02d:%02d", raidTimeHrs, raidTimeMinutes, raidTimeSeconds),
		RaidStartUnixTime:    unixTime,
		RaidStartTimeString:  time.UnixMilli(unixTime).Format(timeLayout),
		TotalDeaths:          deathCounter,
		MetaData: logsBase{
			LoggerName: mapSemiUnwrapped["owner"].(map[string]any)["name"].(string),
			Code:       mapSemiUnwrapped["code"].(string),
		},
		RaidTitle: mapSemiUnwrapped["title"].(string),
		RaidNames: raidNames,
	}

	return returnBeforeCleaning, actorIDs
}

func CapitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func UpdateClassSpec(logs []logAllData) []logAllData {
	mapOfPlayers := make(map[string]bool)
	for x, log := range logs {
		for y, player := range log.Players {
			for z, spec := range player.Specs {
				if spec.TypeRole == "tank" || spec.TypeRole == "healer" {
					logs[x].Players[y].Specs[z].TypeRole = CapitalizeFirst(spec.TypeRole)
					mapOfPlayers[player.Name] = true
					break
				}
			}
		}
		for x, log := range logs {
			for y, player := range log.Players {
				for z, spec := range player.Specs {
					if (spec.Name == "Elemental" || spec.Name == "Shadow" || spec.Name == "Balance" || player.ClassName == "Mage" || player.ClassName == "Warlock" || player.ClassName == "Hunter") && spec.TypeRole != "Tank" && spec.TypeRole != "Healer" && spec.MainSpec {
						logs[x].Players[y].Specs[z].TypeRole = "Ranged"
						break
					} else if (spec.Name == "Enhancement" || spec.Name == "Feral" || player.ClassName == "Warrior" || player.ClassName == "Rogue" || spec.Name == "Retribution") && spec.TypeRole != "Tank" && spec.TypeRole != "Healer" && spec.MainSpec {
						logs[x].Players[y].Specs[z].TypeRole = "Melee"
						break
					}
				}
			}
		}
	}
	return logs
}

func VerifyWarcraftLogData(mapToVerify map[string]any) bool {
	if logs, ok := mapToVerify["logs"].(map[string]any); ok {
		if data, ok := logs["data"].(map[string]any); ok {
			if reportData, ok := data["reportData"].(map[string]any); ok {
				if report, ok := reportData["report"].(map[string]any); ok {
					if raidTitle, ok := report["title"].(string); ok {
						if len(strings.Split(raidTitle, " ")) >= 3 {
							return true
						} else {
							WriteErrorLog(fmt.Sprintf("The warcraftlog: %s does not have a correct title, the title must be in format <raid 1> <notes raid 1> <raid 2> <notes raid 2> - <dd-mm-yyyy>, log invalid... During the function VerifyWarcraftLogData()", raidTitle), "Warcraftlog title invalid")
						}
					} else {
						WriteErrorLog("The warcraftlog being verified does NOT have a raid-title which is crucial for the bot, log skipped..., during the function VerifyWarcraftLogData()", "Warcraftlog title invalid")
					}
				}
			}
		}
	}
	/*
		if mapToVerify == nil {
			return false
		} else {
			WriteInformationLog("The map provided is nil. During function VerifyWarcraftLogData()", "Verify Warcraft logs data")
		}

		if mapOfLogs, ok := mapToVerify["logs"].(map[string]any); ok {
			if sliceOfErrors, ok := mapOfLogs["errors"].([]any); ok {
				WriteInformationLog(fmt.Sprintf("The map provided contains the following GraphQL errors: %s\nDuring function VerifyWarcraftLogData()", sliceOfErrors...), "Verify Warcraft logs data")
			}
			return false
		} else {
			WriteInformationLog("The warcraft logs data is valid...", "During function VerifyWarcraftLogData()")
			return true
		}
	*/
	return false
}

func VerifyAllLogData(currentLog logAllData) bool {
	if len(currentLog.Players) == 0 {
		WriteInformationLog(fmt.Sprintf("The log with name: %s and code: %s does not have any players and is therefor not valid", currentLog.MetaData.Code, currentLog.RaidTitle), "Verifying Warcraftlogs data")
		return false
	}

	return true
}

func RetrieveCustomSchedules() []schedule {
	customSchedules := []schedule{}
	if customSchedulesBytes := CheckForExistingCache(customSchedulePath); len(customSchedulesBytes) > 0 {
		err := json.Unmarshal(customSchedulesBytes, &customSchedules)
		if err != nil {
			WriteErrorLog("An error occured while trying to unmarshal json content for the custom schedules: Inside function RetrieceCustomSchedules()", err.Error())
		}
	}

	for x, customSchedule := range customSchedules {
		switch customSchedule.Weekday {
		case 0:
			{
				customSchedules[x].Weekday = time.Sunday
			}
		case 1:
			{
				customSchedules[x].Weekday = time.Monday
			}
		case 2:
			{
				customSchedules[x].Weekday = time.Tuesday
			}
		case 3:
			{
				customSchedules[x].Weekday = time.Wednesday
			}
		case 4:
			{
				customSchedules[x].Weekday = time.Thursday
			}
		case 5:
			{
				customSchedules[x].Weekday = time.Friday
			}
		case 6:
			{
				customSchedules[x].Weekday = time.Saturday
			}
		}
	}

	return customSchedules
}

func GetIntPointer(n int) *int {
	return &n
}

func GetStringPointer(s string) *string {
	return &s
}

func NewSlashCommand(session *discordgo.Session) {
	sliceOfSlashCommandMaps := []map[string]applicationCommand{}
	//sliceOfSlashCommandMaps = append(sliceOfSlashCommandMaps, slashCommandTemplates)
	sliceOfSlashCommandMaps = append(sliceOfSlashCommandMaps, slashCommandAdminCenter)
	sliceOfSlashCommandMaps = append(sliceOfSlashCommandMaps, slashCommandAllUsers)
	for _, slice := range sliceOfSlashCommandMaps {
		for name, template := range slice {
			_, err := session.ApplicationCommandCreate(session.State.User.ID, serverID, template.Template)
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to create Slash template: %s inside function NewSlashCommand()", name), err.Error())
			}
		}
	}
}

func DeleteOldSlashCommand(session *discordgo.Session) {
	mapOfApplicationCommandsToKeep := make(map[string]bool)
	for nameOfUserCommand := range slashCommandAllUsers {
		mapOfApplicationCommandsToKeep[nameOfUserCommand] = false //Initalize keys basically
	}

	for nameOfAdminCommand := range slashCommandAdminCenter {
		mapOfApplicationCommandsToKeep[nameOfAdminCommand] = false //Initalize keys basically
	}
	botID := session.State.User.ID
	allBotApplicationCommands, err := session.ApplicationCommands(botID, serverID)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve all application commands for bot %s, during the function DeleteOldSlashCommand()", botID), err.Error())
		return
	}
	for _, discordCommand := range allBotApplicationCommands {
		fmt.Println("NAME OF APP", discordCommand.Name)
		if _, ok := mapOfApplicationCommandsToKeep[discordCommand.Name]; !ok {
			//Found channel to delete
			err = session.ApplicationCommandDelete(botID, serverID, discordCommand.ID)
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to delete application command %s for bot %s, during the function DeleteOldSlashCommand()", discordCommand.Name, botID), err.Error())
				continue
			}
			WriteInformationLog(fmt.Sprintf("Application slashcommand with name %s has been deleted successfully from the server %s, during the function DeleteOldSlashCommand()", discordCommand.Name, serverID), "Successfully deleted slash command")
		}
	}
}

func RoundTwoDecimalsFloat(num float64) float64 {
	return math.Round(num*100) / 100
}

func CalculateAverageSum(sumSlice []int) float64 {
	var totalSum int
	for _, partOfSum := range sumSlice {
		totalSum += partOfSum
	}
	return RoundTwoDecimalsFloat(float64(totalSum) / float64(len(sumSlice)))
}

func CalculatePercentDifference(start, end float64) float64 {
	if start == 0 {
		if end == 0 {
			return 0
		}
		// If we go from 0 to something, we treat it as 100% increase
		return 100
	}

	diff := ((end - start) / start) * 100
	fmt.Println("START VALUE:", start, "END VALUE:", end, "DIFF", diff)
	return RoundTwoDecimalsFloat(diff)
}

func CalculateAveragePercentChange(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	var totalChange float64
	var count int

	for i := 1; i < len(values); i++ {
		start := values[i-1]
		end := values[i]

		if start == 0 {
			// Treat jump from 0 to something as 100% increase, skip if both are 0
			if end == 0 {
				continue
			}
			totalChange += 100
		} else {
			change := ((end - start) / start) * 100
			totalChange += change
		}
		count++
	}

	if count == 0 {
		return 0
	}

	return RoundTwoDecimalsFloat(totalChange / float64(count))
}

func FormatFloatSmart(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%.2f", f)
}

func IntSliceToFloat64(ints []int) []float64 {
	floats := make([]float64, len(ints))
	for i, v := range ints {
		floats[i] = float64(v)
	}
	return floats
}

func SummarizedMergedRaidLogsMeta(dataLoggedSlice []logAllData) ([]string, string) {
	baseToFactor := 10
	returnWarcraftLogsLinksSlice := []string{}
	returnRaidData := ""
	percentDifferenceItemLevel := ""
	percentDifferenceDeaths := ""
	percentDifferencePlayerCount := ""
	percentDifferenceRaidTime := ""
	totalNumberOfLogs := len(dataLoggedSlice)
	sliceOfWarcraftLogLinks := []string{}
	sliceOfAverageDeath := []int{}
	sliceOfAverageItemLevel := []int{}
	sliceOfAveragePlayerCount := []int{}
	sliceOfAverageRaidTime := []int{}
	var splitDataFactor int
	if len(dataLoggedSlice) >= baseToFactor {
		splitFloat := float64(len(dataLoggedSlice)) / float64(baseToFactor)
		splitDataFactor = int(math.Ceil(splitFloat))
	} else {
		splitDataFactor = 1
	}
	maxCounter := 0
	minCounter := 0
	mapOfSeenLogs := make(map[string]bool)
	logCounter := 0
	//mapOfLowestDeaths := make(map[int]string)                //Should only contain 1 key
	//mapOfLowestRaidTimer := make(map[int64]string)           //Should only contain 1 key
	//mapOfHighestPlayerCount := make(map[int]string)          //Should only contain 1 key
	//mapOfHighestAverageItemlevel := make(map[float64]string) //Should only contain 1 key
	lowestDeaths := 0
	var lowestRaidTimer int64
	highestPlayerCount := 0
	var highestAverageItemLevel float64
	for x := 1; x <= splitDataFactor; x++ {
		maxCounter = x * baseToFactor
		minCounter = maxCounter - baseToFactor
		for z, log := range dataLoggedSlice {
			logCounter++
			fmt.Println("LENGTH OF LOGS:", totalNumberOfLogs)
			if (z >= minCounter && z <= maxCounter-1 || splitDataFactor == 1) && !mapOfSeenLogs[log.MetaData.Code] {
				sliceOfWarcraftLogLinks = append(sliceOfWarcraftLogLinks, fmt.Sprintf("URL => https://fresh.warcraftlogs.com/reports/%s", log.MetaData.Code))
				mapOfSeenLogs[log.MetaData.Code] = true
				logCounter = 0
			}
			if x == 1 {
				sliceOfAverageDeath = append(sliceOfAverageDeath, log.TotalDeaths)
				sliceOfAverageItemLevel = append(sliceOfAverageItemLevel, int(log.RaidAverageItemLevel))
				sliceOfAverageRaidTime = append(sliceOfAverageRaidTime, int(log.RaidTime))
				sliceOfAveragePlayerCount = append(sliceOfAveragePlayerCount, log.PlayersCount)
				if z == 0 {
					lowestDeaths = log.TotalDeaths
					lowestRaidTimer = int64(log.RaidTime)
					highestPlayerCount = log.PlayersCount
					highestAverageItemLevel = log.RaidAverageItemLevel
				} else {
					if log.TotalDeaths < lowestDeaths {
						lowestDeaths = log.TotalDeaths
					}

					if int64(log.RaidTime) < lowestRaidTimer {
						lowestRaidTimer = int64(log.RaidTime)
					}

					if log.RaidAverageItemLevel > highestAverageItemLevel {
						highestAverageItemLevel = log.RaidAverageItemLevel
					}

					if log.PlayersCount > highestPlayerCount {
						highestPlayerCount = log.PlayersCount
					}
				}
			}
			if z+1 == maxCounter || z == len(dataLoggedSlice)-1 {
				returnWarcraftLogsLinks := fmt.Sprintf(`
				**[Warcraft log links from newest to oldest]**
				%s`,
					strings.Join(sliceOfWarcraftLogLinks, "\n"),
				)
				sliceOfWarcraftLogLinks = nil
				returnWarcraftLogsLinksSlice = append(returnWarcraftLogsLinksSlice, returnWarcraftLogsLinks)
				minCounter = maxCounter
				maxCounter += 10
			}
		}

		if x == splitDataFactor {
			percentDifferenceItemLevel = FormatFloatSmart(CalculateAveragePercentChange(IntSliceToFloat64(sliceOfAverageItemLevel)))
			percentDifferenceDeaths = fmt.Sprintf("%d", int(math.Round(
				CalculateAveragePercentChange(IntSliceToFloat64(sliceOfAverageDeath)),
			)))
			percentDifferencePlayerCount = fmt.Sprintf("%d", int(math.Round(
				CalculateAveragePercentChange(IntSliceToFloat64(sliceOfAveragePlayerCount)),
			)))
			percentDifferenceRaidTime = FormatFloatSmart(CalculateAveragePercentChange(IntSliceToFloat64(sliceOfAverageRaidTime)))

			returnRaidData = fmt.Sprintf(`
            **Data extration period**
From date: **%s** 
To date: **%s**

Number of logs in scope of the raid type: %d
			
			**Disclaimer**
If logs are NOT split, data between raid types will effect the results below! %s

            **ALL TIME BEST**
Fastest clear: %s with code => %s

Least amount of deaths: %d with code => %s

Highest amount of players: %d with code => %s

Highest average raid item-level: %s with code => %s

            **Average difference in period (Percentage)**
Average raid-time in %%: **%s%%**

Average death-rate in %%: **%s%%**

Average Item-level of all raiders in %%: **%s%%**

Average player-count in %%: **%s%%**

			**Average difference in period (Absolute Values)**
Average raid-time: **%s**

Average death-rate: **%s**

Average Item-level of all raiders: **%s**

Average player-count: **%s**
`, dataLoggedSlice[len(dataLoggedSlice)-1].RaidStartTimeString, dataLoggedSlice[0].RaidStartTimeString, totalNumberOfLogs, antiCrackedBuiltin, percentDifferenceRaidTime, percentDifferenceDeaths, percentDifferenceItemLevel, percentDifferencePlayerCount, FormatDurationFromMilliseconds(CalculateAverageSum(sliceOfAverageRaidTime)), FormatFloatSmart(CalculateAverageSum(sliceOfAverageDeath)), FormatFloatSmart(CalculateAverageSum(sliceOfAverageItemLevel)), FormatFloatSmart(CalculateAverageSum(sliceOfAveragePlayerCount)))

		}
	}

	preparedURLOutputSlice := []string{}
	preparedURLOutputSlice = append(preparedURLOutputSlice, returnWarcraftLogsLinksSlice...)

	for _, slice := range returnWarcraftLogsLinksSlice {
		fmt.Println("LEN OF STRING LOG:", len(slice))
	}

	fmt.Println("LEN OF DATA PART:", len(returnRaidData))

	return preparedURLOutputSlice, returnRaidData

}

func FormatSecondsToStringFormat(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func FormatDurationFromMilliseconds(ms float64) string {
	seconds := ms / 1000
	total := int(math.Round(seconds))
	hours := total / 3600
	minutes := (total % 3600) / 60
	secs := total % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func SummarizeRaidLogMeta(dataLogged logAllData) string {
	return fmt.Sprintf(`
	Date of the raid: %s

	Warcraftlogs report code: **%s**

	Number of raiders present: **%d**

	Number of unique player deaths in the raid: **%d**

	Time to complete raids in hrs/min/s: **%s**

	Raids completed: **%s**

	Raid average item level: **%.2f**
	`, dataLogged.RaidStartTimeString, dataLogged.MetaData.Code, dataLogged.PlayersCount, dataLogged.TotalDeaths, dataLogged.RaidTimeString, strings.Join(dataLogged.RaidNames, " & "), dataLogged.RaidAverageItemLevel)
}

func CreateMessageEmbedsLargeLogData(allLogData []logAllData) []*discordgo.MessageEmbed {
	var embeds []*discordgo.MessageEmbed
	warcraftLogMessages, messageDataString := SummarizedMergedRaidLogsMeta(allLogData)

	embedDataMessage := &discordgo.MessageEmbed{
		Description: "Please see all calculations of the specific raid type below:",
		Color:       greenColor,
		Title:       fmt.Sprintf("For raid type: **%s**", allLogData[0].RaidNames[0]),
		Fields: []*discordgo.MessageEmbedField{
			{
				Value: messageDataString,
			},
		},
	}
	embeds = append(embeds, embedDataMessage)
	allWarcraftlogLinks := len(warcraftLogMessages)
	fmt.Println("LEN OF LINKS:", allWarcraftlogLinks)
	if allWarcraftlogLinks == 1 {
		amountOfLinks := len(strings.Split(warcraftLogMessages[0], "\n")[1 : len(strings.Split(warcraftLogMessages[0], "\n"))-1])
		embed := &discordgo.MessageEmbed{
			Description: fmt.Sprintf("%d logs of raid type: **%s**", amountOfLinks, allLogData[0].RaidNames[0]),
			Color:       greenColor,
			Title:       fmt.Sprintf("Warcraft log links for message above **%s**", allLogData[0].RaidNames[0]),
			Fields: []*discordgo.MessageEmbedField{
				{
					Value: warcraftLogMessages[0],
				},
			},
		}
		embeds = append(embeds, embed)
	} else if allWarcraftlogLinks > 1 {
		fmt.Println("DO WE REACXH ERE??!")
		for x, logLinks := range warcraftLogMessages {
			amountOfLinks := len(strings.Split(warcraftLogMessages[0], "\n")[1 : len(strings.Split(warcraftLogMessages[0], "\n"))-1])
			embed := &discordgo.MessageEmbed{
				Description: fmt.Sprintf("%d total logs and on page %d/%d of raid type: **%s**", amountOfLinks, x+1, allWarcraftlogLinks, allLogData[0].RaidNames[0]),
				Color:       greenColor,
				Title:       fmt.Sprintf("for raid type: **%s**", allLogData[0].RaidNames[0]),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Log links as a list",
						Value: logLinks,
					},
				},
			}
			embeds = append(embeds, embed)
		}
	}

	return embeds
}

func NewInteractionOutput(data any, currentResponse *discordgo.InteractionResponse) *discordgo.InteractionResponse {
	if data == nil {
		WriteErrorLog("No data found which is crucial for its operation, returning early...", "during function NewInteractionOutput()")
		return nil
	}
	returnInteractionResponse := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{},
		},
	}
	if dataLog, ok := data.(logAllData); ok {
		embed := &discordgo.MessageEmbed{
			URL:         fmt.Sprintf("https://fresh.warcraftlogs.com/reports/%s", dataLog.MetaData.Code),
			Description: "Please note that this will only show overall data about each raid\nYou can also get more specific information using other commands...",
			Color:       greenColor,
			Title:       fmt.Sprintf("General information about log title %s", dataLog.RaidTitle),
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  dataLog.RaidTitle,
					Value: SummarizeRaidLogMeta(dataLog),
				},
			},
		}
		returnInteractionResponse.Data.Embeds = append(returnInteractionResponse.Data.Embeds, embed)
		return returnInteractionResponse
	} else if allLogs, ok := data.([]logAllData); ok {
		fmt.Println("WE REACH HERE:")
		returnInteractionResponse.Data.Embeds = append(returnInteractionResponse.Data.Embeds, CreateMessageEmbedsLargeLogData(allLogs)...)
		fmt.Println("LENGTHJ EMBEDFSD ??", len(returnInteractionResponse.Data.Embeds))
		return returnInteractionResponse
	} else if stringValue, ok := data.(string); ok {
		embed := &discordgo.MessageEmbed{
			Description: fmt.Sprintf("You cannot respond to this message %s", crackedBuiltin),
			Color:       blueColor,
			Title:       crackedBuiltin,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  crackedBuiltin,
					Value: stringValue,
				},
			},
		}
		returnInteractionResponse.Data.Embeds = append(returnInteractionResponse.Data.Embeds, embed)
		return returnInteractionResponse
	} else {
		WriteErrorLog("An error occured while trying to create a new result interaction message", "During function NewInteractionOutput()")
		return nil
	}
}

func CapturePointInTimeRaidLogData(timeString string, onlyMainRaid bool) ([]logAllData, error) {
	var finalTimeParsed time.Duration
	timePassedDefault := 30 * 24 * time.Hour
	timeStringLower := strings.ToLower(strings.TrimSpace(timeString))

	if len(timeStringLower) < 2 {
		finalTimeParsed = timePassedDefault
	} else {
		timeDigitsOnly := timeStringLower[:len(timeStringLower)-1]
		timeInt, err := strconv.Atoi(timeDigitsOnly)
		if err != nil {
			WriteErrorLog("An error occurred while converting time string to int in CapturePointInTimeRaidLogData():", err.Error())
			return []logAllData{}, err
		}

		suffix := timeStringLower[len(timeStringLower)-1:]
		switch suffix {
		case "d":
			finalTimeParsed = time.Duration(timeInt) * 24 * time.Hour
		case "w":
			finalTimeParsed = time.Duration(timeInt) * 7 * 24 * time.Hour
		default:
			finalTimeParsed = timePassedDefault
		}
	}

	dataToParse := time.Now().Add(-finalTimeParsed)
	currentRaids, err := ReadRaidDataCache(dataToParse, onlyMainRaid)
	fmt.Println("THIS IS THE NUMBER OF OF LOGS:", dataToParse.Format(timeLayout), len(currentRaids))
	if err != nil {
		WriteErrorLog("Error reading raid data cache:", err.Error())
		return []logAllData{}, err
	}

	return currentRaids, nil
}

func CheckUserBoolResponseFlag(interactionData []*discordgo.ApplicationCommandInteractionDataOption, optionName string) (bool, *discordgo.ApplicationCommandInteractionDataOption) {
	returnBool := true
	returnOption := &discordgo.ApplicationCommandInteractionDataOption{}
	for _, option := range interactionData {
		for _, innerOption := range option.Options {
			if innerOption.Name == optionName && optionName != "" {
				returnBool = false
				returnOption = innerOption
				break
			}
			if !returnBool {
				break
			}
		}
	}
	return returnBool, returnOption
}

func RaidNameShortHandConversion(raidName string) string {
	switch raidName {
	case "Onyxia":
		{
			return "ony"
		}
	case "Molten Core":
		{
			return "mc"
		}
	case "Blackwing Lair":
		{
			return "bwl"
		}
	case "Zul'Gurub":
		{
			return "zg"
		}
	case "Temple of Ahn'Qiraj":
		{
			return "aq40"
		}
	case "Naxxramas":
		{
			return "naxx"
		}
	}
	return ""
}

func RaidNameLongHandConversion(shortName string) string {
	switch shortName {
	case "ony":
		{
			return "Onyxia"
		}
	case "mc":
		{
			return "Molten Core"
		}
	case "bwl":
		{
			return "Blackwing Lair"
		}
	case "zg":
		{
			return "Zul'Gurub"
		}
	case "aq40":
		{
			return "Temple of Ahn'Qiraj"
		}
	case "naxx":
		{
			return "Naxxramas"
		}
	}
	return ""
}

func NewWarcraftLogsGeneralDataResponse(useOnlyMainRaids bool, periodAsString string) ([]*discordgo.InteractionResponse, error) {
	currentRaids, err := CapturePointInTimeRaidLogData(periodAsString, useOnlyMainRaids)
	maxBytes := 6000
	returnInteractionResponses := []*discordgo.InteractionResponse{}
	//fmt.Println("THIS IS THE OPTION CHOSEN:", useOnlyMainRaids, "DATA", currentRaids, len(currentRaids))
	if err != nil {
		return nil, err
	}
	fmt.Println("NUMBER OF RAIDS FOUND:", len(currentRaids))
	mapOfRaids, _ := SortRaidsInSpecificMaps(currentRaids)
	template := &discordgo.InteractionResponse{}
	if len(currentRaids) > 9 {
		template = DeepCopyInteractionResponse(slashCommandAdminCenter["raidsummary"].Responses["resultraidsmerged"].Response)
	} else {
		template = DeepCopyInteractionResponse(slashCommandAdminCenter["raidsummary"].Responses["result"].Response)
	}

	for _, allLogData := range mapOfRaids {
		if len(currentRaids) <= 9 {
			for _, logData := range allLogData {
				singleRaidOutput := NewInteractionOutput(logData, template)
				template.Data.Embeds = append(template.Data.Embeds, singleRaidOutput.Data.Embeds...)
			}
		} else {
			fmt.Println("LEN OF ALL DATA BEFORE", len(allLogData))
			mergedRaidOutput := NewInteractionOutput(allLogData, template)
			fmt.Println("EMBEDS FOUND:;", len(mergedRaidOutput.Data.Embeds))
			template.Data.Embeds = append(template.Data.Embeds, mergedRaidOutput.Data.Embeds...)
		}
	}

	var sliceOfSliceTotalMessageEmbeds [][]*discordgo.MessageEmbed
	var sliceOfCurrentMessageEmbed []*discordgo.MessageEmbed
	var currentSize int

	for _, embed := range template.Data.Embeds {
		b, _ := json.Marshal(embed)
		embedSize := len(b)

		// If the current embed would push us over the limit, start a new chunk
		if currentSize+embedSize > maxBytes && len(sliceOfCurrentMessageEmbed) > 0 {
			sliceOfSliceTotalMessageEmbeds = append(sliceOfSliceTotalMessageEmbeds, sliceOfCurrentMessageEmbed)
			sliceOfCurrentMessageEmbed = []*discordgo.MessageEmbed{}
			currentSize = 0
		}

		sliceOfCurrentMessageEmbed = append(sliceOfCurrentMessageEmbed, embed)
		currentSize += embedSize
	}

	// Append the last chunk
	if len(sliceOfCurrentMessageEmbed) > 0 {
		sliceOfSliceTotalMessageEmbeds = append(sliceOfSliceTotalMessageEmbeds, sliceOfCurrentMessageEmbed)
	}

	totalCountEmbeds := 0
	for _, messageEmbedsPerSlice := range sliceOfSliceTotalMessageEmbeds {
		totalCountEmbeds += len(messageEmbedsPerSlice)
	}

	for x, slice := range sliceOfSliceTotalMessageEmbeds {
		responseString := ""
		if totalCountEmbeds < 11 {
			responseString = fmt.Sprintf("raidsummary daysorweeks|Single large message %s\n\nInclude non-main raids: %v", crackedBuiltin, !useOnlyMainRaids)
		} else {
			responseString = fmt.Sprintf("raidsummary daysorweeks|Chained message page %d of %d\n\nInclude non-main raids: %v", x+1, len(sliceOfSliceTotalMessageEmbeds), !useOnlyMainRaids)
		}
		fmt.Println("RESPONSE STRING", responseString)
		interactionResponse := NewInteractionResponseToSpecificCommand(2, responseString)
		interactionResponse.Data.Embeds = append(interactionResponse.Data.Embeds, slice...)
		returnInteractionResponses = append(returnInteractionResponses, &interactionResponse)
	}

	return returnInteractionResponses, nil
}

func GetDiscordUser(event *discordgo.InteractionCreate) *discordgo.User {
	if event.User != nil {
		return event.User
	}
	return event.Member.User
}

func UseSlashCommand(session *discordgo.Session) {
	session.AddHandler(func(innerSession *discordgo.Session, event *discordgo.InteractionCreate) {
		userID := GetDiscordUser(event).ID

		if event.Type == discordgo.InteractionMessageComponent {
			innerSession.ChannelMessageDelete(event.ChannelID, event.Message.ID)
			switch event.MessageComponentData().CustomID {
			case "general":
				{
					interactionResponse := NewInteractionResponseToSpecificCommand(3, fmt.Sprintf("Bot categories %s (NOT ALL COMMANDS)|", crackedBuiltin))
					fields := []discordgo.MessageEmbedField{
						{
							Name:   "✋ Say `hi` from any channel",
							Value:  "\u200B",
							Inline: true,
						},
						{
							Name:   "✍ Say `feedback` from any channel",
							Value:  "\u200B",
							Inline: true,
						},
						{
							Name:   "⏰ Say `myreminder` from any channel",
							Value:  "\u200B",
							Inline: true,
						},
						{
							Name:   "\u200B",
							Value:  "Run `/hi` to greet the bot and get a response",
							Inline: true,
						},
						{
							Name:   "\u200B",
							Value:  "Run `/feedback` to see sub-categories. This will notify the officer team",
							Inline: true,
						},
						{
							Name:   "\u200B",
							Value:  fmt.Sprintf("Run `/myreminder` to make custom alerts for ANYTHING you need\nYou can even create more alerts at once %s", crackedBuiltin),
							Inline: true,
						},
						{
							Name:   "\u200B",
							Value:  "Run `/joke` to see what happens 👀",
							Inline: false,
						},
					}
					pointerFields := make([]*discordgo.MessageEmbedField, len(fields))
					for i := range fields {
						pointerFields[i] = &fields[i]
					}
					interactionResponse.Data.Embeds[0].Fields = pointerFields
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error occured while trying to sen", err.Error())
					}
					time.Sleep(time.Second * 5)
					_, err = innerSession.FollowupMessageCreate(event.Interaction, false, NewWebhookParamGIF("the-hi-command.gif"))
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to sent a GIF to user %s during the slash command /howto during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
					}
					return
				}
			case "stats":
				{
					interactionResponse := NewInteractionResponseToSpecificCommand(3, fmt.Sprintf("Bot commands related to raiding %s|", crackedBuiltin))
					fields := []*discordgo.MessageEmbedField{
						{
							Name:   fmt.Sprintf("Type `/my` See simple raid-stats about YOU %s", crackedBuiltin),
							Value:  "`/myattendance` => (See your attendance since you joined)\n\n`/myraiderperformance` => (See average raid stats about you)\n\n`/mymissedraids` => (Get a list of specific raids u have missed)\n\n",
							Inline: true,
						},
						{
							Name:   "\u200B",
							Value:  "\u200B",
							Inline: false,
						},
						{
							Name:   fmt.Sprintf("Type `/aboutme` See a more detailed view of your raid-performance %s\n", crackedBuiltin),
							Value:  "⚠️`/aboutme logs` => (See log-specific data about you)⚠️\n\n⚠️`/aboutme playerinfo` => (See general data about your main)⚠️",
							Inline: false,
						},
					}
					interactionResponse.Data.Embeds[0].Fields = fields
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error occured while trying to sent response to user using slash command howto, during the function UseSlashCommand()", err.Error())
					}
					return
				}
			}
			customID := event.MessageComponentData().CustomID
			customIDSplit := strings.Split(customID, "/")
			if len(customIDSplit) != 2 {
				WriteErrorLog(fmt.Sprintf("The customID provided by the modolar response is not in the correct fomat: %s, expected <command>/<value>", customID), "Wrong format")
				interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("button %s|This button is not yet supported... Please contact %s", customID, SplitOfficerName(officerGMArlissa)["name"]))
				err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error response to user %s, using button %s, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession), customID), err.Error())
				}
				return
			}
			switch customIDSplit[0] {
			case "benchreason":
				{
					raidName := customIDSplit[1] //Will be safe as we check for len(2)
					trackedRaids := make(map[string]trackRaid)
					messageID := ""
					if raidCache := CheckForExistingCache(raidHelperCachePath); len(raidCache) > 0 {
						err := json.Unmarshal(raidCache, &trackedRaids)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to unmarshal raid-helper cache with path %s using button %s during the function UseSlashCommand()", raidHelperCachePath, customID), err.Error())
							return
						}
						foundBenchedRaidSlice := []trackRaid{}
						for ID, raid := range trackedRaids {
							if raid.RaidDiscordTitle == raidName {
								messageID = ID
								foundBenchedRaidSlice = append(foundBenchedRaidSlice, raid)
							}
						}
						if len(foundBenchedRaidSlice) == 0 {
							interactionResponse := NewInteractionResponseToSpecificCommand(0, "Raid %s not found|This might be due to the raid being deleted, please make sure a raid is already created before running this command")
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error response to user %s, using button %s during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession), customID), err.Error())
							}
							return
						}
						foundBenchRaid := foundBenchedRaidSlice[0]
						if len(foundBenchRaid.PlayersAlreadyTracked) == 0 {
							WriteErrorLog(fmt.Sprintf("There was no players currently benched from raid %s, during the button %s, during the function UseSlashCommand()", raidName, customID), "None benched")
							interactionResponse := NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("No one benched|No raiders are currently benched from raid: %s, please bench people first, then rerun command `/benchreason`", raidName))
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to tell the user %s that no raiders are currently benched in raid %s, using button %s during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession), raidName, customID), err.Error())
							}
							return
						}
						specificBenchReasonRaiderSlice := []string{}
						for raiderName := range foundBenchRaid.PlayersAlreadyTracked {
							specificBenchReasonRaiderSlice = append(specificBenchReasonRaiderSlice, fmt.Sprintf("\n%s=", raiderName))
						}
						benchReasonCopy := cloneBenchReasonResponse(slashCommandAdminCenter["benchreason"].Responses["reason"])
						benchReasonCopy.Response.Data.CustomID = fmt.Sprintf("%s/%s", benchReasonCopy.Response.Data.CustomID, messageID)
						for x, component := range benchReasonCopy.Response.Data.Components {
							row, ok := component.(discordgo.ActionsRow)
							if !ok {
								continue
							}
							for _, inner := range row.Components {
								input, ok := inner.(*discordgo.TextInput)
								if !ok {
									continue
								}
								if input.CustomID == "specific_reason" {
									input.Label = "Specific reasons for this week"
									input.Value = strings.Join(specificBenchReasonRaiderSlice, "\n")
									input.Required = true
								}
							}
							benchReasonCopy.Response.Data.Components[x] = row
						}
						err = innerSession.InteractionRespond(event.Interaction, benchReasonCopy.Response)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent modolar response to user %s using button %s, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession), customID), err.Error())
						}
					}
				}
			}
		} else if event.Type == discordgo.InteractionModalSubmit {
			interactionResponse := NewInteractionResponseToSpecificCommand(1, "feedback|Analysing the description provided...", discordgo.InteractionResponseDeferredChannelMessageWithSource)
			err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
			if err != nil {
				WriteErrorLog("An error occured while trying to sent initial response to the submit of a feedback description using slash command /feedback, during the function UseSlashCommand()", err.Error())
			}
			modolarData := event.ModalSubmitData()
			customIDSlice := strings.Split(modolarData.CustomID, "/")
			switch customIDSlice[0] {
			case "feedback_modal":
				{
					feedbackDescription := ""
					content := ""
					for _, component := range modolarData.Components {
						if row, ok := component.(*discordgo.ActionsRow); ok {
							if description, ok := row.Components[0].(*discordgo.TextInput); ok {
								feedbackDescription = description.Value
								break
							}
						}
					}
					anonymous, err := strconv.ParseBool(customIDSlice[3])
					if err != nil {
						WriteErrorLog("An error occured while trying to convert string value of %s to bool, during the slash command feedback, during the function UseSlashCommand()", err.Error())
						interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("feedback|Problem submitting feedback, please contact <@%s>", SplitOfficerName(officerGMArlissa)["ID"]))
						err = innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog("An error occured while trying to respond to slash command /feedback after submitting a description, during the function UshSlashCommand()", err.Error())
						}
						break
					}
					content = "**############# FEEDBACK START #############**"
					playerName := ""
					if !anonymous {
						playerName = ResolvePlayerID(userID, innerSession)
					} else {
						playerName = "Anonymous"
					}
					content = fmt.Sprintf("%s\n\n**Raider:** %s\n\n**Category:** %s\n\n**Description:** %s\n\n**############# FEEDBACK END #############**", content, playerName, customIDSlice[1], feedbackDescription)
					threadName := fmt.Sprintf("Topic: %s - From: %s", customIDSlice[1], playerName)
					thread, err := innerSession.ThreadStart(channelFeedback, threadName, discordgo.ChannelTypeGuildPublicThread, 10080)
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to create new thread %s in channel %s, using slash command /feedback, during the function UseSlashCommand()", threadName), err.Error())
					} else {
						WriteInformationLog(fmt.Sprintf("The new thread %s has been successfully created for feedback given by player: %s, using the slash command /feedback, during the function UseSlashCommand()", threadName, playerName), "Creating thread")
					}
					_, err = innerSession.ChannelMessageSend(thread.ID, content)
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to send feedback to the thread with name %s and ID: %s, using slash command /feedback, during the function UseSlashCommand()", threadName, thread.ID), err.Error())
					}
				}
			case "bench_modal":
				{
					generalBenchReason := ""
					messageID := customIDSlice[1]
					mapOfSpecificReason := make(map[string]string)
					for _, component := range modolarData.Components {
						if row, ok := component.(*discordgo.ActionsRow); ok {
							for _, anyType := range row.Components {
								if component, ok := anyType.(*discordgo.TextInput); ok {
									switch component.CustomID {
									case "specific_reason":
										{
											sliceOfSpecificPlayersNotCleaned := strings.Split(component.Value, "\n")
											for _, playerLine := range sliceOfSpecificPlayersNotCleaned {
												playerLineSlice := strings.Split(playerLine, "=")
												if len(playerLineSlice) == 2 {
													playerName := playerLineSlice[0]
													if mapOfSpecificReason[playerName] == "" {
														mapOfSpecificReason[playerName] = playerLineSlice[1]
													}
												} else {
													WriteErrorLog(fmt.Sprintf("The format provided in the modal is incorrect - got %s but need format raider=reason", component.Value), "Wrong provided format")
													continue
												}
											}
										}
									default:
										{
											generalBenchReason = component.Value
										}
									}
								}
							}
						}
					}
					allTrackedRaids := ReadWriteRaidHelperCache()
					trackedRaid := trackRaid{}
					if _, ok := allTrackedRaids[messageID]; !ok {
						interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("Raid to bench for not found|This should not happen - Please contact %s", SplitOfficerName(officerGMArlissa)["name"]))
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent an error response to user %s using the modolar benchreason, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
						}
						return
					}
					trackedRaid = allTrackedRaids[messageID]
					currentBenchedRaiders := []raiderProfile{}
					allRaiders := ReadWriteRaiderProfiles([]raiderProfile{}, false)
					for playerName, reason := range mapOfSpecificReason {
						for name := range trackedRaid.PlayersAlreadyTracked {
							for _, raider := range allRaiders {
								if name == raider.MainCharName {
									currentBenchedRaiders = append(currentBenchedRaiders, raider)
									break
								}
							}
							if playerName == name {
								currentBench := trackedRaid.PlayersAlreadyTracked[playerName]
								if len(reason) != 0 {
									currentBench.Reason = reason
								} else {
									currentBench.Reason = generalBenchReason
								}
								trackedRaid.PlayersAlreadyTracked[playerName] = currentBench
							}
						}
					}

					for x, raider := range currentBenchedRaiders {
						benchInfoLastWeek := make(map[string][]bench)
						benchInfoLastWeek["lastWeek"] = append(benchInfoLastWeek["lastWeek"], trackedRaid.PlayersAlreadyTracked[raider.MainCharName])
						currentBenchedRaiders[x].BenchInfo = benchInfoLastWeek
					}
					ReadWriteRaiderProfiles(currentBenchedRaiders, false)
					//interactionResponse = NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("Successfully updated %d raider-profiles|Bench reasons updated", len(currentRaiderProfiles)))
					_, err := innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
						Embeds: &interactionResponse.Data.Embeds,
					})
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to do final success respond to user %s using modolar bench reason, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
					}
					return
				}
			}

			interactionResponse = NewInteractionResponseToSpecificCommand(1, "submit feedback|Message success", discordgo.InteractionResponseDeferredChannelMessageWithSource)
			innerSession.InteractionRespond(event.Interaction, &interactionResponse)
			_, err = innerSession.FollowupMessageCreate(event.Interaction, true, &discordgo.WebhookParams{
				Content: "Thank you - Your feedback is **HIGHLY** appriciated. ALL officers has just recieved this information",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				WriteErrorLog("An error occured while trying to respond to slash command /feedback and the users final message, during the function UshSlashCommand()", err.Error())
			}

		} else if event.Type != discordgo.InteractionApplicationCommand && event.GuildID != "" && userID != innerSession.State.User.ID {
			return
		}
		// Acknowledge the interaction immediately to avoid the "Application did not respond" error
		interactionData := discordgo.ApplicationCommandInteractionData{}
		if event.Type == discordgo.InteractionApplicationCommand {
			interactionData = event.ApplicationCommandData()
		}
		if CheckForOfficerRank(userID, innerSession) {
			//userNam e := user.Username
			if len(interactionData.Options) == 0 {
				switch interactionData.Name {
				case "deletebotchannel":
					{
						interactionResponse := NewInteractionResponseToSpecificCommand(1, "Running command...|", discordgo.InteractionResponseDeferredChannelMessageWithSource)
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured during the initial defered response to user %s, using slash command /deletebotchannel, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
							return
						}
						channels, err := innerSession.GuildChannels(serverID)
						for _, channel := range channels {
							if channel.Name == channelNameAnnouncement {
								_, err = innerSession.ChannelDelete(channel.ID)
								if err != nil {
									WriteErrorLog(fmt.Sprintf("An error occured while trying to delete channel with ID %s, using slash command /deletebotchannel, during the function UseSlashCommand()", channel.ID), err.Error())
									interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("deletebotchannel|It was not possible to delete channel with ID: %s, please let %s know", channel.Name, SplitOfficerName(officerGMArlissa)["Name"]))
									_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
										Embeds: &interactionResponse.Data.Embeds,
									})
									if err != nil {
										WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error response to user %s, using slash command /deletebotchannel, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
									}
									return
								}
							}
						}
						interactionResponse = NewInteractionResponseToSpecificCommand(2, "deletebotchannel|Bot channel(s) deleted.. Please give the bot a moment to recreate it...")
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while truing to sent success response to user %s, using slash command /deletebotchannel, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
						}
					}
				case "benchreason":
					{
						interactionResponse := NewInteractionResponseToSpecificCommand(1, "Initiating benching|", discordgo.InteractionResponseDeferredChannelMessageWithSource)
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured during the initial defered response to user %s, using slash command /benchreason, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
							return
						}
						cacheRaidHelper := CheckForExistingCache(raidHelperCachePath)
						if len(cacheRaidHelper) == 0 {
							interactionResponse = NewInteractionResponseToSpecificCommand(1, "benchreason|No raids found... Please make sure to have created the actual raid-signup using raid-helper, before using this command")
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to tell user %s that no current raids are active using slash command /benchreason, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
							}
							return
						}
						raidCacheMap := make(map[string]trackRaid)
						err = json.Unmarshal(cacheRaidHelper, &raidCacheMap)
						if err != nil {
							interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("benchreason|An error occured inside the bot when trying to convert raid-helper bytes to struct - Please contact %s", ResolvePlayerID(SplitOfficerName(officerGMArlissa)["name"], innerSession)))
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error response to user %s using slash command /benchreason, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
							}
							return
						}

						buttonsOfRaids := []discordgo.Button{}
						for _, raid := range raidCacheMap {
							raidTitle := raid.RaidDiscordTitle
							button := discordgo.Button{
								Label:    raidTitle,
								Style:    discordgo.PrimaryButton,
								CustomID: fmt.Sprintf("benchreason/%s", raidTitle),
							}
							buttonsOfRaids = append(buttonsOfRaids, button)
						}
						interactionResponse = NewInteractionResponseToSpecificCommand(1, "Raid selection|Please press the raid, where you want to give reasons for benching raiders")
						row := discordgo.ActionsRow{}
						for _, button := range buttonsOfRaids {
							row.Components = append(row.Components, button)
						}
						interactionResponse.Data.Components = []discordgo.MessageComponent{
							row,
						}

						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Content:    GetStringPointer("`Select raid below`"),
							Components: &interactionResponse.Data.Components,
						})
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent a length of %d buttons using slash command /benchreason, during the function UseSlashCommand()", len(row.Components)), err.Error())
						}
					}
				case "resetraidcache":
					{
						interactionResponse := NewInteractionResponseToSpecificCommand(1, "Starting full reset of raiding cache|", discordgo.InteractionResponseDeferredChannelMessageWithSource)
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to make initial response to user %s using slash command /resetraidcache, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
						}
						interactionResponse = NewInteractionResponseToSpecificCommand(1, "Calculating amount of logs!|")
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog("An error occured while trying to sent a message to user %s using the slash command /resetraidcache 1, during the function UseSlashCommand", err.Error())
						}
						currentLogsBase := GetWarcraftLogsData(mapOfWarcaftLogsQueries["guildLogsRaidIDs"])
						lenCurrentLogBase := 0
						if allLogs, ok := currentLogsBase["logs"].([]logsBase); ok {
							lenCurrentLogBase = len(allLogs)
						} else {
							WriteErrorLog(fmt.Sprintf("Was not possible to find any valid guild raids using the function GetWarcraftLogsData() on slash command raidreset from user %s", userID), "During function UseSlashCommand()")

						}
						interactionResponse = NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("Progress on command|The bot found a total of %d logs - Please wait...", lenCurrentLogBase))
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog("An error occured while trying to sent a message to user %s using the slash command /resetraidcache 2, during the function UseSlashCommand", err.Error())
						}
						GetAllWarcraftLogsRaidData(false, false, "", innerSession, event.Interaction)
						interactionResponse = NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("resetraidcache|The raid-data is now syncronized directly with Warcraftlogs - ALL data provided by any bot command and related to raid-info is now valid %s", crackedBuiltin))
						_, err = innerSession.ChannelMessageSendEmbeds(event.ChannelID, interactionResponse.Data.Embeds)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to send a final response to the user %s using slash command resetraidcache, inside of the function UseSlashCommand()", userID), err.Error())
						} else {
							WriteInformationLog(fmt.Sprintf("A message successfully sent to the user %s during the function UseSlashCommand()", userID), "Successfully sent embed message")
						}
					}
				case "deletechannelcontent":
					{
						interactionResponse := NewInteractionResponseToSpecificCommand(1, "Deleting channel content|Please wait...", discordgo.InteractionResponseDeferredChannelMessageWithSource)
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog("An error occured while trying to sent initial response to user %s using slash command /deletechannelcontent, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession))
						}
						DeleteMessagesInBulk(event.ChannelID, innerSession)
						if err != nil {
							WriteErrorLog("An error occured while trying to delete messages", err.Error())

						}
						interactionResponse = NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("deletechannelcontent|Deletion complete... %s", crackedBuiltin))
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog("An error occured while making a final response to user using slash command /deletechannelcontent, during function UseSlashCommand()", err.Error())
						}
					}
				case "updateweeklyattendance":
					{
						interactionResponse := NewInteractionResponseToSpecificCommand(1, "Starting work on attendance update|", discordgo.InteractionResponseDeferredChannelMessageWithSource)
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog("An error occured while trying to start the initial deferred response to the user, with the slash command /updateweeklyattendance during the function UseSlashCommand()", err.Error())
							break
						}
						interactionResponse = NewInteractionResponseToSpecificCommand(1, "Starting job|Analysing raider attendance...")
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog("An error occured while trying to letting the admin know that the bot is analysing raider attendance using slash command /updateweeklyattendance, during the functioin UseSlashCommand()", err.Error())
						}
						returnString := AddWeeklyRaiderAttendance(innerSession, event.Interaction)
						if strings.Contains(returnString, "error") || strings.Contains(returnString, "cannot") {
							interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("updateweeklyattendance|The following error occured inside the bot: %s", returnString))
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							if err != nil {
								WriteErrorLog("An error occured while trying to sent error response to user %s with slash command updateweekyattendance 1, during the function UseSlashCommand()", err.Error())
							}
							break
						}
						interactionResponse = NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("updateweeklyattendance|%s", returnString))
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog("An error occured while trying to sent error response to user %s with slash command updateweekyattendance 2, during the function UseSlashCommand()", err.Error())
						}
					}
				case "seeraiderattendance":
					{
						currentRaidersBytes := CheckForExistingCache(raiderProfilesCachePath)
						if len(currentRaidersBytes) == 0 {
							interactionResponse := NewInteractionResponseToSpecificCommand(0, "seeraiderattendance|No raider-profiles found... Please contact Arlissa")
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog("An error occured while trying to sent error response to user %s with slash command seeraiderattendance, during the function UseSlashCommand()", err.Error())
							}
							break
						}
						raiderStruct := raiderProfiles{}
						onlyCurrentRaiders := []string{}
						err := json.Unmarshal(currentRaidersBytes, &raiderStruct)
						if err != nil {
							interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("seeraiderattendance|%s... Please contact Arlissa", err.Error()))
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog("An error occured while trying to sent error response to user %s with slash command seeraiderattendance, during the function UseSlashCommand()", err.Error())
							}
							break
						}
						sort.Slice(raiderStruct.Raiders, func(i, j int) bool {
							return raiderStruct.Raiders[i].AttendanceInfo["guildStart"].RaidProcent > raiderStruct.Raiders[j].AttendanceInfo["guildStart"].RaidProcent
						})
						for _, raider := range raiderStruct.Raiders {
							if strings.Contains(strings.Join(raider.DiscordRoles, ","), roleRaider) {
								totalOGPoints := math.Floor(float64(10*raider.AttendanceInfo["guildStart"].RaidCount) * (100/raider.AttendanceInfo["guildStart"].RaidProcent + 1))
								onlyCurrentRaiders = append(onlyCurrentRaiders, fmt.Sprintf("%s => %.0f => %.0f", raider.MainCharName, raider.AttendanceInfo["guildStart"].RaidProcent, totalOGPoints))
							}
						}
						interactionResponse := NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("seeraiderattendance|playerName => OG %% => 3 Months %% => 2 Months %% => 1 Month %%\n%s", strings.Join(onlyCurrentRaiders, "\n")))
						innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					}
				}
			} else {
				switch interactionData.Name {
				case "announcebot":
				{
					patternDiscordMentions := regexp.MustCompile(`<(@!?|@&|#)\d{17,19}>`)
					active := false
					interactionResponse := NewInteractionResponseToSpecificCommand(1, "Starting announcement|", discordgo.InteractionResponseDeferredChannelMessageWithSource)
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to sent the initial response to user %s using slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
						return
					}
					announceDescription := ""
					announceTitle := ""
					channelID := ""
					for _, option := range interactionData.Options {
						if _, ok := option.Value.(string); !ok {
							WriteErrorLog(fmt.Sprintf("The discordgo template is set to garuentee this to be a string, but its not, for officer %s, using slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), "Data is not string")
							interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("announcebot|A bug happened inside the bot - Please report this to %s", SplitOfficerName(officerGMArlissa)["name"]))
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent an error response to user %s using slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
							}
							return
						}	
						switch option.Name {
							case "title": {
								announceTitle = option.Value.(string)
								fmt.Println("TITLE", announceTitle)
								if exists, channelID := CheckForPost(announceTitle); exists {
									interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("announcebot|An announcement with that title already exists <#%s>\nPlease delete the old one first or run again with a different title", channelID))
									_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
										Embeds: &interactionResponse.Data.Embeds,
									})
									if err != nil {
										WriteErrorLog(fmt.Sprintf("An error occured while trying to sent an error response of announcement already exists in channel ID %s to user %s using slash command /announcebot, during the function UseSlashCommand()", channelID, ResolvePlayerID(userID, innerSession)), err.Error())
									}
									return
								} else {
									fmt.Println("TITLE DOESNT EXIST")
								}
							}
							case "description": {
								announceDescription = option.StringValue()
								fmt.Println("Descriptiojm", announceDescription, len(announceDescription))
								if len(announceDescription) <= 25 {
									interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("announcebot|The announcement provided of `%s` is less than 25 characters long...", announceDescription))
									_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
										Embeds: &interactionResponse.Data.Embeds,
									})
									if err != nil {
										WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error reponse to user %s about description being too short, found len %d but need len 25 minimum, using slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession), len(announceDescription)), err.Error())
									}
									fmt.Println("Why dont we stop on this return?")
									return
								}
								mentions := patternDiscordMentions.FindAllString(announceDescription, -1)
								if len(mentions) == 0 && (strings.Contains(announceDescription, "#") || strings.Contains(announceDescription, "@")) {
									interactionResponse = NewInteractionResponseToSpecificCommand(1, "announcebot|It looks like you tried to add either users, channels or roles to the description.\nMake sure tags are added to the description correctly and try again!")
									_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
										Embeds: &interactionResponse.Data.Embeds,
									})
									if err != nil {
										WriteErrorLog(fmt.Sprintf("An error occured while trying to sent warning to user %s using slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
									}
									return
								} else {
									fmt.Println("DO WE REACH HERE?", announceDescription)
								}
							}
							case "channel": {
									if id := RetrieveChannelID(option.StringValue()); id != "" {
										channelID = id
									} else {
										interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("announcebot|The channel provided %s is not valid - Please rerun and use format #<channel>.\nMake sure the channel link is parsed, so DONT type it manually", option.StringValue()))
										_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
											Embeds: &interactionResponse.Data.Embeds,
										})
										if err != nil {
											WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error response to user %s about the channel format being incorrect, using the slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
										}
										return
									}
								}
							}
						}
						fmt.Println("DID WE FIND A CHANNEL ID?", channelID)
						embedToSend := &discordgo.MessageEmbed{
							Title: announceTitle,
							Color: blueColor,
							Fields: []*discordgo.MessageEmbedField{
								{	
									Value: announceDescription,
									Name: "\u200B",
								},
							},
						}
						linkedChannelName := GetChannelName(channelID, innerSession)
						if linkedChannelName == "" && active {
							WriteErrorLog(fmt.Sprintf("The length of the resolved linked channel name is 0, which means we cannot find the channel again when it changes id - This means the config found at %s was changed BEFORE this code could run, which means it will return early, for user %s using slash command /announcebot, during the function UseSlashCommand()", configPath, ResolvePlayerID(userID, innerSession)), "Channel not found")
							interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("announcebot|The channel <#%s> has changed before the bot could find it, please rerun command `/announcebot`\nPro tip: press arrow up, in the chat to reuse the same command and values", channelID))
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while tryhing to sent error response to user %s using slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
							}
							return
						}
						message, err := innerSession.ChannelMessageSendComplex(channelInfo, &discordgo.MessageSend{
							Embeds: []*discordgo.MessageEmbed{embedToSend},
						})
						active = channelID != ""
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent the announce message in channel %s, user %s using slash command /botannounce, during the function UseSlashCommand()", channelInfo, ResolvePlayerID(userID, innerSession)), err.Error())
							interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("announcebot|An error happened inside the bot - %s", err.Error()))
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error response to user %s about the announcement message that could not be sent, using slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
							}
							return
						}
						_, err = innerSession.ChannelMessageSend(channelInfo, strings.Join(SeperateAnyTagsInMessage(announceDescription), ", "))
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent tag message, for user %s using slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
							interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("announcebot|An error happened inside the bot - %s", err.Error()))
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error response to user %s about the announcement message that could not be sent, using slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
							}
							return
						}
						post := trackPost{
							ChannelID: message.ChannelID,
							LinkedChannelID: channelID,
							Active: active,
							MessageID: message.ID,
							LinkedChannelName:linkedChannelName,
						}
						ReadWriteTrackPosts(post)
						select {
							case trackCacheChanged <- struct{}{}:
							default:
						}
						fmt.Println("DO WE REACH TOWARDS THE END=?", post)
						returnString := ""
						if active {
							returnString = fmt.Sprintf("Successfully created and will be tracked to channel ID <#%s>\nTo stop the tracking of a specific post, please run `/stopannouncebot`", channelID)
						} else {
							returnString = fmt.Sprintf("Successfully created message and can be found here => https://discord.com/channels/%s/%s/%s", serverID, post.ChannelID, post.MessageID)
						}
						interactionResponse = NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("announcebot|%s", returnString))
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent final successfull response to user %s, using the slash command /announcebot, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
						}
						
				}
				case "raidsummary":
					{
						useOnlyMainRaids, _ := CheckUserBoolResponseFlag(interactionData.Options, "includesmallraids")
						switch interactionData.Options[0].Name {
						case "alltime":
							{
								fmt.Println("YEP WE ARE HERE")
							}
						case "month":
							{
								interactionResponses, err := NewWarcraftLogsGeneralDataResponse(useOnlyMainRaids, "")
								if err != nil {
									responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary month|An error inside the bot, please report this error to %s\n\nError: %s", SplitOfficerName(officerGMArlissa)["Name"], err.Error()))
									err := innerSession.InteractionRespond(event.Interaction, &responseError)
									if err != nil {
										WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary month, during the function UseSlashCommand()", err.Error())
									}
									break
								}
								if len(interactionResponses[0].Data.Embeds) > 10 {
									responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary month|An error inside the bot, please report this error to %s\n\nError: The number of embeds exceeded 10, raids must be merged incorrectly %s", SplitOfficerName(officerGMArlissa)["Name"], crackedBuiltin))
									for _, embed := range interactionResponses[0].Data.Embeds {
										fmt.Println("EMBED TITLE::", embed.Title)
									}
									err := innerSession.InteractionRespond(event.Interaction, &responseError)
									if err != nil {
										WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary month, during the function UseSlashCommand()", err.Error())
									}
									break
								}
								err = innerSession.InteractionRespond(event.Interaction, interactionResponses[0])
								if err != nil {
									WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary month, during the function UseSlashCommand()", err.Error())
								}
							}
						case "lastraid":
							{

							}
						case "specificraid":
							{

							}
						case "daysorweeks":
							{
								if useDefaultTimeString, _ := CheckUserBoolResponseFlag(interactionData.Options, "timestring"); useDefaultTimeString {
									interactionResponses, err := NewWarcraftLogsGeneralDataResponse(useOnlyMainRaids, "30d")
									if err != nil {
										responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary month|An error inside the bot, please report this error to %s\n\nError: %s", SplitOfficerName(officerGMArlissa)["Name"], err.Error()))
										err := innerSession.InteractionRespond(event.Interaction, &responseError)
										if err != nil {
											WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary month, during the function UseSlashCommand()", err.Error())
										}
										break
									}
									if len(interactionResponses) == 0 {
										fmt.Println("WE BROKE FROM THE INTERACTION")
										break
									}
									if len(interactionResponses[0].Data.Embeds) > 9 {
										responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary month|An error inside the bot, please report this error to %s\n\nError: The number of embeds exceeded 10, raids must be merged incorrectly %s", SplitOfficerName(officerGMArlissa)["Name"], antiCrackedBuiltin))
										err := innerSession.InteractionRespond(event.Interaction, &responseError)
										if err != nil {
											WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary month, during the function UseSlashCommand()", err.Error())
										}
										break
									}
									err = innerSession.InteractionRespond(event.Interaction, interactionResponses[0])
									if err != nil {
										WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary month, during the function UseSlashCommand()", err.Error())
									}
								} else {
									subCommand := interactionData.Options[0]
									timeStringValue := ""
									for _, option := range subCommand.Options {
										if option.Name == "timestring" {
											if option.Value != nil {
												timeStringValue = option.Value.(string)
												// use timestring here
											}
										}
									}
									if timeStringValue == "" {
										for _, option := range interactionData.Options {
											fmt.Println("OPTIONAL NAME", option.Name, option.Value, len(interactionData.Options))
										}
										responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary dayorweek|An error inside the bot, please report this error to %s\n\nError: Even though check through inner function checkUserBoolResponseFlag(), the command response is nil.. %s", SplitOfficerName(officerGMArlissa)["Name"], antiCrackedBuiltin))
										err := innerSession.InteractionRespond(event.Interaction, &responseError)
										if err != nil {
											WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary daysorweeks, during the function UseSlashCommand()", err.Error())
										}
										break
									}
									interactionResponses, err := NewWarcraftLogsGeneralDataResponse(useOnlyMainRaids, timeStringValue)
									fmt.Println("RESPONSES FOUND:", len(interactionResponses))
									if err != nil {
										WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary daysorweeks, during the function UseSlashCommand()", err.Error())
									}
									if len(interactionResponses) > 0 {
										interactionResponse := NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("raidsummary daysorweeks|Request period: **%s**\n\n**Please note that the command output might be split into multiple messages** %s", timeStringValue, crackedBuiltin))
										err = innerSession.InteractionRespond(event.Interaction, &interactionResponse)

										if err != nil {
											WriteErrorLog("An error occurred while sending the initial interaction response", err.Error())
										}

										for _, response := range interactionResponses {
											// Ensure the followup is ephemeral
											response.Data.Flags = discordgo.MessageFlagsEphemeral

											// Convert InteractionResponseData into WebhookParams
											followup := &discordgo.WebhookParams{
												Content: response.Data.Content,
												Embeds:  response.Data.Embeds,
												Flags:   discordgo.MessageFlagsEphemeral,
											}

											_, err = innerSession.FollowupMessageCreate(event.Interaction, true, followup)
											if err != nil {
												WriteErrorLog("An error occurred while sending follow-up embeds", err.Error())
											}
										}
									}

									if err != nil {
										WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary daysorweeks, during the function UseSlashCommand()", err.Error())
									}
								}
							}
						}
					}
				case "promotetrial":
					{
						nameSlice := strings.Split(interactionData.Options[0].StringValue(), "@")
						if len(nameSlice) != 2 {
							interactionResponse := NewInteractionResponseToSpecificCommand(0, "promotetrial|Format incorrect - Must be @<playername>, e.g. @Arlissa")
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent response to user %s and command playername during the function UseSlashCommand()", userID), err.Error())
							}
							break
						}
						taggedID := strings.ReplaceAll(nameSlice[1], ">", "")
						nickNameTrial := ResolvePlayerID(taggedID, innerSession)
						matched := false
						for _, trial := range RetrieveUsersInRole([]string{roleTrial}, innerSession) {
							if trial == taggedID {

								matched = true
								err := innerSession.GuildMemberRoleAdd(serverID, trial, roleRaider)
								err2 := innerSession.GuildMemberRoleRemove(serverID, trial, roleTrial)
								if err != nil || err2 != nil {
									response := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("promotetrial|The following error occured when trying to add %s to the raider role or remove trial", nickNameTrial))
									innerSession.InteractionRespond(event.Interaction, &response)
									WriteErrorLog(fmt.Sprintf("An error occured while trying to promote user %s to raider or remove trial role, during the function UseSlashCommand()", nickNameTrial), fmt.Sprintf("%s%s", err.Error(), err2.Error()))
									break
								}
								break
							}
						}
						if !matched {
							response := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("promotetrial|The user %s did not have the trial role...", nickNameTrial))
							err := innerSession.InteractionRespond(event.Interaction, &response)
							if err != nil {
								WriteErrorLog("An error ocurred while trying to respond to admin, during the function UseSlashCommand()", err.Error())
							}
						}
						response := NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("promotetrial|The player %s has been successfully promoted to rank Raider", nickNameTrial))
						err := innerSession.InteractionRespond(event.Interaction, &response)
						if err != nil {
							WriteErrorLog("An error occured while trying to respond to admin, during the function UseSlashCommand()", err.Error())
						}
					}
				case "simplemessage":
					{
						if interactionData.Options == nil {
							WriteInformationLog("The user %s did not provide any value to the command, this is crucial for the code to run, during function UseSlashCommand() breaing from loop...", "User input is nil")
							break
						}

						if stringValue, ok := interactionData.Options[0].Value.(string); ok {
							innerSession.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
							})
							copyTemplate := DeepCopyInteractionResponse(slashCommandAdminCenter["simplemessage"].Responses["messagetouser"].Response)
							sliceOfResponseString := SeperateAnyTagsInMessage(stringValue)
							embedFields := []*discordgo.MessageEmbedField{
								{Value: stringValue},
							}
							copyTemplate.Data.Embeds[0].Fields = embedFields
							_, err := innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &copyTemplate.Data.Embeds,
							})
							if err != nil {
								err = innerSession.InteractionRespond(event.Interaction, copyTemplate)
								if err != nil {
									WriteErrorLog("It was not possible to sent a failed slash command error back to the admin, breaking early inside function UseSlashCommand()", err.Error())
									break
								}

							}
							if sliceOfResponseString != nil {
								tagUsersString := strings.Join(sliceOfResponseString, ", ")
								innerSession.ChannelMessageSend(event.ChannelID, tagUsersString)
							}
						} else {
							WriteErrorLog(fmt.Sprintf("It was not possible to convert the value %s to string, this is crucial for this slash command, will break early...", interactionData.Options[0].Value), "During function UseSlashCommand()")
						}
					}
				case "seebench":
					{
						options := interactionData.Options
						raiderDiscordID := ""
						period := ""
						singleRaider := false
						for _, option := range options {
							switch option.Name {
							case "playername":
								{
									singleRaider = true
									raiderDiscordID = option.StringValue()
								}
							case "period":
								{
									period = option.StringValue()
								}
							}
						}

						interactionResponse := NewInteractionResponseToSpecificCommand(1, "seebench|Running command")
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog("An error occured while trying to send initial defered response to the user %s, using slash command /seebench, during the function UseSlashCommand()", err.Error())
							break
						}
						if singleRaider && len(strings.Split(raiderDiscordID, "@")) != 2 && raiderDiscordID != "" {
							fmt.Println("VALUES:", singleRaider, len(strings.Split(raiderDiscordID, "@")), raiderDiscordID)
							interactionResponse := NewInteractionResponseToSpecificCommand(0, "seebench|Format incorrect - Must be @<playername>, e.g. @Arlissa")
							_, err := innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent response to user %s and command playername during the function UseSlashCommand()", userID), err.Error())
							}
							break
						}
						raiderProfiles := []raiderProfile{}
						if raiderDiscordID == "" {
							raiderProfiles = GetRaiderProfiles()
						} else {
							profile, _ := GetRaiderProfile(FormatRaiderID(raiderDiscordID))
							raiderProfiles = append(raiderProfiles, profile)
						}
						switch len(raiderProfiles) {
						case 0:
							{
								interactionResponse := NewInteractionResponseToSpecificCommand(0, "seebench|No raider profiles found... Please run bot command `/resetraidcache` Then run this command again")
								err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
								if err != nil {
									WriteErrorLog("An error occured while trying to send error response to user %s using slash command /seebench, during the function UseSlashCommand()", err.Error())
								}
								break
							}
						default:
							{
								returnString := NewRaidProfileBenchSummaries(raiderProfiles, period)

								interactionResponse = NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("`seebench` For period: **%s**|%s", period, returnString))
								_, err = session.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
									Embeds: &interactionResponse.Data.Embeds,
								})
								if err != nil {
									WriteErrorLog("An error occured while trying to send final response to user %s, during the slash command /seebench, during the function UseSlashCommand()", err.Error())
								}
								//interactionResponse := NewInteractionResponseToSpecificCommand(2, )
							}
						}
					}
				case "seeraiderattendance":
					{
						nameSlice := strings.Split(interactionData.Options[0].StringValue(), "@")
						if len(nameSlice) != 2 {
							interactionResponse := NewInteractionResponseToSpecificCommand(0, "seeraiderattendance|Format incorrect - Must be @<playername>, e.g. @Arlissa")
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent response to user %s and command playername during the function UseSlashCommand()", userID), err.Error())
							}
							break
						}
						raiderProfile, errString := GetRaiderProfile(nameSlice[1])
						if errString != "" {
							interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("playername|%s", errString))
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent response to user %s and command playername during the function UseSlashCommand()", userID), err.Error())
							}
							break
						}
						attendanceSummary := NewRaidProfileAttendanceSummary(raiderProfile)
						interactionResponse := NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("seeraiderattendance|%s", attendanceSummary))
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent response to user %s and command playername during the function UseSlashCommand()", userID), err.Error())
						}
					}
				case "seeraidermissedraids":
					{
						nameSlice := strings.Split(interactionData.Options[0].StringValue(), "@")
						if len(nameSlice) != 2 {
							interactionResponse := NewInteractionResponseToSpecificCommand(0, "seeraiderattendance|Format incorrect - Must be @<playername>, e.g. @Arlissa")
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent response to user %s and command playername during the function UseSlashCommand()", userID), err.Error())
							}
							break
						}
						raiderProfile, errString := GetRaiderProfile(nameSlice[1])
						if errString != "" {
							interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("seeraidermissedraids|The player %s is not a raider...", raiderProfile.MainCharName))
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog("An error occured while trying to sent respond to user from slash command seeraidermissedraids, during the function UseSlashCommand()", err.Error())
								break
							}
						}
						mapOfRaidsMissed := make(map[string]bool)
						listOfRaids := []string{}
						for period, raiderAttendance := range raiderProfile.AttendanceInfo {
							for _, missedRaid := range raiderAttendance.RaidsMissed {
								if !mapOfRaidsMissed[missedRaid] && period != "guildStart" {
									sliceOfRaidName := strings.Split(missedRaid, "/")
									listOfRaids = append(listOfRaids, fmt.Sprintf("(%s) https://fresh.warcraftlogs.com/reports/%s", sliceOfRaidName[0], sliceOfRaidName[1]))
									mapOfRaidsMissed[missedRaid] = true
								}
							}
						}
						if len(listOfRaids) == 0 {
							interactionResponse := NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("seeraidermissedraids|No raids missed for raider: %s", raiderProfile.MainCharName))
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog("An error occured while trying to sent respond to the user with command seeraidermissedraids, during the function UseSlashCommand()", err.Error())
								break
							}
						} else {
							interactionResponse := NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("seeraidermissedraids|The following %d raids has been missed last 3 months by: %s\n\n%s", len(listOfRaids), raiderProfile.MainCharName, strings.Join(listOfRaids, "\n")))
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog("An error occured while trying to sent respond to the user with command seeraidermissedraids, during the function UseSlashCommand()", err.Error())
								break
							}
						}
					}
				case "syncdiscordroles":
					{
						if !interactionData.Options[0].BoolValue() {
							interactionResponse := NewInteractionResponseToSpecificCommand(1, "syncdiscordroles|Delta sync is not enabled yet... Please set fullsync to true")
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog("An error occured while trying to sent respond to the user with command syncdiscordroles, during the function UseSlashCommand()", err.Error())
							}
						}
						interactionResponse := NewInteractionResponseToSpecificCommand(1, "syncdiscordroles|Starting sync, please wait...", discordgo.InteractionResponseDeferredChannelMessageWithSource)
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog("An error occured while trying to sent respond to the user with command syncdiscordroles, during the function UseSlashCommand()", err.Error())
							break
						}
						returnAnswer := ManageMergedGroups(innerSession, "full")

						if len(returnAnswer) == 1 {
							interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("syncdiscordroles|%s", returnAnswer[0]))
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Content: GetStringPointer(returnAnswer[0]),
							})
							if err != nil {
								WriteErrorLog("An error occured while trying to sent respond to the user with command syncdiscordroles, during the function UseSlashCommand()", err.Error())
								break
							}
						}

						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Content: GetStringPointer(returnAnswer[0]),
						})

						if err != nil {
							WriteErrorLog("An error occured while trying to sent respond to the user with command syncdiscordroles, during the function UseSlashCommand()", err.Error())
						}
					}
				case "aboutme":
					{
						fmt.Println("WE DONT REACH HERE?")
						interactionResponse := NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("aboutme|This feature is not out yet 🚧 please contact <@%s> for more information", SplitOfficerName(officerGMArlissa)["ID"]))
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog("An error occured while trying to reply to user %s using slash command /aboutme, during function UseSlashCommand()", err.Error())
						}
					}
				}
			}
		}
		if CheckForRaiderRank(userID, innerSession) && strings.Contains("myattendance,mymissedraids,mynewmain,myraiderperformance,myreminder,hi,howto,joke,feedback", interactionData.Name) {
			newRaiderProfile, _ := GetRaiderProfile(userID)
			switch interactionData.Name {
			case "myattendance":
				{
					attendanceSummary := NewRaidProfileAttendanceSummary(newRaiderProfile)
					interactionResponse := NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("myattendance|%s", attendanceSummary))
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error ocurred while trying to sent the response error to the user from slash command /myattendance, during the function UseSlashCommand()", err.Error())
					}
				}
			case "mymissedraids":
				{
					mapOfRaidsMissed := make(map[string]bool)
					listOfRaids := []string{}
					for period, raiderAttendance := range newRaiderProfile.AttendanceInfo {
						for _, missedRaid := range raiderAttendance.RaidsMissed {
							if !mapOfRaidsMissed[missedRaid] && period != "guildStart" {
								sliceOfRaidName := strings.Split(missedRaid, "/")
								listOfRaids = append(listOfRaids, fmt.Sprintf("(%s) https://fresh.warcraftlogs.com/reports/%s", sliceOfRaidName[0], sliceOfRaidName[1]))
								mapOfRaidsMissed[missedRaid] = true
							}
						}
					}
					if len(listOfRaids) == 0 {
						interactionResponse := NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("mymissedraids|Not 1 single raid missed the last 3 months, pumper %s", crackedBuiltin))
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog("An error occured while trying to sent the respond to user %s using command /mymissedraids, during the function UseSlashCommand()", err.Error())
						}
						break
					}
					responseString := ""
					if len(listOfRaids) > 4 {
						responseString = fmt.Sprintf(
							"You have missed over 4 raids the last 3 months %s, that is 25%% of all the raids in the period.\n\n"+
								"If it's due to vacation, all good — if it's instead due to motivation, please reach out to %s or %s, let's talk.\n\n"+
								"The list of missed raids:\n\n%s",
							antiCrackedBuiltin,
							fmt.Sprintf("<@%s>", SplitOfficerName(officerGMArlissa)["ID"]),
							fmt.Sprintf("<@%s>", SplitOfficerName(officerRogue)["ID"]),
							strings.Join(listOfRaids, "\n"),
						)
						fmt.Println("STRING:", responseString)
					} else {
						responseString = fmt.Sprintf("You have missed %d raids over the last 3 months, please see the list of raids:\n\n%s", len(listOfRaids), strings.Join(listOfRaids, "\n"))
					}
					interactionResponse := NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("mymissedraids|%s", responseString))
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error occured while trying to sent the response to user %s using command /mymissedraids, during the function UseSlashCommand()", err.Error())
					}
				}
			case "mynewmain":
				{
					userInput := interactionData.Options[0].StringValue()
					raiderProfileOld, errString := GetRaiderProfile(userInput)
					if errString != "" {
						interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("mynewmain|The char with name %s is not found in any raids, please contact @Arlissa", userInput))
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog("An error occured while trying to sent the response to user %s using command /mynewmain, during the function UseSlashCommand()", err.Error())
							break
						}
					}
					interactionResponse := NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("mynewmain|Request sent to the officers about linking old main %s with new main %s\n\nYou will recieve a response from the bot, once an officer responds to the request", raiderProfileOld.MainCharName, newRaiderProfile.MainCharName))
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error occured while trying to sent the response to user %s using command /mynewmain, during the function UseSlashCommand()", err.Error())
						break
					}
					_, err = innerSession.ChannelMessageSendComplex(channelOfficer, &discordgo.MessageSend{
						Content: fmt.Sprintf("The raider %s has requested to have his/hers raider attendance added from char %s", newRaiderProfile.MainCharName, raiderProfileOld.MainCharName),
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{
								Components: []discordgo.MessageComponent{
									discordgo.Button{
										Label:    "Yes",
										Style:    discordgo.PrimaryButton,
										CustomID: "button_yes",
									},
									discordgo.Button{
										Label:    "No",
										Style:    discordgo.DangerButton,
										CustomID: "button_no",
									},
								},
							},
						},
					})
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to sent a complex message in the officer channel from the use of command /mynewmain from raider %s, during the function UseSlashCommand()", newRaiderProfile.MainCharName), err.Error())
					}
				}
			case "myraiderperformance":
				{
					raider := raiderProfile{}
					interactionResponse := NewInteractionResponseToSpecificCommand(1, "Calculating performance|", discordgo.InteractionResponseDeferredChannelMessageWithSource)
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error occured while trying to sent initial response to the user %s, using the slash command /myraiderperformance, during the function UseSlashCommand()", err.Error())
						return
					}
					raiderProfiles := ReadWriteRaiderProfiles([]raiderProfile{}, false)
					if len(raiderProfiles) == 0 {
						interactionResponse = NewInteractionResponseToSpecificCommand(1, "No raider-profiles found|Please ask an officer to run command `resetraidcache`")
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog("An error ocurred while trying to send error response to user %s, using slash command /myraiderperformance, during the function UseSlashCommand", err.Error())
						}
						return
					}
					currentRaiderProfileMap := map[string]raiderProfile{}
					for _, raider := range raiderProfiles {
						if raider.ID == userID {
							currentRaiderProfileMap[raider.ID] = raider
						}
					}
					if _, ok := currentRaiderProfileMap[userID]; !ok {
						interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("Raider not found|This is either due to %s not being in 1 main raid yet or a bug. You can verify this by running command `/myattendance`", ResolvePlayerID(userID, innerSession)))
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						WriteErrorLog(fmt.Sprintf("An error ocurred while trying to send error response to user 2 %s , using slash command /myraiderperformance, during the function UseSlashCommand",userID), "User not found")
						return
					}
					timeRaidDataLastUpdated, _ := time.Parse(timeLayoutLogs, currentRaiderProfileMap[userID].RaidData.TimeOfData)
					isDataOld := CheckForLaterThanDuration(timeRaidDataLastUpdated, 7)
					if isDataOld {
						logsLastThreeMonth, err := ReadRaidDataCache(time.Now().AddDate(0, -3, 0), true)
						logsToKeep := []logAllData{}
						if err != nil {
							interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("myraiderperformance|%s", err.Error()))
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							if err != nil {
								WriteErrorLog("An error occured while trying to send an error message to the user, the message is %s, using slash command /myraiderperformance, during the function UseSlashCommand()", err.Error())
							}
							return
						}
						for x, log := range logsLastThreeMonth {
							mapOfPlayers := make(map[string]logPlayer)
							interactionResponse = NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("myraiderperformance|Retrieving Warcraftlogs data - **%.1f%% so far**", float64(x)/float64(len(logsLastThreeMonth))*100)) //interactionResponse := NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("Progess on job|**Completed %.1f%% so far**", float64(x) / float64(len(quriesToRun)) * 100))
							_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Embeds: &interactionResponse.Data.Embeds,
							})
							for _, player := range log.Players {
								mapOfPlayers[player.Name] = player
							}
							raiderName := ResolvePlayerID(userID, innerSession)
							if _, ok := mapOfPlayers[raiderName]; ok {
								logsToKeep = append(logsToKeep, log)
							}
						}
						interactionResponse = NewInteractionResponseToSpecificCommand(1, "myraiderperformance|Calculating your raid-data, please wait...")
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog("An error occured while trying to sent initial response to the user %s, using the slash command /myraiderperformance 2, during the function UseSlashCommand()", err.Error())
							break
						}
						time.Sleep(time.Second * 2)
						messageGIF, err := innerSession.FollowupMessageCreate(event.Interaction, true, NewWebhookParamGIF("the-calculator.gif", yellowColor))
						fmt.Println("WHO IS THIS GUY", currentRaiderProfileMap[userID], userID, len(logsToKeep))
						raider = CalculateRaiderPerformance(currentRaiderProfileMap[userID], logsToKeep)
						if err != nil {
							WriteErrorLog("An error occured while trying to clear last FollowUpMessageCreate using slash command /myraiderperformance, during the function UseSlashCommand()", err.Error())
						}
						err = innerSession.FollowupMessageDelete(event.Interaction, messageGIF.ID)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to delete a follow up message from user %s using slash command /myraiderperformance, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
						}
					} else {
						raider = currentRaiderProfileMap[userID]
					}
					if len(raider.RaidData.LastRaid.Specs) == 0 {
						WriteErrorLog(fmt.Sprintf("The raider %s does not have any calculcated raider performance, please see any earlier error... during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), "Missing data")
						interactionResponse = NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("myraiderperformance|Raidata could not be calculated\nPlease consult %s for help!", SplitOfficerName(officerGMArlissa)["Name"]))
						_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Embeds: &interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error response to user %s using slash command /myraiderperformance", ResolvePlayerID(userID, innerSession)), err.Error())
						}
						return
					}
					//rowBreak := &discordgo.MessageEmbedField{Name: "\u200B", Value: "\u200B", Inline: false}
					statLineBestBoss := &discordgo.MessageEmbedField{}
					statLineWorseBoss := &discordgo.MessageEmbedField{}
					topLine := &discordgo.MessageEmbedField{}
					statLineBestBoss.Inline = true
					statLineWorseBoss.Inline = true
					if raider.RaidData.LastRaid.HealingDone > raider.RaidData.LastRaid.DamageDone {
						statLineBestBoss.Name = "HPS"
						statLineWorseBoss.Name = "HPS"
						statLineBestBoss.Value = fmt.Sprintf("`%.f`", raider.RaidData.Parses.BestBoss.HPS)
						statLineWorseBoss.Value = fmt.Sprintf("`%.f`", raider.RaidData.Parses.WorstBoss.HPS)
					} else {
						statLineBestBoss.Name = "DPS"
						statLineWorseBoss.Name = "DPS"
						statLineBestBoss.Value = fmt.Sprintf("`⚔ %.2f`", raider.RaidData.Parses.BestBoss.DPS)
						statLineWorseBoss.Value = fmt.Sprintf("`⚔ %.2f`", raider.RaidData.Parses.WorstBoss.DPS)
					}

					topString := ""
					switch {
					case raider.RaidData.Parses.Top1:
						{
							topString = "#1 🥇"
						}
					case raider.RaidData.Parses.Top2:
						{
							topString = "#2 🥈"
						}
					case raider.RaidData.Parses.Top3:
						{
							topString = "#3 🥉"
						}
					case raider.RaidData.Parses.Top5:
						{
							topString = "#5"
						}
					default:
						{
							topString = "Around top 5" //Safegaurd
						}
					}
					topLine = &discordgo.MessageEmbedField{Name: "Rank in class", Value: fmt.Sprintf("`%s`", topString), Inline: true}

					fieldsRaiderPerformance := []*discordgo.MessageEmbedField{
						{Name: "**▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒**\r", Value: "\u200B", Inline: false},
						topLine,
						{Name: "From top 1", Value: fmt.Sprintf("`%.f%%`", raider.RaidData.Parses.RelativeToTop), Inline: true},
						{Name: "Δ vs last week", Value: fmt.Sprintf("`%.f%%`", raider.RaidData.Parses.Deviation), Inline: true},
					}
					scalingOrdered := SortFloat64FromMap(false, mapOfPointScaleProcent)
					weightStringSlice := []string{}
					mapOfUniqueWeightNames := make(map[string]bool)
					for _, scaling := range scalingOrdered {
						for weightName, weightProcent := range mapOfPointScaleProcent {
							if scaling == weightProcent && !mapOfUniqueWeightNames[weightName] {
								weightStringSlice = append(weightStringSlice, fmt.Sprintf("↳ %s → `%.f%%`", strings.Split(weightName, "/")[1], weightProcent))
								mapOfUniqueWeightNames[weightName] = true
							}
						}
					}
					specName := ""
					if len(raider.RaidData.LastRaid.Specs) > 0 {
						specName = raider.RaidData.LastRaid.Specs[0].Name
					} else {
						specName = "Not detected..."
					}
					embedRaiderPerformance := &discordgo.MessageEmbed{
						Title: fmt.Sprintf("LINK TO WARCRAFTLOGS PROFILE %s", crackedBuiltin),
						URL:   raider.RaidData.URL,
						Description: strings.Join([]string{
							"**📊 Calculations (last 3 months) 📊**",
							"\u200B",
							fmt.Sprintf("**Raid tier:** `%s`", raider.RaidData.Parses.RaidTier),
							"\u200B",
							fmt.Sprintf("**Raider name:** `%s`", raider.MainCharName),
							"\u200B",
							fmt.Sprintf("**Raider spec:** `%s`", specName),
							"\u200B",
							fmt.Sprintf("**Count of raiders in calculation:** `%d`", raider.RaidData.CountOfRaidersInCalculation),
							"\u200B",
							"**How this works**",
							"• Bot collects data from `all raiders` over the last 3 months",
							"• Only raiders of the `same class` are compared",
							"• Raiders must be present in `≥ 50%` of raids to be included",
							"• Eligible raiders are added to the `compare pool`",
							"• Points are now calculated for each raider in the current `compare pool`",
							"\u200B",
							strings.Join(weightStringSlice, "\n"),
							"↳ Sums to 100% — higher % = higher value in calculation / more points",
							"↳ Metrics are subject to change, and `feedback` can be given using the bot!",
							"\u200B",
							"• The `metrics` defined above is used to calculate the following:",
							"↳ `Rank in class` → In terms of total points, where do you stand?",
							"↳ `From top 1 in %` → How far performance wise, are you from top 1?",
							"↳ `Δ vs last week` → What is your performance difference (delta) compared to the week before?",
						}, "\n"),
						Fields: fieldsRaiderPerformance,
						Footer: &discordgo.MessageEmbedFooter{
							Text: fmt.Sprintf("Source: https://fresh.warcraftlogs.com/guild/id/%d", warcraftLogsGuildID),
						},
					}
					_, err = innerSession.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
						Embeds: []*discordgo.MessageEmbed{embedRaiderPerformance},
						Flags:  discordgo.MessageFlagsEphemeral,
					})
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to sent a result message to user %s using slash command /myraiderperformance, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
					}
					splitBestBossName := strings.Split(raider.RaidData.Parses.BestBoss.Name, " ")
					splitWorstBossName := strings.Split(raider.RaidData.Parses.WorstBoss.Name, " ")
					bestBossNameFinal := ""
					worstBossNameFinal := ""
					if len(splitBestBossName) >= 2 {
						if strings.ToLower(splitBestBossName[1]) == "the" {
							bestBossNameFinal = splitBestBossName[0]
						} else {
							bestBossNameFinal = strings.Join(splitBestBossName[:2], " ")
						}
					} else {
						bestBossNameFinal = raider.RaidData.Parses.BestBoss.Name
					}
					if len(splitWorstBossName) >= 2 {
						if strings.ToLower(splitWorstBossName[1]) == "the" {
							worstBossNameFinal = splitWorstBossName[0]
						} else {
							worstBossNameFinal = strings.Join(splitWorstBossName[:2], " ")
						}
					} else {
						worstBossNameFinal = raider.RaidData.Parses.WorstBoss.Name
					}
					if isDataOld {
						time.Sleep(time.Second * 5)
					}
					fieldsRankings := []*discordgo.MessageEmbedField{
						{Name: "\u200B", Value: "**┌────────────────\u2003Rankings\u2003──────────────┐**", Inline: false},
						{Name: "World", Value: fmt.Sprintf("`#%.f`", raider.RaidData.Parses.RankWorld), Inline: true},
						{Name: "Region", Value: fmt.Sprintf("`#%.f`", raider.RaidData.Parses.RankRegion), Inline: true},
						{Name: "Server", Value: fmt.Sprintf("`#%.f`", raider.RaidData.Parses.RankServer), Inline: true},
						// ── Highlights bosses ──
						{Name: "\u200B", Value: "**┌────────────────\u2003Bosses\u2003────────────────┐**", Inline: false},
						{Name: "Best", Value: fmt.Sprintf("`%s`", bestBossNameFinal), Inline: true},
						statLineBestBoss,
						{Name: "Kill time", Value: fmt.Sprintf("`⏱ %s`", raider.RaidData.Parses.BestBoss.KillTime), Inline: true},
						{Name: "Worst", Value: fmt.Sprintf("`%s`", worstBossNameFinal), Inline: true},
						statLineWorseBoss,
						{Name: "Kill time", Value: fmt.Sprintf("`⏱ %s`", raider.RaidData.Parses.WorstBoss.KillTime), Inline: true},
						{Name: "\u200B", Value: "**┌────────────────\u2003Parses\u2003────────────────┐**", Inline: false},
						{Name: "Best avg", Value: fmt.Sprintf("`%.2f`", raider.RaidData.Parses.Parse["bestAverage"]), Inline: true},
						{Name: "Highest", Value: fmt.Sprintf("`%.2f`", raider.RaidData.Parses.Parse["highest"]), Inline: true},
						{Name: "Lowest", Value: fmt.Sprintf("`%.2f`", raider.RaidData.Parses.Parse["lowest"]), Inline: true},
					}

					embedRankings := &discordgo.MessageEmbed{
						Title: fmt.Sprintf("LINK TO WARCRAFTLOGS PROFILE %s", crackedBuiltin),
						URL:   raider.RaidData.URL,
						Description: strings.Join([]string{
							"**🏆 Warcraftlogs rankings and bosses 🏆**",
							"\u200B",
							"**• Please see the following sections:**",
							"\u200B",
							"↳ Where do you place in the world?",
							"↳ Which boss is best, which is worst?",
							"↳ What does your raw parses look like?",
						}, "\n"),
						Fields: fieldsRankings,
						Footer: &discordgo.MessageEmbedFooter{
							Text: fmt.Sprintf("Source: https://fresh.warcraftlogs.com/guild/id/%d", warcraftLogsGuildID),
						},
					}
					_, err = innerSession.FollowupMessageCreate(event.Interaction, true, &discordgo.WebhookParams{
						Embeds: []*discordgo.MessageEmbed{embedRankings},
						Flags:  discordgo.MessageFlagsEphemeral,
					})
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to sent ranking data to user %s, using slash command /myraiderperformance, during the function UseSlashCommand()", ResolvePlayerID(userID, innerSession)), err.Error())
					}
				}
			case "myreminder":
				{
					var err error
					raiderName := ResolvePlayerID(userID, innerSession)
					timesToNotify := 5
					waitBeforeDeleteChannel := time.Minute * 2
					title := interactionData.Options[0].Value.(string)
					err = innerSession.InteractionRespond(event.Interaction, slashCommandAllUsers["myreminder"].Responses["examples"].Response)
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to sent initial response to user %s using slash command /myreminder, during the function UseSlashCommand()", raiderName), err.Error())
						return
					}
					_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
						Embeds: &slashCommandAllUsers["myreminder"].Responses["examples"].Response.Data.Embeds,
					})
					if err != nil {
						WriteErrorLog("An error occured while trying to sent examples response using slash command /myreminder, during the function UseSlashCommand()", err.Error())
						return
					}
					timeNoneFiltered := strings.ToLower(strings.TrimSpace(interactionData.Options[1].Value.(string)))
					duration := time.Duration(0)
					hitError := false
					patternDigitalClock := regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d:[0-5]\d$`)
					patternCountdown := regexp.MustCompile(`^(?:\d+h(?:\d+m(?:\d+s)?)?|\d+m(?:\d+s)?|\d+s)$`)
					if patternDigitalClock.MatchString(timeNoneFiltered) {
						timePartsSlice := strings.Split(timeNoneFiltered, ":")
						if len(timePartsSlice) != 3 {
							WriteErrorLog("The time provided is invalid, value of %s cannot be used in slash command /myreminder, during the function UseSlashCommand()", err.Error())
							hitError = true
						}
						hours, err := strconv.Atoi(timePartsSlice[0])
						if err != nil {
							WriteErrorLog("An eror occured while trying to perform string to int convert on value %s using the slash command /myreminder 1, during the function UseSlashCommand()", err.Error())
							hitError = true
						}

						minutes, err := strconv.Atoi(timePartsSlice[1])
						if err != nil {
							WriteErrorLog("An eror occured while trying to perform string to int convert on value %s using the slash command /myreminder 2, during the function UseSlashCommand()", err.Error())
							hitError = true
						}
						seconds, err := strconv.Atoi(timePartsSlice[2])
						if err != nil {
							WriteErrorLog("An eror occured while trying to perform string to int convert on value %s using the slash command /myreminder 3, during the function UseSlashCommand()", err.Error())
							hitError = true
						}
						if hitError {
							interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("myreminder|An internal bot error occured, please contact <@%s>", SplitOfficerName(officerGMArlissa)["ID"]))
							err = innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog("An error occured while trying to send an error message to the user %s using the slash command /myreminder, during the function UseSlashCommand()", err.Error())
								return
							}
						}
						duration = time.Hour*time.Duration(hours) + time.Minute*time.Duration(minutes) + time.Second*time.Duration(seconds)
					} else if patternCountdown.MatchString(timeNoneFiltered) {
						duration, err = time.ParseDuration(timeNoneFiltered)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to convert time string %s to time.Duration using slash command /myreminder, during the function UseSlashCommand()", timeNoneFiltered), err.Error())
							interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("myreminder|An internal bot error occured, please contact <@%s>", SplitOfficerName(officerGMArlissa)["ID"]))
							err = innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog("An error occured while trying to send error message to user %s using the slash command /myreminder, during the function UseSlashCommand()", err.Error())
								return
							}
						}
					} else {
						interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("myreminder|Your time of `%s` is not valid. Please see the examples for help.", timeNoneFiltered))
						_, err := innerSession.FollowupMessageCreate(event.Interaction, true, &discordgo.WebhookParams{
							Flags: discordgo.MessageFlagsEphemeral,
							Embeds: interactionResponse.Data.Embeds,
						})
						if err != nil {
							WriteErrorLog("An error occured while trying to send initial response to user %s using the slash command /myreminder 2, during the function UseSlashCommand()", err.Error())
						}
						return
					}
					convertTime := time.Now().Local().Add(duration).Format(timeLayoutLogs)
					interactionResponse := NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("Alert for %s|**The bot will attempt to contact you at: %s**", title, convertTime))
					_, err = innerSession.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
						Flags:  discordgo.MessageFlagsEphemeral,
						Embeds: interactionResponse.Data.Embeds,
					})
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to notify the user %s for last message using the slash command /myreminder, during the function UseSlashCommand()", raiderName), err.Error())
						return
					}
					WriteInformationLog(fmt.Sprintf("User %s has requested a thread to sleep for %s due to a userdefined alert being set using the slash command /myreminder, during the function UseSlashCommand()", raiderName, duration.String()), "Sleeping thread")
					time.Sleep(duration)
					userChannel, err := session.UserChannelCreate(userID)
					failed := false
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured when trying to create a DM channel to the user %s", raiderName), err.Error())
						failed = true
					}
					if !failed {
						for x := range timesToNotify {
							x++
							_, err := innerSession.ChannelMessageSend(userChannel.ID, fmt.Sprintf("**%s** REMINDER!\n\nMESSAGE WILL REPEAT %d/%d more times!", title, x, timesToNotify))
							if err != nil {
								failed = true
								break
							}
							time.Sleep(time.Second * 3)
						}
					}

					if failed {
						newChannel, err := innerSession.GuildChannelCreateComplex(serverID, discordgo.GuildChannelCreateData{
							Name:  fmt.Sprintf("alert-%s-%s", event.ID, duration.String()),
							Type:  discordgo.ChannelTypeGuildText,
							Topic: "This channel will close in 2min",
							PermissionOverwrites: []*discordgo.PermissionOverwrite{
								{
									ID:   serverID,
									Type: discordgo.PermissionOverwriteTypeMember,
									Deny: permissionViewChannel,
								},
								{
									ID:    userID,
									Type:  discordgo.PermissionOverwriteTypeMember,
									Allow: permissionViewChannel,
								},
							},
						})
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to create channel: %s for user %s using slash command /myreminder, during the function UseSlashCommand()", newChannel.Name, raiderName), err.Error())
							return
						}
						for x := range timesToNotify {
							x++
							_, err = innerSession.ChannelMessageSend(newChannel.ID, fmt.Sprintf("Hi <@%s> THIS IS YOUR REMINDER FOR: %s\n\nMESSAGE WILL REPEAT %d/%d more times!", userID, title, x, timesToNotify))
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to sent a message in new channel created to alert user %s about a user specified alert, using the slash command /myreminder during the function UseSlashCommand()", raiderName), err.Error())
								return
							}
							time.Sleep(time.Second * 3)
						}
						WriteInformationLog(fmt.Sprintf("Waiting 2min before deleting the channel that the bot had to create, because it could not create or sent a direct channel to the user %s, using slash command /myreminder during the function UseSlashCommand()", raiderName), "Sleeping thread")
						time.Sleep(waitBeforeDeleteChannel)
						_, err = innerSession.ChannelDelete(newChannel.ID)
						if err != nil {
							WriteErrorLog("An error occured while trying to delete channel with name %s and ID %s, using slash command /myreminder, during the function UseSlashCommand()", err.Error())
						}
					}
				}
			case "howto":
				{
					interactionResponse := NewInteractionResponseToSpecificCommand(3, "Information about how to use the bot|Please select one of the buttons below:")
					interactionResponse.Data.Components = []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "general",
									Style:    discordgo.PrimaryButton,
									CustomID: "general",
								},
								discordgo.Button{
									Label:    "your raiding stats",
									Style:    discordgo.PrimaryButton,
									CustomID: "stats",
								},
							},
						},
					}
					innerSession.InteractionRespond(event.Interaction, &interactionResponse)
				}
			case "hi":
				{
					responseString := ""
					playerID := event.Member.User.ID
					interactionResponse := NewInteractionResponseToSpecificCommand(3, "Saying hi|", discordgo.InteractionResponseDeferredChannelMessageWithSource)
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error occured while trying to sent error response to user %s with slash command hi 1, during the function UseSlashCommand()", err.Error())
					}
					playerName := ResolvePlayerID(playerID, innerSession)
					if CheckForRaiderRank(playerID, innerSession) {
						responseString = fmt.Sprintf("Hi %s you damn pumper %s - How are you?\n\nUse **`/howto`** OR make %s do a joke with **`/joke`**", playerName, crackedBuiltin, crackedBuiltin)
					} else {
						responseString = fmt.Sprintf("Hi %s - Good to meet you!", playerName)
					}
					_, err = innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
						Content: GetStringPointer(responseString),
					})
					if err != nil {
						WriteErrorLog("An error occured while trying to sent error response to user %s with slash command hi, during the function UseSlashCommand()", err.Error())
						break
					}
					WriteInformationLog(fmt.Sprintf("The user %s has said hi to the bot", playerName), "Success interaction")
				}
			case "joke":
				{
					interactionResponse := NewInteractionResponseToSpecificCommand(3, "Incomming joke|", discordgo.InteractionResponseDeferredChannelMessageWithSource)
					interactionResponse.Data.Flags &^= discordgo.MessageFlagsEphemeral
					innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					jokeData := GetHttpResponseData("GET", "", "https://v2.jokeapi.dev/joke/Programming,Miscellaneous,Dark,Pun,Spooky,Christmas?type=twopart", nil, false)
					if data, ok := jokeData.(map[string]any); !ok {
						_, err := innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
							Content: GetStringPointer(fmt.Sprintf("The service does not respond, please contact <@%s>", SplitOfficerName(officerGMArlissa)["ID"])),
						})
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent error response to user %s with slash command jokes 2, during the function UseSlashCommand()", userID), err.Error())
							break
						}
					} else {
						if flags, ok := data["flags"].(map[string]any); ok {
							if racistFlag, ok := flags["racist"]; ok && racistFlag.(bool) {
								_, err := innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
									Content: GetStringPointer(fmt.Sprintf("Wuuuuupsi..... I cannot tell this joke as it would get me in trouble! %s", antiCrackedBuiltin)),
								})
								if err != nil {
									WriteErrorLog("An error occured while trying to sent error response to user %s with slash command jokes, during the function UseSlashCommand()", err.Error())
								}
								break
							} else if darkFlag, ok := data["category"].(string); ok && darkFlag == "Dark" {
								_, err := innerSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("This one is going to be spicy... I apologize in advance <@%s> %s", userID, antiCrackedBuiltin))
								if err != nil {
									WriteErrorLog("An error occured while trying to sent a warning about a dark joke in the channel %s, using slash command /joke, during the function UseSlashCommand()", err.Error())
								}
								//fmt.Sprintf("This one is going to be spicy... I apologize in advance <@%s> %s", userID, antiCrackedBuiltin
							}
						} else {
							WriteErrorLog(fmt.Sprintf("No data found under the joke API payload flags - object: %s, with slash command jokes, during the function UseSlashCommand()"), "Missing data for flags payload using the joke API - Flags MUST be present to check for racist content")
						}
						returnStringSlice := []string{}
						if value, ok := data["setup"].(string); ok && len(value) != 0 {
							_, err := innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Content: GetStringPointer(value),
							})
							if err != nil {
								WriteErrorLog("An error occured while trying to sent error response to user %s with slash command jokes, during the function UseSlashCommand()", err.Error())
								break
							}
							returnStringSlice = append(returnStringSlice, value)
							time.Sleep(5 * time.Second)
						}
						if value, ok := data["delivery"].(string); ok && len(value) != 0 {
							returnStringSlice = append(returnStringSlice, value)
							_, err := innerSession.InteractionResponseEdit(event.Interaction, &discordgo.WebhookEdit{
								Content: GetStringPointer(strings.Join(returnStringSlice, "\n\n")),
							})
							if err != nil {
								WriteErrorLog("An error occured while trying to sent error response to user %s with slash command jokes, during the function UseSlashCommand()", err.Error())
								break
							}
						}
					}
				}
			case "feedback":
				{
					optionsCount := len(interactionData.Options)
					if optionsCount < 2 {
						interactionResponse := NewInteractionResponseToSpecificCommand(0, "feedback|There is an issue with the feedback service at this time")
						err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
						if err != nil {
							WriteErrorLog(fmt.Sprintf("The length of the options provided for slash command /feedback is %d, which is supposed to be 2 or more", optionsCount), "Error in template / slash-command")
							break
						}
					}
					deepCopy := *slashCommandAllUsers["feedback"].Responses["description"].Response
					dataCopy := *deepCopy.Data
					deepCopy.Data = &dataCopy
					deepCopy.Data.CustomID = fmt.Sprintf("%s/%s/%s/%v", deepCopy.Data.CustomID, interactionData.Options[0].StringValue(), userID, interactionData.Options[1].BoolValue())
					err := innerSession.InteractionRespond(event.Interaction, &deepCopy)
					if err != nil {
						fmt.Println("ID:", deepCopy.Data.CustomID)
						WriteErrorLog("An error occured while trying to respond to slash command /feedback and the users welcome message, during the function UshSlashCommand()", err.Error())
					}
				}
			case "aboutme":
				{
					fmt.Println("WE DONT REACH HERE?")
					interactionResponse := NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("aboutme|This feature is not out yet 🚧 please contact <@%s> for more information", SplitOfficerName(officerGMArlissa)["ID"]))
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error occured while trying to reply to user %s using slash command /aboutme, during function UseSlashCommand()", err.Error())
					}
				}
			}
		}
	})
}

func CheckForPost(title string, channelID ...string) (bool, string) {
	messagesCount := 100
	id := channelInfo
	if len(channelID) > 0 {
		id = channelID[0]
	}
	messages, err := BotSessionMain.ChannelMessages(id, messagesCount, "", "", "")
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve the last %d messages from channel ID %s, returning early, during the function CheckForPost()", messagesCount, id), err.Error())
		return false, ""
	}
	for _, message := range messages {
		if len(message.Embeds) == 0 {
			continue
		}

		if strings.EqualFold(message.Embeds[0].Title, title) {
			return true, message.ID
		}
	}
	return false, ""
}


func CheckForLaterThanDuration(timeToCheck time.Time, days int) bool {
	return time.Since(timeToCheck) > time.Duration(days)*24*time.Hour
}

func NewWebhookParamGIF(fileName string, color ...int) *discordgo.WebhookParams {
	fileGIF, err := os.Open(fileName)
	currentColor := 0
	if len(color) == 0 {
		currentColor = blueColor
	} else {
		currentColor = color[0]
	}
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to open GIF file using the slash command /%s, during the function UseSlashCommand()", fileName), err.Error())
	}
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Image: &discordgo.MessageEmbedImage{
					URL: fmt.Sprintf("attachment://%s", fileName),
				},
				Color: currentColor,
			},
		},
		Files: []*discordgo.File{
			{
				Reader: fileGIF,
				Name:   fileName,
			},
		},
		Flags: discordgo.MessageFlagsEphemeral,
	}
}

func cloneBenchReasonResponse(in applicationResponse) applicationResponse {
	// Start by shallow-copying the applicationResponse
	out := in

	if in.Response == nil || in.Response.Data == nil {
		return out
	}

	// Copy InteractionResponse
	resp := *in.Response

	// Copy InteractionResponseData
	data := *in.Response.Data

	// Deep copy Components: []discordgo.MessageComponent
	if len(data.Components) > 0 {
		data.Components = make([]discordgo.MessageComponent, len(in.Response.Data.Components))

		for i, comp := range in.Response.Data.Components {
			// We know this is discordgo.ActionsRow in your case
			ar, ok := comp.(discordgo.ActionsRow)
			if !ok {
				// If something unexpected appears, just carry it over shallowly
				data.Components[i] = comp
				continue
			}

			newRow := discordgo.ActionsRow{}

			if len(ar.Components) > 0 {
				newRow.Components = make([]discordgo.MessageComponent, len(ar.Components))

				for j, inner := range ar.Components {
					// In your code this is *discordgo.TextInput
					if ti, ok := inner.(*discordgo.TextInput); ok && ti != nil {
						tiCopy := *ti
						newRow.Components[j] = &tiCopy
					} else {
						// Fallback shallow copy for unknown types
						newRow.Components[j] = inner
					}
				}
			}

			data.Components[i] = newRow
		}
	}

	resp.Data = &data
	out.Response = &resp

	return out
}

func FormatRaiderID(raiderName string) string {
	return strings.ReplaceAll(strings.ReplaceAll(raiderName, ">", ""), "<@", "")
}

func GetRaiderProfiles() []raiderProfile {
	if raiders := ReadWriteRaiderProfiles(nil, false); raiders != nil {
		return raiders
	}
	return []raiderProfile{}
}

func GetRaiderProfile(raiderName string) (raiderProfile, string) {
	raiderName = FormatRaiderID(raiderName)
	cachedRaiders := ReadWriteRaiderProfiles(nil, false)
	match := false
	raiderProfile := raiderProfile{}
	for _, raider := range cachedRaiders {
		if raiderName == raider.ID {
			match = true
			raiderProfile = raider
		} else if raiderName == raider.MainCharName {
			match = true
			raiderProfile = raider
		}
	}
	if !match {
		return raiderProfile, fmt.Sprintf("No raider profile found for %s in cache", raiderName)
	}
	return raiderProfile, ""
}

func ReadWriteConfig(currentConfig ...config) config {
	configCacheMutex.Lock()
	defer configCacheMutex.Unlock()
	configCache := config{}
	updated := false
	var err error
	if bytes := CheckForExistingCache(configPath); len(bytes) != 0 {
		err = json.Unmarshal(bytes, &configCache)
		if err != nil {
			WriteErrorLog(fmt.Sprintf("The config retrieved from path %s could not be unmarshalled, returning early, during the function ReadWriteConfig()", configPath), err.Error())
			return configCurrent
		}
	}

	if len(currentConfig) == 0 {
		if configCache.ServerID == "" {
			return configCurrent
		} else {
			return configCache
		}
	}

	if configCache.Announce == nil {
		configCache.Announce = make(map[string]topic)
	}
	configIn := currentConfig[0]
	if configIn.ServerID != configCache.ServerID {
		WriteInformationLog(fmt.Sprintf("Server ID has changed from %s to %s which is unusual unless the bot is being run from a new server, during the function ReadWriteConfig()", configCache.ServerID, configIn.ServerID), "ServerID changed")
		updated = true
		configCache.ServerID = configIn.ServerID
	}

	if configIn.ChannelID != configCache.ChannelID {
		WriteInformationLog(fmt.Sprintf("Channel ID has changed, which means the channel has been recreated, from ID %s to ID %s", configCache.ChannelID, configIn.ChannelID), "Channel recreated")
		updated = true
		configCache.ChannelID = configIn.ChannelID
	}

	if configIn.WarcraftLogsGuildID != configCache.WarcraftLogsGuildID {
		WriteInformationLog(fmt.Sprintf("WarcraftlogsGuildID has changed from %s to %s which means the guild that the bot is tracking has changed, during the function ReadWriteConfig()", configCache.WarcraftLogsGuildID, configIn.WarcraftLogsGuildID), "WarcrftlogsGuildID changed")
		updated = true
		configCache.WarcraftLogsGuildID = configIn.WarcraftLogsGuildID
	}

	if configIn.WarcraftLogsAppID != configCache.WarcraftLogsAppID {
		WriteInformationLog(fmt.Sprintf("The application used to talk to warcraft logs has been changed from ID %s to ID %s, make sure it has the correct permissions in Warcraftlogs... During the function ReadWriteConfig()", configCache.WarcraftLogsAppID, configIn.WarcraftLogsAppID), "WarcraftlogsAppID changed")
		updated = true
		configCache.WarcraftLogsAppID = configIn.WarcraftLogsAppID
	}

	if configIn.DiscordAppID != configCache.DiscordAppID && configCache.DiscordAppID != "" {
		WriteErrorLog(fmt.Sprintf("The discord app ID cannot change but was attempted. Must have ID %s but tried to parse ID %s, during the function ReadWriteConfig()", configCache.DiscordAppID, configIn.DiscordAppID), "Cannot change ID")
	}

	for nameThread, topic := range configIn.Announce {
		if _, ok := configCache.Announce[nameThread]; !ok {
			configCache.Announce[nameThread] = topic
			updated = true
			WriteInformationLog(fmt.Sprintf("Thread topic of name %s was not found in cache and will therefor be added. New length of threads %d, during the function ReadWriteConfig()", nameThread, len(configCache.Announce)), "Added announcement")
		}
	}

	marshal, err := json.MarshalIndent(configCache, "", " ")
	err = os.WriteFile(configPath, marshal, 0644)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to write the config cache on path %s, during the function ReadWriteConfig()", configPath), err.Error())
		return configCache
	}
	if updated {
		WriteInformationLog(fmt.Sprintf("Config found at %s has been successfully updated at %s, during the function ReadWriteConfig()", configPath, GetTimeString()), "Updating config file")
	}
	return configCache
}

func ReadWriteTrackPosts(post ...trackPost) map[string]trackPost {
	postTrackMutex.Lock()
	defer postTrackMutex.Unlock()
	returnMapOfTrackPosts := make(map[string]trackPost)
	if bytes := CheckForExistingCache(cacheTrackedPostsCache); len(bytes) > 0 {
			err := json.Unmarshal(bytes, &returnMapOfTrackPosts)
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to unmarshal bytes of len %d on path %s, during the function ReadWriteTrackPosts()", len(bytes), cacheTrackedPostsCache), err.Error())
			}
	}
	if len(post) == 0 {
		return returnMapOfTrackPosts
	}
	returnMapOfTrackPosts[post[0].MessageID] = post[0]
	marshal, err := json.MarshalIndent(returnMapOfTrackPosts, "", " ")
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to marshal json of returnMapOfTrackPosts, Message ID of new message to save: %s, len of total map %d, cache on path %s cannot be updated, during the function ReadWriteTrackPosts()", post[0].MessageID, len(returnMapOfTrackPosts), cacheTrackedPostsCache), err.Error())
		return returnMapOfTrackPosts
	}
	err = os.WriteFile(cacheTrackedPostsCache, marshal, 0644)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to write cache to the file of returnMapOfTrackPosts, Message ID of new message to save: %s, len of total map %d, cache on path %s cannot be updated, during the function ReadWriteTrackPosts()", post[0].MessageID, len(returnMapOfTrackPosts), cacheTrackedPostsCache), err.Error())
	}
	if len(returnMapOfTrackPosts) == 0 {
		WriteInformationLog(fmt.Sprintf("The cache objects found on path %s, this means no channel has ever been tracked yet, during the function ReadWriteTrackPosts()", cacheTrackedPostsCache), "No cache found")
	}
	return returnMapOfTrackPosts
}

func ReadWriteRaiderProfiles(raiders []raiderProfile, initial bool) []raiderProfile {
	if initial && raiders != nil { //Will overwrite any existing file as part of initial run
		WriteInformationLog("WARNING - Reinstating raiderProfile cache, during the function ReadWriteRaiderProfiles()", "Resetting Cache")
		marshal, err := json.MarshalIndent(raiders, "", " ")
		if err != nil {
			WriteErrorLog("An error occured while trying to marshal go-struct for []raiderProfile, during the function ReadWriteRaiderProfiles()", err.Error())
			return nil
		}
		os.WriteFile(raiderProfilesCachePath, marshal, 0644)
		return nil
	}

	bytes := CheckForExistingCache(raiderProfilesCachePath)
	if len(bytes) == 0 {
		WriteErrorLog(fmt.Sprintf("The current length of cache found for %s is 0, please reinstate all logdata by running /resetraidcache", raiderProfilesCachePath), "Cache is nil, must reinstate")
		return nil
	}
	cachedRaiderProfiles := raiderProfiles{}
	json.Unmarshal(bytes, &cachedRaiderProfiles)
	if raiders == nil {
		return cachedRaiderProfiles.Raiders
	}
	mapOfMissingProfiles := make(map[string]bool)
	allMissingProfiles := []raiderProfile{}
	for _, raider := range raiders {
		mapOfMissingProfiles[raider.MainCharName] = false
		for x, cachedRaider := range cachedRaiderProfiles.Raiders {
			if raider.MainCharName == cachedRaider.MainCharName {
				mapOfMissingProfiles[raider.MainCharName] = true
				//fmt.Println("DO WE GET HERE?", "RAIDER NAME", raider.MainCharName, "CACHED:", cachedRaider.AttendanceInfo["oneMonth"].RaidCount, cachedRaider.AttendanceInfo["twoMonth"].RaidCount, cachedRaider.AttendanceInfo["threeMonth"].RaidCount, cachedRaider.AttendanceInfo["guildStart"].RaidCount, "NEW",  raider.AttendanceInfo["oneMonth"].RaidCount, raider.AttendanceInfo["twoMonth"].RaidCount, raider.AttendanceInfo["threeMonth"].RaidCount, raider.AttendanceInfo["guildStart"].RaidCount,)
				if cachedRaider.AttendanceInfo["oneMonth"].RaidCount != raider.AttendanceInfo["oneMonth"].RaidCount || cachedRaider.AttendanceInfo["guildStart"].RaidCount != raider.AttendanceInfo["guildStart"].RaidCount || len(cachedRaider.AttendanceInfo["oneMonth"].RaidsMissed) != len(raider.AttendanceInfo["oneMonth"].RaidsMissed) || len(cachedRaider.AttendanceInfo["twoMonth"].RaidsMissed) != len(raider.AttendanceInfo["twoMonth"].RaidsMissed) || len(cachedRaider.AttendanceInfo["threeMonth"].RaidsMissed) != len(raider.AttendanceInfo["threeMonth"].RaidsMissed) {
					WriteInformationLog("The raider: %s 's attendance will be updated, during the function ReadWriteRaiderCache()", "Updating RaiderProfile Attendance")
					cachedRaiderProfiles.Raiders[x].AttendanceInfo = raider.AttendanceInfo
				}

				if cachedRaider.IsOfficer != raider.IsOfficer {
					cachedRaiderProfiles.Raiders[x].IsOfficer = raider.IsOfficer
				}

				if len(raider.BenchInfo["lastWeek"]) > 0 {
					var err error
					mapOfTakenBenches := make(map[string]bool) //We dont want 1 bench event being shared accross multiple periods
					if cachedRaider.BenchInfo == nil {
						cachedRaiderProfiles.Raiders[x].BenchInfo = make(map[string][]bench)
					}
					if len(cachedRaiderProfiles.Raiders[x].BenchInfo) == 0 {
						cachedRaiderProfiles.Raiders[x].BenchInfo["lastWeek"] = raider.BenchInfo["lastWeek"]
						mapOfTakenBenches[raider.BenchInfo["lastWeek"][0].DateString] = true
					}
					timeBefore, err := time.Parse(timeLayOutShort, cachedRaiderProfiles.Raiders[x].BenchInfo["lastWeek"][0].DateString)
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to update raider: %s weekly bench info, during the function ReadWriteRaiderProfiles()", raider.MainCharName), err.Error())
						continue
					}
					timeAfter, err := time.Parse(timeLayOutShort, raider.BenchInfo["lastWeek"][0].DateString)
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to update raider: %s weekly bench info 2, during the function ReadWriteRaiderProfiles()", raider.MainCharName), err.Error())
						continue
					}
					fmt.Println("TIME BNEFORE_:", timeBefore.String(), "TIME AFTER:", timeAfter.String())
					if timeAfter.After(timeBefore) || timeAfter.Equal(timeBefore) {
						fmt.Println("DO WE GET HERE??33")
						timeNow := time.Now()
						oneMonthBack := timeNow.AddDate(0, 0, -31)
						twoMonthBack := timeNow.AddDate(0, 0, -60)
						threeMonthBack := timeNow.AddDate(0, 0, -90)
						oneMonthBenches := []bench{}
						twoMonthBenches := []bench{}
						threeMonthBenches := []bench{}
						startBenches := []bench{}

						if _, ok := raider.BenchInfo["oneMonth"]; !ok {
							cachedRaiderProfiles.Raiders[x].BenchInfo["oneMonth"] = []bench{}
						}
						if _, ok := raider.BenchInfo["twoMonth"]; !ok {
							cachedRaiderProfiles.Raiders[x].BenchInfo["twoMonth"] = []bench{}
						}
						if _, ok := raider.BenchInfo["threeMonth"]; !ok {
							cachedRaiderProfiles.Raiders[x].BenchInfo["threeMonth"] = []bench{}
						}
						for _, benches := range raider.BenchInfo {
							if len(benches) == 0 {
								continue
							}
							for _, bench := range benches {
								raidTime, err := time.Parse(timeLayOutShort, bench.DateString)
								if err != nil {
									WriteErrorLog(fmt.Sprintf("An error occured while trying to parse time string %s with layout %s, during the function ReadWriteRaiderProfiles()", bench.DateString, timeLayOutShort), err.Error())
									continue
								}
								switch {
								case raidTime.After(oneMonthBack) && !mapOfTakenBenches[bench.DateString]:
									mapOfTakenBenches[bench.DateString] = true
									// 8–30 days ago
									oneMonthBenches = append(oneMonthBenches, bench)

								case raidTime.After(twoMonthBack) && !mapOfTakenBenches[bench.DateString]:
									// 30–60 days ago
									mapOfTakenBenches[bench.DateString] = true
									twoMonthBenches = append(twoMonthBenches, bench)

								case raidTime.After(threeMonthBack) && !mapOfTakenBenches[bench.DateString]:
									// 60–90 days ago
									mapOfTakenBenches[bench.DateString] = true
									threeMonthBenches = append(threeMonthBenches, bench)

								default:
									// older than 90 days
								}
							}

							if len(oneMonthBenches) > 0 {
								cachedRaiderProfiles.Raiders[x].BenchInfo["oneMonth"] = oneMonthBenches
								WriteInformationLog(fmt.Sprintf("The raider %s has had his one month bench information updated, during the function ReadWriteRaiderProfiles()", raider.MainCharName), "Updating bench info")
								continue
							}

							if len(twoMonthBenches) > 0 {
								cachedRaiderProfiles.Raiders[x].BenchInfo["twoMonth"] = twoMonthBenches
								WriteInformationLog(fmt.Sprintf("The raider %s has had his two month bench information updated, during the function ReadWriteRaiderProfiles()", raider.MainCharName), "Updating bench info")
								continue
							}

							if len(threeMonthBenches) > 0 {
								cachedRaiderProfiles.Raiders[x].BenchInfo["threeMonth"] = threeMonthBenches
								WriteInformationLog(fmt.Sprintf("The raider %s has had his three month bench information updated, during the function ReadWriteRaiderProfiles()", raider.MainCharName), "Updating bench info")
								continue
							}

							if len(startBenches) > 0 {
								cachedRaiderProfiles.Raiders[x].BenchInfo["start"] = startBenches
								WriteInformationLog(fmt.Sprintf("The raider %s has had his all time bench information updated, during the function ReadWriteRaiderProfiles()", raider.MainCharName), "Updating bench info")
							}
							fmt.Println("THIS IS THE BENCH INFO", cachedRaiderProfiles.Raiders[x].BenchInfo)
						}
					}
				}

				if cachedRaider.ID == "" {
					cachedRaiderProfiles.Raiders[x].ID = raider.ID
				}

				if cachedRaider.DateJoinedGuild == "" || cachedRaider.DateJoinedGuild != raider.DateJoinedGuild {
					fmt.Println("MATCHED", raider.MainCharName)
					cachedRaiderProfiles.Raiders[x].DateJoinedGuild = raider.DateJoinedGuild
				}

				if cachedRaider.Username == "" {
					cachedRaiderProfiles.Raiders[x].Username = raider.Username
				}

				if cachedRaider.MainSwitch == nil {
					cachedRaiderProfiles.Raiders[x].MainSwitch = raider.MainSwitch
				} else if raider.MainSwitch != nil {
					mapOfMissingMainSwitch := make(map[string]bool)
					for raiderSwitchName := range raider.MainSwitch {
						for innerRaiderSwitchName := range cachedRaider.MainSwitch {
							mapOfMissingMainSwitch[raiderSwitchName] = false
							if raiderSwitchName == innerRaiderSwitchName {
								mapOfMissingMainSwitch[raiderSwitchName] = true
							}
						}
					}
					for missingMainSwitch := range mapOfMissingMainSwitch {
						cachedRaiderProfiles.Raiders[x].MainSwitch[missingMainSwitch] = true
					}
				}

				missingDiscordRoles := []string{}
				for _, discordRole := range raider.DiscordRoles {
					if !strings.Contains(strings.Join(cachedRaider.DiscordRoles, ","), discordRole) {
						missingDiscordRoles = append(missingDiscordRoles, discordRole)
					}
				}
				cachedRaiderProfiles.Raiders[x].DiscordRoles = append(cachedRaiderProfiles.Raiders[x].DiscordRoles, missingDiscordRoles...)

				if cachedRaider.GuildRole != raider.GuildRole {
					cachedRaiderProfiles.Raiders[x].GuildRole = raider.GuildRole
				}
				checkForNilTime := ""
				if len(cachedRaider.RaidData.TimeOfData) == 0 {
					checkForNilTime = time.Now().Format(timeLayoutLogs)
				} else {
					checkForNilTime = cachedRaider.RaidData.TimeOfData
				}
				timeCacheData, err := time.Parse(timeLayoutLogs, checkForNilTime)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to parse time 2 %s using layout %s, during the function ReadWriteRaiderProfiles()", cachedRaider.RaidData.TimeOfData, timeLayoutLogs), err.Error())
					break
				}
				timeNewData, err := time.Parse(timeLayoutLogs, raider.RaidData.TimeOfData)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to parse time 3 %s using layout %s, during the function ReadWriteRaiderProfiles()", raider.RaidData.TimeOfData, timeLayoutLogs), err.Error())
					break
				}
				if !timeCacheData.Equal(timeNewData) {
					WriteInformationLog(fmt.Sprintf("Changes to object logRaid detected - old cache time %s and new cache time %s, during the function ReadWriteReaiderProfiles()", cachedRaider.RaidData.TimeOfData, raider.RaidData.TimeOfData), "Updating raider profile")
					cachedRaiderProfiles.Raiders[x].RaidData = raider.RaidData
				}
				break
			}
		}
		if !mapOfMissingProfiles[raider.MainCharName] {
			WriteInformationLog("Raider Profile with name %s was not found in cache and will be added, during the function ReadWriteRaiderProfiles()", "Adding Raider Profile to cache")
			allMissingProfiles = append(allMissingProfiles, raider)
		}
	}
	cachedRaiderProfiles.Raiders = append(cachedRaiderProfiles.Raiders, allMissingProfiles...)
	cachedRaiderProfiles.GuildName = guildName
	//cachedRaiderProfiles.CountOfLogs = 0
	marshal, err := json.MarshalIndent(cachedRaiderProfiles, "", " ")
	if err != nil {
		WriteErrorLog("An error occured while trying to write to the RaiderProfiles cache, during the function ReadWriteRaiderProfiles()", err.Error())
		return nil
	}
	os.WriteFile(raiderProfilesCachePath, marshal, 0644)
	return cachedRaiderProfiles.Raiders
}

func NewRaidProfileAttendanceSummary(raider raiderProfile) string {
	return fmt.Sprintf("```md\n[ Hardened Member ]\n\n/ Raider Name: %s\n/ Joined Guild At: %s\n\n/ Period (Numbers Change Weekly) / Value              \n/--------------------------------/---------------------/\n/ Total Raids Done               / %d                 \n/ Last Month Attendance          / %.0f%%              \n/ Last 2 Months Attendance       / %.0f%%              \n/ Last 3 Months Attendance       / %.0f%%              \n/ Since Guild Started (OG's)     / %.0f%%              \n```",
		raider.MainCharName,
		strings.Join(strings.Split(raider.DateJoinedGuild, " ")[:len(strings.Split(raider.DateJoinedGuild, " "))-1], " "),
		raider.AttendanceInfo["guildStart"].RaidCount,
		raider.AttendanceInfo["oneMonth"].RaidProcent,
		raider.AttendanceInfo["twoMonth"].RaidProcent,
		raider.AttendanceInfo["threeMonth"].RaidProcent,
		raider.AttendanceInfo["guildStart"].RaidProcent)
}

func NewRaidProfileBenchSummaries(raiders []raiderProfile, periodKey string) string {
	lastWeek := false
	var returnStringWriter strings.Builder
	if periodKey == "lastWeek" {
		lastWeek = true
	}
	oneRaider := false

	mapOfBenchedRaiders := make(map[string]map[string][]bench)
	for _, raider := range raiders {
		if len(raider.BenchInfo) == 0 {
			continue
		}
		for period, benched := range raider.BenchInfo {
			fmt.Println("RAIDER NAME:", raider.MainCharName, "LEN OF BENCH", len(benched))
			if period == periodKey {
				if mapOfBenchedRaiders[period] == nil {
					mapOfBenchedRaiders[period] = make(map[string][]bench)
				}
				if len(benched) > 0 {
					mapOfBenchedRaiders[period][raider.MainCharName] = benched
					fmt.Println("THIS IS THE BENCH:", mapOfBenchedRaiders[period][raider.MainCharName])
				}
			}

		}
	}

	returnStringWriter.WriteString(
		fmt.Sprintf(
			"```md\n[ Hardened Bench Overview ]\n\nPlayers benched during period: %d\n\nPeriod: %s\n\n",
			len(mapOfBenchedRaiders[periodKey]),
			periodKey,
		),
	)

	headerLabel := "Reason"
	if !lastWeek && len(mapOfBenchedRaiders[periodKey]) != 1 {
		headerLabel = "Count"
	}

	if len(raiders) == 1 || len(mapOfBenchedRaiders[periodKey]) == 1 {
		returnStringWriter.WriteString(fmt.Sprintf(
			"Player Name (Since only 1 found): %s\n\n",
			raiders[0].MainCharName,
		))
		oneRaider = true
	}

	// header
	returnStringWriter.WriteString(
		fmt.Sprintf("/ %-20s / %-25s /\n", "Player Name", headerLabel),
	)
	returnStringWriter.WriteString(
		"/----------------------/---------------------------/\n",
	)

	// rows
	for playerName, benchSlice := range mapOfBenchedRaiders[periodKey] {
		if oneRaider {
			for _, bench := range benchSlice {
				if len(benchSlice) == 1 {
					returnStringWriter.WriteString(
						fmt.Sprintf("/ %-20s / %-25s /\n", playerName, bench.Reason),
					)
					break
				}
				returnStringWriter.WriteString(
					fmt.Sprintf("/ %-20s / %-25s /\n", bench.DateString, bench.Reason),
				)
			}
			break
		}
		if lastWeek {
			returnStringWriter.WriteString(
				fmt.Sprintf("/ %-20s / %-25s /\n", playerName, benchSlice[0].Reason),
			)
		} else {
			returnStringWriter.WriteString(
				fmt.Sprintf("/ %-20s / %-25d /\n", playerName, len(benchSlice)),
			)
		}
	}

	returnStringWriter.WriteString("```")
	return returnStringWriter.String()
}

func ResolvePlayerID(playerID string, innerSession *discordgo.Session) string {
	returnNickName := "" //Will be username if nickname is ""
	user, err := innerSession.GuildMember(serverID, playerID)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve the guild member: %s", playerID), err.Error())
		return ""
	}
	if user.Nick == "" {
		returnNickName = user.User.Username
	} else {
		returnNickName = user.Nick
	}
	return returnNickName
}

func ResolvePlayerName(playerName string, session *discordgo.Session) string {
	returnID := ""
	users, err := session.GuildMembers(serverID, "", 1000)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrive all discord users on server: %s, during the function ResolvePlayerName()", serverID), err.Error())
		return ""
	}
	for _, user := range users {
		if user.Nick == playerName || user.User.Username == playerName {
			returnID = user.User.ID
		}
	}
	return returnID
}

func ResolveRoleIDs(session *discordgo.Session, roleIDs ...string) []string {
	allRoles, err := session.GuildRoles(serverID)
	returnStringSlice := []string{}
	fmt.Println("LEN OF ALL ROLES", len(allRoles))
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve all Discord roles on server %s, during the function ResolveRoleIDs", serverID), err.Error())
		return returnStringSlice
	}
	mapOfAllRoles := make(map[string]string, len(allRoles))
	fmt.Println("ROLE ID222:", roleIDs)
	for _, role := range allRoles {
		mapOfAllRoles[role.ID] = role.Name
		//fmt.Println("role ID", role.ID, "role name", role.Name, )
	}
	for _, roleID := range roleIDs {
		if role, ok := mapOfAllRoles[roleID]; ok {
			returnStringSlice = append(returnStringSlice, role)
		} else {
			fmt.Println("NOT OK!", roleID, "ROLE IDS", roleIDs, len(roleIDs), len(mapOfAllRoles))
		}
	}
	fmt.Println("RETURN SLICE:", returnStringSlice)
	return returnStringSlice
}

func ChunkEmbeds(embeds []*discordgo.MessageEmbed, maxBytes int) [][]*discordgo.MessageEmbed {
	var sliceOfSliceTotalMessageEmbeds [][]*discordgo.MessageEmbed
	var sliceOfCurrentMessageEmbed []*discordgo.MessageEmbed
	var currentSize int

	for _, embed := range embeds {
		b, _ := json.Marshal(embed)
		if currentSize+len(b) >= maxBytes {
			sliceOfSliceTotalMessageEmbeds = append(sliceOfSliceTotalMessageEmbeds, sliceOfCurrentMessageEmbed)
			sliceOfCurrentMessageEmbed = []*discordgo.MessageEmbed{}
			currentSize = 0
		}
		sliceOfCurrentMessageEmbed = append(sliceOfCurrentMessageEmbed, embed)
		currentSize += len(b)
	}
	if len(sliceOfCurrentMessageEmbed) > 0 {
		sliceOfSliceTotalMessageEmbeds = append(sliceOfSliceTotalMessageEmbeds, sliceOfCurrentMessageEmbed)
	}
	return sliceOfSliceTotalMessageEmbeds
}

func SortRaidsInSpecificMaps(raids []logAllData) (map[string][]logAllData, []string) {
	mapOfRaids := make(map[string][]logAllData)
	sliceOfRaidNames := []string{}
	mapOfRaidNames := make(map[string]bool)
	for _, logData := range raids {
		if logData.RaidNames == nil {
			WriteErrorLog("No raid names found in log with title %s - The raid names are crucial for this function and therefor the log will be skipped... Continuing", "During function SortRaidsInSpecificMaps()")
			continue
		}
		if !mapOfRaidNames[logData.RaidNames[0]] {
			sliceOfRaidNames = append(sliceOfRaidNames, logData.RaidNames[0])
			mapOfRaidNames[logData.RaidNames[0]] = true
		}
		mapOfRaids[logData.RaidNames[0]] = append(mapOfRaids[logData.RaidNames[0]], logData)
	}
	return mapOfRaids, sliceOfRaidNames
}

/*
	func MergeRaidDataForUserOutput(allLogDataSlice []logAllData) []*discordgo.MessageEmbed {
		mapOfDifferentRaidTypes := make(map[string][]logAllData)
		returnEmbedsOfMergedRaids := []*discordgo.MessageEmbed{}
		for _, logData := range allLogDataSlice {
			nameOfRaid := ""
			if raidTitleSplit := strings.Split(logData.RaidTitle, " "); len(raidTitleSplit) > 1 {
				nameOfRaid = strings.ToLower(raidTitleSplit[0])
			} else if raidTitleSplit := strings.Split(logData.RaidTitle, "_"); len(raidTitleSplit) > 1 {
				nameOfRaid = strings.ToLower(raidTitleSplit[0])
			}
			mapOfDifferentRaidTypes[nameOfRaid] = append(mapOfDifferentRaidTypes[nameOfRaid], logData)
		}

		for raidName, allLogs := range mapOfDifferentRaidTypes {
			allFields := []*discordgo.MessageEmbedField{}
			for _, allLog := range allLogs {

			}

			messageEmbed := &discordgo.MessageEmbed{
				Title: fmt.Sprintf("Overall summary of raid with name %s", raidName),
				Color: greenColor,
			}
		}
	}
*/
func DeepCopyInteractionResponse(original *discordgo.InteractionResponse) *discordgo.InteractionResponse {
	var copy discordgo.InteractionResponse
	b, _ := json.Marshal(original)
	json.Unmarshal(b, &copy)
	return &copy
}

func InitializeDiscordProfiles(raiders []raiderProfile, innerSession *discordgo.Session, onlyRaiders bool) []raiderProfile {
	discordMembers, err := innerSession.GuildMembers(serverID, "", 1000)
	if err != nil {
		WriteErrorLog("An error occured while trying to retrieve all discord members during the function InitializeDiscordProfiles()", err.Error())
		return nil
	}

	allRaiders := []raiderProfile{}

	for x, raider := range raiders {
		for _, discordMember := range discordMembers {
			if raider.MainCharName == discordMember.Nick {
				raiders[x].DiscordRoles = discordMember.Roles
				if strings.Contains(strings.Join(raiders[x].DiscordRoles, ","), roleRaider) {
					raiders[x].GuildRole.RoleID = roleRaider
					raiders[x].GuildRole.RoleName = ResolveRoleIDs(innerSession, roleRaider)[0]

				} else if strings.Contains(strings.Join(raiders[x].DiscordRoles, ","), roleTrial) && raiders[x].GuildRole.RoleID != roleRaider {
					raiders[x].GuildRole.RoleID = roleTrial
					raiders[x].GuildRole.RoleName = ResolveRoleIDs(innerSession, roleTrial)[0]
				} else {
					raiders[x].GuildRole.RoleID = rolePuggie
					raiders[x].GuildRole.RoleName = ResolveRoleIDs(innerSession, rolePuggie)[0]
				}

				if strings.Contains(strings.Join(raiders[x].DiscordRoles, ","), roleOfficer) {
					raiders[x].IsOfficer = true
				}

				raiders[x].Username = discordMember.User.Username
				raiders[x].ID = discordMember.User.ID
				raiders[x].LastTimeChangedString = GetTimeString()
			}
		}
	}

	if onlyRaiders {
		for _, raider := range raiders {
			if raider.GuildRole.RoleID == roleRaider || raider.GuildRole.RoleID == roleTrial || raider.GuildRole.RoleID != rolePuggie {
				allRaiders = append(allRaiders, raider)
			} else {
				WriteInformationLog(fmt.Sprintf("Skipping discord member %s due to flag onlyRaiders is true, during the function InitializeDiscordProfiles()", raider.MainCharName), "Skipping Discord user")
			}
		}
	} else {
		allRaiders = raiders
	}
	ReadWriteRaiderProfiles(allRaiders, false)
	WriteInformationLog(fmt.Sprintf("Initialized %d number of raiders - Some may be inactive...", len(allRaiders)), "Successfully initialized raiderProfiles")
	return allRaiders
}

func AddWeeklyRaiderAttendance(botInfo ...any) string {
	currentRaiders := raiderProfiles{}
	currentRaids := []logAllData{}
	raiderCacheBytes := CheckForExistingCache(raiderProfilesCachePath)
	if raiderCacheBytes == nil {
		WriteErrorLog("The raider-profiles cache is nil, which causes this function to fail, during the function AddWeeklyReaiderAttendance", "Raider-profiles cache is nil")
		return "The raider-profiles cache is nil"
	}
	logsCacheBytes := CheckForExistingCache(raidAllDataPath)
	if logsCacheBytes == nil {
		WriteErrorLog("The raid cache is nil, which causes this function to fail, during the function AddWeeklyReaiderAttendance", "Raid cache is nil")
		return "The raid cache is nil"
	}
	err := json.Unmarshal(raiderCacheBytes, &currentRaiders)
	if err != nil {
		WriteErrorLog("An error occured while trying to unmarshal the raider cache, during the function AddWeeklyRaiderAttendance()", err.Error())
		return fmt.Sprintf("An error occured while trying to unmarshal the raider cache - %s", err.Error())
	}
	err = json.Unmarshal(logsCacheBytes, &currentRaids)
	if err != nil {
		WriteErrorLog("An error occured while trying to unmarshal the raid cache, during the function AddWeeklyRaiderAttendance()", err.Error())
		return fmt.Sprintf("An error occured while trying to unmarshal the raid cache - %s", err.Error())
	}

	newRaiders := CalculateAttendance(currentRaiders.Raiders, currentRaids, botInfo...)

	ReadWriteRaiderProfiles(newRaiders, false)
	return fmt.Sprintf("A total of %d raider-profiles has had attendance updated", len(newRaiders)) //len(updatedRaiderProfiles))
}

func SortOnlyMainRaids(timePeriod time.Duration, raids []logAllData, mergedRaids bool) []logAllData {
	mainRaidFilter := "ony,aq20,zg"
	timeConverted := time.Time{}
	filteredRaids := []logAllData{}
	raidFilterName := "+"
	if timePeriod > 0 {
		timeConverted = time.Now().Add(timePeriod)
	} else {
		timeConverted = raids[len(raids)-1].MetaData.startTime
	}

	for _, raid := range raids {
		if mergedRaids {
			if strings.Contains(raid.RaidTitle, raidFilterName) && !strings.Contains(mainRaidFilter, raid.RaidNames[0]) && (timeConverted.After(raid.MetaData.startTime) || timeConverted.Equal(raid.MetaData.startTime)) {
				filteredRaids = append(filteredRaids, raid)
			}
		} else {
			if !strings.Contains(raid.RaidTitle, raidFilterName) && !strings.Contains(mainRaidFilter, raid.RaidNames[0]) && (timeConverted.After(raid.MetaData.startTime) || timeConverted.Equal(raid.MetaData.startTime)) {
				filteredRaids = append(filteredRaids, raid)
			}
		}
	}

	return nil
}

func IsRaiderTank(raider logPlayer) bool {
	for _, abillity := range raider.Abilities {
		if slices.Contains(tankAbillities, abillity.Name) && abillity.TotalCasts > 4 {
			return true
		}
	}
	return false
}

func CalculateRaiderPerformance(raider raiderProfile, raids []logAllData) raiderProfile { //This function expects raids to ONLY be ones, where the raider parsed is part of, otherwise these calculations wont make sense
	currentRaider := raider
	currentRaider.RaidData.AverageRaid = make(map[string]logPlayer)
	playersInScope := make(map[string]bool) //Identify players relevant given the parsed raiderProfile
	playersOfSameClass := make(map[string]bool)
	raiderProfiles := GetRaiderProfiles()
	currentRaiderProfiles := []raiderProfile{}
	returnRaiderProfile := raiderProfile{}

	isCurrentTank := IsRaiderTank(currentRaider.RaidData.AverageRaid["lastWeek"])

	for _, log := range raids {
		for _, player := range log.Players {
			if player.ClassName == currentRaider.ClassInfo.IngameClass { //We expect the tank role to be set first, in case the raider has more than 1 spec
				playersOfSameClass[player.Name] = true
			}
		}
	}

	for raider := range playersOfSameClass {
		for _, raiderProfile := range raiderProfiles {
			if raiderProfile.MainCharName == raider && slices.Contains(raiderProfile.DiscordRoles, roleRaider) {
				if IsRaiderTank(raiderProfile.RaidData.AverageRaid["lastWeek"]) && !isCurrentTank {
					WriteInformationLog(fmt.Sprintf("The player %s has been skipped due to the raider asking for performance is NOT a tank, but the player in scope, is, during the function CalculateRaiderPerformance()", raider), "Player skipped")
					continue
				}
				playersInScope[raider] = true
				currentRaiderProfiles = append(currentRaiderProfiles, raiderProfile)
				continue
			}
		}
	}
	fmt.Println("len of raider profiles", len(currentRaiderProfiles), playersInScope)
	logsToAnalyze := []logAllData{}
	for _, log := range raids {
		mapOfUniquePlayers := make(map[string]bool)
		for _, player := range log.Players {
			for _, raider := range currentRaiderProfiles {
				if raider.MainCharName == player.Name && !mapOfUniquePlayers[player.Name] {
					mapOfUniquePlayers[player.Name] = true
				}
			}
		}
		if len(mapOfUniquePlayers) == 0 {
			WriteInformationLog("None of the current raiders %s were seen in the log %s, log will be skipped", "Skipping log")
			continue
		}
		containsProcentPlayers := float64(len(mapOfUniquePlayers)) / float64(len(currentRaiderProfiles)) * 100
		fmt.Println("LEN OF UNIQUE PLAYERS", len(mapOfUniquePlayers), len(currentRaiderProfiles), containsProcentPlayers)
		if containsProcentPlayers >= 50 {
			logsToAnalyze = append(logsToAnalyze, log)
			fmt.Println("THE LOG:", log.RaidTitle, "HAS ALL RAIDERS IN", len(currentRaiderProfiles), len(mapOfUniquePlayers))
		}
	}
	switch {
	case len(logsToAnalyze) < 3:
		{
			WriteInformationLog(fmt.Sprintf("The amount of logs found for raider %s is %d and therefor less than 4 - Will return early, during the function CalculateRaiderPerformance()", raider.MainCharName, len(logsToAnalyze)), "Returning early")
			return (raiderProfile{})
		}
	}
	mapOfUniquePlayerLogs := make(map[string][]logPlayer)
	for _, logToAnalyze := range logsToAnalyze {
		for _, player := range logToAnalyze.Players {
			if _, ok := playersInScope[player.Name]; !ok {
				continue
			}
			mapOfUniquePlayerLogs[player.Name] = append(mapOfUniquePlayerLogs[player.Name], player)
		}
	}

	if _, ok := mapOfUniquePlayerLogs[raider.MainCharName]; !ok {
		WriteInformationLog(fmt.Sprintf("The raider %s was not found in map of unique player logs - Will return early, during the function CalculateRaiderPerformance()", raider.MainCharName), "Returning early")
		return (raiderProfile{})
	}

	countOfPlayersInCalculation := 0
	x := 0
	statType := "" //DPS - Healer, Tank
	healingRatio := float64(mapOfUniquePlayerLogs[currentRaider.MainCharName][0].HealingDone) / float64(mapOfUniquePlayerLogs[currentRaider.MainCharName][0].DamageDone)
	if healingRatio > 1 {
		statType = "Healer"
	} else if currentRaider.ClassInfo.ClassType != "Tank" {
		statType = "dps"
	}
	mapOfDeathCount := make(map[string]int)
	mapOfStatCount := make(map[string]int64)
	mapOfCPMCount := make(map[string]float64)
	mapOfParseHighAverage := make(map[string]float64)
	mapOfParseLowAverage := make(map[string]float64)
	mapOfDeathAverageCount := make(map[string]float64)
	mapOfStatAverageCount := make(map[string]float64)
	mapOfCPMAverageCount := make(map[string]float64)
	x = 0
	for raiderName, raids := range mapOfUniquePlayerLogs {
		x++
		countOfPlayersInCalculation++
		for _, raid := range raids {
			switch statType {
			case "Healer":
				{
					mapOfStatCount[raiderName] += raid.HealingDone
				}
			default:
				{
					mapOfStatCount[raiderName] += raid.DamageDone
				}
			}
			mapOfDeathCount[raiderName] += len(raid.Deaths)
			mapOfCPMCount[raiderName] += raid.MinuteAPM
		}

		if len(mapOfUniquePlayerLogs) == x {
			for raiderName, count := range mapOfCPMCount {
				mapOfCPMAverageCount[raiderName] = float64(count) / float64(len(raids))
			}

			for raiderName, count := range mapOfDeathCount {
				mapOfDeathAverageCount[raiderName] = float64(count) / float64(len(raids))
			}

			for raiderName, count := range mapOfStatCount {
				mapOfStatAverageCount[raiderName] = float64(count) / float64(len(raids))
			}
		}
	}

	for _, raider := range currentRaiderProfiles {
		mapOfParseHighAverage[raider.MainCharName] = raider.RaidData.Parses.Parse["bestAverage"]
		mapOfParseLowAverage[raider.MainCharName] = raider.RaidData.Parses.Parse["mediumAverage"]
	}

	sortedParseHighAverage := SortFloat64FromMap(false, mapOfParseHighAverage)
	sortedParseLowAverage := SortFloat64FromMap(false, mapOfParseLowAverage)
	sortedCPMAverage := SortFloat64FromMap(false, mapOfCPMAverageCount)
	sortedDeathAverage := SortFloat64FromMap(true, mapOfDeathAverageCount)
	sortedStatAverage := SortFloat64FromMap(false, mapOfStatAverageCount)
	mapOfCurrentRaiderPoints := make(map[string]int)
	for x, raider := range currentRaiderProfiles {
		mapOfName := make(map[string]string)
		mapOfName["name"] = raider.MainCharName
		mapOfWarcraftLogsQuery := SetWarcraftLogQueryVariables(mapOfWarcaftLogsQueries["playerRankings"], mapOfName)
		if len(mapOfWarcraftLogsQuery) == 0 {
			WriteErrorLog(fmt.Sprintf("The Warcraftlogs query with key playerRankins is empty, cannot look for parse for raider %s", raider.MainCharName), "Warcraftlog query is nil")
			continue
		}
		playerLogs := mapOfUniquePlayerLogs[raider.MainCharName]

		fmt.Println("PLAYER LOGS FOR:", raider.MainCharName)
		for _, log := range playerLogs {
			fmt.Println("NAME:", log.Name)
		}
		if len(playerLogs) == 0 {
			WriteErrorLog(fmt.Sprintf("No playerLog data found for raider %s, this is an issue because this data is required to continue these calculations, during the function CalculateRaiderPerformance()", raider.MainCharName), "Missing data")
			continue
		}
		mapOfData := GetWarcraftLogsData(mapOfWarcraftLogsQuery[0])
		mapOfCurrentRaiderPoints[raider.MainCharName] = raider.RaidData.Parses.Points
		currentRaiderProfiles[x].RaidData = UnwrapWarcraftLogRaiderRanking(mapOfData, raider, playerLogs[0])
	}
	raiders := []raiderProfile{}
	for _, raider := range currentRaiderProfiles {
		if len(raider.RaidData.LastRaid.Specs) != 0 {
			raiders = append(raiders, raider)
		}
	}
	pointsSorted := []float64{}
	pointsPlayers := make(map[string]float64)
	for x, raider := range raiders {
		averageCPM := mapOfCPMAverageCount[raider.MainCharName]
		for y, cpm := range sortedCPMAverage {
			if averageCPM == cpm {
				if y == 0 {
					pointsPlayers[raider.MainCharName] += 2*float64(mapOfPointScale["pointScaleAPM/APM"]) + 1
				} else if y <= 2 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleAPM/APM"]) + 1
				} else if y <= 4 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleAPM/APM"]) + 0.5
				}
			}
		}
		averageStat := mapOfStatAverageCount[raider.MainCharName]
		for y, stat := range sortedStatAverage {
			if averageStat == stat {
				if y == 0 {
					pointsPlayers[raider.MainCharName] += 2*float64(mapOfPointScale["pointScaleStat/DPS & HPS"]) + 1
				} else if y <= 2 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleStat/DPS & HPS"]) + 1
				} else if y <= 4 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleStat/DPS & HPS"]) + 0.5
				}
			}
		}

		averageDeath := mapOfDeathAverageCount[raider.MainCharName]
		for y, death := range sortedDeathAverage {
			if averageDeath == death {
				if y == 0 {
					pointsPlayers[raider.MainCharName] += 2*float64(mapOfPointScale["pointScaleDeath/Death rate"]) + 1
				} else if y <= 2 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleDeath/Death rate"]) + 1
				} else if y <= 4 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleDeath/Death rate"]) + 0.5
				}
			}
		}

		averageHighParse := mapOfParseHighAverage[raider.MainCharName]
		for y, parse := range sortedParseHighAverage {
			if averageHighParse == parse {
				if y == 0 {
					pointsPlayers[raider.MainCharName] += 2*float64(mapOfPointScale["pointScaleParse/Parse low & high"]) + 1
				} else if y <= 2 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleParse/Parse low & high"]) + 1
				} else if y <= 4 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleParse/Parse low & high"]) + 0.5
				}
			}
		}

		averageLowParse := mapOfParseHighAverage[raider.MainCharName]
		for y, parse := range sortedParseLowAverage {
			if averageLowParse == parse {
				if y == 0 {
					pointsPlayers[raider.MainCharName] += 2*float64(mapOfPointScale["pointScaleParse/Parse low & high"]) + 1
				} else if y <= 2 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleParse/Parse low & high"]) + 1
				} else if y <= 4 {
					pointsPlayers[raider.MainCharName] += float64(mapOfPointScale["pointScaleParse/Parse low & high"]) + 0.5
				}
			}
		}

		if x == len(raiders)-1 {
			pointsSorted = SortFloat64FromMap(false, pointsPlayers)
		}
	}

	fmt.Println("POINTS PLAYERS:", pointsPlayers, "SORTED:", pointsSorted)
	maxPoints := 0.0
	for x, raider := range raiders {
		for raiderName, points := range pointsPlayers {
			for y, innerPoints := range pointsSorted {
				if innerPoints == points && raiderName == raider.MainCharName {
					switch {
					case y == 0:
						{
							maxPoints = innerPoints
							raiders[x].RaidData.Parses.Top1 = true
						}
					case y == 1:
						{
							raiders[x].RaidData.Parses.Top2 = true
						}
					case y >= 2:
						{
							raiders[x].RaidData.Parses.Top3 = true
						}
					case y <= 4:
						{
							raiders[x].RaidData.Parses.Top5 = true
						}
					}
					break
				}
			}
		}
	}

	for x, raider := range raiders {
		//fmt.Println("RAIDER:", raider.MainCharName, "OLD", mapOfCurrentRaiderPoints[raider.MainCharName], "NEW", pointsPlayers[raider.MainCharName])
		if mapOfCurrentRaiderPoints[raider.MainCharName] != 0 && pointsPlayers[raider.MainCharName] != 0 {
			//fmt.Println("DO WE REACH HERE?2132131")
			raiders[x].RaidData.Parses.Deviation = math.Round((pointsPlayers[raider.MainCharName] - float64(mapOfCurrentRaiderPoints[raider.MainCharName])) / float64(mapOfCurrentRaiderPoints[raider.MainCharName]) * 100)
			if !raider.RaidData.Parses.Top1 {
				raiders[x].RaidData.Parses.RelativeToTop = math.Round((pointsPlayers[raider.MainCharName] - maxPoints) / maxPoints * 100)
			}
		}
		raiders[x].RaidData.Parses.Points = int(math.Round(pointsPlayers[raider.MainCharName]))

	}

	for x, raider := range raiders {
		if raider.MainCharName == currentRaider.MainCharName {
			raiders[x].RaidData.CountOfRaidersInCalculation = len(mapOfUniquePlayerLogs)
			returnRaiderProfile = raiders[x]
		}
	}
	ReadWriteRaiderProfiles(raiders, false)

	/*
		mostDoneAbility := 0
		mapOfMostDoneAbility := make(map[string]int)
		//abilityNameToTrack := ""
		//typeToTrack := ""
		for _, raid := range playersOfSameClass[raider.MainCharName] { //Create template values and specific aggregated values for raider that was parsed
			abilityName := ""
			for x, ability := range raid.Abilities {
				if ability.TotalCasts > mostDoneAbility {
					mostDoneAbility = ability.TotalCasts
					abilityName = ability.Name
				}
				if x == len(raid.Abilities) -1 {
					mapOfMostDoneAbility[abilityName]++
				}
			}
			if raid.HealingDone > raid.DamageDone {
				//typeToTrack = "healing"
			}
		}

		biggestCount := 0
		x = 0
		for _, count := range mapOfMostDoneAbility {
			if count > biggestCount {
				biggestCount = count
			}
			x++
			if x == len(mapOfMostDoneAbility) -1 {
				//abilityNameToTrack = abilityName
			}
		}
		/*
		deathsToTrack := logPlayerDeath{} //Will contain the aggregated values from the entire period
		mostDoneStat := 0

		mapOfAveragePerformance := make(map[string]logPlayer)
		for raiderName, raidLogs := range playersOfSameClass {
			var totalCPM int64
			totalDeaths := map[string][]logPlayerDeath{}
			totalMostDoneStat := 0 //Will prioritize whether u did most dps, healed the most or took the most dmg (relative to urself only)
			mostUsedAbility := logPlayerAbility{}
			totalCasts := 0
			for _, playerLog := range raidLogs {
				totalCPM += playerLog.ActiveTimeMS

				for _, death := range playerLog.Deaths {
					if !death.PartOfWipe {
						totalDeaths[raiderName] = append(totalDeaths[raiderName], death)
					}
				}
				for _, ability := range playerLog.Abilities {
					if ability.Name == abilityNameToTrack {
						totalCasts += ability.TotalCasts
					}
				}

			}
		}
	*/
	return returnRaiderProfile
}

func SortFloat64FromMap[K comparable](ascending bool, data map[K]float64) []float64 {
	values := make([]float64, 0, len(data))
	for _, value := range data {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if ascending {
			return values[i] < values[j]
		}
		return values[i] > values[j]
	})
	return values
}

func CalculateAttendance(raiders []raiderProfile, raids []logAllData, botInfo ...any) []raiderProfile {
	mapOfAttendancePeriods := map[string]time.Time{
		"oneMonth":   time.Now().AddDate(0, -1, 0),
		"twoMonth":   time.Now().AddDate(0, -2, 0),
		"threeMonth": time.Now().AddDate(0, -3, 0),
		"guildStart": GuildStartTime,
	}
	innerSession := &discordgo.Session{}
	event := &discordgo.Interaction{}
	doStatusCount := 0
	doStatus := false
	if len(botInfo) > 1 {
		if session, ok := botInfo[0].(*discordgo.Session); ok {
			innerSession = session
			doStatusCount++
		}

		if interaction, ok := botInfo[1].(*discordgo.Interaction); ok {
			event = interaction
			doStatusCount++
		}
		if doStatusCount == 2 {
			doStatus = true
		}
	}

	mapOfMainRaidPeriods := make(map[string][]logAllData)
	mapOfPeriodsAndRaidsTotal := make(map[string]int)
	for periodName, raidTime := range mapOfAttendancePeriods {
		mapOfUniqueRaids := make(map[string]bool)
		mainRaidsInPeriod := []logAllData{}
		altRaidsInPeriod := []logAllData{}
		for _, raid := range raids {
			if len(raid.RaidNames) == 0 {
				continue
			}
			raidCurrentTime := time.UnixMilli(raid.RaidStartUnixTime)
			raidTitleSlice := strings.Split(raid.RaidTitle, " ")
			timeParsed, _ := time.Parse(timeLayout, raid.RaidStartTimeString)
			raidKey := timeParsed.Format(timeLayout)
			if raidCurrentTime.After(raidTime) && !mapOfUniqueRaids[raidKey] && !strings.Contains("zg,ony,aq20", strings.ToLower(raidTitleSlice[0])) {
				mainRaidsInPeriod = append(mainRaidsInPeriod, raid)
				mapOfUniqueRaids[raidKey] = true
			} else if raidCurrentTime.After(raidTime) && !mapOfUniqueRaids[raidKey] {
				altRaidsInPeriod = append(altRaidsInPeriod, raid)
				mapOfUniqueRaids[raidKey] = true
			} else {
			}
		}
		mapOfPeriodsAndRaidsTotal[periodName] = len(mainRaidsInPeriod)
		mapOfMainRaidPeriods[periodName] = mainRaidsInPeriod
	}
	mapOfMainRaidersAttendance := make(map[string]map[string]int)
	mapOfMissingRaids := make(map[string]map[string][]string)
	for _, raider := range raiders {
		mainSwitchSlice := []string{}
		if raider.MainSwitch != nil {
			for mainSwitchName := range raider.MainSwitch {
				mainSwitchSlice = append(mainSwitchSlice, mainSwitchName)
			}
		}
		raiderAttendance := make(map[string]int)
		for period, logs := range mapOfMainRaidPeriods {
			for _, log := range logs {
				mapOfUniquePlayers := make(map[string]bool)
				playerSeen := false
				for x, player := range log.Players {
					if player.Name == raider.MainCharName || strings.Contains(strings.Join(mainSwitchSlice, ","), player.Name) {
						raiderAttendance[period]++
						playerSeen = true
					}
					if !playerSeen && x == len(log.Players)-1 && !mapOfUniquePlayers[raider.MainCharName] {
						if _, ok := mapOfMissingRaids[raider.MainCharName]; !ok {
							mapOfMissingRaids[raider.MainCharName] = make(map[string][]string)
						}
						mapOfMissingRaids[raider.MainCharName][period] = append(mapOfMissingRaids[raider.MainCharName][period], fmt.Sprintf("%s/%s", log.RaidTitle, log.MetaData.Code))
					}
				}
			}
			mapOfMainRaidersAttendance[raider.MainCharName] = raiderAttendance
		}
	}

	for raiderName, mapOfCount := range mapOfMainRaidersAttendance {
		mapOfAttendance := make(map[string]attendance)
		for period, count := range mapOfCount {
			amountOfRaids := mapOfPeriodsAndRaidsTotal[period]
			attendance := attendance{
				RaidCount:   count,
				RaidProcent: math.Floor(float64(count) / float64(amountOfRaids) * 100),
				MainRaid:    true,
				RaidsMissed: mapOfMissingRaids[raiderName][period],
			}
			mapOfAttendance[period] = attendance
		}
		for x, raiderProfile := range raiders {
			if raiderProfile.MainCharName == raiderName {
				raiders[x].AttendanceInfo = mapOfAttendance
			}
		}
	}

	mapOfRaiders := make(map[string]bool)
	for x, raider := range raiders {
		match := false
		for y := len(raids) - 1; y >= 0; y-- {
			for _, player := range raids[y].Players {
				if raider.MainCharName == player.Name && !mapOfRaiders[player.Name] {
					raiders[x].DateJoinedGuild = raids[y].RaidStartTimeString
					match = true
					mapOfRaiders[player.Name] = true
					break
				}
			}
			if match {
				break
			}
		}
	}
	for x, raider := range raiders {
		if doStatus {
			interactionResponse := NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("Progess on job|**Completed %.1f%% so far**", float64(x)/float64(len(raiders))*100))
			_, err := innerSession.InteractionResponseEdit(event, &discordgo.WebhookEdit{
				Embeds: &interactionResponse.Data.Embeds,
			})
			if err != nil {
				WriteErrorLog("An error occured while trying to call for new progress status during the function CalculateAttendance()", err.Error())
			}
		}
		mapOfRaidsToKeep := make(map[string]bool)
		raidsToKeep := []string{}
		joinedGuild, err := time.Parse(timeLayout, raider.DateJoinedGuild)
		if err != nil {
			WriteErrorLog("An error occured while trying to convert string %s to time, during the function CalculateAttenance()", err.Error())
			continue
		}
		for period, attendance := range raider.AttendanceInfo {
			raidsToRemoveCounter := 0
			if period == "guildStart" {
				continue
			}
			for _, raidMissed := range attendance.RaidsMissed {
				raidTime, err := ConvertMissedRaidToTime(raidMissed)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to convert string to %s to time 2, during the function CalculateAttendance()", raider.MainCharName), err.Error())
					continue
				}
				if raidTime.Before(joinedGuild) {
					WriteInformationLog(fmt.Sprintf("The raider %s was not in guild when this raid was missed => %s, will not be used in any calculations, during the function CalculateAttendance()", raider.MainCharName, raidMissed), "Recalculating raider attendance")
					raidsToRemoveCounter++
				} else if raidTime.After(joinedGuild) && !mapOfRaidsToKeep[raidMissed] {
					raidsToKeep = append(raidsToKeep, raidMissed)
					mapOfRaidsToKeep[raidMissed] = true
				}
			}

			if raidsToRemoveCounter > 0 {
				allRaidCount := float64(attendance.RaidCount) * 100.0 / attendance.RaidProcent
				allRaidCountInt := int(allRaidCount)
				currentAttendance := attendance
				currentAttendance.RaidsMissed = raidsToKeep
				currentAttendance.RaidProcent = math.Floor(float64(currentAttendance.RaidCount+raidsToRemoveCounter) / float64(allRaidCountInt) * 100)
				raiders[x].AttendanceInfo[period] = currentAttendance
			}
		}
	}

	return raiders
}

func ConvertMissedRaidToTime(raidTitle string) (time.Time, error) {
	raidTimeWithTitle := strings.Split(raidTitle, "/")[0]
	raidTimeTitleSlice := strings.Split(raidTimeWithTitle, " ")
	raidTime, err := time.Parse(timeLayoutLogs, raidTimeTitleSlice[len(raidTimeTitleSlice)-1]+" 15:00:00")
	if err != nil {
		WriteErrorLog("An error occured while trying to translate raid title %s to time.Time, during the funcion ConvertMissedRaidToTime()", err.Error())
	}
	return raidTime, err
}

func InitializeRaiderProfiles() []raiderProfile {
	mapOfPlayers := make(map[string]bool)
	newRaiderProfiles := []raiderProfile{}
	allRaidLogs := []logAllData{}
	if data := CheckForExistingCache(raidAllDataPath); len(data) == 0 {
		allRaidLogs = GetAllWarcraftLogsRaidData(false, false, "")
	} else {
		err := json.Unmarshal(data, &allRaidLogs)
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to marshal json from file path %s, during the function InitializeRaiderProfiles()", raidAllDataPath), err.Error())
			return nil
		}
	}

	for _, log := range allRaidLogs {
		for _, raider := range log.Players {
			if !mapOfPlayers[raider.Name] {
				classType := ""
				if len(raider.Specs) > 0 {
					classType = raider.Specs[0].TypeRole
				}
				raiderProfile := raiderProfile{
					MainCharName: raider.Name,
					ClassInfo: class{
						IngameClass: raider.ClassName,
						Name:        raider.Name,
						ClassType:   classType,
					},
				}
				newRaiderProfiles = append(newRaiderProfiles, raiderProfile)
				mapOfPlayers[raider.Name] = true
			}
		}
	}

	ReadWriteRaiderProfiles(CalculateAttendance(newRaiderProfiles, allRaidLogs), true)
	return newRaiderProfiles
}

func ReadRaidDataCache(firstPossibleTime time.Time, onlyMainRaid bool) ([]logAllData, error) {
	allLogDataFromCache := []logAllData{}
	returnLogData := []logAllData{}
	if raidDataBytes := CheckForExistingCache(raidAllDataPath); raidDataBytes != nil {
		if err := json.Unmarshal(raidDataBytes, &allLogDataFromCache); err != nil {
			return []logAllData{}, err
		}
	} else {
		return []logAllData{}, errors.New("No logAllData found in cache")
	}
	for _, logData := range allLogDataFromCache {
		timeOfRaid, _ := time.Parse(timeLayout, logData.RaidStartTimeString)
		if !timeOfRaid.Before(firstPossibleTime) {
			if len(logData.RaidNames) == 1 && (strings.Contains(strings.ToLower(logData.RaidNames[0]), "ony") || strings.Contains(strings.ToLower(logData.RaidNames[0]), "zul")) && onlyMainRaid {
				continue
			}
			returnLogData = append(returnLogData, logData)
		}
	}

	if len(returnLogData) == 0 {
		return []logAllData{}, errors.New(fmt.Sprintf("The length of logs found from path %s is 0", raidAllDataPath))
	}
	return returnLogData, nil
}

func DetermineNextSecondaryRaid(session *discordgo.Session) []commingRaid { //E.g. ONY, ZG and AQ20 - This function must be run using the function RunAtSpecificTime()
	currentTime, _ := time.ParseInLocation(timeLayout, GetTimeString(), time.Local)
	cacheRaidDates := ReadWriteRaidCache([]commingRaid{})
	newCachedRaidDates := []commingRaid{}
	if len(cacheRaidDates) == 0 {
		mapOfSecondaryRaidNames := make(map[int]string) //key = name/next reset date
		mapOfSecondaryRaidNames[0] = fmt.Sprintf("ony/%s", currentTime.AddDate(0, 0, 5).Format(timeLayout))
		mapOfSecondaryRaidNames[1] = fmt.Sprintf("zg/%s", currentTime.AddDate(0, 0, 3).Format(timeLayout))

		for x := 0; x <= 1; x++ {
			lastRaid := commingRaid{
				Name:      strings.Split(mapOfSecondaryRaidNames[x], "/")[0],
				NextReset: strings.Split(mapOfSecondaryRaidNames[x], "/")[1],
			}
			cacheRaidDates = append(cacheRaidDates, lastRaid)
		}
		ReadWriteRaidCache(cacheRaidDates)
		return cacheRaidDates
	}
	DetermineNewLogger(cacheRaidDates, session)
	for x, cachedRaid := range cacheRaidDates {
		currentReset, err := time.ParseInLocation(timeLayout, cachedRaid.NextReset, time.Local)
		if err != nil {
			WriteErrorLog(fmt.Sprintf("Was not possible to convert the cached raid time: %s in format: %s", cachedRaid.NextReset, timeLayout), err.Error())
			return []commingRaid{}
		}
		if currentTime.Before(currentReset) {
			WriteInformationLog(fmt.Sprintf("The current time of: %s has not passed thursday, no need to check new", GetTimeString()), "Skipping check for raid reset")
			return []commingRaid{}
		}
		nextReset := currentReset.AddDate(0, 0, cachedRaid.ResetLength)
		daysUntilThursday := (4 - int(currentTime.Weekday()) + 7) % 7
		if daysUntilThursday == 0 {
			daysUntilThursday = 7 // If today is Thursday, move to next week
		}
		nextThursday := currentTime.AddDate(0, 0, daysUntilThursday)
		nextMainReset := time.Date(nextThursday.Year(), nextThursday.Month(), nextThursday.Day(), currentTime.Hour(), currentTime.Minute(), 0, 0, currentTime.Location())
		if nextReset.Before(nextMainReset) {
			newCachedRaidDates = append(newCachedRaidDates, cachedRaid)
		}
		cacheRaidDates[x].NextReset = nextReset.Format(timeLayout)
	}
	ReadWriteRaidCache(cacheRaidDates)
	return newCachedRaidDates
}

func ReadWriteRaidCache(newcommingRaid []commingRaid) []commingRaid {
	cachedRaids := []commingRaid{}
	// Check if the cache file exists
	if _, err := os.Stat(belowRaidersCachePath); err != nil {
		WriteErrorLog("raid cache does not exist yet, creating file...", err.Error())
		return []commingRaid{}
	}

	if len(newcommingRaid) == 0 {
		raidCacheFile, err := os.OpenFile(raidCachePath, os.O_RDONLY, 0644)
		if err != nil {
			WriteErrorLog("Error opening cache file:", err.Error())
			return []commingRaid{}
		}
		defer raidCacheFile.Close() // Always close the file
		// Decode JSON directly from file
		err = json.NewDecoder(raidCacheFile).Decode(&cachedRaids)
		if err != nil {
			WriteErrorLog("Error decoding JSON:", err.Error())
			return []commingRaid{}
		}
		return cachedRaids
	} else {
		commingRaidJson, err := json.MarshalIndent(newcommingRaid, "", " ")
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to marshal json: %s", commingRaidJson), err.Error())
		}
		os.WriteFile(raidCachePath, commingRaidJson, 0644)
		WriteInformationLog("Raid cache has been updated", "Update cache")
		return []commingRaid{}
	}

}

func ReadBelowRaiderCache(userID string) raiderProfile {
	cachedRaiderProfiles := []raiderProfile{}

	// Check if the cache file exists
	if _, err := os.Stat(belowRaidersCachePath); err != nil {
		WriteInformationLog("The function DetrermineNewLogger could not find any present cache, therefor the function will return", "Function return")
		return raiderProfile{}
	}

	// Open the cache file
	raiderCacheFile, err := os.OpenFile(belowRaidersCachePath, os.O_RDONLY, 0644)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("Error opening cache file: %s", belowRaidersCachePath), err.Error())
		return raiderProfile{}
	}
	defer raiderCacheFile.Close() // Always close the file

	// Decode JSON directly from file
	err = json.NewDecoder(raiderCacheFile).Decode(&cachedRaiderProfiles)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("Error decoding JSON: Inside function ReadBelowRaiderCache %v", cachedRaiderProfiles), err.Error())
		return raiderProfile{}
	}

	// Search for the userID in the cache
	for _, raider := range cachedRaiderProfiles {
		if raider.ID == userID {
			return raider
		}
	}

	WriteInformationLog("User not found in cache", "During reading of trial-pug cache")
	return raiderProfile{}
}

func CheckForExistingCache(cachePath string) []byte {
	cacheRaidersBytes := []byte{}
	// Read existing cache
	if _, err := os.Stat(cachePath); err == nil { // Check if file exists
		cacheRaidersBytes, err = os.ReadFile(cachePath)
	} else {
		WriteInformationLog(fmt.Sprintf("Error reading file: Inside function CheckForExistingCache() - %s", cachePath), "Error during cache Read")
	}
	return cacheRaidersBytes
}

func UpdateRaidHelperCache(raidHelperDump any) {
	raidHelperMap, ok := raidHelperDump.(map[string]any)
	if !ok {
		WriteErrorLog(fmt.Sprintf("It was not possible to format the raid to a map function is returning %s", raidHelperDump), "During function UpdateRaidHelperCache()")
		return
	}
	var cachedRaidHelper []map[string]any

	// Read existing cache
	if cacheRaidHelperBytes := CheckForExistingCache(raidHelperCachePath); cacheRaidHelperBytes != nil { // Check if file exists
		err := json.Unmarshal(cacheRaidHelperBytes, &cachedRaidHelper)
		if err != nil {
			WriteErrorLog("Error deconstructing JSON: Inside function UpdateRaidHelperCahce()", err.Error())
		} else {
			cachedRaidHelper = append(cachedRaidHelper, raidHelperMap)
		}
	}

	// Write updated JSON back to file
	jsonRaids, err := json.MarshalIndent(cachedRaidHelper, "", "    ")
	if err != nil {
		WriteErrorLog("Error marshaling JSON: Inside function UpdateRaidHelperCache()", err.Error())
	}

	// Open file with truncate mode
	cacheFile, err := os.OpenFile(raidHelperCachePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("Error opening file: %s inside function UpdateRaidHelperCache()", raidHelperCachePath), err.Error())
	}
	defer cacheFile.Close()

	_, err = cacheFile.Write(jsonRaids)
	if err != nil {
		WriteErrorLog("Error writing to file: Inside function UpdateRaidHelperCache()", err.Error())
	}
}

func UpdateRaiderCache(raider raiderProfile, cachePath string) {
	raiderCacheMutex.Lock()
	defer raiderCacheMutex.Unlock()

	cachedRaiderProfiles := []raiderProfile{}
	uniqueRaiderProfile := raider
	countOfMatches := 0
	timeConvertNewCache := time.Now().Local()

	// Read existing cache
	if cacheRaidersBytes := CheckForExistingCache(cachePath); cacheRaidersBytes != nil { // Check if file exists
		err := json.Unmarshal(cacheRaidersBytes, &cachedRaiderProfiles)
		if err != nil {
			WriteErrorLog("Error deconstructing JSON (RAIDER CHACHE) Inside function UpdateRaiderCache()", err.Error())
		}
	}

	// Update existing profiles
	for i, raider := range cachedRaiderProfiles {
		if raider.ID == uniqueRaiderProfile.ID {
			countOfMatches++
			timeConvertOldCache, err := time.ParseInLocation(timeLayout, raider.LastTimeChangedString, time.Local)
			if err != nil {
				WriteErrorLog("Error parsing old cache time: Inside function UpdateRiaderCache()", err.Error())
			}
			if timeConvertNewCache.After(timeConvertOldCache) {
				if raider.Username != uniqueRaiderProfile.Username {
					WriteInformationLog(fmt.Sprintf("Name updated from: %s to %s", raider.Username, uniqueRaiderProfile.Username), "Updating username for raider")
					cachedRaiderProfiles[i].Username = uniqueRaiderProfile.Username
				}

				sort.Strings(cachedRaiderProfiles[i].DiscordRoles)
				sort.Strings(uniqueRaiderProfile.DiscordRoles)
				if !equalSlices(cachedRaiderProfiles[i].DiscordRoles, uniqueRaiderProfile.DiscordRoles) {
					WriteInformationLog(fmt.Sprintf("Raider with ID: %s has had hes/hers guild roles changed", raider.ID), "Updating discordRoles for raider")
					cachedRaiderProfiles[i].DiscordRoles = uniqueRaiderProfile.DiscordRoles
				}

				if raider.ClassInfo != uniqueRaiderProfile.ClassInfo {
					WriteInformationLog(fmt.Sprintf("Raider with ID: %s has had hes/hers classInfo updated", raider.ID), "Updating classInfo for raider")
					cachedRaiderProfiles[i].ClassInfo = uniqueRaiderProfile.ClassInfo
				} else {
					WriteInformationLog(fmt.Sprintf("Raidier with ID: %s has NO class spec changes...", raider.ID), "Updating classInfo for raider")
				}

				if raider.GuildRole != uniqueRaiderProfile.GuildRole {
					WriteInformationLog(fmt.Sprintf("Raider with ID: %s has had hes/hers guildrole updated", raider.ID), "Updating guildrole for raider")
					cachedRaiderProfiles[i].GuildRole = uniqueRaiderProfile.GuildRole
				}

				if raider.MainCharName != uniqueRaiderProfile.MainCharName {
					WriteInformationLog(fmt.Sprintf("Raider with ID: %s has had hes/hers main char name updated", raider.ID), "Updating charName for raider")
					cachedRaiderProfiles[i].MainCharName = uniqueRaiderProfile.MainCharName
				}

			}
		}
	}

	// Add new profile if no match was found
	if countOfMatches == 0 {
		WriteInformationLog(fmt.Sprintf("The user %s was not found in the database %s", uniqueRaiderProfile.Username, cachePath), "Updating cache")
		uniqueRaiderProfile.LastTimeChangedString = timeConvertNewCache.Format(timeLayout)
		cachedRaiderProfiles = append(cachedRaiderProfiles, uniqueRaiderProfile)
	}
	// Write updated JSON back to file
	jsonRaiderProfiles, err := json.MarshalIndent(cachedRaiderProfiles, "", "    ")
	if err != nil {
		WriteErrorLog("Error marshaling JSON: Inside function UpdateRaiderCache()", err.Error())
	}

	// Open file with truncate mode

	cacheFile, err := os.OpenFile(cachePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("Error opening file: %s inside function UpdateRaiderCache()", cachePath), err.Error())
	}
	defer cacheFile.Close()

	_, err = cacheFile.Write(jsonRaiderProfiles)
	if err != nil {
		WriteErrorLog("Error writing to file: Inside function UpdateRaiderCache()", err.Error())
	}
}

// Helper function to compare slices
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
		if match, _ := regexp.MatchString(`^\d+$`, emoji.ID); match {
			returnEmojies[x].Wrapper = fmt.Sprintf("<:%s:%s>", emoji.Name, emoji.ID)
		} else {
			returnEmojies[x].Wrapper = emoji.ID
		}

	}
	return returnEmojies
}

func NewDiscordSession(debug bool) *discordgo.Session {
	botSession, err := discordgo.New("Bot " + mapOfTokens["botToken"])
	if debug {
		botSession.LogLevel = discordgo.LogDebug
	}
	botSession.Identify.Intents = discordgo.IntentGuildMessages | discordgo.IntentGuildMessageReactions | discordgo.IntentDirectMessageReactions | discordgo.IntentsGuildMembers | discordgo.IntentsDirectMessages | discordgo.IntentGuilds

	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to open a new discord server session to server with id: %s", serverID), err.Error())
	}
	err = botSession.Open()
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to open a new web socket to the discord server with id: %s", serverID), err.Error())
	}

	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to connect the bot to the discord server %s, during the function NewDiscordSession()", serverID), err.Error())
		log.Fatal("An error occured while trying to establish a new discord connection. Please check the keyvault config")
	}
	return botSession
}

func BotMultiReaction(emojiesToUse []emojies, event *discordgo.Message, botSession *discordgo.Session) {
	mapOfAlreadyReactedEmojie := make(map[string]bool)
	for _, emojie := range emojiesToUse {
		if !mapOfAlreadyReactedEmojie[emojie.ID] {
			botSession.MessageReactionAdd(event.ChannelID, event.ID, emojie.ID)
			mapOfAlreadyReactedEmojie[emojie.ID] = true
		}

	}
}

func GetSpecificFieldByValue(discordgoMessageEmbed *discordgo.MessageEmbed, partOfValue string) *discordgo.MessageEmbedField {
	for _, field := range discordgoMessageEmbed.Fields {
		if strings.Contains(field.Value, partOfValue) {
			return field
		}
	}
	return nil
}

func GetSpecificRaidEvent(allCurrentRaids []trackRaid, messageID string) trackRaid {
	for _, raid := range allCurrentRaids {
		if raid.DiscordMessageID == messageID {
			return raid
		}
	}
	return trackRaid{}
}

func GetWeeklyBench(event *discordgo.MessageUpdate) map[string]bench {
	benchRaidersSlice := []string{}
	benchRaidersMap := make(map[string]bench)
	raidLeaderName := ""
	mapOfFields := make(map[string]discordgo.MessageEmbedField)
	for _, embed := range event.Embeds {
		for _, field := range embed.Fields {
			valueToLower := strings.ToLower(field.Value)
			if strings.Contains(valueToLower, "bench") {
				mapOfFields["bench"] = *field
			} else if strings.Contains(valueToLower, "leader") {
				mapOfFields["leader"] = *field
			}
		}
	}

	patternCaptureNameAndTime := regexp.MustCompile(`<:LeaderX:\d+>\s*([^\r\n]+)\s*\r?\n<:DateX:\d+>\s*__<t:(\d+):D>__`)
	matchesSlice := patternCaptureNameAndTime.FindStringSubmatch(mapOfFields["leader"].Value)
	raidTime := ""
	if len(matchesSlice) >= 3 {
		raidLeaderName = matchesSlice[1]
		timeInSeconds, err := strconv.ParseInt(matchesSlice[2], 10, 64)
		if err != nil {
			fmt.Println("THE FOLLOWING ERROR OCVC", err.Error())
		}
		raidTime = time.Unix(timeInSeconds, 0).Format(timeLayOutShort)
		fmt.Println("RAID TIME:", raidTime)
	}

	patternCaptureNames := regexp.MustCompile(`\*\*(.*?)\*\*`)
	matches := patternCaptureNames.FindAllStringSubmatch(mapOfFields["bench"].Value, -1)
	for _, slices := range matches {
		if len(slices) > 1 {
			benchRaidersSlice = append(benchRaidersSlice, slices[1])
		}
	}

	for _, benchedRaider := range benchRaidersSlice {
		currentMap := benchRaidersMap[benchedRaider]
		currentMap.RaidLeaderName = raidLeaderName
		currentMap.RaidLeaderDiscordID = ResolvePlayerName(raidLeaderName, BotSessionMain)
		currentMap.DateString = raidTime
		benchRaidersMap[benchedRaider] = currentMap
	}

	return benchRaidersMap
}

func ReadWriteRaidHelperCache(trackedRaids ...map[string]trackRaid) map[string]trackRaid {
	raidHelperCascheMutex.Lock()
	defer raidHelperCascheMutex.Unlock()
	raids := make(map[string]trackRaid)
	currentRaidCacheBytes := CheckForExistingCache(raidHelperCachePath)
	if len(currentRaidCacheBytes) > 0 {
		err := json.Unmarshal(currentRaidCacheBytes, &raids)
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to unmarshal the raid-helper cache on path %s, during the function ReadWriteRaidHelperCache()", raidHelperCachePath), err.Error())
			return raids
		}
	}

	if len(trackedRaids) > 0 {
		marshal, err := json.MarshalIndent(trackedRaids[0], "", " ")
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to marshalindent cache on path %s, during the function ReadWriteRaidHelperCache()", raidHelperCachePath), err.Error())
			return make(map[string]trackRaid)
		}
		err = os.WriteFile(raidHelperCachePath, marshal, 0644)
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to write cache file %s, during the function ReadWriteRaidHelperCache()", raidHelperCachePath), err.Error())
			return make(map[string]trackRaid)
		}
		return trackedRaids[0]
	}
	return raids
}

func AutoTrackPosts() {
	tracked := ReadWriteTrackPosts()
	checkInterval := time.Second * 10
	// stop channels owned by this goroutine only
	stops := map[string]chan struct{}{}

	start := func(p trackPost) {
		if _, ok := stops[p.MessageID]; ok {
			return // already running
		}
		ch := make(chan struct{})
		stops[p.MessageID] = ch
		go UpdateAnnouncePost(p, ch, checkInterval)
	}

	stop := func(id string) {
		if ch, ok := stops[id]; ok {
			close(ch)
			delete(stops, id)
		}
	}

	// Startup: start active ones
	for _, p := range tracked {
		if p.Active {
			start(p)
			WriteInformationLog(fmt.Sprintf("Starting tracking of all existing posts found on path %s as part of system start-up, len of all tracked posts %d and with interval %s, during the function AutoTrackPosts()", cacheTrackedPostsCache, len(tracked), checkInterval.String()), "Tracking post")
		}
	}

	// React to cache-change signals
	for range trackCacheChanged {
		after := ReadWriteTrackPosts()

		for id, pAfter := range after {
			pBefore, existed := tracked[id]
			if !existed {
				// if you truly never remove and only add, this is fine
				if pAfter.Active {
					start(pAfter)
					WriteInformationLog(fmt.Sprintf("Starting new tracking of post with ID: %s and channel ID to track: %s, and interval %s, during the function AutoTrackPosts()", pAfter.MessageID, pAfter.LinkedChannelID, checkInterval.String()), "Tracking post")
				}
				continue
			}

			// false -> true : start
			if !pBefore.Active && pAfter.Active {
				start(pAfter)
				WriteInformationLog(fmt.Sprintf("Starting an existing tracking of post with ID: %s and channel ID to track: %s, and interval %s, during the function AutoTrackPosts()", pAfter.MessageID, pAfter.LinkedChannelID, checkInterval.String()), "Tracking post")
			}

			// true -> false : stop
			if pBefore.Active && !pAfter.Active {
				stop(id)
				WriteInformationLog(fmt.Sprintf("Stopping an existing tracking of post with ID: %s and channel ID to track: %s, and interval %s, during the function AutoTrackPosts()", pAfter.MessageID, pAfter.LinkedChannelID, checkInterval.String()), "Tracking post")
			}
		}
		tracked = after
	}
}

func UpdateAnnouncePost(post trackPost, channel <-chan struct{}, interval time.Duration) {
ticker := time.NewTicker(interval)
timeToWaitInLoop := time.Second * 5
	defer ticker.Stop()
	for {
		select {
		case <-channel:
			WriteInformationLog(fmt.Sprintf("The post %s has been deactived and will no longer be tracked by the bot, during the function UpdateAnnouncePost()", post.MessageID), "Stop tracking")
			return
		case <-ticker.C:
			message, err := BotSessionMain.ChannelMessage(post.ChannelID, post.MessageID)
			if message == nil {
				//WriteErrorLog(fmt.Sprintf("No message found for post with channel ID %s and message ID: %s", post.ChannelID, post.MessageID), err.Error())
				continue
			}
			if len(message.Embeds) == 0 {
				WriteErrorLog(fmt.Sprintf("The post %s that is supposed to be tracked is not created with embeds, which is required by this function, skipping, during the function UpdateAnnouncePost()", post.MessageID), "Skipping post")
				continue
			}
			embed := message.Embeds[0] //Only care about the first
			if len(embed.Fields) == 0 {
				WriteErrorLog(fmt.Sprintf("The post %s that is supposed to be tracked is not created with fields, which is required by this function, skipping, during the function UpdateAnnouncePost()", post.MessageID), "Skipping post")
				continue
			}
			if len(embed.Fields) == 0 {
				continue
			}
			field := embed.Fields[0]
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve message ID %s from channel ID %s, which is required by this function, skipping, during the function UpdateAnnounceBot()", post.ChannelID, post.MessageID), err.Error())
				continue
			}
			match := false
			notChanged := false
			oldID := ""
			for x := range 5 {
				x = x + 1
				channels, err := BotSessionMain.GuildChannels(serverID)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to get all discord channels on server %s, this is crusial for the tracking of a post, skipping, during the function UpdateAnnounceBot()", serverID), err.Error())
					continue
				}
				for _, channel := range channels {
					if channel.Name == post.LinkedChannelName {
						match = true
						if channel.ID != post.LinkedChannelID {
							oldID = post.LinkedChannelID
							post.LinkedChannelID = channel.ID
							ReadWriteTrackPosts(post)
							WriteInformationLog(fmt.Sprintf("Post with old channel ID %s has changed to ID: %s, which means the post must also be changed to reflect this, during the function UpdateAnnounceBot()", post.ChannelID, channel.ID), "Found new channel")
							break
						}
						notChanged = true
					}
				}
				if match {
					break
				}

				if notChanged {
					break
				}
				WriteInformationLog(fmt.Sprintf("Waiting %s for next try of finding the new channel, it might not just be created yet, for post with ID %s, try %d/5, during the function UpdateAnnounceBot()", timeToWaitInLoop.String(), post.MessageID, x), "Sleeping thread")
				time.Sleep(time.Second * 5)
			}

			if !match {
				WriteErrorLog(fmt.Sprintf("The message ID %s being tracked could not find the new channel ID for tracked channel %s, skipping, during the function UpdateAnnounceBot()", post.MessageID, post.ChannelID), "Skipping post")
				continue
			}

			if notChanged {
				continue
			}

			field.Value = strings.ReplaceAll(field.Value, oldID, post.LinkedChannelID)
			embed.Fields[0] = field
			embeds := []*discordgo.MessageEmbed{embed}
			_, err = BotSessionMain.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Embeds: &embeds,
				ID: post.MessageID,
				Channel: post.ChannelID,
			})
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to send the new embed to change the channel ID inside field, on message ID %s, during the function UpdateAnnounceBot()", post.MessageID), err.Error())
				continue
			}
			WriteInformationLog(fmt.Sprintf("Successfully updated message ID %s to change channel ID from %s and to %s, during the function UpdateAnnounceBot()", post.MessageID, oldID, post.LinkedChannelID), "Updated tracked post")
			/*
			if RetrieveChannelID(field.Value) == "" {
				WriteErrorLog(fmt.Sprintf("Channel ID could not be retrieved from field %s which makes it impossible to verify the tracking of post %s, skipping, during the function UpdateAnnounceBot()", field.Value, post.MessageID), "Field missing channel ID")
				continue
			}
				*/
		}
	}
}

func AutoChangeAnnounceChannel(botSession *discordgo.Session) {
	createChannel := false
	var err error

	channels, err := botSession.GuildChannels(serverID)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrive all guild channels on server with ID %s, returning early, during the function AutoChangeAnnounceChannel()", serverID), err.Error())
		return
	}

	sliceOfBotChannels := []*discordgo.Channel{}
	for _, channel := range channels {
		if channel.Name == channelNameAnnouncement {
			sliceOfBotChannels = append(sliceOfBotChannels, channel)
		}
	}

	if len(sliceOfBotChannels) > 1 { //Channel structure is corrupt and all channels of that name will be deleted and only 1 will be recreated
		createChannel = true
		for _, channel := range sliceOfBotChannels {
			_, err := botSession.ChannelDelete(channel.ID)
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to delete channel with name %s and ID %s, during the function AutoChangeAnnounceChannel()", channel.Name, channel.ID), err.Error())
			}
			time.Sleep(500 * time.Millisecond)
		}
	} else if len(sliceOfBotChannels) == 0 {
		createChannel = true
	}

	if createChannel {
		automaticAnnounceDiscordChannel, err = botSession.GuildChannelCreateComplex(serverID, discordgo.GuildChannelCreateData{
			Name:     channelNameAnnouncement,
			Type:     discordgo.ChannelTypeGuildText,
			ParentID: categoryAssistance,
			PermissionOverwrites: []*discordgo.PermissionOverwrite{
				{
					ID:   roleRaider,
					Type: discordgo.PermissionOverwriteTypeRole,
					Allow: discordgo.PermissionViewChannel |
						discordgo.PermissionSendMessagesInThreads,
					Deny: discordgo.PermissionSendMessages,
				},
				{
					ID: roleTrial,
					Type: discordgo.PermissionOverwriteTypeRole,
					Allow: discordgo.PermissionViewChannel |
						discordgo.PermissionSendMessagesInThreads,
					Deny: discordgo.PermissionSendMessages,
				},
				{
					ID:   serverID, // @everyone
					Type: discordgo.PermissionOverwriteTypeRole,
					Deny: discordgo.PermissionViewChannel,
				},
			}})
	} else {
		automaticAnnounceDiscordChannel = sliceOfBotChannels[0]
	}
	configCurrent.ChannelID = automaticAnnounceDiscordChannel.ID
	ReadWriteConfig(configCurrent)

	threadList, err := botSession.GuildThreadsActive(serverID)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve all threads active from server %s, returning early, during the function AutoChangeAnnounceChannel()", serverID), err.Error())
		return
	}
	for _, thread := range threadList.Threads {
		if thread.ParentID != configCurrent.ChannelID {
			continue
		}
		if _, ok := configCurrent.Announce[thread.Name]; !ok {
			_, err = botSession.ChannelDelete(thread.ID)
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to delete thread with name %s and ID %s, during the function AutoChangeAnnounceChannel()", thread.Name, thread.ID), err.Error())
			}
		}
	}
	mapOfExistingThreads := make(map[string]bool)
	mapOfThreads := make(map[string]discordgo.Channel)
	for threadName := range configCurrent.Announce { //Keep threads alive if they are already present
		for _, thread := range threadList.Threads {
			if thread.ParentID != configCurrent.ChannelID {
				//WriteInformationLog(fmt.Sprintf("Thread with name %s has been skipped before it is not contained below category %s", threadName, categoryAssistance), "Thread skipped")
				continue
			}
			if threadName == thread.Name {
				mapOfExistingThreads[threadName] = true
				mapOfThreads[threadName] = *thread
				tempMessage, err := botSession.ChannelMessageSend(thread.ID, "\u200B")
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to sent a temp message to keep thread with name %s alive, during the function AutoChangeAnnounceChannel()", threadName), err.Error())
					break
				}
				time.Sleep(500 * time.Millisecond)
				err = botSession.ChannelMessageDelete(thread.ID, tempMessage.ID)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to delete a temp message to keep thread with name %s alive, during the function AutoChangeAnnounceChannel()", threadName), err.Error())
					break
				}
				break
			}
		}
	}
	editMainPost := false
	for nameThread, config := range configCurrent.Announce {
		if mapOfExistingThreads[nameThread] {
			//WriteInformationLog(fmt.Sprintf("Thread already exists %s and will not be created, during the function AutoChangeAnnounceChannel()", nameThread), "Skipping thread")
			continue
		}

		if config.MainThread {
			//WriteInformationLog(fmt.Sprintf("The thread with name %s shall be part of the main-thread and will be skipped to not make a stand-alone thread, during the function AutoChangeAnnounceChannel()", nameThread), "Skipping thread")
			continue
		}
		thread, err := botSession.ThreadStartComplex(configCurrent.ChannelID, &discordgo.ThreadStart{
			Name:                nameThread,
			AutoArchiveDuration: 10080,
			Type:                discordgo.ChannelTypeGuildPublicThread,
		})
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to create new thread with name %s, during the function AutoChangeAnnounceChannel()", nameThread), err.Error())
			continue
		}
		editMainPost = true
		mapOfThreads[thread.Name] = *thread

		responseMessage := &discordgo.MessageSend{}
		responseMessage.Content = config.Description
		if config.GIFLocalPath != "" {
			responseMessage.Files = append(responseMessage.Files, NewWebhookParamGIF(config.GIFLocalPath).Files...)

		}
		_, err = botSession.ChannelMessageSendComplex(thread.ID, responseMessage)
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to sent the information message for the given thread %s, during the function AutoChangeAnnounceChannel()", nameThread), err.Error())
			continue
		}
		_, err = botSession.ChannelMessageSend(thread.ID, fmt.Sprintf("\n\n*Want to go back? <#%s> *", configCurrent.ChannelID))
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to sent the anchor back link to thread %s, during the function AutoChangeAnnounceChannel()", nameThread), err.Error())
			continue
		}
		WriteInformationLog(fmt.Sprintf("Thread with name %s has been created and its description has been added to the channel, during the function AutoChangeAnnounceChannel()", nameThread), "Created thread")
	}

	announceMap := configCurrent.Announce
	topics := make([]topic, 0, len(announceMap))
	topicsNotMainThread := make([]topic, 0, len(announceMap))
	for _, t := range announceMap {
		topics = append(topics, t)
	}

	sort.Slice(topics, func(i, j int) bool {
		return topics[i].Order < topics[j].Order
	})

	if editMainPost {
		topicsOfMainThread := []topic{}
		DeleteMessagesInBulk(configCurrent.ChannelID, botSession)
		for _, topic := range topics {
			if topic.MainThread {
				topicsOfMainThread = append(topicsOfMainThread, topic)
			} else {
				topicsNotMainThread = append(topicsNotMainThread, topic)
			}
		}
		count := 0
		for x, topic := range topicsOfMainThread {
			if topic.ToC {
				count++
			}

			if topic.GIFLocalPath != "" {
				gif := NewWebhookParamGIF(topic.GIFLocalPath)
				_, err := botSession.ChannelMessageSendComplex(configCurrent.ChannelID, &discordgo.MessageSend{
					Content: topic.Description,
					Files:   gif.Files,
				})
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to sent index part of the main thread %d, where thread slice length is %d, during the function AutoChangeAnnounceChannel()", x, len(topicsOfMainThread)), err.Error())
				}
			} else {
				_, err := botSession.ChannelMessageSend(configCurrent.ChannelID, topic.Description)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to sent the final message of the main-thread, the remaining length of the message is %d, during the function AutoChangeAnnounceChannel()", len(topicsOfMainThread)), err.Error())
				}
			}
			if count == 1 { //Build ToC
				x := 0
				sliceToc := []string{}
				for _, topic := range topicsNotMainThread {
					x++
					sliceToc = append(sliceToc, fmt.Sprintf("> **%d)** %s ➡ **For details see link <#%s>**", x,  topic.ShortDescription, mapOfThreads[topic.Name].ID))
				}
				_, err := botSession.ChannelMessageSend(configCurrent.ChannelID, strings.Join(sliceToc, "\n"))
				if err != nil {
					WriteErrorLog("An error occured while trying to sent index part of the main thread, where ToC is supposed to be in the channel, during the function AutoChangeAnnounceChannel()", err.Error())
				}
			}
		}
		if count > 1 {
			WriteErrorLog(fmt.Sprintf("Within the %s file more than 1 topic has flag ToC set to true - Only 1 topic can be used, the first one in loop used, during ther function AutoChangeAnnounceChannel()", configPath), "Issue in config")
		}
	}
}

func AutoAnnounceTracker(interval time.Duration, botSession *discordgo.Session) {
	staticConfig := configCurrent
	botSession.AddHandler(func(innerSession *discordgo.Session, message *discordgo.MessageCreate) {
		if automaticAnnounceDiscordChannel != nil && message.Author.ID != innerSession.State.User.ID && message != nil {
			ch, err := innerSession.State.Channel(message.ChannelID)
			if err == nil && ch.IsThread() && ch.ParentID == configCurrent.ChannelID {
				err = botSession.ChannelMessageDelete(ch.ID, message.ID)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to delete message %s from user %s in thread %s as part of auto-removing messages that is not from the bot, during the function AutoAnnounceTracker()", message.ID, message.Author.Username, ch.ID), err.Error())
				}
			}
		}
	})

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		if bytes := CheckForExistingCache(configPath); len(bytes) == 0 {
			ReadWriteConfig(staticConfig)
		}
		configBytesBefore, err := json.Marshal(configCurrent)
		if err != nil {
			WriteErrorLog("An error occured while trying to marshal the old config to bytes, during the function AutoAnnounceTracker()", err.Error())
		}
		lengthBytesBefore := len(configBytesBefore)
		configCurrent = ReadWriteConfig()
		configBytesAfter, err := json.Marshal(configCurrent)
		if err != nil {
			WriteErrorLog("An error occured while trying to marshal the new config to bytes, during the function AutoAnnounceTracker()", err.Error())
		}
		lengthBytesAfter := len(configBytesAfter)
		if lengthBytesAfter != lengthBytesBefore {
			WriteInformationLog(fmt.Sprintf("The config found on path %s has been altered, length before in bytes %d, length after %d, checking if channel exists, during the function AutoAnnounceTracker()", configPath, lengthBytesBefore, lengthBytesAfter), "Config changed")
			channels, err := botSession.GuildChannels(serverID)
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to retrive all channels on server with ID %s, this is crucial for the function and will return early, during the function AutoAnnnounceTracker()", serverID), err.Error())
				continue
			}
			for _, channel := range channels {
				if channel.Name == channelNameAnnouncement {
					WriteInformationLog(fmt.Sprintf("Since config is changed but channel %s already exists, it will be removed, then recreated later, during the function AutoAnnounceTracker()", channelNameAnnouncement), "Found channel")
					_, err = botSession.ChannelDelete(channel.ID)
					if err != nil {
						WriteErrorLog(fmt.Sprintf("An error occured while trying to delete channel %s, during the function AutoAnnounceTracker()", channelNameAnnouncement), err.Error())
					} else {
						WriteInformationLog(fmt.Sprintf("Channel %s successfully deleted due to the config being changed, during the function AutoAnnounceTracker()", channelNameAnnouncement), "Channel deleted")
					}
				}
			}
		}
		if configCurrent.Announce == nil {
			WriteInformationLog(fmt.Sprintf("Since no threads has been found in the config on path %s, this loop will wait till next tick at: %s", configPath, GetTimeString()), "No threads in config")
			continue
		}
		AutoChangeAnnounceChannel(botSession)
	}
}

func AutoTrackRaidEvents(session *discordgo.Session) {
	session.AddHandler(func(session *discordgo.Session, event *discordgo.MessageUpdate) {
		if event.Author.ID == raidHelperId {
			start := time.Now()
			raidHelperCascheMutex.Lock()
			defer raidHelperCascheMutex.Unlock()
			allTrackedRaids := make(map[string]trackRaid)
			raidHelperMap := RetriveRaidHelperEvent(time.Now().Add(time.Hour * 24 * -7))
			messageID := event.Message.ID
			raidTitle := ""

			if raidMap, ok := raidHelperMap[messageID]; ok {
				if mapInner, ok := raidMap.(map[string]any); ok {
					if title, ok := mapInner["title"]; ok {
						raidTitle = title.(string)
					}
				}
			}
			mapOfBenchedPlayers := GetWeeklyBench(event)
			currentTrackRaid := trackRaid{
				DiscordMessageID:      messageID,
				ChannelID:             event.Message.ChannelID,
				RaidDiscordTitle:      raidTitle,
				PlayersAlreadyTracked: mapOfBenchedPlayers,
			}
			allTrackedRaids[currentTrackRaid.DiscordMessageID] = currentTrackRaid
			if currentCache := CheckForExistingCache(raidHelperCachePath); len(currentCache) > 0 {
				trackRaidsCache := make(map[string]trackRaid)
				err := json.Unmarshal(currentCache, &trackRaidsCache)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to unmarshal file %s to type map[string]trackRaid, during the function AutoAtrackRaidEvents", raidHelperCachePath), err.Error())
					return
				}
				for discordMessageID, raid := range trackRaidsCache {
					if _, ok := allTrackedRaids[discordMessageID]; !ok {
						allTrackedRaids[discordMessageID] = raid
					}
				}
			}

			convertBytes, err := json.MarshalIndent(allTrackedRaids, "", " ")
			if err != nil {
				WriteErrorLog("An error occured while trying to marshalindent data of type map[string]trackRaid{}, during the function AutoTrackRaidEvents()", err.Error())
				return
			}
			err = os.WriteFile(raidHelperCachePath, convertBytes, 0644)
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to create cache file: %s, during the function AutoTrackRaidEvents()", raidHelperCachePath), err.Error())
			}
			elapsed := time.Since(start)
			WriteInformationLog(fmt.Sprintf("It took the event handler discordMessageUpdate %s time to finish", elapsed.String()), "Stopwatch")
		}
	})

	session.AddHandler(func(session *discordgo.Session, event *discordgo.MessageDelete) {
		messageID := event.ID //Must not use author or user struct on MessageDelete

		raidHelperCache := ReadWriteRaidHelperCache()
		if len(raidHelperCache) == 0 {
			WriteInformationLog(fmt.Sprintf("The following messageID %s was deleted by the raid-helper bot, but the raid-helper cache is len 0 and therefor nothing to remove, during the function AutoTrackRaidEvents()", messageID), "Cache is len 0")
			return
		}

		if _, ok := raidHelperCache[messageID]; !ok {
			WriteInformationLog(fmt.Sprintf("A message was deleted from the server %s, but this is either not a raid or at least not a raid in cache %s, during the function AutoTrackRaidEvents()", serverID, event.ChannelID), "Ignoring event")
			return
		}

		removeDeletedRaid := make(map[string]trackRaid)
		for ID, raid := range raidHelperCache {
			if messageID != ID {
				removeDeletedRaid[ID] = raid
			}
		}
		ReadWriteRaidHelperCache(removeDeletedRaid)
	})
} //This handler function triggers when a raid-helper event on the discord server is updated
func AutoUpdateRaidLogCache(session *discordgo.Session, sliceOfLoggers []string) {
	session.AddHandler(func(session *discordgo.Session, event *discordgo.MessageCreate) {
		if event.Author.ID == warcraftLogsNativeID && event.ChannelID == channelLog {
			raidLogID := ""
			for _, embed := range event.Embeds {
				sliceOfURL := strings.Split(embed.URL, "/")
				if len(sliceOfURL) == 1 {
					WriteErrorLog("A new raid was found to be posted in the logs channel but the URL in the embed message is malformed %s", "During function AutoUpdateRaidLogCache()")
					break
				}
				raidLogID = sliceOfURL[len(sliceOfURL)-1]
				if raidLogID == "" {
					WriteErrorLog("No raid-log code found in the URL from the embed message sent by the warcraftlogs app", "During function AutoUpdateRaidLogCache()")
					break
				}
				WriteInformationLog("New raid has been detected, therefor a new log will be retrieved using function GetAllWarcraftLogsRaidData() inside of the function AutoUpdateRaidLogCache()", "Sleeping threat")
				fmt.Println("DO WE REACH HERE AT UPDATE?")
				time.Sleep(60 * time.Second)
				logs := GetAllWarcraftLogsRaidData(false, true, raidLogID)
				if len(logs) == 0 {
					WriteErrorLog("An error occured while trying to Retrieve the log posted by a valid discord logger, len of logs is 0 of return on function GetAllWarcraftLogsRaidData()", "During function AutoUpdateRaidLogCache()")
				} else {
					WriteInformationLog(fmt.Sprintf("The following log was found %s and the len of the return slice is %d during the function AutoUpdateRaidLogCache()", logs[0].MetaData.Code, len(logs)), "Warcraftlog retrieved")
				}
			}
		}
	})
}

func GetChannelName(channelID string, session *discordgo.Session) string {
	channelName, err := session.Channel(channelID)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to find the channel name for id: %s", channelID), err.Error())
	}
	if channelName != nil {
		return channelName.Name
	}
	return ""
}

func NewPlayerJoin(botSession *discordgo.Session) {
	mapOfMessageReactions := make(map[string]bool) //MessageID -> bool
	mapOfUsedConnections := make(map[string]bool)
	mapOfNewUsers := make(map[string]raiderProfile)
	stage0 := "Welcome to the server"
	stage2 := "Please answer all the following questions:"
	stage3 := "What is your class? (Use reactions)"
	stage4 := "Please select your in-game race:"
	stage5 := "Please select your spec:"
	stage5a := "Your in-game race has been auto selected as"
	stage6 := "Do you have MC douse?"
	stage7 := "Please type your in-game name with same symbols"
	stage8 := "Please select your hardened role:"
	stage8a := "Do you have engineering?"
	stage9 := "In-game name set to"

	// Handler for when a new user joins
	botSession.AddHandler(func(session *discordgo.Session, eventOuter *discordgo.GuildMemberAdd) {
		raidProfile := raiderProfile{
			Username: eventOuter.User.Username,
			ID:       eventOuter.User.ID,
		}
		err := botSession.GuildMemberRoleAdd(serverID, raidProfile.ID, roleTemp)
		if err != nil {
			WriteErrorLog("An error occured while trying to add user %s to roleTemp", err.Error())
		}

		botSession.ChannelMessageSend(channelBot, fmt.Sprintf("%s <@%s> %s", stage0, raidProfile.ID, crackedBuiltin))
		// Create a new channel for the user
		channelName := fmt.Sprintf("automatic-%s", raidProfile.ID)
		newChannelTemplate := discordgo.GuildChannelCreateData{
			Name:     channelName,
			Type:     discordgo.ChannelTypeGuildText,
			Topic:    "Set roles / raid status for user",
			ParentID: categoryBot,
			PermissionOverwrites: []*discordgo.PermissionOverwrite{
				{
					ID:   serverID,
					Type: discordgo.PermissionOverwriteTypeRole,
					Deny: permissionViewChannel | permissionReadMessages,
				},
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

		if err != nil {
			WriteErrorLog(fmt.Sprintf("Error creating channel %s", newChannelTemplate.Name), err.Error())
			return
		}
		tagUser := discordgo.MessageEmbed{
			Fields: messageTemplates["New_user"].Fields,
		}
		botSession.ChannelMessageSendEmbed(newChannelWithUser.ID, &tagUser)
		botSession.ChannelMessageSend(newChannelWithUser.ID, fmt.Sprintf("<@%s> %s", raidProfile.ID, stage2))

		// Notify in bot channel if user setup is required
		if !mapOfUsedConnections[newChannelWithUser.ID] {
			botSession.ChannelMessageSend(channelBot, fmt.Sprintf("<@%s> %s VISIT <#%s> TO GET SETUP", raidProfile.ID, crackedBuiltin, newChannelWithUser.ID))
			mapOfUsedConnections[newChannelWithUser.ID] = true
			raidProfile.ChannelID = newChannelWithUser.ID
			raidProfile.LastTimeChangedString = GetTimeString()
			UpdateRaiderCache(raidProfile, belowRaidersCachePath)
		}
	})

	//CREATE AN IF FOR LOOKING AT WARCRAFT LOGS, FIND THE PLAYER, LINK THE USER THIS URL AND LET THEM REACT!!

	// Separate Message Handler (Fixes multiple registrations)
	botSession.AddHandler(func(session *discordgo.Session, event *discordgo.MessageCreate) {
		if event.Content != "" && strings.Contains(GetChannelName(event.ChannelID, session), "bot-chat") || strings.Contains(GetChannelName(event.ChannelID, session), "automatic-") {
			/*channel, err := session.State.Channel(event.ChannelID)
			if err != nil {
				WriteErrorLog("An error occured while trying to retrieve the channel of which a message was sent from by user %s, during the function UseSlashCommand()", err.Error())
				return
			}

			*/
			fmt.Println("DO WE REACh", event.Content)
			if event.Author.ID == session.State.User.ID {
				if strings.Contains(event.Content, " VISIT ") {
					patternUserID := regexp.MustCompile(`<@(\d+)>`) //strings.Contains(event.Content, " VISIT ") &&
					patternChannelID := regexp.MustCompile(`<#(\d+)>`)
					userID := patternUserID.FindStringSubmatch(event.Content)
					channelID := patternChannelID.FindStringSubmatch(event.Content)
					newUser := userID[1]
					newChannel := channelID[1]
					raidProfile := raiderProfile{
						LastTimeChangedString: GetTimeString(),
						ID:                    newUser,
					}
					mapOfNewUsers[event.ChannelID] = raidProfile
					UpdateRaiderCache(raidProfile, belowRaidersCachePath)
					if newUser != "" {
						botSession.ChannelMessageSend(newChannel, stage3)
					} else {
						WriteInformationLog(fmt.Sprintf("Was not possible to find the userID of whom joined the server: %s", serverID), "During function NewPlayerJoin()")
					}
					return
				} else if strings.Contains(event.Content, "Please define the amount of time to set for the alarm") {
					fmt.Println("YEP NOW WE HERE!")
				}
			}
			if currentClassSlice := strings.Split(event.Content, " "); len(currentClassSlice) > 2 && !strings.Contains(event.Content, stage3) && event.Author.ID == session.State.User.ID && !strings.Contains(event.Content, stage4) && !strings.Contains(event.Content, stage5) && !strings.Contains(event.Content, stage6) && !strings.Contains(event.Content, stage7) && !strings.Contains(event.Content, stage9) && !strings.Contains(event.Content, stage8a) {
				raceEmojie := emojies{}
				raceEmojieString := ""
				if len(currentClassSlice) > 2 && !strings.Contains(strings.Join(currentClassSlice, " "), "Please") {
					if strings.Contains(currentClassSlice[1], "_") || strings.Contains(currentClassSlice[0], "_") {
						if strings.Contains(currentClassSlice[1], "_") {
							raceEmojieString = strings.Split(currentClassSlice[1], ":")[1]
						} else {
							raceEmojieString = strings.Split(currentClassSlice[0], ":")[1]
						}

					} else if strings.Contains(event.Content, stage5a) {
						raceEmojieString = strings.Split(currentClassSlice[len(currentClassSlice)-1], ":")[len(strings.Split(currentClassSlice[len(currentClassSlice)-1], ":"))-2]
					}
				}
				raceEmojie = DetermineEmoji(raceEmojieString)
				if classEmoji := DetermineEmoji(currentClassSlice[2]); classEmoji.TypeInt == 1 {
					allEmojiesRace := GetEmojies(0, []string{"race"})
					possibleRacesStringSlice := []string{}
					possibleRaces := []emojies{}
					currentClass := classEmoji.Name
					raceEmojiStringSlice := []string{}

					for _, class := range classesImport {
						if strings.Contains(currentClass, strings.ToLower(class.Name)) {
							possibleRacesStringSlice = append(possibleRacesStringSlice, class.PossibleRaces...)
							break
						}
					}

					if len(possibleRacesStringSlice) == 0 {
						WriteErrorLog("Was not possible to find any class specs", "During NewPlayerJoin()")
						return
					}

					for _, raceName := range possibleRacesStringSlice {
						for _, emojiRace := range allEmojiesRace {
							if strings.Contains(strings.ToLower(raceName), emojiRace.ShortName) {
								possibleRaces = append(possibleRaces, emojiRace)
								break
							}
						}
					}

					if len(possibleRaces) == 0 {
						WriteErrorLog(fmt.Sprintf("Was not possile to find any race for whatever the user reacted with %s", event.Content), "During NewPlayerJoin()")
						return
					}

					for _, raceEmoji := range possibleRaces {
						raceEmojiStringSlice = append(raceEmojiStringSlice, fmt.Sprintf("<:%s:%s> %s", raceEmoji.Name, raceEmoji.ID, strings.Split(raceEmoji.Name, "_")[1]))
					}

					message := &discordgo.Message{}
					var err error
					if len(possibleRaces) == 1 {
						_, err = botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("\n\nYour in-game race has been auto selected as %s", possibleRaces[0].Wrapper))
						if err != nil {
							WriteErrorLog("An error occured while trying to add reactions to the in-game race question:", err.Error())
						}
					} else {
						message, err = botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("\n\n%s\n\n%s", stage4, strings.Join(raceEmojiStringSlice, "\n\n")))
						if err != nil {
							WriteErrorLog("An error occured while trying to add reactions to the in-game race question:", err.Error())
						}
						BotMultiReaction(possibleRaces, message, botSession)
					}
				} else if raceEmojie.TypeInt == 0 && !strings.Contains(event.Content, "Please") && raceEmojie != (emojies{}) {
					if _, exist := mapOfNewUsers[event.ChannelID]; exist {
						mapOfSpecs := make(map[string]bool)
						specsPossible := []classSpecs{}
						specsPossibleNoMeme := []classSpecs{}
						allEmojiesSpec := GetEmojies(2, []string{"spec"})
						uniqueClassSpecificEmojies := []emojies{}
						emojieSpecStringSlice := []string{}

						for _, class := range classesImport {
							if matchSpecEmojieSlice := strings.Split(mapOfNewUsers[event.ChannelID].ClassInfo.IngameClass, "_"); len(matchSpecEmojieSlice) > 1 {
								if matchSpecEmojieSlice[1] == strings.ToLower(class.Name) {
									specsPossible = append(specsPossible, class.ClassSpecs...)
									break
								}
							}
						}
						if len(specsPossible) == 0 {
							WriteErrorLog("Due to no specs found, the function will return and the player screening will fail", "During NewPlayerJoin()")
							return
						}

						for _, classSpec := range specsPossible {
							if !classSpec.MemeSpec {
								specsPossibleNoMeme = append(specsPossibleNoMeme, classSpec)
							}
						}

						for _, classSpec := range specsPossibleNoMeme {
							for _, specEmojie := range allEmojiesSpec {
								if classSpec.ClassNickName == specEmojie.ShortName && !mapOfSpecs[specEmojie.ID] {
									uniqueClassSpecificEmojies = append(uniqueClassSpecificEmojies, specEmojie)
									mapOfSpecs[specEmojie.ID] = true
								} else if splitName := strings.Split(specEmojie.Name, "_"); len(splitName) > 1 && !mapOfSpecs[specEmojie.ID] {
									if splitName[1] == strings.ToLower(classSpec.ClassSpec) {
										uniqueClassSpecificEmojies = append(uniqueClassSpecificEmojies, specEmojie)
										mapOfSpecs[specEmojie.ID] = true
									}
								}
							}
						}

						if len(uniqueClassSpecificEmojies) == 0 {
							WriteErrorLog(fmt.Sprintf("No possible specs found for whatever the user reacted with: %s", event.Content), "During NewPlayerJoin()")
							return
						}
						for _, specEmojie := range uniqueClassSpecificEmojies {
							if match, _ := regexp.MatchString(`^\d+$`, specEmojie.ID); !match {
								emojieSpecStringSlice = append(emojieSpecStringSlice, fmt.Sprintf("%s %s", specEmojie.ID, specEmojie.ShortName))
							} else {
								if strings.Contains(specEmojie.ShortName, "-") {
									emojieSpecStringSlice = append(emojieSpecStringSlice, fmt.Sprintf("<:%s:%s> %s", specEmojie.Name, specEmojie.ID, specEmojie.ShortName))
								} else {
									emojieSpecStringSlice = append(emojieSpecStringSlice, fmt.Sprintf("<:%s:%s> %s", specEmojie.Name, specEmojie.ID, strings.Split(specEmojie.Name, "_")[1]))
								}
							}

						}
						message, err := botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("%s\n\n%s", stage5, strings.Join(emojieSpecStringSlice, "\n\n")))
						if err != nil {
							WriteErrorLog("An error occured while trying to send the message asking for user spec:", err.Error())
						}
						BotMultiReaction(uniqueClassSpecificEmojies, message, botSession)
					} else {
						WriteInformationLog("NO USER FOUND IN DATABASE YET", event.ChannelID)
					}
				} else if raceEmojie.TypeInt == 3 && raceEmojie.Name != "cracked" {
					botSession.ChannelMessageSend(event.ChannelID, stage7)
				} else if !event.Author.Bot && len(strings.Split(event.Content, " ")) == 1 {
					if _, exist := mapOfNewUsers[event.ChannelID]; exist {
						botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf(`The name of "%s" recieved, this is correct?`, event.Content))
					}
				} else if strings.Contains(event.Content, "Your in-game race has") {
					emojiIDFromLastMessage := UnwrapEmojiID(event.Content)
					botSession.MessageReactionAdd(event.ChannelID, event.ID, emojiIDFromLastMessage)
				}
			} else if strings.Contains(event.Content, stage3) {
				allClassEmojies := GetEmojies(1, []string{"class"})
				fmt.Println("WE REACH HERE????", allClassEmojies)
				for _, classEmoji := range allClassEmojies {
					err := botSession.MessageReactionAdd(event.ChannelID, event.ID, fmt.Sprintf("%s:%s", classEmoji.Name, classEmoji.ID))
					if err != nil {
						WriteErrorLog("An error occured while trying to create class reactions during NewPlayerJoin()", err.Error())
					}
					time.Sleep(100 * time.Millisecond)
				}
				mapOfMessageReactions[event.ID] = true
			} else if strings.Contains(event.Content, stage4) && !mapOfMessageReactions[event.ID] {
				allRaceEmojies := GetEmojies(0, []string{"race"})
				uniquePossibleRaceSlice := []string{}
				for _, class := range classesImport {
					if strings.ToLower(class.Name) == strings.Split(mapOfNewUsers[event.ChannelID].ClassInfo.IngameClass, "_")[1] {
						uniquePossibleRaceSlice = append(uniquePossibleRaceSlice, class.PossibleRaces...)
						break
					}
				}

				for _, race := range uniquePossibleRaceSlice {
					for _, possibleRaceEmojie := range allRaceEmojies {
						if strings.ToLower(race) == possibleRaceEmojie.ShortName {
							err := botSession.MessageReactionAdd(event.ChannelID, event.ID, fmt.Sprintf("%s:%s", possibleRaceEmojie.Name, possibleRaceEmojie.ID))
							if err != nil {
								WriteErrorLog("An error occured while trying to create race reactions during NewPlayerJoin()", err.Error())
							}
							time.Sleep(10 * time.Millisecond)
						}
					}
				}
				mapOfMessageReactions[event.ID] = true
			} else if strings.Contains(event.Content, stage5) || strings.Contains(event.Content, stage5a) && !mapOfMessageReactions[event.ID] {
				allSpecEmojies := GetEmojies(2, []string{"spec"})
				specificSpecEmojies := []emojies{}
				patternIDOrUnicode := regexp.MustCompile(`<:[a-zA-Z0-9_]+:(\d+)>`)
				emojieIDMatches := patternIDOrUnicode.FindAllStringSubmatch(event.Content, -1)
				if len(emojieIDMatches) < 1 {
					WriteErrorLog("An error occured while trying to find the emojie-spec-IDs from bots last message...", event.Content)
					return
				}

				for _, messageEmojieID := range emojieIDMatches {
					for _, specEmojie := range allSpecEmojies {
						if messageEmojieID[1] == specEmojie.ID {
							specificSpecEmojies = append(specificSpecEmojies, specEmojie)
							break
						}
					}
				}

				if len(specificSpecEmojies) == 0 {
					WriteInformationLog("Could not find any specs", "Function NewPlayerJoin() Stage5")
				}

				for _, specEmojie := range specificSpecEmojies {
					err := botSession.MessageReactionAdd(event.ChannelID, event.ID, fmt.Sprintf("%s:%s", specEmojie.Name, specEmojie.ID))
					if err != nil {
						WriteErrorLog("An error occured while trying to send the reaction during NewPlayerJoin()", err.Error())
					}
				}
				mapOfMessageReactions[event.ID] = true
			} else if strings.Contains(event.Content, stage6) && !mapOfMessageReactions[event.ID] {
				allFunEmojies := GetEmojies(3, []string{"fun"})
				for _, funEmojie := range allFunEmojies {
					if strings.Contains(funEmojie.Name, "yes") || strings.Contains(funEmojie.Name, "no") {
						botSession.MessageReactionAdd(event.ChannelID, event.ID, fmt.Sprintf("%s:%s", funEmojie.Name, funEmojie.ID))
					}
				}

				mapOfMessageReactions[event.ID] = true
			} else if raiderIngameName := strings.Split(event.Content, " "); len(raiderIngameName) == 1 && !event.Author.Bot && !mapOfMessageReactions[event.ID] && event.ChannelID == mapOfNewUsers[event.ChannelID].ChannelID {
				if mapOfNewUsers[event.ChannelID].Username != event.Content {
					raidProfile := mapOfNewUsers[event.ChannelID]
					raidProfile.Username = event.Content
					raidProfile.LastTimeChangedString = GetTimeString()
					UpdateRaiderCache(raidProfile, belowRaidersCachePath)
					mapOfNewUsers[event.ChannelID] = raidProfile
					roleTypeEmojies := GetEmojies(4, []string{"type"})
					roleTypeEmojiesSpecific := []emojies{}
					emojieTypeSlice := []string{}
					for _, roleEmojie := range roleTypeEmojies {
						if roleEmojie.ShortName == "puggie" || roleEmojie.ShortName == "trial" {
							roleTypeEmojiesSpecific = append(roleTypeEmojiesSpecific, roleEmojie)
						}
					}
					for _, typeEmojie := range roleTypeEmojiesSpecific {
						emojieTypeSlice = append(emojieTypeSlice, fmt.Sprintf("%s %s", typeEmojie.Wrapper, typeEmojie.ShortName))
					}
					botSession.GuildMemberNickname(serverID, raidProfile.ID, raidProfile.Username)
					mapOfMessageReactions[event.ID] = true
					botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("%s %s\n\n%s\n\n%s", stage9, event.Content, stage8, strings.Join(emojieTypeSlice, "\n\n")))
				}
			} else if strings.Contains(event.Content, stage9) && !mapOfMessageReactions[event.ID] {
				roleTypeEmojies := GetEmojies(4, []string{"type"})
				roleTypeEmojiesSpecific := []emojies{}
				for _, roleEmojie := range roleTypeEmojies {
					if roleEmojie.ShortName == "puggie" || roleEmojie.ShortName == "trial" {
						roleTypeEmojiesSpecific = append(roleTypeEmojiesSpecific, roleEmojie)
					}
				}
				for _, specificRoleTypeEmojie := range roleTypeEmojiesSpecific {
					botSession.MessageReactionAdd(event.ChannelID, event.ID, fmt.Sprintf("%s:%s", specificRoleTypeEmojie.Name, specificRoleTypeEmojie.ID))
				}
				mapOfMessageReactions[event.ID] = true
			} else if strings.Contains(event.Content, stage8a) && !mapOfMessageReactions[event.ID] {

				mapOfMessageReactions[event.ID] = true
			}
		}
	})

	botSession.AddHandler(func(session *discordgo.Session, event *discordgo.MessageReactionAdd) {
		channel, err := session.Channel(event.ChannelID)
		if err != nil {
			WriteErrorLog("An error occured while trying to retrieve the current channel of where the event came from, during the function NewPlayerJoin()", err.Error())
			return
		}

		if channel.GuildID == "" {
			return
		}
		raidProfile := mapOfNewUsers[event.ChannelID]
		raidProfile.ChannelID = event.ChannelID
		raidProfile.ID = event.Member.User.ID
		mapOfNewUsers[event.ChannelID] = raidProfile
		if event.UserID != session.State.User.ID && strings.Contains(GetChannelName(event.ChannelID, session), "automatic") {
			if raider, exist := mapOfNewUsers[event.ChannelID]; exist {
				if emojie := DetermineEmoji(event.Emoji.Name); emojie.TypeInt == 1 {
					allEmojiesClass := GetEmojies(1, []string{"class"})
					for _, emojie := range allEmojiesClass {
						if strings.Contains(strings.ToLower(emojie.Name), strings.ToLower(event.Emoji.Name)) {
							classEmoji := DetermineEmoji(event.Emoji.Name)
							if classEmoji != (emojies{}) {
								existingProfile, exists := mapOfNewUsers[event.ChannelID]
								if !exists {
									existingProfile = mapOfNewUsers[event.ChannelID]
								}
								existingProfile.ClassInfo.IngameClass = event.Emoji.Name
								existingProfile.ClassInfo.IngameClassEmojiID = event.Emoji.ID
								existingProfile.LastTimeChangedString = GetTimeString()

								mapOfNewUsers[event.ChannelID] = existingProfile
								UpdateRaiderCache(mapOfNewUsers[event.ChannelID], belowRaidersCachePath)
								botSession.ChannelMessageSend(raider.ChannelID, fmt.Sprintf("Class %s %s Chosen\n\n", classEmoji.Wrapper, classEmoji.Name))
								break
							} else {
								botSession.ChannelMessageSend(raider.ChannelID, fmt.Sprintf("Emoji: <:%s:%s> with name: %s is NOT accepted. Try again...", event.Emoji.Name, event.Emoji.ID, event.Emoji.Name))
								break
							}
						}
					}
				} else if emojie := DetermineEmoji(event.Emoji.Name); emojie.TypeInt == 2 {
					raidProfile := mapOfNewUsers[event.ChannelID]
					raidProfile.ClassInfo.SpecEmoji = event.Emoji.Name
					raidProfile.ClassInfo.SpecEmojiID = event.Emoji.ID
					raidProfile.LastTimeChangedString = GetTimeString()
					UpdateRaiderCache(raidProfile, belowRaidersCachePath)
					mapOfNewUsers[event.ChannelID] = raidProfile
					getEmojiesFun := GetEmojies(3, []string{"fun"})
					yesAndNoEmojies := []string{}
					specEmojie := DetermineEmoji(event.Emoji.Name)
					for _, funEmojie := range getEmojiesFun {
						if funEmojie.ShortName == "yes" || funEmojie.ShortName == "no" {
							yesAndNoEmojies = append(yesAndNoEmojies, fmt.Sprintf("%s %s", funEmojie.Wrapper, funEmojie.ShortName))
						}
					}
					botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("Spec %s %s Chosen\nDo you have MC douse?\n\n%s", specEmojie.Wrapper, specEmojie.ShortName, strings.Join(yesAndNoEmojies, "\n\n")))
				} else if emojie := DetermineEmoji(event.Emoji.Name); emojie.TypeInt == 0 {
					allEmojiesRace := GetEmojies(0, []string{"race"})
					for _, emojie := range allEmojiesRace {
						if strings.Contains(strings.ToLower(emojie.Name), strings.ToLower(event.Emoji.Name)) {
							raceEmoji := DetermineEmoji(event.Emoji.Name)
							if raceEmoji != (emojies{}) {
								existingProfile, exists := mapOfNewUsers[event.ChannelID]
								if !exists {
									existingProfile = mapOfNewUsers[event.ChannelID]
								}
								existingProfile.ClassInfo.IngameRace = event.Emoji.Name
								existingProfile.ClassInfo.IngameRaceEmojiID = event.Emoji.ID
								existingProfile.LastTimeChangedString = GetTimeString()
								mapOfNewUsers[event.ChannelID] = existingProfile
								UpdateRaiderCache(mapOfNewUsers[event.ChannelID], belowRaidersCachePath)
								botSession.ChannelMessageSend(raider.ChannelID, fmt.Sprintf("Race %s %s Chosen", raceEmoji.Wrapper, strings.Split(raceEmoji.Name, "_")[1]))
								break
							} else {
								botSession.ChannelMessageSend(raider.ChannelID, fmt.Sprintf("Emoji: <:%s:%s> with name: %s is NOT accepted. Try again...", event.Emoji.Name, event.Emoji.ID, event.Emoji.Name))
								break
							}
						}
					}
				} else if emojie := DetermineEmoji(event.Emoji.Name); emojie.TypeInt == 3 && emojie.Name != "cracked" {
					existingProfile, exists := mapOfNewUsers[event.ChannelID]
					if !exists {
						existingProfile = mapOfNewUsers[event.ChannelID]
					}
					existingProfile.ClassInfo.HasDouseEmojiID = event.Emoji.ID
					existingProfile.ClassInfo.HasDouseEmoji = event.Emoji.Name
					existingProfile.LastTimeChangedString = GetTimeString()
					mapOfNewUsers[event.ChannelID] = existingProfile
					UpdateRaiderCache(mapOfNewUsers[event.ChannelID], belowRaidersCachePath)
					if strings.Contains(event.Emoji.Name, "yes") {
						botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("%s for YES Chosen", emojie.Wrapper))
					} else {
						botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("%s for NO Chosen", emojie.Wrapper))
					}
				} else if emojie := DetermineEmoji(event.Emoji.Name); emojie.TypeInt == 4 {
					raidProfile := ReadBelowRaiderCache(event.UserID)
					raidProfile.GuildRoleEmojieID = emojie.ID
					raidProfile.LastTimeChangedString = GetTimeString()

					finalMessageSlice := []string{}
					switch emojie.ShortName {
					case "puggie":
						{
							botSession.GuildMemberRoleAdd(serverID, raidProfile.ID, rolePuggie)
							finalMessageSlice = append(finalMessageSlice, fmt.Sprintf("Server role puggie assigned, thank you for joining <Hardened> as a pug\n\nBefore signing up, please add your toon to the Gear-check channel:\n\n <#%s>\n\nSee the raid-signups:\n\nAQ40 / BWL <#%s> and NAXX <#%s>", channelGearCheck, channelSignUp, channelSignUpNaxx)) //Must be changed when we run pug raids
						}
					case "trial":
						{
							botSession.GuildMemberRoleAdd(serverID, raidProfile.ID, roleTrial)
							botSession.GuildMemberRoleAdd(serverID, raidProfile.ID, roleGuildMember)
							classDiscordRole := ""
							classLeader := ""
							classChannel := ""
							for roleName, roleValue := range mapOfConstantRoles {
								if emojieNameSplit := strings.Split(raidProfile.ClassInfo.IngameClass, "_"); len(emojieNameSplit) > 1 {
									if emojieNameSplit[1] == strings.ToLower(strings.Replace(roleName, "role", "", -1)) {
										classDiscordRole = roleValue
										for roleLeaderName, roleLeaderValue := range mapOfConstantOfficers {
											if strings.Contains(strings.ToLower(roleLeaderName), emojieNameSplit[1]) {
												classLeader = roleLeaderValue
												break
											}
										}

										for channelRoleName, channelRoleValue := range mapOfConstantClasses {
											if strings.ToLower(strings.Replace(channelRoleName, "channel", "", -1)) == emojieNameSplit[1] {
												classChannel = channelRoleValue
												break
											}
										}
									}
								}
								if classDiscordRole != "" {
									break
								}
							}
							if classDiscordRole == "" {
								WriteInformationLog(fmt.Sprintf("Discord class role not found for player with username: %s id: %s nick: %s", event.Member.User.Username, event.Member.User.ID, event.Member.Nick), "Final message to new player")
							}

							if classChannel == "" {
								WriteInformationLog(fmt.Sprintf("Discord class channel not found for player with username: %s id: %s nick: %s", event.Member.User.Username, event.Member.User.ID, event.Member.Nick), "Final message to new player")
							}

							if classLeader == "" {
								WriteInformationLog(fmt.Sprintf("Discord classleader not found for player with username: %s id: %s nick: %s", event.Member.User.Username, event.Member.User.ID, event.Member.Nick), "Final message to new player")
								classLeader = officerGMArlissa
							}
							botSession.GuildMemberRoleAdd(serverID, raidProfile.ID, classDiscordRole)
							finalMessageSlice = append(finalMessageSlice, fmt.Sprintf("Server role trial assigned, welcome to the <Hardened> Team! %s\n\n**Loot rules are different for trials** - As a general rule of thumb, biggest items are off-limits for first raid minimum.\n\nYour new class leader: @ %s\n\nRaid-leader: %s\n\nGet familiar with your class channel: <#%s>\n\nRaid sign-ups channels: AQ40 / BWL <#%s> NAXX <#%s>\n\nGuild general chat channel: <#%s>", crackedBuiltin, strings.Split(classLeader, "/")[1], SplitOfficerName(officerGMArlissa)["Name"], classChannel, channelSignUp, channelSignUpNaxx, channelGeneral))

						}
					}
					botSession.GuildMemberRoleRemove(serverID, raidProfile.ID, roleTemp)
					botSession.ChannelMessageSend(event.ChannelID, fmt.Sprintf("%s\n\nServer-rules channel: <#%s> %s", strings.Join(finalMessageSlice, "\n\n"), channelServerRules, crackedBuiltin))
					UpdateRaiderCache(raidProfile, belowRaidersCachePath)
					time.Sleep(1 * time.Minute)
					botSession.ChannelDelete(event.ChannelID)
					WriteInformationLog(fmt.Sprintf("Bot channel med ID: %s is deleted", event.ChannelID), "Deleting channel")
				}
			}
		}
	})
}

func SetWarcraftLogQueryVariables(query map[string]any, variableData any) []map[string]any {
	returnWarcraftQueries := []map[string]any{}
	if logBases, ok := variableData.([]logsBase); ok {
		for _, log := range logBases {
			copyQuery := deepCopyMap(query)
			copyQuery["variables"].(map[string]any)["code"] = log.Code
			returnWarcraftQueries = append(returnWarcraftQueries, copyQuery)
		}
	} else if mapOfVariableData, ok := variableData.(map[string]any); ok {
		if fightIDs, ok := mapOfVariableData["IDs"].([]int64); ok {
			copyQuery := query
			copyQuery["variables"].(map[string]any)["fightIDs"] = fightIDs
			returnWarcraftQueries = append(returnWarcraftQueries, copyQuery)
		} else if encounterID, ok := mapOfVariableData["encounterID"].(int64); ok {
			copyQuery := query
			copyQuery["variables"].(map[string]any)["encounterID"] = encounterID
			returnWarcraftQueries = append(returnWarcraftQueries, copyQuery)
		} else if actorID, ok := mapOfVariableData["actorID"].(int); ok {
			copyQuery := query
			copyQuery["variables"].(map[string]any)["actorID"] = actorID
			returnWarcraftQueries = append(returnWarcraftQueries, copyQuery)
		}
	} else if number, ok := variableData.(int); ok {
		copyQuery := query
		copyQuery["variables"].(map[string]any)["page"] = number
		returnWarcraftQueries = append(returnWarcraftQueries, copyQuery)
	} else if name, ok := variableData.(map[string]string); ok {
		copyQuery := query
		copyQuery["variables"].(map[string]string)["name"] = name["name"]
		returnWarcraftQueries = append(returnWarcraftQueries, copyQuery)
	}
	return returnWarcraftQueries
}

func deepCopyMap(original map[string]any) map[string]any {
	copyBytes, _ := json.Marshal(original) // Serialize to JSON
	var copyMap map[string]any
	json.Unmarshal(copyBytes, &copyMap) // Deserialize back to a new map
	return copyMap
}

func GetHttpResponseData(httpMethod string, token string, URL string, customHeaders []string, OAuth2 bool) any { //Must be parsed as key = app id, value = secret
	var returnJson any
	var httpRequest *http.Request
	var err error
	OAuth2Body := url.Values{}
	if OAuth2 {
		OAuth2Body.Add("grant_type", "client_credentials")
		httpRequest, err = http.NewRequest(
			httpMethod,
			URL,
			strings.NewReader(OAuth2Body.Encode()),
		)
		if err != nil {
			WriteErrorLog("An error occured while trying to create a new http request with OAuth2, during the function GetHttpResponseData()", err.Error())
			return nil
		}
		httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		httpRequest.SetBasicAuth(warcraftLogsAppID, mapOfTokens["raidHelperToken"])
	} else {
		httpRequest, err = http.NewRequest(
			httpMethod,
			URL,
			nil,
		)
		if err != nil {
			WriteErrorLog("An error occured while trying to create a new http request without OAuth2, during the function GetHttpResponseData()", err.Error())
			return nil
		}
	}

	for _, customHeader := range customHeaders {
		splitHeader := strings.Split(customHeader, ":")
		if len(splitHeader) != 2 {
			WriteErrorLog(fmt.Sprintf("When providing customHeaders to this function, they must be in format key:value, but got: %s, during the function GetHttpResonseData()", customHeader), "Wrong format")
			continue
		}
		httpRequest.Header.Set(splitHeader[0], splitHeader[1])
	}
	if token != "" && !OAuth2 {
		httpRequest.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	responseHTTP, err := client.Do(httpRequest)
	if err != nil {
		WriteErrorLog("An error occured while trying to serve the request to warcraftLogs during function GetHttpRequestResponseData()", err.Error())
		return nil
	}

	bodyBytes, err := io.ReadAll(responseHTTP.Body)
	if err != nil {
		WriteErrorLog("An error occured while trying to read the iostream of the response body during function GetHttpRequestResponseData()", err.Error())
		return nil
	}
	defer responseHTTP.Body.Close()
	if err := json.Unmarshal(bodyBytes, &returnJson); err != nil {
		WriteErrorLog("An error occured while trying to unmarshal the bytes of the body into a map during function GetHttpRequestResponseData()", err.Error())
		return nil
	}
	return returnJson

} //Only supports JSON data

func GetWarcraftLogsData(query map[string]any) map[string]any {
	var returnMap map[string]any
	url := "https://fresh.warcraftlogs.com/api/v2/client"

	jsonData, err := json.Marshal(query)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to marshal the warcraft logs query %s during function GetWarcraftLogsData()", query), err.Error())
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		WriteErrorLog("An error occured while trying to create the http request struct and allocate a new buffer during function GetWarcraftLogsData()", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mapOfTokens["warcraftLogsRefreshToken"])

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		WriteErrorLog("An error occured while trying to serve the request to warcraftLogs during function GetWarcraftLogsData()", err.Error())
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		WriteErrorLog("An error occured while trying to read the iostream of the response body during function GetWarcraftLogsData()", err.Error())
	}
	defer resp.Body.Close()
	if err := json.Unmarshal(body, &returnMap); err != nil {
		WriteErrorLog("An error occured while trying to unmarshal the bytes of the body into a map during function GetWarcraftLogsData()", err.Error())
	}
	contentOfQuery := query["query"].(string)

	if returnMap == nil {
		WriteInformationLog(fmt.Sprintf("The return map from warcraftlogs API is nill during query %s during function GetWarcraftLogsData()", query), "Warcraft log return")
		return nil
	}
	if errors, ok := returnMap["error"].(string); ok {
		WriteErrorLog(fmt.Sprintf("The warcraftlogs query %s did present an error from return %s", query, errors), "During function GetWarcraftLogsData()")
		fmt.Println("RESPONSE:", returnMap["error"], "MORE")
		fmt.Println("header", req.Header.Get("Authorization"))
		return nil
	}

	if strings.Contains(contentOfQuery, "GetFights") {
		mapOfFightIDs := map[string]any{
			"IDs": UnwrapBaseLogFightIDs(returnMap),
		}
		return mapOfFightIDs
	} else if strings.Contains(contentOfQuery, "GuildLogs") {
		mapOfLogs := map[string]any{
			"logs": UnwrapBaseWarcraftLogRaids(returnMap),
		}
		return mapOfLogs
	} else if strings.Contains(contentOfQuery, "GetLogByCode") {
		mapOfLogs := map[string]any{
			"logs": returnMap,
		}
		return mapOfLogs
	} else if strings.Contains(contentOfQuery, "GetEncounterInfo") {
		mapOfEncounter := map[string]any{
			"encounter": returnMap,
		}
		return mapOfEncounter
	} else if strings.Contains(contentOfQuery, "GetCharacter") {
		mapOfRankings := map[string]any{
			"ranking": returnMap,
		}
		return mapOfRankings
	}
	return nil
}

/*
func UnwrapBaseWarcraftLogRaids(mapToUnwrap map[string]any) []logsBase {
	sliceOfLogs := []logsBase{}
	for _, value := range mapToUnwrap {
		for _, value := range value.(map[string]any) {
			for _, value := range value.(map[string]any) {
				for _, value := range value.(map[string]any) {
					for _, value := range value.([]any) {
						newLog := logsBase{}
						for name, value := range value.(map[string]any) {
							if name == "owner" {
								if strings.ToLower(value.(map[string]any)["name"].(string)) == "throyn1986" {
									newLog.LoggerName = value.(map[string]any)["name"].(string)
								} else if strings.ToLower(value.(map[string]any)["name"].(string)) == "shufflez26" {
									newLog.LoggerName = value.(map[string]any)["name"].(string)
								} else if strings.ToLower(value.(map[string]any)["name"].(string)) == "zyrtec" {
									newLog.LoggerName = value.(map[string]any)["name"].(string)
								}
							} else if name == "code" {
								newLog.Code = value.(string)
							} else if name == "startTime" {
								newLog.startTime = time.Unix(int64(value.(float64))/1000, 0).Local()
							} else if name == "endTime" {
								newLog.endTime = time.Unix(int64(value.(float64))/1000, 0).Local()
							}
						}

						if newLog.LoggerName != "" {
							sliceOfLogs = append(sliceOfLogs, newLog)
						}
					}
				}
			}
		}
	}

	return sliceOfLogs
}
*/

func UnwrapWarcraftLogRaiderRanking(mapToUnwrap map[string]any, raider raiderProfile, logs ...logPlayer) logsRaider {
	raiderData := logsRaider{}
	raiderData.TimeOfData = time.Now().Format(timeLayoutLogs)
	if len(logs) > 0 {
		raiderData.LastRaid = logs[0]
	}
	if len(raiderData.LastRaid.Specs) == 0 {
		WriteInformationLog(fmt.Sprintf("Not possible to run rankings on raider %s due to the log provided being invalid, during the function UnwrapWarcraftLogRaiderRanking()", raider.MainCharName), "Returning early")
		return (logsRaider{})
	}
	ranking, ok := mapToUnwrap["ranking"].(map[string]any)
	if !ok {
		WriteInformationLog("The map parsed does not contain attribute 'ranking' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
		return (logsRaider{})
	}

	data, ok := ranking["data"].(map[string]any)
	if !ok {
		WriteInformationLog("The map parsed does not contain attribute 'data' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
		return (logsRaider{})
	}

	charData, ok := data["characterData"].(map[string]any)
	if !ok {
		WriteInformationLog("The map parsed does not contain attribute 'characterData' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
		return (logsRaider{})
	}

	character, ok := charData["character"].(map[string]any)
	if !ok {
		WriteInformationLog("The map parsed does not contain attribute 'character' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
		return (logsRaider{})
	}

	id, ok := character["id"].(float64)
	if !ok {
		WriteInformationLog("The map parsed does not contain attribute 'id' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
		return (logsRaider{})
	}
	raiderData.URL = fmt.Sprintf("https://fresh.warcraftlogs.com/character/id/%d", int64(id))

	zoneRankings, ok := character["zoneRankings"].(map[string]any)
	if !ok {
		WriteInformationLog("The map parsed does not contain attribute 'zoneRankings' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
		return (logsRaider{})
	}
	if raiderData.Parses.Parse == nil {
		raiderData.Parses.Parse = make(map[string]float64)
	}
	raiderData.Parses.Parse["bestAverage"] = math.Round(zoneRankings["bestPerformanceAverage"].(float64)*100) / 100
	raiderData.Parses.Parse["mediumAverage"] = math.Round(zoneRankings["medianPerformanceAverage"].(float64)*100) / 100
	raidTier, ok := zoneRankings["size"].(float64)
	if !ok {
		WriteInformationLog("The map parsed does not contain attribute 'size' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
		return (logsRaider{})
	}
	raiderData.Parses.RaidTier = fmt.Sprintf("Raid size: %d players", int(raidTier))
	allStars, ok := zoneRankings["allStars"].([]any)
	if !ok {
		WriteInformationLog("The map parsed does not contain attribute 'allStars' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
		return (logsRaider{})
	}
	for _, rank := range allStars {
		if len(raiderData.LastRaid.Specs) == 0 {
			fmt.Println("IS IT TRUELY 0?", raiderData.LastRaid)
			WriteInformationLog(fmt.Sprintf("Cannot calculate allstars for raider %s due to the RaidData.LastRaid is nil, during the function UnwrapWarcraftLogRaiderRanking()", raider.MainCharName), "Skipping calculation")
			break
		}
		specLogName := raiderData.LastRaid.Specs[0].Name
		attribute, ok := rank.(map[string]any)
		if !ok {
			WriteInformationLog("The map parsed does not contain attribute 'rank' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
			continue
		}
		spec, ok := attribute["spec"].(string)
		if !ok {
			WriteInformationLog("The map parsed does not contain attribute 'spec' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
			continue
		}
		if spec != specLogName {
			continue
		}
		raiderData.Parses.SpecName = spec
		if spec != specLogName {
			WriteInformationLog(fmt.Sprintf("The current spec %s does not match the spec played in last raid %s from raider %s, during the function UnwrapWarcraftLogRaiderRanking()", spec, specLogName, raider.MainCharName), "Skipping raider")
			continue
		}
		worldRank, ok := attribute["rank"].(float64)
		if !ok {
			WriteInformationLog("The map parsed does not contain attribute 'rank' 2 and will continue loop, during the function UnwrapWarcraftLogRaiderRanking()", "Skipping raider")
			continue
		}
		raiderData.Parses.RankWorld = worldRank

		regionRank, ok := attribute["regionRank"].(float64)
		if !ok {
			WriteInformationLog("The map parsed does not contain attribute 'regionRank' and will continue loop, during the function UnwrapWarcraftLogRaiderRanking()", "Skipping raider")
			continue
		}
		raiderData.Parses.RankRegion = regionRank

		serverRank, ok := attribute["serverRank"].(float64)
		if !ok {
			WriteInformationLog("The map parsed does not contain attribute 'serverRank' and will continue loop, during the function UnwrapWarcraftLogRaiderRanking()", "Skipping raider")
			continue
		}
		raiderData.Parses.RankServer = serverRank
	}

	ranks, ok := zoneRankings["rankings"].([]any)
	if !ok {
		WriteInformationLog("The map parsed does not contain attribute 'rankings' and will return early, during the function UnwrapWarcraftLogRaiderRanking()", "Returning early")
		return (logsRaider{})
	}

	mapOfRanks := make(map[string]any)
	mapOfAllRanks := make(map[float64]map[string]any)
	maxRankPercent := 0.0
	lowestRankPercent := 0.0
	marsheler, _ := json.MarshalIndent(ranks, "", " ")
	os.WriteFile("dumblydore.json", marsheler, 0644)
	for _, rank := range ranks {
		fmt.Println()
		if ranks, ok := rank.(map[string]any); !ok {
			continue
		} else {
			mapOfRanks = ranks
		}
		//fmt.Println("RANK:", rank)
		//os.Exit(0)
		rankPercent := mapOfRanks["rankPercent"].(float64)
		if rankPercent != 0 {
			rankPercent = math.Round(rankPercent*100) / 100
		}
		if rankPercent > maxRankPercent {
			maxRankPercent = rankPercent
		}

		if lowestRankPercent == 0.0 {
			lowestRankPercent = rankPercent
		}

		if rankPercent < lowestRankPercent {
			lowestRankPercent = rankPercent
		}
		mapOfAllRanks[rankPercent] = rank.(map[string]any)
	}
	maxAndLowestPercent := make(map[string]float64)
	maxAndLowestPercent["highest"] = math.Round(maxRankPercent*100) / 100
	maxAndLowestPercent["lowest"] = math.Round(lowestRankPercent*100) / 100

	for scale, percent := range maxAndLowestPercent {
		encounter, ok := mapOfAllRanks[percent]["encounter"].(map[string]any)
		high := false
		if scale == "highest" {
			high = true
		}
		if ok {
			if high {
				raiderData.Parses.BestBoss.Name = encounter["name"].(string)
			} else {
				raiderData.Parses.WorstBoss.Name = encounter["name"].(string)
			}

		} else {
			WriteInformationLog(fmt.Sprintf("The map provided does not contain key %f, this means raider %s will not have the following scale calculated for bosses %s, during the function UnwrapWarcraftLogRaiderRanking()", percent, raiderData.LastRaid.Name, scale), "Missing map key")
		}
		raiderData.Parses.Parse[scale] = maxAndLowestPercent[scale]
		killInMS, ok := mapOfAllRanks[percent]["fastestKill"].(float64)
		maxDamage := 0.0
		if ok {
			maxDPSSecond, ok := mapOfAllRanks[percent]["bestAmount"].(float64)
			if !ok {
				WriteInformationLog("The map of encounter is missing attribute 'bestAmount' And will break, during the function UnwrapWarcrafrLogRaiderRanking()", "Missing map key")
				break
			}
			timeDuration := time.Duration(killInMS) * time.Millisecond
			minutes := int(timeDuration.Minutes())
			seconds := int(timeDuration.Seconds()) % 60
			maxDamage = math.Round(maxDPSSecond * float64(seconds))
			if high {
				raiderData.Parses.BestBoss.KillTime = fmt.Sprintf("%02d:%02d", minutes, seconds)
				raiderData.Parses.BestBoss.DPS = math.Round(maxDPSSecond)
				raiderData.Parses.BestBoss.MaxTotalDamage = maxDamage
			} else {
				raiderData.Parses.WorstBoss.KillTime = fmt.Sprintf("%02d:%02d", minutes, seconds)
				raiderData.Parses.WorstBoss.DPS = math.Round(maxDPSSecond)
				raiderData.Parses.WorstBoss.MaxTotalDamage = maxDamage
			}
		} else {
			WriteInformationLog("The map of encounter is missing attribute 'fastestKill' And will break, during the function UnwrapWarcrafrLogRaiderRanking()", "Missing map key")
			break
		}
		totalKills, ok := mapOfAllRanks[percent]["totalKills"].(float64)
		if ok {
			if high {
				raiderData.Parses.BestBoss.KillCount = int(totalKills)
			} else {
				raiderData.Parses.WorstBoss.KillCount = int(totalKills)
			}
		} else {
			WriteInformationLog("The map of encounter is missing attribute 'totalKills' And will break, during the function UnwrapWarcrafrLogRaiderRanking()", "Missing map key")
		}
	}
	//fmt.Println("RAIDER NEW DATA:", raiderData.Parses)
	return raiderData
}

func UnwrapBaseWarcraftLogRaids(mapToUnwrap map[string]any) []logsBase {
	sliceOfLogs := []logsBase{}
	dataMap, ok := mapToUnwrap["data"].(map[string]any)
	if !ok {
		return sliceOfLogs
	}

	reportDataMap, ok := dataMap["reportData"].(map[string]any)
	if !ok {
		return sliceOfLogs
	}

	reportsMap, ok := reportDataMap["reports"].(map[string]any)
	if !ok {
		return sliceOfLogs
	}

	dataSlice, ok := reportsMap["data"].([]any)
	if !ok {
		return sliceOfLogs
	}

	for _, item := range dataSlice {
		report, ok := item.(map[string]any)
		if !ok {
			continue
		}

		newLog := logsBase{}

		for name, value := range report {
			switch name {
			case "owner":
				ownerMap, ok := value.(map[string]any)
				if !ok {
					continue
				}
				ownerName, _ := ownerMap["name"].(string)
				lowerName := strings.ToLower(ownerName)
				if lowerName == "throyn1986" || lowerName == "shufflez26" || lowerName == "zyrtec" {
					newLog.LoggerName = ownerName
				}

			case "code":
				if codeStr, ok := value.(string); ok {
					newLog.Code = codeStr
				}

			case "startTime":
				if ts, ok := value.(float64); ok {
					newLog.startTime = time.Unix(int64(ts)/1000, 0).Local()
				}

			case "endTime":
				if ts, ok := value.(float64); ok {
					newLog.endTime = time.Unix(int64(ts)/1000, 0).Local()
				}
			}
		}

		if newLog.LoggerName != "" {
			sliceOfLogs = append(sliceOfLogs, newLog)
		}
	}
	return sliceOfLogs
}

func UnwrapBaseLogFightIDs(mapToUnwrap map[string]any) []int64 {
	fightIDs := []int64{}
	if reportData, ok := mapToUnwrap["data"].(map[string]any)["reportData"].(map[string]any)["report"].(map[string]any)["fights"]; ok {
		if fights, ok := reportData.([]any); ok {
			for _, fight := range fights {
				for name, value := range fight.(map[string]any) {
					if name == "id" {
						if id, ok := value.(float64); ok {
							fightIDs = append(fightIDs, int64(id))
						}
					}
				}
			}
			return fightIDs
		}
	}
	return nil
}

func UnwrapLogRaid(mapToUnwrap map[string]any) {}

func GetTimeString() string {
	logCurrentTime := time.Now().Local()
	return logCurrentTime.Format(timeLayout)
}

func UnwrapEmojiID(content string) string {
	return strings.TrimSuffix(strings.Split(strings.Split(content, " ")[len(strings.Split(content, " "))-1], ":")[2], ">")
}

func DetermineEmoji(nickName string) emojies {
	patternConvertString := regexp.MustCompile(`^([a-zA-Z]+)_\d+$`)
	convertNickNameString := ""
	if convertNickName := patternConvertString.FindSubmatch([]byte(nickName)); len(convertNickName) > 1 {
		convertNickNameString = strings.ToLower(string(convertNickName[1]))
	} else {
		convertNickNameString = strings.ToLower(nickName)
	}

	returnEmojie := emojies{}
	for _, emojie := range emojiesImport {
		if emojie.Name == convertNickNameString {
			returnEmojie = emojie
			break
		}
	}
	if returnEmojie == (emojies{}) {
		for _, emojie := range emojiesImport {
			if strings.ToLower(emojie.ShortName) == convertNickNameString {
				returnEmojie = emojie
				break
			}
		}
	}

	if returnEmojie == (emojies{}) {
		for _, emojie := range emojiesImport {
			if strings.Contains(convertNickNameString, strings.ToLower(emojie.ShortName)) {
				returnEmojie = emojie
				break
			}
		}
	}

	if returnEmojie != (emojies{}) {
		if match, _ := regexp.MatchString(`^\d+$`, returnEmojie.ID); !match {
			returnEmojie.Wrapper = fmt.Sprintf("%s %s", returnEmojie.ID, returnEmojie.ID)
		} else {
			returnEmojie.Wrapper = fmt.Sprintf("<:%s:%s>", returnEmojie.Name, returnEmojie.ID)
		}
	}
	return returnEmojie
}

func SplitScheduleTime(schedule schedule) (time.Time, int, int) {
	timeNow := time.Now().Local()

	// Parse hour and minute from the schedule
	parts := strings.Split(schedule.HourMinute, ":")
	if len(parts) != 2 {
		WriteErrorLog("Invalid time format. Please use HH:MM (e.g., 19:30) Inside function SplitScheduleTime()", "Wrong time format string for input in function 'RunAtSpecificTime'")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		WriteErrorLog("Invalid hour in schedule: Inside function SplitScheduleTime()", err.Error())
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		WriteErrorLog("Invalid minute in schedule: Inside function SplitScheduleTime()", err.Error())
	}
	return timeNow, hour, minute
}

func RunAtSpecificTime(taskToRun func(), schedule schedule, runToday bool) {
	go func() {
		if runToday {
			timeNow, hour, minute := SplitScheduleTime(schedule)
			targetTime := time.Date(timeNow.Year(), timeNow.Month(), timeNow.Day(), hour, minute, 0, 0, timeNow.Location())
			timeLeft := time.Until(targetTime)
			if timeLeft < 0 {
				WriteInformationLog(fmt.Sprintf("The time parsed in the custom schedule: %s is in the past, and wont run...", schedule.Name), "Skipping custom schedule")
				return
			}

			time.Sleep(timeLeft)
			WriteInformationLog(fmt.Sprintf("Executing function with name: %s", runtime.FuncForPC(reflect.ValueOf(taskToRun).Pointer()).Name()), "Scheduled task")
			// Execute the task
			taskToRun()
		} else {
			for {
				timeNow, hour, minute := SplitScheduleTime(schedule)
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
				WriteInformationLog(fmt.Sprintf("Executing function with name: %s", runtime.FuncForPC(reflect.ValueOf(taskToRun).Pointer()).Name()), "Scheduled task")
				// Execute the task
				taskToRun()
			}
		}
	}()
}

func ConvertUnixTime(unixTime float64) string {
	// Convert milliseconds to seconds
	sec := int64(unixTime / 1000)            // Get whole seconds
	nsec := int64(unixTime*1e6) % int64(1e9) // Get nanoseconds (remaining fraction)

	// Convert to time.Time and return formatted string
	return time.Unix(sec, nsec).Local().Format(timeLayout)
}

func NotifyPlayerRaidPlan(session *discordgo.Session) {
	session.AddHandler(func(innerSession *discordgo.Session, message *discordgo.MessageCreate) {
		if message.ChannelID == channelSignUp {
			officerIDs := []string{}
			for _, value := range mapOfConstantOfficers {
				officerIDs = append(officerIDs, strings.Split(value, "/")[0])
			}
			if strings.Contains(strings.Join(officerIDs, ","), message.Author.ID) && strings.Contains(message.Content, googleSheetBaseURL) {
				usersToNotify := RetrieveUsersInRole([]string{roleTrial, roleRaider}, innerSession)
				for _, id := range usersToNotify {
					WriteInformationLog(fmt.Sprintf("Message sent directly to user %s", id), "Notifying player")
					InformPlayerDirectly(fmt.Sprintf("Hi the raid plan has been published for next main raid %s\n\nPlease make sure you look at your specific assignments\n\nIts also VERY important that you FOLLOW them\n\nLink => %s", crackedBuiltin, fmt.Sprintf("https://discordapp.com/channels/%s/%s/%s", serverID, channelSignUp, message.ID)), id, innerSession)
				}
			}
		}
	})
}

func PrepareTemplateWithEmojie(template messageTemplate) messageTemplate {
	allFunEmojies := GetEmojies(template.EmojieGroupType, []string{template.EmojieGroup[0]}) //Must filter after the function as GetEmojies() has too many other dependencies now
	emojieGroupString := strings.Join(template.EmojieGroup, ",")
	if len(allFunEmojies) == 0 {
		WriteInformationLog(fmt.Sprintf("No emojies found for template name: %s type: %d and group: %s", template.Name, template.EmojieGroupType, emojieGroupString), "During function PrepareTemplateWithEmojie()")
		return template
	}
	for _, emojie := range allFunEmojies {
		if strings.Contains(emojieGroupString, emojie.ShortName) {
			template.EmojiesCaptured = append(template.EmojiesCaptured, emojie)
		}
	}
	if len(template.EmojiesCaptured) == 0 {
		WriteInformationLog(fmt.Sprintf("No specific emojies found for template name: %s type: %d and group: %s", template.Name, template.EmojieGroupType, emojieGroupString), "During function PrepareTemplateWithEmojie()")
	}
	return template
}

func NotifyPlayerRaidQuestion(template messageTemplate, session *discordgo.Session) {
	//currentRaiders := []string{}
	//currentRaiders = RetrieveUsersInRole([]string{roleTrial, roleRaider}, session)
	mapOfUsesDone := make(map[string]int)
	test := []string{"340477324258705419"}
	for _, raider := range test {
		time.Sleep(200 * time.Millisecond)
		if dmChannel, err := session.UserChannelCreate(raider); err == nil {
			tagUser := discordgo.MessageEmbed{
				Fields: template.Fields[:2],
			}

			time.Sleep(200 * time.Millisecond)

			if message, err := session.ChannelMessageSendEmbed(dmChannel.ID, &tagUser); err == nil {
				WriteInformationLog(fmt.Sprintf("Direct message successfully sent to user: %s in channel: %s", raider, message.ID), "Sending direct message")

			} else {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to send an embeded message to the user: %s", raider), err.Error())
			}

		} else {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to create a direct channel to the user: %s", raider), err.Error())
		}
	}

	session.AddHandler(func(innerSession *discordgo.Session, message *discordgo.MessageCreate) {
		contentLower := strings.ToLower(message.Content)
		if contentLower == "yes" || contentLower == "no" && message.GuildID == "" && session.State.User.ID != message.Author.ID && mapOfUsesDone[message.ChannelID] == 0 && message.Author.ID != crackedAppID {
			for _, emojie := range template.EmojiesCaptured {
				if emojie.ShortName == "yes" || emojie.ShortName == "no" {
					raiderProfile := raiderProfile{}
					raiderProfile.ID = message.Author.ID
					if contentLower == emojie.ShortName {
						raiderProfile.ClassInfo.HasDouseEmoji = emojie.Name
						raiderProfile.ClassInfo.HasDouseEmojiID = emojie.ID
						UpdateRaiderCache(raiderProfile, raidersCachePath)
						template.Fields[2].Value = fmt.Sprintf("You have responded with '**%s**' - Thank you %s\n\n**Arlissa will make a douse list every week GOING FORWARD**", message.Content, crackedBuiltin)

						if emojie.ShortName == "no" {
							template.Fields[2].Value = template.Fields[2].Value + "\n\nPlease make sure to get this done soon, not 2 days before raid - Visit => https://www.wowhead.com/classic/guide/blackwing-lair-attunement-blackhands-command-classic-wow"
						}
						template.Fields[2].Value = template.Fields[2].Value + "\nPlease also sign for the first BWL raid by doing so:\n\n1. Click on SR link on the raid-helper sign-ups itself\n\n2. Click ur class to actually sign up\n\n<#>"
						tagUser := discordgo.MessageEmbed{
							Fields: template.Fields,
						}
						time.Sleep(200 * time.Millisecond)
						if message, err := session.ChannelMessageSendEmbed(message.ChannelID, &tagUser); err == nil {
							WriteInformationLog(fmt.Sprintf("Direct message successfully sent to user: %s in channel: %s", message.Author.ID, message.ID), "Sending direct message")
						} else {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to sent a direct message in channel: %s to user: %s", message.ChannelID, message.Author.ID), "During function NotifyPlayerRaidQuestion()")
						}
						mapOfUsesDone[message.ChannelID]++
					}
				}
			}
		} else if message.GuildID == "" && session.State.User.ID != message.Author.ID && mapOfUsesDone[message.ChannelID] == 0 && crackedAppID != message.Author.ID {
			WriteInformationLog(fmt.Sprintf("The message: %s recieved does not match 'yes' or 'no' user: %s and channel: %s", message.Content, message.Author.ID, message.ChannelID), "User gave incorrect feedback during direct message")
			_, err := innerSession.ChannelMessageSend(message.ChannelID, fmt.Sprintf("The input given: '**%s**' is invalid. The bot only accepts '**yes**' OR '**no**", message.Content))
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to sent a direct message in channel: %s to user: %s", message.ChannelID, message.Author.ID), "During function NotifyPlayerRaidQuestion()")
			}
		}
	})
}

/*
	func NotifyPlayerSignUp(notifyType string, session *discordgo.Session) {
		currentRaiders := []string{}
		mapOfMissingSignUp := make(map[string]bool)
		mapOfPlayersToContact := make(map[string]bool)
		embedMessageRaidReminder := messageTemplates["Signup_reminder"]
		if strings.Contains(notifyType, "pug") {
			currentRaiders = RetrieveUsersInRole([]string{roleTrial, rolePuggie, roleRaider}, session)
		} else {
			currentRaiders = RetrieveUsersInRole([]string{roleTrial, roleRaider}, session)
		}

		signUpsAsInterface, eventLink, srLink := RetriveRaidHelperEvent(session, false)

		for _, signUpsInterface := range signUpsAsInterface {
			for propertyName, propertyValue := range signUpsInterface {
				if propertyName == "userId" {
					for _, currentRaider := range currentRaiders {
						if strings.Split(currentRaider, "/")[0] == propertyValue.(string) {
							mapOfMissingSignUp[propertyValue.(string)] = true
							break
						}
					}
				}
			}
		}

		for _, raiderName := range currentRaiders {
			if !mapOfMissingSignUp[raiderName] {
				mapOfPlayersToContact[raiderName] = true
			}
		}

		for name, _ := range mapOfPlayersToContact {
			dmChannel, err := session.UserChannelCreate(name)
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to create a private channel with user: %s inside function NotifyPlayerSignUp()", name), err.Error())
			}
			_, err = session.GuildMember(serverID, name)
			if err != nil {
				WriteErrorLog(fmt.Sprintf("An error occured while trying to retrive guild user info: %s inside function NotifyPlayerSignUp()", name), err.Error())
			}
			user, _ := session.GuildMember(serverID, name)

			for x, _ := range embedMessageRaidReminder.Fields {
				if x == 0 {
					embedMessageRaidReminder.Fields[x].Value = fmt.Sprintf("Hi %s\n%s\n", user.Nick, embedMessageRaidReminder.Fields[x].Value)
				} else if x == len(embedMessageRaidReminder.Fields)-2 {
					embedMessageRaidReminder.Fields[x].Value = eventLink
				} else if x == len(embedMessageRaidReminder.Fields)-1 {
					embedMessageRaidReminder.Fields[x].Value = srLink
				}
			}

			tagUser := discordgo.MessageEmbed{
				Fields: embedMessageRaidReminder.Fields,
			}
			_, err = session.ChannelMessageSendEmbed(dmChannel.ID, &tagUser)
			if err != nil {
				WriteErrorLog("An error occured while trying to create the embeded notify for user:", err.Error())
			}
			WriteInformationLog(fmt.Sprintf("Message reminder for raid successfully sent to: %s with name: %s", user.User.ID, user.User.Username), "Remind raider of signing up")
		}
	}
*/
func CheckForOfficerRank(playerID string, botSession *discordgo.Session) bool {
	playerRoles, err := botSession.GuildMember(serverID, playerID)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve discord stats about player: %s inside function CheckForOfficerRank()", playerID), err.Error())
	}
	if strings.Contains(strings.Join(playerRoles.Roles, ","), roleOfficer) || strings.Contains(strings.Join(playerRoles.Roles, ","), roleRaidLeader) {
		WriteInformationLog(fmt.Sprintf("The player with ID: %s has been verifified as having the officer role on discord: %s", playerID, roleOfficer), "Checking for Officer rank")
		return true
	} else {
		WriteInformationLog(fmt.Sprintf("Player with ID: %s does not have role: %s", playerID, roleOfficer), "Check for Officer rank")
	}
	return false
}

func CheckForRaiderRank(playerID string, botSession *discordgo.Session) bool {
	player, err := botSession.GuildMember(serverID, playerID)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve discord stats about player: %s inside function CheckForRaiderRank()", playerID), err.Error())
	}
	for _, playerRole := range player.Roles {
		if strings.Contains(strings.Join([]string{roleRaider, roleTrial}, ","), playerRole) {
			return true
		}
	}
	WriteInformationLog(fmt.Sprintf("The user %s with nick %s is not a raider nor a trial, access denied during the function CheckForRaiderRank()", player.User.ID, player.Nick), "Access denied")
	return false
}

/*
	func RetrieveSecondaryRaidTemplate() []raidHelperEvent {
		customRaidTemplates := []raidHelperEvent{}
		if customRaidTemplateBytes := CheckForExistingCache(customEventTemplatesPath); customRaidTemplateBytes != nil {
			err := json.Unmarshal(customRaidTemplateBytes, &customRaidTemplates)
			if err != nil {
				WriteErrorLog("An error occured while trying to unmarshal json for custom event templates: Inside function RetrieveSecondaryRaidTemplate()", err.Error())
				return nil
			}
		}
		return customRaidTemplates
	}
*/
func DetermineNewLogger(commingRaids []commingRaid, session *discordgo.Session) {
	mapOfSeenLoggers := make(map[string]bool)
	userID := ""
	if len(commingRaids) == 0 {
		WriteInformationLog("The function DetrermineNewLogger was run but no new but no new raids are occruing, therefor the function will return", "Function return")
		return
	}

	for _, value := range mapOfLoggers {
		userID = ""
		if userIDSlice := strings.Split(value, "/"); len(userIDSlice) > 0 {
			userID = userIDSlice[0]
		} else {
			userID = value
		}
		for _, raidLogger := range commingRaids[0].Logger {
			if raidLogger.UserID == userID {
				mapOfSeenLoggers[userID] = true
			}
		}
		if mapOfSeenLoggers[userID] {
			continue
		} else {
			mapOfSeenLoggers[userID] = false
		}
	}

	for name, isSeen := range mapOfSeenLoggers {
		if !isSeen {
			for loggerName, loggerValue := range mapOfLoggers {
				userID = ""
				if strings.Contains(loggerValue, "/") {
					userID = strings.Split(loggerValue, "/")[0]
				} else {
					userID = loggerValue
				}
				if name == userID {
					InformPlayerDirectly(fmt.Sprintf("Hi %s\n\nThis is simply to inform you that our discord bot has defined you as an offical <Hardened> Warcraft-logger! %s\n\nYou will not recieve this message again.. Thank you! %s", loggerName, crackedBuiltin, crackedBuiltin), userID, session)
					commingRaids[0].Logger = append(commingRaids[0].Logger, raidLogger{UserID: userID})
					WriteInformationLog(fmt.Sprintf("Direct-message sent to user: %s", userID), "Notify about being an official logger")
					ReadWriteRaidCache(commingRaids)
					break
				}
			}
		}
	}
}

/*
	func NewSecondaryRaid(raidShortName string, dayOfTheWeek time.Weekday, session *discordgo.Session, cleanRaidChannel bool) {
		// Define the request body
		if cleanRaidChannel {
			DeleteMessagesInBulk(channelSignUpPug, session)
			WriteInformationLog(fmt.Sprintf("Deleted all messages in channel %s", channelSignUp), "Delete all messages from channel")
			time.Sleep(2 * time.Second)
		}

		currentTime := time.Now().Local()
		newCommingRaids := DetermineNextSecondaryRaid(session)
		if len(newCommingRaids) == 0 {
			WriteInformationLog("No raids to post before next main raid", "Skip posting signup")
			return
		}
		fmt.Println("THIS IS THE DETERMINED RAIDS:", newCommingRaids)
		newCommingRaid := commingRaid{}
		for _, commingRaid := range newCommingRaids {
			if commingRaid.Name == raidShortName {
				newCommingRaid = commingRaid
			}
		}
		daysUntilMainRaid := int(time.Thursday) - int(currentTime.Weekday()) //Change time.WeekDay from thursday if your main raid day is say Wednesday, e.g. time.Wednesday
		if daysUntilMainRaid < 0 {
			daysUntilMainRaid += 7
		}

		resetTimeOfSecondaryRaid, _ := time.ParseInLocation(timeLayout, newCommingRaid.NextReset, time.Local)
		nextMainRaidDate := currentTime.AddDate(0, 0, daysUntilMainRaid)
		if nextMainRaidDate.Before(resetTimeOfSecondaryRaid) {
			WriteInformationLog(fmt.Sprintf("Not possible to run %s before main raid as it resets on: %s", resetTimeOfSecondaryRaid.String(), nextMainRaidDate.String()), "Analyzing for an in-between raid")
			return //We cannot run an extra raid between main-raids
		}

		customEvents := RetrieveSecondaryRaidTemplate()
		if len(customEvents) == 0 {
			WriteErrorLog("An error occured while trying to retrieve the custom event templates... Please define at-least 1 otherwise the bot cannot create mid-week pop-up raids", "During function NewSecondaryRaid()")
			return
		}
		daysUntilSunday := (7 - int(currentTime.Weekday())) % 7 // Days to add to reach next Sunday
		customEvents[0].Date = currentTime.AddDate(0, 0, daysUntilSunday).Format("02-January-2006")
		customEvents[0].Time = "19:30"
		customEvents[0].Title = fmt.Sprintf("%s %s", raidShortName, "BEFORE NEXT main raid")
		customEvents[0].Softres.Instance = raidShortName

		// Marshal the data to JSON
		jsonData, err := json.Marshal(customEvents[0])
		if err != nil {
			WriteErrorLog("Error marshalling JSON: Inside function NewSecondaryRaid()", err.Error())
			return
		}

		// Create the request
		url := fmt.Sprintf("https://raid-helper.dev/api/v2/servers/%s/channels/%s/event", serverID, channelSignUpPug)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			WriteErrorLog("Error defining request to raid-helper inside function NewSecondaryRaid()", err.Error())
			return
		}

		// Set headers
		//req.Header.Set("Authorization", apiToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", mapOfTokens["Raid-helper"])

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			WriteErrorLog("Error making request to raid-helper inside function NewSecondaryRaid()", err.Error())
			return
		}
		defer resp.Body.Close()

		// Read the response
		_, err = io.ReadAll(resp.Body)
		if err != nil {
			WriteErrorLog("Error reading response: inside function NewSecondaryRaid()", err.Error())
			return
		}
		WriteInformationLog(fmt.Sprintf("A in-between raid before next main raid has been created for date: %s", customEvents[0].Date), "Adding an extra raid event")
		InformPlayerDirectly(fmt.Sprintf("**--------------------------------------------------------------**\n\nA new raid of type: %s has been created: %s\n\nSecondary raid resets on: %s\n\nNext main raid resets on: %s", raidShortName, customEvents[0].Date, resetTimeOfSecondaryRaid.String(), nextMainRaidDate.String()), SplitOfficerName(officerGMArlissa)["ID"], session)
	}
*/
func SeperateAnyTagsInMessage(messageValue string) []string {
	returnMessageTagsSlice := []string{}
	if messageValueSplit := strings.Split(messageValue, " "); len(messageValueSplit) > 1 || len(messageValueSplit) == 1 && strings.Contains(messageValueSplit[0], "@") {
		for _, stringPartOfMessage := range messageValueSplit {
			if strings.Contains(stringPartOfMessage, "@") {
				patternUserID := regexp.MustCompile(`<@&?\d+>`)
				returnMessageTagsSlice = append(returnMessageTagsSlice, patternUserID.FindAllString(stringPartOfMessage, -1)...)
			}
		}

	}
	return returnMessageTagsSlice
}

func InformPlayerDirectly(message string, userID string, session *discordgo.Session) {
	channel, err := session.UserChannelCreate(userID)
	if err != nil {
		WriteErrorLog("An error occured while trying to create a direct channel with the user inside function InformPlayerDirectly()", err.Error())
		return
	}
	_, err = session.ChannelMessageSend(channel.ID, message)
	if err != nil {
		WriteErrorLog("An error occured while trying to sent a direct message to the user inside function InformPLayerDirectly()", err.Error())
	}
}

/*
	func HttpServerForOauth2() WarcraftLogTokenCurrent {
		newWarcraftLogsTokenBody := WarcraftLogTokenCurrent{
			TokenType:    "refresh_token",
			RefreshToken: `"`,
		}
		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", newWarcraftLogsTokenBody.RefreshToken)
		data.Set("client_id", warcraftLogsAppID)
		req, err := http.NewRequest("POST", "https://www.warcraftlogs.com/oauth/token", bytes.NewBufferString(data.Encode()))
		if err != nil {
			fmt.Println("An error occured while trying to prepare the POST request for a new warcraftlogs access token:", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		client := &http.Client{}
		resp, err := client.Do(req) // ← **This is the actual HTTP request**
		if err != nil {
			fmt.Println("Error making request:", err)
		}
		defer resp.Body.Close()
		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response:", err)
		}

		// Handle response
		if resp.StatusCode == http.StatusOK {
			var result map[string]any
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Println("Error parsing JSON:", err)
			}
			fmt.Println("New Access Token:", result["access_token"])
			if newRefreshToken, ok := result["refresh_token"]; ok {
				fmt.Println("New Refresh Token:", newRefreshToken)
			}
		} else {
			fmt.Println("Error:", string(body))
		}
		return WarcraftLogTokenCurrent{}
	}
*/
func DeleteMessagesInBulk(channelID string, botSession *discordgo.Session) {
	messagesToDelete, err := botSession.ChannelMessages(channelID, 25, "", "", "")
	messageIDsToDelete := []string{}
	for _, messageString := range messagesToDelete {
		messageIDsToDelete = append(messageIDsToDelete, messageString.ID)
	}
	if err != nil {
		WriteErrorLog("An error occured while trying to get discord messages from the main sign-ups channel:", err.Error())
	}
	err = botSession.ChannelMessagesBulkDelete(channelID, messageIDsToDelete)
	if err != nil {
		WriteErrorLog("An error occured while trying to delete all the messages from the main sign-ups channel:", err.Error())
	}
}

func RetrieveChannelID(channelWithTag string) string {
	patternOnlyID := regexp.MustCompile(`<#(\d{17,19})>`)
	if idSlice := patternOnlyID.FindStringSubmatch(channelWithTag); len(idSlice) == 2 {
		return idSlice[1] // digits only
	}
	return ""
}

func RetrieveUsersInRole(roleIDs []string, session *discordgo.Session) []string {
	guildMembers, err := session.GuildMembers(serverID, "", 500)
	guildMembersInCorrectRoles := []string{}
	if err != nil {
		fmt.Println("An error occured while trying to retrieve all guild members of the server:", serverID, err)
	}
	for _, guildMember := range guildMembers {
		for _, role := range guildMember.Roles {
			if strings.Contains(strings.Join(roleIDs, ","), role) {
				guildMembersInCorrectRoles = append(guildMembersInCorrectRoles, guildMember.User.ID)
				break
			}
		}
	}

	return guildMembersInCorrectRoles
}

func RetriveRaidHelperEvent(periodBack time.Time) map[string]any {
	newRaidURL := fmt.Sprintf("https://raid-helper.dev/api/v3/servers/%s/events", serverID)
	raidEvents := make(map[string]any)
	var response any
	getSignupData, _ := http.NewRequest("GET", newRaidURL, nil)
	getSignupData.Header = http.Header{
		"Authorization": {mapOfTokens["raidHelperToken"]},
		"Content-Type":  {"application/json"},
	}
	client := &http.Client{}
	data, err := client.Do(getSignupData)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to sent a HTTP GET to URI: %s, during the function RetrieveRaidHelperEvent()", newRaidURL), err.Error())
		return nil
	}
	// Read and print the response body
	body, err := io.ReadAll(data.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to unmarshal json body data from URI: %s, during the function RetrieveRaidHelperEvent()", newRaidURL), err.Error())
		return nil
	}

	if value, ok := response.(map[string]any); !ok {
		WriteErrorLog("The format of the raidHelperResponse is invalid, it must be map of any, during the function RetrieveRaidHelperEvent()", "Wrong format")
		return nil
	} else {
		for attributeName, value := range value {
			if attributeName == "postedEvents" {
				if events, ok := value.([]any); ok {
					for _, event := range events {
						if currentEvent, ok := event.(map[string]any); ok {
							messageID := currentEvent["id"].(string)
							startF := currentEvent["startTime"].(float64)
							startUnix := int64(startF)
							eventTime := time.Unix(startUnix, 0)
							if eventTime.After(periodBack) {
								raidEvents[messageID] = currentEvent
							}
						}
					}
				}
			}
		}
		if len(raidEvents) == 0 {
			WriteErrorLog("The length of raid-helper events is 0 looking back to time %s, during the function RetrieveRaidHelperEvent()", "Length 0")
		}
	}
	return raidEvents
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
		log.Fatal("An error occured while trying to load the keyvault config: Inside function ImportKeyVaultConfig()", err.Error())
	}
	json.Unmarshal(keyvaultImportBytes, &KeyvaultConfig)

	if len(KeyvaultConfig.Tokens) < 1 {
		log.Fatal("An error occured while counting the amount of secrets available: Less than 1 token definitions are present")
	}

	KeyvaultConfig = keyvault{
		Name:   fmt.Sprintf("https://%s.vault.azure.net", KeyvaultConfig.Name),
		Tokens: KeyvaultConfig.Tokens,
	}
}

// Any errors occruing within this function will kill this program
func WriteErrorLog(message string, errorMessage string) {
	errorLogMutex.Lock()
	defer errorLogMutex.Unlock()
	cachedErrors := []errorLog{}
	errStruct := errorLog{
		Message:   message,
		Error:     errorMessage,
		TimeStamp: GetTimeString(),
	}
	if currentCache := CheckForExistingCache(errorLogPath); len(currentCache) > 0 {
		err := json.Unmarshal(currentCache, &cachedErrors)
		if err != nil {
			log.Fatal("An error occured while trying to unmarshal json from information log cache: inside function WriteErrorLog()", err)
		}
	}
	cachedErrors = append(cachedErrors, errStruct)

	errJson, err := json.MarshalIndent(cachedErrors, "", " ")
	if err != nil {
		log.Fatal("An error occured while trying to marshal information log to json: inside function WriteErrorLog()", err)
	}

	cacheFile, err := os.OpenFile(errorLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error opening file: Inside function WriteErrorLog() %s", errorLogPath), err.Error())
	}
	defer cacheFile.Close()
	cacheFile.Write(errJson)
}

func WriteInformationLog(message string, action string) {
	cachedLogs := []informationLog{}
	information := informationLog{
		Action:    action,
		Message:   message,
		TimeStamp: GetTimeString(),
	}

	if currentCache := CheckForExistingCache(informationLogPath); currentCache != nil {
		err := json.Unmarshal(currentCache, &cachedLogs)
		if err != nil {
			WriteErrorLog("An error occured while trying to unmarshal json from information log cache: Inside function WriteInformationLog()", err.Error())
		}
	}
	cachedLogs = append(cachedLogs, information)

	informationJson, err := json.MarshalIndent(cachedLogs, "", " ")
	if err != nil {
		WriteErrorLog("An error occured while trying to marshal information log to json: Inside function WriteInformationLog()", err.Error())
	}

	cacheFile, err := os.OpenFile(informationLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("Error opening file: %s inside function WriteInformationLog()", informationLogPath), err.Error())
	}
	defer cacheFile.Close()

	cacheFile.Write(informationJson)
}

func PrepareRaidResponse(raids []logAllData, summary bool) (string, error) {
	returnResponseString := ""
	if raids == nil {
		return "", errors.New("No raids found during function PrepareRaidResponse(). This is an internal error, please let Wyzz know")
	}

	if summary {

	}
	return returnResponseString, nil
}

func StorageAccountClient(storageAccountURI string) *azblob.Client {
	// Authenticate using Default Azure Credentials (Make sure you're logged in via 'az login' if using local dev)
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		WriteErrorLog("An error occured while trying to create a azidentity cred:", err.Error())
		return nil
	}
	if storageClient, err := azblob.NewClient(storageAccountURI, cred, &azblob.ClientOptions{}); err == nil {
		return storageClient
	} else {
		WriteErrorLog("An error occured while trying to create a storage client:", err.Error())
	}
	return nil
}

func StorageAccountAppendBlob(blobName string, containerName string, logName string, storageClient *azblob.Client, context context.Context) error {
	//Runs first time as part of app starting when being deployed (The appendblob wont be there yet)
	storageClient.CreateContainer(context, containerName, nil)

	logFile, err := os.Open(logName)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to open the log file %s, during the function StorageAccountAppendBlob()", logName), err.Error())
		return err
	}

	defer logFile.Close()

	_, err = storageClient.UploadFile(context, containerName, blobName, logFile, nil)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to upload the log file %s to %s, during the function StorageAccountAppendBlob()", logName, storageClient.URL()), err.Error())
		return err
	}
	return nil
}
