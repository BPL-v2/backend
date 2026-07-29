package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SignupController struct {
	signupService service.SignupService
	userService   service.UserService
	teamService   service.TeamService
	eventService  service.EventService
	discordClient *client.LocalDiscordClient
}

func NewSignupController() *SignupController {
	return &SignupController{
		signupService: service.NewSignupService(),
		userService:   service.NewUserService(),
		teamService:   service.NewTeamService(),
		eventService:  service.NewEventService(),
	}
}

func setupSignupController() []RouteInfo {
	e := NewSignupController()
	e.discordClient = client.NewLocalDiscordClient()
	basePath := "/events/:event_id/signups"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getSignupsForEvent(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin, repository.PermissionManager}},
		{Method: "GET", Path: "/self", HandlerFunc: e.getPersonalSignupHandler(), Authenticated: true},
		{Method: "PUT", Path: "/self", HandlerFunc: e.createSignupHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/:user_id", HandlerFunc: e.deleteSignupHandler(), Authenticated: true, RequiresUserSelf: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetPersonalSignup
// @Description Fetches an authenticated user's signup for the event
// @Tags signup
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Signup
// @Param event_id path int true "Event Id"
// @Router /events/{event_id}/signups/self [get]
func (e *SignupController) getPersonalSignupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		signup, err := e.signupService.GetSignupForUser(user.Id, event.Id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Not signed up"})
			} else {
				c.JSON(400, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toSignupResponse(signup))
	}
}

// @id CreateSignup
// @Description Creates a signup for the authenticated user
// @Tags signup
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} Signup
// @Param event_id path int true "Event Id"
// @Param signupCreate body SignupCreate true "Signup"
// @Router /events/{event_id}/signups/self [put]
func (e *SignupController) createSignupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		if event.ApplicationStartTime.After(time.Now()) || event.ApplicationEndTime.Before(time.Now()) {
			c.JSON(400, gin.H{"error": "Applications are not open"})
			return
		}
		_, err = e.teamService.GetTeamForUser(event.Id, user.Id)
		if err == nil {
			c.JSON(400, gin.H{"error": "Cannot change signup after being added to a team"})
			return
		}
		discordId := user.GetDiscordId()
		if discordId == nil {
			c.JSON(400, gin.H{"error": "User does not have a linked Discord account"})
			return
		}
		isMember, err := e.discordClient.CheckForServerMemberShip(*discordId)
		if err != nil {
			fmt.Printf("Error checking Discord membership: %v\n", err)
		} else if !isMember {
			c.JSON(403, gin.H{"error": "User is not a member of the Discord server"})
			return
		}
		var signupCreate SignupCreate
		if err := c.BindJSON(&signupCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		signup, err := e.signupService.GetSignupForUser(user.Id, event.Id)
		if err != nil {
			signup = &repository.Signup{
				UserId:    user.Id,
				User:      user,
				EventId:   event.Id,
				Timestamp: time.Now(),
			}
		}
		signup.ExpectedPlayTime = signupCreate.ExpectedPlaytime
		signup.NeedsHelp = signupCreate.NeedsHelp
		signup.WantsToHelp = signupCreate.WantsToHelp
		signup.Extra = signupCreate.Extra
		signup.PartnerWish = signupCreate.PartnerAccountName
		signup, err = e.signupService.SaveSignup(signup)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, toSignupResponse(signup))
	}
}

// @id DeleteSignup
// @Description Deletes a user's signup for the event
// @Tags signup
// @Produce json
// @Security BearerAuth
// @Success 204
// @Param event_id path int true "Event Id"
// @Param user_id path int true "User Id"
// @Router /events/{event_id}/signups/{user_id} [delete]
func (e *SignupController) deleteSignupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		userId, err := strconv.Atoi(c.Param("user_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		err = e.signupService.RemoveSignupForUser(userId, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{})
	}
}

// @id GetEventSignups
// @Description Fetches all signups for the event
// @Tags signup
// @Security BearerAuth
// @Produce json
// @Success 200 {object} []ExtendedSignup
// @Param event_id path int true "Event Id"
// @Router /events/{event_id}/signups [get]
func (e *SignupController) getSignupsForEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		events, err := e.eventService.GetAllEvents()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		eventDurations := make(map[int]float64)
		for _, ev := range events {
			eventDurations[ev.Id] = ev.EventEndTime.Sub(ev.EventStartTime).Hours() / 24
		}
		signups, userEventActivityCount, highestCharacterLevels, err := e.signupService.GetExtendedSignupsForEvent(event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		signups = signups[:min(event.MaxSize, len(signups))]
		teamUsers, err := e.teamService.GetTeamUsersForEvent(event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		teamUsersMap := make(map[int]*repository.TeamUser, 0)
		for _, teamUser := range teamUsers {
			teamUsersMap[teamUser.UserId] = teamUser
		}
		signupsWithUsers := make([]*ExtendedSignup, 0)
		partnerMap := repository.GetSignupPartners(signups)
		for _, signup := range signups {
			playtimes := make(map[int]float64)
			for eventId, duration := range userEventActivityCount[signup.UserId] {
				playtimes[eventId] = duration.Hours() / eventDurations[eventId]
			}
			resp := &ExtendedSignup{
				User:                               toNonSensitiveUserResponse(signup.User),
				Timestamp:                          signup.Timestamp,
				ExpectedPlaytime:                   signup.ExpectedPlayTime,
				NeedsHelp:                          signup.NeedsHelp,
				WantsToHelp:                        signup.WantsToHelp,
				Extra:                              signup.Extra,
				PlaytimesInLastEventsPerDayInHours: playtimes,
				HighestCharacterLevels:             highestCharacterLevels[signup.UserId],
			}
			partnerSignup := partnerMap[signup.UserId]
			if partnerSignup != nil && partnerMap[partnerSignup.User.Id] != nil && partnerMap[partnerSignup.User.Id].UserId == signup.UserId {
				resp.PartnerId = &partnerSignup.UserId
				resp.Partner = toNonSensitiveUserResponse(partnerSignup.User)
			}
			if teamUser, ok := teamUsersMap[signup.UserId]; ok {
				resp.TeamId = &teamUser.TeamId
				resp.IsTeamLead = teamUser.IsTeamLead
			}
			signupsWithUsers = append(signupsWithUsers, resp)
		}
		c.JSON(200, signupsWithUsers)
	}
}

type Signup struct {
	User             *NonSensitiveUser `json:"user" binding:"required"`
	PartnerWish      *string
	Partner          *NonSensitiveUser `json:"partner"`
	PartnerId        *int              `json:"partner_id"`
	Timestamp        time.Time         `json:"timestamp" binding:"required" format:"date-time"`
	ExpectedPlaytime int               `json:"expected_playtime" binding:"required"`
	TeamId           *int              `json:"team_id"`
	IsTeamLead       bool              `json:"team_lead" binding:"required"`
	NeedsHelp        bool              `json:"needs_help"`
	WantsToHelp      bool              `json:"wants_to_help"`
	Extra            *string           `json:"extra"`
}

type ExtendedSignup struct {
	User             *NonSensitiveUser `json:"user" binding:"required"`
	PartnerWish      *string
	Partner          *NonSensitiveUser `json:"partner"`
	PartnerId        *int              `json:"partner_id"`
	Timestamp        time.Time         `json:"timestamp" binding:"required" format:"date-time"`
	ExpectedPlaytime int               `json:"expected_playtime" binding:"required"`
	TeamId           *int              `json:"team_id"`
	IsTeamLead       bool              `json:"team_lead" binding:"required"`
	NeedsHelp        bool              `json:"needs_help"`
	WantsToHelp      bool              `json:"wants_to_help"`
	Extra            *string           `json:"extra"`

	PlaytimesInLastEventsPerDayInHours map[int]float64 `json:"playtimes_in_last_events_per_day_in_hours" binding:"required"`
	HighestCharacterLevels             map[int]int     `json:"highest_character_levels" binding:"required"`
}

type SignupCreate struct {
	ExpectedPlaytime   int     `json:"expected_playtime" binding:"required"`
	NeedsHelp          bool    `json:"needs_help"`
	WantsToHelp        bool    `json:"wants_to_help"`
	PartnerAccountName *string `json:"partner_account_name"`
	Extra              *string `json:"extra"`
}

func toSignupResponse(signup *repository.Signup) *Signup {
	if signup == nil {
		return nil
	}

	return &Signup{
		User:             toNonSensitiveUserResponse(signup.User),
		PartnerWish:      signup.PartnerWish,
		Timestamp:        signup.Timestamp,
		ExpectedPlaytime: signup.ExpectedPlayTime,
		NeedsHelp:        signup.NeedsHelp,
		WantsToHelp:      signup.WantsToHelp,
		Extra:            signup.Extra,
	}
}
