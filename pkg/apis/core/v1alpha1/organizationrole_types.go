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

// OrganizationRoleSpec describes the desired state of OrganizationRole.
type OrganizationRoleSpec struct {
	// Rules defines the Role that this OrganizationRole refers to.
	Rules []rbacv1.PolicyRule `json:"rules"`
}

// OrganizationRoleStatus represents the observed state of OrganizationRole.
type OrganizationRoleStatus struct {
	// ObservedGeneration is the most recent generation observed for this OrganizationRole by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represents the latest available observations of a OrganizationRole's current state.
	Conditions []OrganizationRoleCondition `json:"conditions,omitempty"`
	// DEPRECATED.
	// Phase represents the current lifecycle state of this object.
	// Consider this field DEPRECATED, it will be removed as soon as there
	// is a mechanism to map conditions to strings when printing the property.
	// This is only for display purpose, for everything else use conditions.
	Phase OrganizationRolePhaseType `json:"phase,omitempty"`
	// AcceptedRules contains the rules that accepted by Bulward.
	AcceptedRules []rbacv1.PolicyRule `json:"acceptedRules,omitempty"`
}

// OrganizationRolePhaseType represents all conditions as a single string for printing by using kubectl commands.
// +kubebuilder:validation:Ready;NotReady;Unknown;Terminating
type OrganizationRolePhaseType string

// Values of OrganizationRolePhaseType.
const (
	OrganizationRolePhaseReady       OrganizationRolePhaseType = "Ready"
	OrganizationRolePhaseNotReady    OrganizationRolePhaseType = "NotReady"
	OrganizationRolePhaseUnknown     OrganizationRolePhaseType = "Unknown"
	OrganizationRolePhaseTerminating OrganizationRolePhaseType = "Terminating"
)

const (
	OrganizationRoleTerminatingReason = "Deleting"
)

// updatePhase updates the phase property based on the current conditions.
// this method should be called every time the conditions are updated.
func (s *OrganizationRoleStatus) updatePhase() {
	for _, condition := range s.Conditions {
		if condition.Type != OrganizationRoleReady {
			continue
		}

		switch condition.Status {
		case ConditionTrue:
			s.Phase = OrganizationRolePhaseReady
		case ConditionFalse:
			if condition.Reason == OrganizationRoleTerminatingReason {
				s.Phase = OrganizationRolePhaseTerminating
			} else {
				s.Phase = OrganizationRolePhaseNotReady
			}
		case ConditionUnknown:
			s.Phase = OrganizationRolePhaseUnknown
		}
		return
	}

	s.Phase = OrganizationRolePhaseUnknown
}

// OrganizationRoleConditionType represents a OrganizationRoleCondition value.
// +kubebuilder:validation:Ready
type OrganizationRoleConditionType string

const (
	// OrganizationRoleReady represents a OrganizationRole condition is in ready state.
	OrganizationRoleReady OrganizationRoleConditionType = "Ready"
)

// OrganizationRoleCondition contains details for the current condition of this OrganizationRole.
type OrganizationRoleCondition struct {
	// Type is the type of the OrganizationRole condition, currently ('Ready').
	Type OrganizationRoleConditionType `json:"type"`
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
func (s *OrganizationRoleStatus) GetCondition(t OrganizationRoleConditionType) (condition OrganizationRoleCondition, exists bool) {
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
func (s *OrganizationRoleStatus) SetCondition(condition OrganizationRoleCondition) {
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

// OrganizationRole is internal representation for organization-scoped Role in Bulward.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.metadata.displayName"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type OrganizationRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationRoleSpec   `json:"spec,omitempty"`
	Status OrganizationRoleStatus `json:"status,omitempty"`
}

// IsReady returns if the OrganizationRole is ready.
func (s *OrganizationRole) IsReady() bool {
	if !s.DeletionTimestamp.IsZero() {
		return false
	}

	if s.Generation != s.Status.ObservedGeneration {
		return false
	}

	for _, condition := range s.Status.Conditions {
		if condition.Type == OrganizationRoleReady &&
			condition.Status == ConditionTrue {
			return true
		}
	}
	return false
}

// OrganizationRoleList contains a list of OrganizationRole.
// +kubebuilder:object:root=true
type OrganizationRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OrganizationRole{}, &OrganizationRoleList{})
}
