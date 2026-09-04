package controller

import (
	"bpl/repository"
	"bpl/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AchievementController struct {
	achievementService service.AchievementService
}

func NewAchievementController() *AchievementController {
	return &AchievementController{
		achievementService: service.NewAchievementService(),
	}
}

func setupAchievementController() []RouteInfo {
	e := NewAchievementController()
	definitionRoles := []repository.Permission{repository.PermissionAdmin, repository.PermissionObjectiveDesigner}
	assignerRoles := []repository.Permission{repository.PermissionAdmin, repository.PermissionAchievementAssigner}
	routes := []RouteInfo{
		{Method: "GET", Path: "/achievements", HandlerFunc: e.getAchievements()},
		{Method: "POST", Path: "/achievements", HandlerFunc: e.createAchievement(), Authenticated: true, RequiredRoles: definitionRoles},
		{Method: "PATCH", Path: "/achievements/:achievement_id", HandlerFunc: e.updateAchievement(), Authenticated: true, RequiredRoles: definitionRoles},
		{Method: "DELETE", Path: "/achievements/:achievement_id", HandlerFunc: e.deleteAchievement(), Authenticated: true, RequiredRoles: definitionRoles},

		{Method: "GET", Path: "/user-achievements", HandlerFunc: e.getUserAchievements()},
		{Method: "POST", Path: "/user-achievements", HandlerFunc: e.grantAchievement(), Authenticated: true, RequiredRoles: assignerRoles},
		{Method: "DELETE", Path: "/user-achievements/:user_id/:achievement_id", HandlerFunc: e.revokeAchievement(), Authenticated: true, RequiredRoles: assignerRoles},

		{Method: "PUT", Path: "/achievements/:achievement_id/icon", HandlerFunc: e.uploadIcon(), Authenticated: true, RequiredRoles: definitionRoles},
		{Method: "GET", Path: "/achievements/:achievement_id/icon", HandlerFunc: e.getIcon()},

		{Method: "POST", Path: "/achievements/sync", HandlerFunc: e.syncAchievements(), Authenticated: true, RequiredRoles: []repository.Permission{repository.PermissionAdmin}},
	}
	return routes
}

// @ID getAchievements
// @Summary List achievement definitions
// @Tags Achievement
// @Produce json
// @Success 200 {array} AchievementResponse
// @Router /achievements [get]
func (c *AchievementController) getAchievements() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		achievements, err := c.achievementService.GetAllAchievements()
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, toAchievementResponses(achievements))
	}
}

// @ID createAchievement
// @Summary Create a custom achievement
// @Security BearerAuth
// @Tags Achievement
// @Accept json
// @Produce json
// @Param body body AchievementCreate true "Achievement to create"
// @Success 201 {object} AchievementResponse
// @Router /achievements [post]
func (c *AchievementController) createAchievement() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var body AchievementCreate
		if err := ctx.ShouldBindJSON(&body); err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		achievement, err := c.achievementService.CreateAchievement(body.Name, body.Description, body.EventId)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(201, toAchievementResponse(achievement))
	}
}

// @ID updateAchievement
// @Summary Update a custom achievement
// @Security BearerAuth
// @Tags Achievement
// @Accept json
// @Produce json
// @Param achievement_id path int true "Achievement ID"
// @Param body body AchievementCreate true "Updated fields"
// @Success 200 {object} AchievementResponse
// @Router /achievements/{achievement_id} [patch]
func (c *AchievementController) updateAchievement() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id, err := strconv.Atoi(ctx.Param("achievement_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid achievement_id"})
			return
		}
		var body AchievementCreate
		if err := ctx.ShouldBindJSON(&body); err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		achievement, err := c.achievementService.UpdateAchievement(id, body.Name, body.Description, body.EventId)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, toAchievementResponse(achievement))
	}
}

// @ID deleteAchievement
// @Summary Delete a custom achievement
// @Security BearerAuth
// @Tags Achievement
// @Param achievement_id path int true "Achievement ID"
// @Success 204
// @Router /achievements/{achievement_id} [delete]
func (c *AchievementController) deleteAchievement() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id, err := strconv.Atoi(ctx.Param("achievement_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid achievement_id"})
			return
		}
		if err := c.achievementService.DeleteAchievement(id); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.Status(204)
	}
}

// @ID getUserAchievements
// @Summary List user achievement grants
// @Tags Achievement
// @Produce json
// @Param user_id query int false "Filter by user ID"
// @Success 200 {array} UserAchievementResponse
// @Router /user-achievements [get]
func (c *AchievementController) getUserAchievements() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var userId *int
		if raw := ctx.Query("user_id"); raw != "" {
			id, err := strconv.Atoi(raw)
			if err != nil {
				ctx.JSON(400, gin.H{"error": "invalid user_id"})
				return
			}
			userId = &id
		}
		grants, err := c.achievementService.GetAllUserAchievements(userId)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, toUserAchievementResponses(grants))
	}
}

// @ID grantAchievement
// @Summary Grant an achievement to a user
// @Security BearerAuth
// @Tags Achievement
// @Accept json
// @Produce json
// @Param body body AchievementGrant true "Grant details"
// @Success 201
// @Router /user-achievements [post]
func (c *AchievementController) grantAchievement() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var body AchievementGrant
		if err := ctx.ShouldBindJSON(&body); err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		grantedBy, _ := getUserId(ctx)
		if err := c.achievementService.GrantAchievement(body.UserId, body.AchievementId, &grantedBy); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.Status(201)
	}
}

// @ID revokeAchievement
// @Summary Revoke an achievement from a user
// @Security BearerAuth
// @Tags Achievement
// @Param user_id path int true "User ID"
// @Param achievement_id path int true "Achievement ID"
// @Success 204
// @Router /user-achievements/{user_id}/{achievement_id} [delete]
func (c *AchievementController) revokeAchievement() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userId, err := strconv.Atoi(ctx.Param("user_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid user_id"})
			return
		}
		achievementId, err := strconv.Atoi(ctx.Param("achievement_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid achievement_id"})
			return
		}
		if err := c.achievementService.RevokeAchievement(userId, achievementId); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.Status(204)
	}
}

// @ID uploadIcon
// @Summary Upload an icon for an achievement
// @Security BearerAuth
// @Tags Achievement
// @Accept multipart/form-data
// @Param achievement_id path int true "Achievement ID"
// @Param icon formData file true "Icon file"
// @Success 204
// @Router /achievements/{achievement_id}/icon [put]
func (c *AchievementController) uploadIcon() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id, err := strconv.Atoi(ctx.Param("achievement_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid achievement_id"})
			return
		}
		file, header, err := ctx.Request.FormFile("icon")
		if err != nil {
			ctx.JSON(400, gin.H{"error": "icon file required"})
			return
		}
		defer file.Close()
		data := make([]byte, header.Size)
		if _, err := file.Read(data); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		mimeType := header.Header.Get("Content-Type")
		if err := c.achievementService.UploadIcon(id, data, mimeType); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.Status(204)
	}
}

// @ID getIcon
// @Summary Get the icon for an achievement
// @Tags Achievement
// @Param achievement_id path int true "Achievement ID"
// @Produce image/*
// @Success 200
// @Router /achievements/{achievement_id}/icon [get]
func (c *AchievementController) getIcon() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id, err := strconv.Atoi(ctx.Param("achievement_id"))
		if err != nil {
			ctx.JSON(400, gin.H{"error": "invalid achievement_id"})
			return
		}
		achievement, err := c.achievementService.GetAchievementById(id)
		if err != nil {
			ctx.JSON(404, gin.H{"error": "achievement not found"})
			return
		}
		if len(achievement.Icon) == 0 {
			ctx.Status(404)
			return
		}
		ctx.Header("Cache-Control", "public, max-age=86400")
		ctx.Data(200, achievement.IconMimeType, achievement.Icon)
	}
}

// @ID syncAchievements
// @Summary Trigger system achievement sync
// @Security BearerAuth
// @Tags Achievement
// @Success 204
// @Router /achievements/sync [post]
func (c *AchievementController) syncAchievements() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := c.achievementService.SyncAchievements(); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.Status(204)
	}
}

// Request / response types

type AchievementCreate struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	EventId     *int   `json:"event_id"`
}

type AchievementGrant struct {
	UserId        int `json:"user_id" binding:"required"`
	AchievementId int `json:"achievement_id" binding:"required"`
}

type AchievementResponse struct {
	Id           int                             `json:"id"`
	Name         string                          `json:"name"`
	Description  string                          `json:"description"`
	IsCustom     bool                            `json:"is_custom"`
	IconUrl      string                          `json:"icon_url,omitempty"`
	AutoCheckKey *repository.AchievementCheckKey `json:"auto_check_key,omitempty"`
	EventId      *int                            `json:"event_id,omitempty"`
}

type UserAchievementResponse struct {
	UserId        int       `json:"user_id"`
	AchievementId int       `json:"achievement_id"`
	GrantedAt     time.Time `json:"granted_at"`
	GrantedBy     *int      `json:"granted_by"`
}

func toAchievementResponse(a *repository.Achievement) *AchievementResponse {
	resp := &AchievementResponse{
		Id:           a.Id,
		Name:         a.Name,
		Description:  a.Description,
		IsCustom:     a.IsCustom,
		AutoCheckKey: a.AutoCheckKey,
		EventId:      a.EventId,
	}
	if len(a.Icon) > 0 {
		resp.IconUrl = "/achievements/" + strconv.Itoa(a.Id) + "/icon"
	}
	return resp
}

func toAchievementResponses(achievements []*repository.Achievement) []*AchievementResponse {
	out := make([]*AchievementResponse, len(achievements))
	for i, a := range achievements {
		out[i] = toAchievementResponse(a)
	}
	return out
}

func toUserAchievementResponse(ua *repository.UserAchievement) *UserAchievementResponse {
	return &UserAchievementResponse{
		UserId:        ua.UserId,
		AchievementId: ua.AchievementId,
		GrantedAt:     ua.GrantedAt,
		GrantedBy:     ua.GrantedBy,
	}
}

func toUserAchievementResponses(grants []*repository.UserAchievement) []*UserAchievementResponse {
	out := make([]*UserAchievementResponse, len(grants))
	for i, ua := range grants {
		out[i] = toUserAchievementResponse(ua)
	}
	return out
}
