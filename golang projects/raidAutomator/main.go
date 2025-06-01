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
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
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

type keyvaultToken struct {
	Name      string `json:"name"`
	VersionID string `json:"version"`
}

type keyvault struct {
	URI    string
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
}

type raiderProfile struct {
	Username              string                `json:"username"`
	MainCharName          string                `json:"mainCharName"`
	ID                    string                `json:"id"`
	IsOfficer             bool                  `json:"isOfficer"`
	GuildRole             string                `json:"guildRole"`
	GuildRoleEmojieID     string                `json:"guildRoleEmojieID"`
	DiscordRoles          []string              `json:"discordRoles"`
	ChannelID             string                `json:"channelId"`
	ClassInfo             class                 `json:"classInfo"`
	AttendanceInfo        map[string]attendance `json:"attendance"`
	LastTimeChangedString string                `json:"lastTimeChangedRaider"`
	DateJoinedGuild       string                `json:"date_joined_guild"`
	RaidData              logsRaider            `json:"raidData"`
	MainSwitch            map[string]bool       `json:"mainSwitch"`
}

type raiderProfiles struct {
	GuildName string `json:"guildName"`
	CountOfLogs int `json:"countOfLogs"`
	LastTimeChangedString string `json:"lastTimeChanged"`
	Raiders []raiderProfile `json:"raiderProfiles"`
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

type attendance struct {
	RaidCount         int
	RaidProcent       float64
	MainRaid          bool
	RaidsMissed       []string
	LateNoticeProcent float64 //Out of the 100% of the time where a raider is ABSCENT, how many % of that time is the notice late
}

type logsRaider struct { //Raw data
	URL         string               `json:"url"`
	WorldBuffs  map[int]logWorldBuff `json:"worldBuffs"`
	Consumes    bool                 `json:"consumes"`
	Parse       int                  `json:"parse"`
	Top1        bool                 `json:"top1"`
	Top3        bool                 `json:"top3"`
	Top5        bool                 `json:"top5"`
	TopProcent  float64              `json:"topProcent"`
	LastRaid    logPlayer            `json:"lastRaidStats"`
	AverageRaid logPlayer            `json:"averageRaidStats"`
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
	InternalLogID    int
	Specs            []logPlayerSpec
	ClassName        string
	WarcraftLogsGUID int64
	DamageTaken      int64
	DamageDone       int64
	HealingDone      int64
	ItemLevel        int
	WorldBuffs       []logWorldBuff
	Enchants         []logPlayerEnchant
	Deaths           []logPlayerDeath
	Abilities        []logPlayerAbility
	MinuteAPM        float64
	ActiveTimeMS     int64
	Consumables      map[string]consumable
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
	LeaderID    string            `json:"leaderId"`
	Time        string            `json:"time"`
	TemplateID  string            `json:"templateId"`
	Date        string            `json:"date"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Softres     raidHelperSoftres `json:"softres"`
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

var (
	classesImport  = []classesInternal{}
	emojiesImport  = []emojies{}
	KeyvaultConfig = keyvault{}

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
	}

	slashCommandAllUsers = map[string]applicationCommand{
		/*
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
						{
							Name:        "playerinfo",
							Description: "Use the 'playerinfo' command to see a sub-menu of options related to you",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionBoolean,
									Name:        "attendance",
									Description: "See information related to your attendance in main-raids",
									Required:    false,
								},
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
						},
					},
				},
				Responses: map[string]applicationResponse{
					"overallinformation": {},
				},
			},
		*/
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
		"myraiderperformance": {
			Template: &discordgo.ApplicationCommand{
				Name: "myraiderperformance",
				Description: fmt.Sprintf("See general information about your mains raid-performance in %s", guildName),
			},
		},
	}

	slashCommandAdminCenter = map[string]applicationCommand{
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
		"simplemessagefromthebot": {
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
				Options: []*discordgo.ApplicationCommandOption{
					{
						Required:    false,
						Name:        "playername",
						Description: "Use @<playername> to see specific raider attendance about user",
						Type:        discordgo.ApplicationCommandOptionString,
					},
				},
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
		"updateweeklyattendance": {
			Template: &discordgo.ApplicationCommand{
				Name: "updateweeklyattendance",
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
			"query": `query GuildLogs($guildID: Int!) {
				reportData {
					reports(guildID: $guildID) {
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
			"variables": map[string]interface{}{
				"guildID": 773986, // Replace with the actual Guild ID from Step 1
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
		"officerGMWyzz":  officerGMWyzz,
		"officerPriest":  officerPriest,
		"officerRogue":   officerRogue,
		"officerWarrior": officerWarrior,
	}

	mapOfLoggers = map[string]string{
		"Throyn1986": officerPriest,
		"Zyrtec":     officialLogger1,
	}

	classesPath             = baseCachePath + "classes.json"
	keyvaultPath            = baseCachePath + "keyvault.json"
	emojiesPath             = baseCachePath + "emojies.json"
	belowRaidersCachePath   = baseCachePath + "cache_trials_pugs.json"
	raidersCachePath        = baseCachePath + "cache_raiders.json"
	raiderProfilesCachePath = baseCachePath + "cache_raider_profiles.json"
	raidHelperCachePath     = baseCachePath + "cache_raid_helper.json"
	raidCachePath           = baseCachePath + "cache_raids.json" // Will be the largest file due to warcraftlogs info
	raidAllDataPath         = baseCachePath + "cache_raid_all_data.json"
	informationLogPath      = baseCachePath + "information_log.json" // Will grow over time
	//errorLogPathWarcraftLogs = baseCachePath + "warcraft_logs_query_errors.json" // Will grow over time
	errorLogPath             = baseCachePath + "error_log.json" // Will grow over time
	customSchedulePath       = baseCachePath + "custom_schedules.json"
	customEventTemplatesPath = baseCachePath + "custom_event_templates.json" // If this is not there when the program starts, it cannot create in-between raids

	ScheduledEvents = []schedule{ //NIL
		{
			Name: "updateweeklyattendance",
			HourMinute: "12:00",
			Weekday: time.Friday,
			Interval: 7,
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

	BotSessionMain = &discordgo.Session{}

	greenColor  = 0x00FF00 // Pure Green
	yellowColor = 0xFFFF00 // Pure Yellow
	redColor    = 0xFF0000 // Pure Red
	blueColor   = 0x0000FF // Pure Blue

	raiderCacheMutex          sync.Mutex
	errorLogMutex             sync.Mutex
	errorLogWarcraftLogsMutex sync.Mutex

	GuildStartTime time.Time
)

const (
	serverID           = "630793944632131594"
	botName            = "raid-automater"
	guildName 		   = "Hardened"
	channelInfo        = "1308521695564402899"
	channelLog         = "1318700380900823103"
	channelGeneral     = "1308521052036530291"
	channelVoting      = "1316379489906855936"
	channelSignUp      = "1308521842407116830"
	channelSignUpPug   = "1334949433208606791"
	channelSignUpBWL   = "1346922479951675483"
	channelWelcome     = "1309312094822203402"
	channelBot         = "1336098468615426189"
	channelServerRules = "1312791528267186216"
	channelOfficer     = "1308522605065539714"

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

	roleOfficer = "1309512897612746752"

	officerGMWyzz         = "346353264461217795/Wyzz"
	officerSecondGMPantse = "211564792794578944/Pantse"
	officerPriest         = "812709542554370098/Throyn"
	officerRogue          = "655113437327917065/Akasuna"
	officerWarrior        = "280120319874695169/Toxico"

	officialLogger1 = "276387587155820544" //Zyrtek

	raidHelperEventBaseURL = "https://raid-helper.dev/api/v2/events/"
	raidHelperId           = "579155972115660803"

	baseCachePath = "./"

	permissionViewChannel    = int64(1 << 0)  // 1
	permissionSendMessages   = int64(1 << 10) // 1024
	permissionReadMessages   = int64(1 << 11) // 2048
	permissionManageMessages = int64(1 << 13) // 8192

	timeLayout     = "January 2, 2006 15:04:05"
	timeLayoutLogs = "02-01-2006 15:04:05"

	warcraftLogsAppID      = "9e368091-5c2c-4593-a997-c08790420e08"
	warcraftLogsServerSlug = "thunderstrike"
	warcraftLogsRegion     = "EU"
	warcraftLogsNativeID   = "1335683225263018024"

	azureStorageURI = "raiderbuild.blob.core.windows.net"
)

func init() {
	//CheckRuntime()
}

func CheckRuntime() {
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
	azKeyvaultClient, err := azsecrets.NewClient(KeyvaultConfig.URI, azCred, nil)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to start a new key vault client to: %s", KeyvaultConfig.URI), err.Error())
	} else {
		WriteInformationLog("Keyvault client successfully established during start-up", "Import Keyvault client")
	}
	azContext := context.Background()
	for _, tokenConfig := range KeyvaultConfig.Tokens {
		secret, err := azKeyvaultClient.GetSecret(azContext, tokenConfig.Name, tokenConfig.VersionID, nil)
		if err != nil {
			WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve the specific secret for %s", tokenConfig.Name), err.Error())
		} else {
			mapOfTokens[tokenConfig.Name] = *secret.Value
		}
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

	/*
		Test different required connections for bot:
		Discord server itself
		Raid-helper API
		WarcraftLogs API
	*/
	WriteInformationLog(fmt.Sprintf("Bot %s successfully established a connection with server id %s", botName, serverID), "Connect to Discord")
	RetriveRaidHelperEvent(BotSessionMain, true)
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
		errors.New("An error occured")
		fmt.Println("WE ARE HERE, ", timeLayout, timeGuildStarted)
	}
	/*
		SIGNALS BELOW
	*/
	ImportEmojies()
	ImportClasses()
	//NotifyPlayerRaidQuestion((PrepareTemplateWithEmojie(messageTemplates["Ask_raider_direct_question_douse"])), BotSessionMain)
	NewPlayerJoin(BotSessionMain)
	NewSlashCommand(BotSessionMain)
	UseSlashCommand(BotSessionMain)
	AutoUpdateRaidLogCache(BotSessionMain, []string{"shufflez26"})
	if profiles := ReadWriteRaiderProfiles(nil, true); len(profiles) == 0 {
		InitializeDiscordProfiles(InitializeRaiderProfiles(), BotSessionMain, true) //Retrieve ALL raiders from ANY time since the guild startet logging
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
		case "ony":
			{
				RunAtSpecificTime(func() {
					NewSecondaryRaid("onyxia", taskSchedule.Weekday, BotSessionMain, false)
				}, taskSchedule, false)
			}
		case "sign1":
			{
				RunAtSpecificTime(func() {
					NotifyPlayerSignUp("sign", BotSessionMain)
				}, taskSchedule, false)
			}
		case "sign2":
			{
				RunAtSpecificTime(func() {
					NotifyPlayerSignUp("sign", BotSessionMain)
				}, taskSchedule, false)
			}
		case "cleanup":
			{
				RunAtSpecificTime(func() {
					DeleteMessagesInBulk(channelSignUpBWL, BotSessionMain)
				}, taskSchedule, false)
				WriteInformationLog(fmt.Sprintf("Deleting all messages in the main-signup channel: %s", channelSignUp), "Delete all channel messages")
			}
		case "updateweeklyattendance": {
			RunAtSpecificTime(func() {
				WriteInformationLog(AddWeeklyRaiderAttendance(), "Updating weekly attendance")
			}, taskSchedule, false)
		}
		default:
			{
				if strings.Contains(taskSchedule.Name, "notify") {
					RunAtSpecificTime(func() {
						NotifyPlayerSignUp("sign", BotSessionMain)
					}, taskSchedule, true)
				} else if strings.Contains(taskSchedule.Name, "cleanup") {
					RunAtSpecificTime(func() {
						NewSecondaryRaid("onyxia", time.Weekday(taskSchedule.WeekdayInt), BotSessionMain, true)
					}, taskSchedule, true)
				}
			}
		}
	}

	//fmt.Println(len(GetAllWarcraftLogsRaidData(false, true)))
	//Since we are running inside a PaaS service, we will never stop unless forced
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}

func NewInteractionResponseToSpecificCommand(logType int, data string) discordgo.InteractionResponse {
	messageSlice := strings.Split(data, "|")
	if len(messageSlice) <= 1 {
		WriteInformationLog("The data provided for the function NewInteractionResponseToSpecificCommand() is missing parts. Please use format commandName/Output", "Create Slash Command response")
		if data == "" {
			WriteInformationLog("No data provided for the function NewInteractionResponseToSpecificCommand() - Data is crucial for this function, so it will return early...", "Return early")
		}
		return discordgo.InteractionResponse{}
	}

	switch logType {
	case 0:
		{
			templateCopy := slashCommandGeneralResponses["errorMessage"].Response
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
			return *templateCopy
		}
	case 1:
		{
			templateCopy := slashCommandGeneralResponses["verboseMessage"].Response
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
			return *templateCopy
		}
	case 2:
		{
			templateCopy := slashCommandGeneralResponses["successMessage"].Response
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
			return *templateCopy
		}
	}
	return discordgo.InteractionResponse{}
}

func GetAllWarcraftLogsRaidData(inMem bool, newestOne bool, logCode string) []logAllData {
	//time.Sleep(30 * time.Second)
	quriesToRun := []map[string]any{}
	WriteInformationLog("Retrieving warcraftlogs data for query with name: 'guildLogsRaidIDs' during function GetAllWarcraftLogsRaidData()", "Getting Warcraft logs data")
	time.Sleep(5 * time.Second)
	currentLogsBase := GetWarcraftLogsData(mapOfWarcaftLogsQueries["guildLogsRaidIDs"])
	mapOfQueries := SetWarcraftLogQueryVariables(mapOfWarcaftLogsQueries["logsByOwnerAndCode"], currentLogsBase["logs"])
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
		index := x
		time.Sleep(1 * time.Second)
		newQuery := SetWarcraftLogQueryVariables(mapOfWarcaftLogsQueries["allFightIDsForRaid"], []logsBase{currentLogsBase["logs"].([]logsBase)[index]})
		WriteInformationLog("Retrieving warcraftlogs data for query with name: 'allFightIDsForRaid' during function GetAllWarcraftLogsRaidData()", "Getting Warcraft logs data")
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

	return WriteRaidCache(logsOfAllRaids)
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
	marsherler, _ := json.MarshalIndent(mapToUnwrap, "", " ")
	os.WriteFile("testerrs.json", marsherler, 0644)
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
		fmt.Println("AN ERROR OCCURED2")
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
						Name:             playerInfo["name"].(string),
						WarcraftLogsGUID: int64(playerInfo["guid"].(float64)),
						Specs:            playerSpecs,
						ClassName:        playerInfo["type"].(string),
					}
					playerLogs = append(playerLogs, playerLog)
				} //Make sure no chickens and other trinket stuff gets a playerObject
			}

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

	raidNameShort := ""
	if raidTitle, ok := mapSemiUnwrapped["title"].(string); ok {
		raidNameShort = strings.ToLower(strings.Split(raidTitle, " ")[0])
	} else {
		fmt.Println("DO WE EVER REACH HERE?", mapSemiUnwrapped)
	}

	mainRaidName := RaidNameLongHandConversion(raidNameShort)
	raidNames := []string{}
	raidNames = append(raidNames, mainRaidName)
	for name := range mapOfZoneNames {
		if name != mainRaidName {
			raidNames = append(raidNames, name)
		}
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

func GetIntPointer(n int64) *int64 {
	return &n
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

/*
	Average raid time in hrs/minutes/seconds: **%s**

Average deaths per raid: **%s**

Average item level: **%s**

Average player count: **%s**

%% Increase/Decrease in average item level: **%s%%**

%% Increase/Decrease in average player count: **%s%%**

%% Increase/Decrease in average deaths: **%s%%**
%% Increase/Drecrease in average raid clear time: **%s%%**


*/

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

//func Determine

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
	fmt.Println("See small raids:", onlyMainRaid)
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
		masherler, _ := json.Marshal(slice)
		responseString := ""
		fmt.Println("RAID TYPE", x, len(slice), len(masherler))
		fmt.Println("WHAT IS THE COUNTER?", totalCountEmbeds)
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

func UseSlashCommand(session *discordgo.Session) {
	session.AddHandler(func(innerSession *discordgo.Session, event *discordgo.InteractionCreate) {
		if event.Type == discordgo.InteractionMessageComponent {
			innerSession.ChannelMessageDelete(event.ChannelID, event.Message.ID)
			switch event.MessageComponentData().CustomID {
			case "button_yes":
				{
					interactionResponse := NewInteractionResponseToSpecificCommand(2, "acceptraiderrequest|The raider`s attendance will be updated...")
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error occured while trying to respond to an officer as he/her tried to accept a raiders recalculation of attendance, during the function UseSlashCommand()", err.Error())
					}
					dataSlice := strings.Split(event.Message.Content, " ")
					lenDataSlice := len(dataSlice)
					oldCharName := dataSlice[lenDataSlice-1]
					newCharName := dataSlice[2]
					newRaiderProfile, _ := GetRaiderProfile(newCharName)
					oldRaiderProfile, _ := GetRaiderProfile(oldCharName)
					RecalculateRaiderAttendance(oldRaiderProfile, newRaiderProfile)
					channelWithUser, err := innerSession.UserChannelCreate(newRaiderProfile.ID)
					if err != nil {
						_, err = innerSession.ChannelMessageSend(channelOfficer, fmt.Sprintf("Raider attendance updated for raider %s but the bot was not able to contact the raider and inform he/her about this...", newCharName))
						if err != nil {
							WriteErrorLog(fmt.Sprintf("An error occured while trying to send a message in the officer channel %s, during the function UseSlashCommand()", channelOfficer), err.Error())
							break
						}
					}
					for x := 0; x <= 1; x++ {
						switch x {
						case 0:
							{
								innerSession.ChannelMessageSend(channelWithUser.ID, "This is simply an informal message - The request for adding an old main to new main`s raider-attendance has been ACCEPTED")
							}
						case 1:
							{
								innerSession.ChannelMessageSend(channelWithUser.ID, fmt.Sprintf("Your mains total raid-count has been increased from %d => %d", newRaiderProfile.AttendanceInfo["guildStart"].RaidCount, oldRaiderProfile.AttendanceInfo["guildStart"].RaidCount+newRaiderProfile.AttendanceInfo["guildStart"].RaidCount))
							}
						}
					}
					return
				}
			case "button_no": {
					innerSession.ChannelMessageDelete(event.ChannelID, event.Message.ID)
					interactionResponse := NewInteractionResponseToSpecificCommand(1, "mynewmain|The request has been denied by an admin")
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
					if err != nil {
						WriteErrorLog("An error occured while trying to respond to user %s, while denying a raider from merging an old main, during the function UseSlashCommand()", err.Error())
					}
					return
				}
			}
		} else if event.Type != discordgo.InteractionApplicationCommand && event.GuildID != "" && event.User.ID != innerSession.State.User.ID {
			return
		}
		// Acknowledge the interaction immediately to avoid the "Application did not respond" error

		var user *discordgo.User
		if event.Member != nil && event.Member.User != nil {
			user = event.Member.User // Guild interaction
		} else if event.User != nil {
			user = event.User // DM interaction
		} else {
			WriteErrorLog("Failed to retrieve user information in UseSlashCommand()", "User is nil")
			return
		}
		if user != nil {
			userID := user.ID
			interactionData := event.ApplicationCommandData()
			if CheckForOfficerRank(userID, innerSession) {
				//userName := user.Username
				if len(interactionData.Options) == 0 {
					switch interactionData.Name {
					case "resetraidcache":
						{
							template := DeepCopyInteractionResponse(slashCommandGeneralResponses["successMessage"].Response)
							currentLogsBase := GetWarcraftLogsData(mapOfWarcaftLogsQueries["guildLogsRaidIDs"])
							lenCurrentLogBase := 0
							if allLogs, ok := currentLogsBase["logs"].([]logsBase); ok {
								lenCurrentLogBase = len(allLogs)
							} else {
								WriteErrorLog(fmt.Sprintf("Was not possible to find any valid guild raids using the function GetWarcraftLogsData() on slash command raidreset from user %s", userID), "During function UseSlashCommand()")

							}

							embedMessages := []*discordgo.MessageEmbed{
								{
									Title: fmt.Sprintf("Status about your last command %s", crackedBuiltin),
									Color: greenColor,
									Fields: []*discordgo.MessageEmbedField{
										{
											Name: "Status from command: resetraidcache",
											Value: fmt.Sprintf(`
											Number of logs found posted by the guild on Warcraftlogs: **%d**
											Estimated time to finish command in hrs/min/s: **%s**
										`, lenCurrentLogBase, FormatDurationFromMilliseconds(float64(lenCurrentLogBase*4*1000))),
										},
									},
								},
								{
									Title: "WARNING",
									Color: yellowColor,
									Fields: []*discordgo.MessageEmbedField{
										{
											Name:  "Please wait for resetraidcache to finish",
											Value: fmt.Sprintf("While the slash command 'resetraidcache' is running, no data returned from any command should be considered valid %s", antiCrackedBuiltin),
										},
									},
								},
							}

							template.Data.Embeds = append(template.Data.Embeds, embedMessages...)

							err := innerSession.InteractionRespond(event.Interaction, template)
							if err != nil {
								WriteErrorLog("An error occured while sending a success message inside the function UseSlashCommand()", err.Error())
							}
							allLogs := GetAllWarcraftLogsRaidData(false, false, "")
							var finalEmbeds []*discordgo.MessageEmbed

							if len(allLogs) == 0 {
								finalTemplate := DeepCopyInteractionResponse(slashCommandGeneralResponses["errorMessage"].Response)
								finalTemplate.Data.Embeds = append(finalTemplate.Data.Embeds, NewInteractionResponseToSpecificCommand(0, "resetraidcache|0 Warcraft logs retrieved. This is the last message...").Data.Embeds...)
								finalEmbeds = finalTemplate.Data.Embeds
							} else {
								finalTemplate := DeepCopyInteractionResponse(slashCommandGeneralResponses["successMessage"].Response)
								finalTemplate.Data.Embeds = append(finalTemplate.Data.Embeds, NewInteractionResponseToSpecificCommand(2, fmt.Sprintf("resetraidcache|A total of %d logs has been successfully retrieved %s", len(allLogs), crackedBuiltin)).Data.Embeds...)
								finalEmbeds = finalTemplate.Data.Embeds
							}

							_, err = innerSession.ChannelMessageSendEmbeds(event.ChannelID, finalEmbeds)
							if err != nil {
								WriteErrorLog(fmt.Sprintf("An error occured while trying to send a final response to the user %s using slash command resetraidcache, inside of the function UseSlashCommand()", userID), err.Error())
							} else {
								WriteInformationLog(fmt.Sprintf("A message successfully sent to the user %s during the function UseSlashCommand()", userID), "Successfully sent embed message")
							}
						}
					case "deletechannelcontent":
						{
							DeleteMessagesInBulk(event.ChannelID, innerSession)
						}
					case "updateweeklyattendance": 
						{
							returnString := AddWeeklyRaiderAttendance()
							returnCode := 0
							if !strings.Contains(returnString, "error") && !strings.Contains(returnString, "cannot") {
								returnCode = 2

							}
							interactionResponse := NewInteractionResponseToSpecificCommand(returnCode, fmt.Sprintf("updateweeklyattendance|%s", returnString))
							err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
							if err != nil {
								WriteErrorLog("An error occured while trying to sent error response to user %s with slash command updateweekyattendance, during the function UseSlashCommand()", err.Error())
							}
						}
					case "promotetrial": {
							fmt.Println("WE REACH HERE")

						}
					}
				} else {
					time.Now()
					switch interactionData.Name {
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
										responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary month|An error inside the bot, please report this error to %s\n\nError: %s", strings.Split(officerGMWyzz, "/")[1], err.Error()))
										err := innerSession.InteractionRespond(event.Interaction, &responseError)
										if err != nil {
											WriteErrorLog("An error occured while trying to sent a error message from the user from slash command /raidsummary month, during the function UseSlashCommand()", err.Error())
										}
										break
									}
									if len(interactionResponses[0].Data.Embeds) > 10 {
										responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary month|An error inside the bot, please report this error to %s\n\nError: The number of embeds exceeded 10, raids must be merged incorrectly %s", strings.Split(officerGMWyzz, "/")[1], crackedBuiltin))
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
											responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary month|An error inside the bot, please report this error to %s\n\nError: %s", strings.Split(officerGMWyzz, "/")[1], err.Error()))
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
											responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary month|An error inside the bot, please report this error to %s\n\nError: The number of embeds exceeded 10, raids must be merged incorrectly %s", strings.Split(officerGMWyzz, "/")[1], antiCrackedBuiltin))
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
											responseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("raidsummary dayorweek|An error inside the bot, please report this error to %s\n\nError: Even though check through inner function checkUserBoolResponseFlag(), the command response is nil.. %s", strings.Split(officerGMWyzz, "/")[1], antiCrackedBuiltin))
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
											fmt.Println("DO WE REACH HERE?????")
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

						}
					case "simplemessage":
						{
							if interactionData.Options == nil {
								WriteInformationLog("The user %s did not provide any value to the command, this is crucial for the code to run, during function UseSlashCommand() breaing from loop...", "User input is nil")
								break
							}

							if stringValue, ok := interactionData.Options[0].Value.(string); ok {
								copyTemplate := DeepCopyInteractionResponse(slashCommandAdminCenter["simplemessagefromthebot"].Responses["messagetouser"].Response)
								sliceOfResponseString := SeperateAnyTagsInMessage(event.ChannelID, stringValue)
								embedFields := []*discordgo.MessageEmbedField{}
								embedField := &discordgo.MessageEmbedField{
									Value: stringValue,
								}
								embedFields = append(embedFields, embedField)
								copyTemplate.Data.Embeds[0].Fields = embedFields
								_, err := innerSession.ChannelMessageSendEmbeds(event.ChannelID, copyTemplate.Data.Embeds)
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
								interaction := NewInteractionResponseToSpecificCommand(2, "simplemessage|Right away")
								err = innerSession.InteractionRespond(event.Interaction, &interaction)
								if err != nil {
									WriteErrorLog("An error occured while trying to affirm the slash command 'simplemessage', during the function UseSlashCommand()", err.Error())
								}

							} else {
								WriteErrorLog(fmt.Sprintf("It was not possible to convert the value %s to string, this is crucial for this slash command, will break early...", interactionData.Options[0].Value), "During function UseSlashCommand()")
							}
						}
					case "messageusingsignups":
						{
							//playersToNotify := []string{}
							if noSubOption, _ := CheckUserBoolResponseFlag(interactionData.Options, "notsigned"); noSubOption {
								currentRaiders := RetrieveUsersInRole([]string{roleTrial, roleRaider}, session)
								mapOfMissingSignUp := make(map[string]bool)
								mapOfPlayersToContact := make(map[string]bool)

								signUpsAsInterface, _, _ := RetriveRaidHelperEvent(session, false)

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
								fmt.Println(len(mapOfPlayersToContact))
								for id, _ := range mapOfPlayersToContact {
									fmt.Println("USER:", ResolvePlayerID(id, innerSession))
								}
							} else {
								//playersToNotify = RetrieveUsersInRole([]string{roleRaider, roleTrial}, innerSession)
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
						/*
							case "aboutme":
								{
									switch interactionData.Options[0].Name {
									case "logs":
										{
											if len(interactionData.Options[0].Options) > 1 {
												interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("aboutme logs/Please do not provide more than 1 argument at a time %s", antiCrackedBuiltin))
												err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
												if err != nil {
													WriteErrorLog("An error occured while trying to sent an error message as interaction response using slash command 'aboutme logs' Inside of function UseSlashCommand()", err.Error())
												}
												break
											}
											subOptionName := ""
											if interactionData.Options[0].Options == nil {
												subOptionName = "newest"
											} else {
												subOptionName = interactionData.Options[0].Options[0].Name
											}

											switch subOptionName {
											case "newest":
												{
													currentLogs, err := CapturePointInTimeRaidLogData("", true)
													if err != nil {
														errorString := ""
														if err.Error() == "unexpected end of JSON input" {
															errorString = "No data found, please ask an officer to run command 'resetraidcache'"
														} else {
															errorString = err.Error()
														}
														interactionResponse := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("aboutme logs newest/%s", errorString))
														err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
														if err != nil {
															WriteErrorLog("An error occured while trying to sent an error message as interaction response using slash command 'aboutme logs newest' Inside of function UseSlashCommand()", err.Error())
														}
														break
													}

													if len(currentLogs) < 1 {
														interactionResponse := NewInteractionResponseToSpecificCommand(1, fmt.Sprintf("aboutme logs newest/%s", "No data found - Please ask an officer to run /resetraidcache"))
														err := innerSession.InteractionRespond(event.Interaction, &interactionResponse)
														if err != nil {
															WriteErrorLog("An error occured while trying to sent an error message as interaction response using slash command 'aboutme logs newest' Inside of function UseSlashCommand()", err.Error())
														}
														break
													}
													logsInScope := logAllData{}
													newestDate, err := time.Parse(timeLayout, currentLogs[0].RaidStartTimeString)
													if err != nil {
														WriteErrorLog(fmt.Sprintf("An error occured while trying to parse the string to time during the slash command aboutme logs newest from user %s", userID), err.Error())
														break
													}
													fmt.Println("CURRENT TIME_", newestDate)
													for _, log := range currentLogs {
														logTime, _ := time.Parse(timeLayout, log.RaidStartTimeString)
														fmt.Println("NEW LOG TIME:", logTime, log.RaidStartTimeString)
														if logTime.After(newestDate) {
															fmt.Println("NEW LOG FOUND TO USE:", log.RaidStartTimeString)
															logsInScope = log
														}
													}
													fmt.Println("LOG TO GO WITH:", logsInScope.RaidTitle)
												}
											}
										}
									}
								}
						*/
					}
				}
			} else if CheckForRaiderRank(userID, innerSession) && strings.Contains("myattendance,mymissedraids,mynewmain,myraiderperformance", interactionData.Name) {
				newRaiderProfile, errString := GetRaiderProfile(userID)
				if errString != "" {
					interactionResponseError := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("myattendance|%s", errString))
					err := innerSession.InteractionRespond(event.Interaction, &interactionResponseError)
					if err != nil {
						WriteErrorLog("An error ocurred while trying to sent the response error to the user from slash command /myattendance, during the function UseSlashCommand()", err.Error())
					}

				}
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
								fmt.Sprintf("<@%s>", strings.Split(officerGMWyzz, "/")[0]),
								fmt.Sprintf("<@%s>", strings.Split(officerRogue, "/")[0]),
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
				case "myraiderperformance": {
						raiderProfilesBytes := CheckForExistingCache(raiderProfilesCachePath)
						raiders := []raiderProfile{}
						json.Unmarshal(raiderProfilesBytes, &raiders)
						raidsBytes := CheckForExistingCache(raidAllDataPath)
						raids := []logAllData{}
						json.Unmarshal(raidsBytes, &raids)
						CalculateRaiderPerformance(raiders, raids)
					}
				}
			} else {
				commandName := interactionData.Name
				response := NewInteractionResponseToSpecificCommand(0, fmt.Sprintf("%s|You do not have the required permissions to run this command %s", commandName, antiCrackedBuiltin))
				err := innerSession.InteractionRespond(event.Interaction, &response)
				if err != nil {
					WriteErrorLog(fmt.Sprintf("An error occured while trying to send a permission denied response to user %s using the slash command %s during the function UseSlashCommand()", userID, commandName), err.Error())
				} else {
					WriteInformationLog(fmt.Sprintf("The user %s was informed that he / her does not have access to the command %s, message: Permission denied, during the function UseSlashCommand()", userID, commandName), "User denied slashcommand")
				}
			}
		} else {
			fmt.Println("DO WE REACH HERE???")
		}
	})
}

func GetRaiderProfile(raiderName string) (raiderProfile, string) {
	raiderName = strings.Replace(raiderName, ">", "", -1)
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
				if cachedRaider.AttendanceInfo["oneMonth"].RaidCount != raider.AttendanceInfo["oneMonth"].RaidCount || cachedRaider.AttendanceInfo["guildStart"].RaidCount != raider.AttendanceInfo["guildStart"].RaidCount {
					WriteInformationLog("The raider: %s 's attendance will be updated, during the function ReadWriteRaiderCache()", "Updating RaiderProfile Attendance")
					cachedRaiderProfiles.Raiders[x].AttendanceInfo = raider.AttendanceInfo
				}

				if cachedRaider.IsOfficer != raider.IsOfficer {
					cachedRaiderProfiles.Raiders[x].IsOfficer = raider.IsOfficer
				}

				if cachedRaider.ID == "" {
					cachedRaiderProfiles.Raiders[x].ID = raider.ID
				}

				if cachedRaider.DateJoinedGuild == "" {
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
	for name, mapOfRaid := range mapOfRaids {
		fmt.Println("NAME OF RAID:", name, len(mapOfRaid))
	}

	for raid, allLogs := range mapOfRaids {
		fmt.Println("SLICE OF RAID NAMES:", raid, len(allLogs))
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
					raiders[x].GuildRole = roleRaider
				} else if strings.Contains(strings.Join(raiders[x].DiscordRoles, ","), roleTrial) && raiders[x].GuildRole != roleRaider {
					raiders[x].GuildRole = roleTrial
				} else {
					raiders[x].GuildRole = rolePuggie
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
			if raider.GuildRole == roleRaider || raider.GuildRole == roleTrial || raider.GuildRole != rolePuggie {
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


func AddWeeklyRaiderAttendance() string{
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
	/*
	newestMainRaid := logAllData{}
	for _, log := range currentRaids {
		if time.UnixMilli(log.RaidStartUnixTime).AddDate(0, 0, 7).After(time.Now()) && log.PlayersCount > 32 {
			newestMainRaid = log
			fmt.Println("LOG FOUND:", newestMainRaid.RaidTitle)
			break
		}
	}
*/
	newRaiders := CalculateAttendance(currentRaiders.Raiders, currentRaids)

	ReadWriteRaiderProfiles(newRaiders, false)
	return fmt.Sprintf("A total of %d raider-profiles has had attendance updated", len(newRaiders)) //len(updatedRaiderProfiles))
}

func RecalculateRaiderAttendance(raiderOld raiderProfile, raiderNew raiderProfile) raiderProfile { //raiderOld = raiders first main
	mapOfNewAttendance := map[string]attendance{}
	for period, attendanceNew := range raiderNew.AttendanceInfo {
		//fmt.Println("PERIOD:", period, "RAID COUNT:", attendanceNew.RaidCount)
		if attendanceNew.RaidCount != 0 {
			newRaidCount := raiderOld.AttendanceInfo[period].RaidCount + attendanceNew.RaidCount
			totalRaidsInPeriod := math.Floor(float64(attendanceNew.RaidCount) / attendanceNew.RaidProcent * 100)
				//mt.Println("COUNT:", newRaidCount, "TOTAL", totalRaidsInPeriod, "OLD NAME:", raiderOld.MainCharName, "NEW", raiderNew.MainCharName)
			attendance := attendance{
				RaidCount:   newRaidCount,
				RaidProcent: math.Floor(float64(newRaidCount) / totalRaidsInPeriod * 100), //math.Floor(float64(count) / float64(amountOfRaids) * 100),
				RaidsMissed: raiderNew.AttendanceInfo[period].RaidsMissed,
				MainRaid:    true,
			}
			mapOfNewAttendance[period] = attendance
		} else {
			fmt.Println("DO WE REACH HERE??", period)
			mapOfNewAttendance[period] = raiderOld.AttendanceInfo[period]
		}
		
	}
	newRaiderProfile := raiderNew
	newRaiderProfile.AttendanceInfo = mapOfNewAttendance
	mapOfMainSwitch := make(map[string]bool)
	if newRaiderProfile.MainSwitch == nil {
		mapOfMainSwitch[raiderOld.MainCharName] = true
		newRaiderProfile.MainSwitch = mapOfMainSwitch
	} else {
		newRaiderProfile.MainSwitch[raiderOld.MainCharName] = true
	}
	ReadWriteRaiderProfiles([]raiderProfile{newRaiderProfile}, false)
	return newRaiderProfile
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
	
	fmt.Println("LEN OF RAIDS", len(filteredRaids))
	for _, raid := range filteredRaids {
		fmt.Println("RAID TITLE:", raid.RaidTitle)
	}
		
	return nil
} 

func CalculateRaiderPerformance (raiders []raiderProfile, raids []logAllData) []raiderProfile {
	//raiderPerformanceProfiles := []logsRaider{}
	SortOnlyMainRaids(0, raids, true)
	return nil
}

func CalculateAttendance(raiders []raiderProfile, raids []logAllData) []raiderProfile {
	mapOfAttendancePeriods := map[string]time.Time{
		"oneMonth":   time.Now().AddDate(0, -1, 0),
		"twoMonth":   time.Now().AddDate(0, -2, 0),
		"threeMonth": time.Now().AddDate(0, -3, 0),
		"guildStart": GuildStartTime,
	}
	mapOfMainRaidPeriods := make(map[string][]logAllData)
	mapOfPeriodsAndRaidsTotal := make(map[string]int)
	for periodName, raidTime := range mapOfAttendancePeriods {
		mapOfUniqueRaids := make(map[string]bool)
		mainRaidsInPeriod := []logAllData{}
		altRaidsInPeriod := []logAllData{}
		raidCount := 0
		for _, raid := range raids {
			if len(raid.RaidNames) == 0 {
				continue
			}
			raidCurrentTime := time.UnixMilli(raid.RaidStartUnixTime)
			raidTitleSlice := strings.Split(raid.RaidTitle, " ")
			raidDate := strings.Split(raid.RaidStartTimeString, ",")[0]
			raidYear := strings.Split(raid.RaidStartTimeString, " ")[2]
			raidKey := fmt.Sprintf("%s-%s", raidDate, raidYear)
			if raidCurrentTime.After(raidTime) && !mapOfUniqueRaids[raidKey] && !strings.Contains("zg,ony,aq20", strings.ToLower(raidTitleSlice[0]))  {
				//fmt.Println("RAID KEY", raidKey)
				raidCount++
				mainRaidsInPeriod = append(mainRaidsInPeriod, raid)
				mapOfUniqueRaids[raidKey] = true
			} else if raidCurrentTime.After(raidTime) && !mapOfUniqueRaids[raidKey] {
				altRaidsInPeriod = append(altRaidsInPeriod, raid)
				mapOfUniqueRaids[raidKey] = true
			}
		}
		fmt.Println("PERIOD", periodName, raidCount)
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

	for x, raider := range raiders {
		for _, log := range raids {
			for _, player := range log.Players {
				if raider.MainCharName == player.Name {
					raiders[x].DateJoinedGuild = log.RaidStartTimeString
				}
			}
		}
	}

	return raiders
}

func AddWeeklyAttendance() {

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
				raiderProfile := raiderProfile{
					MainCharName: raider.Name,
					ClassInfo: class{
						IngameClass: raider.ClassName,
						Name:        raider.Name,
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
	fmt.Println("LOGS FOUND IN CACHE;", len(allLogDataFromCache))
	for _, logData := range allLogDataFromCache {
		timeOfRaid, _ := time.Parse(timeLayout, logData.RaidStartTimeString)
		if !timeOfRaid.Before(firstPossibleTime) {
			if len(logData.RaidNames) == 1 && (strings.Contains(strings.ToLower(logData.RaidNames[0]), "ony") || strings.Contains(strings.ToLower(logData.RaidNames[0]), "zul")) && onlyMainRaid {
				fmt.Println("SKIPPING RAID:", logData.RaidTitle)
				continue
			}
			returnLogData = append(returnLogData, logData)
		}
	}

	for _, test := range returnLogData {
		fmt.Println("RAID NAME FOUND:;", test.RaidTitle, test.RaidTimeString)
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

func ReadCache(userID string, filePath string) []any {
	var sliceOfCache []any
	if cacheBytes := CheckForExistingCache(filePath); len(cacheBytes) > 0 {
		err := json.Unmarshal(cacheBytes, &sliceOfCache)
		if err != nil {
			WriteErrorLog("An error occured while trying to unmarshal json - type is not []any:", err.Error())
			return []any{}
		}

		if userID == "" {
			return sliceOfCache
		}

		for _, slice := range sliceOfCache {
			if mapOfSlice, ok := slice.(map[string]any); ok {
				for name, value := range mapOfSlice {
					if strings.ToLower(name) == "id" {
						if value.(string) == userID {
							sliceOfCache = append(sliceOfCache, slice)
							return sliceOfCache
						}
					}
				}
			} else {
				WriteInformationLog(fmt.Sprintf("Not possible to convert cache of value: %s", slice), "Cannot convert type to map")
			}
		}
	}

	return []any{}
}

//func WriteCache()

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
	botSession, err := discordgo.New("Bot " + "ZOe2ZAgfTaDjCH6Tx_M4Kp-brX3-g31s")
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
	return channelName.Name
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
			if strings.Contains(event.Content, " VISIT ") && event.Author.ID == session.State.User.ID {
				patternUserID := regexp.MustCompile(`<@(\d+)>`)
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
			} else if currentClassSlice := strings.Split(event.Content, " "); len(currentClassSlice) > 2 && !strings.Contains(event.Content, stage3) && event.Author.ID == session.State.User.ID && !strings.Contains(event.Content, stage4) && !strings.Contains(event.Content, stage5) && !strings.Contains(event.Content, stage6) && !strings.Contains(event.Content, stage7) && !strings.Contains(event.Content, stage9) {
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
			} else if strings.Contains(event.Content, stage3) && !mapOfMessageReactions[event.ID] {
				allClassEmojies := GetEmojies(1, []string{"class"})
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
			}
		}
	})

	botSession.AddHandler(func(session *discordgo.Session, event *discordgo.MessageReactionAdd) {
		channel, _ := session.Channel(event.ChannelID)
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
					raidProfile.GuildRole = emojie.Name
					raidProfile.GuildRoleEmojieID = emojie.ID
					raidProfile.LastTimeChangedString = GetTimeString()

					finalMessageSlice := []string{}
					switch emojie.ShortName {
					case "puggie":
						{
							botSession.GuildMemberRoleAdd(serverID, raidProfile.ID, rolePuggie)
							finalMessageSlice = append(finalMessageSlice, fmt.Sprintf("Server role puggie assigned, thank you for joining <Hardened> as a pug\nIn order to sign up do the following:\n\n1. Click on the SR link on the sign-up page and select ur SRs\n2. Click sign-up to the raid itself.\n3. If you try to sign up first you will get an error\n\nPlease see the raid-signups:\n\nMC <#%s> and BWL <#%s>", channelSignUp, channelSignUpBWL)) //Must be changed when we run pug raids
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
								classLeader = officerGMWyzz
							}
							botSession.GuildMemberRoleAdd(serverID, raidProfile.ID, classDiscordRole)
							finalMessageSlice = append(finalMessageSlice, fmt.Sprintf("Server role trial assigned, welcome to the <Hardened> Team! %s\n\n**Loot rules are different for trials** - As a general rule of thumb, biggest items are off-limits for first raid minimum.\n\nYour new class leader: @ %s\n\nRaid-leader: %s\n\nGet familiar with your class channel: <#%s>\n\nRaid sign-ups channels: MC <#%s> BWL <#%s>\n\nGuild general chat channel: <#%s>", crackedBuiltin, strings.Split(classLeader, "/")[1], strings.Split(officerGMWyzz, "/")[1], classChannel, channelSignUp, channelSignUpBWL, channelGeneral))

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
	}
	return returnWarcraftQueries
}

func deepCopyMap(original map[string]any) map[string]any {
	copyBytes, _ := json.Marshal(original) // Serialize to JSON
	var copyMap map[string]any
	json.Unmarshal(copyBytes, &copyMap) // Deserialize back to a new map
	return copyMap
}

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
	req.Header.Set("Authorization", "Bearer "+"eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.eyJhdWQiOiI5ZTM2ODA5MS01YzJjLTQ1OTMtYTk5Ny1jMDg3OTA0MjBlMDgiLCJqdGkiOiIxYTdhZDEyYzU3OWJjNGVjNTJlZDlhMmZiMmFjMDUxMDdiZTMwNGZkZjVkODU1ZjVmMTMzZDBmYzcxYTJmYTU2M2I0OWRhY2ZmNDFkYzQwYiIsImlhdCI6MTczOTU3ODY1Ny41MTg3NTQsIm5iZiI6MTczOTU3ODY1Ny41MTg3NTcsImV4cCI6MTc3MDY4MjY1Ny40OTQ1NzIsInN1YiI6IjExNDE5MjAiLCJzY29wZXMiOlsidmlldy11c2VyLXByb2ZpbGUiLCJ2aWV3LXByaXZhdGUtcmVwb3J0cyJdfQ.pPX5ynLfm04J2qBf09fNhCEdxk3ZkY_3J2ufhEdjBod3NZ0uCaiU5U_UTQ-esxuQqg3NIVQEXqZF2vNVlnDwUMJd0ZKMp91CAUgDFOrGa-2nZCpf529qYnz1pnxn5dJ_KmCSLWOyL6bOCohCmp0-lt0d67YOOLnWL9vqh2faD6MEVxyNe2kLwPx1h1b__-XvBOiZxhMtaLO7p3TO8Zb8wj3wRRpT90Ym-cWF8L0hM8lfTbSXagvwdI5o8mtjK4XkMFEEqknBAsMyK_X2pl1LPsCAdJZ0AJY0GzO401dSjd8I-AHmh-TK_IL2E9G5deFcGoWjxWltkX75NhM0r7O-CkO3qrOsPVjNrCiVPy6oXHqnVeqCYdg0bzMkuaOdZBbohlQ2TpqU-Fvb3XWYIW34N2ONJwVClqA3lKR0PpW2_Anrew02mj3awg5sxwi9asC59UBTDNlA3Nfp0lUALVowrtBaFTI-QcT3ASovuwXyc3tgulEE-MURo_aRglcpNg4LXaUUMgoKKiS2iQumObe0ltxCmnqq76gZjqBzt215_WOWgpSs0qZuLdE7DzxGkSRWdd1XWskA-8Aq6nxhDM5ITo2t3gWqJ32VmPyURVd0A8F0hY2lK3-2hblyW_H9TUeuzMhFx55EVXPP8dTD0PgSnKSkeLqMdL099OdF7xkh62E")

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
		bytes, _ := json.Marshal(returnMap)
		os.WriteFile("MYJSON.json", bytes, 0644)
		mapOfLogs := map[string]any{
			"logs": returnMap,
		}
		return mapOfLogs
	} else if strings.Contains(contentOfQuery, "GetEncounterInfo") {
		mapOfEncounter := map[string]any{
			"encounter": returnMap,
		}
		return mapOfEncounter
	}
	return nil
}

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

func CheckForOfficerRank(playerID string, botSession *discordgo.Session) bool {
	playerRoles, err := botSession.GuildMember(serverID, playerID)
	if err != nil {
		WriteErrorLog(fmt.Sprintf("An error occured while trying to retrieve discord stats about player: %s inside function CheckForOfficerRank()", playerID), err.Error())
	}
	if strings.Contains(strings.Join(playerRoles.Roles, ","), roleOfficer) {
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
	InformPlayerDirectly(fmt.Sprintf("**--------------------------------------------------------------**\n\nA new raid of type: %s has been created: %s\n\nSecondary raid resets on: %s\n\nNext main raid resets on: %s", raidShortName, customEvents[0].Date, resetTimeOfSecondaryRaid.String(), nextMainRaidDate.String()), strings.Split(officerGMWyzz, "/")[0], session)
}

func SeperateAnyTagsInMessage(channelID string, messageValue string) []string {
	returnMessageTagsSlice := []string{}
	//returnMessageTagsSlice = append(returnMessageTagsSlice, messageValue)

	if messageValueSplit := strings.Split(messageValue, " "); len(messageValueSplit) > 1 || len(messageValueSplit) == 1 && strings.Contains(messageValueSplit[0], "@") {
		for _, stringPartOfMessage := range messageValueSplit {
			if strings.Contains(stringPartOfMessage, "@") {
				patternUserID := regexp.MustCompile(`<@&?\d+>`)
				fmt.Println("Do wer rerdch here? BEFIORE", returnMessageTagsSlice)
				returnMessageTagsSlice = append(returnMessageTagsSlice, patternUserID.FindAllString(stringPartOfMessage, -1)...)
				fmt.Println("FOUND:", patternUserID.FindAllString(stringPartOfMessage, -1))
				fmt.Println("Do wer rerdch here? AFTER", returnMessageTagsSlice)
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

func RetriveRaidHelperEvent(botSession *discordgo.Session, test bool) ([]map[string]any, string, string) {
	newRaidURL := ""
	raidHelperEventID := ""
	if test {
		newRaidURL = fmt.Sprintf("https://raid-helper.dev/api/v3/servers/%s/events", serverID)
	} else {
		retrieveDiscordMessages, err := botSession.ChannelMessages(channelSignUp, 50, "", "", "")
		if err != nil {
			WriteErrorLog("An error occured while trying to retrieve the last 50 messages from signups inside function RetrieveRaidHelperEvent()", err.Error())
			return []map[string]any{}, "", ""
		}
		for x := len(retrieveDiscordMessages) - 1; x >= 0; x-- {
			if retrieveDiscordMessages[x].Author.ID == raidHelperId {
				raidHelperEventID = retrieveDiscordMessages[x].ID
				break
			}
		}
		newRaidURL = fmt.Sprintf("https://raid-helper.dev/api/v2/events/%s", raidHelperEventID)
	}
	var signUps []map[string]any
	var raidHelperResponse any
	getSignupData, _ := http.NewRequest("GET", newRaidURL, nil)
	getSignupData.Header = http.Header{
		"Authorization": {mapOfTokens["Raid_helper"]},
		"Content-Type":  {"application/json"},
	}
	client := &http.Client{}
	data, err := client.Do(getSignupData)
	if err != nil {
		WriteErrorLog("An error occured while trying run the HTTP request to raid-helper", err.Error())
		if test {
			log.Fatal("Since the raid-helper API denies the GET request the program will stop...")
		}
	}
	if test {
		return []map[string]any{}, "", ""
	}
	// Read and print the response body
	body, err := io.ReadAll(data.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	json.Unmarshal(body, &raidHelperResponse)
	softresId := ""
	if mapOfSignUps, ok := raidHelperResponse.(map[string]any); ok {
		for name, value := range mapOfSignUps {
			fmt.Println("NAME OF ATTRIBUTE:", name)
			if name == "signUps" {
				marshaler, _ := json.Marshal(value)
				sliceOfSignUpMaps := []map[string]any{}
				json.Unmarshal(marshaler, &sliceOfSignUpMaps)
				signUps = sliceOfSignUpMaps
			} else if name == "softresId" {
				softresId = value.(string)
			}
		}
	}

	return signUps, fmt.Sprintf("https://discord.com/events/%s/%s", serverID, raidHelperEventID), fmt.Sprintf("https://softres.it/raid/%s", softresId)
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

	if len(KeyvaultConfig.Tokens) < 2 {
		log.Fatal("An error occured while counting the amount of secrets available: Less than 2 token definitions are present", err.Error())
	}

	KeyvaultConfig = keyvault{
		URI:    fmt.Sprintf("%s.vault.azure.net", KeyvaultConfig.URI),
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

/*
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
*/
/*
func StorageAccountAppendBlob(blobName string, storageClient *azblob.Client, appendData []byte) {
	//Runs first time as part of app starting when being deployed (The appendblob wont be there yet)
	// ✅ Manually construct the AppendBlobClient
	appendBlobClient, err := azblob.NewAppendBlobClientFromConnectionString(
		storageClient.URL(), containerName, blobName, storageClient.Credential(), nil)
	if err != nil {
		log.Fatalf("Failed to create AppendBlobClient: %v", err)
	}
}
*/
