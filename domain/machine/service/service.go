// Copyright 2024 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"context"

	"github.com/juju/errors"
)

// State describes retrieval and persistence methods for machines.
type State interface {
	// UpsertMachine persists the input machine entity.
	UpsertMachine(context.Context, string) (string, error)

	// DeleteMachine deletes the input machine entity.
	DeleteMachine(context.Context, string) error

	// InitialWatchStatement returns the table and the initial watch statement
	// for the machines.
	InitialWatchStatement() (string, string)
}

// Service provides the API for working with machines.
type Service struct {
	st State
}

// NewService returns a new service reference wrapping the input state.
func NewService(st State) *Service {
	return &Service{
		st: st,
	}
}

// CreateMachine creates the specified machine.
func (s *Service) CreateMachine(ctx context.Context, machineId string) (string, error) {
	machineUUID, err := s.st.UpsertMachine(ctx, machineId)
	return machineUUID, errors.Annotatef(err, "creating machine %q", machineId)
}

// DeleteMachine deletes the specified machine.
func (s *Service) DeleteMachine(ctx context.Context, machineId string) error {
	err := s.st.DeleteMachine(ctx, machineId)
	return errors.Annotatef(err, "deleting machine %q", machineId)
}
