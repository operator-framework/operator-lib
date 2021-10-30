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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (config Config) removeResources(ctx context.Context, resources []ResourceInfo) (err error) {

	if config.DryRun {
		return nil
	}

	for i := 0; i < len(resources); i++ {
		r := resources[i]

		if config.PreDeleteHook != nil {
			err = config.PreDeleteHook(config, r)
			if err != nil {
				return err
			}
		}

		switch resources[i].GVK.Kind {
		case PodKind:
			err := config.Clientset.CoreV1().Pods(r.Namespace).Delete(ctx, r.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		case JobKind:
			err := config.Clientset.BatchV1().Jobs(r.Namespace).Delete(ctx, r.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported resource kind")
		}
	}

	return nil
}
