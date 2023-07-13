// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/facades/controller/externalcontrollerupdater (interfaces: EcService)

// Package externalcontrollerupdater_test is a generated GoMock package.
package externalcontrollerupdater_test

import (
	context "context"
	reflect "reflect"

	crossmodel "github.com/juju/juju/core/crossmodel"
	watcher "github.com/juju/juju/core/watcher"
	gomock "go.uber.org/mock/gomock"
)

// MockEcService is a mock of EcService interface.
type MockEcService struct {
	ctrl     *gomock.Controller
	recorder *MockEcServiceMockRecorder
}

// MockEcServiceMockRecorder is the mock recorder for MockEcService.
type MockEcServiceMockRecorder struct {
	mock *MockEcService
}

// NewMockEcService creates a new mock instance.
func NewMockEcService(ctrl *gomock.Controller) *MockEcService {
	mock := &MockEcService{ctrl: ctrl}
	mock.recorder = &MockEcServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEcService) EXPECT() *MockEcServiceMockRecorder {
	return m.recorder
}

// Controller mocks base method.
func (m *MockEcService) Controller(arg0 context.Context, arg1 string) (*crossmodel.ControllerInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Controller", arg0, arg1)
	ret0, _ := ret[0].(*crossmodel.ControllerInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Controller indicates an expected call of Controller.
func (mr *MockEcServiceMockRecorder) Controller(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Controller", reflect.TypeOf((*MockEcService)(nil).Controller), arg0, arg1)
}

// UpdateExternalController mocks base method.
func (m *MockEcService) UpdateExternalController(arg0 context.Context, arg1 crossmodel.ControllerInfo) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateExternalController", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateExternalController indicates an expected call of UpdateExternalController.
func (mr *MockEcServiceMockRecorder) UpdateExternalController(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateExternalController", reflect.TypeOf((*MockEcService)(nil).UpdateExternalController), arg0, arg1)
}

// Watch mocks base method.
func (m *MockEcService) Watch() (watcher.Watcher[[]string], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch")
	ret0, _ := ret[0].(watcher.Watcher[[]string])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch.
func (mr *MockEcServiceMockRecorder) Watch() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockEcService)(nil).Watch))
}
