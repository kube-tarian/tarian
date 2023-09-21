package clusteragent

import (
	"context"
	"sync"
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
)

// ConfigCache is responsible for caching and synchronizing constraints with the Tarian server.
type ConfigCache struct {
	configClient tarianpb.ConfigClient

	constraints            []*tarianpb.Constraint
	constraintsLock        sync.RWMutex
	constraintsInitialized bool

	context      context.Context
	logger       *logrus.Logger
	syncInterval time.Duration
}

// NewConfigCache creates a new ConfigCache instance and initializes it.
// It takes a context, logger, and a configClient which is a gRPC client for Tarian configuration.
func NewConfigCache(ctx context.Context, logger *logrus.Logger, configClient tarianpb.ConfigClient) *ConfigCache {
	c := &ConfigCache{
		context:                ctx,
		logger:                 logger,
		configClient:           configClient,
		syncInterval:           5 * time.Second,
		constraintsInitialized: false,
	}

	go c.Run()
	return c
}

// Run starts the synchronization loop for constraints.
// It periodically syncs constraints from the Tarian server based on the syncInterval.
// This function should be run as a goroutine.
func (cc *ConfigCache) Run() {
	for {
		cc.SyncConstraints()

		select {
		case <-time.After(cc.syncInterval):
		case <-cc.context.Done():
			return
		}
	}
}

// SyncConstraints fetches constraints from the Tarian server and updates the cache.
func (cc *ConfigCache) SyncConstraints() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := cc.configClient.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{})
	if err != nil {
		cc.logger.WithError(err).Error("error while getting constraints from the server")
	}

	cancel()
	cc.SetConstraints(r.GetConstraints())
	cc.constraintsInitialized = true
}

// SetConstraints sets the constraints in the cache.
func (cc *ConfigCache) SetConstraints(constraints []*tarianpb.Constraint) {
	cc.constraintsLock.Lock()
	defer cc.constraintsLock.Unlock()

	cc.constraints = constraints
}

// GetConstraints returns the cached constraints.
func (cc *ConfigCache) GetConstraints() []*tarianpb.Constraint {
	return cc.constraints
}

// IsConstraintInitialized checks if the constraints have been initialized.
// It returns true if the constraints have been synchronized at least once.
func (cc *ConfigCache) IsConstraintInitialized() bool {
	return cc.constraintsInitialized
}
