package main

import "github.com/labstack/echo/v4"

func (app *application) addRoutes(e *echo.Echo) {
	static := echo.MustSubFS(staticFS, "assets/web")
	e.FileFS("/robots.txt", "robots.txt", static)
	e.StaticFS("/static", static)
	e.GET("/", app.handleIndex)
	e.GET("/test_panic", app.handleTestPanic)
	e.GET("/test_notifications", app.handleTestNotification)
}
