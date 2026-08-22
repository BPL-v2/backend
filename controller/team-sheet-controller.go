package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TeamSheetController struct {
	teamSheetService service.TeamSheetService
}

func NewTeamSheetController() *TeamSheetController {
	return &TeamSheetController{
		teamSheetService: service.NewTeamSheetService(),
	}
}

func setupTeamSheetController() []RouteInfo {
	e := NewTeamSheetController()
	basePath := "events/:event_id/teams/:team_id/sheet"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getTeamSheetHandler(), Authenticated: true, RequiresTeamSelf: true},
		{Method: "PUT", Path: "", HandlerFunc: e.saveMyTeamSheetEntryHandler(), Authenticated: true, RequiresTeamSelf: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetTeamSheet
// @Description Fetches the team sheet (planned characters/roles) for your team for an event
// @Tags team
// @Security BearerAuth
// @Produce json
// @Param event_id path int true "Event Id"
// @Param team_id path int true "Team Id"
// @Success 200 {array} TeamSheetEntry
// @Router /events/{event_id}/teams/{team_id}/sheet [get]
func (e *TeamSheetController) getTeamSheetHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid team ID"})
			return
		}
		entries, err := e.teamSheetService.GetEntriesForTeam(event.Id, teamId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(entries, toTeamSheetEntryResponse))
	}
}

// @id SaveMyTeamSheetEntry
// @Description Creates or updates your own row in your team's sheet for an event
// @Tags team
// @Accept json
// @Security BearerAuth
// @Produce json
// @Param event_id path int true "Event Id"
// @Param team_id path int true "Team Id"
// @Param entry body TeamSheetEntryUpdate true "Sheet entry data"
// @Success 200 {object} TeamSheetEntry
// @Router /events/{event_id}/teams/{team_id}/sheet [put]
func (e *TeamSheetController) saveMyTeamSheetEntryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		userId, ok := getUserId(c)
		if !ok {
			c.AbortWithStatus(401)
			return
		}
		var update TeamSheetEntryUpdate
		if err := c.ShouldBindJSON(&update); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		entry, err := e.teamSheetService.GetEntryForUser(event.Id, userId)
		if err != nil {
			entry = &repository.TeamSheetEntry{EventId: event.Id, UserId: userId}
		}
		entry.CharacterName = update.CharacterName
		entry.Role = update.Role
		entry.SecondaryRole = update.SecondaryRole
		entry.Ascendancy = update.Ascendancy
		entry.MainSkill = update.MainSkill
		entry.BuildNotes = update.BuildNotes
		entry.PobUrl = update.PobUrl
		entry.Realm = update.Realm
		entry.UniquesNeeded = update.UniquesNeeded
		entry.Altars = update.Altars
		entry.LookingForGroup = update.LookingForGroup

		saved, err := e.teamSheetService.SaveEntry(entry)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toTeamSheetEntryResponse(saved))
	}
}

func toTeamSheetEntryResponse(entry *repository.TeamSheetEntry) *TeamSheetEntry {
	return &TeamSheetEntry{
		User:            toNonSensitiveUserResponse(entry.User),
		CharacterName:   entry.CharacterName,
		Role:            entry.Role,
		SecondaryRole:   entry.SecondaryRole,
		Ascendancy:      entry.Ascendancy,
		MainSkill:       entry.MainSkill,
		BuildNotes:      entry.BuildNotes,
		PobUrl:          entry.PobUrl,
		Realm:           entry.Realm,
		UniquesNeeded:   entry.UniquesNeeded,
		Altars:          entry.Altars,
		LookingForGroup: entry.LookingForGroup,
	}
}

type TeamSheetEntry struct {
	User            *NonSensitiveUser `json:"user" binding:"required"`
	CharacterName   *string           `json:"character_name"`
	Role            *string           `json:"role"`
	SecondaryRole   *string           `json:"secondary_role"`
	Ascendancy      *string           `json:"ascendancy"`
	MainSkill       *string           `json:"main_skill"`
	BuildNotes      *string           `json:"build_notes"`
	PobUrl          *string           `json:"pob_url"`
	Realm           *string           `json:"realm"`
	UniquesNeeded   *string           `json:"uniques_needed"`
	Altars          *string           `json:"altars"`
	LookingForGroup bool              `json:"looking_for_group"`
}

type TeamSheetEntryUpdate struct {
	CharacterName   *string `json:"character_name"`
	Role            *string `json:"role"`
	SecondaryRole   *string `json:"secondary_role"`
	Ascendancy      *string `json:"ascendancy"`
	MainSkill       *string `json:"main_skill"`
	BuildNotes      *string `json:"build_notes"`
	PobUrl          *string `json:"pob_url"`
	Realm           *string `json:"realm"`
	UniquesNeeded   *string `json:"uniques_needed"`
	Altars          *string `json:"altars"`
	LookingForGroup bool    `json:"looking_for_group"`
}
