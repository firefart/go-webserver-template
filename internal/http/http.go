package http

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"

	"github.com/firefart/go-webserver-template/internal/config"
)

type Client struct {
	userAgent string
	client    *http.Client
	debug     bool
	logger    *slog.Logger
}

func NewHTTPClient(config config.Configuration, logger *slog.Logger, debugMode bool) (*Client, error) {
	// use default transport so proxy is respected
	tr := http.DefaultTransport.(*http.Transport)
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	if config.Proxy != nil && config.Proxy.URL != "" {
		proxy, err := newProxy(*config.Proxy)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy: %w", err)
		}
		tr.Proxy = proxy.ProxyFromConfig
	}
	httpClient := http.Client{
		Timeout:   config.Timeout,
		Transport: tr,
	}
	return &Client{
		userAgent: config.UserAgent,
		client:    &httpClient,
		debug:     debugMode,
		logger:    logger,
	}, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	if c.debug {
		reqDump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			c.logger.Error("error on DumpRequestOut", slog.String("err", err.Error()))
		} else {
			c.logger.Debug("sending http request", slog.String("req", string(reqDump)))
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if c.debug {
		respDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			c.logger.Error("error on DumpResponse", slog.String("err", err.Error()))
		} else {
			c.logger.Debug("got http response", slog.String("resp", string(respDump)))
		}
	}

	return resp, nil
}
