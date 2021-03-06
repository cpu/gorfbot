// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/cpu/gorfbot/botcmd (interfaces: PatternHandler)

// Package mocks is a generated GoMock package.
package mocks

import (
	botcmd "github.com/cpu/gorfbot/botcmd"
	config "github.com/cpu/gorfbot/config"
	gomock "github.com/golang/mock/gomock"
	logrus "github.com/sirupsen/logrus"
	reflect "reflect"
)

// MockPatternHandler is a mock of PatternHandler interface
type MockPatternHandler struct {
	ctrl     *gomock.Controller
	recorder *MockPatternHandlerMockRecorder
}

// MockPatternHandlerMockRecorder is the mock recorder for MockPatternHandler
type MockPatternHandlerMockRecorder struct {
	mock *MockPatternHandler
}

// NewMockPatternHandler creates a new mock instance
func NewMockPatternHandler(ctrl *gomock.Controller) *MockPatternHandler {
	mock := &MockPatternHandler{ctrl: ctrl}
	mock.recorder = &MockPatternHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPatternHandler) EXPECT() *MockPatternHandlerMockRecorder {
	return m.recorder
}

// Configure mocks base method
func (m *MockPatternHandler) Configure(arg0 *logrus.Logger, arg1 *config.Config) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Configure", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Configure indicates an expected call of Configure
func (mr *MockPatternHandlerMockRecorder) Configure(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Configure", reflect.TypeOf((*MockPatternHandler)(nil).Configure), arg0, arg1)
}

// Run mocks base method
func (m *MockPatternHandler) Run(arg0 [][]string, arg1 botcmd.RunContext) (botcmd.RunResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", arg0, arg1)
	ret0, _ := ret[0].(botcmd.RunResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Run indicates an expected call of Run
func (mr *MockPatternHandlerMockRecorder) Run(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockPatternHandler)(nil).Run), arg0, arg1)
}
