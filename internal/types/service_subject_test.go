package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeServiceSubjectTypeAllowsConfiguredDomainValues(t *testing.T) {
	for _, value := range []string{"person", "family", "organization", "research_topic", "custom"} {
		normalized, err := NormalizeServiceSubjectType(value)
		require.NoError(t, err)
		require.Equal(t, value, normalized)
	}

	normalized, err := NormalizeServiceSubjectType("  Project-Team  ")
	require.NoError(t, err)
	require.Equal(t, "project-team", normalized)
}

func TestNormalizeServiceSubjectTypeRejectsInvalidKeys(t *testing.T) {
	tests := []string{"", "1person", "person type", "person/child", strings.Repeat("a", ServiceSubjectTypeMaxLen+1)}
	for _, value := range tests {
		_, err := NormalizeServiceSubjectType(value)
		require.Error(t, err, value)
	}
}

func TestServiceSubjectBeforeCreateAppliesGenericDefaults(t *testing.T) {
	subject := &ServiceSubject{
		TenantID:   7,
		ServiceID:  "service-1",
		SubjectKey: "project-001",
	}

	require.NoError(t, subject.BeforeCreate(nil))
	require.NotEmpty(t, subject.ID)
	require.Equal(t, ServiceSubjectTypeCustom, subject.SubjectType)
	require.Equal(t, ServiceSpaceVisibilityPrivate, subject.VisibilityScope)
	require.NotNil(t, subject.Metadata)
	require.NotNil(t, subject.Aliases)
	require.NotNil(t, subject.ExternalRefs)
}

func TestServiceSubjectBeforeCreateRequiresServiceScope(t *testing.T) {
	subject := &ServiceSubject{SubjectKey: "subject-1"}
	require.ErrorContains(t, subject.BeforeCreate(nil), "service_id is required")
}
