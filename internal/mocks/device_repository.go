// Code generated by mockery v2.42.3. DO NOT EDIT.

package mocks

import (
	context "context"
	domain "glooko/internal/domain"

	mock "github.com/stretchr/testify/mock"
)

// DeviceRepository is an autogenerated mock type for the DeviceRepository type
type DeviceRepository struct {
	mock.Mock
}

// FindByID provides a mock function with given fields: ctx, id
func (_m *DeviceRepository) FindByID(ctx context.Context, id string) (domain.Device, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for FindByID")
	}

	var r0 domain.Device
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (domain.Device, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) domain.Device); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(domain.Device)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Save provides a mock function with given fields: ctx, device
func (_m *DeviceRepository) Save(ctx context.Context, device domain.Device) (domain.Device, error) {
	ret := _m.Called(ctx, device)

	if len(ret) == 0 {
		panic("no return value specified for Save")
	}

	var r0 domain.Device
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.Device) (domain.Device, error)); ok {
		return rf(ctx, device)
	}
	if rf, ok := ret.Get(0).(func(context.Context, domain.Device) domain.Device); ok {
		r0 = rf(ctx, device)
	} else {
		r0 = ret.Get(0).(domain.Device)
	}

	if rf, ok := ret.Get(1).(func(context.Context, domain.Device) error); ok {
		r1 = rf(ctx, device)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewDeviceRepository creates a new instance of DeviceRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDeviceRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *DeviceRepository {
	mock := &DeviceRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}