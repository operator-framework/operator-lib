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

package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/util/workqueue"
)

var _ = Describe("InstrumentedEnqueueRequestForObject", func() {
	var q workqueue.RateLimitingInterface
	var instance InstrumentedEnqueueRequestForObject
	var pod *corev1.Pod

	BeforeEach(func() {
		q = controllertest.Queue{Interface: workqueue.New()}
		instance = InstrumentedEnqueueRequestForObject{}
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "biz",
				Name:      "biz",
			},
		}
	})
	Describe("Create", func() {
		It("should enqueue a request & emit a metric on a CreateEvent", func() {
			evt := event.CreateEvent{
				Object: pod,
				Meta:   pod.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(1))
			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: pod.Namespace,
					Name:      pod.Name,
				},
			}))

			gauges, err := metrics.Registry.Gather()
			Expect(err).Should(BeNil())
			Expect(len(gauges)).To(Equal(1))
			for _, l := range gauges[0].Metric[0].Label {
				if l.Name == ptrString("name") || l.Name == ptrString("namespace") {
					Expect(l.Value).To(Equal("biz"))
				}
			}
		})
	})

	Describe("Delete", func() {
		It("should enqueue a request & remove the metric on a DeleteEvent", func() {
			evt := event.DeleteEvent{
				Object: pod,
				Meta:   pod.GetObjectMeta(),
			}

			instance.Delete(evt, q)
			Expect(q.Len()).To(Equal(1))
			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: pod.Namespace,
					Name:      pod.Name,
				},
			}))

			gauges, err := metrics.Registry.Gather()
			Expect(err).Should(BeNil())
			Expect(len(gauges)).To(Equal(0))
		})
	})

	Describe("Update", func() {
		It("should enqueue a request in case of UpdateEvent", func() {
			newpod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "baz",
					Name:      "baz",
				},
			}
			evt := event.UpdateEvent{
				ObjectOld: pod,
				MetaOld:   pod.GetObjectMeta(),
				ObjectNew: newpod,
				MetaNew:   newpod.GetObjectMeta(),
			}

			instance.Update(evt, q)
			Expect(q.Len()).To(Equal(2))
			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: pod.Namespace,
					Name:      pod.Name,
				},
			}))

			gauges, err := metrics.Registry.Gather()
			Expect(err).Should(BeNil())
			Expect(len(gauges)).To(Equal(1))
			for _, l := range gauges[0].Metric[0].Label {
				if l.Name == ptrString("name") || l.Name == ptrString("namespace") {
					Expect(l.Value).To(Equal("biz"))
				}
			}
		})
	})

	Describe("getResourceLabels", func() {
		It("should fill out map with values from given objects", func() {
			labelMap := getResourceLabels(pod.GetObjectMeta(), pod)
			Expect(labelMap).ShouldNot(BeEmpty())
			Expect(len(labelMap)).To(Equal(5))
			Expect(labelMap["name"]).To(Equal(pod.GetObjectMeta().GetName()))
			Expect(labelMap["namespace"]).To(Equal(pod.GetObjectMeta().GetNamespace()))
			Expect(labelMap["group"]).To(Equal(pod.GetObjectKind().GroupVersionKind().Group))
			Expect(labelMap["version"]).To(Equal(pod.GetObjectKind().GroupVersionKind().Version))
			Expect(labelMap["kind"]).To(Equal(pod.GetObjectKind().GroupVersionKind().Kind))
		})
	})
})

func ptrString(s string) *string {
	return &s
}
