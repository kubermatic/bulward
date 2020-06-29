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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InternalOrganizationSpec describes the desired state of InternalOrganization.
type InternalOrganizationSpec struct {
	// Metadata	contains additional human readable internalOrganization details.
	Metadata *InternalOrganizationMetadata `json:"metadata,omitempty"`
}

// InternalOrganizationMetadata contains the metadata of the InternalOrganization.
type InternalOrganizationMetadata struct {
	// DisplayName is the human-readable name of this InternalOrganization.
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`
	// Description is the long and detailed description of the InternalOrganization.
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`
}

// InternalOrganizationStatus represents the observed state of InternalOrganization.
type InternalOrganizationStatus struct {
	// NamespaceName is the name of the Namespace that the InternalOrganization manages.
	Namespace *ObjectReference `json:"namespace,omitempty"`
	// ObservedGeneration is the most recent generation observed for this InternalOrganization by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represents the latest available observations of a InternalOrganization's current state.
	Conditions []InternalOrganizationCondition `json:"conditions,omitempty"`
	// DEPRECATED.
	// Phase represents the current lifecycle state of this object.
	// Consider this field DEPRECATED, it will be removed as soon as there
	// is a mechanism to map conditions to strings when printing the property.
	// This is only for display purpose, for everything else use conditions.
	Phase InternalOrganizationPhaseType `json:"phase,omitempty"`
}

// InternalOrganizationPhaseType represents all conditions as a single string for printing by using kubectl commands.
// +kubebuilder:validation:Ready;NotReady;Unknown;Terminating
type InternalOrganizationPhaseType string

// Values of InternalOrganizationPhaseType.
const (
	InternalOrganizationPhaseReady       InternalOrganizationPhaseType = "Ready"
	InternalOrganizationPhaseNotReady    InternalOrganizationPhaseType = "NotReady"
	InternalOrganizationPhaseUnknown     InternalOrganizationPhaseType = "Unknown"
	InternalOrganizationPhaseTerminating InternalOrganizationPhaseType = "Terminating"
)

const (
	InternalOrganizationTerminatingReason = "Deleting"
)

// updatePhase updates the phase property based on the current conditions.
// this method should be called every time the conditions are updated.
func (s *InternalOrganizationStatus) updatePhase() {
	for _, condition := range s.Conditions {
		if condition.Type != InternalOrganizationReady {
			continue
		}

		switch condition.Status {
		case ConditionTrue:
			s.Phase = InternalOrganizationPhaseReady
		case ConditionFalse:
			if condition.Reason == InternalOrganizationTerminatingReason {
				s.Phase = InternalOrganizationPhaseTerminating
			} else {
				s.Phase = InternalOrganizationPhaseNotReady
			}
		case ConditionUnknown:
			s.Phase = InternalOrganizationPhaseUnknown
		}
		return
	}

	s.Phase = InternalOrganizationPhaseUnknown
}

// InternalOrganizationConditionType represents a InternalOrganizationCondition value.
// +kubebuilder:validation:Ready
type InternalOrganizationConditionType string

const (
	// InternalOrganizationReady represents a InternalOrganization condition is in ready state.
	InternalOrganizationReady InternalOrganizationConditionType = "Ready"
)

// InternalOrganizationCondition contains details for the current condition of this InternalOrganization.
type InternalOrganizationCondition struct {
	// Type is the type of the InternalOrganization condition, currently ('Ready').
	Type InternalOrganizationConditionType `json:"type"`
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
func (s *InternalOrganizationStatus) GetCondition(t InternalOrganizationConditionType) (condition InternalOrganizationCondition, exists bool) {
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
func (s *InternalOrganizationStatus) SetCondition(condition InternalOrganizationCondition) {
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

// InternalOrganization is internal representation for Organization in Bulward.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="InternalOrganization Namespace",type="string",JSONPath=".status.namespace.name"
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.metadata.displayName"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,shortName=iorg
type InternalOrganization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InternalOrganizationSpec   `json:"spec,omitempty"`
	Status InternalOrganizationStatus `json:"status,omitempty"`
}

// IsReady returns if the InternalOrganization is ready.
func (s *InternalOrganization) IsReady() bool {
	if !s.DeletionTimestamp.IsZero() {
		return false
	}

	if s.Generation != s.Status.ObservedGeneration {
		return false
	}

	for _, condition := range s.Status.Conditions {
		if condition.Type == InternalOrganizationReady &&
			condition.Status == ConditionTrue {
			return true
		}
	}
	return false
}

// InternalOrganizationList contains a list of InternalOrganization.
// +kubebuilder:object:root=true
type InternalOrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InternalOrganization `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InternalOrganization{}, &InternalOrganizationList{})
}
