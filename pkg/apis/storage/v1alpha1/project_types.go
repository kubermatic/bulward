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

// Project is internal representation for Project in Bulward.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project Namespace",type="string",JSONPath=".status.namespace.name"
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".metadata.name"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,shortName=iprj
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// IsReady returns if the Project is ready.
func (s *Project) IsReady() bool {
	if !s.DeletionTimestamp.IsZero() {
		return false
	}

	if s.Generation != s.Status.ObservedGeneration {
		return false
	}

	for _, condition := range s.Status.Conditions {
		if condition.Type == ProjectReady &&
			condition.Status == ConditionTrue {
			return true
		}
	}
	return false
}

// ProjectList contains a list of Projects.
// +kubebuilder:object:root=true
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

// ProjectSpec describes the desired state of Project.
type ProjectSpec struct {
	// Owners holds the RBAC subjects that represent the owners of this project.
	// +kubebuilder:validation:MinItems=1
	Owners []rbacv1.Subject `json:"owners"`
}

// ProjectStatus describes the observed state of Project.
type ProjectStatus struct {
	// NamespaceName is the name of the Namespace that the Project manages.
	Namespace *ObjectReference `json:"namespace,omitempty"`
	// ObservedGeneration is the most recent generation observed for this Project by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represents the latest available observations of a Project's current state.
	Conditions []ProjectCondition `json:"conditions,omitempty"`
	// DEPRECATED.
	// Phase represents the current lifecycle state of this object.
	// Consider this field DEPRECATED, it will be removed as soon as there
	// is a mechanism to map conditions to strings when printing the property.
	// This is only for display purpose, for everything else use conditions.
	Phase ProjectPhaseType `json:"phase,omitempty"`

	// Members enumerate all rbacv1.Subject mentioned in the Project's RoleBinding's
	Members []rbacv1.Subject `json:"members,omitempty"`
}

// ProjectPhaseType represents all conditions as a single string for printing by using kubectl commands.
// +kubebuilder:validation:Ready;NotReady;Unknown;Terminating
type ProjectPhaseType string

// Values of ProjectPhaseType.
const (
	ProjectPhaseReady       ProjectPhaseType = "Ready"
	ProjectPhaseNotReady    ProjectPhaseType = "NotReady"
	ProjectPhaseUnknown     ProjectPhaseType = "Unknown"
	ProjectPhaseTerminating ProjectPhaseType = "Terminating"
)

const (
	ProjectTerminatingReason = "Deleting"
)

// updatePhase updates the phase property based on the current conditions.
// this method should be called every time the conditions are updated.
func (s *ProjectStatus) updatePhase() {
	for _, condition := range s.Conditions {
		if condition.Type != ProjectReady {
			continue
		}

		switch condition.Status {
		case ConditionTrue:
			s.Phase = ProjectPhaseReady
		case ConditionFalse:
			if condition.Reason == ProjectTerminatingReason {
				s.Phase = ProjectPhaseTerminating
			} else {
				s.Phase = ProjectPhaseNotReady
			}
		case ConditionUnknown:
			s.Phase = ProjectPhaseUnknown
		}
		return
	}

	s.Phase = ProjectPhaseUnknown
}

// ProjectConditionType represents a ProjectCondition value.
// +kubebuilder:validation:Ready
type ProjectConditionType string

const (
	// OProjectReady represents a Project condition is in ready state.
	ProjectReady ProjectConditionType = "Ready"
)

// ProjectCondition contains details for the current condition of this Project.
type ProjectCondition struct {
	// Type is the type of the Project condition, currently ('Ready').
	Type ProjectConditionType `json:"type"`
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
func (s *ProjectStatus) GetCondition(t ProjectConditionType) (condition ProjectCondition, exists bool) {
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
func (s *ProjectStatus) SetCondition(condition ProjectCondition) {
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

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
