package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type LadderController struct {
	ladderService    service.LadderService
	characterService service.CharacterService
	userService      service.UserService
	teamService      service.TeamService
	signupService    service.SignupService
	activityService  service.ActivityService
}

func NewLadderController(poeClient *client.PoEClient) *LadderController {
	return &LadderController{
		ladderService:    service.NewLadderService(),
		characterService: service.NewCharacterService(poeClient),
		userService:      service.NewUserService(),
		teamService:      service.NewTeamService(),
		signupService:    service.NewSignupService(),
		activityService:  service.NewActivityService(),
	}
}

func setupLadderController(poeClient *client.PoEClient) []RouteInfo {
	c := NewLadderController(poeClient)
	baseUrl := "events/:event_id"
	routes := []RouteInfo{
		{Method: "GET", Path: "/ladder", HandlerFunc: c.getLadderHandler()},
		{Method: "GET", Path: "/characters", HandlerFunc: c.GetCharactersForEvent()},
		{Method: "GET", Path: "/team/:team_id/atlas", HandlerFunc: c.getAtlasesForEvent(), Authenticated: true, RequiresTeamSelf: true},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id GetLadder
// @Description Get the ladder for an event
// @Tags ladder
// @Produce json
// @Param event_id path int true "Event ID"
// @Param hours_after_event_start query int false "only show ladder entries from this timestamp after event start"
// @Success 200 {array} LadderEntry
// @Router /events/{event_id}/ladder [get]
func (c *LadderController) getLadderHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		ladder, err := c.ladderService.GetLadderForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		hours_after_event_start := ctx.Query("hours_after_event_start")
		cutoff := event.EventEndTime
		if hours_after_event_start != "" {
			hours, err := time.ParseDuration(hours_after_event_start + "h")
			if err != nil {
				ctx.JSON(400, gin.H{"error": "Invalid hours_after_event_start"})
				return
			}
			cutoff = event.EventStartTime.Add(hours)
		}
		characters, err := c.characterService.GetCharactersForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		usersWithTeam, err := c.userService.GetUsersWithTeamForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		characterStats, err := c.characterService.GetCharacterStatsForEvent(event.Id, cutoff)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		lastActivities, err := c.activityService.GetLatestActiveTimestampsForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, toLadderResponse(usersWithTeam, ladder, characters, characterStats, lastActivities))
	}
}

// @id GetCharactersForEvent
// @Description Get all characters for an event
// @Tags characters
// @Produce json
// @Param event_id path int true "Event ID"
// @Success 200 {array} BplCharacter
// @Router /events/{event_id}/characters [get]
func (c *LadderController) GetCharactersForEvent() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		characters, err := c.characterService.GetCharactersForEvent(event.Id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(200, utils.Map(characters, toCharacterResponse))
	}
}

// @id GetTeamAtlasesForEvent
// @Description Get atlas trees for your team for an event
// @Tags atlas
// @Produce json
// @Security BearerAuth
// @Param event_id path int true "Event ID"
// @Param team_id path int true "Team ID"
// @Success 200 {array} Atlas
// @Router /events/{event_id}/team/{team_id}/atlas [get]
func (c *LadderController) getAtlasesForEvent() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		event := getEvent(ctx)
		if event == nil {
			return
		}
		teamId, err := strconv.Atoi(ctx.Param("team_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "Invalid team ID"})
			return
		}
		atlases, err := c.characterService.GetTeamAtlasesForEvent(event.Id, teamId)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(200, toAtlasResponses(atlases))
	}
}

type LadderEntry struct {
	UserId      int     `json:"user_id" binding:"required"`
	PoEAccount  string  `json:"poe_account" binding:"required"`
	DiscordName string  `json:"discord_name" binding:"required"`
	DiscordId   string  `json:"discord_id" binding:"required"`
	TwitchName  *string `json:"twitch_name" omitempty:"true"`
	TeamId      int     `json:"team_id" binding:"required"`

	CharacterName    string  `json:"character_name" binding:"required"`
	CharacterId      string  `json:"character_id" binding:"required"`
	Pantheon         bool    `json:"pantheon" binding:"required"`
	AtlasPoints      int     `json:"atlas_points" binding:"required"`
	AscendancyPoints int     `json:"ascendancy_points" binding:"required"`
	ItemIndexes      []int32 `json:"item_indexes" binding:"required"`
	Voidstones       int     `json:"voidstones" binding:"required"`

	Level           int    `json:"level" binding:"required"`
	XP              int64  `json:"xp" binding:"required"`
	Ascendancy      string `json:"ascendancy" binding:"required"`
	Mainskill       string `json:"main_skill" binding:"required"`
	DPS             int64  `json:"dps" binding:"required"`
	EHP             int32  `json:"ehp" binding:"required"`
	PhysMaxHit      int32  `json:"phys_max_hit" binding:"required"`
	EleMaxHit       int32  `json:"ele_max_hit" binding:"required"`
	HP              int32  `json:"hp" binding:"required"`
	Mana            int32  `json:"mana" binding:"required"`
	ES              int32  `json:"es" binding:"required"`
	Armour          int32  `json:"armour" binding:"required"`
	Evasion         int32  `json:"evasion" binding:"required"`
	MovementSpeed   int32  `json:"movement_speed" binding:"required"`
	AttackBlock     int8   `json:"attack_block" binding:"required"`
	SpellBlock      int8   `json:"spell_block" binding:"required"`
	LowestEleRes    int8   `json:"lowest_ele_res" binding:"required"`
	HighLevelFlasks int8   `json:"high_level_flasks" binding:"required"`

	LastActive int64 `json:"last_active" binding:"required"`

	DelveDepth int `json:"delve_depth" binding:"required"`
	Rank       int `json:"rank" binding:"required"`
}

type Atlas struct {
	UserId       int           `json:"user_id" binding:"required"`
	PrimaryIndex int           `json:"primary_index" binding:"required"`
	Trees        map[int][]int `json:"trees" binding:"required"`
}

func toAtlasResponses(atlases []*repository.AtlasTree) []*Atlas {
	userAtlases := make(map[int]map[int]*repository.AtlasTree)
	for _, atlas := range atlases {
		if userAtlases[atlas.UserID] == nil {
			userAtlases[atlas.UserID] = make(map[int]*repository.AtlasTree)
		}
		userAtlases[atlas.UserID][atlas.Index] = atlas
	}

	mappedAtlases := make([]*Atlas, 0)
	for userId, trees := range userAtlases {
		atlas := &Atlas{
			UserId: userId,
			Trees:  make(map[int][]int),
		}
		for index, tree := range trees {
			atlas.Trees[index] = tree.Nodes
		}
		primaryIndex := 0
		latestTimestamp := time.Time{}
		for index, tree := range trees {
			if tree.Timestamp.After(latestTimestamp) {
				latestTimestamp = tree.Timestamp
				primaryIndex = index
			}
		}
		atlas.PrimaryIndex = primaryIndex
		mappedAtlases = append(mappedAtlases, atlas)
	}
	return mappedAtlases
}

func toLadderResponse(usersWithTeam map[int]*repository.UserWithTeam, ladderEntries []*repository.LadderEntry, characters []*repository.Character, stats map[string]*repository.CharacterPob, lastActivities map[int]time.Time) []*LadderEntry {
	response := make([]*LadderEntry, 0, len(ladderEntries))
	ladderMap := make(map[string]*repository.LadderEntry)
	for _, entry := range ladderEntries {
		ladderMap[entry.Character] = entry
	}
	statsMap := make(map[string]*repository.CharacterPob)
	for _, stat := range stats {
		statsMap[stat.CharacterId] = stat
	}

	for _, character := range characters {
		stats := statsMap[character.Id]
		if character.UserId == nil {
			continue
		}
		user := usersWithTeam[*character.UserId]
		if user == nil {
			continue
		}
		resp := &LadderEntry{
			CharacterName:    character.Name,
			CharacterId:      character.Id,
			Pantheon:         character.Pantheon,
			AtlasPoints:      character.AtlasPoints,
			AscendancyPoints: character.AscendancyPoints,
			UserId:           *character.UserId,
			Voidstones:       len(character.VoidStones),

			PoEAccount:  user.PoEAccount,
			DiscordName: user.DiscordName,
			DiscordId:   user.DiscordId,
			TwitchName:  &user.TwitchName,
			TeamId:      user.TeamId,
		}
		if stats != nil {
			resp.Level = stats.Level
			resp.XP = stats.XP
			resp.Ascendancy = stats.Ascendancy
			resp.Mainskill = stats.MainSkill
			resp.DPS = stats.DPS
			resp.EHP = stats.EHP
			resp.PhysMaxHit = stats.PhysMaxHit
			resp.EleMaxHit = stats.EleMaxHit
			resp.HP = stats.HP
			resp.Mana = stats.Mana
			resp.ES = stats.ES
			resp.Armour = stats.Armour
			resp.Evasion = stats.Evasion
			resp.MovementSpeed = stats.MovementSpeed
			resp.ItemIndexes = stats.Items
			resp.AttackBlock = stats.AttackBlock
			resp.SpellBlock = stats.SpellBlock
			resp.LowestEleRes = stats.LowestEleRes
			resp.HighLevelFlasks = stats.HighIlevelFlasks
		} else {
			resp.Level = character.Level
			resp.Ascendancy = character.Ascendancy
			resp.Mainskill = character.MainSkill
		}
		if lastActive, ok := lastActivities[*character.UserId]; ok {
			resp.LastActive = lastActive.Unix()
		}
		if ladderEntry, ok := ladderMap[character.Name]; ok {
			resp.DelveDepth = ladderEntry.Delve
		}
		response = append(response, resp)
	}
	return response
}
