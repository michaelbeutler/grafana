package resources

import (
	"testing"

	"github.com/stretchr/testify/require"

	alertingv0alpha1 "github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	dashboardV1 "github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v1beta1"
	folders "github.com/grafana/grafana/apps/folder/pkg/apis/folder/v1beta1"
)

func TestSupportedProvisioningResources(t *testing.T) {
	// Verify that SupportedProvisioningResources contains the expected resources
	require.Len(t, SupportedProvisioningResources, 4, "should have 4 supported provisioning resources")

	// Verify each expected resource is in the list
	resourceGroups := make(map[string]bool)
	for _, gvr := range SupportedProvisioningResources {
		resourceGroups[gvr.GroupResource().String()] = true
	}

	require.True(t, resourceGroups[FolderResource.GroupResource().String()], "should include folders")
	require.True(t, resourceGroups[DashboardResource.GroupResource().String()], "should include dashboards")
	require.True(t, resourceGroups[AlertRuleResource.GroupResource().String()], "should include alert rules")
	require.True(t, resourceGroups[RecordingRuleResource.GroupResource().String()], "should include recording rules")
}

func TestSupportsFolderAnnotation(t *testing.T) {
	// Verify that SupportsFolderAnnotation contains the expected resources
	require.Len(t, SupportsFolderAnnotation, 4, "should have 4 resources that support folder annotation")

	// Verify each expected resource is in the list
	resourceGroups := make(map[string]bool)
	for _, gr := range SupportsFolderAnnotation {
		resourceGroups[gr.String()] = true
	}

	require.True(t, resourceGroups[FolderResource.GroupResource().String()], "should include folders")
	require.True(t, resourceGroups[DashboardResource.GroupResource().String()], "should include dashboards")
	require.True(t, resourceGroups[AlertRuleResource.GroupResource().String()], "should include alert rules")
	require.True(t, resourceGroups[RecordingRuleResource.GroupResource().String()], "should include recording rules")
}

func TestResourceInfoDefinitions(t *testing.T) {
	// Verify AlertRuleResource is correctly defined
	require.Equal(t, alertingv0alpha1.APIGroup, AlertRuleResource.Group)
	require.Equal(t, alertingv0alpha1.APIVersion, AlertRuleResource.Version)
	require.Equal(t, "alertrules", AlertRuleResource.Resource)

	// Verify RecordingRuleResource is correctly defined
	require.Equal(t, alertingv0alpha1.APIGroup, RecordingRuleResource.Group)
	require.Equal(t, alertingv0alpha1.APIVersion, RecordingRuleResource.Version)
	require.Equal(t, "recordingrules", RecordingRuleResource.Resource)

	// Verify DashboardResource is correctly defined
	require.Equal(t, dashboardV1.GROUP, DashboardResource.Group)
	require.Equal(t, dashboardV1.VERSION, DashboardResource.Version)
	require.Equal(t, dashboardV1.DASHBOARD_RESOURCE, DashboardResource.Resource)

	// Verify FolderResource is correctly defined
	require.Equal(t, folders.GROUP, FolderResource.Group)
	require.Equal(t, folders.VERSION, FolderResource.Version)
	require.Equal(t, folders.RESOURCE, FolderResource.Resource)
}
