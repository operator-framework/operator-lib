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
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiv2 "github.com/operator-framework/api/pkg/operators/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubeclock "k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Condition", func() {
	var ns = "default"
	ctx := context.TODO()
	var clock kubeclock.Clock = &kubeclock.RealClock{}
	var transitionTime = metav1.Time{Time: clock.Now()}
	var cl client.Client

	Describe("Get/Set", func() {
		var operatorCond *apiv2.OperatorCondition

		objKey := types.NamespacedName{
			Name:      "operator-condition-test",
			Namespace: ns,
		}

		BeforeEach(func() {
			operatorCond = &apiv2.OperatorCondition{
				ObjectMeta: metav1.ObjectMeta{Name: "operator-condition-test", Namespace: ns},
				Spec: apiv2.OperatorConditionSpec{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditionFoo),
							Status:             metav1.ConditionTrue,
							Reason:             "foo",
							Message:            "The operator is in foo condition",
							LastTransitionTime: transitionTime,
						},
					},
				},
			}

			// Create Operator Condition
			err := os.Setenv(operatorCondEnvVar, "operator-condition-test")
			Expect(err).NotTo(HaveOccurred())
			readNamespace = func() (string, error) {
				return ns, nil
			}

			// create a new client
			sch := runtime.NewScheme()
			err = apiv2.AddToScheme(sch)
			Expect(err).NotTo(HaveOccurred())
			cl = fake.NewClientBuilder().WithScheme(sch).Build()

			// create an operator Condition resource
			err = cl.Create(ctx, operatorCond)
			Expect(err).NotTo(HaveOccurred())

			// Update its status
			err = cl.Status().Update(ctx, operatorCond)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			deleteCondition(ctx, cl, operatorCond)
		})

		Context("Get", func() {
			It("should fetch the right condition", func() {
				By("creating a new Condition")
				c, err := NewCondition(cl, conditionFoo)
				Expect(err).NotTo(HaveOccurred())

				By("Fetching the condition from the cluster")
				con, err := c.Get(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(con).NotTo(BeNil())
				Expect(con.Status).To(BeEquivalentTo(metav1.ConditionTrue))
			})

			It("should error when the condition cannot be found", func() {
				c, err := NewCondition(cl, conditionBar)
				Expect(err).NotTo(HaveOccurred())

				con, err := c.Get(ctx)
				Expect(err).To(HaveOccurred())
				Expect(con).To(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("conditionType %v not found", conditionBar)))
			})

			It("should error when operator Condition is not present in cluster", func() {
				err := os.Setenv(operatorCondEnvVar, "NON_EXISTING_COND")
				Expect(err).NotTo(HaveOccurred())

				By("setting the status of a new condition")
				c, err := NewCondition(cl, conditionFoo)
				Expect(err).NotTo(HaveOccurred())
				con, err := c.Get(ctx)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
				Expect(con).To(BeNil())
			})
		})

		Context("Set", func() {
			It("should update a condition correctly", func() {
				By("setting the status of an existing condition")
				c, err := NewCondition(cl, conditionFoo)
				Expect(err).NotTo(HaveOccurred())
				err = c.Set(ctx, metav1.ConditionFalse, WithReason("not_in_foo_state"), WithMessage("test"))
				Expect(err).NotTo(HaveOccurred())

				By("fetching the condition from cluster")
				op := &apiv2.OperatorCondition{}
				err = cl.Get(ctx, objKey, op)
				Expect(err).NotTo(HaveOccurred())

				By("checking if the condition has been updated")
				res := op.Spec.Conditions[0]
				Expect(res.Message).To(BeEquivalentTo("test"))
				Expect(res.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(res.Reason).To(BeEquivalentTo("not_in_foo_state"))
			})

			It("should add a condition if not present", func() {
				By("setting the status of a new condition")
				c, err := NewCondition(cl, conditionBar)
				Expect(err).NotTo(HaveOccurred())
				err = c.Set(ctx, metav1.ConditionTrue, WithReason("in_bar_state"), WithMessage("test"))
				Expect(err).NotTo(HaveOccurred())

				By("fetching the condition from cluster")
				op := &apiv2.OperatorCondition{}
				err = cl.Get(ctx, objKey, op)
				Expect(err).NotTo(HaveOccurred())

				By("checking if the condition has been updated")
				res := op.Spec.Conditions
				Expect(len(res)).To(BeEquivalentTo(2))
				Expect(meta.IsStatusConditionTrue(res, string(conditionBar))).To(BeTrue())
			})
			It("should error when operator Condition is not present in cluster", func() {
				err := os.Setenv(operatorCondEnvVar, "NON_EXISTING_COND")
				Expect(err).NotTo(HaveOccurred())

				By("setting the status of a new condition")
				c, err := NewCondition(cl, conditionBar)
				Expect(err).NotTo(HaveOccurred())
				err = c.Set(ctx, metav1.ConditionTrue, WithReason("in_bar_state"), WithMessage("test"))
				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})
	})
})
