package version

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/msfidelis/sales-rest-api/pkg/configuration"
)

type Response struct {
	Application string `json:"application" binding:"required"`
	Version     string `json:"version" binding:"required"`
}

// Get godoc
// @Summary Return 200 status Get in version
// @Tags Version
// @Produce json
// @Success 200 {object} Response
// @Router /version [get]
func Get(c *gin.Context) {
	configs := configuration.Load()

	response := Response{
		Version:     configs.Version,
		Application: configs.Application,
	}

	c.JSON(http.StatusOK, response)
}
