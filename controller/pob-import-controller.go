package controller

import (
	"bpl/client"

	"github.com/gin-gonic/gin"
)

func setupPoBImportController() []RouteInfo {
	return []RouteInfo{
		{Method: "GET", Path: "pob-import", HandlerFunc: getPoBImportHandler()},
	}
}

// @id ImportPoBFromShareLink
// @Description Resolves a Path of Building share link (Maxroll, pobb.in, pob.codes, PoE Ninja, Pastebin.com, PastebinP.com, Rentry.co, poedb.tw) into its raw PoB export code
// @Tags items
// @Produce plain
// @Param url query string true "Share link URL"
// @Success 200 {string} string "raw PoB export code"
// @Router /pob-import [get]
func getPoBImportHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		shareUrl := c.Query("url")
		if shareUrl == "" {
			c.JSON(400, gin.H{"error": "Missing url query parameter"})
			return
		}
		code, err := client.FetchPoBFromShareLink(shareUrl)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.String(200, code)
	}
}
