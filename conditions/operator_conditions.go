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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeclock "k8s.io/apimachinery/pkg/util/clock"
)

var (
	// clock is used to set status condition timestamps.
	clock kubeclock.Clock = &kubeclock.RealClock{}
	// ErrNoOperatorCondition indicates that the operator condition CRD is nil
	ErrNoOperatorCondition = fmt.Errorf("operator Condition CRD is nil")
)

// TODO: verify from OLM if this will be the name of the environment variable
// which is set for the Condition resource owned by the operator.
const operatorCondEnvVar = "OPERATOR_CONDITION_NAME"

func GetNamespacedName() (*types.NamespacedName, error) {
	conditionName := os.Getenv(operatorCondEnvVar)
	if conditionName == "" {
		return nil, fmt.Errorf("required env %s not set, cannot find the operator condition for the operator", operatorCondEnvVar)
	}
	return nil, nil
}

func SetConditionStatus(operatorCondition *api.OperatorCondition, newCond metav1.Condition) error {
	if operatorCondition == nil {
		return ErrNoOperatorCondition
	}

	meta.SetStatusCondition(&operatorCondition.Status.Conditions, newCond)
	return nil
}

func RemoveConditionStatus(operatorCondition *api.OperatorCondition, conditionType string) error {
	if operatorCondition == nil {
		return ErrNoOperatorCondition
	}

	meta.RemoveStatusCondition(&operatorCondition.Status.Conditions, conditionType)
	return nil

}

func FindConditionStatus(operatorCondition *api.OperatorCondition, conditionType string) (*metav1.Condition, error) {
	if operatorCondition == nil {
		return nil, ErrNoOperatorCondition
	}

	con := meta.FindStatusCondition(operatorCondition.Status.Conditions, conditionType)

	if con == nil {
		return nil, fmt.Errorf("conditionType %s not found", conditionType)
	}
	return con, nil
}

func IsConditionStatusTrue(operatorCondition *api.OperatorCondition, conditionType string) (bool, error) {
	return IsConditionStatusPresentAndEqual(operatorCondition, conditionType, metav1.ConditionTrue)
}

func IsConditionStatusFalse(operatorCondition *api.OperatorCondition, conditionType string) (bool, error) {
	return IsConditionStatusPresentAndEqual(operatorCondition, conditionType, metav1.ConditionFalse)
}

func IsConditionStatusUnknown(operatorCondition *api.OperatorCondition, conditionType string) (bool, error) {
	return IsConditionStatusPresentAndEqual(operatorCondition, conditionType, metav1.ConditionUnknown)
}

func IsConditionStatusPresentAndEqual(operatorCondition *api.OperatorCondition, conditionType string, conditionStatus metav1.ConditionStatus) (bool, error) {
	c, err := FindConditionStatus(operatorCondition, conditionType)
	if err != nil {
		return false, err
	}

	if c.Status == conditionStatus {
		return true, nil
	}
	return false, nil
}
