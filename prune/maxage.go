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
	"time"
)

// maxAge looks for and prunes resources, currently jobs and pods,
// that exceed a user specified age (e.g. 3d)
func (config Config) pruneByMaxAge(resources []ResourceInfo) (err error) {
	log.V(1).Info("maxAge running", "setting", config.Strategy.MaxAgeSetting)

	maxAgeDuration, _ := time.ParseDuration(config.Strategy.MaxAgeSetting)
	maxAgeTime := time.Now().Add(-maxAgeDuration)

	for i := 0; i < len(resources); i++ {
		log.V(1).Info("age of pod ", "age", time.Now().Sub(resources[i].StartTime), "maxage", maxAgeTime)
		if resources[i].StartTime.Before(maxAgeTime) {
			log.V(1).Info("pruning ", "kind", resources[i].Kind, "name", resources[i].Name)
			if !config.DryRun {
				err := config.removeResource(resources[i])
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
