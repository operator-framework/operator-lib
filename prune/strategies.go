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
	"sort"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewPruneByCountStrategy returns a StrategyFunc that will return a list of
// resources to prune based on a maximum count of resources allowed.
// If the max count of resources is exceeded, the oldest resources are prioritized for pruning
func NewPruneByCountStrategy(count int) StrategyFunc {
	return func(ctx context.Context, objs []client.Object) ([]client.Object, error) {
		if len(objs) <= count {
			return nil, nil
		}

		// sort objects by creation date
		sortedObjs := objs

		sort.Slice(sortedObjs, func(i, j int) bool {
			iTimestamp := sortedObjs[i].GetCreationTimestamp()
			jTimestamp := sortedObjs[j].GetCreationTimestamp()
			return iTimestamp.Before(&jTimestamp)
		})

		return sortedObjs[count:], nil
	}
}

// NewPruneByDateStrategy returns a StrategyFunc that will return a list of
// resources to prune where the resource CreationTimestamp is after the given time.Time.
func NewPruneByDateStrategy(date time.Time) StrategyFunc {
	return func(ctx context.Context, objs []client.Object) ([]client.Object, error) {
		var objsToPrune []client.Object

		for _, obj := range objs {
			if obj.GetCreationTimestamp().After(date) {
				objsToPrune = append(objsToPrune, obj)
			}
		}

		return objsToPrune, nil
	}
}
