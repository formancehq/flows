// Code generated by MockGen. DO NOT EDIT.
// Source: backend.go
//
// Generated by this command:
//
//	mockgen -source backend.go -destination backend_generated.go -package api . Backend
//
// Package api is a generated GoMock package.
package api

import (
	context "context"
	reflect "reflect"

	triggers "github.com/formancehq/orchestration/internal/triggers"
	workflow "github.com/formancehq/orchestration/internal/workflow"
	bunpaginate "github.com/formancehq/stack/libs/go-libs/bun/bunpaginate"
	gomock "go.uber.org/mock/gomock"
)

// MockBackend is a mock of Backend interface.
type MockBackend struct {
	ctrl     *gomock.Controller
	recorder *MockBackendMockRecorder
}

// MockBackendMockRecorder is the mock recorder for MockBackend.
type MockBackendMockRecorder struct {
	mock *MockBackend
}

// NewMockBackend creates a new mock instance.
func NewMockBackend(ctrl *gomock.Controller) *MockBackend {
	mock := &MockBackend{ctrl: ctrl}
	mock.recorder = &MockBackendMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackend) EXPECT() *MockBackendMockRecorder {
	return m.recorder
}

// AbortRun mocks base method.
func (m *MockBackend) AbortRun(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AbortRun", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// AbortRun indicates an expected call of AbortRun.
func (mr *MockBackendMockRecorder) AbortRun(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AbortRun", reflect.TypeOf((*MockBackend)(nil).AbortRun), ctx, id)
}

// Create mocks base method.
func (m *MockBackend) Create(ctx context.Context, config workflow.Config) (*workflow.Workflow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, config)
	ret0, _ := ret[0].(*workflow.Workflow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockBackendMockRecorder) Create(ctx, config any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockBackend)(nil).Create), ctx, config)
}

// CreateTrigger mocks base method.
func (m *MockBackend) CreateTrigger(context context.Context, data triggers.TriggerData) (*triggers.Trigger, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTrigger", context, data)
	ret0, _ := ret[0].(*triggers.Trigger)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateTrigger indicates an expected call of CreateTrigger.
func (mr *MockBackendMockRecorder) CreateTrigger(context, data any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTrigger", reflect.TypeOf((*MockBackend)(nil).CreateTrigger), context, data)
}

// DeleteTrigger mocks base method.
func (m *MockBackend) DeleteTrigger(ctx context.Context, triggerID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTrigger", ctx, triggerID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTrigger indicates an expected call of DeleteTrigger.
func (mr *MockBackendMockRecorder) DeleteTrigger(ctx, triggerID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTrigger", reflect.TypeOf((*MockBackend)(nil).DeleteTrigger), ctx, triggerID)
}

// DeleteWorkflow mocks base method.
func (m *MockBackend) DeleteWorkflow(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteWorkflow", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteWorkflow indicates an expected call of DeleteWorkflow.
func (mr *MockBackendMockRecorder) DeleteWorkflow(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteWorkflow", reflect.TypeOf((*MockBackend)(nil).DeleteWorkflow), ctx, id)
}

// GetInstance mocks base method.
func (m *MockBackend) GetInstance(ctx context.Context, id string) (*workflow.Instance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInstance", ctx, id)
	ret0, _ := ret[0].(*workflow.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInstance indicates an expected call of GetInstance.
func (mr *MockBackendMockRecorder) GetInstance(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInstance", reflect.TypeOf((*MockBackend)(nil).GetInstance), ctx, id)
}

// GetTrigger mocks base method.
func (m *MockBackend) GetTrigger(ctx context.Context, triggerID string) (*triggers.Trigger, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTrigger", ctx, triggerID)
	ret0, _ := ret[0].(*triggers.Trigger)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTrigger indicates an expected call of GetTrigger.
func (mr *MockBackendMockRecorder) GetTrigger(ctx, triggerID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTrigger", reflect.TypeOf((*MockBackend)(nil).GetTrigger), ctx, triggerID)
}

// ListInstances mocks base method.
func (m *MockBackend) ListInstances(ctx context.Context, pagination workflow.ListInstancesQuery) (*bunpaginate.Cursor[workflow.Instance], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListInstances", ctx, pagination)
	ret0, _ := ret[0].(*bunpaginate.Cursor[workflow.Instance])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListInstances indicates an expected call of ListInstances.
func (mr *MockBackendMockRecorder) ListInstances(ctx, pagination any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListInstances", reflect.TypeOf((*MockBackend)(nil).ListInstances), ctx, pagination)
}

// ListTriggers mocks base method.
func (m *MockBackend) ListTriggers(ctx context.Context, query triggers.ListTriggersQuery) (*bunpaginate.Cursor[triggers.Trigger], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTriggers", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[triggers.Trigger])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTriggers indicates an expected call of ListTriggers.
func (mr *MockBackendMockRecorder) ListTriggers(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTriggers", reflect.TypeOf((*MockBackend)(nil).ListTriggers), ctx, query)
}

// ListTriggersOccurrences mocks base method.
func (m *MockBackend) ListTriggersOccurrences(ctx context.Context, query triggers.ListTriggersOccurrencesQuery) (*bunpaginate.Cursor[triggers.Occurrence], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTriggersOccurrences", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[triggers.Occurrence])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTriggersOccurrences indicates an expected call of ListTriggersOccurrences.
func (mr *MockBackendMockRecorder) ListTriggersOccurrences(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTriggersOccurrences", reflect.TypeOf((*MockBackend)(nil).ListTriggersOccurrences), ctx, query)
}

// ListWorkflows mocks base method.
func (m *MockBackend) ListWorkflows(ctx context.Context, query bunpaginate.OffsetPaginatedQuery[any]) (*bunpaginate.Cursor[workflow.Workflow], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListWorkflows", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[workflow.Workflow])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListWorkflows indicates an expected call of ListWorkflows.
func (mr *MockBackendMockRecorder) ListWorkflows(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListWorkflows", reflect.TypeOf((*MockBackend)(nil).ListWorkflows), ctx, query)
}

// PostEvent mocks base method.
func (m *MockBackend) PostEvent(ctx context.Context, id string, event workflow.Event) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostEvent", ctx, id, event)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostEvent indicates an expected call of PostEvent.
func (mr *MockBackendMockRecorder) PostEvent(ctx, id, event any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostEvent", reflect.TypeOf((*MockBackend)(nil).PostEvent), ctx, id, event)
}

// ReadInstanceHistory mocks base method.
func (m *MockBackend) ReadInstanceHistory(ctx context.Context, id string) ([]workflow.StageHistory, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadInstanceHistory", ctx, id)
	ret0, _ := ret[0].([]workflow.StageHistory)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadInstanceHistory indicates an expected call of ReadInstanceHistory.
func (mr *MockBackendMockRecorder) ReadInstanceHistory(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadInstanceHistory", reflect.TypeOf((*MockBackend)(nil).ReadInstanceHistory), ctx, id)
}

// ReadStageHistory mocks base method.
func (m *MockBackend) ReadStageHistory(ctx context.Context, instanceID string, stage int) ([]*workflow.ActivityHistory, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadStageHistory", ctx, instanceID, stage)
	ret0, _ := ret[0].([]*workflow.ActivityHistory)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadStageHistory indicates an expected call of ReadStageHistory.
func (mr *MockBackendMockRecorder) ReadStageHistory(ctx, instanceID, stage any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadStageHistory", reflect.TypeOf((*MockBackend)(nil).ReadStageHistory), ctx, instanceID, stage)
}

// ReadWorkflow mocks base method.
func (m *MockBackend) ReadWorkflow(ctx context.Context, id string) (workflow.Workflow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadWorkflow", ctx, id)
	ret0, _ := ret[0].(workflow.Workflow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadWorkflow indicates an expected call of ReadWorkflow.
func (mr *MockBackendMockRecorder) ReadWorkflow(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadWorkflow", reflect.TypeOf((*MockBackend)(nil).ReadWorkflow), ctx, id)
}

// RunWorkflow mocks base method.
func (m *MockBackend) RunWorkflow(ctx context.Context, id string, input map[string]string) (*workflow.Instance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RunWorkflow", ctx, id, input)
	ret0, _ := ret[0].(*workflow.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RunWorkflow indicates an expected call of RunWorkflow.
func (mr *MockBackendMockRecorder) RunWorkflow(ctx, id, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunWorkflow", reflect.TypeOf((*MockBackend)(nil).RunWorkflow), ctx, id, input)
}

// TestTrigger mocks base method.
func (m *MockBackend) TestTrigger(ctx context.Context, triggerID string, data map[string]any) (*triggers.TestTriggerResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TestTrigger", ctx, triggerID, data)
	ret0, _ := ret[0].(*triggers.TestTriggerResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TestTrigger indicates an expected call of TestTrigger.
func (mr *MockBackendMockRecorder) TestTrigger(ctx, triggerID, data any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TestTrigger", reflect.TypeOf((*MockBackend)(nil).TestTrigger), ctx, triggerID, data)
}

// Wait mocks base method.
func (m *MockBackend) Wait(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Wait", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Wait indicates an expected call of Wait.
func (mr *MockBackendMockRecorder) Wait(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Wait", reflect.TypeOf((*MockBackend)(nil).Wait), ctx, id)
}
