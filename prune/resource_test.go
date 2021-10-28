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
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("Prune", func() {
	Describe("test pods", func() {
		var (
			client kubernetes.Interface
			cfg    Config
			ctx    context.Context
		)
		BeforeEach(func() {
			client = testclient.NewSimpleClientset()
			ctx = context.Background()
			cfg = Config{
				log:           logf.Log.WithName("prune"),
				DryRun:        false,
				Clientset:     client,
				LabelSelector: "app=churro",
				Resources: []schema.GroupVersionKind{
					{Group: "", Version: "", Kind: PodKind},
				},
				Namespaces: []string{"default"},
				Strategy: StrategyConfig{
					Mode:            MaxCountStrategy,
					MaxCountSetting: 1,
				},
				PreDeleteHook: myhook,
			}

			_ = createTestPods(client)
		})
		It("test pod maxCount strategy", func() {
			err := cfg.Execute(ctx)
			Expect(err).Should(BeNil())
			var pods []ResourceInfo
			pods, err = cfg.getSucceededPods(ctx)
			Expect(err).Should(BeNil())
			Expect(len(pods)).To(Equal(1))
			Expect(containsName(pods, "churro1")).To(Equal(true))
		})
		It("test pod maxAge strategy", func() {
			cfg.Strategy.Mode = MaxAgeStrategy
			cfg.Strategy.MaxAgeSetting = "3h"
			err := cfg.Execute(ctx)
			Expect(err).Should(BeNil())
			var pods []ResourceInfo
			pods, err = cfg.getSucceededPods(ctx)
			Expect(err).Should(BeNil())
			Expect(containsName(pods, "churro1")).To(Equal(true))
			Expect(containsName(pods, "churro2")).To(Equal(true))
		})
		It("test pod custom strategy", func() {
			cfg.Strategy.Mode = CustomStrategy
			cfg.Strategy.CustomSettings = make(map[string]interface{})
			cfg.CustomStrategy = myStrategy
			err := cfg.Execute(ctx)
			Expect(err).Should(BeNil())
			var pods []ResourceInfo
			pods, err = cfg.getSucceededPods(ctx)
			Expect(err).Should(BeNil())
			Expect(len(pods)).To(Equal(3))
		})
	})

	Describe("config validation", func() {
		var (
			ctx context.Context
			cfg Config
		)
		BeforeEach(func() {
			cfg = Config{}
			cfg.log = logf.Log.WithName("prune")
			ctx = context.Background()
		})
		It("should return an error when LabelSelector is not set", func() {
			err := cfg.Execute(ctx)
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error is Namespaces is empty", func() {
			cfg.LabelSelector = "app=churro"
			err := cfg.Execute(ctx)
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error when labels dont parse", func() {
			cfg.Namespaces = []string{"one"}
			cfg.LabelSelector = "-"
			err := cfg.Execute(ctx)
			Expect(err).ShouldNot(BeNil())
		})
	})

	Describe("test jobs", func() {
		var (
			jobclient kubernetes.Interface
			jobcfg    Config
			ctx       context.Context
		)
		BeforeEach(func() {
			jobclient = testclient.NewSimpleClientset()

			ctx = context.Background()
			jobcfg = Config{
				DryRun:        false,
				log:           logf.Log.WithName("prune"),
				Clientset:     jobclient,
				LabelSelector: "app=churro",
				Resources: []schema.GroupVersionKind{
					{Group: "", Version: "", Kind: JobKind},
				},
				Namespaces: []string{"default"},
				Strategy: StrategyConfig{
					Mode:            MaxCountStrategy,
					MaxCountSetting: 1,
				},
				PreDeleteHook: myhook,
			}

			_ = createTestJobs(jobclient)
		})
		It("test job maxAge strategy", func() {
			jobcfg.Strategy.Mode = MaxAgeStrategy
			jobcfg.Strategy.MaxAgeSetting = "3h"
			err := jobcfg.Execute(ctx)
			Expect(err).Should(BeNil())
			var jobs []ResourceInfo
			jobs, err = jobcfg.getCompletedJobs(ctx)
			Expect(err).Should(BeNil())
			Expect(containsName(jobs, "churro1")).To(Equal(true))
			Expect(containsName(jobs, "churro2")).To(Equal(true))
		})
		It("test job maxCount strategy", func() {
			err := jobcfg.Execute(ctx)
			Expect(err).Should(BeNil())
			var jobs []ResourceInfo
			jobs, err = jobcfg.getCompletedJobs(ctx)
			Expect(err).Should(BeNil())
			Expect(len(jobs)).To(Equal(1))
			Expect(containsName(jobs, "churro1")).To(Equal(true))
		})
		It("test job custom strategy", func() {
			jobcfg.Strategy.Mode = CustomStrategy
			jobcfg.Strategy.CustomSettings = make(map[string]interface{})
			jobcfg.CustomStrategy = myStrategy
			err := jobcfg.Execute(ctx)
			Expect(err).Should(BeNil())
			var jobs []ResourceInfo
			jobs, err = jobcfg.getCompletedJobs(ctx)
			Expect(err).Should(BeNil())
			Expect(len(jobs)).To(Equal(3))
		})
	})
})

// create 3 jobs with different start times (now, 2 days old, 4 days old)
func createTestJobs(client kubernetes.Interface) (err error) {
	// some defaults
	ns := "default"
	labels := make(map[string]string)
	labels["app"] = "churro"

	// delete any existing jobs
	_ = client.BatchV1().Jobs(ns).Delete(context.TODO(), "churro1", metav1.DeleteOptions{})
	_ = client.BatchV1().Jobs(ns).Delete(context.TODO(), "churro2", metav1.DeleteOptions{})
	_ = client.BatchV1().Jobs(ns).Delete(context.TODO(), "churro3", metav1.DeleteOptions{})

	// create 3 jobs with different CompletionTime
	now := time.Now() //initial start time
	startTime := metav1.NewTime(now)
	j1 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "churro1",
			Namespace: ns,
			Labels:    labels,
		},
		Status: batchv1.JobStatus{
			CompletionTime: &startTime,
		},
	}
	_, err = client.BatchV1().Jobs(ns).Create(context.TODO(), j1, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	twoHoursPriorToNow := now.Add(time.Hour * time.Duration(-2))
	// create start time 2 hours before now
	startTime = metav1.NewTime(twoHoursPriorToNow)
	j2 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "churro2",
			Namespace: ns,
			Labels:    labels,
		},
		Status: batchv1.JobStatus{
			CompletionTime: &startTime,
		},
	}
	_, err = client.BatchV1().Jobs(ns).Create(context.TODO(), j2, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	// create start time 4 hours before now
	fourHoursPriorToNow := now.Add(time.Hour * time.Duration(-4))
	startTime = metav1.NewTime(fourHoursPriorToNow)
	j3 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "churro3",
			Namespace: ns,
			Labels:    labels,
		},
		Status: batchv1.JobStatus{
			CompletionTime: &startTime,
		},
	}
	_, err = client.BatchV1().Jobs(ns).Create(context.TODO(), j3, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// create 3 pods and 3 jobs with different start times (now, 2 days old, 4 days old)
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
	fmt.Println("myhook is called ")
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
