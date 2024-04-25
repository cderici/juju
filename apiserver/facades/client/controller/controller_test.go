// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package controller_test

import (
	"context"
	stdcontext "context"
	"encoding/json"
	"regexp"
	"time"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/loggo/v2"
	"github.com/juju/names/v5"
	"github.com/juju/pubsub/v2"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version/v2"
	"github.com/juju/worker/v4/workertest"
	"github.com/kr/pretty"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"
	gc "gopkg.in/check.v1"
	"gopkg.in/macaroon.v2"

	"github.com/juju/juju/apiserver"
	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/apiserver/facade"
	"github.com/juju/juju/apiserver/facade/facadetest"
	"github.com/juju/juju/apiserver/facades/client/controller"
	"github.com/juju/juju/apiserver/facades/client/controller/mocks"
	apiservertesting "github.com/juju/juju/apiserver/testing"
	"github.com/juju/juju/cloud"
	corecontroller "github.com/juju/juju/controller"
	coremultiwatcher "github.com/juju/juju/core/multiwatcher"
	"github.com/juju/juju/core/permission"
	"github.com/juju/juju/core/watcher/registry"
	"github.com/juju/juju/domain/access"
	servicefactorytesting "github.com/juju/juju/domain/servicefactory/testing"
	"github.com/juju/juju/environs"
	environscloudspec "github.com/juju/juju/environs/cloudspec"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/internal/docker"
	loggertesting "github.com/juju/juju/internal/logger/testing"
	pscontroller "github.com/juju/juju/internal/pubsub/controller"
	"github.com/juju/juju/internal/uuid"
	"github.com/juju/juju/internal/worker/multiwatcher"
	jujujujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/rpc/params"
	"github.com/juju/juju/state"
	statetesting "github.com/juju/juju/state/testing"
	"github.com/juju/juju/testing"
	"github.com/juju/juju/testing/factory"
)

type controllerSuite struct {
	statetesting.StateSuite
	servicefactorytesting.ServiceFactorySuite

	controllerConfigAttrs map[string]any

	controller      *controller.ControllerAPI
	resources       *common.Resources
	watcherRegistry facade.WatcherRegistry
	authorizer      apiservertesting.FakeAuthorizer
	hub             *pubsub.StructuredHub
	context         facadetest.ModelContext
}

var _ = gc.Suite(&controllerSuite{})

func (s *controllerSuite) SetUpSuite(c *gc.C) {
	s.StateSuite.SetUpSuite(c)
	s.ServiceFactorySuite.SetUpSuite(c)
}

func (s *controllerSuite) SetUpTest(c *gc.C) {
	if s.controllerConfigAttrs == nil {
		s.controllerConfigAttrs = map[string]any{}
	}
	// Initial config needs to be set before the StateSuite SetUpTest.
	s.InitialConfig = testing.CustomModelConfig(c, testing.Attrs{
		"name": "controller",
	})
	controllerCfg := testing.FakeControllerConfig()
	for key, value := range s.controllerConfigAttrs {
		controllerCfg[key] = value
	}

	s.StateSuite.ControllerConfig = controllerCfg
	s.StateSuite.SetUpTest(c)
	s.ServiceFactorySuite.SetUpTest(c)
	jujujujutesting.SeedDatabase(c, s.ControllerSuite.TxnRunner(), controllerCfg)

	allWatcherBacking, err := state.NewAllWatcherBacking(s.StatePool)
	c.Assert(err, jc.ErrorIsNil)
	multiWatcherWorker, err := multiwatcher.NewWorker(multiwatcher.Config{
		Clock:                clock.WallClock,
		Logger:               loggertesting.WrapCheckLog(c),
		Backing:              allWatcherBacking,
		PrometheusRegisterer: noopRegisterer{},
	})
	c.Assert(err, jc.ErrorIsNil)
	// The worker itself is a coremultiwatcher.Factory.
	s.AddCleanup(func(c *gc.C) { workertest.CleanKill(c, multiWatcherWorker) })

	s.hub = pubsub.NewStructuredHub(nil)

	s.watcherRegistry, err = registry.NewRegistry(clock.WallClock)
	c.Assert(err, jc.ErrorIsNil)
	s.AddCleanup(func(c *gc.C) { workertest.DirtyKill(c, s.watcherRegistry) })

	s.resources = common.NewResources()
	s.AddCleanup(func(_ *gc.C) { s.resources.StopAll() })

	s.authorizer = apiservertesting.FakeAuthorizer{
		Tag:      s.Owner,
		AdminTag: s.Owner,
	}

	s.context = facadetest.ModelContext{
		State_:               s.State,
		StatePool_:           s.StatePool,
		Resources_:           s.resources,
		WatcherRegistry_:     s.watcherRegistry,
		Auth_:                s.authorizer,
		Hub_:                 s.hub,
		MultiwatcherFactory_: multiWatcherWorker,
		ServiceFactory_:      s.ControllerServiceFactory(c),
	}
	controller, err := controller.LatestAPI(context.Background(), s.context)
	c.Assert(err, jc.ErrorIsNil)
	s.controller = controller

	loggo.GetLogger("juju.apiserver.controller").SetLogLevel(loggo.TRACE)
}

func (s *controllerSuite) TearDownTest(c *gc.C) {
	s.StateSuite.TearDownTest(c)
	s.ServiceFactorySuite.TearDownTest(c)
}

func (s *controllerSuite) TestNewAPIRefusesNonClient(c *gc.C) {
	anAuthoriser := apiservertesting.FakeAuthorizer{
		Tag: names.NewUnitTag("mysql/0"),
	}
	endPoint, err := controller.LatestAPI(context.Background(), facadetest.ModelContext{
		State_:          s.State,
		Resources_:      s.resources,
		Auth_:           anAuthoriser,
		ServiceFactory_: s.ControllerServiceFactory(c),
	})
	c.Assert(endPoint, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, "permission denied")
}

func (s *controllerSuite) checkModelMatches(c *gc.C, model params.Model, expected *state.Model) {
	c.Check(model.Name, gc.Equals, expected.Name())
	c.Check(model.UUID, gc.Equals, expected.UUID())
	c.Check(model.OwnerTag, gc.Equals, expected.Owner().String())
}

func (s *controllerSuite) TestAllModels(c *gc.C) {
	admin := s.Factory.MakeUser(c, &factory.UserParams{Name: "foobar"})

	s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "owned", Owner: admin.UserTag()}).Close()
	remoteUserTag := names.NewUserTag("user@remote")
	st := s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "user", Owner: remoteUserTag})
	defer func() { _ = st.Close() }()
	model, err := st.Model()
	c.Assert(err, jc.ErrorIsNil)

	model.AddUser(
		state.UserAccessSpec{
			User:        admin.UserTag(),
			CreatedBy:   remoteUserTag,
			DisplayName: "Foo Bar",
			Access:      permission.WriteAccess})

	s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "no-access", Owner: remoteUserTag}).Close()

	response, err := s.controller.AllModels(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)
	// The results are sorted.
	expected := []string{"controller", "no-access", "owned", "user"}
	var obtained []string
	for _, userModel := range response.UserModels {
		c.Assert(userModel.Type, gc.Equals, "iaas")
		obtained = append(obtained, userModel.Name)
		stateModel, ph, err := s.StatePool.GetModel(userModel.UUID)
		c.Assert(err, jc.ErrorIsNil)
		defer ph.Release()
		s.checkModelMatches(c, userModel.Model, stateModel)
	}
	c.Assert(obtained, jc.DeepEquals, expected)
}

func (s *controllerSuite) TestHostedModelConfigs_OnlyHostedModelsReturned(c *gc.C) {
	owner := s.Factory.MakeUser(c, nil)
	s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "first", Owner: owner.UserTag()}).Close()
	remoteUserTag := names.NewUserTag("user@remote")
	s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "second", Owner: remoteUserTag}).Close()

	results, err := s.controller.HostedModelConfigs(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(len(results.Models), gc.Equals, 2)

	one := results.Models[0]
	two := results.Models[1]

	c.Assert(one.Name, gc.Equals, "first")
	c.Assert(one.OwnerTag, gc.Equals, owner.UserTag().String())
	c.Assert(two.Name, gc.Equals, "second")
	c.Assert(two.OwnerTag, gc.Equals, remoteUserTag.String())
}

func (s *controllerSuite) makeCloudSpec(c *gc.C, pSpec *params.CloudSpec) environscloudspec.CloudSpec {
	c.Assert(pSpec, gc.NotNil)
	var credential *cloud.Credential
	if pSpec.Credential != nil {
		credentialValue := cloud.NewCredential(
			cloud.AuthType(pSpec.Credential.AuthType),
			pSpec.Credential.Attributes,
		)
		credential = &credentialValue
	}
	spec := environscloudspec.CloudSpec{
		Type:             pSpec.Type,
		Name:             pSpec.Name,
		Region:           pSpec.Region,
		Endpoint:         pSpec.Endpoint,
		IdentityEndpoint: pSpec.IdentityEndpoint,
		StorageEndpoint:  pSpec.StorageEndpoint,
		Credential:       credential,
	}
	c.Assert(spec.Validate(), jc.ErrorIsNil)
	return spec
}

func (s *controllerSuite) TestHostedModelConfigs_CanOpenEnviron(c *gc.C) {
	owner := s.Factory.MakeUser(c, nil)
	_ = s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "first", Owner: owner.UserTag()}).Close()
	remoteUserTag := names.NewUserTag("user@remote")
	_ = s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "second", Owner: remoteUserTag}).Close()

	results, err := s.controller.HostedModelConfigs(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(len(results.Models), gc.Equals, 2)

	for _, model := range results.Models {
		c.Assert(model.Error, gc.IsNil)

		cfg, err := config.New(config.NoDefaults, model.Config)
		c.Assert(err, jc.ErrorIsNil)
		spec := s.makeCloudSpec(c, model.CloudSpec)
		_, err = environs.New(stdcontext.Background(), environs.OpenParams{
			Cloud:  spec,
			Config: cfg,
		})
		c.Assert(err, jc.ErrorIsNil)
	}
}

func (s *controllerSuite) TestListBlockedModels(c *gc.C) {
	st := s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "test"})
	defer func() { _ = st.Close() }()

	_ = s.State.SwitchBlockOn(state.DestroyBlock, "TestBlockDestroyModel")
	_ = s.State.SwitchBlockOn(state.ChangeBlock, "TestChangeBlock")
	_ = st.SwitchBlockOn(state.DestroyBlock, "TestBlockDestroyModel")
	_ = st.SwitchBlockOn(state.ChangeBlock, "TestChangeBlock")

	list, err := s.controller.ListBlockedModels(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(list.Models, jc.DeepEquals, []params.ModelBlockInfo{
		{
			Name:     "controller",
			UUID:     s.State.ModelUUID(),
			OwnerTag: s.Owner.String(),
			Blocks: []string{
				"BlockDestroy",
				"BlockChange",
			},
		},
		{
			Name:     "test",
			UUID:     st.ModelUUID(),
			OwnerTag: s.Owner.String(),
			Blocks: []string{
				"BlockDestroy",
				"BlockChange",
			},
		},
	})

}

func (s *controllerSuite) TestListBlockedModelsNoBlocks(c *gc.C) {
	list, err := s.controller.ListBlockedModels(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(list.Models, gc.HasLen, 0)
}

func (s *controllerSuite) TestModelConfig(c *gc.C) {
	cfg, err := s.controller.ModelConfig(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cfg.Config["name"], jc.DeepEquals, params.ConfigValue{Value: "controller"})
}

func (s *controllerSuite) TestModelConfigFromNonController(c *gc.C) {
	st := s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "test"})
	defer func() { _ = st.Close() }()

	authorizer := &apiservertesting.FakeAuthorizer{
		Tag:      s.Owner,
		AdminTag: s.Owner,
	}
	controller, err := controller.NewControllerAPIv11(
		context.Background(),
		facadetest.ModelContext{
			State_:          st,
			StatePool_:      s.StatePool,
			Resources_:      common.NewResources(),
			Auth_:           authorizer,
			ServiceFactory_: s.ControllerServiceFactory(c),
		})

	c.Assert(err, jc.ErrorIsNil)
	cfg, err := controller.ModelConfig(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cfg.Config["name"], jc.DeepEquals, params.ConfigValue{Value: "controller"})
}

func (s *controllerSuite) TestControllerConfig(c *gc.C) {
	cfg, err := s.controller.ControllerConfig(context.Background())
	c.Assert(err, jc.ErrorIsNil)

	controllerConfigService := s.ControllerServiceFactory(c).ControllerConfig()

	cfgFromDB, err := controllerConfigService.ControllerConfig(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cfg.Config["controller-uuid"], gc.Equals, cfgFromDB.ControllerUUID())
	c.Assert(cfg.Config["state-port"], gc.Equals, cfgFromDB.StatePort())
	c.Assert(cfg.Config["api-port"], gc.Equals, cfgFromDB.APIPort())
}

func (s *controllerSuite) TestControllerConfigFromNonController(c *gc.C) {
	st := s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "test"})
	defer func() { _ = st.Close() }()

	authorizer := &apiservertesting.FakeAuthorizer{Tag: s.Owner}
	controller, err := controller.NewControllerAPIv11(
		context.Background(),
		facadetest.ModelContext{
			State_:          st,
			Resources_:      common.NewResources(),
			Auth_:           authorizer,
			ServiceFactory_: s.ControllerServiceFactory(c),
		})
	c.Assert(err, jc.ErrorIsNil)
	cfg, err := controller.ControllerConfig(context.Background())
	c.Assert(err, jc.ErrorIsNil)

	controllerConfigService := s.ControllerServiceFactory(c).ControllerConfig()

	cfgFromDB, err := controllerConfigService.ControllerConfig(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cfg.Config["controller-uuid"], gc.Equals, cfgFromDB.ControllerUUID())
	c.Assert(cfg.Config["state-port"], gc.Equals, cfgFromDB.StatePort())
	c.Assert(cfg.Config["api-port"], gc.Equals, cfgFromDB.APIPort())
}

func (s *controllerSuite) TestRemoveBlocks(c *gc.C) {
	st := s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "test"})
	defer func() { _ = st.Close() }()

	s.State.SwitchBlockOn(state.DestroyBlock, "TestBlockDestroyModel")
	s.State.SwitchBlockOn(state.ChangeBlock, "TestChangeBlock")
	st.SwitchBlockOn(state.DestroyBlock, "TestBlockDestroyModel")
	st.SwitchBlockOn(state.ChangeBlock, "TestChangeBlock")

	err := s.controller.RemoveBlocks(stdcontext.Background(), params.RemoveBlocksArgs{All: true})
	c.Assert(err, jc.ErrorIsNil)

	blocks, err := s.State.AllBlocksForController()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(blocks, gc.HasLen, 0)
}

func (s *controllerSuite) TestRemoveBlocksNotAll(c *gc.C) {
	err := s.controller.RemoveBlocks(stdcontext.Background(), params.RemoveBlocksArgs{})
	c.Assert(err, gc.ErrorMatches, "not supported")
}

func (s *controllerSuite) TestWatchAllModels(c *gc.C) {
	watcherId, err := s.controller.WatchAllModels(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)

	var disposed bool
	watcherAPI_, err := apiserver.NewAllWatcher(context.Background(), facadetest.ModelContext{
		State_:           s.State,
		Resources_:       s.resources,
		WatcherRegistry_: s.watcherRegistry,
		ServiceFactory_:  s.ControllerServiceFactory(c),
		Auth_:            s.authorizer,
		ID_:              watcherId.AllWatcherId,
		Dispose_:         func() { disposed = true },
	})
	c.Assert(err, jc.ErrorIsNil)
	watcherAPI := watcherAPI_.(*apiserver.SrvAllWatcher)
	defer func() {
		err := watcherAPI.Stop()
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(disposed, jc.IsTrue)
	}()

	done := make(chan bool)
	defer close(done)
	resultC := make(chan params.AllWatcherNextResults)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				result, err := watcherAPI.Next(context.Background())
				if err != nil {
					c.Assert(err, jc.Satisfies, coremultiwatcher.IsErrStopped)
					return
				}
				resultC <- result
			}
		}
	}()

	select {
	case result := <-resultC:
		// Expect to see the initial model be reported.
		deltas := result.Deltas
		c.Assert(deltas, gc.HasLen, 1)
		modelInfo := deltas[0].Entity.(*params.ModelUpdate)
		c.Assert(modelInfo.ModelUUID, gc.Equals, s.State.ModelUUID())
		c.Assert(modelInfo.IsController, gc.Equals, s.State.IsController())
	case <-time.After(testing.LongWait):
		c.Fatal("timed out")
	}

	// To ensure we really watch all models, make another one.
	st := s.Factory.MakeModel(c, &factory.ModelParams{
		Name: "test"})
	defer st.Close()

	// Update the model agent versions to ensure settings changes cause an update.
	err = s.State.SetModelAgentVersion(version.MustParse("2.6.666"), nil, true, stubUpgrader{})
	c.Assert(err, jc.ErrorIsNil)
	err = st.SetModelAgentVersion(version.MustParse("2.6.667"), nil, true, stubUpgrader{})
	c.Assert(err, jc.ErrorIsNil)
	expectedVersions := map[string]string{
		s.State.ModelUUID(): "2.6.666",
		st.ModelUUID():      "2.6.667",
	}

	for resultCount := 0; resultCount != 2; {
		select {
		case result := <-resultC:
			c.Logf("got change: %# v", pretty.Formatter(result))
			for _, d := range result.Deltas {
				if d.Removed {
					continue
				}
				modelInfo, ok := d.Entity.(*params.ModelUpdate)
				if !ok {
					continue
				}
				if modelInfo.Config["agent-version"] == expectedVersions[modelInfo.ModelUUID] {
					resultCount = resultCount + 1
				}
			}
		case <-time.After(testing.LongWait):
			c.Fatalf("timed out waiting for 2 model updates, got %d", resultCount)
		}
	}
}

func (s *controllerSuite) TestInitiateMigration(c *gc.C) {
	// Create two hosted models to migrate.
	st1 := s.Factory.MakeModel(c, nil)
	defer func() { _ = st1.Close() }()
	model1, err := st1.Model()
	c.Assert(err, jc.ErrorIsNil)

	st2 := s.Factory.MakeModel(c, nil)
	defer func() { _ = st2.Close() }()
	model2, err := st2.Model()
	c.Assert(err, jc.ErrorIsNil)

	mac, err := macaroon.New([]byte("secret"), []byte("id"), "location", macaroon.LatestVersion)
	c.Assert(err, jc.ErrorIsNil)
	macsJSON, err := json.Marshal([]macaroon.Slice{{mac}})
	c.Assert(err, jc.ErrorIsNil)

	controller.SetPrecheckResult(s, nil)

	// Kick off migrations
	args := params.InitiateMigrationArgs{
		Specs: []params.MigrationSpec{
			{
				ModelTag: model1.ModelTag().String(),
				TargetInfo: params.MigrationTargetInfo{
					ControllerTag:   randomControllerTag(),
					ControllerAlias: "", // intentionally left empty; simulates older client
					Addrs:           []string{"1.1.1.1:1111", "2.2.2.2:2222"},
					CACert:          "cert1",
					AuthTag:         names.NewUserTag("admin1").String(),
					Password:        "secret1",
				},
			}, {
				ModelTag: model2.ModelTag().String(),
				TargetInfo: params.MigrationTargetInfo{
					ControllerTag:   randomControllerTag(),
					ControllerAlias: "target-controller",
					Addrs:           []string{"3.3.3.3:3333"},
					CACert:          "cert2",
					AuthTag:         names.NewUserTag("admin2").String(),
					Macaroons:       string(macsJSON),
					Password:        "secret2",
				},
			},
		},
	}
	out, err := s.controller.InitiateMigration(stdcontext.Background(), args)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(out.Results, gc.HasLen, 2)

	states := []*state.State{st1, st2}
	for i, spec := range args.Specs {
		c.Log(i)
		st := states[i]
		result := out.Results[i]

		c.Assert(result.Error, gc.IsNil)
		c.Check(result.ModelTag, gc.Equals, spec.ModelTag)
		expectedId := st.ModelUUID() + ":0"
		c.Check(result.MigrationId, gc.Equals, expectedId)

		// Ensure the migration made it into the DB correctly.
		mig, err := st.LatestMigration()
		c.Assert(err, jc.ErrorIsNil)
		c.Check(mig.Id(), gc.Equals, expectedId)
		c.Check(mig.ModelUUID(), gc.Equals, st.ModelUUID())
		c.Check(mig.InitiatedBy(), gc.Equals, s.Owner.Id())

		targetInfo, err := mig.TargetInfo()
		c.Assert(err, jc.ErrorIsNil)
		c.Check(targetInfo.ControllerTag.String(), gc.Equals, spec.TargetInfo.ControllerTag)
		c.Check(targetInfo.ControllerAlias, gc.Equals, spec.TargetInfo.ControllerAlias)
		c.Check(targetInfo.Addrs, jc.SameContents, spec.TargetInfo.Addrs)
		c.Check(targetInfo.CACert, gc.Equals, spec.TargetInfo.CACert)
		c.Check(targetInfo.AuthTag.String(), gc.Equals, spec.TargetInfo.AuthTag)
		c.Check(targetInfo.Password, gc.Equals, spec.TargetInfo.Password)

		if spec.TargetInfo.Macaroons != "" {
			macJSONdb, err := json.Marshal(targetInfo.Macaroons)
			c.Assert(err, jc.ErrorIsNil)
			c.Check(string(macJSONdb), gc.Equals, spec.TargetInfo.Macaroons)
		}
	}
}

func (s *controllerSuite) TestInitiateMigrationSpecError(c *gc.C) {
	// Create a hosted model to migrate.
	st := s.Factory.MakeModel(c, nil)
	defer func() { _ = st.Close() }()
	model, err := st.Model()
	c.Assert(err, jc.ErrorIsNil)

	// Kick off the migration with missing details.
	args := params.InitiateMigrationArgs{
		Specs: []params.MigrationSpec{{
			ModelTag: model.ModelTag().String(),
			// TargetInfo missing
		}},
	}
	out, err := s.controller.InitiateMigration(stdcontext.Background(), args)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(out.Results, gc.HasLen, 1)
	result := out.Results[0]
	c.Check(result.ModelTag, gc.Equals, args.Specs[0].ModelTag)
	c.Check(result.MigrationId, gc.Equals, "")
	c.Check(result.Error, gc.ErrorMatches, "controller tag: .+ is not a valid tag")
}

func (s *controllerSuite) TestInitiateMigrationPartialFailure(c *gc.C) {
	st := s.Factory.MakeModel(c, nil)
	defer func() { _ = st.Close() }()
	controller.SetPrecheckResult(s, nil)

	m, err := st.Model()
	c.Assert(err, jc.ErrorIsNil)

	args := params.InitiateMigrationArgs{
		Specs: []params.MigrationSpec{
			{
				ModelTag: m.ModelTag().String(),
				TargetInfo: params.MigrationTargetInfo{
					ControllerTag: randomControllerTag(),
					Addrs:         []string{"1.1.1.1:1111", "2.2.2.2:2222"},
					CACert:        "cert",
					AuthTag:       names.NewUserTag("admin").String(),
					Password:      "secret",
				},
			}, {
				ModelTag: randomModelTag(), // Doesn't exist.
			},
		},
	}
	out, err := s.controller.InitiateMigration(stdcontext.Background(), args)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(out.Results, gc.HasLen, 2)

	c.Check(out.Results[0].ModelTag, gc.Equals, m.ModelTag().String())
	c.Check(out.Results[0].Error, gc.IsNil)

	c.Check(out.Results[1].ModelTag, gc.Equals, args.Specs[1].ModelTag)
	c.Check(out.Results[1].Error, gc.ErrorMatches, "model not found")
}

func (s *controllerSuite) TestInitiateMigrationInvalidMacaroons(c *gc.C) {
	st := s.Factory.MakeModel(c, nil)
	defer st.Close()

	m, err := st.Model()
	c.Assert(err, jc.ErrorIsNil)

	args := params.InitiateMigrationArgs{
		Specs: []params.MigrationSpec{
			{
				ModelTag: m.ModelTag().String(),
				TargetInfo: params.MigrationTargetInfo{
					ControllerTag: randomControllerTag(),
					Addrs:         []string{"1.1.1.1:1111", "2.2.2.2:2222"},
					CACert:        "cert",
					AuthTag:       names.NewUserTag("admin").String(),
					Macaroons:     "BLAH",
				},
			},
		},
	}
	out, err := s.controller.InitiateMigration(stdcontext.Background(), args)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(out.Results, gc.HasLen, 1)
	result := out.Results[0]
	c.Check(result.ModelTag, gc.Equals, args.Specs[0].ModelTag)
	c.Check(result.Error, gc.ErrorMatches, "invalid macaroons: .+")
}

func (s *controllerSuite) TestInitiateMigrationPrecheckFail(c *gc.C) {
	st := s.Factory.MakeModel(c, nil)
	defer st.Close()

	controller.SetPrecheckResult(s, errors.New("boom"))

	m, err := st.Model()
	c.Assert(err, jc.ErrorIsNil)

	args := params.InitiateMigrationArgs{
		Specs: []params.MigrationSpec{{
			ModelTag: m.ModelTag().String(),
			TargetInfo: params.MigrationTargetInfo{
				ControllerTag: randomControllerTag(),
				Addrs:         []string{"1.1.1.1:1111"},
				CACert:        "cert1",
				AuthTag:       names.NewUserTag("admin1").String(),
				Password:      "secret1",
			},
		}},
	}
	out, err := s.controller.InitiateMigration(stdcontext.Background(), args)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(out.Results, gc.HasLen, 1)
	c.Check(out.Results[0].Error, gc.ErrorMatches, "boom")

	active, err := st.IsMigrationActive()
	c.Assert(err, jc.ErrorIsNil)
	c.Check(active, jc.IsFalse)
}

func randomControllerTag() string {
	uuid := uuid.MustNewUUID().String()
	return names.NewControllerTag(uuid).String()
}

func randomModelTag() string {
	uuid := uuid.MustNewUUID().String()
	return names.NewModelTag(uuid).String()
}

func (s *controllerSuite) TestGrantControllerInvalidUserTag(c *gc.C) {
	for _, testParam := range []struct {
		tag      string
		validTag bool
	}{{
		tag:      "unit-foo/0",
		validTag: true,
	}, {
		tag:      "application-foo",
		validTag: true,
	}, {
		tag:      "relation-wordpress:db mysql:db",
		validTag: true,
	}, {
		tag:      "machine-0",
		validTag: true,
	}, {
		tag:      "user@local",
		validTag: false,
	}, {
		tag:      "user-Mua^h^h^h^arh",
		validTag: true,
	}, {
		tag:      "user@",
		validTag: false,
	}, {
		tag:      "user@ubuntuone",
		validTag: false,
	}, {
		tag:      "user@ubuntuone",
		validTag: false,
	}, {
		tag:      "@ubuntuone",
		validTag: false,
	}, {
		tag:      "in^valid.",
		validTag: false,
	}, {
		tag:      "",
		validTag: false,
	},
	} {
		var expectedErr string
		errPart := `could not modify controller access: "` + regexp.QuoteMeta(testParam.tag) + `" is not a valid `

		if testParam.validTag {
			// The string is a valid tag, but not a user tag.
			expectedErr = errPart + `user tag`
		} else {
			// The string is not a valid tag of any kind.
			expectedErr = errPart + `tag`
		}

		args := params.ModifyControllerAccessRequest{
			Changes: []params.ModifyControllerAccess{{
				UserTag: testParam.tag,
				Action:  params.GrantControllerAccess,
				Access:  string(permission.SuperuserAccess),
			}}}

		result, err := s.controller.ModifyControllerAccess(stdcontext.Background(), args)
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(result.OneError(), gc.ErrorMatches, expectedErr)
	}
}

type accessSuite struct {
	statetesting.StateSuite

	resources  *common.Resources
	authorizer apiservertesting.FakeAuthorizer

	accessService *mocks.MockControllerAccessService
}

var _ = gc.Suite(&accessSuite{})

func (s *accessSuite) SetUpSuite(c *gc.C) {
	s.StateSuite.SetUpSuite(c)
}
func (s *accessSuite) SetUpTest(c *gc.C) {
	// Initial config needs to be set before the StateSuite SetUpTest.
	s.InitialConfig = testing.CustomModelConfig(c, testing.Attrs{
		"name": "controller",
	})
	controllerCfg := testing.FakeControllerConfig()

	s.StateSuite.ControllerConfig = controllerCfg
	s.StateSuite.SetUpTest(c)

	s.resources = common.NewResources()
	s.AddCleanup(func(_ *gc.C) { s.resources.StopAll() })

	s.authorizer = apiservertesting.FakeAuthorizer{
		Tag:      s.Owner,
		AdminTag: s.Owner,
	}
}

func (s *controllerSuite) TestModelStatus(c *gc.C) {
	// Check that we don't err out immediately if a model errs.
	results, err := s.controller.ModelStatus(context.Background(), params.Entities{Entities: []params.Entity{{
		Tag: "bad-tag",
	}, {
		Tag: s.Model.ModelTag().String(),
	}}})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(results.Results, gc.HasLen, 2)
	c.Assert(results.Results[0].Error, gc.ErrorMatches, `"bad-tag" is not a valid tag`)

	// Check that we don't err out if a model errs even if some firsts in collection pass.
	results, err = s.controller.ModelStatus(context.Background(), params.Entities{Entities: []params.Entity{{
		Tag: s.Model.ModelTag().String(),
	}, {
		Tag: "bad-tag",
	}}})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(results.Results, gc.HasLen, 2)
	c.Assert(results.Results[1].Error, gc.ErrorMatches, `"bad-tag" is not a valid tag`)

	// Check that we return successfully if no errors.
	results, err = s.controller.ModelStatus(context.Background(), params.Entities{Entities: []params.Entity{{
		Tag: s.Model.ModelTag().String(),
	}}})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(results.Results, gc.HasLen, 1)
}

func (s *controllerSuite) TestConfigSet(c *gc.C) {
	controllerConfigService := s.ControllerServiceFactory(c).ControllerConfig()

	config, err := controllerConfigService.ControllerConfig(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	// Sanity check.
	c.Assert(config.AuditingEnabled(), gc.Equals, false)

	err = s.controller.ConfigSet(stdcontext.Background(), params.ControllerConfigSet{Config: map[string]interface{}{
		"auditing-enabled": true,
	}})
	c.Assert(err, jc.ErrorIsNil)

	config, err = controllerConfigService.ControllerConfig(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(config.AuditingEnabled(), gc.Equals, true)
}

func (s *controllerSuite) TestConfigSetRequiresSuperUser(c *gc.C) {
	user := s.Factory.MakeUser(c, &factory.UserParams{
		Access: permission.ReadAccess,
	})
	anAuthoriser := apiservertesting.FakeAuthorizer{
		Tag: user.Tag(),
	}
	endpoint, err := controller.NewControllerAPIv11(
		context.Background(),
		facadetest.ModelContext{
			State_:          s.State,
			Resources_:      s.resources,
			Auth_:           anAuthoriser,
			ServiceFactory_: s.ControllerServiceFactory(c),
		})
	c.Assert(err, jc.ErrorIsNil)

	err = endpoint.ConfigSet(stdcontext.Background(), params.ControllerConfigSet{Config: map[string]interface{}{
		"something": 23,
	}})

	c.Assert(err, gc.ErrorMatches, "permission denied")
}

func (s *controllerSuite) TestConfigSetPublishesEvent(c *gc.C) {
	done := make(chan struct{})
	var config corecontroller.Config
	s.hub.Subscribe(pscontroller.ConfigChanged, func(topic string, data pscontroller.ConfigChangedMessage, err error) {
		c.Check(err, jc.ErrorIsNil)
		config = data.Config
		close(done)
	})

	err := s.controller.ConfigSet(stdcontext.Background(), params.ControllerConfigSet{Config: map[string]interface{}{
		"features": "foo,bar",
	}})
	c.Assert(err, jc.ErrorIsNil)

	select {
	case <-done:
	case <-time.After(testing.LongWait):
		c.Fatal("no event sent}")
	}

	c.Assert(config.Features().SortedValues(), jc.DeepEquals, []string{"bar", "foo"})
}

func (s *controllerSuite) TestConfigSetCAASImageRepo(c *gc.C) {
	// TODO(dqlite): move this test when ConfigSet CAASImageRepo logic moves.
	controllerConfigService := s.ControllerServiceFactory(c).ControllerConfig()

	config, err := controllerConfigService.ControllerConfig(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(config.CAASImageRepo(), gc.Equals, "")

	err = s.controller.ConfigSet(stdcontext.Background(), params.ControllerConfigSet{Config: map[string]interface{}{
		"caas-image-repo": "juju-repo.local",
	}})
	c.Assert(err, gc.ErrorMatches, `cannot change caas-image-repo as it is not currently set`)

	err = controllerConfigService.UpdateControllerConfig(
		context.Background(),
		map[string]interface{}{
			"caas-image-repo": "jujusolutions",
		}, nil)
	c.Assert(err, jc.ErrorIsNil)

	err = s.controller.ConfigSet(stdcontext.Background(), params.ControllerConfigSet{Config: map[string]interface{}{
		"caas-image-repo": "juju-repo.local",
	}})
	c.Assert(err, gc.ErrorMatches, `cannot change caas-image-repo: repository read-only, only authentication can be updated`)

	err = s.controller.ConfigSet(stdcontext.Background(), params.ControllerConfigSet{Config: map[string]interface{}{
		"caas-image-repo": `{"repository":"jujusolutions","username":"foo","password":"bar"}`,
	}})
	c.Assert(err, gc.ErrorMatches, `cannot change caas-image-repo: unable to add authentication details`)

	err = controllerConfigService.UpdateControllerConfig(
		context.Background(),
		map[string]interface{}{
			"caas-image-repo": `{"repository":"jujusolutions","username":"bar","password":"foo"}`,
		}, nil)
	c.Assert(err, jc.ErrorIsNil)

	err = s.controller.ConfigSet(stdcontext.Background(), params.ControllerConfigSet{Config: map[string]interface{}{
		"caas-image-repo": `{"repository":"jujusolutions","username":"foo","password":"bar"}`,
	}})
	c.Assert(err, jc.ErrorIsNil)

	config, err = controllerConfigService.ControllerConfig(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	repoDetails, err := docker.NewImageRepoDetails(config.CAASImageRepo())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(repoDetails, gc.DeepEquals, docker.ImageRepoDetails{
		Repository: "jujusolutions",
		BasicAuthConfig: docker.BasicAuthConfig{
			Username: "foo",
			Password: "bar",
		},
	})
}

func (s *controllerSuite) TestMongoVersion(c *gc.C) {
	result, err := s.controller.MongoVersion(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)

	var resErr *params.Error
	c.Assert(result.Error, gc.Equals, resErr)
	// We can't guarantee which version of mongo is running, so let's just
	// attempt to match it to a very basic version (major.minor.patch)
	c.Assert(result.Result, gc.Matches, "^([0-9]{1,}).([0-9]{1,}).([0-9]{1,})$")
}

func (s *controllerSuite) TestIdentityProviderURL(c *gc.C) {
	// Preserve default controller config as we will be mutating it just
	// for this test
	defer func(orig map[string]interface{}) {
		s.controllerConfigAttrs = orig
	}(s.controllerConfigAttrs)

	// Our default test configuration does not specify an IdentityURL
	urlRes, err := s.controller.IdentityProviderURL(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(urlRes.Result, gc.Equals, "")

	// IdentityURL cannot be changed after bootstrap; we need to spin up
	// another controller with IdentityURL pre-configured
	s.TearDownTest(c)

	expURL := "https://api.jujucharms.com/identity"
	s.controllerConfigAttrs = map[string]any{
		corecontroller.IdentityURL: expURL,
	}

	s.SetUpTest(c)

	urlRes, err = s.controller.IdentityProviderURL(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(urlRes.Result, gc.Equals, expURL)
}

func (s *controllerSuite) newSummaryWatcherFacade(c *gc.C, id string) *apiserver.SrvModelSummaryWatcher {
	context := s.context
	context.ID_ = id
	watcher, err := apiserver.NewModelSummaryWatcher(context)
	c.Assert(err, jc.ErrorIsNil)
	return watcher
}

func (s *controllerSuite) TestWatchAllModelSummariesByAdmin(c *gc.C) {
	// TODO(dqlite) - implement me
	c.Skip("watch model summaries to be implemented")
	// Default authorizer is an admin.
	result, err := s.controller.WatchAllModelSummaries(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)

	watcherAPI := s.newSummaryWatcherFacade(c, result.WatcherID)

	resultC := make(chan params.SummaryWatcherNextResults)
	go func() {
		result, err := watcherAPI.Next(context.Background())
		c.Assert(err, jc.ErrorIsNil)
		resultC <- result
	}()

	select {
	case result := <-resultC:
		// Expect to see the initial environment be reported.
		c.Assert(result, jc.DeepEquals, params.SummaryWatcherNextResults{
			Models: []params.ModelAbstract{
				{
					UUID:       "deadbeef-0bad-400d-8000-4b1d0d06f00d",
					Controller: "", // TODO(thumper): add controller name next branch
					Name:       "controller",
					Admins:     []string{"test-admin"},
					Cloud:      "dummy",
					Region:     "dummy-region",
					Status:     "green",
					Messages:   []params.ModelSummaryMessage{},
				}}})
	case <-time.After(testing.LongWait):
		c.Fatal("timed out")
	}
}

func (s *controllerSuite) TestWatchAllModelSummariesByNonAdmin(c *gc.C) {
	anAuthoriser := apiservertesting.FakeAuthorizer{
		Tag: names.NewLocalUserTag("bob"),
	}
	endPoint, err := controller.LatestAPI(
		context.Background(),
		facadetest.ModelContext{
			State_:          s.State,
			Resources_:      s.resources,
			Auth_:           anAuthoriser,
			ServiceFactory_: s.ControllerServiceFactory(c),
		})
	c.Assert(err, jc.ErrorIsNil)

	_, err = endPoint.WatchAllModelSummaries(stdcontext.Background())
	c.Assert(err, gc.ErrorMatches, "permission denied")
}

func (s *controllerSuite) makeBobsModel(c *gc.C) string {
	bob := s.Factory.MakeUser(c, &factory.UserParams{
		Name:        "bob",
		NoModelUser: true,
	})
	st := s.Factory.MakeModel(c, &factory.ModelParams{
		Owner: bob.UserTag(),
		Name:  "bobs-model"})
	uuid := st.ModelUUID()
	st.Close()
	s.WaitForModelWatchersIdle(c, uuid)
	return uuid
}

func (s *controllerSuite) TestWatchModelSummariesByNonAdmin(c *gc.C) {
	// TODO(dqlite) - implement me
	c.Skip("watch model summaries to be implemented")
	s.makeBobsModel(c)

	// Default authorizer is an admin. As a user, admin can't see
	// Bob's model.
	result, err := s.controller.WatchModelSummaries(stdcontext.Background())
	c.Assert(err, jc.ErrorIsNil)

	watcherAPI := s.newSummaryWatcherFacade(c, result.WatcherID)

	resultC := make(chan params.SummaryWatcherNextResults)
	go func() {
		result, err := watcherAPI.Next(context.Background())
		c.Assert(err, jc.ErrorIsNil)
		resultC <- result
	}()

	select {
	case result := <-resultC:
		// Expect to see the initial environment be reported.
		c.Assert(result, jc.DeepEquals, params.SummaryWatcherNextResults{
			Models: []params.ModelAbstract{
				{
					UUID:       "deadbeef-0bad-400d-8000-4b1d0d06f00d",
					Controller: "", // TODO(thumper): add controller name next branch
					Name:       "controller",
					Admins:     []string{"test-admin"},
					Cloud:      "dummy",
					Region:     "dummy-region",
					Status:     "green",
					Messages:   []params.ModelSummaryMessage{},
				}}})
	case <-time.After(testing.LongWait):
		c.Fatal("timed out")
	}

}

func (s *accessSuite) controllerAPI(c *gc.C) *controller.ControllerAPI {
	api, err := controller.NewControllerAPI(
		context.Background(),
		s.State,
		s.StatePool,
		s.authorizer,
		s.resources,
		nil,
		nil,
		nil,
		loggertesting.WrapCheckLog(c),
		nil,
		nil,
		nil,
		nil,
		nil,
		s.accessService,
	)
	c.Assert(err, jc.ErrorIsNil)

	return api
}

func (s *accessSuite) TestModifyControllerAccess(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	userName := "test-user"

	s.accessService = mocks.NewMockControllerAccessService(ctrl)
	updateArgs := access.UpdatePermissionArgs{
		AccessSpec: permission.ControllerForAccess(permission.SuperuserAccess),
		AddUser:    true,
		ApiUser:    "test-admin",
		Change:     permission.Grant,
		Subject:    userName,
	}
	s.accessService.EXPECT().UpdatePermission(gomock.Any(), updateArgs).Return(nil)

	args := params.ModifyControllerAccessRequest{Changes: []params.ModifyControllerAccess{{
		UserTag: names.NewUserTag(userName).String(),
		Action:  params.GrantControllerAccess,
		Access:  string(permission.SuperuserAccess),
	}}}

	result, err := s.controllerAPI(c).ModifyControllerAccess(stdcontext.Background(), args)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Results, gc.HasLen, 1)
}

func (s *accessSuite) TestGetControllerAccessPermissions(c *gc.C) {
	// Set up the user making the call.
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	userName := "test-user"
	userTag := names.NewUserTag(userName)
	differentUser := "different-test-user"

	s.accessService = mocks.NewMockControllerAccessService(ctrl)
	target := permission.ControllerForAccess(permission.SuperuserAccess)
	s.accessService.EXPECT().ReadUserAccessLevelForTarget(gomock.Any(), userName, target.Target).Return(permission.SuperuserAccess, nil)

	s.authorizer = apiservertesting.FakeAuthorizer{
		Tag: userTag,
	}

	req := params.Entities{
		Entities: []params.Entity{{Tag: userTag.String()}, {Tag: names.NewUserTag(differentUser).String()}},
	}
	results, err := s.controllerAPI(c).GetControllerAccess(stdcontext.Background(), req)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(results.Results, gc.HasLen, 2)
	c.Assert(*results.Results[0].Result, jc.DeepEquals, params.UserAccess{
		Access:  "superuser",
		UserTag: userTag.String(),
	})
	c.Assert(*results.Results[1].Error, gc.DeepEquals, params.Error{
		Message: "permission denied", Code: "unauthorized access",
	})
}

type noopRegisterer struct {
	prometheus.Registerer
}

func (noopRegisterer) Register(prometheus.Collector) error {
	return nil
}

func (noopRegisterer) Unregister(prometheus.Collector) bool {
	return true
}

type stubUpgrader struct{}

func (stubUpgrader) IsUpgrading() (bool, error) {
	return false, nil
}
