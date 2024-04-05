// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/internal/upgrades/upgradevalidation (interfaces: StatePool,State,Model)
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/state_mock.go github.com/juju/juju/internal/upgrades/upgradevalidation StatePool,State,Model
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	state "github.com/juju/juju/state"
	names "github.com/juju/names/v5"
	replicaset "github.com/juju/replicaset/v3"
	version "github.com/juju/version/v2"
	gomock "go.uber.org/mock/gomock"
)

// MockStatePool is a mock of StatePool interface.
type MockStatePool struct {
	ctrl     *gomock.Controller
	recorder *MockStatePoolMockRecorder
}

// MockStatePoolMockRecorder is the mock recorder for MockStatePool.
type MockStatePoolMockRecorder struct {
	mock *MockStatePool
}

// NewMockStatePool creates a new mock instance.
func NewMockStatePool(ctrl *gomock.Controller) *MockStatePool {
	mock := &MockStatePool{ctrl: ctrl}
	mock.recorder = &MockStatePoolMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStatePool) EXPECT() *MockStatePoolMockRecorder {
	return m.recorder
}

// MongoVersion mocks base method.
func (m *MockStatePool) MongoVersion() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MongoVersion")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MongoVersion indicates an expected call of MongoVersion.
func (mr *MockStatePoolMockRecorder) MongoVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MongoVersion", reflect.TypeOf((*MockStatePool)(nil).MongoVersion))
}

// MockState is a mock of State interface.
type MockState struct {
	ctrl     *gomock.Controller
	recorder *MockStateMockRecorder
}

// MockStateMockRecorder is the mock recorder for MockState.
type MockStateMockRecorder struct {
	mock *MockState
}

// NewMockState creates a new mock instance.
func NewMockState(ctrl *gomock.Controller) *MockState {
	mock := &MockState{ctrl: ctrl}
	mock.recorder = &MockStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockState) EXPECT() *MockStateMockRecorder {
	return m.recorder
}

// HasUpgradeSeriesLocks mocks base method.
func (m *MockState) HasUpgradeSeriesLocks() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasUpgradeSeriesLocks")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HasUpgradeSeriesLocks indicates an expected call of HasUpgradeSeriesLocks.
func (mr *MockStateMockRecorder) HasUpgradeSeriesLocks() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasUpgradeSeriesLocks", reflect.TypeOf((*MockState)(nil).HasUpgradeSeriesLocks))
}

// MachineCountForBase mocks base method.
func (m *MockState) MachineCountForBase(arg0 ...state.Base) (map[string]int, error) {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "MachineCountForBase", varargs...)
	ret0, _ := ret[0].(map[string]int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MachineCountForBase indicates an expected call of MachineCountForBase.
func (mr *MockStateMockRecorder) MachineCountForBase(arg0 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MachineCountForBase", reflect.TypeOf((*MockState)(nil).MachineCountForBase), arg0...)
}

// MongoCurrentStatus mocks base method.
func (m *MockState) MongoCurrentStatus() (*replicaset.Status, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MongoCurrentStatus")
	ret0, _ := ret[0].(*replicaset.Status)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MongoCurrentStatus indicates an expected call of MongoCurrentStatus.
func (mr *MockStateMockRecorder) MongoCurrentStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MongoCurrentStatus", reflect.TypeOf((*MockState)(nil).MongoCurrentStatus))
}

// MockModel is a mock of Model interface.
type MockModel struct {
	ctrl     *gomock.Controller
	recorder *MockModelMockRecorder
}

// MockModelMockRecorder is the mock recorder for MockModel.
type MockModelMockRecorder struct {
	mock *MockModel
}

// NewMockModel creates a new mock instance.
func NewMockModel(ctrl *gomock.Controller) *MockModel {
	mock := &MockModel{ctrl: ctrl}
	mock.recorder = &MockModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModel) EXPECT() *MockModelMockRecorder {
	return m.recorder
}

// AgentVersion mocks base method.
func (m *MockModel) AgentVersion() (version.Number, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AgentVersion")
	ret0, _ := ret[0].(version.Number)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AgentVersion indicates an expected call of AgentVersion.
func (mr *MockModelMockRecorder) AgentVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AgentVersion", reflect.TypeOf((*MockModel)(nil).AgentVersion))
}

// MigrationMode mocks base method.
func (m *MockModel) MigrationMode() state.MigrationMode {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MigrationMode")
	ret0, _ := ret[0].(state.MigrationMode)
	return ret0
}

// MigrationMode indicates an expected call of MigrationMode.
func (mr *MockModelMockRecorder) MigrationMode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MigrationMode", reflect.TypeOf((*MockModel)(nil).MigrationMode))
}

// Name mocks base method.
func (m *MockModel) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockModelMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockModel)(nil).Name))
}

// Owner mocks base method.
func (m *MockModel) Owner() names.UserTag {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Owner")
	ret0, _ := ret[0].(names.UserTag)
	return ret0
}

// Owner indicates an expected call of Owner.
func (mr *MockModelMockRecorder) Owner() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Owner", reflect.TypeOf((*MockModel)(nil).Owner))
}
