package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/mattn/go-isatty"
)

func newLogger(debugMode, jsonOutput bool) *slog.Logger {
	w := os.Stdout
	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)

	var replaceFunc func(groups []string, a slog.Attr) slog.Attr
	if debugMode {
		level.Set(slog.LevelDebug)
		// add source file information
		wd, err := os.Getwd()
		if err != nil {
			panic("unable to determine working directory")
		}
		replaceFunc = func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source, ok := a.Value.Any().(*slog.Source)
				if !ok {
					return a
				}
				// remove current working directory and only leave the relative path to the program
				if file, ok := strings.CutPrefix(source.File, wd); ok {
					source.File = file
				}
			}
			return a
		}
	}

	var handler slog.Handler
	slogHandlerOpts := &slog.HandlerOptions{
		Level:       level,
		AddSource:   debugMode,
		ReplaceAttr: replaceFunc,
	}
	switch {
	case jsonOutput:
		handler = slog.NewJSONHandler(w, slogHandlerOpts)
	case !isatty.IsTerminal(w.Fd()):
		handler = slog.NewTextHandler(w, slogHandlerOpts)
	default:
		l := log.InfoLevel
		if debugMode {
			l = log.DebugLevel
		}
		handler = log.NewWithOptions(w, log.Options{
			Level:        l,
			ReportCaller: debugMode,
		})
	}
	return slog.New(handler)
}
