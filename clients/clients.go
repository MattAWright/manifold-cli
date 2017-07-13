package clients

import (
	"fmt"
	"net/url"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/manifoldco/manifold-cli/config"
	iClient "github.com/manifoldco/manifold-cli/generated/identity/client"
	mClient "github.com/manifoldco/manifold-cli/generated/marketplace/client"
)

// NewIdentity returns a swagger generated client for the Identity service
func NewIdentity(cfg *config.Config) (*iClient.Identity, error) {
	u, err := deriveURL(cfg, "identity")
	if err != nil {
		return nil, err
	}

	c := iClient.DefaultTransportConfig()
	c.WithHost(u.Host)
	c.WithBasePath(u.Path)
	c.WithSchemes([]string{u.Scheme})

	transport := httptransport.New(c.Host, c.BasePath, c.Schemes)

	if cfg.AuthToken != "" {
		transport.DefaultAuthentication = NewBearerToken(cfg.AuthToken)
	}

	return iClient.New(transport, strfmt.Default), nil
}

// NewMarketplace returns a swagger generated client for the Marketplace service
func NewMarketplace(cfg *config.Config) (*mClient.Marketplace, error) {
	u, err := deriveURL(cfg, "marketplace")
	if err != nil {
		return nil, err
	}

	c := mClient.DefaultTransportConfig()
	c.WithHost(u.Host)
	c.WithBasePath(u.Path)
	c.WithSchemes([]string{u.Scheme})

	transport := httptransport.New(c.Host, c.BasePath, c.Schemes)

	if cfg.AuthToken != "" {
		transport.DefaultAuthentication = NewBearerToken(cfg.AuthToken)
	}

	return mClient.New(transport, strfmt.Default), nil
}

// NewBearerToken returns a bearer token authenticator for use with a
// go-swagger generated client.
func NewBearerToken(token string) runtime.ClientAuthInfoWriter {
	return httptransport.BearerToken(token)
}

func deriveURL(cfg *config.Config, service string) (*url.URL, error) {
	u := fmt.Sprintf("%s://api.%s.%s/v1", cfg.TransportScheme, service, cfg.Hostname)
	return url.Parse(u)
}
