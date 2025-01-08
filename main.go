package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/firefart/go-webserver-template/internal/cacher"
	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	"github.com/firefart/go-webserver-template/internal/http"
	"github.com/firefart/go-webserver-template/internal/mail"
	"github.com/firefart/go-webserver-template/internal/metrics"
	"github.com/firefart/go-webserver-template/internal/server"
	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/automaxprocs/maxprocs"

	_ "net/http/pprof"
)

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
	var version bool
	var configCheckMode bool
	flag.BoolVar(&debugMode, "debug", false, "Enable DEBUG mode")
	flag.StringVar(&configFilename, "config", "", "config file to use")
	flag.BoolVar(&jsonOutput, "json", false, "output in json instead")
	flag.BoolVar(&configCheckMode, "configcheck", false, "just check the config")
	flag.BoolVar(&version, "version", false, "show version")
	flag.Parse()

	if version {
		buildInfo, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Println("Unable to determine version information")
			os.Exit(1)
		}
		fmt.Printf("%s", buildInfo)
		os.Exit(0)
	}

	logger := newLogger(debugMode, jsonOutput)
	ctx := context.Background()
	var err error
	if configCheckMode {
		err = configCheck(configFilename)
	} else {
		err = run(ctx, logger, configFilename, debugMode)
	}

	if err != nil {
		// check if we have a multierror
		var merr *multierror.Error
		if errors.As(err, &merr) {
			for _, e := range merr.Errors {
				logger.Error(e.Error())
			}
			os.Exit(1)
		}
		// a normal error
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func configCheck(configFilename string) error {
	_, err := config.GetConfig(configFilename)
	return err
}

func run(ctx context.Context, logger *slog.Logger, configFilename string, debugMode bool) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	if configFilename == "" {
		return fmt.Errorf("please provide a config file")
	}

	configuration, err := config.GetConfig(configFilename)
	if err != nil {
		return err
	}

	notify, err := setupNotifications(configuration, logger)
	if err != nil {
		return err
	}

	db, err := database.New(ctx, configuration, logger, debugMode)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(configuration.Server.GracefulTimeout); err != nil {
			logger.Error("error on database close", slog.String("err", err.Error()))
		}
	}()

	reg := prometheus.NewRegistry()
	m, err := metrics.NewMetrics(reg)
	if err != nil {
		return err
	}

	cache := cacher.New[string](ctx, logger, m, "cache", configuration.Cache.Timeout)

	httpClient, err := http.NewHTTPClient(configuration, logger, debugMode)
	if err != nil {
		return err
	}

	logger.Info("Starting server",
		slog.String("host", configuration.Server.Listen),
		slog.Duration("gracefultimeout", configuration.Server.GracefulTimeout),
		slog.Duration("timeout", configuration.Timeout),
		slog.Bool("debug", debugMode),
	)

	options := []server.OptionsServerFunc{
		server.WithLogger(logger),
		server.WithConfig(configuration),
		server.WithDB(db),
		server.WithNotify(notify),
		server.WithDebug(debugMode),
		server.WithMetrics(m, reg),
		server.WithCache(cache),
		server.WithHTTPClient(httpClient),
	}

	if configuration.Mail.Enabled {
		mailer, err := mail.New(configuration, logger)
		if err != nil {
			return err
		}
		options = append(options, server.WithMailer(mailer))
	}

	s := server.NewServer(
		ctx,
		options...,
	)

	srv := &nethttp.Server{
		Addr:         configuration.Server.Listen,
		Handler:      s,
		ReadTimeout:  configuration.Timeout,
		WriteTimeout: configuration.Timeout,
	}

	if configuration.Server.TLS.PublicKey != "" && configuration.Server.TLS.PrivateKey != "" {
		tlsConfig, err := setupTLSConfig(logger, configuration)
		if err != nil {
			return err
		}
		srv.TLSConfig = tlsConfig

		go func() {
			if err := srv.ListenAndServeTLS(configuration.Server.TLS.PublicKey, configuration.Server.TLS.PrivateKey); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
				logger.Error("error on listenandserveTLS", slog.String("err", err.Error()))
				// emit signal to kill server
				cancel()
			}
		}()
	} else {
		go func() {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
				logger.Error("error on listenandserve", slog.String("err", err.Error()))
				// emit signal to kill server
				cancel()
			}
		}()
	}

	logger.Info("Starting pprof server",
		slog.String("host", configuration.Server.PprofListen),
	)

	pprofSrv := &nethttp.Server{
		Addr: configuration.Server.PprofListen,
	}
	go func() {
		pprofMux := nethttp.NewServeMux()
		// pprof is automatically added to the DefaultServeMux
		// this is not used anywhere so it's safe to use the auto
		// implementation. The following line makes sure we use
		// the configuration from defaultservermux
		// If you ever want to use the default serve mux: don't
		// otherwise you will expose the debug endpoints
		pprofMux.Handle("/debug/pprof/", nethttp.DefaultServeMux) // pprof is defined on the default servemux
		pprofSrv.Handler = pprofMux
		if err := pprofSrv.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			logger.Error("error on pprof", slog.String("err", err.Error()))
			cancel()
		}
	}()

	logger.Info("Starting metrics server",
		slog.String("host", configuration.Server.MetricsListen),
	)

	metricsSrv := &nethttp.Server{
		Addr: configuration.Server.MetricsListen,
	}
	go func() {
		metricsMux := nethttp.NewServeMux()
		metricsMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
		metricsSrv.Handler = metricsMux
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			logger.Error("error on metric", slog.String("err", err.Error()))
			cancel()
		}
	}()

	// wait for a signal
	<-ctx.Done()
	logger.Info("received shutdown signal")
	// create a new context for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), configuration.Server.GracefulTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("error on srv shutdown", slog.String("err", err.Error()))
	}
	if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("error on pprofsrv shutdown", slog.String("err", err.Error()))
	}
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("error on metricsSrv shutdown", slog.String("err", err.Error()))
	}
	return nil
}
