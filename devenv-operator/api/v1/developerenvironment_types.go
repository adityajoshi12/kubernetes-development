/*
Copyright 2024.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeveloperEnvironmentSpec defines the desired state of DeveloperEnvironment
type DeveloperEnvironmentSpec struct {
	// Language and framework configuration
	// +kubebuilder:validation:Enum=nodejs;go;python;java;rust;
	Language string `json:"language"`
	Version  string `json:"version"`

	// Development tools and IDE
	IDE IDEConfig `json:"ide,omitempty"`

	// Database configuration
	Database DatabaseSpec `json:"database,omitempty"`

	// Additional dependencies
	Dependencies []DependencySpec `json:"dependencies,omitempty"`
}

// IDEConfig defines IDE and development tool settings
type IDEConfig struct {
	Type           string            `json:"type"`
	Extensions     []string          `json:"extensions,omitempty"`
	Settings       map[string]string `json:"settings,omitempty"`
	PasswordSecret string            `json:"passwordSecret,omitempty"`
}

// DatabaseSpec defines database configuration
type DatabaseSpec struct {
	// +kubebuilder:validation:Enum=postgres;redis;
	Type string `json:"type"`
	// +kubebuilder:default=latest
	Version string `json:"version"`
}

// DependencySpec defines additional tool dependencies
type DependencySpec struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// DeveloperEnvironmentStatus defines the observed state of DeveloperEnvironment
type DeveloperEnvironmentStatus struct {
	Phase       string      `json:"phase"`
	Conditions  []Condition `json:"conditions,omitempty"`
	AccessURL   string      `json:"accessURL,omitempty"`
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// Condition contains details for the current condition of the DevEnv
type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DeveloperEnvironment is the Schema for the developerenvironments API
type DeveloperEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeveloperEnvironmentSpec   `json:"spec,omitempty"`
	Status DeveloperEnvironmentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeveloperEnvironmentList contains a list of DeveloperEnvironment
type DeveloperEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeveloperEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeveloperEnvironment{}, &DeveloperEnvironmentList{})
}
