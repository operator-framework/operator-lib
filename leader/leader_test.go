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
	"errors"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Leader election", func() {

	Describe("Become", func() {
		var (
			client crclient.Client
		)
		BeforeEach(func() {
			client = fake.NewClientBuilder().WithObjects(
				&corev1.Pod{
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
			err := Become(context.TODO(), "leader-test")
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an ErrNoNamespace", func() {
			os.Setenv("POD_NAME", "leader-test")
			readNamespace = func() (string, error) {
				return "", ErrNoNamespace
			}
			err := Become(context.TODO(), "leader-test", WithClient(client))
			Expect(err).ShouldNot(BeNil())
			Expect(err).To(Equal(ErrNoNamespace))
			Expect(errors.Is(err, ErrNoNamespace)).To(Equal(true))
		})
		It("should not return an error", func() {
			os.Setenv("POD_NAME", "leader-test")
			readNamespace = func() (string, error) {
				return "testns", nil
			}

			err := Become(context.TODO(), "leader-test", WithClient(client))
			Expect(err).Should(BeNil())
		})
	})
	Describe("isPodEvicted", func() {
		var (
			leaderPod *corev1.Pod
		)
		BeforeEach(func() {
			leaderPod = &corev1.Pod{}
		})
		It("should return false with an empty status", func() {
			Expect(isPodEvicted(*leaderPod)).To(Equal(false))
		})
		It("should return false if reason is incorrect", func() {
			leaderPod.Status.Phase = corev1.PodFailed
			leaderPod.Status.Reason = "invalid"
			Expect(isPodEvicted(*leaderPod)).To(Equal(false))
		})
		It("should return false if pod is in the wrong phase", func() {
			leaderPod.Status.Phase = corev1.PodRunning
			Expect(isPodEvicted(*leaderPod)).To(Equal(false))
		})
		It("should return true when Phase and Reason are set", func() {
			leaderPod.Status.Phase = corev1.PodFailed
			leaderPod.Status.Reason = "Evicted"
			Expect(isPodEvicted(*leaderPod)).To(Equal(true))
		})
	})
	Describe("myOwnerRef", func() {
		var (
			client crclient.Client
		)
		BeforeEach(func() {
			client = fake.NewClientBuilder().WithObjects(
				&corev1.Pod{
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
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error if no pod is found", func() {
			os.Setenv("POD_NAME", "thisisnotthepodyourelookingfor")
			_, err := myOwnerRef(context.TODO(), client, "")
			Expect(err).ShouldNot(BeNil())
		})
		It("should return the owner reference without error", func() {
			os.Setenv("POD_NAME", "mypod")
			owner, err := myOwnerRef(context.TODO(), client, "testns")
			Expect(err).Should(BeNil())
			Expect(owner.APIVersion).To(Equal("v1"))
			Expect(owner.Kind).To(Equal("Pod"))
			Expect(owner.Name).To(Equal("mypod"))
		})
	})
	Describe("getPod", func() {
		var (
			client crclient.Client
		)
		BeforeEach(func() {
			client = fake.NewClientBuilder().WithObjects(
				&corev1.Pod{
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
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error if no pod is found", func() {
			os.Setenv("POD_NAME", "thisisnotthepodyourelookingfor")
			_, err := getPod(context.TODO(), client, "")
			Expect(err).ShouldNot(BeNil())
		})
		It("should return the pod with the given name", func() {
			os.Setenv("POD_NAME", "mypod")
			pod, err := getPod(context.TODO(), client, "testns")
			Expect(err).Should(BeNil())
			Expect(pod).ShouldNot(BeNil())
			Expect(pod.TypeMeta.APIVersion).To(Equal("v1"))
			Expect(pod.TypeMeta.Kind).To(Equal("Pod"))
		})
	})

	Describe("getNode", func() {
		var (
			client crclient.Client
		)
		BeforeEach(func() {
			client = fake.NewClientBuilder().WithObjects(
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mynode",
					},
				},
			).Build()
		})
		It("should return an error if no node is found", func() {
			node := corev1.Node{}
			err := getNode(context.TODO(), client, "", &node)
			Expect(err).ShouldNot(BeNil())
		})
		It("should return the node with the given name", func() {
			node := corev1.Node{}
			err := getNode(context.TODO(), client, "mynode", &node)
			Expect(err).Should(BeNil())
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
			Expect(ret).To(Equal(false))
		})
		It("should return false if no NodeCondition is found", func() {
			client = fake.NewClientBuilder().WithObjects(node).Build()
			ret := isNotReadyNode(context.TODO(), client, nodeName)
			Expect(ret).To(Equal(false))
		})
		It("should return false if type is incorrect", func() {
			node.Status.Conditions[0].Type = corev1.NodeMemoryPressure
			node.Status.Conditions[0].Status = corev1.ConditionFalse
			client = fake.NewClientBuilder().WithObjects(node).Build()
			ret := isNotReadyNode(context.TODO(), client, nodeName)
			Expect(ret).To(Equal(false))
		})
		It("should return false if NodeReady's type is true", func() {
			node.Status.Conditions[0].Type = corev1.NodeReady
			node.Status.Conditions[0].Status = corev1.ConditionTrue
			client = fake.NewClientBuilder().WithObjects(node).Build()
			ret := isNotReadyNode(context.TODO(), client, nodeName)
			Expect(ret).To(Equal(false))
		})
		It("should return true when Type is set and Status is set to false", func() {
			node.Status.Conditions[0].Type = corev1.NodeReady
			node.Status.Conditions[0].Status = corev1.ConditionFalse
			client = fake.NewClientBuilder().WithObjects(node).Build()
			ret := isNotReadyNode(context.TODO(), client, nodeName)
			Expect(ret).To(Equal(true))
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
			err := deleteLeader(context.TODO(), client, pod, configmap)
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error if pod is not found", func() {
			client = fake.NewClientBuilder().WithObjects(configmap).Build()
			err := deleteLeader(context.TODO(), client, pod, configmap)
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error if pod is nil", func() {
			client = fake.NewClientBuilder().WithObjects(pod, configmap).Build()
			err := deleteLeader(context.TODO(), client, nil, configmap)
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error if configmap is nil", func() {
			client = fake.NewClientBuilder().WithObjects(pod, configmap).Build()
			err := deleteLeader(context.TODO(), client, pod, nil)
			Expect(err).ShouldNot(BeNil())
		})
		It("should return nil if pod and configmap exists and configmap's owner is the pod", func() {
			client = fake.NewClientBuilder().WithObjects(pod, configmap).Build()
			err := deleteLeader(context.TODO(), client, pod, configmap)
			Expect(err).Should(BeNil())
		})

	})
})
