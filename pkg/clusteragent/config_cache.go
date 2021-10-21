package clusteragent

import (
	"context"
	"sync"
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

type ConfigCache struct {
	configClient tarianpb.ConfigClient

	constraints            []*tarianpb.Constraint
	constraintsLock        sync.RWMutex
	constraintsInitialized bool

	context context.Context

	syncInterval time.Duration
}

func NewConfigCache(ctx context.Context, configClient tarianpb.ConfigClient) *ConfigCache {
	c := &ConfigCache{configClient: configClient, constraintsInitialized: false, context: ctx, syncInterval: 5 * time.Second}

	ctx.Done()
	return c
}

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

func (cc *ConfigCache) SyncConstraints() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := cc.configClient.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{})

	if err != nil {
		logger.Errorw("error while getting constraints from the server", "err", err)
	}

	cancel()

	cc.SetConstraints(r.GetConstraints())

	cc.constraintsInitialized = true
}

func (cc *ConfigCache) SetConstraints(constraints []*tarianpb.Constraint) {
	cc.constraintsLock.Lock()
	defer cc.constraintsLock.Unlock()

	cc.constraints = constraints
}

func (cc *ConfigCache) GetConstraints() []*tarianpb.Constraint {
	return cc.constraints
}

func (cc *ConfigCache) IsConstraintInitialized() bool {
	return cc.constraintsInitialized
}
