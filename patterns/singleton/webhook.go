// Copyright 2021 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package singleton

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConstraintViolationError is returned when the singleton constraint is violated cluster-wide.
type ConstraintViolationError struct {
	schema.GroupVersionKind
	// ExpectedName is the expected name of the singleton object.
	ExpectedName string
	// ViolatingName is the violating object's name, where ViolatingName != ExpectedName.
	ViolatingName string
}

func (e ConstraintViolationError) Error() string {
	return fmt.Sprintf("singleton constraint violated: only one %s with name %q may exist, have %q",
		e.GroupVersionKind, e.ExpectedName, e.ViolatingName)
}

// NewConstraintViolation returns a ConstraintViolationError for obj and expectedName.
func NewConstraintViolation(obj client.Object, expectedName string) error {
	return ConstraintViolationError{
		GroupVersionKind: obj.GetObjectKind().GroupVersionKind(),
		ExpectedName:     expectedName,
		ViolatingName:    obj.GetName(),
	}
}

// ValidateCreate returns an error if new's name != expectedName.
// Call this function within a webhook.Validator.ValidateCreate() method.
// If new's underlying type is external, you can call this function from admission.Handler for that type.
func ValidateCreate(new client.Object, expectedName string) error {
	if new.GetName() != expectedName {
		return NewConstraintViolation(new, expectedName)
	}
	return nil
}

// ValidateDelete logs a debug message if obj's name == expectedName.
// Call this function within a webhook.Validator.ValidateDelete() method.
// If obj's underlying type is external, you can call this function from admission.Handler for that type.
func ValidateDelete(obj client.Object, expectedName string) error {
	if obj.GetName() == expectedName {
		ctrl.Log.WithName("singleton-webhook").V(1).Info("singleton object being deleted",
			"Name", expectedName,
			"GroupVersionKind", obj.GetObjectKind().GroupVersionKind())
	}
	return nil
}
