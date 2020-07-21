package status

import (
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
			It(" length of conditions should be equal to 1", func() {
				Expect(len(conditions)).To(Equal(2))
			})
		})

		Describe("After adding the transition time", func() {
			It(fmt.Sprintf(" Expected condition should be %+v", expectedCondition), func() {
				Expect(expectedCondition).To(Equal(*actualCondition))
			})
		})

	})

})

// func TestConditionsSetNotExists(t *testing.T) {
// 	conditions := initConditions(generateCondition("B", corev1.ConditionTrue))

// 	setCondition := generateCondition("A", corev1.ConditionTrue)
// 	assert.True(t, conditions.SetCondition(setCondition))

// 	expectedCondition := withLastTransitionTime(setCondition, initTime.Add(clockInterval))
// 	actualCondition := conditions.GetCondition(expectedCondition.Type)
// 	assert.Equal(t, 2, len(conditions))
// 	assert.Equal(t, expectedCondition, *actualCondition)
// }
