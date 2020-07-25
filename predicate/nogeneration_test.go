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

package predicate

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ = Describe("NoGenerationPredicate", func() {
	var (
		e    event.UpdateEvent
		pred NoGenerationPredicate
	)

	It("returns true", func() {
		By("both the old and new objects having a generation of 0", func() {
			e = makeUpdateEventFor(&corev1.Pod{}, &corev1.Pod{})
			Expect(pred.Update(e)).To(BeTrue())
		})
	})

	It("returns false", func() {
		By("the new object having a non-zero generation", func() {
			old, new := &appsv1.Deployment{}, &appsv1.Deployment{}
			new.SetGeneration(1)
			e = makeUpdateEventFor(old, new)
			Expect(pred.Update(e)).To(BeFalse())
		})
		// The old generation will never be lower than the new, so we don't have to test that case.
		By("the old and new objects having equal non-zero generations", func() {
			old, new := &appsv1.Deployment{}, &appsv1.Deployment{}
			old.SetGeneration(1)
			new.SetGeneration(1)
			e = makeUpdateEventFor(old, new)
			Expect(pred.Update(e)).To(BeFalse())
		})
		By("the old and new objects having unequal non-zero generations", func() {
			old, new := &appsv1.Deployment{}, &appsv1.Deployment{}
			old.SetGeneration(1)
			new.SetGeneration(2)
			e = makeUpdateEventFor(old, new)
			Expect(pred.Update(e)).To(BeFalse())
		})
	})

	It("logs a message and returns false", func() {
		By("the old object being nil", func() {
			old, new := &appsv1.Deployment{}, &appsv1.Deployment{}
			e = makeUpdateEventFor(old, new)
			e.ObjectOld = nil
			Expect(pred.Update(e)).To(BeFalse())
		})
		By("the old metadata being nil", func() {
			old, new := &appsv1.Deployment{}, &appsv1.Deployment{}
			e = makeUpdateEventFor(old, new)
			e.MetaOld = nil
			Expect(pred.Update(e)).To(BeFalse())
		})
		By("the new object being nil", func() {
			old, new := &appsv1.Deployment{}, &appsv1.Deployment{}
			e = makeUpdateEventFor(old, new)
			e.ObjectNew = nil
			Expect(pred.Update(e)).To(BeFalse())
		})
		By("the new metadata being nil", func() {
			old, new := &appsv1.Deployment{}, &appsv1.Deployment{}
			e = makeUpdateEventFor(old, new)
			e.MetaNew = nil
			Expect(pred.Update(e)).To(BeFalse())
		})
	})

})
