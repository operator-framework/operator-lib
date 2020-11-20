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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	api "github.com/operator-framework/api/pkg/operators/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	// ErrNoOperatorCondition indicates that the operator condition CRD is nil
	ErrNoOperatorCondition = fmt.Errorf("operator Condition CRD is nil")
)

// TODO: verify from OLM if this will be the name of the environment variable
// which is set for the Condition resource owned by the operator.
const operatorCondEnvVar = "OPERATOR_CONDITION_NAME"

var readNamespace = func() ([]byte, error) {
	return ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
}

// GetNamespacedName returns the NamespacedName of the CR. It returns an error
// when the name of the CR cannot be found from the environment variable set by
// OLM, or when the namespace cannot be found from the associated service account
// secret.
func GetNamespacedName() (*types.NamespacedName, error) {
	conditionName := os.Getenv(operatorCondEnvVar)
	if conditionName == "" {
		return nil, fmt.Errorf("required env %s not set, cannot find operator condition CR for the operator", operatorCondEnvVar)
	}
	operatorNs, err := getOperatorNamespace()
	if err != nil {
		return nil, err
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

// getOperatorNamespace returns the namespace the operator should be running in.
func getOperatorNamespace() (string, error) {
	nsBytes, err := readNamespace()
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("cannot find namespace of the operator")
		}
		return "", err
	}
	ns := strings.TrimSpace(string(nsBytes))
	return ns, nil
}
