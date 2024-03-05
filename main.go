package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nikoksr/notify"

	_ "net/http/pprof"

	_ "go.uber.org/automaxprocs"
)

var secretKeyHeaderName = http.CanonicalHeaderKey("X-Secret-Key-Header")
var cloudflareIPHeaderName = http.CanonicalHeaderKey("CF-Connecting-IP")

type application struct {
	logger *slog.Logger
	debug  bool
	config Configuration
	cache  *Cache[string]
	notify *notify.Notify
}

func main() {
	var debugMode bool
	var configFilename string
	var jsonOutput bool
	flag.BoolVar(&debugMode, "debug", false, "Enable DEBUG mode")
	flag.StringVar(&configFilename, "config", "", "config file to use")
	flag.BoolVar(&jsonOutput, "json", false, "output in json instead")
	flag.Parse()

	logger := newLogger(debugMode, jsonOutput)
	ctx := context.Background()
	if err := run(ctx, logger, configFilename, debugMode); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, configFilename string, debug bool) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	app := &application{
		logger: logger,
		debug:  debug,
	}

	if configFilename == "" {
		return fmt.Errorf("please provide a config file")
	}

	config, err := GetConfig(configFilename)
	if err != nil {
		return err
	}
	app.config = config

	app.notify, err = setupNotifications(config, logger)
	if err != nil {
		return err
	}

	app.logger.Info("Starting server",
		slog.String("host", config.Server.Listen),
		slog.Duration("gracefultimeout", config.Server.GracefulTimeout),
		slog.Duration("timeout", config.Timeout),
		slog.Bool("debug", app.debug),
	)

	app.cache = NewCache[string](ctx, logger, "cache", config.Cache.Timeout)

	tlsConfig, err := app.setupTLSConfig()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:         config.Server.Listen,
		Handler:      app.newServer(ctx),
		TLSConfig:    tlsConfig,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("error on listenandserve", slog.String("err", err.Error()))
			// emit signal to kill server
			cancel()
		}
	}()

	app.logger.Info("Starting pprof server",
		slog.String("host", app.config.Server.PprofListen),
	)

	pprofSrv := &http.Server{
		Addr: app.config.Server.PprofListen,
	}
	go func() {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/debug/pprof/", http.DefaultServeMux)
		if err := pprofSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("error on pprof listenandserve", slog.String("err", err.Error()))
			// emit signal to kill server
			cancel()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// wait for a signal
		<-ctx.Done()
		app.logger.Info("received shutdown signal")
		// create a new context for shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), config.Server.GracefulTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			app.logger.Error("error on srv shutdown", slog.String("err", err.Error()))
		}
		if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
			app.logger.Error("error on pprofsrv shutdown", slog.String("err", err.Error()))
		}
	}()
	wg.Wait()
	return nil
}
