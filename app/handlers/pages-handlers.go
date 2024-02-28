package handlers

import (
	templates "github.com/meetnearme/api/app/templates/pages"

	"github.com/gin-gonic/gin"
)

func GetEventsPageContent(c *gin.Context) {
	page := templates.EventsPage()
	shouldWrapLayout := c.GetBool("shouldWrapLayout")
	if shouldWrapLayout {
		page = templates.Layout(templates.EventsPage())
	}
	page.Render(c.Request.Context(), c.Writer)
}

func GetAccountPageContent(c *gin.Context) {
	page := templates.AccountPage()
	shouldWrapLayout := c.GetBool("shouldWrapLayout")
	if shouldWrapLayout {
		page = templates.Layout(templates.AccountPage())
	}
	page.Render(c.Request.Context(), c.Writer)
}

func GetLoginPageContent(c *gin.Context) {
	page := templates.LoginPage()
	shouldWrapLayout := c.GetBool("shouldWrapLayout")
	if shouldWrapLayout {
		page = templates.Layout(templates.LoginPage())
	}
	page.Render(c.Request.Context(), c.Writer)
}
