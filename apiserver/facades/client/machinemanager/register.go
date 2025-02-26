// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package machinemanager

import (
	"reflect"

	"github.com/juju/errors"

	"github.com/juju/juju/apiserver/facade"
)

// Register is called to expose a package of facades onto a given registry.
func Register(registry facade.FacadeRegistry) {
	registry.MustRegister("MachineManager", 9, func(ctx facade.Context) (facade.Facade, error) {
		return newFacadeV9(ctx)
	}, reflect.TypeOf((*MachineManagerAPI)(nil)))
}

// newFacadeV9 creates a new server-side MachineManager API facade.
func newFacadeV9(ctx facade.Context) (*MachineManagerAPI, error) {
	machineManagerAPI, err := NewFacadeV8(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return machineManagerAPI, nil
}
