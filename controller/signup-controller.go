package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SignupController struct {
	signupService service.SignupService
	groupService  service.SignupGroupService
	userService   service.UserService
	teamService   service.TeamService
	eventService  service.EventService
	discordClient *client.LocalDiscordClient
}

func NewSignupController() *SignupController {
	return &SignupController{
		signupService: service.NewSignupService(),
		groupService:  service.NewSignupGroupService(),
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
		{Method: "POST", Path: "/self/group", HandlerFunc: e.createGroupHandler(), Authenticated: true},
		{Method: "POST", Path: "/self/group/join", HandlerFunc: e.joinGroupHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/self/group", HandlerFunc: e.leaveGroupHandler(), Authenticated: true},
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
		resp := toSignupResponse(signup)
		e.attachGroup(resp, signup, event.MaxGroupSize)
		c.JSON(200, resp)
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
		if signupCreate.ExpectedPlaytime < 0 || signupCreate.ExpectedPlaytime > 24 {
			c.JSON(400, gin.H{"error": "Fuck you frisbee"})
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
		signup, err = e.signupService.SaveSignup(signup)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		resp := toSignupResponse(signup)
		e.attachGroup(resp, signup, event.MaxGroupSize)
		c.JSON(201, resp)
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
		groupKeys, err := e.groupService.GetGroupKeysForEvent(event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
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
			if key, ok := groupKeys[signup.UserId]; ok {
				resp.GroupKey = &key
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

type SignupGroup struct {
	Key     string              `json:"key" binding:"required"`
	Members []*NonSensitiveUser `json:"members" binding:"required"`
	MaxSize int                 `json:"max_size" binding:"required"`
	Locked  bool                `json:"locked" binding:"required"`
}

type GroupJoin struct {
	GroupKey string `json:"group_key" binding:"required"`
}

type Signup struct {
	Group            *SignupGroup      `json:"group"`
	User             *NonSensitiveUser `json:"user" binding:"required"`
	Timestamp        time.Time         `json:"timestamp" binding:"required" format:"date-time"`
	ExpectedPlaytime int               `json:"expected_playtime" binding:"required"`
	TeamId           *int              `json:"team_id"`
	IsTeamLead       bool              `json:"team_lead" binding:"required"`
	NeedsHelp        bool              `json:"needs_help"`
	WantsToHelp      bool              `json:"wants_to_help"`
	Extra            *string           `json:"extra"`
}

type ExtendedSignup struct {
	GroupKey         *string           `json:"group_key"`
	User             *NonSensitiveUser `json:"user" binding:"required"`
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
	ExpectedPlaytime int     `json:"expected_playtime" binding:"required"`
	NeedsHelp        bool    `json:"needs_help"`
	WantsToHelp      bool    `json:"wants_to_help"`
	Extra            *string `json:"extra"`
}

func toSignupResponse(signup *repository.Signup) *Signup {
	if signup == nil {
		return nil
	}

	return &Signup{
		User:             toNonSensitiveUserResponse(signup.User),
		Timestamp:        signup.Timestamp,
		ExpectedPlaytime: signup.ExpectedPlayTime,
		NeedsHelp:        signup.NeedsHelp,
		WantsToHelp:      signup.WantsToHelp,
		Extra:            signup.Extra,
	}
}

// attachGroup adds the group the signup's user is part of (if any) to the response.
func (e *SignupController) attachGroup(resp *Signup, signup *repository.Signup, maxGroupSize int) {
	key, err := e.groupService.GetGroupKeyForUser(signup.EventId, signup.UserId)
	if err != nil || key == nil {
		return
	}
	members, err := e.groupService.GetGroupMembers(signup.EventId, *key)
	if err != nil {
		return
	}
	group := &SignupGroup{Key: *key, MaxSize: maxGroupSize, Members: make([]*NonSensitiveUser, 0, len(members))}
	for _, member := range members {
		group.Members = append(group.Members, toNonSensitiveUserResponse(member.User))
		group.Locked = group.Locked || member.Locked
	}
	resp.Group = group
}

func groupErrorStatus(err error) int {
	switch err {
	case repository.ErrGroupNotFound:
		return 404
	case repository.ErrGroupFull, repository.ErrGroupLocked, repository.ErrAlreadyInGroup, repository.ErrNotInGroup:
		return 400
	default:
		return 500
	}
}

// groupContext resolves the event and user for group endpoints and makes sure the user has a signup.
func (e *SignupController) groupContext(c *gin.Context) (*repository.Event, *repository.User, *repository.Signup) {
	event := getEvent(c)
	if event == nil {
		return nil, nil, nil
	}
	user, err := e.userService.GetUserFromAuthHeader(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Not authenticated"})
		return nil, nil, nil
	}
	if event.MaxGroupSize < 2 {
		c.JSON(400, gin.H{"error": "Group signups are not enabled for this event"})
		return nil, nil, nil
	}
	signup, err := e.signupService.GetSignupForUser(user.Id, event.Id)
	if err != nil {
		c.JSON(400, gin.H{"error": "You need to sign up before you can use groups"})
		return nil, nil, nil
	}
	if _, err := e.teamService.GetTeamForUser(event.Id, user.Id); err == nil {
		c.JSON(400, gin.H{"error": repository.ErrGroupLocked.Error()})
		return nil, nil, nil
	}
	signup.User = user
	return event, user, signup
}

// @id CreateSignupGroup
// @Description Creates a new group containing the authenticated user and returns the signup
// @Tags signup
// @Produce json
// @Security BearerAuth
// @Success 201 {object} Signup
// @Param event_id path int true "Event Id"
// @Router /events/{event_id}/signups/self/group [post]
func (e *SignupController) createGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event, user, signup := e.groupContext(c)
		if signup == nil {
			return
		}
		if _, err := e.groupService.CreateGroup(event.Id, user.Id, event.MaxGroupSize); err != nil {
			c.JSON(groupErrorStatus(err), gin.H{"error": err.Error()})
			return
		}
		resp := toSignupResponse(signup)
		e.attachGroup(resp, signup, event.MaxGroupSize)
		c.JSON(201, resp)
	}
}

// @id JoinSignupGroup
// @Description Joins the group with the given key and returns the signup
// @Tags signup
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Signup
// @Param event_id path int true "Event Id"
// @Param groupJoin body GroupJoin true "Group"
// @Router /events/{event_id}/signups/self/group/join [post]
func (e *SignupController) joinGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event, user, signup := e.groupContext(c)
		if signup == nil {
			return
		}
		var body GroupJoin
		if err := c.BindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if err := e.groupService.JoinGroup(event.Id, user.Id, strings.TrimSpace(body.GroupKey), event.MaxGroupSize); err != nil {
			c.JSON(groupErrorStatus(err), gin.H{"error": err.Error()})
			return
		}
		resp := toSignupResponse(signup)
		e.attachGroup(resp, signup, event.MaxGroupSize)
		c.JSON(200, resp)
	}
}

// @id LeaveSignupGroup
// @Description Removes the authenticated user from their group
// @Tags signup
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Signup
// @Param event_id path int true "Event Id"
// @Router /events/{event_id}/signups/self/group [delete]
func (e *SignupController) leaveGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event, user, signup := e.groupContext(c)
		if signup == nil {
			return
		}
		if err := e.groupService.LeaveGroup(event.Id, user.Id); err != nil {
			c.JSON(groupErrorStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toSignupResponse(signup))
	}
}
