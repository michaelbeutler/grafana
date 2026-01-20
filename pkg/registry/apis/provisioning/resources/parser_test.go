package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	alertingv0alpha1 "github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	dashboardV0 "github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v0alpha1"
	dashboardV1 "github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v1beta1"
	provisioning "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/grafana/apps/provisioning/pkg/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParser(t *testing.T) {
	clients := NewMockResourceClients(t)
	clients.On("ForKind", mock.Anything, dashboardV0.DashboardResourceInfo.GroupVersionKind()).
		Return(nil, dashboardV0.DashboardResourceInfo.GroupVersionResource(), nil).Maybe()
	clients.On("ForKind", mock.Anything, dashboardV1.DashboardResourceInfo.GroupVersionKind()).
		Return(nil, dashboardV1.DashboardResourceInfo.GroupVersionResource(), nil).Maybe()

	parser := &parser{
		repo: provisioning.ResourceRepositoryInfo{
			Type:      provisioning.LocalRepositoryType,
			Namespace: "xxx",
			Name:      "repo",
		},
		clients: clients,
		config: &provisioning.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "xxx",
				Name:      "repo",
			},
			Spec: provisioning.RepositorySpec{
				Type: provisioning.LocalRepositoryType,
				Sync: provisioning.SyncOptions{Target: provisioning.SyncTargetTypeFolder},
			},
		},
	}

	t.Run("invalid input", func(t *testing.T) {
		_, err := parser.Parse(context.Background(), &repository.FileInfo{
			Data: []byte("hello"), // not a real resource
		})
		require.Error(t, err)
		require.Equal(t, "unable to read file as a resource", err.Error())
	})

	t.Run("dashboard parsing (with and without name)", func(t *testing.T) {
		dash, err := parser.Parse(context.Background(), &repository.FileInfo{
			Data: []byte(`apiVersion: dashboard.grafana.app/v0alpha1
kind: Dashboard
metadata:
  name: test-v0
spec:
  title: Test dashboard
`),
		})
		require.NoError(t, err)
		require.Equal(t, "test-v0", dash.Obj.GetName())
		require.Equal(t, "dashboard.grafana.app", dash.GVK.Group)
		require.Equal(t, "v0alpha1", dash.GVK.Version)
		require.Equal(t, "dashboard.grafana.app", dash.GVR.Group)
		require.Equal(t, "v0alpha1", dash.GVR.Version)

		// Now try again without a name
		_, err = parser.Parse(context.Background(), &repository.FileInfo{
			Data: []byte(`apiVersion: ` + dashboardV1.APIVERSION + `
kind: Dashboard
spec:
  title: Test dashboard
`),
		})
		require.EqualError(t, err, "name.metadata.name: Required value: missing name in resource")
	})

	t.Run("generate name will generate a name", func(t *testing.T) {
		dash, err := parser.Parse(context.Background(), &repository.FileInfo{
			Data: []byte(`apiVersion: dashboard.grafana.app/v0alpha1
kind: Dashboard
metadata:
  generateName: rand-
spec:
  title: Test dashboard
`),
		})
		require.NoError(t, err)
		require.Equal(t, "dashboard.grafana.app", dash.GVK.Group)
		require.Equal(t, "v0alpha1", dash.GVK.Version)
		require.True(t, strings.HasPrefix(dash.Obj.GetName(), "rand-"), "set name")
	})

	t.Run("dashboard classic format", func(t *testing.T) {
		dash, err := parser.Parse(context.Background(), &repository.FileInfo{
			Data: []byte(`{ "uid": "test", "schemaVersion": 30, "panels": [], "tags": [] }`),
		})
		require.NoError(t, err)
		require.Equal(t, "test", dash.Obj.GetName())
		require.Equal(t, provisioning.ClassicDashboard, dash.Classic)
		require.Equal(t, "dashboard.grafana.app", dash.GVK.Group)
		require.Equal(t, "v0alpha1", dash.GVK.Version)
		require.Equal(t, "dashboard.grafana.app", dash.GVR.Group)
		require.Equal(t, "v0alpha1", dash.GVR.Version)
	})

	t.Run("validate proper folder metadata is set", func(t *testing.T) {
		testCases := []struct {
			name           string
			filePath       string
			expectedFolder string
		}{
			{
				name:           "file in subdirectory should use parsed folder ID",
				filePath:       "team-a/testing-valid-dashboard.json",
				expectedFolder: ParseFolder("team-a/", "repo").ID,
			},
			{
				name:           "file in first-level directory should use parent folder id",
				filePath:       "testing-valid-dashboard.json",
				expectedFolder: parser.repo.Name,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				dash, err := parser.Parse(context.Background(), &repository.FileInfo{
					Path: tc.filePath,
					Data: []byte(`apiVersion: dashboard.grafana.app/v0alpha1
kind: Dashboard
metadata:
  name: test-dashboard
spec:
  title: Test dashboard
`),
				})
				require.NoError(t, err)
				require.Equal(t, tc.expectedFolder, dash.Meta.GetFolder(), "folder should match expected")
				annotations := dash.Obj.GetAnnotations()
				require.NotNil(t, annotations, "annotations should not be nil")
				require.Equal(t, tc.expectedFolder, annotations["grafana.app/folder"], "folder annotation should match expected")
			})
		}
	})
}

func TestParserAlertRules(t *testing.T) {
	clients := NewMockResourceClients(t)
	clients.On("ForKind", mock.Anything, alertingv0alpha1.AlertRuleResourceInfo.GroupVersionKind()).
		Return(nil, alertingv0alpha1.AlertRuleResourceInfo.GroupVersionResource(), nil).Maybe()
	clients.On("ForKind", mock.Anything, alertingv0alpha1.RecordingRuleResourceInfo.GroupVersionKind()).
		Return(nil, alertingv0alpha1.RecordingRuleResourceInfo.GroupVersionResource(), nil).Maybe()

	parser := &parser{
		repo: provisioning.ResourceRepositoryInfo{
			Type:      provisioning.LocalRepositoryType,
			Namespace: "xxx",
			Name:      "repo",
		},
		clients: clients,
		config: &provisioning.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "xxx",
				Name:      "repo",
			},
			Spec: provisioning.RepositorySpec{
				Type: provisioning.LocalRepositoryType,
				Sync: provisioning.SyncOptions{Target: provisioning.SyncTargetTypeFolder},
			},
		},
	}

	t.Run("alert rule parsing", func(t *testing.T) {
		alertRule, err := parser.Parse(context.Background(), &repository.FileInfo{
			Path: "alerts/test-alert.yaml",
			Data: []byte(`apiVersion: rules.alerting.grafana.app/v0alpha1
kind: AlertRule
metadata:
  name: test-alert-rule
spec:
  title: Test Alert Rule
  noDataState: NoData
  execErrState: Error
  trigger:
    interval: 1m
  expressions:
    A:
      queryType: query
      datasourceUID: datasource-uid
      source: true
      model:
        refId: A
        expr: up == 0
      relativeTimeRange:
        from: 5m
        to: 0s
`),
		})
		require.NoError(t, err)
		require.Equal(t, "test-alert-rule", alertRule.Obj.GetName())
		require.Equal(t, "rules.alerting.grafana.app", alertRule.GVK.Group)
		require.Equal(t, "v0alpha1", alertRule.GVK.Version)
		require.Equal(t, "AlertRule", alertRule.GVK.Kind)
		require.Equal(t, "rules.alerting.grafana.app", alertRule.GVR.Group)
		require.Equal(t, "v0alpha1", alertRule.GVR.Version)
		require.Equal(t, "alertrules", alertRule.GVR.Resource)

		// Verify folder is set from path
		expectedFolder := ParseFolder("alerts/", "repo").ID
		require.Equal(t, expectedFolder, alertRule.Meta.GetFolder(), "folder should be set from file path")
	})

	t.Run("recording rule parsing", func(t *testing.T) {
		recordingRule, err := parser.Parse(context.Background(), &repository.FileInfo{
			Path: "recording/test-recording.yaml",
			Data: []byte(`apiVersion: rules.alerting.grafana.app/v0alpha1
kind: RecordingRule
metadata:
  name: test-recording-rule
spec:
  title: Test Recording Rule
  metric: test_metric
  targetDatasourceUID: prometheus
  trigger:
    interval: 1m
  expressions:
    A:
      queryType: query
      datasourceUID: datasource-uid
      source: true
      model:
        refId: A
        expr: sum(rate(http_requests_total[5m]))
      relativeTimeRange:
        from: 5m
        to: 0s
`),
		})
		require.NoError(t, err)
		require.Equal(t, "test-recording-rule", recordingRule.Obj.GetName())
		require.Equal(t, "rules.alerting.grafana.app", recordingRule.GVK.Group)
		require.Equal(t, "v0alpha1", recordingRule.GVK.Version)
		require.Equal(t, "RecordingRule", recordingRule.GVK.Kind)
		require.Equal(t, "rules.alerting.grafana.app", recordingRule.GVR.Group)
		require.Equal(t, "v0alpha1", recordingRule.GVR.Version)
		require.Equal(t, "recordingrules", recordingRule.GVR.Resource)

		// Verify folder is set from path
		expectedFolder := ParseFolder("recording/", "repo").ID
		require.Equal(t, expectedFolder, recordingRule.Meta.GetFolder(), "folder should be set from file path")
	})
}
