// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package machine_test

import (
	stdtesting "testing"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	apiservertesting "github.com/juju/juju/apiserver/testing"
	"github.com/juju/juju/core/status"
	"github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state"
	coretesting "github.com/juju/juju/testing"
)

func Test(t *stdtesting.T) {
	coretesting.MgoTestPackage(t)
}

type commonSuite struct {
	testing.ApiServerSuite

	authorizer apiservertesting.FakeAuthorizer

	machine0 *state.Machine
	machine1 *state.Machine
}

func (s *commonSuite) SetUpTest(c *gc.C) {
	s.ApiServerSuite.SetUpTest(c)

	st := s.ControllerModel(c).State()
	var err error
	s.machine0, err = st.AddMachine(state.NoopInstancePrechecker{}, state.UbuntuBase("12.10"), status.NoopStatusHistoryRecorder, state.JobManageModel)
	c.Assert(err, jc.ErrorIsNil)

	s.machine1, err = st.AddMachine(state.NoopInstancePrechecker{}, state.UbuntuBase("12.10"), status.NoopStatusHistoryRecorder, state.JobHostUnits)
	c.Assert(err, jc.ErrorIsNil)

	// Create a FakeAuthorizer so we can check permissions,
	// set up assuming machine 1 has logged in.
	s.authorizer = apiservertesting.FakeAuthorizer{
		Tag: s.machine1.Tag(),
	}
}
