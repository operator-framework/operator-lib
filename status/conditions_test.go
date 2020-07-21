package status

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	"time"

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

var _ = Describe("Testing Conditions.go\n", func() {

	Describe("TestConditionDeepCopy", func() {

		var (
			a     Condition
			aCopy Condition
		)

		// BeforeEach(func() {
		a = generateCondition("A", corev1.ConditionTrue)
		a.DeepCopyInto(&aCopy)

		// })

		Context("Testing for Type", func() {
			It(fmt.Sprintf(" should be equal to %+v", a.Type), func() {
				Expect(aCopy.Type).To(Equal(a.Type))
			})
		})

		Context("Testing for Satus", func() {
			It(fmt.Sprintf(" should be equal to %+v", a.Status), func() {
				Expect(aCopy.Status).To(Equal(a.Status))
			})
		})

		Context("Testing for Reason", func() {
			It(fmt.Sprintf(" should be equal to %+v", a.Reason), func() {
				Expect(aCopy.Reason).To(Equal(a.Reason))
			})
		})

		Context("Testing for Message", func() {
			It(fmt.Sprintf(" should be equal to %+v", a.Message), func() {
				Expect(aCopy.Message).To(Equal(a.Message))
			})
		})

		Context("Testing for LastTransactionTime", func() {
			It(fmt.Sprintf(" should be equal to %+v", a.LastTransitionTime), func() {
				Expect(aCopy.LastTransitionTime).To(Equal(a.LastTransitionTime))
			})
		})

	})

	Describe("TestConditionsSetEmpty", func() {

		var (
			conditions        Conditions
			setCondition      Condition
			expectedCondition Condition
			actualCondition   *Condition
		)

		// ask about this
		// BeforeEach(func() {
		conditions = initConditions()
		setCondition = generateCondition("A", corev1.ConditionTrue)
		// })

		// ask about this
		temp := conditions.SetCondition(setCondition)
		Describe("Before adding the transition time", func() {
			It(" should be true for setCondition", func() {
				Expect(temp).Should(BeTrue())
			})
		})

		expectedCondition = withLastTransitionTime(setCondition, initTime.Add(clockInterval))
		actualCondition = conditions.GetCondition(setCondition.Type)

		Describe("After adding the transition time", func() {
			It(" length of conditions should be equal to 1", func() {
				Expect(len(conditions)).To(Equal(1))
			})
		})

		Describe("After adding the transition time", func() {
			It(fmt.Sprintf(" Expected condition should be %+v", expectedCondition), func() {
				Expect(expectedCondition).To(Equal(*actualCondition))
			})
		})

	})

	Describe("TestConditionsSetNotExists", func() {

		var (
			conditions        Conditions
			setCondition      Condition
			expectedCondition Condition
			actualCondition   *Condition
		)

		// BeforeEach(func() {
		conditions = initConditions(generateCondition("B", corev1.ConditionTrue))
		setCondition = generateCondition("A", corev1.ConditionTrue)
		// })

		temp := conditions.SetCondition(setCondition)
		Describe("Before adding the transition time", func() {
			It(" should be true for setCondition", func() {
				Expect(temp).Should(BeTrue())
			})
		})

		expectedCondition = withLastTransitionTime(setCondition, initTime.Add(clockInterval))
		actualCondition = conditions.GetCondition(expectedCondition.Type)

		Describe("After adding the transition time", func() {
			It(" length of conditions should be equal to 2", func() {
				Expect(len(conditions)).To(Equal(2))
			})
		})

		Describe("After adding the transition time", func() {
			It(fmt.Sprintf(" Expected condition should be %+v", expectedCondition), func() {
				Expect(expectedCondition).To(Equal(*actualCondition))
			})
		})

	})

	Describe("TestConditionsSetExistsIdentical", func() {

		var (
			conditions        Conditions
			setCondition      Condition
			expectedCondition Condition
			actualCondition   *Condition
			existingCondition Condition
		)

		// BeforeEach(func() {
		existingCondition = generateCondition("A", corev1.ConditionTrue)
		conditions = initConditions(existingCondition)
		setCondition = existingCondition
		// })

		temp := conditions.SetCondition(setCondition)
		Describe("Before adding the transition time", func() {
			It(" should be false for setCondition", func() {
				Expect(temp).Should(BeFalse())
			})
		})

		expectedCondition = withLastTransitionTime(setCondition, initTime)
		actualCondition = conditions.GetCondition(expectedCondition.Type)

		Describe("After adding the transition time", func() {
			It(" length of conditions should be equal to 1", func() {
				Expect(len(conditions)).To(Equal(1))
			})
		})

		Describe("After adding the transition time", func() {
			It(fmt.Sprintf(" Expected condition should be %+v", expectedCondition), func() {
				Expect(expectedCondition).To(Equal(*actualCondition))
			})
		})

	})

	Describe("TestConditionsSetExistsDifferentReason", func() {

		var (
			conditions        Conditions
			setCondition      Condition
			expectedCondition Condition
			actualCondition   *Condition
			existingCondition Condition
		)

		// BeforeEach(func() {
		existingCondition = generateCondition("A", corev1.ConditionTrue)
		conditions = initConditions(existingCondition)
		setCondition = existingCondition
		setCondition.Reason = "ChangedReason"
		// })

		temp := conditions.SetCondition(setCondition)
		Describe("Before adding the transition time", func() {
			It(" should be true for setCondition", func() {
				Expect(temp).Should(BeTrue())
			})
		})

		expectedCondition = withLastTransitionTime(setCondition, initTime)
		actualCondition = conditions.GetCondition(expectedCondition.Type)

		Describe("After adding the transition time", func() {
			It(" length of conditions should be equal to 1", func() {
				Expect(len(conditions)).To(Equal(1))
			})
		})

		Describe("After adding the transition time", func() {
			It(fmt.Sprintf(" Expected condition should be %+v", expectedCondition), func() {
				Expect(expectedCondition).To(Equal(*actualCondition))
			})
		})

	})

	Describe("TestConditionsSetExistsDifferentStatus", func() {

		var (
			conditions        Conditions
			setCondition      Condition
			expectedCondition Condition
			actualCondition   *Condition
			existingCondition Condition
		)

		// BeforeEach(func() {
		existingCondition = generateCondition("A", corev1.ConditionTrue)
		conditions = initConditions(existingCondition)
		setCondition = existingCondition
		setCondition.Status = corev1.ConditionFalse
		setCondition.Reason = "ChangedReason"
		// })

		temp := conditions.SetCondition(setCondition)
		Describe("Before adding the transition time", func() {
			It(" should be true for setCondition", func() {
				Expect(temp).Should(BeTrue())
			})
		})

		expectedCondition = withLastTransitionTime(setCondition, initTime.Add(clockInterval))
		actualCondition = conditions.GetCondition(expectedCondition.Type)

		Describe("After adding the transition time", func() {
			It(" length of conditions should be equal to 1", func() {
				Expect(len(conditions)).To(Equal(1))
			})
		})

		Describe("After adding the transition time", func() {
			It(fmt.Sprintf(" Expected condition should be %+v", expectedCondition), func() {
				Expect(expectedCondition).To(Equal(*actualCondition))
			})
		})

	})

	Describe("TestConditionsGetNotExists", func() {
		var (
			conditions      Conditions
			actualCondition *Condition
		)

		// BeforeEach(func() {
		conditions = initConditions(generateCondition("A", corev1.ConditionTrue))
		actualCondition = conditions.GetCondition(ConditionType("B"))
		// })

		Describe("Testing Getcondtion for a non existing value", func() {
			It(" should be empty for getCondition", func() {
				Expect(actualCondition).Should(BeNil())
			})
		})

	})

	Describe("TestConditionsRemoveFromNilConditions", func() {
		var (
			conditions *Conditions
		)

		Describe("Testing if we can remove non present element", func() {
			It(" should be false for RemoveCondition", func() {
				Expect(conditions.RemoveCondition(ConditionType("C"))).Should(BeFalse())
			})
		})

	})

	Describe("TestConditionsRemoveNotExists", func() {
		var (
			conditions Conditions
		)

		conditions = initConditions(
			generateCondition("A", corev1.ConditionTrue),
			generateCondition("B", corev1.ConditionTrue),
		)

		Describe("Testing if we can remove non present element", func() {
			It(" should be false for RemoveCondition", func() {
				Expect(conditions.RemoveCondition(ConditionType("C"))).Should(BeFalse())
			})
		})

		a := conditions.GetCondition(ConditionType("A"))
		b := conditions.GetCondition(ConditionType("B"))

		Describe("Testing Getcondtion for a non existing value", func() {
			It(" should not be empty for getCondition for a", func() {
				Expect(a).ShouldNot(BeNil())
			})
		})

		Describe("Testing Getcondtion for a non existing value", func() {
			It(" should not be empty for getCondition for b", func() {
				Expect(b).ShouldNot(BeNil())
			})
		})

		Describe("Making sure GetCondition does not remove the Condition from Conditions", func() {
			It(" length of conditions should be equal to 2", func() {
				Expect(len(conditions)).To(Equal(2))
			})
		})

	})

	Describe("TestConditionsRemoveExists", func() {
		var (
			conditions Conditions
		)

		conditions = initConditions(
			generateCondition("A", corev1.ConditionTrue),
			generateCondition("B", corev1.ConditionTrue),
		)

		temp := conditions.RemoveCondition(ConditionType("A"))
		Describe("Testing if we can remove non present element", func() {
			It(" should be false for RemoveCondition", func() {
				Expect(temp).Should(BeTrue())
			})
		})

		a := conditions.GetCondition(ConditionType("A"))
		b := conditions.GetCondition(ConditionType("B"))

		Describe("Testing Getcondtion for a non existing value", func() {
			It(" should be empty for getCondition for a", func() {
				Expect(a).Should(BeNil())
			})
		})

		Describe("Testing Getcondtion for a non existing value", func() {
			It(" should not be empty for getCondition for b", func() {
				Expect(b).ShouldNot(BeNil())
			})
		})

		Describe("Checking if only one of the Condition was removed from Conditions", func() {
			It(" length of conditions should be equal to 1", func() {
				Expect(len(conditions)).To(Equal(1))
			})
		})

	})

	Describe("TestConditionsIsTrueFor", func() {
		var (
			conditions Conditions
		)

		conditions = NewConditions(
			generateCondition("False", corev1.ConditionFalse),
			generateCondition("True", corev1.ConditionTrue),
			generateCondition("Unknown", corev1.ConditionUnknown),
		)

		Describe("Testing if True for conditiontype True", func() {
			It(" should be True for IsTrueFor", func() {
				Expect(conditions.IsTrueFor(ConditionType("True"))).Should(BeTrue())
			})
		})

		Describe("Testing if False for conditiontype False", func() {
			It(" should be False for IsTrueFor", func() {
				Expect(conditions.IsTrueFor(ConditionType("False"))).Should(BeFalse())
			})
		})

		Describe("Testing if False for conditiontype Unknown", func() {
			It(" should be False for IsTrueFor", func() {
				Expect(conditions.IsTrueFor(ConditionType("Unknown"))).Should(BeFalse())
			})
		})

		Describe("Testing if False for conditiontype DoesNotExist", func() {
			It(" should be False for IsTrueFor", func() {
				Expect(conditions.IsTrueFor(ConditionType("DoesNotExist"))).Should(BeFalse())
			})
		})

	})

	Describe("TestConditionsIsFalseFor", func() {
		var (
			conditions Conditions
		)

		conditions = NewConditions(
			generateCondition("False", corev1.ConditionFalse),
			generateCondition("True", corev1.ConditionTrue),
			generateCondition("Unknown", corev1.ConditionUnknown),
		)

		Describe("Testing if False for conditiontype True", func() {
			It(" should be False for IsTrueFor", func() {
				Expect(conditions.IsFalseFor(ConditionType("True"))).Should(BeFalse())
			})
		})

		Describe("Testing if True for conditiontype False", func() {
			It(" should be True for IsTrueFor", func() {
				Expect(conditions.IsFalseFor(ConditionType("False"))).Should(BeTrue())
			})
		})

		Describe("Testing if False for conditiontype Unknown", func() {
			It(" should be False for IsTrueFor", func() {
				Expect(conditions.IsFalseFor(ConditionType("Unknown"))).Should(BeFalse())
			})
		})

		Describe("Testing if False for conditiontype DoesNotExist", func() {
			It(" should be False for IsTrueFor", func() {
				Expect(conditions.IsFalseFor(ConditionType("DoesNotExist"))).Should(BeFalse())
			})
		})

	})

	Describe("TestConditionsIsUnknownFor", func() {
		var (
			conditions Conditions
		)

		conditions = NewConditions(
			generateCondition("False", corev1.ConditionFalse),
			generateCondition("True", corev1.ConditionTrue),
			generateCondition("Unknown", corev1.ConditionUnknown),
		)

		Describe("Testing if False for conditiontype True", func() {
			It(" should be False for IsUnknownFor", func() {
				Expect(conditions.IsUnknownFor(ConditionType("True"))).Should(BeFalse())
			})
		})

		Describe("Testing if True for conditiontype False", func() {
			It(" should be False for IsUnknownFor", func() {
				Expect(conditions.IsUnknownFor(ConditionType("False"))).Should(BeFalse())
			})
		})

		Describe("Testing if True for conditiontype Unknown", func() {
			It(" should be True for IsUnknownFor", func() {
				Expect(conditions.IsUnknownFor(ConditionType("Unknown"))).Should(BeTrue())
			})
		})

		Describe("Testing if True for conditiontype DoesNotExist", func() {
			It(" should be True for IsUnknownFor", func() {
				Expect(conditions.IsUnknownFor(ConditionType("DoesNotExist"))).Should(BeTrue())
			})
		})

	})

	Describe("TestConditionsMarshalUnmarshalJSON", func() {

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

		Describe("A", func() {
			It(" A", func() {
				Expect(in[0].Type).To(Equal(a.Type))
			})
		})

		Describe("B", func() {
			It(" B", func() {
				Expect(in[1].Type).To(Equal(b.Type))
			})
		})

		Describe("C", func() {
			It(" C", func() {
				Expect(in[2].Type).To(Equal(c.Type))
			})
		})

		Describe("D", func() {
			It(" D", func() {
				Expect(in[3].Type).To(Equal(d.Type))
			})
		})

		// Test that the marshal/unmarshal cycle is lossless.
		unmarshalConds := Conditions{}
		err = json.Unmarshal(data, &unmarshalConds)
		if err != nil {
			Fail(fmt.Sprintf("Failed to unmarshal JSON: %s", err))
		}

		Describe("XYZ", func() {
			It(" XYZ", func() {
				Expect(unmarshalConds).To(Equal(conditions))
			})
		})

	})

})
