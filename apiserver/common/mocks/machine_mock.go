// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/common (interfaces: MachineService)
//
// Generated by this command:
//
//	mockgen -typed -package mocks -destination mocks/machine_mock.go github.com/juju/juju/apiserver/common MachineService
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	watcher "github.com/juju/juju/core/watcher"
	gomock "go.uber.org/mock/gomock"
)

// MockMachineService is a mock of MachineService interface.
type MockMachineService struct {
	ctrl     *gomock.Controller
	recorder *MockMachineServiceMockRecorder
}

// MockMachineServiceMockRecorder is the mock recorder for MockMachineService.
type MockMachineServiceMockRecorder struct {
	mock *MockMachineService
}

// NewMockMachineService creates a new mock instance.
func NewMockMachineService(ctrl *gomock.Controller) *MockMachineService {
	mock := &MockMachineService{ctrl: ctrl}
	mock.recorder = &MockMachineServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMachineService) EXPECT() *MockMachineServiceMockRecorder {
	return m.recorder
}

// WatchMachines mocks base method.
func (m *MockMachineService) WatchMachines(arg0 context.Context) (watcher.Watcher[[]string], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchMachines", arg0)
	ret0, _ := ret[0].(watcher.Watcher[[]string])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchMachines indicates an expected call of WatchMachines.
func (mr *MockMachineServiceMockRecorder) WatchMachines(arg0 any) *MockMachineServiceWatchMachinesCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchMachines", reflect.TypeOf((*MockMachineService)(nil).WatchMachines), arg0)
	return &MockMachineServiceWatchMachinesCall{Call: call}
}

// MockMachineServiceWatchMachinesCall wrap *gomock.Call
type MockMachineServiceWatchMachinesCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockMachineServiceWatchMachinesCall) Return(arg0 watcher.Watcher[[]string], arg1 error) *MockMachineServiceWatchMachinesCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockMachineServiceWatchMachinesCall) Do(f func(context.Context) (watcher.Watcher[[]string], error)) *MockMachineServiceWatchMachinesCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockMachineServiceWatchMachinesCall) DoAndReturn(f func(context.Context) (watcher.Watcher[[]string], error)) *MockMachineServiceWatchMachinesCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
