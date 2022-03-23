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
)

// pruneByMaxCount looks for and prunes resources, currently jobs and pods,
// that exceed a user specified count (e.g. 3), the oldest resources
// are pruned, resources to remove are returned
func pruneByMaxCount(ctx context.Context, config Config, resources []ResourceInfo) (resourcesToRemove []ResourceInfo, err error) {
	log := Logger(ctx, config)
	log.V(1).Info("pruneByMaxCount running ", "max count", config.Strategy.MaxCountSetting, "resource count", len(resources))
	if config.Strategy.MaxCountSetting < 0 {
		return resourcesToRemove, fmt.Errorf("max count setting less than zero")
	}

	if len(resources) > config.Strategy.MaxCountSetting {
		removeCount := len(resources) - config.Strategy.MaxCountSetting
		for i := len(resources) - 1; i >= 0; i-- {
			log.V(1).Info("pruning pod ", "pod name", resources[i].Name, "age", time.Since(resources[i].StartTime))

			resourcesToRemove = append(resourcesToRemove, resources[i])

			removeCount--
			if removeCount == 0 {
				break
			}
		}
	}

	return resourcesToRemove, nil
}
