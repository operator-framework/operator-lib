// Copyright 2021 The Operator-SDK Authors
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

package annotation_test

import (
	"github.com/operator-framework/operator-lib/internal/annotation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("filter", func() {
	const annotationKey = "my.app/paused"

	var (
		err error
		q   workqueue.RateLimitingInterface
		pod *corev1.Pod
	)
	BeforeEach(func() {
		q = controllertest.Queue{Interface: workqueue.New()}

		pod = &corev1.Pod{}
		pod.SetName("foo")
		pod.SetNamespace("default")
	})

	Context("Falsy", func() {

		var (
			pred predicate.Predicate
			hdlr handler.EventHandler
		)
		BeforeEach(func() {
			pred, err = annotation.NewFalsyPredicate(annotationKey, annotation.Options{Log: logf.NullLogger{}})
			Expect(err).NotTo(HaveOccurred())
			hdlr, err = annotation.NewFalsyEventHandler(annotationKey, annotation.Options{Log: logf.NullLogger{}})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("Create", func() { //nolint:dupl
			var e event.CreateEvent

			Context("returns true", func() {
				It("receives an event for a nil object", func() {
					e = makeCreateEventFor(nil)
					Expect(pred.Create(e)).To(BeTrue())
				})
				It("receives an event for an object not having any annotations", func() {
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeTrue())
					hdlr.Create(e, q)
					verifyQueueHasPod(q, pod)
				})
				It("receives an event for an object with a registered key and false value", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeTrue())
					hdlr.Create(e, q)
					verifyQueueHasPod(q, pod)
				})
				It("receives an event for an object with a non-registered key and true value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "true"})
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeTrue())
					hdlr.Create(e, q)
					verifyQueueHasPod(q, pod)
				})
				It("receives an event for an object with a non-registered key and false value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "false"})
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeTrue())
					hdlr.Create(e, q)
					verifyQueueHasPod(q, pod)
				})
			})
			Context("returns false", func() {
				It("receives an event for an object with a registered key", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeFalse())
					hdlr.Create(e, q)
					verifyQueueEmpty(q)
				})
			})
		})

		Context("Delete", func() { //nolint:dupl
			var e event.DeleteEvent

			Context("returns true", func() {
				It("receives an event for a nil object", func() {
					e = makeDeleteEventFor(nil)
					Expect(pred.Delete(e)).To(BeTrue())
				})
				It("receives an event for an object not having any annotations", func() {
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeTrue())
					hdlr.Delete(e, q)
					verifyQueueHasPod(q, pod)
				})
				It("receives an event for an object with a registered key and false value", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeTrue())
					hdlr.Delete(e, q)
					verifyQueueHasPod(q, pod)
				})
				It("receives an event for an object with a non-registered key and true value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "true"})
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeTrue())
					hdlr.Delete(e, q)
					verifyQueueHasPod(q, pod)
				})
				It("receives an event for an object with a non-registered key and false value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "false"})
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeTrue())
					hdlr.Delete(e, q)
					verifyQueueHasPod(q, pod)
				})
			})
			Context("returns false", func() {
				It("receives an event for an object with a registered key", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeFalse())
					hdlr.Delete(e, q)
					verifyQueueEmpty(q)
				})
			})
		})

		Context("Generic", func() { //nolint:dupl
			var e event.GenericEvent

			Context("returns true", func() {
				It("receives an event for a nil object", func() {
					e = makeGenericEventFor(nil)
					Expect(pred.Generic(e)).To(BeTrue())
				})
				It("receives an event for an object not having any annotations", func() {
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeTrue())
					hdlr.Generic(e, q)
					verifyQueueHasPod(q, pod)
				})
				It("receives an event for an object with a registered key and false value", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeTrue())
					hdlr.Generic(e, q)
					verifyQueueHasPod(q, pod)
				})
				It("receives an event for an object with a non-registered key and true value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "true"})
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeTrue())
					hdlr.Generic(e, q)
					verifyQueueHasPod(q, pod)
				})
				It("receives an event for an object with a non-registered key and false value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "false"})
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeTrue())
					hdlr.Generic(e, q)
					verifyQueueHasPod(q, pod)
				})
			})
			Context("returns false", func() {
				It("receives an event for an object with a registered key", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeFalse())
					hdlr.Generic(e, q)
					verifyQueueEmpty(q)
				})
			})
		})

		Context("Update", func() {
			var e event.UpdateEvent

			Context("returns true", func() {
				It("receives both objects being nil", func() {
					e = makeUpdateEventFor(nil, nil)
					Expect(pred.Update(e)).To(BeTrue())
				})
				It("receives neither objects having any annotations", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, new)
				})
				It("receives the new object with a registered key and false value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					new.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, new)
				})
				It("receives the old object with a registered key and false value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					old.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, new)
				})
				It("receives the new object with a non-registered key and true value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					new.SetAnnotations(map[string]string{"my.app/foo": "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, new)
				})
				It("receives the new object with a non-registered key and false value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					new.SetAnnotations(map[string]string{"my.app/foo": "false"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, new)
				})
			})
			Context("returns false", func() {
				It("receives the new object with a registered key", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					new.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
				It("receives the old object with a registered key", func() {
					old := pod.DeepCopy()
					old.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, nil)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
				It("receives both objects with a registered key", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					new.SetAnnotations(map[string]string{annotationKey: "true"})
					old.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
				It("receives the old object with a registered key and false value, and new with true", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					old.SetAnnotations(map[string]string{annotationKey: "false"})
					new.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
				It("receives the old object having no annotations, and new with a registered key and true value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					new.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
			})
		})
	})

	Context("Truthy", func() {

		var (
			pred predicate.Predicate
			hdlr handler.EventHandler
		)
		BeforeEach(func() {
			pred, err = annotation.NewTruthyPredicate(annotationKey, annotation.Options{Log: logf.NullLogger{}})
			Expect(err).NotTo(HaveOccurred())
			hdlr, err = annotation.NewTruthyEventHandler(annotationKey, annotation.Options{Log: logf.NullLogger{}})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("Create", func() { //nolint:dupl
			var e event.CreateEvent

			Context("returns false", func() {
				It("receives an event for a nil object", func() {
					e = makeCreateEventFor(nil)
					Expect(pred.Create(e)).To(BeFalse())
				})
				It("receives an event for an object not having any annotations", func() {
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeFalse())
					hdlr.Create(e, q)
					verifyQueueEmpty(q)
				})
				It("receives an event for an object with a registered key and false value", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeFalse())
					hdlr.Create(e, q)
					verifyQueueEmpty(q)
				})
				It("receives an event for an object with a non-registered key and true value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "true"})
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeFalse())
					hdlr.Create(e, q)
					verifyQueueEmpty(q)
				})
				It("receives an event for an object with a non-registered key and false value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "false"})
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeFalse())
					hdlr.Create(e, q)
					verifyQueueEmpty(q)
				})
			})
			Context("returns true", func() {
				It("receives an event for an object with a registered key", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeCreateEventFor(pod)
					Expect(pred.Create(e)).To(BeTrue())
					hdlr.Create(e, q)
					verifyQueueHasPod(q, pod)
				})
			})
		})

		Context("Delete", func() { //nolint:dupl
			var e event.DeleteEvent

			Context("returns false", func() {
				It("receives an event for a nil object", func() {
					e = makeDeleteEventFor(nil)
					Expect(pred.Delete(e)).To(BeFalse())
				})
				It("receives an event for an object not having any annotations", func() {
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeFalse())
					hdlr.Delete(e, q)
					verifyQueueEmpty(q)
				})
				It("receives an event for an object with a registered key and false value", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeFalse())
					hdlr.Delete(e, q)
					verifyQueueEmpty(q)
				})
				It("receives an event for an object with a non-registered key and true value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "true"})
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeFalse())
					hdlr.Delete(e, q)
					verifyQueueEmpty(q)
				})
				It("receives an event for an object with a non-registered key and false value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "false"})
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeFalse())
					hdlr.Delete(e, q)
					verifyQueueEmpty(q)
				})
			})
			Context("returns true", func() {
				It("receives an event for an object with a registered key", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeDeleteEventFor(pod)
					Expect(pred.Delete(e)).To(BeTrue())
					hdlr.Delete(e, q)
					verifyQueueHasPod(q, pod)
				})
			})
		})

		Context("Generic", func() { //nolint:dupl
			var e event.GenericEvent

			Context("returns false", func() {
				It("receives an event for a nil object", func() {
					e = makeGenericEventFor(nil)
					Expect(pred.Generic(e)).To(BeFalse())
				})
				It("receives an event for an object not having any annotations", func() {
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeFalse())
					hdlr.Generic(e, q)
					verifyQueueEmpty(q)
				})
				It("receives an event for an object with a registered key and false value", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeFalse())
					hdlr.Generic(e, q)
					verifyQueueEmpty(q)
				})
				It("receives an event for an object with a non-registered key and true value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "true"})
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeFalse())
					hdlr.Generic(e, q)
					verifyQueueEmpty(q)
				})
				It("receives an event for an object with a non-registered key and false value", func() {
					pod.SetAnnotations(map[string]string{"my.app/foo": "false"})
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeFalse())
					hdlr.Generic(e, q)
					verifyQueueEmpty(q)
				})
			})
			Context("returns true", func() {
				It("receives an event for an object with a registered key", func() {
					pod.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeGenericEventFor(pod)
					Expect(pred.Generic(e)).To(BeTrue())
					hdlr.Generic(e, q)
					verifyQueueHasPod(q, pod)
				})
			})
		})

		Context("Update", func() {
			var e event.UpdateEvent

			Context("returns false", func() {
				It("receives both objects being nil", func() {
					e = makeUpdateEventFor(nil, nil)
					Expect(pred.Update(e)).To(BeFalse())
				})
				It("receives neither objects having any annotations", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
				It("receives the new object with a registered key and false value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					new.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
				It("receives the old object with a registered key and false value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					old.SetAnnotations(map[string]string{annotationKey: "false"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
				It("receives the new object with a non-registered key and true value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					new.SetAnnotations(map[string]string{"my.app/foo": "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
				It("receives the new object with a non-registered key and false value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					new.SetAnnotations(map[string]string{"my.app/foo": "false"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeFalse())
					hdlr.Update(e, q)
					verifyQueueEmpty(q)
				})
			})
			Context("returns true", func() {
				It("receives the new object with a registered key", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					new.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, new)
				})
				It("receives the old object with a registered key", func() {
					old := pod.DeepCopy()
					old.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, nil)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, old)
				})
				It("receives both objects with a registered key", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					new.SetAnnotations(map[string]string{annotationKey: "true"})
					old.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, new)
				})
				It("receives the old object with a registered key and false value, and new with true", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					old.SetAnnotations(map[string]string{annotationKey: "false"})
					new.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, new)
				})
				It("receives the old object having no annotations, and new with a registered key and true value", func() {
					old, new := pod.DeepCopy(), pod.DeepCopy()
					old.SetLabels(map[string]string{"id": "old"})
					new.SetLabels(map[string]string{"id": "new"})
					new.SetAnnotations(map[string]string{annotationKey: "true"})
					e = makeUpdateEventFor(old, new)
					Expect(pred.Update(e)).To(BeTrue())
					hdlr.Update(e, q)
					verifyQueueHasPod(q, new)
				})
			})
		})
	})

})

func verifyQueueHasPod(q workqueue.RateLimitingInterface, pod *corev1.Pod) {
	ExpectWithOffset(1, q.Len()).To(Equal(1))
	i, _ := q.Get()
	ExpectWithOffset(1, i).To(Equal(reconcile.Request{
		NamespacedName: client.ObjectKeyFromObject(pod),
	}))
}

func verifyQueueEmpty(q workqueue.RateLimitingInterface) {
	ExpectWithOffset(1, q.Len()).To(Equal(0))
}
