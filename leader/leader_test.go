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

package leader

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var _ = Describe("Leader election", func() {
	Describe("Become", func() {
		var client crclient.Client
		BeforeEach(func() {
			client = fake.NewClientBuilder().WithObjects(
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "leader-test",
						Namespace: "testns",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "v1",
								Kind:       "Pod",
								Name:       "leader-test",
							},
						},
					},
				},
				&corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "ConfigMap",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "leader-test",
						Namespace: "testns",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "v1",
								Kind:       "Pod",
								Name:       "leader-test",
							},
						},
					},
				},
			).Build()
		})
		It("should return an error when POD_NAME is not set", func() {
			os.Unsetenv("POD_NAME")
			Expect(Become(context.TODO(), "leader-test")).ShouldNot(Succeed())
		})
		It("should return an ErrNoNamespace", func() {
			os.Setenv("POD_NAME", "leader-test")
			readNamespace = func() (string, error) {
				return "", ErrNoNamespace
			}
			err := Become(context.TODO(), "leader-test", WithClient(client))
			Expect(err).Should(MatchError(ErrNoNamespace))
		})
		It("should not return an error", func() {
			os.Setenv("POD_NAME", "leader-test")
			readNamespace = func() (string, error) {
				return "testns", nil
			}

			Expect(Become(context.TODO(), "leader-test", WithClient(client))).To(Succeed())
		})
		It("should become leader when pod is evicted and rescheduled", func() {
			evictedPodStatusClient := fake.NewClientBuilder().WithObjects(
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "leader-test-new",
						Namespace: "testns",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "v1",
								Kind:       "Pod",
								Name:       "leader-test-new",
							},
						},
					},
				},
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "leader-test",
						Namespace: "testns",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "v1",
								Kind:       "Pod",
								Name:       "leader-test",
							},
						},
					},
					Status: corev1.PodStatus{
						Phase:  corev1.PodFailed,
						Reason: "Evicted",
					},
				},
				&corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "ConfigMap",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "leader-test",
						Namespace: "testns",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "v1",
								Kind:       "Pod",
								Name:       "leader-test",
							},
						},
					},
				},
			).WithInterceptorFuncs(
				interceptor.Funcs{
					// Mock garbage collection of the ConfigMap when the Pod is deleted.
					Delete: func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, _ ...crclient.DeleteOption) error {
						if obj.GetObjectKind() != nil && obj.GetObjectKind().GroupVersionKind().Kind == "Pod" && obj.GetName() == "leader-test" {
							cm := &corev1.ConfigMap{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "leader-test",
									Namespace: "testns",
								},
							}

							err := client.Delete(ctx, cm)
							if err != nil {
								return err
							}
						}
						return nil
					},
				},
			).Build()

			os.Setenv("POD_NAME", "leader-test-new")
			readNamespace = func() (string, error) {
				return "testns", nil
			}

			Expect(Become(context.TODO(), "leader-test", WithClient(evictedPodStatusClient))).To(Succeed())
		})
		It("should become leader when pod is preempted and rescheduled", func() {
			preemptedPodStatusClient := fake.NewClientBuilder().WithObjects(
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "leader-test-new",
						Namespace: "testns",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "v1",
								Kind:       "Pod",
								Name:       "leader-test-new",
							},
						},
					},
				},
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "leader-test",
						Namespace: "testns",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "v1",
								Kind:       "Pod",
								Name:       "leader-test",
							},
						},
					},
					Status: corev1.PodStatus{
						Phase:  corev1.PodFailed,
						Reason: "Preempting",
					},
				},
				&corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "ConfigMap",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "leader-test",
						Namespace: "testns",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "v1",
								Kind:       "Pod",
								Name:       "leader-test",
							},
						},
					},
				},
			).WithInterceptorFuncs(
				interceptor.Funcs{
					// Mock garbage collection of the ConfigMap when the Pod is deleted.
					Delete: func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, _ ...crclient.DeleteOption) error {
						if obj.GetObjectKind() != nil && obj.GetObjectKind().GroupVersionKind().Kind == "Pod" && obj.GetName() == "leader-test" {
							cm := &corev1.ConfigMap{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "leader-test",
									Namespace: "testns",
								},
							}

							err := client.Delete(ctx, cm)
							if err != nil {
								return err
							}
						}
						return nil
					},
				},
			).Build()

			os.Setenv("POD_NAME", "leader-test-new")
			readNamespace = func() (string, error) {
				return "testns", nil
			}

			Expect(Become(context.TODO(), "leader-test", WithClient(preemptedPodStatusClient))).To(Succeed())
		})
	})
	Describe("isPodEvicted", func() {
		var leaderPod *corev1.Pod
		BeforeEach(func() {
			leaderPod = &corev1.Pod{}
		})
		It("should return false with an empty status", func() {
			Expect(isPodEvicted(*leaderPod)).To(BeFalse())
		})
		It("should return false if reason is incorrect", func() {
			leaderPod.Status.Phase = corev1.PodFailed
			leaderPod.Status.Reason = "invalid"
			Expect(isPodEvicted(*leaderPod)).To(BeFalse())
		})
		It("should return false if pod is in the wrong phase", func() {
			leaderPod.Status.Phase = corev1.PodRunning
			Expect(isPodEvicted(*leaderPod)).To(BeFalse())
		})
		It("should return true when Phase and Reason are set", func() {
			leaderPod.Status.Phase = corev1.PodFailed
			leaderPod.Status.Reason = "Evicted"
			Expect(isPodEvicted(*leaderPod)).To(BeTrue())
		})
	})
	Describe("isPodPreempted", func() {
		var leaderPod *corev1.Pod
		BeforeEach(func() {
			leaderPod = &corev1.Pod{}
		})
		It("should return false with an empty status", func() {
			Expect(isPodPreempted(*leaderPod)).To(BeFalse())
		})
		It("should return false if reason is incorrect", func() {
			leaderPod.Status.Phase = corev1.PodFailed
			leaderPod.Status.Reason = "invalid"
			Expect(isPodPreempted(*leaderPod)).To(BeFalse())
		})
		It("should return false if pod is in the wrong phase", func() {
			leaderPod.Status.Phase = corev1.PodRunning
			Expect(isPodPreempted(*leaderPod)).To(BeFalse())
		})
		It("should return true when Phase and Reason are set", func() {
			leaderPod.Status.Phase = corev1.PodFailed
			leaderPod.Status.Reason = "Preempting"
			Expect(isPodPreempted(*leaderPod)).To(BeTrue())
		})
	})
	Describe("myOwnerRef", func() {
		var client crclient.Client
		BeforeEach(func() {
			client = fake.NewClientBuilder().WithObjects(
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mypod",
						Namespace: "testns",
					},
				},
			).Build()
		})
		It("should return an error when POD_NAME is not set", func() {
			os.Unsetenv("POD_NAME")
			_, err := myOwnerRef(context.TODO(), client, "")
			Expect(err).Should(HaveOccurred())
		})
		It("should return an error if no pod is found", func() {
			os.Setenv("POD_NAME", "thisisnotthepodyourelookingfor")
			_, err := myOwnerRef(context.TODO(), client, "")
			Expect(err).Should(HaveOccurred())
		})
		It("should return the owner reference without error", func() {
			os.Setenv("POD_NAME", "mypod")
			owner, err := myOwnerRef(context.TODO(), client, "testns")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(owner.APIVersion).To(Equal("v1"))
			Expect(owner.Kind).To(Equal("Pod"))
			Expect(owner.Name).To(Equal("mypod"))
		})
	})
	Describe("getPod", func() {
		var client crclient.Client
		BeforeEach(func() {
			client = fake.NewClientBuilder().WithObjects(
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mypod",
						Namespace: "testns",
					},
				},
			).Build()
		})
		It("should return an error when POD_NAME is not set", func() {
			os.Unsetenv("POD_NAME")
			_, err := getPod(context.TODO(), nil, "")
			Expect(err).Should(HaveOccurred())
		})
		It("should return an error if no pod is found", func() {
			os.Setenv("POD_NAME", "thisisnotthepodyourelookingfor")
			_, err := getPod(context.TODO(), client, "")
			Expect(err).Should(HaveOccurred())
		})
		It("should return the pod with the given name", func() {
			os.Setenv("POD_NAME", "mypod")
			pod, err := getPod(context.TODO(), client, "testns")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(pod).ShouldNot(BeNil())
			Expect(pod.TypeMeta.APIVersion).To(Equal("v1"))
			Expect(pod.TypeMeta.Kind).To(Equal("Pod"))
		})
	})

	Describe("getNode", func() {
		var client crclient.Client
		BeforeEach(func() {
			client = fake.NewClientBuilder().WithObjects(
				&corev1.Node{
					TypeMeta: metav1.TypeMeta{
						APIVersion: schema.GroupVersion{
							Group:   corev1.SchemeGroupVersion.Group,
							Version: corev1.SchemeGroupVersion.Version,
						}.String(),
						Kind: "Node",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "mynode",
					},
				},
			).Build()
		})
		It("should return an error if no node is found", func() {
			node := corev1.Node{}
			Expect(getNode(context.TODO(), client, "", &node)).ToNot(Succeed())
		})
		It("should return the node with the given name", func() {
			node := corev1.Node{}
			Expect(getNode(context.TODO(), client, "mynode", &node)).Should(Succeed())
			Expect(node.TypeMeta.APIVersion).To(Equal("v1"))
			Expect(node.TypeMeta.Kind).To(Equal("Node"))
		})
	})

	Describe("isNotReadyNode", func() {
		var (
			nodeName string
			node     *corev1.Node
			client   crclient.Client
		)
		BeforeEach(func() {
			nodeName = "mynode"
			node = &corev1.Node{
				TypeMeta: metav1.TypeMeta{
					APIVersion: schema.GroupVersion{
						Group:   corev1.SchemeGroupVersion.Group,
						Version: corev1.SchemeGroupVersion.Version,
					}.String(),
					Kind: "Node",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				},
				Status: corev1.NodeStatus{
					Conditions: make([]corev1.NodeCondition, 1),
				},
			}
		})

		It("should return false if node is invalid", func() {
			client = fake.NewClientBuilder().WithObjects().Build()
			ret := isNotReadyNode(context.TODO(), client, "")
			Expect(ret).To(BeFalse())
		})
		It("should return false if no NodeCondition is found", func() {
			client = fake.NewClientBuilder().WithObjects(node).Build()
			ret := isNotReadyNode(context.TODO(), client, nodeName)
			Expect(ret).To(BeFalse())
		})
		It("should return false if type is incorrect", func() {
			node.Status.Conditions[0].Type = corev1.NodeMemoryPressure
			node.Status.Conditions[0].Status = corev1.ConditionFalse
			client = fake.NewClientBuilder().WithObjects(node).Build()
			ret := isNotReadyNode(context.TODO(), client, nodeName)
			Expect(ret).To(BeFalse())
		})
		It("should return false if NodeReady's type is true", func() {
			node.Status.Conditions[0].Type = corev1.NodeReady
			node.Status.Conditions[0].Status = corev1.ConditionTrue
			client = fake.NewClientBuilder().WithObjects(node).Build()
			ret := isNotReadyNode(context.TODO(), client, nodeName)
			Expect(ret).To(BeFalse())
		})
		It("should return true when Type is set and Status is set to false", func() {
			node.Status.Conditions[0].Type = corev1.NodeReady
			node.Status.Conditions[0].Status = corev1.ConditionFalse
			client = fake.NewClientBuilder().WithObjects(node).Build()
			ret := isNotReadyNode(context.TODO(), client, nodeName)
			Expect(ret).To(BeTrue())
		})
	})
	Describe("deleteLeader", func() {
		var (
			configmap *corev1.ConfigMap
			pod       *corev1.Pod
			client    crclient.Client
		)
		BeforeEach(func() {
			pod = &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: schema.GroupVersion{
						Group:   corev1.SchemeGroupVersion.Group,
						Version: corev1.SchemeGroupVersion.Version,
					}.String(),
					Kind: "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "leader-test",
					Namespace: "testns",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "Pod",
							Name:       "leader-test",
						},
					},
				},
			}
			configmap = &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: schema.GroupVersion{
						Group:   corev1.SchemeGroupVersion.Group,
						Version: corev1.SchemeGroupVersion.Version,
					}.String(),
					Kind: "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "leader-test",
					Namespace: "testns",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "Pod",
							Name:       "leader-test",
						},
					},
				},
			}
		})
		It("should return an error if existing is not found", func() {
			client = fake.NewClientBuilder().WithObjects(pod).Build()
			Expect(deleteLeader(context.TODO(), client, pod, configmap)).ToNot(Succeed())
		})
		It("should return an error if pod is not found", func() {
			client = fake.NewClientBuilder().WithObjects(configmap).Build()
			Expect(deleteLeader(context.TODO(), client, pod, configmap)).ToNot(Succeed())
		})
		It("should return an error if pod is nil", func() {
			client = fake.NewClientBuilder().WithObjects(pod, configmap).Build()
			Expect(deleteLeader(context.TODO(), client, nil, configmap)).ToNot(Succeed())
		})
		It("should return an error if configmap is nil", func() {
			client = fake.NewClientBuilder().WithObjects(pod, configmap).Build()
			Expect(deleteLeader(context.TODO(), client, pod, nil)).ToNot(Succeed())
		})
		It("should return nil if pod and configmap exists and configmap's owner is the pod", func() {
			client = fake.NewClientBuilder().WithObjects(pod, configmap).Build()
			Expect(deleteLeader(context.TODO(), client, pod, configmap)).To(Succeed())
		})
	})
})
