package controller

import (
	"bpl/client"
	"bpl/cron"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type GuildStashController struct {
	guildStashService service.GuildStashService
	userService       service.UserService
	objectiveService  service.ObjectiveService
	eventService      service.EventService
	poeClient         *client.PoEClient
}

func NewGuildStashController(PoEClient *client.PoEClient) *GuildStashController {
	return &GuildStashController{
		guildStashService: service.NewGuildStashService(PoEClient),
		userService:       service.NewUserService(),
		objectiveService:  service.NewObjectiveService(),
		eventService:      service.NewEventService(),
		poeClient:         PoEClient,
	}
}

func setupGuildStashController(PoEClient *client.PoEClient) []RouteInfo {
	e := NewGuildStashController(PoEClient)
	basePath := ""
	routes := []RouteInfo{

		{Method: "GET", Path: "/:event_id/teams/:team_id/guild-stash", HandlerFunc: e.getGuildStashForUser(), Authenticated: true, RequiresTeamSelf: true},
		{Method: "GET", Path: "/:event_id/teams/:team_id/guild-stash/:stash_id", HandlerFunc: e.getGuildStashTab(), Authenticated: true, RequiresTeamSelf: true},
		{Method: "PATCH", Path: "/:event_id/teams/:team_id/guild-stash/:stash_id", HandlerFunc: e.switchStashFetch(), Authenticated: true, RequiresTeamLeader: true},
		{Method: "POST", Path: "/:event_id/teams/:team_id/guild-stash/:stash_id/update", HandlerFunc: e.updateStashTab(), Authenticated: true, RequiresTeamLeader: true},

		{Method: "GET", Path: "/:event_id/guilds", HandlerFunc: e.getGuilds()},
		{Method: "PUT", Path: "/:event_id/guilds/:guildId", HandlerFunc: e.saveGuild(), Authenticated: true},
		{Method: "GET", Path: "/:event_id/guilds/:guildId/stash-history", HandlerFunc: e.getLogEntriesForGuild(), Authenticated: true},
		{Method: "POST", Path: "/:event_id/guilds/:guildId/stash-history", HandlerFunc: e.addHistory(), Authenticated: true},
		{Method: "GET", Path: "/:event_id/guilds/:guildId/stash-history/latest_timestamp", HandlerFunc: e.getLatestTimestampForUser(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetGuilds
// @Description Get all guilds for current event with their respective team ids
// @Tags guild-stash
// @Produce json
// @Security BearerAuth
// @Param eventId path int true "Event Id"
// @Success 200 {array} Guild
// @Router  /{eventId}/guilds [get]
func (e *GuildStashController) getGuilds() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		guilds, err := e.guildStashService.GetGuildsForEvent(event)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(guilds, toGuild))
	}
}

// @id SaveGuild
// @Description Saves a guild for the current event
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param guildId path int true "Guild Id"
// @Param guild body Guild true "Guild"
// @Success 200 {object} Guild
// @Router  /{eventId}/guilds/{guildId} [put]
func (e *GuildStashController) saveGuild() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		guildId, err := strconv.Atoi(c.Param("guildId"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid guild id"})
			return
		}
		existingGuild, err := e.guildStashService.GetGuildById(guildId, event.Id)

		teamUser, _, err := e.userService.GetTeamForUser(c, event)
		if err != nil || (existingGuild != nil && existingGuild.TeamId != teamUser.TeamId) || !teamUser.IsTeamLead {
			c.JSON(403, gin.H{"message": "Only team leads can modify guilds for their team"})
			return
		}

		var guild Guild
		if err := c.ShouldBindJSON(&guild); err != nil {
			fmt.Println("Error binding JSON:", err)
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}
		guild.Id = guildId
		guild.TeamId = teamUser.TeamId
		guild.EventId = event.Id
		model := guild.toModel()
		if err := e.guildStashService.SaveGuild(model); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toGuild(model))
	}
}

// @id GetLatestTimestampForUser
// @Description Fetches the latest timestamp for a user's guild stash history
// @Tags guild-stash
// @Produce json
// @Security BearerAuth
// @Param eventId path int true "Event Id"
// @Param guildId path int true "Guild Id"
// @Success 200 {object} GuildStashLogTimestampResponse
// @Router  /{eventId}/guilds/{guildId}/stash-history/latest_timestamp [get]
func (e *GuildStashController) getLatestTimestampForUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		guildId, err := strconv.Atoi(c.Param("guildId"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid guild id"})
			return
		}
		earliest, latest := e.guildStashService.GetLatestLogEntryTimestampForGuild(event, guildId)
		c.JSON(200, GuildStashLogTimestampResponse{
			Earliest:    earliest,
			Latest:      latest,
			LeagueStart: event.EventStartTime.Unix(),
			LeagueEnd:   event.EventEndTime.Unix(),
		})
	}
}

// @id AddGuildstashHistory
// @Description Adds a new entry to the guild stash history
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param guildId path int true "Guild Id"
// @Param guildStashChanges body GuildStashChangeResponse true "Request body"
// @Success 201 {object} AddGuildStashHistoryResponse
// @Router /{eventId}/guilds/{guildId}/stash-history [post]
func (e *GuildStashController) addHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		guildId, err := strconv.Atoi(c.Param("guildId"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid guild id"})
			return
		}
		existingGuild, err := e.guildStashService.GetGuildById(guildId, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		var body GuildStashChangeResponse
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}
		teamUser, _, err := e.userService.GetTeamForUser(c, event)
		if err != nil || !teamUser.IsTeamLead || existingGuild.TeamId != teamUser.TeamId {
			c.JSON(403, gin.H{"message": "Team lead access required"})
			return
		}
		events, err := e.eventService.GetAllEvents()
		if err != nil {
			fmt.Println("Error fetching events:", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		logEntries := body.toLogEntries(events, guildId)
		err = e.guildStashService.SaveGuildstashLogs(logEntries)
		if err != nil {
			fmt.Println("Error saving guild stash logs:", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, AddGuildStashHistoryResponse{NumberOfAddedEntries: len(logEntries)})
	}
}

// @id GetLogEntriesForGuild
// @Description Fetches log entries for a guild in an event
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param guildId path int true "Guild Id"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param username query string false "Name of the user doing the action (Make sure to replace the pound sign with a minus)"
// @Param itemname query string false "Name of the item (Can be partial)"
// @Param stashname query string false "Name of the stash tab"
// @Success 200 {array} GuildStashChangelog
// @Router /{eventId}/guilds/{guildId}/stash-history [get]
func (e *GuildStashController) getLogEntriesForGuild() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		guildId, err := strconv.Atoi(c.Param("guildId"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid guild id"})
			return
		}
		existingGuild, err := e.guildStashService.GetGuildById(guildId, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		teamUser, _, err := e.userService.GetTeamForUser(c, event)
		if err != nil || existingGuild.TeamId != teamUser.TeamId {
			c.JSON(403, gin.H{"message": "Team lead access required"})
			return
		}
		limit, err := getIntQueryParam(c, "limit")
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid limit"})
			return
		}
		offset, err := getIntQueryParam(c, "offset")
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid offset"})
			return
		}
		username := getStringQueryParam(c, "username")
		itemname := getStringQueryParam(c, "itemname")
		stashname := getStringQueryParam(c, "stashname")
		logEntries, err := e.guildStashService.GetLogs(event.Id, guildId, limit, offset, username, stashname, itemname)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(logEntries, toChangeLog))
	}
}

func getIntQueryParam(c *gin.Context, param string) (*int, error) {
	if v := c.Query(param); v != "" {
		intValue, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		return &intValue, nil
	}
	return nil, nil
}
func getStringQueryParam(c *gin.Context, param string) *string {
	if v := c.Query(param); v != "" {
		return &v
	}
	return nil
}

// @id GetGuildStashForUser
// @Description Fetches all guild stash tabs for a user
// @Tags guild-stash
// @Produce json
// @Security BearerAuth
// @Param eventId path int true "Event Id"
// @Param teamId path int true "Team Id"
// @Success 200 {array} GuildStashTab
// @Router /{eventId}/teams/{teamId}/guild-stash [get]
func (e *GuildStashController) getGuildStashForUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid team id"})
			return
		}
		tabs, err := e.guildStashService.GetGuildStashesForTeam(teamId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(tabs, toModel))
	}
}

// @id UpdateStashTab
// @Description Fetches current items for specific guild stash tab
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param teamId path int true "Team Id"
// @Param stash_id path string true "Stash Tab Id"
// @Success 204
// @Router /{eventId}/teams/{teamId}/guild-stash/{stash_id}/update [post]
func (e *GuildStashController) updateStashTab() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		stashId := c.Param("stash_id")
		tab, err := e.guildStashService.GetGuildStash(stashId, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		err = cron.NewFetchingService(ctx, event, e.poeClient).FetchGuildStashTab(tab)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
	}
}

type TabSwitchRequest struct {
	FetchEnabled  bool `json:"fetch_enabled"`
	PriorityFetch bool `json:"priority_fetch"`
}

// @id SwitchStashFetching
// @Description Enables fetching for a specific guild stash tab
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param teamId path int true "Team Id"
// @Param stash_id path string true "Stash Tab Id"
// @Param body body TabSwitchRequest true "Request body"
// @Success 204
// @Router /{eventId}/teams/{teamId}/guild-stash/{stash_id} [patch]
func (e *GuildStashController) switchStashFetch() gin.HandlerFunc {
	return func(c *gin.Context) {
		stashId := c.Param("stash_id")
		var req TabSwitchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid team id"})
			return
		}

		err = e.guildStashService.SwitchStashFetch(stashId, teamId, req.FetchEnabled, req.PriorityFetch)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
	}
}

// @id GetGuildStashTab
// @Description Fetches a specific guild stash tab
// @Tags guild-stash
// @Security BearerAuth
// @Produce json
// @Param eventId path int true "Event Id"
// @Param teamId path int true "Team Id"
// @Param stash_id path string true "Stash Tab Id"
// @Success 200 {object} StashTabWithCompletions
// @Router /{eventId}/teams/{teamId}/guild-stash/{stash_id}  [get]
func (e *GuildStashController) getGuildStashTab() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		stashId := c.Param("stash_id")
		tab, err := e.guildStashService.GetGuildStash(stashId, event.Id)
		if err != nil || tab.Raw == "" || tab.Raw == "{}" {
			fmt.Printf("Error fetching guild stash tab: %v\n", err)
			c.JSON(404, gin.H{"error": "stash tab not found"})
			return
		}
		parser, err := e.objectiveService.GetParser(event.Id, false)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.Status(200)
		c.JSON(200, toGGGModel(tab, parser))
	}
}

type GuildStashTab struct {
	Id            string    `json:"id" binding:"required"`
	Name          string    `json:"name" binding:"required"`
	Type          string    `json:"type" binding:"required"`
	Index         *int      `json:"index"`
	Color         *string   `json:"color"`
	ParentId      *string   `json:"parent_id"`
	FetchEnabled  bool      `json:"fetch_enabled" binding:"required"`
	PriorityFetch bool      `json:"priority_fetch" binding:"required"`
	LastFetch     time.Time `json:"last_fetch" binding:"required" format:"date-time"`
	UserIds       []int     `json:"user_ids" binding:"required"`
}

func toModel(tab *repository.GuildStashTab) *GuildStashTab {
	if tab == nil {
		return nil
	}
	return &GuildStashTab{
		Id:            tab.Id,
		Name:          tab.Name,
		Type:          tab.Type,
		Index:         tab.Index,
		Color:         tab.Color,
		ParentId:      tab.ParentId,
		FetchEnabled:  tab.FetchEnabled,
		PriorityFetch: tab.PriorityFetch,
		LastFetch:     tab.LastFetch,
		UserIds:       utils.Map(tab.UserIds, func(id int32) int { return int(id) }),
	}
}

func toGGGModel(tab *repository.GuildStashTab, parser *parser.ItemChecker) *StashTabWithCompletions {
	if tab == nil {
		return nil
	}
	model := &StashTabWithCompletions{}
	err := json.Unmarshal([]byte(tab.Raw), &model)
	if err != nil || model == nil {
		return nil
	}

	model.Name = tab.Name
	if model.Items != nil {
		items := make([]ItemWithCompletions, 0, len(*model.Items))
		for _, item := range *model.Items {
			if item.Item == nil {
				items = append(items, item)
				continue
			}
			completions := parser.CheckForCompletions(item.Item)
			if len(completions) > 0 {
				objectiveId := completions[0].ObjectiveId
				item.ObjectiveId = &objectiveId
			}
			items = append(items, item)
		}
		model.Items = &items
	}

	children := make([]StashTabWithCompletions, 0, len(tab.Children))
	for _, child := range tab.Children {
		childModel := toGGGModel(child, parser)
		if childModel != nil {
			children = append(children, *childModel)
		}
	}
	model.Children = &children
	return model
}

type AddGuildStashHistoryResponse struct {
	NumberOfAddedEntries int `json:"number_of_added_entries" binding:"required"`
}

type GuildStashLogTimestampResponse struct {
	Earliest    *int64 `json:"earliest"`
	Latest      *int64 `json:"latest"`
	LeagueStart int64  `json:"league_start" binding:"required"`
	LeagueEnd   int64  `json:"league_end" binding:"required"`
}

type GuildStashChangeResponse struct {
	Entries []struct {
		Id      string  `json:"id"`
		Time    int64   `json:"time"`
		League  string  `json:"league"`
		Stash   *string `json:"stash"`
		Item    string  `json:"item"`
		Action  string  `json:"action"`
		Account struct {
			Name string `json:"name"`
		} `json:"account"`
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"entries"`
	Truncated bool `json:"truncated"`
}

func (g *GuildStashChangeResponse) toLogEntries(events []*repository.Event, guildId int) []*repository.GuildStashChangelog {
	re := regexp.MustCompile(`^(\d+)× (.+)$`)
	eventMap := make(map[string]*repository.Event)
	for _, event := range events {
		eventMap[event.Name] = event
	}
	var logs []*repository.GuildStashChangelog
	for _, entry := range g.Entries {
		id, err := strconv.Atoi(entry.Id)
		if err != nil {
			continue
		}
		event, ok := eventMap[entry.League]
		if !ok {
			continue
		}
		number := 1
		itemName := entry.Item
		matches := re.FindStringSubmatch(entry.Item)
		if len(matches) == 3 {
			number, err = strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			itemName = matches[2]
		}

		logs = append(logs, &repository.GuildStashChangelog{
			Id:          id,
			Timestamp:   time.Unix(entry.Time, 0),
			GuildId:     guildId,
			EventId:     event.Id,
			StashName:   entry.Stash,
			AccountName: entry.Account.Name,
			Action:      repository.ActionFromString(entry.Action),
			Number:      number,
			ItemName:    itemName,
			X:           entry.X,
			Y:           entry.Y,
		})
	}
	return logs
}

type Action string

const (
	ActionAdded    Action = "added"
	ActionModified Action = "modified"
	ActionRemoved  Action = "removed"
)

type GuildStashChangelog struct {
	Timestamp   int64   `json:"timestamp" binding:"required"`
	AccountName string  `json:"account_name" binding:"required"`
	StashName   *string `json:"stash_name,omitempty"`
	ItemName    string  `json:"item_name" binding:"required"`
	Number      int     `json:"number" binding:"required"`
	Action      Action  `json:"action" binding:"required"`
}

type Guild struct {
	Id      int    `json:"id" binding:"required"`
	TeamId  int    `json:"team_id"`
	EventId int    `json:"event_id"`
	Name    string `json:"name" binding:"required"`
	Tag     string `json:"tag" binding:"required"`
}

func (t *Guild) toModel() *repository.Guild {
	if t == nil {
		return nil
	}
	return &repository.Guild{
		TeamId:  t.TeamId,
		Id:      t.Id,
		Name:    t.Name,
		Tag:     t.Tag,
		EventId: t.EventId,
	}
}

func toGuild(model *repository.Guild) *Guild {
	if model == nil {
		return nil
	}
	return &Guild{
		Id:      model.Id,
		TeamId:  model.TeamId,
		EventId: model.EventId,
		Name:    model.Name,
		Tag:     model.Tag,
	}
}

func toActionModel(action repository.Action) Action {
	switch action {
	case repository.ActionAdded:
		return ActionAdded
	case repository.ActionModified:
		return ActionModified
	case repository.ActionRemoved:
		return ActionRemoved
	default:
		return ActionAdded
	}
}

func toChangeLog(tab *repository.GuildStashChangelog) *GuildStashChangelog {
	if tab == nil {
		return nil
	}

	return &GuildStashChangelog{
		Timestamp:   tab.Timestamp.Unix(),
		AccountName: tab.AccountName,
		StashName:   tab.StashName,
		ItemName:    tab.ItemName,
		Number:      tab.Number,
		Action:      toActionModel(tab.Action),
	}
}

type StashTabWithCompletions struct {
	*client.StashTab
	Items    *[]ItemWithCompletions     `json:"items,omitempty"`
	Children *[]StashTabWithCompletions `json:"children,omitempty"`
}

type ItemWithCompletions struct {
	*client.Item
	ObjectiveId *int `json:"objective_id,omitempty"`
}
