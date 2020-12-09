// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrevisionupdater_test

import (
	"github.com/juju/charm/v8"
	"github.com/juju/errors"
	"github.com/juju/names/v4"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver/facade"
	"github.com/juju/juju/apiserver/facades/controller/charmrevisionupdater"
	"github.com/juju/juju/apiserver/facades/controller/charmrevisionupdater/testing"
	apiservertesting "github.com/juju/juju/apiserver/testing"
	"github.com/juju/juju/charmstore"
	jujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state"
)

type charmVersionSuite struct {
	testing.CharmSuite
	jujutesting.JujuConnSuite

	charmrevisionupdater *charmrevisionupdater.CharmRevisionUpdaterAPI
	authoriser           apiservertesting.FakeAuthorizer
}

var _ = gc.Suite(&charmVersionSuite{})

func (s *charmVersionSuite) SetUpSuite(c *gc.C) {
	s.JujuConnSuite.SetUpSuite(c)
	s.CharmSuite.SetUpSuite(c, &s.JujuConnSuite)
}

func (s *charmVersionSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)
	s.CharmSuite.SetUpTest(c)

	s.authoriser = apiservertesting.FakeAuthorizer{
		Controller: true,
		Tag:        names.NewMachineTag("99"),
	}

	state := charmrevisionupdater.StateShim{State: s.State}
	newClient := func(st charmrevisionupdater.State) (charmstore.Client, error) {
		return charmstore.NewCustomClient(s.Store), nil
	}

	var err error
	s.charmrevisionupdater, err = charmrevisionupdater.NewCharmRevisionUpdaterAPIState(state, newClient, nil)
	c.Assert(err, jc.ErrorIsNil)
}

type facadeContextShim struct {
	facade.Context // Make it fulfil the interface, but we only define a couple of methods
	state          *state.State
	authorizer     facade.Authorizer
}

func (s facadeContextShim) Auth() facade.Authorizer {
	return s.authorizer
}

func (s facadeContextShim) State() *state.State {
	return s.state
}

func (s *charmVersionSuite) TestNewCharmRevisionUpdaterAPIAcceptsStateManager(c *gc.C) {
	facadeCtx := facadeContextShim{state: s.State, authorizer: s.authoriser}
	endPoint, err := charmrevisionupdater.NewCharmRevisionUpdaterAPI(facadeCtx)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(endPoint, gc.NotNil)
}

func (s *charmVersionSuite) TestNewCharmRevisionUpdaterAPIRefusesNonStateManager(c *gc.C) {
	s.authoriser.Controller = false
	facadeCtx := facadeContextShim{state: s.State, authorizer: s.authoriser}
	endPoint, err := charmrevisionupdater.NewCharmRevisionUpdaterAPI(facadeCtx)
	c.Assert(endPoint, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, "permission denied")
}

func (s *charmVersionSuite) TestUpdateRevisions(c *gc.C) {
	s.AddMachine(c, "0", state.JobManageModel)
	s.SetupScenario(c)

	curl := charm.MustParseURL("cs:quantal/mysql")
	_, err := s.State.LatestPlaceholderCharm(curl)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)

	curl = charm.MustParseURL("cs:quantal/wordpress")
	_, err = s.State.LatestPlaceholderCharm(curl)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)

	result, err := s.charmrevisionupdater.UpdateLatestRevisions()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)

	curl = charm.MustParseURL("cs:quantal/mysql")
	pending, err := s.State.LatestPlaceholderCharm(curl)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(pending.String(), gc.Equals, "cs:quantal/mysql-23")

	// Latest wordpress is already deployed, so no pending charm.
	curl = charm.MustParseURL("cs:quantal/wordpress")
	_, err = s.State.LatestPlaceholderCharm(curl)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)

	// Varnish has an error when updating, so no pending charm.
	curl = charm.MustParseURL("cs:quantal/varnish")
	_, err = s.State.LatestPlaceholderCharm(curl)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)

	// Update mysql version and run update again.
	app, err := s.State.Application("mysql")
	c.Assert(err, jc.ErrorIsNil)
	ch := s.AddCharmWithRevision(c, "mysql", 23)
	cfg := state.SetCharmConfig{
		Charm:      ch,
		ForceUnits: true,
	}
	err = app.SetCharm(cfg)
	c.Assert(err, jc.ErrorIsNil)

	result, err = s.charmrevisionupdater.UpdateLatestRevisions()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)

	// Latest mysql is now deployed, so no pending charm.
	curl = charm.MustParseURL("cs:quantal/mysql")
	_, err = s.State.LatestPlaceholderCharm(curl)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
}

func (s *charmVersionSuite) TestWordpressCharmNoReadAccessIsNotVisible(c *gc.C) {
	s.AddMachine(c, "0", state.JobManageModel)
	s.SetupScenario(c)

	// Disallow read access to the wordpress charm in the charm store.
	s.CharmSuite.SetStoreError("wordpress", errors.New("boom"))

	// Run the revision updater and check that the public charm updates are
	// still properly notified.
	result, err := s.charmrevisionupdater.UpdateLatestRevisions()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)

	curl := charm.MustParseURL("cs:quantal/mysql")
	pending, err := s.State.LatestPlaceholderCharm(curl)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(pending.String(), gc.Equals, "cs:quantal/mysql-23")

	// No pending charm for wordpress.
	curl = charm.MustParseURL("cs:quantal/wordpress")
	_, err = s.State.LatestPlaceholderCharm(curl)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
}
