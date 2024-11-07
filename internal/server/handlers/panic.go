package handlers

import (
	"github.com/labstack/echo/v4"
)

type PanicHandler struct{}

func NewPanicHandler() *PanicHandler {
	return &PanicHandler{}
}

func (*PanicHandler) EchoHandler(_ echo.Context) error {
	panic("test")
}
