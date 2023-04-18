package main

import (
	"context"
	"flag"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Info(args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	SetOutput(io.Writer)
	SetLevel(logrus.Level)
}

type application struct {
	logger  Logger
	timeout time.Duration
}

func lookupEnvOrString(log Logger, key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func lookupEnvOrBool(log Logger, key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		v, err := strconv.ParseBool(val)
		if err != nil {
			log.Errorf("lookupEnvOrBool[%s]: %v", key, err)
			return defaultVal
		}
		return v
	}
	return defaultVal
}

func lookupEnvOrDuration(log Logger, key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		v, err := time.ParseDuration(val)
		if err != nil {
			log.Errorf("lookupEnvOrDuration[%s]: %v", key, err)
			return defaultVal
		}
		return v
	}
	return defaultVal
}

func main() {
	app := &application{
		logger: logrus.New(),
	}

	var host string
	var wait time.Duration
	var debugOutput bool
	flag.StringVar(&host,
		"host",
		lookupEnvOrString(app.logger, "APP_HOST", ":8080"),
		"IP and Port to bind to. Can also be set through the APP_HOST environment variable.")
	flag.BoolVar(&debugOutput,
		"debug",
		lookupEnvOrBool(app.logger, "APP_DEBUG", false),
		"Enable DEBUG mode. Can also be set through the APP_DEBUG environment variable.")
	flag.DurationVar(&wait,
		"graceful-timeout",
		lookupEnvOrDuration(app.logger, "APP_GRACEFUL_TIMEOUT", 5*time.Second),
		"the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m. Can also be set through the APP_GRACEFUL_TIMEOUT environment variable.")
	flag.DurationVar(&app.timeout,
		"timeout",
		lookupEnvOrDuration(app.logger, "APP_TIMEOUT", 10*time.Second),
		"the timeout for http calls. Can also be set through the APP_TIMEOUT environment variable.")
	flag.Parse()

	gin.SetMode(gin.ReleaseMode)

	app.logger.SetOutput(os.Stdout)
	app.logger.SetLevel(logrus.InfoLevel)
	if debugOutput {
		gin.SetMode(gin.DebugMode)
		app.logger.SetLevel(logrus.DebugLevel)
		app.logger.Debug("DEBUG mode enabled")
	}

	app.logger.Info("Starting server with the following parameters:")
	app.logger.Infof("host: %s", host)
	app.logger.Infof("debug: %t", debugOutput)
	app.logger.Infof("graceful timeout: %s", wait)
	app.logger.Infof("timeout: %s", app.timeout)

	srv := &http.Server{
		Addr:    host,
		Handler: app.routes(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			app.logger.Error(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.logger.Error(err)
	}
	app.logger.Info("shutting down")
	os.Exit(0)
}

func errorJson(errorText string) gin.H {
	return gin.H{
		"error": errorText,
	}
}

func (app *application) routes() http.Handler {
	r := gin.Default()
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, errorJson("Page not found"))
	})
	r.GET("/path/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"content": "OK",
		})
	})
	return r
}
