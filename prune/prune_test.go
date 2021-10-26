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

package prune

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Prune", func() {

	Describe("config validation", func() {
		BeforeEach(func() {

		})
		It("should return an error when LabelSelector is not set", func() {
			cfg := Config{}
			err := cfg.Execute()
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error is Namespaces is empty", func() {
			cfg := Config{}
			cfg.LabelSelector = "app=churro"
			err := cfg.Execute()
			Expect(err).ShouldNot(BeNil())
		})
	})
	Describe("test maxCount strategy", func() {
		var (
			client kubernetes.Interface
			cfg    Config
		)
		BeforeEach(func() {
			client = testclient.NewSimpleClientset()

			cfg = Config{
				Ctx:           context.Background(),
				DryRun:        false,
				Clientset:     client,
				LabelSelector: "app=churro",
				Resources:     []ResourceKind{PodKind},
				Namespaces:    []string{"default"},
				Strategy: StrategyConfig{
					//Mode: MaxAgeStrategy,
					Mode: MaxCountStrategy,
					//MaxAgeSetting: "30m",
					MaxCountSetting: 1,
				},
				PreDeleteHook: myhook,
			}

			_ = createTestPods(client)
		})
		It("test maxCount strategy", func() {
			err := cfg.Execute()
			Expect(err).Should(BeNil())
			var pods []ResourceInfo
			pods, err = cfg.getSucceededPods()
			Expect(err).Should(BeNil())
			Expect(len(pods)).To(Equal(1))
		})
		It("test maxAge strategy", func() {
			cfg.Strategy.Mode = MaxAgeStrategy
			cfg.Strategy.MaxAgeSetting = "3h"
			err := cfg.Execute()
			Expect(err).Should(BeNil())
			var pods []ResourceInfo
			pods, err = cfg.getSucceededPods()
			Expect(err).Should(BeNil())
			Expect(len(pods)).To(Equal(2))
		})
		It("test custom strategy", func() {
			cfg.Strategy.Mode = CustomStrategy
			cfg.Strategy.CustomSettings = make(map[string]interface{})
			cfg.CustomStrategy = myStrategy
			err := cfg.Execute()
			Expect(err).Should(BeNil())
			var pods []ResourceInfo
			pods, err = cfg.getSucceededPods()
			Expect(err).Should(BeNil())
			Expect(len(pods)).To(Equal(3))
		})
	})
})

// create 3 pods with different start times (now, 2 days old, 4 days old)
func createTestPods(client kubernetes.Interface) (err error) {
	// some defaults
	ns := "default"
	labels := make(map[string]string)
	labels["app"] = "churro"

	// delete any existing pods
	_ = client.CoreV1().Pods(ns).Delete(context.TODO(), "churro1", metav1.DeleteOptions{})
	_ = client.CoreV1().Pods(ns).Delete(context.TODO(), "churro2", metav1.DeleteOptions{})
	_ = client.CoreV1().Pods(ns).Delete(context.TODO(), "churro3", metav1.DeleteOptions{})

	// create 3 pods with different StartTimes
	now := time.Now() //initial start time
	startTime := metav1.NewTime(now)
	p1 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "churro1",
			Namespace: ns,
			Labels:    labels,
		},
		Status: v1.PodStatus{
			Phase:     v1.PodSucceeded,
			StartTime: &startTime,
		},
	}
	_, err = client.CoreV1().Pods(ns).Create(context.TODO(), p1, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	twoHoursPriorToNow := now.Add(time.Hour * time.Duration(-2))
	// create start time 2 hours before now
	startTime = metav1.NewTime(twoHoursPriorToNow)
	p2 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "churro2",
			Namespace: ns,
			Labels:    labels,
		},
		Status: v1.PodStatus{
			Phase:     v1.PodSucceeded,
			StartTime: &startTime,
		},
	}
	_, err = client.CoreV1().Pods(ns).Create(context.TODO(), p2, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	// create start time 4 hours before now
	fourHoursPriorToNow := now.Add(time.Hour * time.Duration(-4))
	startTime = metav1.NewTime(fourHoursPriorToNow)
	p3 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "churro3",
			Namespace: ns,
			Labels:    labels,
		},
		Status: v1.PodStatus{
			Phase:     v1.PodSucceeded,
			StartTime: &startTime,
		},
	}
	_, err = client.CoreV1().Pods(ns).Create(context.TODO(), p3, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func myhook(cfg Config, x ResourceInfo) error {
	fmt.Printf("myhook is called with resource %v strategy %v\n", x, cfg)
	return nil
}

// myStrategy shows how you can write your own strategy, in this
// example, the strategy doesn't really do another other than count
// the number of resources
func myStrategy(cfg Config, resources []ResourceInfo) error {
	fmt.Printf("myStrategy is called with resources %v config %v\n", resources, cfg)
	if len(resources) != 3 {
		return fmt.Errorf("count of resources did not equal our expectation")
	}
	return nil
}
