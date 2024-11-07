package server

import (
	"github.com/firefart/go-webserver-template/internal/server/handlers"
	"github.com/firefart/go-webserver-template/internal/server/middleware"
	"github.com/labstack/echo/v4"
)

func (s *server) addRoutes(e *echo.Echo) {
	secretKeyHeaderMW := middleware.SecretKeyHeader(middleware.SecretKeyHeaderConfig{
		// skip the middleware in debug mode
		Skipper: func(_ echo.Context) bool {
			return s.debug
		},
		SecretKeyHeaderName:  s.config.SecretKeyHeaderName,
		SecretKeyHeaderValue: s.config.SecretKeyHeaderValue,
		Logger:               s.logger,
	})

	static := echo.MustSubFS(fsAssets, "assets/web")
	e.FileFS("/robots.txt", "robots.txt", static)
	e.StaticFS("/scripts", echo.MustSubFS(static, "scripts"))
	e.StaticFS("/css", echo.MustSubFS(static, "css"))

	e.GET("/", handlers.NewIndexHandler(s.debug).EchoHandler)

	testGroup := e.Group("/test", secretKeyHeaderMW)
	testGroup.GET("/panic", handlers.NewPanicHandler().EchoHandler)
	testGroup.GET("/notifications", handlers.NewNotificationHandler().EchoHandler)

	versionGroup := e.Group("/version", secretKeyHeaderMW)
	versionGroup.GET("", handlers.NewVersionHandler().EchoHandler)
}
