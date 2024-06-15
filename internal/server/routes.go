package server

import (
	"net/http"

	"github.com/firefart/go-webserver-template/internal/server/handlers"
	"github.com/labstack/echo/v4"
)

func (s *server) addRoutes(e *echo.Echo) {
	var secretKeyHeaderName = http.CanonicalHeaderKey(s.config.Notifications.SecretKeyHeaderName)

	static := echo.MustSubFS(fsAssets, "assets/web")
	e.FileFS("/robots.txt", "robots.txt", static)
	e.StaticFS("/scripts", echo.MustSubFS(static, "scripts"))
	e.StaticFS("/css", echo.MustSubFS(static, "css"))

	e.GET("/", handlers.NewIndexHandler(s.debug).EchoHandler)

	e.GET("/test/panic", handlers.NewPanicHandler(s.logger, s.debug, secretKeyHeaderName, s.config.Notifications.SecretKeyHeaderValue).EchoHandler)
	e.GET("/test/notifications", handlers.NewNotificationHandler(s.logger, s.debug, secretKeyHeaderName, s.config.Notifications.SecretKeyHeaderValue).EchoHandler)
}
