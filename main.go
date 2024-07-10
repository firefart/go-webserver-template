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

	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	"github.com/firefart/go-webserver-template/internal/server"

	"github.com/nikoksr/notify"

	_ "net/http/pprof"

	"go.uber.org/automaxprocs/maxprocs"
)

type application struct {
	logger *slog.Logger
	debug  bool
	config config.Configuration
	cache  *Cache[string]
	notify *notify.Notify
	db     *database.Database
}

func init() {
	// added in init to prevent the forced logline
	if _, err := maxprocs.Set(); err != nil {
		panic(fmt.Sprintf("Error on gomaxprocs: %v\n", err))
	}
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

	configuration, err := config.GetConfig(configFilename)
	if err != nil {
		return err
	}
	app.config = configuration

	app.notify, err = setupNotifications(configuration, logger)
	if err != nil {
		return err
	}

	db, err := database.New(ctx, configuration, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			app.logger.Error("error on database close", slog.String("err", err.Error()))
		}
	}()
	app.db = db

	app.cache = NewCache[string](ctx, logger, "cache", configuration.Cache.Timeout)

	app.logger.Info("Starting server",
		slog.String("host", configuration.Server.Listen),
		slog.Duration("gracefultimeout", configuration.Server.GracefulTimeout),
		slog.Duration("timeout", configuration.Timeout),
		slog.Bool("debug", app.debug),
	)

	s := server.NewServer(
		ctx,
		server.WithLogger(app.logger),
		server.WithConfig(app.config),
		server.WithDB(app.db),
		server.WithNotify(app.notify),
		server.WithDebug(app.debug),
	)

	srv := &http.Server{
		Addr:         configuration.Server.Listen,
		Handler:      s,
		ReadTimeout:  configuration.Timeout,
		WriteTimeout: configuration.Timeout,
	}

	if configuration.Server.TLS.PublicKey != "" && configuration.Server.TLS.PrivateKey != "" {
		tlsConfig, err := app.setupTLSConfig()
		if err != nil {
			return err
		}
		srv.TLSConfig = tlsConfig

		go func() {
			if err := srv.ListenAndServeTLS(configuration.Server.TLS.PublicKey, configuration.Server.TLS.PrivateKey); err != nil && !errors.Is(err, http.ErrServerClosed) {
				app.logger.Error("error on listenandserveTLS", slog.String("err", err.Error()))
				// emit signal to kill server
				cancel()
			}
		}()
	} else {
		go func() {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				app.logger.Error("error on listenandserve", slog.String("err", err.Error()))
				// emit signal to kill server
				cancel()
			}
		}()
	}

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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), configuration.Server.GracefulTimeout)
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
