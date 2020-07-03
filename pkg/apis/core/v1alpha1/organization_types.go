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

// OrganizationSpec describes the desired state of Organization.
type OrganizationSpec struct {
	// Metadata	contains additional human readable Organization details.
	Metadata *OrganizationMetadata `json:"metadata,omitempty"`
}

// OrganizationMetadata contains the metadata of the Organization.
type OrganizationMetadata struct {
	// DisplayName is the human-readable name of this Organization.
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`
	// Description is the long and detailed description of the Organization.
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`
}

// OrganizationStatus represents the observed state of Organization.
type OrganizationStatus struct {
	// NamespaceName is the name of the Namespace that the Organization manages.
	Namespace *ObjectReference `json:"namespace,omitempty"`
	// ObservedGeneration is the most recent generation observed for this Organization by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represents the latest available observations of a Organization's current state.
	Conditions []OrganizationCondition `json:"conditions,omitempty"`
	// DEPRECATED.
	// Phase represents the current lifecycle state of this object.
	// Consider this field DEPRECATED, it will be removed as soon as there
	// is a mechanism to map conditions to strings when printing the property.
	// This is only for display purpose, for everything else use conditions.
	Phase OrganizationPhaseType `json:"phase,omitempty"`

	// Members enumerate all rbacv1.Subject mentioned in the Organization RoleBinding's
	Members []rbacv1.Subject `json:"members,omitempty"`
}

// OrganizationPhaseType represents all conditions as a single string for printing by using kubectl commands.
// +kubebuilder:validation:Ready;NotReady;Unknown;Terminating
type OrganizationPhaseType string

// Values of OrganizationPhaseType.
const (
	OrganizationPhaseReady       OrganizationPhaseType = "Ready"
	OrganizationPhaseNotReady    OrganizationPhaseType = "NotReady"
	OrganizationPhaseUnknown     OrganizationPhaseType = "Unknown"
	OrganizationPhaseTerminating OrganizationPhaseType = "Terminating"
)

const (
	OrganizationTerminatingReason = "Deleting"
)

// updatePhase updates the phase property based on the current conditions.
// this method should be called every time the conditions are updated.
func (s *OrganizationStatus) updatePhase() {
	for _, condition := range s.Conditions {
		if condition.Type != OrganizationReady {
			continue
		}

		switch condition.Status {
		case ConditionTrue:
			s.Phase = OrganizationPhaseReady
		case ConditionFalse:
			if condition.Reason == OrganizationTerminatingReason {
				s.Phase = OrganizationPhaseTerminating
			} else {
				s.Phase = OrganizationPhaseNotReady
			}
		case ConditionUnknown:
			s.Phase = OrganizationPhaseUnknown
		}
		return
	}

	s.Phase = OrganizationPhaseUnknown
}

// OrganizationConditionType represents a OrganizationCondition value.
// +kubebuilder:validation:Ready
type OrganizationConditionType string

const (
	// OrganizationReady represents a Organization condition is in ready state.
	OrganizationReady OrganizationConditionType = "Ready"
)

// OrganizationCondition contains details for the current condition of this Organization.
type OrganizationCondition struct {
	// Type is the type of the Organization condition, currently ('Ready').
	Type OrganizationConditionType `json:"type"`
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
func (s *OrganizationStatus) GetCondition(t OrganizationConditionType) (condition OrganizationCondition, exists bool) {
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
func (s *OrganizationStatus) SetCondition(condition OrganizationCondition) {
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

// Organization is internal representation for Organization in Bulward.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Organization Namespace",type="string",JSONPath=".status.namespace.name"
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.metadata.displayName"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,shortName=iorg
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec,omitempty"`
	Status OrganizationStatus `json:"status,omitempty"`
}

// IsReady returns if the Organization is ready.
func (s *Organization) IsReady() bool {
	if !s.DeletionTimestamp.IsZero() {
		return false
	}

	if s.Generation != s.Status.ObservedGeneration {
		return false
	}

	for _, condition := range s.Status.Conditions {
		if condition.Type == OrganizationReady &&
			condition.Status == ConditionTrue {
			return true
		}
	}
	return false
}

// OrganizationList contains a list of Organization.
// +kubebuilder:object:root=true
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
