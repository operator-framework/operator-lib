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

package status

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclock "k8s.io/apimachinery/pkg/util/clock"
)

func TestStatus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Status Suite")
}

var (
	initTime      time.Time
	clockInterval time.Duration
)

func init() {
	loc, _ := time.LoadLocation("Local")
	initTime = time.Date(2015, time.July, 11, 0, 1, 0, 0, loc)
	clockInterval = time.Hour
}

func initConditions(init ...Condition) Conditions {
	// Use the same initial time for all initial conditions
	clock = kubeclock.NewFakeClock(initTime)
	conditions := Conditions{}
	for _, c := range init {
		conditions.SetCondition(c)
	}

	// Use an incrementing clock for the rest of the test
	clock = &kubeclock.IntervalClock{
		Time:     initTime,
		Duration: clockInterval,
	}

	return conditions
}

func generateCondition(t ConditionType, s corev1.ConditionStatus) Condition {
	c := Condition{
		Type:    t,
		Status:  s,
		Reason:  ConditionReason(fmt.Sprintf("My%s%s", t, s)),
		Message: fmt.Sprintf("Condition %s is %s", t, s),
	}
	return c
}

func withLastTransitionTime(c Condition, t time.Time) Condition {
	c.LastTransitionTime = metav1.Time{Time: t}
	return c
}

var _ = Describe("Condition", func() {

	Describe("DeepCopyInto", func() {

		var aCopy Condition

		a := generateCondition("A", corev1.ConditionTrue)

		It("should copy one condition to another in its entirety", func() {
			a.DeepCopyInto(&aCopy)
			Expect(aCopy.Type).To(Equal(a.Type))
			Expect(aCopy.Status).To(Equal(a.Status))
			Expect(aCopy.Reason).To(Equal(a.Reason))
			Expect(aCopy.Message).To(Equal(a.Message))
			Expect(aCopy.LastTransitionTime).To(Equal(a.LastTransitionTime))
		})
	})

	Describe("Checking for IsTrue, IsFalse and IsUnknown", func() {

		var temp Condition

		It(" should be True for IsTrue if present", func() {
			temp = generateCondition("Temp", corev1.ConditionTrue)
			Expect(temp.IsTrue()).Should(BeTrue())
		})
		It(" should be False for IsTrue if not present", func() {
			temp = generateCondition("Temp", corev1.ConditionFalse)
			Expect(temp.IsTrue()).Should(BeFalse())
		})
		It(" should be True for IsFalse not present", func() {
			temp = generateCondition("Temp", corev1.ConditionFalse)
			Expect(temp.IsFalse()).Should(BeTrue())
		})
		It(" should be False for IsFalse if present", func() {
			temp = generateCondition("Temp", corev1.ConditionTrue)
			Expect(temp.IsFalse()).Should(BeFalse())
		})
		It(" should be True for IsUnknown", func() {
			temp = generateCondition("Temp", corev1.ConditionUnknown)
			Expect(temp.IsUnknown()).Should(BeTrue())
		})
	})
})

var _ = Describe("Conditions", func() {

	Describe("TestConditions IsTrueFor, IsFalseFor and IsUnknownFor", func() {

		conditions := NewConditions(
			generateCondition("False", corev1.ConditionFalse),
			generateCondition("True", corev1.ConditionTrue),
			generateCondition("Unknown", corev1.ConditionUnknown),
		)

		Describe("IsTrueFor", func() {
			It(" should be true only for True", func() {
				Expect(conditions.IsTrueFor(ConditionType("True"))).Should(BeTrue())
				Expect(conditions.IsTrueFor(ConditionType("False"))).Should(BeFalse())
				Expect(conditions.IsTrueFor(ConditionType("Unknown"))).Should(BeFalse())
				Expect(conditions.IsTrueFor(ConditionType("DoesNotExist"))).Should(BeFalse())
			})
		})

		Describe("IsFalseFor", func() {

			It(" should be true only for False", func() {
				Expect(conditions.IsFalseFor(ConditionType("True"))).Should(BeFalse())
				Expect(conditions.IsFalseFor(ConditionType("False"))).Should(BeTrue())
				Expect(conditions.IsFalseFor(ConditionType("Unknown"))).Should(BeFalse())
				Expect(conditions.IsFalseFor(ConditionType("DoesNotExist"))).Should(BeFalse())
			})
		})

		Describe("IsUnknownFor", func() {

			It(" should be true for Unknown and DoesNotExist", func() {
				Expect(conditions.IsUnknownFor(ConditionType("True"))).Should(BeFalse())
				Expect(conditions.IsUnknownFor(ConditionType("False"))).Should(BeFalse())
				Expect(conditions.IsUnknownFor(ConditionType("Unknown"))).Should(BeTrue())
				Expect(conditions.IsUnknownFor(ConditionType("DoesNotExist"))).Should(BeTrue())
			})
		})
	})

	Describe("Unmarshall JSON", func() {

		a := generateCondition("A", corev1.ConditionTrue)
		b := generateCondition("B", corev1.ConditionTrue)
		c := generateCondition("C", corev1.ConditionTrue)
		d := generateCondition("D", corev1.ConditionTrue)

		// Insert conditions unsorted
		conditions := initConditions(b, d, c, a)

		data, err := json.Marshal(conditions)
		if err != nil {
			Fail(fmt.Sprintf("Failed to marshal JSON: %s", err))
		}

		// Test that conditions are in sorted order by type.
		in := []Condition{}
		err = json.Unmarshal(data, &in)
		if err != nil {
			Fail(fmt.Sprintf("Failed to unmarshal JSON: %s", err))
		}

		Describe("Condition A", func() {
			It(fmt.Sprintf(" should be equal to %+v", a.Type), func() {
				Expect(in[0].Type).To(Equal(a.Type))
			})
		})
		Describe("Condition B", func() {
			It(fmt.Sprintf(" should be equal to %+v", b.Type), func() {
				Expect(in[1].Type).To(Equal(b.Type))
			})
		})
		Describe("Condition C", func() {
			It(fmt.Sprintf(" should be equal to %+v", c.Type), func() {
				Expect(in[2].Type).To(Equal(c.Type))
			})
		})
		Describe("Condition D", func() {
			It(fmt.Sprintf(" should be equal to %+v", d.Type), func() {
				Expect(in[3].Type).To(Equal(d.Type))
			})
		})
		// Test that the marshal/unmarshal cycle is lossless.
		unmarshalConds := Conditions{}
		err = json.Unmarshal(data, &unmarshalConds)
		if err != nil {
			Fail(fmt.Sprintf("Failed to unmarshal JSON: %s", err))
		}

		Describe("Unmarshalled conditions", func() {
			It(" should equal original conditions", func() {
				Expect(unmarshalConds).To(Equal(conditions))
			})
		})
	})

	Describe("Testing Getcondtion for a non existing value", func() {

		conditions := initConditions(generateCondition("A", corev1.ConditionTrue))
		actualCondition := conditions.GetCondition(ConditionType("B"))

		It(" should be empty for getCondition", func() {
			Expect(actualCondition).Should(BeNil())
		})
	})

	Describe("Testing if we can remove non present element", func() {
		var conditions *Conditions

		It(" should be false for RemoveCondition", func() {
			Expect(conditions.RemoveCondition(ConditionType("C"))).Should(BeFalse())
		})
	})

	Describe("Testing RemoveCondition", func() {

		var conditions Conditions

		BeforeEach(func() {
			conditions = initConditions(
				generateCondition("A", corev1.ConditionTrue),
				generateCondition("B", corev1.ConditionTrue),
			)
		})

		It(" should remove A and not B", func() {
			Expect(conditions.RemoveCondition(ConditionType("A"))).Should(BeTrue())
			Expect(conditions.GetCondition(ConditionType("A"))).Should(BeNil())
			Expect(conditions.GetCondition(ConditionType("B"))).ShouldNot(BeNil())
			Expect(len(conditions)).To(Equal(1))
		})
		It(" should be False for Removing C as it does not exist", func() {
			Expect(conditions.RemoveCondition(ConditionType("C"))).Should(BeFalse())
			Expect(conditions.GetCondition(ConditionType("A"))).ShouldNot(BeNil())
			Expect(conditions.GetCondition(ConditionType("B"))).ShouldNot(BeNil())
			Expect(len(conditions)).To(Equal(2))
		})
	})

	Describe("Testing ConditionsSetEmpty", func() {

		It(" should be equal to the actial one", func() {
			conditions := initConditions()
			setCondition := generateCondition("A", corev1.ConditionTrue)

			Expect(conditions.SetCondition(setCondition)).Should(BeTrue())

			expectedCondition := withLastTransitionTime(setCondition, initTime.Add(clockInterval))
			actualCondition := conditions.GetCondition(setCondition.Type)

			Expect(len(conditions)).To(Equal(1))
			Expect(expectedCondition).To(Equal(*actualCondition))
		})
	})

	Describe("TestConditionsSetNotExists", func() {

		It(" should be equal to the axtial one", func() {
			conditions := initConditions(generateCondition("B", corev1.ConditionTrue))
			setCondition := generateCondition("A", corev1.ConditionTrue)

			Expect(conditions.SetCondition(setCondition)).Should(BeTrue())

			expectedCondition := withLastTransitionTime(setCondition, initTime.Add(clockInterval))
			actualCondition := conditions.GetCondition(expectedCondition.Type)

			Expect(len(conditions)).To(Equal(2))
			Expect(expectedCondition).To(Equal(*actualCondition))
		})
	})

	Describe("TestConditionsSetExistsIdentical", func() {

		existingCondition := generateCondition("A", corev1.ConditionTrue)
		conditions := initConditions(existingCondition)
		setCondition := existingCondition

		temp := conditions.SetCondition(setCondition)
		Describe("Before adding the transition time", func() {
			It(" should be false for setCondition", func() {
				Expect(temp).Should(BeFalse())
			})
		})

		expectedCondition := withLastTransitionTime(setCondition, initTime)
		actualCondition := conditions.GetCondition(expectedCondition.Type)

		Describe("After adding the transition time", func() {
			It(" length of conditions should be equal to 1", func() {
				Expect(len(conditions)).To(Equal(1))
			})
			It(fmt.Sprintf(" Expected condition should be %+v", expectedCondition), func() {
				Expect(expectedCondition).To(Equal(*actualCondition))
			})
		})
	})

	Describe("TestConditionsSetExistsDifferentReasonAndStatus", func() {
		var (
			conditions   Conditions
			setCondition Condition
		)

		BeforeEach(func() {
			existingCondition := generateCondition("A", corev1.ConditionTrue)
			conditions = initConditions(existingCondition)
			setCondition = existingCondition
			setCondition.Reason = "ChangedReason"
		})

		It(" should exist even with different Reason", func() {

			Expect(conditions.SetCondition(setCondition)).Should(BeTrue())

			expectedCondition := withLastTransitionTime(setCondition, initTime)
			actualCondition := conditions.GetCondition(expectedCondition.Type)

			Expect(len(conditions)).To(Equal(1))
			Expect(expectedCondition).To(Equal(*actualCondition))
		})
		It(" should exist even with different Status", func() {

			setCondition.Status = corev1.ConditionFalse
			Expect(conditions.SetCondition(setCondition)).Should(BeTrue())

			expectedCondition := withLastTransitionTime(setCondition, initTime.Add(clockInterval))
			actualCondition := conditions.GetCondition(expectedCondition.Type)

			Expect(len(conditions)).To(Equal(1))
			Expect(expectedCondition).To(Equal(*actualCondition))

		})
	})
})
