// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package firewaller

import (
	"context"

	"github.com/juju/errors"
	"github.com/juju/worker/v4"
	"github.com/juju/worker/v4/dependency"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/api"
	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/controller/remoterelations"
	coredependency "github.com/juju/juju/core/dependency"
	"github.com/juju/juju/core/logger"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/models"
	"github.com/juju/juju/internal/servicefactory"
	"github.com/juju/juju/internal/worker/apicaller"
	"github.com/juju/juju/internal/worker/common"
)

// GetMachineFunc is a helper function that gets a service from the manifold.
type GetMachineServiceFunc func(getter dependency.Getter, name string) (MachineService, error)

// ManifoldConfig describes the resources used by the firewaller worker.
type ManifoldConfig struct {
	AgentName          string
	APICallerName      string
	EnvironName        string
	Logger             logger.Logger
	ServiceFactoryName string

	NewControllerConnection      apicaller.NewExternalControllerConnectionFunc
	NewRemoteRelationsFacade     func(base.APICaller) *remoterelations.Client
	NewFirewallerFacade          func(base.APICaller) (FirewallerAPI, error)
	NewFirewallerWorker          func(Config) (worker.Worker, error)
	NewCredentialValidatorFacade func(base.APICaller) (common.CredentialAPI, error)

	GetMachineService GetMachineServiceFunc
}

// Manifold returns a Manifold that encapsulates the firewaller worker.
func Manifold(cfg ManifoldConfig) dependency.Manifold {
	return dependency.Manifold{
		Inputs: []string{
			cfg.AgentName,
			cfg.APICallerName,
			cfg.EnvironName,
			cfg.ServiceFactoryName,
		},
		Start: cfg.start,
	}
}

// Validate is called by start to check for bad configuration.
func (cfg ManifoldConfig) Validate() error {
	if cfg.AgentName == "" {
		return errors.NotValidf("empty AgentName")
	}
	if cfg.APICallerName == "" {
		return errors.NotValidf("empty APICallerName")
	}
	if cfg.EnvironName == "" {
		return errors.NotValidf("empty EnvironName")
	}
	if cfg.Logger == nil {
		return errors.NotValidf("nil Logger")
	}
	if cfg.ServiceFactoryName == "" {
		return errors.NotValidf("empty ServiceFactoryName")
	}
	if cfg.GetMachineService == nil {
		return errors.NotValidf("nil GetMachineService")
	}
	if cfg.NewControllerConnection == nil {
		return errors.NotValidf("nil NewControllerConnection")
	}
	if cfg.NewRemoteRelationsFacade == nil {
		return errors.NotValidf("nil NewRemoteRelationsFacade")
	}
	if cfg.NewFirewallerFacade == nil {
		return errors.NotValidf("nil NewFirewallerFacade")
	}
	if cfg.NewFirewallerWorker == nil {
		return errors.NotValidf("nil NewFirewallerWorker")
	}
	if cfg.NewCredentialValidatorFacade == nil {
		return errors.NotValidf("nil NewCredentialValidatorFacade")
	}
	return nil
}

// start is a StartFunc for a Worker manifold.
func (cfg ManifoldConfig) start(ctx context.Context, getter dependency.Getter) (worker.Worker, error) {
	if err := cfg.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	var agent agent.Agent
	if err := getter.Get(cfg.AgentName, &agent); err != nil {
		return nil, errors.Trace(err)
	}
	var apiConn api.Connection
	if err := getter.Get(cfg.APICallerName, &apiConn); err != nil {
		return nil, errors.Trace(err)
	}

	var environ environs.Environ
	if err := getter.Get(cfg.EnvironName, &environ); err != nil {
		return nil, errors.Trace(err)
	}

	// Check if the env supports global firewalling.  If the
	// configured mode is instance, we can ignore fwEnv being a
	// nil value, as it won't be used.
	fwEnv, fwEnvOK := environ.(environs.Firewaller)

	modelFw, _ := environ.(models.ModelFirewaller)

	mode := environ.Config().FirewallMode()
	if mode == config.FwNone {
		cfg.Logger.Infof("stopping firewaller (not required)")
		return nil, dependency.ErrUninstall
	} else if mode == config.FwGlobal {
		if !fwEnvOK {
			cfg.Logger.Infof("Firewall global mode set on provider with no support. stopping firewaller")
			return nil, dependency.ErrUninstall
		}
	}

	firewallerAPI, err := cfg.NewFirewallerFacade(apiConn)
	if err != nil {
		return nil, errors.Trace(err)
	}

	credentialAPI, err := cfg.NewCredentialValidatorFacade(apiConn)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Check if the env supports IPV6 CIDRs for firewall ingress rules.
	var envIPV6CIDRSupport bool
	if featQuerier, ok := environ.(environs.FirewallFeatureQuerier); ok {
		var err error
		cloudCtx := common.NewCloudCallContextFunc(credentialAPI)(ctx)
		if envIPV6CIDRSupport, err = featQuerier.SupportsRulesWithIPV6CIDRs(cloudCtx); err != nil {
			return nil, errors.Trace(err)
		}
	}

	machineService, err := cfg.GetMachineService(getter, cfg.ServiceFactoryName)
	if err != nil {
		return nil, errors.Trace(err)
	}

	w, err := cfg.NewFirewallerWorker(Config{
		ModelUUID:               agent.CurrentConfig().Model().Id(),
		RemoteRelationsApi:      cfg.NewRemoteRelationsFacade(apiConn),
		FirewallerAPI:           firewallerAPI,
		EnvironFirewaller:       fwEnv,
		EnvironModelFirewaller:  modelFw,
		EnvironInstances:        environ,
		EnvironIPV6CIDRSupport:  envIPV6CIDRSupport,
		Mode:                    mode,
		NewCrossModelFacadeFunc: crossmodelFirewallerFacadeFunc(cfg.NewControllerConnection),
		CredentialAPI:           credentialAPI,
		Logger:                  cfg.Logger,
		MachineService:          machineService,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return w, nil
}

// GetMachineService is a helper function that gets a service from the
// manifold.
func GetMachineService(getter dependency.Getter, name string) (MachineService, error) {
	return coredependency.GetDependencyByName(getter, name, func(factory servicefactory.ModelServiceFactory) MachineService {
		return factory.Machine()
	})
}
