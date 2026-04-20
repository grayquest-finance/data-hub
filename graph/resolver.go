package graph

import (
	"data-hub/cache"
	"data-hub/client"
	"data-hub/config"

	"go.uber.org/zap"
)

// Resolver is the root resolver — it holds all shared dependencies.
// Created once in main.go and injected into the GraphQL handler.
// gqlgen generates the wiring between this struct and the schema in schema.resolvers.go.
type Resolver struct {
	AdminClient     *client.AdminClient
	KycClient       *client.KycClient
	RepaymentClient *client.RepaymentClient
	Cache           *cache.Cache
	Config          *config.Config
	Logger          *zap.Logger
}