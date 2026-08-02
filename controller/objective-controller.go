package controller

import (
	"bpl/client"
	"bpl/cron"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ObjectiveController struct {
	objectiveService        service.ObjectiveService
	objectiveMatchService   service.ObjectiveMatchService
	eventService            service.EventService
	poeClient               *client.PoEClient
	validationContextCancel *context.CancelFunc
}

func NewObjectiveController() *ObjectiveController {
	return &ObjectiveController{
		objectiveService:      service.NewObjectiveService(),
		eventService:          service.NewEventService(),
		objectiveMatchService: service.NewObjectiveMatchService(),
	}
}

func setupObjectiveController(poeClient *client.PoEClient) []RouteInfo {
	e := NewObjectiveController()
	e.poeClient = poeClient
	baseUrl := "/events/:event_id/objectives"
	editorRoles := []repository.Permission{repository.PermissionAdmin, repository.PermissionManager, repository.PermissionObjectiveDesigner}
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.GetObjectiveTreeForEventHandler()},
		{Method: "PUT", Path: "", HandlerFunc: e.createObjectiveHandler(), Authenticated: true, RequiredRoles: editorRoles},
		{Method: "GET", Path: "/:id", HandlerFunc: e.getObjectiveByIdHandler(), Authenticated: true, RequiredRoles: editorRoles},
		{Method: "DELETE", Path: "/:id", HandlerFunc: e.deleteObjectiveHandler(), Authenticated: true, RequiredRoles: editorRoles},
		{Method: "POST", Path: "/:id/copy", HandlerFunc: e.copyObjectiveHandler(), Authenticated: true, RequiredRoles: editorRoles},
		// todo: move this somewhere else
		{Method: "POST", Path: "/parser", HandlerFunc: e.getObjectiveParserHandler(), Authenticated: true, RequiredRoles: editorRoles},
		{Method: "POST", Path: "/validations", HandlerFunc: e.validateObjectivesHandler(), Authenticated: true, RequiredRoles: editorRoles},
		{Method: "GET", Path: "/validations", HandlerFunc: e.getObjectiveValidationsHandler(), Authenticated: true, RequiredRoles: editorRoles},
		{Method: "GET", Path: "/valid-mappings", HandlerFunc: e.getValidMappingsHandler(), Authenticated: true, RequiredRoles: editorRoles},
	}
	for i, route := range routes {
		routes[i].Path = baseUrl + route.Path
	}
	return routes
}

// @id GetValidMappings
// @Description Get valid mappings for conditions
// @Security BearerAuth
// @Tags objective
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {object} ConditionMappings
// @Router /events/{event_id}/objectives/valid-mappings [get]
func (e *ObjectiveController) getValidMappingsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, ConditionMappings{
			FieldToType:                  repository.FieldToType,
			ValidOperators:               repository.OperatorsForTypes,
			ObjectiveTypeToTrackedValues: repository.ObjectiveTypeToTrackedValues,
		})
	}
}

type ValidationRequest struct {
	TimeoutSeconds int `json:"timeout_seconds" binding:"required"`
}

// @id ValidateObjectives
// @Description Validates item objectives for an event seeing if there are completions on trade
// @Tags objective
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param validationRequest body ValidationRequest true "Validation request"
// @Success 204
// @Router /events/{event_id}/objectives/validations [post]
func (e *ObjectiveController) validateObjectivesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var validationRequest ValidationRequest
		if err := c.BindJSON(&validationRequest); err != nil {
			validationRequest.TimeoutSeconds = 300
		}
		if e.validationContextCancel != nil {
			(*e.validationContextCancel)()
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(validationRequest.TimeoutSeconds)*time.Second)
		e.validationContextCancel = &cancel
		go func() {
			defer cancel()
			cron.ValidationLoop(ctx, e.poeClient)
		}()
		c.JSON(204, nil)
	}
}

// @id GetObjectiveValidations
// @Description Gets objective validations for an event
// @Tags objective
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {array} ObjectiveValidation
// @Router /events/{event_id}/objectives/validations [get]
func (e *ObjectiveController) getObjectiveValidationsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		validations, err := e.objectiveMatchService.GetValidationsByEventId(event.Id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, utils.Map(validations, toObjectiveValidationResponse))
	}
}

// @id GetObjectiveTreeForEvent
// @Description Gets all objectives for an event
// @Tags objective
// @Produce json
// @Param event_id path int true "Event Id"
// @Success 200 {object} Objective
// @Router /events/{event_id}/objectives [get]
func (e *ObjectiveController) GetObjectiveTreeForEventHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		rootObjective, err := e.objectiveService.GetObjectiveTreeForEvent(event.Id, "ScoringRules")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Objectives not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		if rootObjective == nil {
			c.JSON(404, gin.H{"error": "Objectives not found"})
			return
		}
		roles := getUserRoles(c)
		public := !(slices.Contains(roles, repository.PermissionAdmin) ||
			slices.Contains(roles, repository.PermissionManager) ||
			slices.Contains(roles, repository.PermissionObjectiveDesigner))
		c.JSON(200, toObjectiveResponse(rootObjective, public, event.EventEndTime))
	}
}

// @id CreateObjective
// @Description Creates a new objective
// @Security BearerAuth
// @Tags objective
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param objectiveCreate body ObjectiveCreate true "Objective to create"
// @Success 201 {object} Objective
// @Router /events/{event_id}/objectives [put]
func (e *ObjectiveController) createObjectiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var objectiveCreate ObjectiveCreate
		if err := c.BindJSON(&objectiveCreate); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "event is locked"})
			return
		}
		model := objectiveCreate.toModel()
		model.EventId = event.Id
		objective, err := e.objectiveService.CreateObjective(model, objectiveCreate.ScoringRuleIds)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Category not found"})
			} else {
				c.JSON(400, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(201, toObjectiveResponse(objective, false, event.EventEndTime))
	}
}

// @id DeleteObjective
// @Description Deletes an objective
// @Security BearerAuth
// @Tags objective
// @Produce json
// @Param event_id path int true "Event Id"
// @Param id path int true "Objective Id"
// @Success 204
// @Router /events/{event_id}/objectives/{id} [delete]
func (e *ObjectiveController) deleteObjectiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "event is locked"})
			return
		}

		err = e.objectiveService.DeleteObjective(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Objective not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(204, nil)
	}
}

// @id GetObjective
// @Description Gets an objective by id
// @Security BearerAuth
// @Tags objective
// @Produce json
// @Param event_id path int true "Event Id"
// @Param id path int true "Objective Id"
// @Success 200 {object} Objective
// @Router /events/{event_id}/objectives/{id} [get]
func (e *ObjectiveController) getObjectiveByIdHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event := getEvent(c)
		if event == nil {
			return
		}
		objective, err := e.objectiveService.GetObjectiveById(id, "ScoringRules")
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Objective not found"})
			} else {
				c.JSON(500, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(200, toObjectiveResponse(objective, false, event.EventEndTime))
	}
}

type CopyObjectiveRequest struct {
	TargetParentId int `json:"target_parent_id" binding:"required"`
}

// @id CopyObjective
// @Description Copies an objective and all its children onto another objective, stripping scoring rules
// @Security BearerAuth
// @Tags objective
// @Accept json
// @Produce json
// @Param event_id path int true "Event Id"
// @Param id path int true "Source Objective Id"
// @Param copyRequest body CopyObjectiveRequest true "Copy request"
// @Success 201 {object} Objective
// @Router /events/{event_id}/objectives/{id}/copy [post]
func (e *ObjectiveController) copyObjectiveHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		var req CopyObjectiveRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		event := getEvent(c)
		if event == nil {
			return
		}
		if event.Locked {
			c.JSON(400, gin.H{"error": "event is locked"})
			return
		}
		copied, err := e.objectiveService.CopyObjectiveSubtree(id, req.TargetParentId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, toObjectiveResponse(copied, false, event.EventEndTime))
	}
}

func (e *ObjectiveController) getObjectiveParserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		event := getEvent(c)
		if event == nil {
			return
		}
		parser, err := e.objectiveService.GetParser(event.Id, true)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		var item client.Item
		if err := c.BindJSON(&item); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, parser.CheckForCompletions(&item))
	}
}

type ObjectiveConditionCreate struct {
	Id         int                  `json:"id"`
	Operator   repository.Operator  `json:"operator" binding:"required,oneof=EQ NEQ GT GTE LT LTE IN NOT_IN MATCHES CONTAINS CONTAINS_ALL CONTAINS_MATCH CONTAINS_ALL_MATCHES"`
	ItemField  repository.ItemField `json:"field" binding:"required,oneof=BASE_TYPE NAME TYPE_LINE RARITY ILVL FRAME_TYPE TALISMAN_TIER ENCHANT_MODS EXPLICIT_MODS IMPLICIT_MODS CRAFTED_MODS FRACTURED_MODS SIX_LINK"`
	FieldValue string               `json:"value" binding:"required"`
}

type ObjectiveCreate struct {
	Id             int                         `json:"id"`
	Name           string                      `json:"name" binding:"required"`
	Extra          string                      `json:"extra"`
	RequiredNumber int                         `json:"required_number" binding:"required"`
	ObjectiveType  repository.ObjectiveType    `json:"objective_type" binding:"required"`
	TrackedValue   repository.TrackedValue     `json:"tracked_value" binding:"required"`
	CountingMethod repository.CountingMethod   `json:"counting_method" binding:"required"`
	ParentId       int                         `json:"parent_id" binding:"required"`
	Conditions     []Condition                 `json:"conditions" binding:"required"`
	ValidFrom      *time.Time                  `json:"valid_from" binding:"omitempty" format:"date-time"`
	ValidTo        *time.Time                  `json:"valid_to" binding:"omitempty" format:"date-time"`
	ScoringRuleIds []int                       `json:"scoring_rule_ids" binding:"required"`
	HideProgress   bool                        `json:"hide_progress"`
	Details        repository.ObjectiveDetails `json:"details"`
}

type Objective struct {
	Id             int                         `json:"id" binding:"required"`
	Name           string                      `json:"name" binding:"required"`
	Extra          string                      `json:"extra" binding:"required"`
	RequiredNumber int                         `json:"required_number" binding:"required"`
	ParentId       *int                        `json:"parent_id" binding:"required"`
	ObjectiveType  repository.ObjectiveType    `json:"objective_type" binding:"required"`
	Conditions     []*Condition                `json:"conditions" binding:"required"`
	ValidFrom      *time.Time                  `json:"valid_from" binding:"omitempty" format:"date-time"`
	ValidTo        *time.Time                  `json:"valid_to" binding:"omitempty" format:"date-time"`
	ScoringRules   []*ScoringRule              `json:"scoring_rules" binding:"required"`
	TrackedValue   repository.TrackedValue     `json:"tracked_value" binding:"required"`
	CountingMethod repository.CountingMethod   `json:"counting_method" binding:"required"`
	Children       []*Objective                `json:"children" binding:"required"`
	HideProgress   bool                        `json:"hide_progress" binding:"required"`
	Details        repository.ObjectiveDetails `json:"details"`
}

func (e *ObjectiveCreate) toModel() *repository.Objective {
	return &repository.Objective{
		Id:             e.Id,
		Name:           e.Name,
		Extra:          e.Extra,
		RequiredAmount: e.RequiredNumber,
		ObjectiveType:  e.ObjectiveType,
		TrackedValue:   e.TrackedValue,
		CountingMethod: e.CountingMethod,
		Conditions:     utils.Map(e.Conditions, func(c Condition) *repository.Condition { return c.toModel() }),
		ValidFrom:      e.ValidFrom,
		ValidTo:        e.ValidTo,
		ParentId:       &e.ParentId,
		HideProgress:   e.HideProgress,
		Details:        e.Details,
	}
}

func toObjectiveResponse(objective *repository.Objective, public bool, eventEnd time.Time) *Objective {
	if objective == nil || (public && objective.ValidTo != nil && (*objective.ValidTo).After(eventEnd)) {
		return nil
	}
	if public && objective.ValidFrom != nil && time.Now().Before(*objective.ValidFrom) {
		return &Objective{
			Id:           objective.Id,
			ParentId:     objective.ParentId,
			ValidFrom:    objective.ValidFrom,
			ValidTo:      objective.ValidTo,
			ScoringRules: utils.Map(objective.ScoringRules, toScoringRuleResponse),
			HideProgress: objective.HideProgress,
			Children:     make([]*Objective, 0),
			Conditions:   make([]*Condition, 0),
		}
	}

	return &Objective{
		Id:             objective.Id,
		Name:           objective.Name,
		Extra:          objective.Extra,
		RequiredNumber: objective.RequiredAmount,
		ParentId:       objective.ParentId,
		ObjectiveType:  objective.ObjectiveType,
		ValidFrom:      objective.ValidFrom,
		ValidTo:        objective.ValidTo,
		Conditions:     utils.FilterNull(utils.Map(objective.Conditions, toConditionResponse)),
		TrackedValue:   objective.TrackedValue,
		CountingMethod: objective.CountingMethod,
		ScoringRules:   utils.FilterNull(utils.Map(objective.ScoringRules, toScoringRuleResponse)),
		Children:       utils.FilterNull(utils.Map(objective.Children, func(o *repository.Objective) *Objective { return toObjectiveResponse(o, public, eventEnd) })),
		HideProgress:   objective.HideProgress,
		Details:        objective.Details,
	}
}

type ObjectiveValidation struct {
	ObjectiveId int         `json:"objective_id" binding:"required"`
	Timestamp   time.Time   `json:"timestamp" binding:"required" format:"date-time"`
	Item        client.Item `json:"item" binding:"required"`
}

func toObjectiveValidationResponse(validation *repository.ObjectiveValidation) *ObjectiveValidation {
	if validation == nil {
		return nil
	}
	return &ObjectiveValidation{
		ObjectiveId: validation.ObjectiveId,
		Timestamp:   validation.Timestamp,
		Item:        validation.Item,
	}
}

type Condition struct {
	Operator   repository.Operator  `json:"operator" binding:"required"`
	ItemField  repository.ItemField `json:"field" binding:"required"`
	FieldValue string               `json:"value" binding:"required"`
}

func (e *Condition) toModel() *repository.Condition {
	return &repository.Condition{
		Operator: repository.Operator(e.Operator),
		Field:    repository.ItemField(e.ItemField),
		Value:    e.FieldValue,
	}
}

func toConditionResponse(condition *repository.Condition) *Condition {
	if condition == nil {
		return nil
	}
	return &Condition{
		Operator:   condition.Operator,
		ItemField:  condition.Field,
		FieldValue: condition.Value,
	}
}

type ConditionMappings struct {
	FieldToType                  map[repository.ItemField]repository.FieldType          `json:"field_to_type" binding:"required"`
	ValidOperators               map[repository.FieldType][]repository.Operator         `json:"valid_operators" binding:"required"`
	ObjectiveTypeToTrackedValues map[repository.ObjectiveType][]repository.TrackedValue `json:"objective_type_to_tracked_values" binding:"required"`
}
