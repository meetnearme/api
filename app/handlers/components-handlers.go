package handlers

import (
	templates "meetnearme-web/templates/components"

	"github.com/gin-gonic/gin"
)

func GetLoginFormComponent(c *gin.Context) {
	component := templates.LoginFormComponent()
	component.Render(c.Request.Context(), c.Writer)
}
