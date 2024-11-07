package handlers

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

func (*NotificationHandler) EchoHandler(_ echo.Context) error {
	return fmt.Errorf("test")
}
