package controller

import (
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ItemWishController struct {
	itemWishService service.ItemWishService
	userService     service.UserService
}

func NewItemWishController() *ItemWishController {
	return &ItemWishController{
		itemWishService: service.NewItemWishService(),
		userService:     service.NewUserService(),
	}
}

func setupItemWishController() []RouteInfo {
	e := NewItemWishController()
	basePath := "events/:event_id/teams/:team_id/item_wishes"
	routes := []RouteInfo{
		{Method: "GET", Path: "", HandlerFunc: e.getItemWishesForTeamHandler(), Authenticated: true, RequiresTeamSelf: true},
		{Method: "POST", Path: "", HandlerFunc: e.creatItemWishHandler(), Authenticated: true, RequiresTeamSelf: true},
		{Method: "PATCH", Path: "/:wish_id", HandlerFunc: e.changeItemWishHandler(), Authenticated: true},
		{Method: "DELETE", Path: "/:wish_id", HandlerFunc: e.deleteItemWishHandler(), Authenticated: true},
	}
	for i, route := range routes {
		routes[i].Path = basePath + route.Path
	}
	return routes
}

// @id GetItemWishesForTeam
// @Description Get item wishes for a team in an event
// @Tags item_wishes
// @Produce json
// @Param event_id path int true "Event ID"
// @Param team_id path int true "Team ID"
// @Success 200 {array} ItemWish
// @Router /events/{event_id}/teams/{team_id}/item_wishes [get]
func (e *ItemWishController) getItemWishesForTeamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid team ID"})
			return
		}
		itemWishes, err := e.itemWishService.GetItemWishesForTeam(teamId)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get item wishes"})
			return
		}
		c.JSON(200, utils.Map(itemWishes, toItemWishModel))

	}
}

// @id CreateItemWish
// @Description Create an item wish for a user in a team
// @Tags item_wishes
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Param team_id path int true "Team ID"
// @Param item_wish body CreateItemWish true "Item Wish"
// @Success 201 {object} ItemWish
// @Router /events/{event_id}/teams/{team_id}/item_wishes [post]
func (e *ItemWishController) creatItemWishHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, ok := getUserId(c)
		if !ok {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		teamId, err := strconv.Atoi(c.Param("team_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid team ID"})
			return
		}

		var itemWishReq CreateItemWish
		if err := c.ShouldBindJSON(&itemWishReq); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		if itemWishReq.Quantity == 0 {
			itemWishReq.Quantity = 1
		} else if itemWishReq.Quantity < 1 || itemWishReq.Quantity > service.MaxItemWishQuantity {
			c.JSON(400, gin.H{"error": fmt.Sprintf("Quantity must be between 1 and %d", service.MaxItemWishQuantity)})
			return
		}

		itemWish := &repository.ItemWish{
			UserID:        userId,
			TeamID:        teamId,
			ItemField:     itemWishReq.ItemField,
			Value:         itemWishReq.Value,
			Extra:         itemWishReq.Extra,
			Fulfilled:     false,
			BuildEnabling: itemWishReq.BuildEnabling,
			Quantity:      itemWishReq.Quantity,
		}

		savedItemWish, err := e.itemWishService.CreateItemWish(itemWish, teamId)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to save item wish"})
			return
		}
		c.JSON(201, toItemWishModel(savedItemWish))
	}
}

// @id ChangeItemWish
// @Description Change an item wish for a user in a team
// @Tags item_wishes
// @Accept json
// @Produce json
// @Param event_id path int true "Event ID"
// @Param team_id path int true "Team ID"
// @Param wish_id path int true "Wish ID"
// @Param item_wish body UpdateItemWish true "Item Wish"
// @Success 200 {object} ItemWish
// @Router /events/{event_id}/teams/{team_id}/item_wishes/{wish_id} [patch]
func (e *ItemWishController) changeItemWishHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		wishId, err := strconv.Atoi(c.Param("wish_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid wish ID"})
			return
		}
		var itemWishReq UpdateItemWish
		if err := c.ShouldBindJSON(&itemWishReq); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		if itemWishReq.Quantity != nil && (*itemWishReq.Quantity < 1 || *itemWishReq.Quantity > service.MaxItemWishQuantity) {
			c.JSON(400, gin.H{"error": fmt.Sprintf("Quantity must be between 1 and %d", service.MaxItemWishQuantity)})
			return
		}
		itemWish, err := e.itemWishService.GetItemWishById(wishId)
		if err != nil {
			c.JSON(404, gin.H{"error": "Failed to get item wish"})
			return
		}
		teamUser, _, err := e.userService.GetTeamForUser(c, getEvent(c))
		if err != nil {
			c.JSON(403, gin.H{"error": "You are not part of a team"})
			return
		}
		if (itemWishReq.BuildEnabling != nil || itemWishReq.Fulfilled != nil || itemWishReq.Quantity != nil) && itemWish.UserID != teamUser.UserId {
			c.JSON(403, gin.H{"error": "Only the user who created the wish can change its fulfilled, build enabling or quantity status"})
			return
		}
		if (itemWishReq.Priority != nil) && !teamUser.IsTeamLead {
			c.JSON(403, gin.H{"error": "Only team leads can change the priority of wishes"})
			return
		}

		updatedItemWish, err := e.itemWishService.UpdateItemWish(itemWish, teamUser.TeamId, itemWishReq.Fulfilled, itemWishReq.BuildEnabling, itemWishReq.Priority, itemWishReq.Quantity)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to update item wish %v", err)})
			return
		}
		c.JSON(200, toItemWishModel(updatedItemWish))
	}
}

// @id DeleteItemWish
// @Description Delete an item wish for a user in a team
// @Tags item_wishes
// @Param event_id path int true "Event ID"
// @Param team_id path int true "Team ID"
// @Param wish_id path int true "Wish ID"
// @Success 204
// @Router /events/{event_id}/teams/{team_id}/item_wishes/{wish_id} [delete]
func (e *ItemWishController) deleteItemWishHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		wishId, err := strconv.Atoi(c.Param("wish_id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid wish ID"})
			return
		}
		user, err := e.userService.GetUserFromAuthHeader(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		itemWish, err := e.itemWishService.GetItemWishById(wishId)
		if err != nil {
			c.JSON(404, gin.H{"error": "Failed to get item wish"})
			return
		}
		if itemWish.UserID != user.Id {
			c.JSON(403, gin.H{"error": "You can only delete your own item wishes"})
			return
		}
		err = e.itemWishService.DeleteItemWish(wishId)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to delete item wish"})
			return
		}
		c.Status(204)
	}
}

type ItemWish struct {
	Id        int                  `json:"id" binding:"required"`
	UserId    int                  `json:"user_id" binding:"required"`
	ItemField repository.ItemField `json:"item_field" binding:"required"`
	Value     string               `json:"value" binding:"required"`
	Extra         string               `json:"extra"`
	Fulfilled bool                 `json:"fulfilled" binding:"required"`
	// BuildEnabling is a 1-5 importance scale (1 = nice to have, 5 = absolutely essential).
	BuildEnabling int `json:"build_enabling" binding:"required"`
	Priority      int `json:"priority" binding:"required"`
	Quantity      int                  `json:"quantity" binding:"required"`
}

type CreateItemWish struct {
	ItemField repository.ItemField `json:"item_field" binding:"required"`
	Value     string               `json:"value" binding:"required"`
	Extra         string               `json:"extra"`
	// BuildEnabling is a mandatory 1-5 importance scale (1 = nice to have, 5 = absolutely essential).
	BuildEnabling int `json:"build_enabling" binding:"required,min=1,max=5"`
	Quantity      int                  `json:"quantity"`
}

type UpdateItemWish struct {
	Fulfilled     *bool `json:"fulfilled"`
	BuildEnabling *int  `json:"build_enabling" binding:"omitempty,min=1,max=5"`
	Priority      *int  `json:"priority"`
	Quantity      *int  `json:"quantity"`
}

func toItemWishModel(iw *repository.ItemWish) *ItemWish {
	return &ItemWish{
		Id:            iw.Id,
		UserId:        iw.UserID,
		ItemField:     iw.ItemField,
		Value:         iw.Value,
		Extra:         iw.Extra,
		Fulfilled:     iw.Fulfilled,
		Priority:      iw.Priority,
		BuildEnabling: iw.BuildEnabling,
		Quantity:      iw.Quantity,
	}
}
