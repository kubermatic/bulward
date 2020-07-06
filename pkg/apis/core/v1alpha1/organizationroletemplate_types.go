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

// OrganizationRoleTemplateSpec describes the desired state of OrganizationRoleTemplate.
type OrganizationRoleTemplateSpec struct {
	// Metadata	contains additional human readable OrganizationRoleTemplate details.
	Metadata *OrganizationRoleTemplateMetadata `json:"metadata,omitempty"`
	// Scopes defines the scopes of this OrganizationRoleTemplate.
	// +kubebuilder:validation:MinItems=1
	Scopes []RoleTemplateScope `json:"scopes"`
	// BindTo defines the member types of the Organization that this OrganizationRoleTemplate will be bound to.
	BindTo []BindingType `json:"bindTo,omitempty"`
	// Rules defnies the Role that this OrganizationRoleTemplate refers to.
	Rules []rbacv1.PolicyRule `json:"rules"`
}

// +kubebuilder:validation:Enum=Organization;Project
type RoleTemplateScope string

const (
	RoleTemplateScopeOrganization RoleTemplateScope = "Organization"
	RoleTemplateScopeProject      RoleTemplateScope = "Project"
)

// +kubebuilder:validation:Enum=Owners;Everyone
type BindingType string

const (
	BindToOwners   BindingType = "Owners"
	BindToEveryone BindingType = "Everyone"
)

// OrganizationRoleTemplateMetadata contains the metadata of the OrganizationRoleTemplate.
type OrganizationRoleTemplateMetadata struct {
	// DisplayName is the human-readable name of this OrganizationRoleTemplate.
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`
	// Description is the long and detailed description of the OrganizationRoleTemplate.
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`
}

// OrganizationRoleTemplateStatus represents the observed state of OrganizationRoleTemplate.
type OrganizationRoleTemplateStatus struct {
	// ObservedGeneration is the most recent generation observed for this OrganizationRoleTemplate by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represents the latest available observations of a OrganizationRoleTemplate's current state.
	Conditions []OrganizationRoleTemplateCondition `json:"conditions,omitempty"`
	// DEPRECATED.
	// Phase represents the current lifecycle state of this object.
	// Consider this field DEPRECATED, it will be removed as soon as there
	// is a mechanism to map conditions to strings when printing the property.
	// This is only for display purpose, for everything else use conditions.
	Phase OrganizationRoleTemplatePhaseType `json:"phase,omitempty"`
	// Targets holds different targets(Organization, Project) that this OrganizationRoleTemplate targets to.
	Targets []OrganizationRoleTemplateTarget `json:"targets,omitempty"`
}

type OrganizationRoleTemplateTarget struct {
	// Kind of target being referenced. Available values can be "Organization", "Project".
	// +kubebuilder:validation:Enum=Organization;Project
	Kind string `json:"kind"`
	// APIGroup holds the API group of the referenced target, default "bulward.io".
	// +kubebuilder:default=bulward.io
	APIGroup string `json:"apiGroup,omitempty"`
	// Name of the target being referenced.
	Name string `json:"name"`
	// ObservedGeneration is the most recent generation observed for this Target by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// OrganizationRoleTemplatePhaseType represents all conditions as a single string for printing by using kubectl commands.
// +kubebuilder:validation:Ready;NotReady;Unknown;Terminating
type OrganizationRoleTemplatePhaseType string

// Values of OrganizationRoleTemplatePhaseType.
const (
	OrganizationRoleTemplatePhaseReady       OrganizationRoleTemplatePhaseType = "Ready"
	OrganizationRoleTemplatePhaseNotReady    OrganizationRoleTemplatePhaseType = "NotReady"
	OrganizationRoleTemplatePhaseUnknown     OrganizationRoleTemplatePhaseType = "Unknown"
	OrganizationRoleTemplatePhaseTerminating OrganizationRoleTemplatePhaseType = "Terminating"
)

const (
	OrganizationRoleTemplateTerminatingReason = "Deleting"
)

// updatePhase updates the phase property based on the current conditions.
// this method should be called every time the conditions are updated.
func (s *OrganizationRoleTemplateStatus) updatePhase() {
	for _, condition := range s.Conditions {
		if condition.Type != OrganizationRoleTemplateReady {
			continue
		}

		switch condition.Status {
		case ConditionTrue:
			s.Phase = OrganizationRoleTemplatePhaseReady
		case ConditionFalse:
			if condition.Reason == OrganizationRoleTemplateTerminatingReason {
				s.Phase = OrganizationRoleTemplatePhaseTerminating
			} else {
				s.Phase = OrganizationRoleTemplatePhaseNotReady
			}
		case ConditionUnknown:
			s.Phase = OrganizationRoleTemplatePhaseUnknown
		}
		return
	}

	s.Phase = OrganizationRoleTemplatePhaseUnknown
}

// OrganizationRoleTemplateConditionType represents a OrganizationRoleTemplateCondition value.
// +kubebuilder:validation:Ready
type OrganizationRoleTemplateConditionType string

const (
	// OrganizationRoleTemplateReady represents a OrganizationRoleTemplate condition is in ready state.
	OrganizationRoleTemplateReady OrganizationRoleTemplateConditionType = "Ready"
)

// OrganizationRoleTemplateCondition contains details for the current condition of this OrganizationRoleTemplate.
type OrganizationRoleTemplateCondition struct {
	// Type is the type of the OrganizationRoleTemplate condition, currently ('Ready').
	Type OrganizationRoleTemplateConditionType `json:"type"`
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
func (s *OrganizationRoleTemplateStatus) GetCondition(t OrganizationRoleTemplateConditionType) (condition OrganizationRoleTemplateCondition, exists bool) {
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
func (s *OrganizationRoleTemplateStatus) SetCondition(condition OrganizationRoleTemplateCondition) {
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

// OrganizationRoleTemplate is internal representation for OrganizationRoleTemplate in Bulward.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.metadata.displayName"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster
type OrganizationRoleTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationRoleTemplateSpec   `json:"spec,omitempty"`
	Status OrganizationRoleTemplateStatus `json:"status,omitempty"`
}

// IsReady returns if the OrganizationRoleTemplate is ready.
func (s *OrganizationRoleTemplate) IsReady() bool {
	if !s.DeletionTimestamp.IsZero() {
		return false
	}

	if s.Generation != s.Status.ObservedGeneration {
		return false
	}

	for _, condition := range s.Status.Conditions {
		if condition.Type == OrganizationRoleTemplateReady &&
			condition.Status == ConditionTrue {
			return true
		}
	}
	return false
}

func (s *OrganizationRoleTemplate) HasScope(organizationRoleScope RoleTemplateScope) bool {
	for _, scope := range s.Spec.Scopes {
		if scope == organizationRoleScope {
			return true
		}
	}
	return false
}

func (s *OrganizationRoleTemplate) HasBinding(bindTo BindingType) bool {
	for _, b := range s.Spec.BindTo {
		if b == bindTo {
			return true
		}
	}
	return false
}

// OrganizationRoleTemplateList contains a list of OrganizationRoleTemplate.
// +kubebuilder:object:root=true
type OrganizationRoleTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationRoleTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OrganizationRoleTemplate{}, &OrganizationRoleTemplateList{})
}
