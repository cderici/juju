// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package firewaller_test

import (
	"context"

	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/worker/v4"
	"github.com/juju/worker/v4/dependency"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/api"
	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/controller/remoterelations"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	loggertesting "github.com/juju/juju/internal/logger/testing"
	coretesting "github.com/juju/juju/internal/testing"
	"github.com/juju/juju/internal/worker/common"
	"github.com/juju/juju/internal/worker/firewaller"
)

type ManifoldSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&ManifoldSuite{})

func (s *ManifoldSuite) TestManifoldFirewallModeNone(c *gc.C) {
	ctx := &mockDependencyGetter{
		env: &mockEnviron{
			config: coretesting.CustomModelConfig(c, coretesting.Attrs{
				"firewall-mode": config.FwNone,
			}),
		},
	}

	manifold := firewaller.Manifold(validConfig(c))
	_, err := manifold.Start(context.Background(), ctx)
	c.Assert(err, gc.Equals, dependency.ErrUninstall)
}

type mockDependencyGetter struct {
	dependency.Getter
	env *mockEnviron
}

func (m *mockDependencyGetter) Get(name string, out interface{}) error {
	if name == "environ" {
		*(out.(*environs.Environ)) = m.env
	}
	return nil
}

type mockEnviron struct {
	environs.Environ
	config *config.Config
}

func (e *mockEnviron) Config() *config.Config {
	return e.config
}

type ManifoldConfigSuite struct {
	testing.IsolationSuite
	config firewaller.ManifoldConfig
}

var _ = gc.Suite(&ManifoldConfigSuite{})

func (s *ManifoldConfigSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)
	s.config = validConfig(c)
}

func validConfig(c *gc.C) firewaller.ManifoldConfig {
	return firewaller.ManifoldConfig{
		AgentName:                    "agent",
		APICallerName:                "api-caller",
		EnvironName:                  "environ",
		ServiceFactoryName:           "service-factory",
		Logger:                       loggertesting.WrapCheckLog(c),
		NewControllerConnection:      func(*api.Info) (api.Connection, error) { return nil, nil },
		NewFirewallerFacade:          func(base.APICaller) (firewaller.FirewallerAPI, error) { return nil, nil },
		NewFirewallerWorker:          func(firewaller.Config) (worker.Worker, error) { return nil, nil },
		NewRemoteRelationsFacade:     func(base.APICaller) *remoterelations.Client { return nil },
		NewCredentialValidatorFacade: func(base.APICaller) (common.CredentialAPI, error) { return nil, nil },
		GetMachineService:            func(dependency.Getter, string) (firewaller.MachineService, error) { return nil, nil },
	}
}

func (s *ManifoldConfigSuite) TestValid(c *gc.C) {
	c.Check(s.config.Validate(), jc.ErrorIsNil)
}

func (s *ManifoldConfigSuite) TestMissingAgentName(c *gc.C) {
	s.config.AgentName = ""
	s.checkNotValid(c, "empty AgentName not valid")
}

func (s *ManifoldConfigSuite) TestMissingAPICallerName(c *gc.C) {
	s.config.APICallerName = ""
	s.checkNotValid(c, "empty APICallerName not valid")
}

func (s *ManifoldConfigSuite) TestMissingEnvironName(c *gc.C) {
	s.config.EnvironName = ""
	s.checkNotValid(c, "empty EnvironName not valid")
}

func (s *ManifoldConfigSuite) TestMissingLogger(c *gc.C) {
	s.config.Logger = nil
	s.checkNotValid(c, "nil Logger not valid")
}

func (s *ManifoldConfigSuite) TestMissingServiceFactoryName(c *gc.C) {
	s.config.ServiceFactoryName = ""
	s.checkNotValid(c, "empty ServiceFactoryName not valid")
}

func (s *ManifoldConfigSuite) TestMissingGetMachineService(c *gc.C) {
	s.config.GetMachineService = nil
	s.checkNotValid(c, "nil GetMachineService not valid")
}

func (s *ManifoldConfigSuite) TestMissingNewFirewallerFacade(c *gc.C) {
	s.config.NewFirewallerFacade = nil
	s.checkNotValid(c, "nil NewFirewallerFacade not valid")
}

func (s *ManifoldConfigSuite) TestMissingNewFirewallerWorker(c *gc.C) {
	s.config.NewFirewallerWorker = nil
	s.checkNotValid(c, "nil NewFirewallerWorker not valid")
}

func (s *ManifoldConfigSuite) TestMissingNewControllerConnection(c *gc.C) {
	s.config.NewControllerConnection = nil
	s.checkNotValid(c, "nil NewControllerConnection not valid")
}

func (s *ManifoldConfigSuite) TestMissingNewRemoteRelationsFacade(c *gc.C) {
	s.config.NewRemoteRelationsFacade = nil
	s.checkNotValid(c, "nil NewRemoteRelationsFacade not valid")
}

func (s *ManifoldConfigSuite) TestMissingNewCredentialValidatorFacade(c *gc.C) {
	s.config.NewCredentialValidatorFacade = nil
	s.checkNotValid(c, "nil NewCredentialValidatorFacade not valid")
}

func (s *ManifoldConfigSuite) checkNotValid(c *gc.C, expect string) {
	err := s.config.Validate()
	c.Check(err, gc.ErrorMatches, expect)
	c.Check(err, jc.ErrorIs, errors.NotValid)
}
