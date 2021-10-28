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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

// ResourceStatus describes the Kubernetes resource status we are evaluating
type ResourceStatus string

// Strategy describes the pruning strategy we want to employ
type Strategy string

const (
	// CustomStrategy maximum age of a resource that is desired, Duration
	CustomStrategy Strategy = "Custom"
	// MaxAgeStrategy maximum age of a resource that is desired, Duration
	MaxAgeStrategy Strategy = "MaxAge"
	// MaxCountStrategy maximum number of a resource that is desired, int
	MaxCountStrategy Strategy = "MaxCount"
	// JobKind equates to a Kube Job resource kind
	JobKind string = "Job"
	// PodKind equates to a Kube Pod resource kind
	PodKind string = "Pod"
)

// StrategyConfig holds settings unique to each pruning mode
type StrategyConfig struct {
	Mode            Strategy
	MaxAgeSetting   string
	MaxCountSetting int
	CustomSettings  map[string]interface{}
}

// StrategyFunc function allows a means to specify
// custom prune strategies
type StrategyFunc func(cfg Config, resources []ResourceInfo) ([]ResourceInfo, error)

// PreDelete function is called before a resource is pruned
type PreDelete func(cfg Config, something ResourceInfo) error

// Config defines a pruning configuration and ultimately
// determines what will get pruned
type Config struct {
	Clientset      kubernetes.Interface      // kube client used by pruning
	LabelSelector  string                    //selector resources to prune
	DryRun         bool                      //true only performs a check, not removals
	Resources      []schema.GroupVersionKind //pods, jobs are supported
	Namespaces     []string                  //empty means all namespaces
	Strategy       StrategyConfig            //strategy for pruning, either age or max
	CustomStrategy StrategyFunc              //custom strategy
	PreDeleteHook  PreDelete                 //called before resource is deleteds
	log            logr.Logger
}

// Execute causes the pruning work to be executed based on its configuration
func (config Config) Execute(ctx context.Context) error {

	config.log.V(1).Info("Execute Prune")

	err := config.validate()
	if err != nil {
		return err
	}

	for i := 0; i < len(config.Resources); i++ {
		var resourceList []ResourceInfo
		var err error

		if config.Resources[i].Kind == PodKind {
			resourceList, err = config.getSucceededPods(ctx)
			if err != nil {
				return err
			}
			config.log.V(1).Info("pods ", "count", len(resourceList))
		} else if config.Resources[i].Kind == JobKind {
			resourceList, err = config.getCompletedJobs(ctx)
			if err != nil {
				return err
			}
			config.log.V(1).Info("jobs ", "count", len(resourceList))
		}

		var resourcesToRemove []ResourceInfo

		switch config.Strategy.Mode {
		case MaxAgeStrategy:
			resourcesToRemove, err = pruneByMaxAge(ctx, config, resourceList)
		case MaxCountStrategy:
			resourcesToRemove, err = pruneByMaxCount(ctx, config, resourceList)
		case CustomStrategy:
			resourcesToRemove, err = config.CustomStrategy(config, resourceList)
		default:
			return fmt.Errorf("unknown strategy")
		}
		if err != nil {
			return err
		}

		err = config.removeResources(ctx, resourcesToRemove)
		if err != nil {
			return err
		}
	}

	config.log.V(1).Info("Prune completed")

	return nil
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

// containsName checks if a string is present in a ResourceInfo slice
func containsName(s []ResourceInfo, str string) bool {
	for _, v := range s {
		if v.Name == str {
			return true
		}
	}

	return false
}
func (config Config) validate() (err error) {

	if config.CustomStrategy == nil && config.Strategy.Mode == CustomStrategy {
		return fmt.Errorf("custom strategies require a strategy function to be specified")
	}

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
