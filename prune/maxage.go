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
	"time"
)

// maxAge looks for and prunes resources, currently jobs and pods,
// that exceed a user specified age (e.g. 3d), resources to be removed
// are returned
func pruneByMaxAge(_ context.Context, config Config, resources []ResourceInfo) (resourcesToRemove []ResourceInfo, err error) {
	config.log.V(1).Info("maxAge running", "setting", config.Strategy.MaxAgeSetting)

	maxAgeDuration, e := time.ParseDuration(config.Strategy.MaxAgeSetting)
	if e != nil {
		return resourcesToRemove, e
	}

	maxAgeTime := time.Now().Add(-maxAgeDuration)

	for i := 0; i < len(resources); i++ {
		config.log.V(1).Info("age of pod ", "age", time.Since(resources[i].StartTime), "maxage", maxAgeTime)
		if resources[i].StartTime.Before(maxAgeTime) {
			config.log.V(1).Info("pruning ", "kind", resources[i].GVK, "name", resources[i].Name)

			resourcesToRemove = append(resourcesToRemove, resources[i])
		}
	}

	return resourcesToRemove, nil
}
