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

package jobharvest

import (
	"k8s.io/client-go/kubernetes"
)

// NewControllerClientGo returns a controller with a Harvester registered for each opt in opts.
// TODO(estroz): flesh this out.
func NewControllerClientGo(k8sClient kubernetes.Interface, opts ...*HarvesterOptions) (HarvestController, error) {
	hc := &harvestController{
		k8sClient: k8sClient,
		hrvs:      make(harvesters),
	}

	for _, opt := range opts {
		if _, err := hc.Create(opt); err != nil {
			return nil, err
		}
	}

	return hc, nil
}
