package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceSpaceTypeContract(t *testing.T) {
	tests := []struct {
		name  string
		value ServiceSpaceType
		valid bool
	}{
		{name: "customer service", value: ServiceSpaceTypeCustomerService, valid: true},
		{name: "operations", value: ServiceSpaceTypeOperations, valid: true},
		{name: "research", value: ServiceSpaceTypeResearch, valid: true},
		{name: "empty", value: "", valid: false},
		{name: "legacy value", value: "legacy", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.valid, tt.value.IsValid())
		})
	}
}

func TestServiceSpaceBeforeCreateAppliesV1Defaults(t *testing.T) {
	space := &ServiceSpace{}
	require.NoError(t, space.BeforeCreate(nil))
	require.Equal(t, ServiceSpaceStateDraft, space.State)
	require.Equal(t, ServiceSpaceTypeCustomerService, space.SpaceType)
	require.NotNil(t, space.StateChangedAt)
	require.WithinDuration(t, time.Now().UTC(), *space.StateChangedAt, time.Second)
}
