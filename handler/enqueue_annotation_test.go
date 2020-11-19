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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/util/workqueue"
)

var _ = Describe("EnqueueRequestForAnnotation", func() {
	var q workqueue.RateLimitingInterface
	var instance EnqueueRequestForAnnotation
	var pod *corev1.Pod
	var podOwner *corev1.Pod

	BeforeEach(func() {
		q = controllertest.Queue{Interface: workqueue.New()}
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "biz",
				Name:      "biz",
			},
		}
		podOwner = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "podOwnerNs",
				Name:      "podOwnerName",
			},
		}

		podOwner.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Kind: "Pod"})

		err := SetOwnerAnnotations(podOwner, pod)
		Expect(err).To(BeNil())
		instance = EnqueueRequestForAnnotation{
			Type: schema.GroupKind{
				Group: "",
				Kind:  "Pod",
			}}
	})

	Describe("Create", func() {
		It("should enqueue a Request with the annotations of the object in case of CreateEvent", func() {
			evt := event.CreateEvent{
				Object: pod,
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

		It("should enqueue a Request to the owner resource when the annotations are applied in child object"+
			" in the Create Event", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}

			err := SetOwnerAnnotations(podOwner, repl)
			Expect(err).To(BeNil())

			evt := event.CreateEvent{
				Object: repl,
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
		It("should not enqueue a request if there are no annotations matching with the object", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}

			evt := event.CreateEvent{
				Object: repl,
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))
		})
		It("should not enqueue a Request if there is no Namespace and name annotation matching the specified object are found", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
					Annotations: map[string]string{
						TypeAnnotation: schema.GroupKind{Group: "", Kind: "Pod"}.String(),
					},
				},
			}

			evt := event.CreateEvent{
				Object: repl,
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))
		})
		It("should not enqueue a Request if there is no TypeAnnotation matching the specified Group and Kind", func() {
			repl := &appsv1.ReplicaSet{
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
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))
		})
		It("should enqueue a Request if there are no Namespace annotation matching the object", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
					Annotations: map[string]string{
						NamespacedNameAnnotation: "AppService",
						TypeAnnotation:           schema.GroupKind{Group: "", Kind: "Pod"}.String(),
					},
				},
			}

			evt := event.CreateEvent{
				Object: repl,
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: "", Name: "AppService"}}))
		})
		It("should enqueue a Request for an object that is cluster scoped which has the annotations", func() {
			nd := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Annotations: map[string]string{
						NamespacedNameAnnotation: "myapp",
						TypeAnnotation:           schema.GroupKind{Group: "apps", Kind: "ReplicaSet"}.String(),
					},
				},
			}

			instance = EnqueueRequestForAnnotation{Type: schema.GroupKind{Group: "apps", Kind: "ReplicaSet"}}

			evt := event.CreateEvent{
				Object: nd,
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: "", Name: "myapp"}}))
		})
		It("should not enqueue a Request for an object that is cluster scoped which does not have annotations", func() {
			nd := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			}

			instance = EnqueueRequestForAnnotation{Type: nd.GetObjectKind().GroupVersionKind().GroupKind()}
			evt := event.CreateEvent{
				Object: nd,
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))
		})
	})

	Describe("Delete", func() {
		It("should enqueue a Request with the annotations of the object in case of DeleteEvent", func() {
			evt := event.DeleteEvent{
				Object: pod,
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
	})

	Describe("Update", func() {
		It("should enqueue a Request with annotations applied to both objects in UpdateEvent", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"

			err := SetOwnerAnnotations(podOwner, pod)
			Expect(err).To(BeNil())

			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newPod,
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
		It("should enqueue a Request with the annotations applied in one of the objects in case of UpdateEvent", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"
			newPod.Annotations = map[string]string{}

			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newPod,
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
		It("should enqueue a Request when the annotations are applied in a different resource in case of UpdateEvent", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}
			instance = EnqueueRequestForAnnotation{Type: schema.GroupKind{Group: "apps", Kind: "ReplicaSet"}}

			evt := event.CreateEvent{
				Object: repl,
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(0))

			newRepl := repl.DeepCopy()
			newRepl.Name = pod.Name + "2"
			newRepl.Namespace = pod.Namespace + "2"

			newRepl.Annotations = map[string]string{
				TypeAnnotation:           schema.GroupKind{Group: "apps", Kind: "ReplicaSet"}.String(),
				NamespacedNameAnnotation: "foo/faz",
			}

			instance2 := EnqueueRequestForAnnotation{Type: schema.GroupKind{Group: "apps", Kind: "ReplicaSet"}}

			evt2 := event.UpdateEvent{
				ObjectOld: repl,
				ObjectNew: newRepl,
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
		It("should enqueue multiple Update Requests when different annotations are applied to multiple objects", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"

			err := SetOwnerAnnotations(podOwner, pod)
			Expect(err).To(BeNil())

			var podOwner2 = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "podOwnerNsTest",
					Name:      "podOwnerNameTest",
				},
			}
			podOwner2.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Kind: "Pod"})

			err = SetOwnerAnnotations(podOwner2, newPod)
			Expect(err).To(BeNil())

			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newPod,
			}
			instance.Update(evt, q)
			Expect(q.Len()).To(Equal(2))
		})
	})

	Describe("Generic", func() {
		It("should enqueue a Request with the annotations of the object in case of GenericEvent", func() {
			evt := event.GenericEvent{
				Object: pod,
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
	})

	Describe("SetWatchOwnerAnnotation", func() {
		It("should add the watch owner annotations without losing existing ones", func() {
			nd := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Annotations: map[string]string{
						"my-test-annotation": "should-keep",
					},
				},
			}

			err := SetOwnerAnnotations(podOwner, nd)
			Expect(err).To(BeNil())

			expected := map[string]string{
				"my-test-annotation":     "should-keep",
				NamespacedNameAnnotation: "podOwnerNs/podOwnerName",
				TypeAnnotation:           schema.GroupKind{Group: "", Kind: "Pod"}.String(),
			}

			Expect(len(nd.GetAnnotations())).To(Equal(3))
			Expect(nd.GetAnnotations()).To(Equal(expected))
		})
		It("should return error when the owner Kind is not present", func() {
			nd := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
				},
			}

			podOwner.SetGroupVersionKind(schema.GroupVersionKind{Group: "Pod", Kind: ""})
			err := SetOwnerAnnotations(podOwner, nd)
			Expect(err).NotTo(BeNil())
		})
		It("should return an error when the owner Name is not set", func() {
			nd := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
				},
			}

			ownerNew := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "newpodOwnerNs",
				},
			}

			ownerNew.SetGroupVersionKind(schema.GroupVersionKind{Group: "Pod", Kind: ""})
			err := SetOwnerAnnotations(ownerNew, nd)
			Expect(err).NotTo(BeNil())
		})
	})
})
