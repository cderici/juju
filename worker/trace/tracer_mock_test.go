// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/worker/trace (interfaces: TrackedTracer,Client,ClientTracer,ClientTracerProvider)

// Package trace is a generated GoMock package.
package trace

import (
	context "context"
	reflect "reflect"

	trace "github.com/juju/juju/core/trace"
	trace0 "go.opentelemetry.io/otel/trace"
	gomock "go.uber.org/mock/gomock"
)

// MockTrackedTracer is a mock of TrackedTracer interface.
type MockTrackedTracer struct {
	ctrl     *gomock.Controller
	recorder *MockTrackedTracerMockRecorder
}

// MockTrackedTracerMockRecorder is the mock recorder for MockTrackedTracer.
type MockTrackedTracerMockRecorder struct {
	mock *MockTrackedTracer
}

// NewMockTrackedTracer creates a new mock instance.
func NewMockTrackedTracer(ctrl *gomock.Controller) *MockTrackedTracer {
	mock := &MockTrackedTracer{ctrl: ctrl}
	mock.recorder = &MockTrackedTracerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTrackedTracer) EXPECT() *MockTrackedTracerMockRecorder {
	return m.recorder
}

// Enabled mocks base method.
func (m *MockTrackedTracer) Enabled() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enabled")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Enabled indicates an expected call of Enabled.
func (mr *MockTrackedTracerMockRecorder) Enabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enabled", reflect.TypeOf((*MockTrackedTracer)(nil).Enabled))
}

// Kill mocks base method.
func (m *MockTrackedTracer) Kill() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Kill")
}

// Kill indicates an expected call of Kill.
func (mr *MockTrackedTracerMockRecorder) Kill() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Kill", reflect.TypeOf((*MockTrackedTracer)(nil).Kill))
}

// Start mocks base method.
func (m *MockTrackedTracer) Start(arg0 context.Context, arg1 string, arg2 ...trace.Option) (context.Context, trace.Span) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Start", varargs...)
	ret0, _ := ret[0].(context.Context)
	ret1, _ := ret[1].(trace.Span)
	return ret0, ret1
}

// Start indicates an expected call of Start.
func (mr *MockTrackedTracerMockRecorder) Start(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockTrackedTracer)(nil).Start), varargs...)
}

// Wait mocks base method.
func (m *MockTrackedTracer) Wait() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Wait")
	ret0, _ := ret[0].(error)
	return ret0
}

// Wait indicates an expected call of Wait.
func (mr *MockTrackedTracerMockRecorder) Wait() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Wait", reflect.TypeOf((*MockTrackedTracer)(nil).Wait))
}

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// Start mocks base method.
func (m *MockClient) Start(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockClientMockRecorder) Start(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockClient)(nil).Start), arg0)
}

// Stop mocks base method.
func (m *MockClient) Stop(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockClientMockRecorder) Stop(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockClient)(nil).Stop), arg0)
}

// MockClientTracer is a mock of ClientTracer interface.
type MockClientTracer struct {
	ctrl     *gomock.Controller
	recorder *MockClientTracerMockRecorder
}

// MockClientTracerMockRecorder is the mock recorder for MockClientTracer.
type MockClientTracerMockRecorder struct {
	mock *MockClientTracer
}

// NewMockClientTracer creates a new mock instance.
func NewMockClientTracer(ctrl *gomock.Controller) *MockClientTracer {
	mock := &MockClientTracer{ctrl: ctrl}
	mock.recorder = &MockClientTracerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClientTracer) EXPECT() *MockClientTracerMockRecorder {
	return m.recorder
}

// Start mocks base method.
func (m *MockClientTracer) Start(arg0 context.Context, arg1 string, arg2 ...trace0.SpanStartOption) (context.Context, trace0.Span) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Start", varargs...)
	ret0, _ := ret[0].(context.Context)
	ret1, _ := ret[1].(trace0.Span)
	return ret0, ret1
}

// Start indicates an expected call of Start.
func (mr *MockClientTracerMockRecorder) Start(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockClientTracer)(nil).Start), varargs...)
}

// MockClientTracerProvider is a mock of ClientTracerProvider interface.
type MockClientTracerProvider struct {
	ctrl     *gomock.Controller
	recorder *MockClientTracerProviderMockRecorder
}

// MockClientTracerProviderMockRecorder is the mock recorder for MockClientTracerProvider.
type MockClientTracerProviderMockRecorder struct {
	mock *MockClientTracerProvider
}

// NewMockClientTracerProvider creates a new mock instance.
func NewMockClientTracerProvider(ctrl *gomock.Controller) *MockClientTracerProvider {
	mock := &MockClientTracerProvider{ctrl: ctrl}
	mock.recorder = &MockClientTracerProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClientTracerProvider) EXPECT() *MockClientTracerProviderMockRecorder {
	return m.recorder
}

// ForceFlush mocks base method.
func (m *MockClientTracerProvider) ForceFlush(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ForceFlush", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ForceFlush indicates an expected call of ForceFlush.
func (mr *MockClientTracerProviderMockRecorder) ForceFlush(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForceFlush", reflect.TypeOf((*MockClientTracerProvider)(nil).ForceFlush), arg0)
}

// Shutdown mocks base method.
func (m *MockClientTracerProvider) Shutdown(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Shutdown", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Shutdown indicates an expected call of Shutdown.
func (mr *MockClientTracerProviderMockRecorder) Shutdown(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Shutdown", reflect.TypeOf((*MockClientTracerProvider)(nil).Shutdown), arg0)
}
