// Copyright 2020 The Operator-SDK Authors
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

package conditions

import (
	"fmt"
	"os"

	api "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/operator-lib/internal/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	// ErrNoOperatorCondition indicates that the operator condition CRD is nil
	ErrNoOperatorCondition = fmt.Errorf("operator Condition CRD is nil")
)

const (
	// operatorCondEnvVar is the env variable which
	// contains the name of the Condition CR associated to the operator,
	// set by OLM.
	operatorCondEnvVar = "OPERATOR_CONDITION_NAME"
)

var readNamespace = utils.GetOperatorNamespace

// GetNamespacedName returns the NamespacedName of the CR. It returns an error
// when the name of the CR cannot be found from the environment variable set by
// OLM. Hence, GetNamespacedName() can provide the NamespacedName when the operator
// is running on cluster and is being managed by OLM. If running locally, operator
// writers are encouraged to skip this method or gracefully handle the errors by logging
// a message.
func GetNamespacedName() (*types.NamespacedName, error) {
	conditionName := os.Getenv(operatorCondEnvVar)
	if conditionName == "" {
		return nil, fmt.Errorf("could not determine operator condition name: environment variable %s not set", operatorCondEnvVar)
	}
	operatorNs, err := readNamespace()
	if err != nil {
		return nil, fmt.Errorf("could not determine operator namespace: %v", err)
	}
	return &types.NamespacedName{Name: conditionName, Namespace: operatorNs}, nil
}

// SetOperatorCondition adds the specific condition to the Condition CR or
// updates the provided status of the condition if already present.
func SetOperatorCondition(operatorCondition *api.OperatorCondition, newCond metav1.Condition) error {
	if operatorCondition == nil {
		return ErrNoOperatorCondition
	}

	meta.SetStatusCondition(&operatorCondition.Status.Conditions, newCond)
	return nil
}

// RemoveOperatorCondition removes the specific condition present in Condition CR.
func RemoveOperatorCondition(operatorCondition *api.OperatorCondition, conditionType string) error {
	if operatorCondition == nil {
		return ErrNoOperatorCondition
	}
	meta.RemoveStatusCondition(&operatorCondition.Status.Conditions, conditionType)
	return nil
}

// FindOperatorCondition returns the specific condition present in the Condition CR.
func FindOperatorCondition(operatorCondition *api.OperatorCondition, conditionType string) (*metav1.Condition, error) {
	if operatorCondition == nil {
		return nil, ErrNoOperatorCondition
	}

	con := meta.FindStatusCondition(operatorCondition.Status.Conditions, conditionType)

	if con == nil {
		return nil, fmt.Errorf("conditionType %s not found", conditionType)
	}
	return con, nil
}

// IsConditionStatusTrue returns true when the condition is present in "True" state in the CR.
func IsConditionStatusTrue(operatorCondition *api.OperatorCondition, conditionType string) (bool, error) {
	return IsConditionStatusPresentAndEqual(operatorCondition, conditionType, metav1.ConditionTrue)
}

// IsConditionStatusFalse returns true when the condition is present in "False" state in the CR.
func IsConditionStatusFalse(operatorCondition *api.OperatorCondition, conditionType string) (bool, error) {
	return IsConditionStatusPresentAndEqual(operatorCondition, conditionType, metav1.ConditionFalse)
}

// IsConditionStatusUnknown returns true when the condition is present in "Unknown" state in the CR.
func IsConditionStatusUnknown(operatorCondition *api.OperatorCondition, conditionType string) (bool, error) {
	return IsConditionStatusPresentAndEqual(operatorCondition, conditionType, metav1.ConditionUnknown)
}

// IsConditionStatusPresentAndEqual returns true when the condition is present in the CR and is in the
// specified state.
func IsConditionStatusPresentAndEqual(operatorCondition *api.OperatorCondition, conditionType string, conditionStatus metav1.ConditionStatus) (bool, error) {
	c, err := FindOperatorCondition(operatorCondition, conditionType)
	if err != nil {
		return false, err
	}

	if c.Status == conditionStatus {
		return true, nil
	}
	return false, nil
}
