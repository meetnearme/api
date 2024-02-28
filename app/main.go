package main

import (
	"meetnearme-web/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"} // Update with your allowed origins
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.Use(shouldWrapLayout())
	r.GET("/", handlers.GetEventsPageContent)
	r.GET("/events", handlers.GetEventsPageContent)
	r.GET("/account", handlers.GetAccountPageContent)
	r.GET("/login", handlers.GetLoginPageContent)

	componentRouterGroup := r.Group("/components")
	{
		componentRouterGroup.GET("/login-form", handlers.GetLoginFormComponent)
	}
	r.SetTrustedProxies(nil)
	r.Run()
}

func shouldWrapLayout() gin.HandlerFunc {
	return func(c *gin.Context) {
		hxRequestHeader := c.Request.Header["Hx-Request"]
		isHxRequest := hxRequestHeader != nil && hxRequestHeader[0] == "true"
		c.Set("shouldWrapLayout", !isHxRequest)
		c.Next()
	}
}
