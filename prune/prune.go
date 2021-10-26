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

package prune

import (
	"context"
	"fmt"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// ResourceKind describes the Kubernetes Kind we are wanting to prune
type ResourceKind string

// ResourceStatus describes the Kubernetes resource status we are evaluating
type ResourceStatus string

// Strategy describes the pruning strategy we want to employ
type Strategy string

const (
	// MaxAgeStrategy maximum age of a resource that is desired, Duration
	MaxAgeStrategy Strategy = "MaxAge"
	// MaxCountStrategy maximum number of a resource that is desired, int
	MaxCountStrategy Strategy = "MaxCount"
	// JobKind equates to a Kube Job resource kind
	JobKind ResourceKind = "job"
	// PodKind equates to a Kube Pod resource kind
	PodKind ResourceKind = "pod"
)

// StrategyConfig holds settings unique to each pruning mode
type StrategyConfig struct {
	Mode            Strategy
	MaxAgeSetting   string
	MaxCountSetting int
}

// PruneStrategyFunction function allows a means to specify
// custom prune strategies
type StrategyImplementation func(resources []ResourceInfo) error

// PreDelete function is called before a resource is pruned
type PreDelete func(cfg Config, something ResourceInfo) error

// Config defines a pruning configuration and ultimately
// determines what will get pruned
type Config struct {
	Ctx           context.Context
	Clientset     kubernetes.Interface
	LabelSelector string         //selector resources to prune
	DryRun        bool           //true only performs a check, not removals
	Resources     []ResourceKind //pods, jobs are supported
	Namespaces    []string       //empty means all namespaces
	Strategy      StrategyConfig //strategy for pruning, either age or max
	PreDeleteHook PreDelete      //called before resource is deleteds
}

var log = logf.Log.WithName("prune")

// Execute causes the pruning work to be executed based on its configuration
func (config Config) Execute() error {

	log.V(1).Info("Execute Prune")

	err := config.validate()
	if err != nil {
		return err
	}

	// get a sorted list, by StartTime,  of pods and jobs
	if contains(config.Resources, PodKind) {
		podList, err := config.GetSucceededPods()
		if err != nil {
			return err
		}

		log.V(1).Info("pods ", "count", len(podList))

		switch config.Strategy.Mode {
		case MaxAgeStrategy:
			err = config.pruneByMaxAge(podList)
			if err != nil {
				return err
			}
		case MaxCountStrategy:
			err = config.pruneByMaxCount(podList)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown strategy")
		}
	}
	if contains(config.Resources, JobKind) {
		jobList, err := config.getCompletedJobs()
		if err != nil {
			return err
		}
		log.V(1).Info("jobs count", len(jobList))
		switch config.Strategy.Mode {
		case MaxAgeStrategy:
			err = config.pruneByMaxAge(jobList)
			if err != nil {
				return err
			}
		case MaxCountStrategy:
			err = config.pruneByMaxCount(jobList)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown strategy")
		}
	}

	log.V(1).Info("Prune completed")

	return nil
}

// contains checks if a ResourceKind is present in a slice
func contains(s []ResourceKind, str ResourceKind) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// containsString checks if a string is present in a slice
func containsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func (config Config) validate() (err error) {
	if len(config.Namespaces) == 0 {
		return fmt.Errorf("namespaces are required")
	}

	if containsString(config.Namespaces, "") {
		return fmt.Errorf("empty namespace value not supported")
	}

	_, err = labels.Parse(config.LabelSelector)
	if err != nil {
		return err
	}

	if config.Strategy.Mode == MaxAgeStrategy {
		_, err = time.ParseDuration(config.Strategy.MaxAgeSetting)
		if err != nil {
			return err
		}
	}
	if config.Strategy.Mode == MaxCountStrategy {
		if config.Strategy.MaxCountSetting < 0 {
			return fmt.Errorf("max count is required to be greater than or equal to 0")
		}
	}
	return nil
}
