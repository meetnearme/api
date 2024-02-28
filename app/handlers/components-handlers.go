package handlers

import (
	templates "github.com/meetnearme/api/app/templates/components"

	"github.com/gin-gonic/gin"
)

func GetLoginFormComponent(c *gin.Context) {
	component := templates.LoginFormComponent()
	component.Render(c.Request.Context(), c.Writer)
}
