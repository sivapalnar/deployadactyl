package mocks

import (
	"github.com/compozed/deployadactyl/config"
	"github.com/stretchr/testify/mock"
)

// Prechecker is an autogenerated mock type for the Prechecker type
type Prechecker struct {
	mock.Mock
}

// AssertAllFoundationsUp provides a mock function with given fields: environment
func (_m *Prechecker) AssertAllFoundationsUp(environment config.Environment) error {
	ret := _m.Called(environment)

	var r0 error
	if rf, ok := ret.Get(0).(func(config.Environment) error); ok {
		r0 = rf(environment)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}