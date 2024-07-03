// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package life

import (
	corelife "github.com/juju/juju/core/life"
)

// Life represents the life of an entity
// as recorded in the life lookup table.
type Life int

const (
	Alive Life = iota
	Dying
	Dead
)

// ToCoreLife converts a life value to a core life value.
func (l *Life) ToCoreLife() corelife.Value {
	switch *l {
	case Alive:
		return corelife.Alive
	case Dying:
		return corelife.Dying
	case Dead:
		return corelife.Dead
	}
	panic("unknown life value")
}

// FromCoreLife converts a core life value into a domain life value.
func FromCoreLife(l corelife.Value) Life {
	switch l {
	case corelife.Alive:
		return Alive
	case corelife.Dying:
		return Dying
	case corelife.Dead:
		return Dead
	}
	panic("unknown life value")
}
