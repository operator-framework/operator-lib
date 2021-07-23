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
	"github.com/operator-framework/operator-lib/handler/internal/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("InstrumentedEnqueueRequestForObject", func() {
	var q workqueue.RateLimitingInterface
	var instance InstrumentedEnqueueRequestForObject
	var pod *corev1.Pod

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.ResourceCreatedAt)

	BeforeEach(func() {
		q = controllertest.Queue{Interface: workqueue.New()}
		instance = InstrumentedEnqueueRequestForObject{}
		pod = &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "biznamespace",
				Name:              "bizname",
				CreationTimestamp: metav1.Now(),
			},
		}
	})
	Describe("Create", func() {
		It("should enqueue a request & emit a metric on a CreateEvent", func() {
			evt := event.CreateEvent{
				Object: pod,
			}

			// test the create
			instance.Create(evt, q)

			// verify workqueue
			Expect(q.Len()).To(Equal(1))
			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: pod.Namespace,
					Name:      pod.Name,
				},
			}))

			// verify metrics
			gauges, err := registry.Gather()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(gauges)).To(Equal(1))
			assertMetrics(gauges[0], 1, []*corev1.Pod{pod})
		})
	})

	Describe("Delete", func() {
		Context("when a gauge already exists", func() {
			BeforeEach(func() {
				evt := event.CreateEvent{
					Object: pod,
				}
				instance.Create(evt, q)
				Expect(q.Len()).To(Equal(1))
			})
			It("should enqueue a request & remove the metric on a DeleteEvent", func() {
				evt := event.DeleteEvent{
					Object: pod,
				}

				// test the delete
				instance.Delete(evt, q)

				// verify workqueue
				Expect(q.Len()).To(Equal(1))
				i, _ := q.Get()
				Expect(i).To(Equal(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: pod.Namespace,
						Name:      pod.Name,
					},
				}))

				// verify metrics
				gauges, err := registry.Gather()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(gauges)).To(Equal(0))
			})
		})
		Context("when a gauge does not exist", func() {
			It("should enqueue a request & there should be no new metric on a DeleteEvent", func() {
				evt := event.DeleteEvent{
					Object: pod,
				}

				// test the delete
				instance.Delete(evt, q)

				// verify workqueue
				Expect(q.Len()).To(Equal(1))
				i, _ := q.Get()
				Expect(i).To(Equal(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: pod.Namespace,
						Name:      pod.Name,
					},
				}))

				// verify metrics
				gauges, err := registry.Gather()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(gauges)).To(Equal(0))
			})
		})

	})

	Describe("Update", func() {
		It("should enqueue a request in case of UpdateEvent", func() {
			newpod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "baznamespace",
					Name:      "bazname",
				},
			}
			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newpod,
			}

			// test the update
			instance.Update(evt, q)

			// verify workqueue
			Expect(q.Len()).To(Equal(1))
			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: newpod.Namespace,
					Name:      newpod.Name,
				},
			}))

			// verify metrics
			gauges, err := registry.Gather()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(gauges)).To(Equal(1))
			assertMetrics(gauges[0], 2, []*corev1.Pod{newpod, pod})
		})
	})

	Describe("getResourceLabels", func() {
		It("should fill out map with values from given objects", func() {
			labelMap := getResourceLabels(pod)
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

func assertMetrics(gauge *dto.MetricFamily, count int, pods []*corev1.Pod) {
	// need variables to compare the pointers
	name := "name"
	namespace := "namespace"
	g := "group"
	v := "version"
	k := "kind"

	Expect(len(gauge.Metric)).To(Equal(count))
	for i := 0; i < count; i++ {
		Expect(*gauge.Metric[i].Gauge.Value).To(Equal(float64(pods[i].GetObjectMeta().GetCreationTimestamp().UTC().Unix())))

		for _, l := range gauge.Metric[i].Label {
			switch l.Name {
			case &name:
				Expect(l.Value).To(Equal(pods[i].GetObjectMeta().GetName()))
			case &namespace:
				Expect(l.Value).To(Equal(pods[i].GetObjectMeta().GetNamespace()))
			case &g:
				Expect(l.Value).To(Equal(pods[i].GetObjectKind().GroupVersionKind().Group))
			case &v:
				Expect(l.Value).To(Equal(pods[i].GetObjectKind().GroupVersionKind().Version))
			case &k:
				Expect(l.Value).To(Equal(pods[i].GetObjectKind().GroupVersionKind().Kind))
			}
		}
	}
}
