package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EventController struct {
	eventService       service.EventService
	teamService        service.TeamService
	userService        service.UserService
	signupService      service.SignupService
	scoringRuleService service.ScoringRuleService
	objectiveService   service.ObjectiveService
}

func NewEventController() *EventController {
	return &EventController{
		eventService:       service.NewEventService(),
		teamService:        service.NewTeamService(),
		userService:        service.NewUserService(),
		signupService:      service.NewSignupService(),
		scoringRuleService: service.NewScoringRulesService(),
		objectiveService:   service.NewObjectiveService(),
	}
}

func setupEventController() []RouteInfo {
	e := NewEventController()
	basePath := "/events"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getEventsHandler()},
		{Method: "PUT", Path: "", HandlerFunc: e.createEventHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/:event_id", HandlerFunc: e.getEvent()},

		{Method: "POST", Path: "/:event_id/duplicate", HandlerFunc: e.duplicateEventHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
		{Method: "GET", Path: "/:event_id/status", HandlerFunc: e.getEventStatus()},
		{Method: "DELETE", Path: "/:event_id", HandlerFunc: e.deleteEventHandler(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetEvents
// @Security BearerAuth
// @Description Fetches all events
// @Tags event
// @Produce json
// @Success 200 {array} Event
// @Router /events [get]
func (e *EventController) getEventsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		events, err := e.eventService.GetAllEvents("Teams")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(events, toEventResponse))
	}
}

// @id GetEvent
// @Description Fetches an event by id
// @Security BearerAuth
// @Tags event
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {object} Event
// @Router /events/{event_id} [get]
func (e *EventController) getEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		ev := getEvent(c)
		if ev == nil {
			return
		}
		event, err := e.eventService.GetEventById(ev.Id, "Teams")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		roles := getUserRoles(c)
		if !slices.Contains(roles, repository.PermissionAdmin) && !event.Public {
			c.JSON(403, gin.H{"error": "You are not allowed to view this event"})
			return
		}
		c.JSON(200, toEventResponse(event))
	}
}

// @id CreateEvent
// @Description Creates or updates an event
// @Security BearerAuth
// @Tags event
// @Accept json
// @Produce json
// @Param event body EventCreate true "Event to create"
// @Success 201 {object} Event
// @Router /events [put]
func (e *EventController) createEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var eventCreate EventCreate
		if err := c.BindJSON(&eventCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		dbevent, err := e.eventService.CreateEvent(eventCreate.toModel())
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Event not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(201, toEventResponse(dbevent))
	}
}

// @id DeleteEvent
// @Description Deletes an event
// @Security BearerAuth
// @Tags event
// @Param event_id path int true "Event Id"
// @Success 204
// @Router /events/{event_id} [delete]
func (e *EventController) deleteEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "Event is locked"})
			return
		}
		err := e.eventService.DeleteEvent(event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(204, nil)
	}
}

// @id GetEventStatus
// @Description Gets the status for an event including the user's application status
// @Tags event
// @Produce json
// @Param event_id path int true "Event Id"
// @Security BearerAuth
// @Success 200 {object} service.EventStatus
// @Router /events/{event_id}/status [get]
func (e *EventController) getEventStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEventNoAuthCheck(c)
		if event == nil {
			return
		}
		user, _ := e.userService.GetUserFromAuthHeader(c)
		response, err := e.eventService.GetEventStatus(event, user)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, response)
	}
}

// @id DuplicateEvent
// @Description Duplicates an event's configuration
// @Tags event
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param event_id path int true "Event Id"
// @Param event body EventCreate true "Event to create"
// @Success 201 {object} Event
// @Router /events/{event_id}/duplicate [post]
func (e *EventController) duplicateEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		oldEvent := getEvent(c)
		if oldEvent == nil {
			return
		}
		var eventCreate EventCreate
		if err := c.BindJSON(&eventCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		eventCreate.Id = nil
		event := eventCreate.toModel()
		event, err := e.eventService.CreateEventWithoutCategory(event)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ruleIdMap, err := e.scoringRuleService.DuplicateRules(oldEvent.Id, event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		err = e.objectiveService.DuplicateObjectives(oldEvent.Id, event.Id, ruleIdMap)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, toEventResponse(event))
	}
}

type EventCreate struct {
	Id                   *int                   `json:"id"`
	Name                 string                 `json:"name" binding:"required"`
	IsCurrent            bool                   `json:"is_current"`
	GameVersion          repository.GameVersion `json:"game_version" binding:"required"`
	Patch                *string                `json:"patch"`
	MaxSize              int                    `json:"max_size" binding:"required"`
	WaitlistSize         int                    `json:"waitlist_size" binding:"required"`
	EventStartTime       time.Time              `json:"event_start_time" binding:"required" format:"date-time"`
	EventEndTime         time.Time              `json:"event_end_time" binding:"required" format:"date-time"`
	ApplicationStartTime time.Time              `json:"application_start_time" binding:"required" format:"date-time"`
	ApplicationEndTime   time.Time              `json:"application_end_time" binding:"required" format:"date-time"`
	Public               bool                   `json:"is_public"`
	Locked               bool                   `json:"is_locked"`
	IsMainEvent          bool                   `json:"is_main_event"`
	UsesMedals           bool                   `json:"uses_medals"`
}

type Event struct {
	Id                   int                    `json:"id" binding:"required"`
	Name                 string                 `json:"name" binding:"required"`
	IsCurrent            bool                   `json:"is_current" binding:"required"`
	GameVersion          repository.GameVersion `json:"game_version" binding:"required"`
	Patch                *string                `json:"patch"`
	MaxSize              int                    `json:"max_size" binding:"required"`
	WaitlistSize         int                    `json:"waitlist_size" binding:"required"`
	Teams                []*Team                `json:"teams" binding:"required"`
	ApplicationStartTime time.Time              `json:"application_start_time" binding:"required" format:"date-time"`
	ApplicationEndTime   time.Time              `json:"application_end_time" binding:"required" format:"date-time"`
	EventStartTime       time.Time              `json:"event_start_time" binding:"required" format:"date-time"`
	EventEndTime         time.Time              `json:"event_end_time" binding:"required" format:"date-time"`
	Public               bool                   `json:"is_public" binding:"required"`
	Locked               bool                   `json:"is_locked" binding:"required"`
	IsMainEvent          bool                   `json:"is_main_event" binding:"required"`
	UsesMedals           bool                   `json:"uses_medals" binding:"required"`
}

func (e *EventCreate) toModel() *repository.Event {
	event := &repository.Event{
		Name:                 e.Name,
		IsCurrent:            e.IsCurrent,
		GameVersion:          e.GameVersion,
		Patch:                e.Patch,
		MaxSize:              e.MaxSize,
		WaitlistSize:         e.WaitlistSize,
		EventStartTime:       e.EventStartTime,
		EventEndTime:         e.EventEndTime,
		ApplicationStartTime: e.ApplicationStartTime,
		ApplicationEndTime:   e.ApplicationEndTime,
		Public:               e.Public,
		Locked:               e.Locked,
		IsMainEvent:          e.IsMainEvent,
	}
	if e.Id != nil {
		event.Id = *e.Id
	}
	return event
}

func toEventResponse(event *repository.Event) *Event {
	if event == nil {
		return nil
	}
	return &Event{
		Id:                   event.Id,
		Name:                 event.Name,
		GameVersion:          event.GameVersion,
		Patch:                event.Patch,
		IsCurrent:            event.IsCurrent,
		MaxSize:              event.MaxSize,
		WaitlistSize:         event.WaitlistSize,
		Teams:                utils.Map(event.Teams, toTeamResponse),
		ApplicationStartTime: event.ApplicationStartTime,
		ApplicationEndTime:   event.ApplicationEndTime,
		EventStartTime:       event.EventStartTime,
		EventEndTime:         event.EventEndTime,
		Public:               event.Public,
		Locked:               event.Locked,
		IsMainEvent:          event.IsMainEvent,
		UsesMedals:           event.UsesMedals,
	}
}
