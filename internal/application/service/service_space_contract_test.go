package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestServiceSpaceCreateRejectsInvalidSpaceType(t *testing.T) {
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	_, err := svc.Create(context.Background(), 101, "owner", types.ServiceSpaceCreateInput{
		Name:      "非法空间类型测试",
		SpaceType: "legacy",
	})
	require.ErrorIs(t, err, ErrServiceSpaceInvalidType)
}

func TestServiceSpaceCreateDefaultsCustomerServiceType(t *testing.T) {
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(context.Background(), 102, "owner", types.ServiceSpaceCreateInput{
		Name: "默认客户服务空间",
	})
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceTypeCustomerService, space.SpaceType)
	require.Equal(t, types.ServiceSpaceStateDraft, space.State)
	require.NotNil(t, space.StateChangedAt)
}
