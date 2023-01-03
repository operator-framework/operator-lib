// Copyright 2018 The Operator-SDK Authors
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

package predicate

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("DependentPredicate", func() {
	var (
		pred DependentPredicate
	)

	Describe("Create", func() {
		It("returns false", func() {
			e := makeCreateEventFor(&unstructured.Unstructured{})
			Expect(pred.Create(e)).To(BeFalse())
		})
	})

	Describe("Update", func() {
		var oldObj, newObj *unstructured.Unstructured
		When("objects are equal", func() {
			BeforeEach(func() {
				oldObj = &unstructured.Unstructured{}
				oldObj.SetUID("A")
				newObj = &unstructured.Unstructured{}
				newObj.SetUID("A")
			})

			It("should return false", func() {
				e := makeUpdateEventFor(oldObj, newObj)
				Expect(pred.Update(e)).To(BeFalse())
			})

			When("except status is different", func() {
				BeforeEach(func() {
					newObj.Object["status"] = "foo"
				})
				It("should return false", func() {
					e := makeUpdateEventFor(oldObj, newObj)
					Expect(pred.Update(e)).To(BeFalse())
				})
			})

			When("except resource version is different", func() {
				BeforeEach(func() {
					newObj.SetResourceVersion("bar")
				})
				It("should return false", func() {
					e := makeUpdateEventFor(oldObj, newObj)
					Expect(pred.Update(e)).To(BeFalse())
				})
			})

			When("except time in managedFields is different", func() {
				BeforeEach(func() {
					curTime := time.Now()
					oldTime := metav1.NewTime(curTime)
					oldObj.SetManagedFields([]metav1.ManagedFieldsEntry{{
						Manager:    "test",
						Operation:  "Update",
						APIVersion: "v1",
						Time:       &oldTime,
					}})

					duration, _ := time.ParseDuration("4h")
					newTime := metav1.NewTime(curTime.Add(duration))
					newObj.SetManagedFields([]metav1.ManagedFieldsEntry{{
						Manager:    "test",
						Operation:  "Update",
						APIVersion: "v1",
						Time:       &newTime,
					}})
				})
				It("should return false", func() {
					e := makeUpdateEventFor(oldObj, newObj)
					Expect(pred.Update(e)).To(BeFalse())
				})
			})
		})

		When("objects are different", func() {
			BeforeEach(func() {
				oldObj = &unstructured.Unstructured{}
				oldObj.SetUID("A")
				newObj = &unstructured.Unstructured{}
				newObj.SetUID("B")
			})

			It("should return true", func() {
				e := makeUpdateEventFor(oldObj, newObj)
				Expect(pred.Update(e)).To(BeTrue())
			})
		})
	})

	Describe("Delete", func() {
		It("returns true", func() {
			e := makeDeleteEventFor(&unstructured.Unstructured{})
			Expect(pred.Delete(e)).To(BeTrue())
		})
	})

	Describe("Generic", func() {
		It("returns false", func() {
			e := makeGenericEventFor(&unstructured.Unstructured{})
			Expect(pred.Generic(e)).To(BeFalse())
		})
	})
})
