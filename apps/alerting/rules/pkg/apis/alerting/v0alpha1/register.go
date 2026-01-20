package v0alpha1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
)

// AlertRuleResourceInfo provides resource information for AlertRule
var AlertRuleResourceInfo = utils.NewResourceInfo(APIGroup, APIVersion,
	"alertrules", "alertrule", "AlertRule",
	func() runtime.Object { return &AlertRule{} },
	func() runtime.Object { return &AlertRuleList{} },
	utils.TableColumns{
		Definition: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Title", Type: "string", Format: "string", Description: "The alert rule title"},
			{Name: "Paused", Type: "boolean", Description: "Whether the rule is paused"},
			{Name: "Created At", Type: "date"},
		},
		Reader: func(obj any) ([]interface{}, error) {
			r, ok := obj.(*AlertRule)
			if !ok {
				return nil, fmt.Errorf("expected alert rule")
			}
			paused := false
			if r.Spec.Paused != nil {
				paused = *r.Spec.Paused
			}
			return []interface{}{
				r.Name,
				r.Spec.Title,
				paused,
				r.CreationTimestamp.UTC().Format(time.RFC3339),
			}, nil
		},
	},
)

// RecordingRuleResourceInfo provides resource information for RecordingRule
var RecordingRuleResourceInfo = utils.NewResourceInfo(APIGroup, APIVersion,
	"recordingrules", "recordingrule", "RecordingRule",
	func() runtime.Object { return &RecordingRule{} },
	func() runtime.Object { return &RecordingRuleList{} },
	utils.TableColumns{
		Definition: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Title", Type: "string", Format: "string", Description: "The recording rule title"},
			{Name: "Paused", Type: "boolean", Description: "Whether the rule is paused"},
			{Name: "Created At", Type: "date"},
		},
		Reader: func(obj any) ([]interface{}, error) {
			r, ok := obj.(*RecordingRule)
			if !ok {
				return nil, fmt.Errorf("expected recording rule")
			}
			paused := false
			if r.Spec.Paused != nil {
				paused = *r.Spec.Paused
			}
			return []interface{}{
				r.Name,
				r.Spec.Title,
				paused,
				r.CreationTimestamp.UTC().Format(time.RFC3339),
			}, nil
		},
	},
)
