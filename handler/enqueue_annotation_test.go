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
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/util/workqueue"
)

var _ = Describe("Enqueue_anootation", func() {
	var q workqueue.RateLimitingInterface
	var instance EnqueueRequestForAnnotation
	var mapper meta.RESTMapper
	var pod *corev1.Pod
	var podOwner = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "podOwnerNs",
			Name:      "podOwnerName",
		},
	}

	BeforeEach(func() {
		q = controllertest.Queue{Interface: workqueue.New()}
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "biz",
				Name:      "biz",
			},
		}

		err := SetOwnerAnnotation(podOwner, pod, schema.GroupKind{Group: "Pods", Kind: "core"})
		Expect(err).To(BeNil())
		Expect(cfg).NotTo(BeNil())
		mapper, err = apiutil.NewDiscoveryRESTMapper(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(mapper).NotTo(BeNil())
	})

	Describe("EnqueueRequestForAnnotation", func() {
		BeforeEach(func() {
			instance = EnqueueRequestForAnnotation{
				Type: schema.GroupKind{
					Group: "Pods",
					Kind:  "core",
				}}
		})

		It("should enqueue a Request with the annotations of the object in the CreateEvent", func() {
			evt := event.CreateEvent{
				Object: pod,
				Meta:   pod.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))

		})

		It("should enqueue a Request with the annotations of the object in the DeleteEvent", func() {
			evt := event.DeleteEvent{
				Object: pod,
				Meta:   pod.GetObjectMeta(),
			}
			instance.Delete(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})

		It("should enqueue a Request with annotations applied to both objects in UpdateEvent", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"

			evt := event.UpdateEvent{
				ObjectOld: pod,
				MetaOld:   pod.GetObjectMeta(),
				ObjectNew: newPod,
				MetaNew:   newPod.GetObjectMeta(),
			}

			instance.Update(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})

		It("should enqueue a Request with the annotations applied in one of the objects in UpdateEvent", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"
			newPod.Annotations = map[string]string{}

			evt := event.UpdateEvent{
				ObjectOld: pod,
				MetaOld:   pod.GetObjectMeta(),
				ObjectNew: newPod,
				MetaNew:   newPod.GetObjectMeta(),
			}

			instance.Update(evt, q)
			Expect(q.Len()).To(Equal(1))
			i, _ := q.Get()

			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})

		It("should enqueue a Request when the annotations are applied in new object in UpdateEvent", func() {
			var repl *appsv1.ReplicaSet

			repl = &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}
			instance = EnqueueRequestForAnnotation{Type: schema.GroupKind{Group: "ReplicaSet", Kind: "apps"}}

			evt := event.CreateEvent{
				Object: repl,
				Meta:   repl.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))

			newRepl := repl.DeepCopy()
			newRepl.Name = pod.Name + "2"
			newRepl.Namespace = pod.Namespace + "2"

			newRepl.Annotations = map[string]string{
				TypeAnnotation:           schema.GroupKind{Group: "ReplicaSet", Kind: "apps"}.String(),
				NamespacedNameAnnotation: "foo/faz",
			}

			instance2 := EnqueueRequestForAnnotation{Type: schema.GroupKind{Group: "ReplicaSet", Kind: "apps"}}

			evt2 := event.UpdateEvent{
				ObjectOld: repl,
				MetaOld:   repl.GetObjectMeta(),
				ObjectNew: newRepl,
				MetaNew:   newRepl.GetObjectMeta(),
			}

			instance2.Update(evt2, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "foo",
					Name:      "faz",
				},
			}))
		})
		It("should enqueue a Request to the owner resource when the annotations are applied in child object"+
			"in the Create Event", func() {
			var repl *appsv1.ReplicaSet

			repl = &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}

			err := SetOwnerAnnotation(podOwner, repl, schema.GroupKind{Group: "Pods", Kind: "core"})
			Expect(err).To(BeNil())

			evt := event.CreateEvent{
				Object: repl,
				Meta:   repl.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})

		It("should enqueue a Request with the annotations of the object in the GenericEvent", func() {
			evt := event.GenericEvent{
				Object: pod,
				Meta:   pod.GetObjectMeta(),
			}
			instance.Generic(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})

		It("should not enqueue a request if there are no annotations matching with the object", func() {
			var repl *appsv1.ReplicaSet

			repl = &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}

			evt := event.CreateEvent{
				Object: repl,
				Meta:   repl.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))
		})

		It("should not enqueue a Request if there is no Namespace and name annotation matching the specified object", func() {
			var repl *appsv1.ReplicaSet

			repl = &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
					Annotations: map[string]string{
						TypeAnnotation: schema.GroupKind{Group: "Pods", Kind: "core"}.String(),
					},
				},
			}

			evt := event.CreateEvent{
				Object: repl,
				Meta:   repl.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))
		})

		It("should not enqueue a Request if there is no TypeAnnotation matching Group and Kind", func() {
			var repl *appsv1.ReplicaSet

			repl = &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",

					Annotations: map[string]string{
						NamespacedNameAnnotation: "AppService",
					},
				},
			}

			evt := event.CreateEvent{
				Object: repl,
				Meta:   repl.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))
		})

		It("should enqueue a Request if there are no Namespace annotation matching for the object", func() {
			var repl *appsv1.ReplicaSet

			repl = &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
					Annotations: map[string]string{
						NamespacedNameAnnotation: "AppService",
						TypeAnnotation:           schema.GroupKind{Group: "Pods", Kind: "core"}.String(),
					},
				},
			}

			evt := event.CreateEvent{
				Object: repl,
				Meta:   repl.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: "", Name: "AppService"}}))
		})

		It("should enqueue a Request for a object that is cluster scoped which has the annotations", func() {
			var nd *corev1.Node

			nd = &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Annotations: map[string]string{
						NamespacedNameAnnotation: "myapp",
						TypeAnnotation:           schema.GroupKind{Group: "ReplicaSet", Kind: "apps"}.String(),
					},
				},
			}

			instance = EnqueueRequestForAnnotation{Type: schema.GroupKind{Group: "ReplicaSet", Kind: "apps"}}

			evt := event.CreateEvent{
				Object: nd,
				Meta:   nd.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: "", Name: "myapp"}}))
		})

		It("should not enqueue a Request for a object that is cluster scoped which does not have annotations", func() {
			var nd *corev1.Node

			nd = &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			}

			instance = EnqueueRequestForAnnotation{Type: nd.GetObjectKind().GroupVersionKind().GroupKind()}
			evt := event.CreateEvent{
				Object: nd,
				Meta:   nd.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))
		})
	})

	Describe("EnqueueRequestForAnnotation.SetWatchOwnerAnnotation", func() {
		It("should add the watch owner annotations without losing existing ones", func() {
			var nd *corev1.Node
			nd = &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Annotations: map[string]string{
						"my-test-annotation": "should-keep",
					},
				},
			}

			err := SetOwnerAnnotation(podOwner, nd, schema.GroupKind{Group: "Pods", Kind: "core"})
			Expect(err).To(BeNil())

			expected := map[string]string{
				"my-test-annotation":     "should-keep",
				NamespacedNameAnnotation: fmt.Sprintf("%v/%v", podOwner.GetNamespace(), podOwner.GetName()),
				TypeAnnotation:           schema.GroupKind{Group: "Pods", Kind: "core"}.String(),
			}

			Expect(len(nd.GetAnnotations())).To(Equal(3))
			Expect(nd.GetAnnotations()).To(Equal(expected))
		})

		It("should return error when the owner Group or Kind is not informed", func() {
			var nd *corev1.Node
			nd = &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
				},
			}

			err := SetOwnerAnnotation(podOwner, nd, schema.GroupKind{Group: "", Kind: "core"})
			Expect(err).NotTo(BeNil())

			err = SetOwnerAnnotation(podOwner, nd, schema.GroupKind{Group: "Pod", Kind: ""})
			Expect(err).NotTo(BeNil())
		})
	})
})
