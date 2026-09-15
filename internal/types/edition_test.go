package types

import "testing"

func TestEditionForTenantMapsPersonalAndEnterpriseWorkspaces(t *testing.T) {
	personal := SpaceTypePersonal
	if got := EditionForTenant(&Tenant{SpaceType: &personal}); got != EditionPersonal {
		t.Fatalf("personal edition = %q, want %q", got, EditionPersonal)
	}

	organization := SpaceTypeOrganization
	if got := EditionForTenant(&Tenant{SpaceType: &organization}); got != EditionEnterprise {
		t.Fatalf("organization edition = %q, want %q", got, EditionEnterprise)
	}
	if got := EditionForTenant(&Tenant{}); got != EditionEnterprise {
		t.Fatalf("legacy edition = %q, want enterprise-compatible edition", got)
	}
}
