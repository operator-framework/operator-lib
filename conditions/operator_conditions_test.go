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
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	api "github.com/operator-framework/api/pkg/operators/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclock "k8s.io/apimachinery/pkg/util/clock"
)

const (
	conditionFoo = "Foo"
	conditionBar = "Bar"
)

var (
	// clock is used to set timestamps for condition status.
	clock kubeclock.Clock = &kubeclock.RealClock{}
)

var _ = Describe("Conditions helpers", func() {
	var (
		operatorCondition *api.OperatorCondition
		transitionTime    metav1.Time = metav1.Time{Time: clock.Now()}
	)

	BeforeEach(func() {
		operatorCondition = &api.OperatorCondition{
			Status: api.OperatorConditionStatus{
				Conditions: []metav1.Condition{
					{
						Type:               conditionFoo,
						Status:             metav1.ConditionTrue,
						Reason:             "foo",
						Message:            "The operator is in foo condition",
						LastTransitionTime: transitionTime,
					},
				},
			},
		}
	})
	Describe("SetOperatorCondition", func() {
		It("should set condition status", func() {
			newCond := metav1.Condition{
				Type:               conditionFoo,
				Status:             metav1.ConditionFalse,
				Reason:             "foo",
				Message:            "The operator is not in foo condition",
				LastTransitionTime: metav1.Time{Time: clock.Now()},
			}
			Expect(SetOperatorCondition(operatorCondition, newCond)).NotTo(HaveOccurred())
			Expect(operatorCondition.Status.Conditions[0].Status).To(BeEquivalentTo(metav1.ConditionFalse))
		})

		It("should add condition status", func() {
			newCond := metav1.Condition{
				Type:               conditionBar,
				Status:             metav1.ConditionTrue,
				Reason:             "bar",
				Message:            "The operator is in bar condition",
				LastTransitionTime: metav1.Time{Time: clock.Now()},
			}
			Expect(SetOperatorCondition(operatorCondition, newCond)).NotTo(HaveOccurred())
			Expect(len(operatorCondition.Status.Conditions)).To(BeEquivalentTo(2))
			Expect(isConditionPresent(operatorCondition.Status.Conditions, newCond)).To(BeTrue())
		})

		It("should preserve the condition if already present", func() {
			newCond := metav1.Condition{
				Type:               conditionFoo,
				Status:             metav1.ConditionTrue,
				Reason:             "foo",
				Message:            "The operator is in foo condition",
				LastTransitionTime: transitionTime,
			}
			Expect(SetOperatorCondition(operatorCondition, newCond)).NotTo(HaveOccurred())
			Expect(operatorCondition.Status.Conditions[0].Status).To(BeEquivalentTo(metav1.ConditionTrue))
		})

		It("should error when operatorCondition is nil", func() {
			newCond := metav1.Condition{
				Type:               conditionFoo,
				Status:             metav1.ConditionTrue,
				Reason:             "foo",
				Message:            "The operator is in foo condition",
				LastTransitionTime: transitionTime,
			}
			err := SetOperatorCondition(nil, newCond)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(ErrNoOperatorCondition))
		})
	})

	Describe("RemoveOperatorCondition", func() {
		var (
			rmvConditionType string
		)
		It("should remove the condition", func() {
			rmvConditionType = conditionFoo
			Expect(RemoveOperatorCondition(operatorCondition, rmvConditionType)).NotTo(HaveOccurred())
			Expect(len(operatorCondition.Status.Conditions)).To(BeEquivalentTo(0))
		})
		It("should not error when condition to be removed is not available", func() {
			rmvConditionType = conditionBar
			Expect(RemoveOperatorCondition(operatorCondition, rmvConditionType)).NotTo(HaveOccurred())
			Expect(len(operatorCondition.Status.Conditions)).To(BeEquivalentTo(1))
		})
		It("should error when operatorCondition is nil", func() {
			rmvConditionType = conditionFoo
			err := RemoveOperatorCondition(nil, rmvConditionType)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(ErrNoOperatorCondition))
		})
	})

	Describe("FindOperatorCondition", func() {
		var (
			findConditionType string
		)

		It("should return the condition if it exists", func() {
			findConditionType = conditionFoo
			conditionToFind := &metav1.Condition{
				Type:               conditionFoo,
				Status:             metav1.ConditionTrue,
				Reason:             "foo",
				Message:            "The operator is in foo condition",
				LastTransitionTime: transitionTime,
			}
			c, err := FindOperatorCondition(operatorCondition, findConditionType)
			Expect(err).NotTo(HaveOccurred())
			Expect(reflect.DeepEqual(c, conditionToFind)).To(BeTrue())
		})
		It("should return error when condition does not exist", func() {
			findConditionType = conditionBar
			c, err := FindOperatorCondition(operatorCondition, findConditionType)
			Expect(err).To(HaveOccurred())
			Expect(c).To(BeNil())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s not found", findConditionType)))
		})
		It("should error when operatorCondition is nil", func() {
			findConditionType = conditionFoo
			c, err := FindOperatorCondition(nil, findConditionType)
			Expect(err).To(HaveOccurred())
			Expect(c).To(BeNil())
			Expect(err).Should(MatchError(ErrNoOperatorCondition))
		})
	})

	Describe("Verfiy status of the condition", func() {
		var (
			conditionStatusFor string
		)

		It("should return correct value when condition exists", func() {
			conditionStatusFor = conditionFoo

			// IsConditionStatusTrue should return true
			val, err := IsConditionStatusTrue(operatorCondition, conditionStatusFor)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeTrue())

			// IsConditionStatusFalse should return false
			val, err = IsConditionStatusFalse(operatorCondition, conditionStatusFor)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeFalse())

			// IsConditionStatusUnknown should return false
			val, err = IsConditionStatusUnknown(operatorCondition, conditionStatusFor)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeFalse())
		})

		It("should return false if condition status is not set to true", func() {
			conditionStatusFor = conditionFoo
			operatorCondition.Status.Conditions[0].Status = metav1.ConditionFalse

			// IsConditionStatusTrue should return false
			val, err := IsConditionStatusTrue(operatorCondition, conditionStatusFor)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeFalse())

			// IsConditionStatusFalse should return true
			val, err = IsConditionStatusFalse(operatorCondition, conditionStatusFor)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeTrue())

			// IsConditionStatusUnknown should return false
			val, err = IsConditionStatusUnknown(operatorCondition, conditionStatusFor)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeFalse())
		})
		It("should return false if condition status is unknown", func() {
			conditionStatusFor = conditionFoo
			operatorCondition.Status.Conditions[0].Status = metav1.ConditionUnknown

			// IsConditionStatusTrue should return false
			val, err := IsConditionStatusTrue(operatorCondition, conditionStatusFor)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeFalse())

			// IsConditionStatusFalse should return false
			val, err = IsConditionStatusFalse(operatorCondition, conditionStatusFor)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeFalse())

			// IsConditionStatusUnknown should return true
			val, err = IsConditionStatusUnknown(operatorCondition, conditionStatusFor)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeTrue())

		})
		It("should error when condition cannot be found", func() {
			conditionStatusFor = conditionBar

			// IsConditionStatusTrue should return error
			val, err := IsConditionStatusTrue(operatorCondition, conditionStatusFor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s not found", conditionStatusFor)))
			Expect(val).To(BeFalse())

			// IsConditionStatusFalse should return error
			val, err = IsConditionStatusFalse(operatorCondition, conditionStatusFor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s not found", conditionStatusFor)))
			Expect(val).To(BeFalse())

			// IsConditionStatusUnknown should return error
			val, err = IsConditionStatusUnknown(operatorCondition, conditionStatusFor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s not found", conditionStatusFor)))
			Expect(val).To(BeFalse())
		})
	})

	Describe("IsConditionStatusPresentAndEqual", func() {
		var (
			conditionType   string
			conditionStatus metav1.ConditionStatus
		)

		It("should return true when condition is in the specified status", func() {
			conditionType = conditionFoo
			conditionStatus = metav1.ConditionTrue

			val, err := IsConditionStatusPresentAndEqual(operatorCondition, conditionType, conditionStatus)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeTrue())
		})

		It("should return false when condition is not present in the specified status", func() {
			conditionType = conditionFoo
			conditionStatus = metav1.ConditionUnknown

			val, err := IsConditionStatusPresentAndEqual(operatorCondition, conditionType, conditionStatus)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeFalse())
		})

		It("should return error when condition is not present", func() {
			conditionType = conditionBar
			conditionStatus = metav1.ConditionTrue

			val, err := IsConditionStatusPresentAndEqual(operatorCondition, conditionType, conditionStatus)
			Expect(err).To(HaveOccurred())
			Expect(val).To(BeFalse())
		})

	})

	Describe("GetNamespacedName", func() {
		It("should error when name of the operator condition cannot be found", func() {
			err := os.Unsetenv(operatorCondEnvVar)
			Expect(err).NotTo(HaveOccurred())

			objKey, err := GetNamespacedName()
			Expect(err).To(HaveOccurred())
			Expect(objKey).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("cannot find operator condition CR for the operator"))
		})

		It("should error when object namespace cannot be found", func() {
			err := os.Setenv(operatorCondEnvVar, "test")
			Expect(err).NotTo(HaveOccurred())

			readNamespace = func() ([]byte, error) {
				return nil, os.ErrNotExist
			}

			objKey, err := GetNamespacedName()
			Expect(err).To(HaveOccurred())
			Expect(objKey).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("cannot find namespace"))
		})

		It("should return the right namespaced name", func() {
			err := os.Setenv(operatorCondEnvVar, "test")
			Expect(err).NotTo(HaveOccurred())

			readNamespace = func() ([]byte, error) {
				return []byte("testns"), nil
			}
			objKey, err := GetNamespacedName()
			Expect(err).NotTo(HaveOccurred())
			Expect(objKey).NotTo(BeNil())
			Expect(objKey.Name).To(BeEquivalentTo("test"))
			Expect(objKey.Namespace).To(BeEquivalentTo("testns"))
		})
	})

})

func isConditionPresent(arr []metav1.Condition, con metav1.Condition) bool {
	for _, c := range arr {
		if c.Type == con.Type {
			return true
		}
	}
	return false
}
