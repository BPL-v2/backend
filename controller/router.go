package controller

import (
	"bpl/auth"
	"bpl/client"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
)

type RouteInfo struct {
	Method             string
	Path               string
	HandlerFunc        gin.HandlerFunc
	Authenticated      bool
	RequiredRoles      []repository.Permission
	RequiresUserSelf   bool
	RequiresTeamSelf   bool
	RequiresTeamLeader bool
}

func SetRoutes(r *gin.Engine) {
	cache := persistence.NewInMemoryStore(60 * time.Second)
	poeClient := client.NewPoEClient(10, false, 600)

	routes := make([]RouteInfo, 0)
	group := r.Group("/api")
	routes = append(routes, setupEventController()...)
	routes = append(routes, setupTeamController()...)
	routes = append(routes, setupObjectiveController(poeClient)...)
	routes = append(routes, setupOauthController()...)
	routes = append(routes, setupUserController(poeClient)...)
	routes = append(routes, setupScoringRuleController()...)
	routes = append(routes, setupSignupController()...)
	routes = append(routes, setupSubmissionController()...)
	routes = append(routes, setupScoreController(poeClient)...)
	routes = append(routes, setupLadderController(poeClient)...)
	routes = append(routes, setupTeamSuggestionController()...)
	routes = append(routes, setupCharacterController(poeClient)...)
	routes = append(routes, setupStreamController(cache)...)
	routes = append(routes, setupRecurringJobsController(poeClient)...)
	routes = append(routes, setupGuildStashController(poeClient)...)
	routes = append(routes, setupActivityController()...)
	routes = append(routes, setupTimingController()...)
	routes = append(routes, setupItemWishController()...)
	routes = append(routes, setupItemController()...)
	routes = append(routes, setupEngagementController()...)
	routes = append(routes, setupAchievementController()...)
	for _, route := range routes {
		handlerfuncs := make([]gin.HandlerFunc, 0)
		if route.Authenticated {
			handlerfuncs = append(handlerfuncs, AuthenticationMiddleware())
		}
		if len(route.RequiredRoles) > 0 {
			handlerfuncs = append(handlerfuncs, AuthorizationMiddleware(route.RequiredRoles))
		}
		handlerfuncs = append(handlerfuncs, LoadEventMiddleware())
		if route.RequiresUserSelf {
			handlerfuncs = append(handlerfuncs, UserSelfMiddleware())
		}
		if route.RequiresTeamSelf {
			handlerfuncs = append(handlerfuncs, TeamSelfMiddleware())
		}
		if route.RequiresTeamLeader {
			handlerfuncs = append(handlerfuncs, TeamLeaderMiddleware())
		}
		handlerfuncs = append(handlerfuncs, route.HandlerFunc)
		group.Handle(route.Method, route.Path, handlerfuncs...)
	}
}

func LoadEventMiddleware() gin.HandlerFunc {
	return func(r *gin.Context) {
		eventParam := r.Param("event_id")
		if eventParam == "" {
			r.Next()
			return
		}
		eventService := service.NewEventService()
		if eventParam == "current" {
			event, err := eventService.GetCurrentEvent()
			if err != nil {
				r.AbortWithStatus(404)
				return
			}
			r.Set("event", event)
			r.Next()
			return
		}
		eventId, err := strconv.Atoi(eventParam)
		if err != nil {
			r.AbortWithStatus(400)
			return
		}
		event, err := eventService.GetEventById(eventId)
		if err != nil {
			r.AbortWithStatus(404)
			return
		}
		r.Set("event", event)
		r.Next()
	}
}

func getEvent(c *gin.Context) *repository.Event {
	event, ok := c.Get("event")
	if !ok {
		c.AbortWithStatus(400)
		return nil
	}
	ev := event.(*repository.Event)
	userRoles := getUserRoles(c)
	if !ev.Public && len(userRoles) == 0 {
		c.AbortWithStatus(404)
		return nil
	}
	return ev
}

func getEventNoAuthCheck(c *gin.Context) *repository.Event {
	event, ok := c.Get("event")
	if !ok {
		c.AbortWithStatus(400)
		return nil
	}
	return event.(*repository.Event)
}

func AuthenticationMiddleware() gin.HandlerFunc {
	return func(r *gin.Context) {
		if !isAuthenticated(r) {
			r.AbortWithStatus(401)
			return
		}
		userRoles := getUserRoles(r)
		r.Set("userRoles", userRoles)
	}
}

func UserSelfMiddleware() gin.HandlerFunc {
	return func(r *gin.Context) {
		userIdFromToken, ok := getUserId(r)
		if !ok {
			r.AbortWithStatus(401)
			return
		}
		userIdParam := r.Param("user_id")
		if userIdParam == "" {
			r.Next()
			return
		}
		userIdFromParam, err := strconv.Atoi(userIdParam)
		if err != nil {
			r.AbortWithStatus(400)
			return
		}
		if userIdFromToken != userIdFromParam && !slices.Contains(getUserRoles(r), repository.PermissionAdmin) {
			r.AbortWithStatus(403)
			return
		}
		r.Next()
	}
}

func TeamSelfMiddleware() gin.HandlerFunc {
	return func(r *gin.Context) {
		teamIdParam := r.Param("team_id")
		if teamIdParam == "" {
			r.Next()
			return
		}
		teamId, err := strconv.Atoi(teamIdParam)
		if err != nil {
			fmt.Println("Error parsing team ID:", err)
			r.AbortWithStatus(400)
			return
		}
		event := getEvent(r)
		if event == nil {
			fmt.Println("Event not found in context")
			r.AbortWithStatus(400)
			return
		}
		teamService := service.NewUserService()
		teamUser, _, err := teamService.GetTeamForUser(r, event)
		if (err != nil || teamUser.TeamId != teamId) && !slices.Contains(getUserRoles(r), repository.PermissionAdmin) {
			r.AbortWithStatus(403)
			return
		}
		r.Next()
	}
}

func TeamLeaderMiddleware() gin.HandlerFunc {
	return func(r *gin.Context) {
		teamIdParam := r.Param("team_id")
		if teamIdParam == "" {
			r.Next()
			return
		}
		teamId, err := strconv.Atoi(teamIdParam)
		if err != nil {
			r.AbortWithStatus(400)
			return
		}
		event := getEvent(r)
		teamService := service.NewUserService()
		teamUser, _, err := teamService.GetTeamForUser(r, event)
		if (err != nil || teamUser.TeamId != teamId || !teamUser.IsTeamLead) && !slices.Contains(getUserRoles(r), repository.PermissionAdmin) {
			r.AbortWithStatus(403)
			return
		}
		r.Next()
	}
}

func AuthorizationMiddleware(requiredRoles []repository.Permission) gin.HandlerFunc {
	return func(r *gin.Context) {
		userRoles := getUserRoles(r)
		if len(requiredRoles) == 0 {
			r.Next()
			return
		}
		for _, requiredRole := range requiredRoles {
			if slices.Contains(userRoles, requiredRole) {
				r.Next()
				return
			}
		}
		r.AbortWithStatus(403)
	}
}

func getUserId(r *gin.Context) (userId int, ok bool) {
	authHeader := r.Request.Header.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return 0, false
	}
	token, err := auth.ParseToken(authHeader[7:])
	if err != nil {
		return 0, false
	}
	claims := &auth.Claims{}
	if !token.Valid {
		return 0, false
	}
	claims.FromJWTClaims(token.Claims)
	if err := claims.Valid(); err != nil {
		return 0, false
	}
	return claims.UserId, true
}

func getUserRoles(r *gin.Context) (permissions []repository.Permission) {
	authHeader := r.Request.Header.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return permissions
	}
	token, err := auth.ParseToken(authHeader[7:])
	if err != nil {
		return permissions
	}
	claims := &auth.Claims{}
	if !token.Valid {
		return permissions
	}
	claims.FromJWTClaims(token.Claims)
	if err := claims.Valid(); err != nil {
		return permissions
	}
	return utils.Map(claims.Permissions, func(perm string) repository.Permission {
		return repository.Permission(perm)
	})
}

func isAuthenticated(r *gin.Context) bool {
	authHeader := r.Request.Header.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return false
	}
	token, err := auth.ParseToken(authHeader[7:])
	if err != nil {
		return false
	}
	return token.Valid
}
