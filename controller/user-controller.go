package controller

import (
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	userService      service.UserService
	eventService     service.EventService
	characterService service.CharacterService
	atlasService     service.AtlasService
}

func NewUserController(poeClient *client.PoEClient) *UserController {
	return &UserController{
		userService:      service.NewUserService(),
		eventService:     service.NewEventService(),
		characterService: service.NewCharacterService(poeClient),
		atlasService:     service.NewAtlasService(),
	}
}

func setupUserController(poeClient *client.PoEClient) []RouteInfo {
	e := NewUserController(poeClient)
	basePath := ""
	routes := []RouteInfo{
		{Method: "GET", Path: "/events/:event_id/users", HandlerFunc: e.getUsersForEventHandler()},
		{Method: "GET", Path: "/events/:event_id/users/:user_id/atlas", HandlerFunc: e.getAtlasProgressionHandler()},
		{Method: "GET", Path: "/users", HandlerFunc: e.getAllUsersHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/users/:user_id", HandlerFunc: e.getUserByIdHandler()},
		{Method: "GET", Path: "/users/self", HandlerFunc: e.getUserHandler(), Authenticated: true},
		{Method: "PATCH", Path: "/users/self", HandlerFunc: e.updateUserHandler(), Authenticated: true},
		{Method: "PATCH", Path: "/users/:user_id", HandlerFunc: e.changePermissionsHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "POST", Path: "/users/remove-auth", HandlerFunc: e.removeAuthHandler(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetAllUsers
// @Description Fetches all users
// @Tags user
// @Produce json
// @Success 200 {array} User
// @Security BearerAuth
// @Router /users [get]
func (e *UserController) getAllUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := e.userService.GetAllUsers("OauthAccounts")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(users, toUserResponse))
	}
}

// @id ChangePermissions
// @Description Changes the permissions of a user
// @Tags user
// @Accept json
// @Produce json
// @Param user_id path int true "User Id"
// @Param permissions body repository.Permissions true "Permissions"
// @Success 200 {object} User
// @Security BearerAuth
// @Router /users/{user_id} [patch]
func (e *UserController) changePermissionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := strconv.Atoi(c.Param("user_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		var permissions repository.Permissions
		if err := c.BindJSON(&permissions); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.ChangePermissions(userId, permissions)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toUserResponse(user))
	}
}

// @id GetUser
// @Description Fetches the authenticated user
// @Tags user
// @Produce json
// @Success 200 {object} User
// @Security BearerAuth
// @Router /users/self [get]
func (e *UserController) getUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		c.JSON(200, toUserResponse(user))
	}
}

// @id RemoveAuth
// @Description Removes an authentication provider from the authenticated user
// @Tags user
// @Produce json
// @Param provider query string true "Provider"
// @Success 200 {object} User
// @Security BearerAuth
// @Router /users/remove-auth [post]
func (e *UserController) removeAuthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := repository.Provider(c.Request.URL.Query().Get("provider"))
		if provider == "" {
			c.JSON(400, gin.H{"error": "No provider specified"})
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		user, err = e.userService.RemoveProvider(user, provider)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toUserResponse(user))
	}
}

// @id GetUsersForEvent
// @Description Fetches all users for an event
// @Tags user
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {object} map[int][]MinimalUser
// @Router /events/{event_id}/users [get]
func (e *UserController) getUsersForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		// loading event again to have preloads
		event, err := e.eventService.GetEventById(event.Id, "Teams.Users.OauthAccounts")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		teamUsers := make(map[int][]*MinimalUser)
		for _, team := range event.Teams {
			teamUsers[team.Id] = make([]*MinimalUser, 0)
			for _, user := range team.Users {
				teamUsers[team.Id] = append(teamUsers[team.Id], toMinimalUserResponse(user))
			}
		}
		c.JSON(200, teamUsers)
	}
}

// @id GetUserById
// @Description Fetches a user by ID
// @Tags user
// @Produce json
// @Param user_id path int true "User Id"
// @Success 200 {object} User
// @Router /users/{user_id} [get]
func (e *UserController) getUserByIdHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, err := strconv.Atoi(c.Param("user_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user, err := e.userService.GetUserById(userId, "OauthAccounts")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "User not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toMinimalUserResponse(user))
	}
}

// @id UpdateUser
// @Description Updates the authenticated users display name
// @Tags user
// @Accept json
// @Produce json
// @Param user body UserUpdate true "User"
// @Success 200 {object} User
// @Security BearerAuth
// @Router /users/self [patch]
func (e *UserController) updateUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Not authenticated"})
			return
		}
		var userUpdate UserUpdate
		if err := c.BindJSON(&userUpdate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user.DisplayName = userUpdate.DisplayName
		user, err = e.userService.SaveUser(user)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, toUserResponse(user))
	}
}

// @id GetAtlasProgression
// @Description Fetches the atlas progression for a user in an event
// @Tags user
// @Produce json
// @Param event_id path int true "Event Id"
// @Param user_id path int true "User Id"
// @Success 200 {object} []AtlasProgression
// @Router /events/{event_id}/users/{user_id}/atlas [get]
func (e *UserController) getAtlasProgressionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		userId, err := strconv.Atoi(c.Param("user_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		atlasProgression, err := e.atlasService.GetAtlasesForEventAndUser(userId, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(atlasProgression, toAtlasProgression))
	}
}

type UserUpdate struct {
	DisplayName string `json:"display_name" binding:"required"`
}

type User struct {
	Id                   int        `json:"id" binding:"required"`
	DisplayName          string     `json:"display_name" binding:"required"`
	AcountName           *string    `json:"account_name"`
	DiscordId            *string    `json:"discord_id"`
	DiscordName          *string    `json:"discord_name"`
	TwitchId             *string    `json:"twitch_id"`
	TwitchName           *string    `json:"twitch_name"`
	TokenExpiryTimestamp *time.Time `json:"token_expiry_timestamp" format:"date-time"`

	Permissions []repository.Permission `json:"permissions" binding:"required"`
}

type NonSensitiveUser struct {
	Id          int     `json:"id" binding:"required"`
	DisplayName string  `json:"display_name" binding:"required"`
	AcountName  *string `json:"account_name"`
	DiscordId   *string `json:"discord_id"`
	DiscordName *string `json:"discord_name"`
	TwitchId    *string `json:"twitch_id"`
	TwitchName  *string `json:"twitch_name"`
}

type MinimalUser struct {
	Id          int     `json:"id" binding:"required"`
	DisplayName string  `json:"display_name" binding:"required"`
	PoEName     *string `json:"poe_account_name"`
	DiscordName *string `json:"discord_name"`
	DiscordId   *string `json:"discord_id"`
}

func toUserResponse(user *repository.User) *User {
	response := &User{
		Id:          user.Id,
		DisplayName: user.DisplayName,
		Permissions: user.Permissions,
	}
	for _, oauth := range user.OauthAccounts {
		switch oauth.Provider {
		case repository.ProviderDiscord:
			response.DiscordId = &oauth.AccountId
			response.DiscordName = &oauth.Name
		case repository.ProviderTwitch:
			response.TwitchId = &oauth.AccountId
			response.TwitchName = &oauth.Name
		case repository.ProviderPoE:
			response.AcountName = &oauth.Name
			response.TokenExpiryTimestamp = &oauth.Expiry
		}
	}

	return response
}

func toNonSensitiveUserResponse(user *repository.User) *NonSensitiveUser {
	if user == nil {
		return nil
	}
	response := &NonSensitiveUser{
		Id:          user.Id,
		DisplayName: user.DisplayName,
	}
	for _, oauth := range user.OauthAccounts {
		switch oauth.Provider {
		case repository.ProviderDiscord:
			response.DiscordId = &oauth.AccountId
			response.DiscordName = &oauth.Name
		case repository.ProviderPoE:
			response.AcountName = &oauth.Name
		}
	}
	return response
}

func toMinimalUserResponse(user *repository.User) *MinimalUser {
	min := &MinimalUser{
		Id:          user.Id,
		DisplayName: user.DisplayName,
	}
	for _, oauth := range user.OauthAccounts {
		switch oauth.Provider {
		case repository.ProviderDiscord:
			min.DiscordName = &oauth.Name
			min.DiscordId = &oauth.AccountId
		case repository.ProviderPoE:
			min.PoEName = &oauth.Name
		}
	}
	return min
}

type AtlasProgression struct {
	Timestamp int64 `json:"timestamp"`
	Index     int   `json:"index"`
	Nodes     []int `json:"nodes"`
}

func toAtlasProgression(atlas *repository.AtlasTree) *AtlasProgression {
	return &AtlasProgression{
		Timestamp: atlas.Timestamp.Unix(),
		Index:     atlas.Index,
		Nodes:     atlas.Nodes,
	}
}
