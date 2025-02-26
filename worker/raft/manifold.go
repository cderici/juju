// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package raft

import (
	"path/filepath"

	"github.com/hashicorp/raft"
	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/worker/v3"
	"github.com/juju/worker/v3/dependency"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/core/raftlease"
)

// ManifoldConfig holds the information necessary to run a raft
// worker in a dependency.Engine.
type ManifoldConfig struct {
	ClockName     string
	AgentName     string
	TransportName string

	FSM                  raft.FSM
	Logger               Logger
	PrometheusRegisterer prometheus.Registerer
	NewWorker            func(Config) (worker.Worker, error)
	NewApplier           func(Raft, ApplierMetrics, clock.Clock, Logger) LeaseApplier

	Queue Queue
}

// Validate validates the manifold configuration.
func (config ManifoldConfig) Validate() error {
	if config.ClockName == "" {
		return errors.NotValidf("empty ClockName")
	}
	if config.AgentName == "" {
		return errors.NotValidf("empty AgentName")
	}
	if config.TransportName == "" {
		return errors.NotValidf("empty TransportName")
	}
	if config.FSM == nil {
		return errors.NotValidf("nil FSM")
	}
	if config.Logger == nil {
		return errors.NotValidf("nil Logger")
	}
	if config.NewWorker == nil {
		return errors.NotValidf("nil NewWorker")
	}
	if config.Queue == nil {
		return errors.NotValidf("nil Queue")
	}
	if config.NewApplier == nil {
		return errors.NotValidf("nil NewApplier")
	}
	return nil
}

// Manifold returns a dependency.Manifold that will run a raft worker.
func Manifold(config ManifoldConfig) dependency.Manifold {
	return dependency.Manifold{
		Inputs: []string{
			config.ClockName,
			config.AgentName,
			config.TransportName,
		},
		Start:  config.start,
		Output: raftOutput,
	}
}

// start is a method on ManifoldConfig because it's more readable than a closure.
func (config ManifoldConfig) start(context dependency.Context) (worker.Worker, error) {
	if err := config.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	var clk clock.Clock
	if err := context.Get(config.ClockName, &clk); err != nil {
		return nil, errors.Trace(err)
	}

	var agent agent.Agent
	if err := context.Get(config.AgentName, &agent); err != nil {
		return nil, errors.Trace(err)
	}

	var transport raft.Transport
	if err := context.Get(config.TransportName, &transport); err != nil {
		return nil, errors.Trace(err)
	}

	agentConfig := agent.CurrentConfig()

	fsm := config.FSM
	if agentConfig.BatchRaftFSM() {
		if raftLeaseFSM, ok := fsm.(*raftlease.FSM); ok {
			fsm = raftlease.NewBatchFSM(raftLeaseFSM)
			config.Logger.Infof("Using batching FSM for processing leases")
		} else {
			// We shouldn't error out here, we can just report that we're not
			// using batching.
			config.Logger.Errorf("Unable to use batching FSM, unknown FSM type %T", fsm)
		}
	}

	// TODO(axw) make the directory path configurable, so we can
	// potentially have multiple Rafts. The dqlite raft should go
	// in <data-dir>/dqlite.
	raftDir := filepath.Join(agentConfig.DataDir(), "raft")

	w, err := config.NewWorker(Config{
		FSM:                      fsm,
		Logger:                   config.Logger,
		StorageDir:               raftDir,
		LocalID:                  raft.ServerID(agentConfig.Tag().Id()),
		Transport:                transport,
		Clock:                    clk,
		PrometheusRegisterer:     config.PrometheusRegisterer,
		Queue:                    config.Queue,
		NonSyncedWritesToRaftLog: agentConfig.NonSyncedWritesToRaftLog(),
		NewApplier:               config.NewApplier,
	})
	return w, errors.Trace(err)
}

func raftOutput(in worker.Worker, out interface{}) error {
	// We always expect the in worker to be a common.CleanupWorker, so we need
	// to unpack it first, to get the underlying worker.
	w, ok := in.(withRaftOutputs)
	if !ok {
		return errors.Errorf("expected input of type withRaftOutputs, got %T", in)
	}
	switch out := out.(type) {
	case **raft.Raft:
		r, err := w.Raft()
		if err != nil {
			return err
		}
		*out = r
	case *raft.LogStore:
		store, err := w.LogStore()
		if err != nil {
			return err
		}
		*out = store
	default:
		return errors.Errorf("expected output of **raft.Raft or *raft.LogStore, got %T", out)
	}
	return nil
}

type withRaftOutputs interface {
	Raft() (*raft.Raft, error)
	LogStore() (raft.LogStore, error)
}
