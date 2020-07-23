/*
Copyright 2020 The Bulward Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectRoleTemplateSpec describes the desired state of ProjectRoleTemplate.
type ProjectRoleTemplateSpec struct {
	// Metadata contains additional human readable ProjectRoleTemplate details.
	Metadata *ProjectRoleTemplateMetadata `json:"metadata,omitempty"`
	// BindTo defines the member types of the Project that this ProjectRoleTemplate will be bound to.
	BindTo []BindingType `json:"bindTo,omitempty"`
	// ProjectSelector selects applicable target Projects.
	ProjectSelector *metav1.LabelSelector `json:"projectSelector,omitempty"`
	// Rules creates RBAC Roles that will be managed by this ProjectRoleTemplate.
	Rules []rbacv1.PolicyRule `json:"rules"`
}

// ProjectRoleTemplateMetadata contains the metadata of the ProjectRoleTemplate.
type ProjectRoleTemplateMetadata struct {
	// DisplayName is the human-readable name of this ProjectRoleTemplate.
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`
	// Description is the long and detailed description of the ProjectRoleTemplate.
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`
}

// ProjectRoleTemplateStatus represents the observed state of ProjectRoleTemplate.
type ProjectRoleTemplateStatus struct {
	// ObservedGeneration is the most recent generation observed for this ProjectRoleTemplate by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represents the latest available observations of a ProjectRoleTemplate's current state.
	Conditions []ProjectRoleTemplateCondition `json:"conditions,omitempty"`
	// DEPRECATED.
	// Phase represents the current lifecycle state of this object.
	// Consider this field DEPRECATED, it will be removed as soon as there
	// is a mechanism to map conditions to strings when printing the property.
	// This is only for display purpose, for everything else use conditions.
	Phase ProjectRoleTemplatePhaseType `json:"phase,omitempty"`
	// Targets holds different targets(Project, Project) that this ProjectRoleTemplate targets to.
	Targets []RoleTemplateTarget `json:"targets,omitempty"`
}

// ProjectRoleTemplatePhaseType represents all conditions as a single string for printing by using kubectl commands.
// +kubebuilder:validation:Ready;NotReady;Unknown;Terminating
type ProjectRoleTemplatePhaseType string

// Values of ProjectRoleTemplatePhaseType.
const (
	ProjectRoleTemplatePhaseReady       ProjectRoleTemplatePhaseType = "Ready"
	ProjectRoleTemplatePhaseNotReady    ProjectRoleTemplatePhaseType = "NotReady"
	ProjectRoleTemplatePhaseUnknown     ProjectRoleTemplatePhaseType = "Unknown"
	ProjectRoleTemplatePhaseTerminating ProjectRoleTemplatePhaseType = "Terminating"
)

const (
	ProjectRoleTemplateTerminatingReason = "Deleting"
)

// updatePhase updates the phase property based on the current conditions.
// this method should be called every time the conditions are updated.
func (s *ProjectRoleTemplateStatus) updatePhase() {
	for _, condition := range s.Conditions {
		if condition.Type != ProjectRoleTemplateReady {
			continue
		}

		switch condition.Status {
		case ConditionTrue:
			s.Phase = ProjectRoleTemplatePhaseReady
		case ConditionFalse:
			if condition.Reason == ProjectRoleTemplateTerminatingReason {
				s.Phase = ProjectRoleTemplatePhaseTerminating
			} else {
				s.Phase = ProjectRoleTemplatePhaseNotReady
			}
		case ConditionUnknown:
			s.Phase = ProjectRoleTemplatePhaseUnknown
		}
		return
	}

	s.Phase = ProjectRoleTemplatePhaseUnknown
}

// ProjectRoleTemplateConditionType represents a ProjectRoleTemplateCondition value.
// +kubebuilder:validation:Ready
type ProjectRoleTemplateConditionType string

const (
	// ProjectRoleTemplateReady represents a ProjectRoleTemplate condition is in ready state.
	ProjectRoleTemplateReady ProjectRoleTemplateConditionType = "Ready"
)

// ProjectRoleTemplateCondition contains details for the current condition of this ProjectRoleTemplate.
type ProjectRoleTemplateCondition struct {
	// Type is the type of the ProjectRoleTemplate condition, currently ('Ready').
	Type ProjectRoleTemplateConditionType `json:"type"`
	// Status is the status of the condition, one of ('True', 'False', 'Unknown').
	Status ConditionStatus `json:"status"`
	// LastTransitionTime is the last time the condition transits from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// Reason is the (brief) reason for the condition's last transition.
	Reason string `json:"reason"`
	// Message is the human readable message indicating details about last transition.
	Message string `json:"message"`
}

// GetCondition returns the Condition of the given condition type, if it exists.
func (s *ProjectRoleTemplateStatus) GetCondition(t ProjectRoleTemplateConditionType) (condition ProjectRoleTemplateCondition, exists bool) {
	for _, cond := range s.Conditions {
		if cond.Type == t {
			condition = cond
			exists = true
			return
		}
	}
	return
}

// SetCondition replaces or adds the given condition.
func (s *ProjectRoleTemplateStatus) SetCondition(condition ProjectRoleTemplateCondition) {
	defer s.updatePhase()

	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = metav1.Now()
	}

	for i := range s.Conditions {
		if s.Conditions[i].Type == condition.Type {

			// Only update the LastTransitionTime when the Status is changed.
			if s.Conditions[i].Status != condition.Status {
				s.Conditions[i].LastTransitionTime = condition.LastTransitionTime
			}

			s.Conditions[i].Status = condition.Status
			s.Conditions[i].Reason = condition.Reason
			s.Conditions[i].Message = condition.Message

			return
		}
	}

	s.Conditions = append(s.Conditions, condition)
}

// ProjectRoleTemplate is used by Organization Owners to manage the same Role across multiple Projects in Bulward.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.metadata.displayName"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ProjectRoleTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectRoleTemplateSpec   `json:"spec,omitempty"`
	Status ProjectRoleTemplateStatus `json:"status,omitempty"`
}

// IsReady returns if the ProjectRoleTemplate is ready.
func (s *ProjectRoleTemplate) IsReady() bool {
	if !s.DeletionTimestamp.IsZero() {
		return false
	}

	if s.Generation != s.Status.ObservedGeneration {
		return false
	}

	for _, condition := range s.Status.Conditions {
		if condition.Type == ProjectRoleTemplateReady &&
			condition.Status == ConditionTrue {
			return true
		}
	}
	return false
}

func (s *ProjectRoleTemplate) HasBinding(bindTo BindingType) bool {
	for _, b := range s.Spec.BindTo {
		if b == bindTo {
			return true
		}
	}
	return false
}

// ProjectRoleTemplateList contains a list of ProjectRoleTemplate.
// +kubebuilder:object:root=true
type ProjectRoleTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectRoleTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectRoleTemplate{}, &ProjectRoleTemplateList{})
}
