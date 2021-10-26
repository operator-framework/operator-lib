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
	"fmt"
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceInfo describes the Kube resources that we are about to consider
// when pruning resources
type ResourceInfo struct {
	Name      string
	Kind      ResourceKind
	Namespace string
	StartTime time.Time
}

func (config Config) GetSucceededPods() (resources []ResourceInfo, err error) {

	listOptions := metav1.ListOptions{LabelSelector: config.LabelSelector}
	for n := 0; n < len(config.Namespaces); n++ {
		pods, err := config.Clientset.CoreV1().Pods(config.Namespaces[n]).List(config.Ctx, listOptions)
		if err != nil {
			return resources, err
		}
		fmt.Printf("got %d pods\n", len(pods.Items))

		for i := 0; i < len(pods.Items); i++ {
			p := pods.Items[i]
			switch p.Status.Phase {
			case v1.PodRunning:
			case v1.PodPending:
			case v1.PodFailed:
			case v1.PodUnknown:
			case v1.PodSucceeded:
				// currently we only care to prune succeeded pods
				resources = append(resources, ResourceInfo{
					Name:      p.Name,
					Kind:      PodKind,
					Namespace: config.Namespaces[n],
					StartTime: p.Status.StartTime.Time,
				})
			default:
			}
		}
	}

	// sort by StartTime, earliest first order
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].StartTime.After(resources[j].StartTime)
	})

	return resources, nil
}

func (config Config) getCompletedJobs() (resources []ResourceInfo, err error) {

	listOptions := metav1.ListOptions{LabelSelector: config.LabelSelector}

	for n := 0; n < len(config.Namespaces); n++ {
		jobs, err := config.Clientset.BatchV1().Jobs(config.Namespaces[n]).List(config.Ctx, listOptions)
		if err != nil {
			return resources, err
		}
		log.V(1).Info("jobs", "count", len(jobs.Items))
		for i := 0; i < len(jobs.Items); i++ {
			j := jobs.Items[i]
			if j.Status.CompletionTime != nil {
				// currently we only care to prune succeeded pods
				resources = append(resources, ResourceInfo{
					Name:      j.Name,
					Kind:      JobKind,
					Namespace: config.Namespaces[n],
					StartTime: j.Status.StartTime.Time,
				})
			}
		}
	}

	// sort by StartTime, earliest first order
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].StartTime.After(resources[j].StartTime)
	})

	return resources, nil
}
