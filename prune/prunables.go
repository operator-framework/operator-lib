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
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// Some default pruneable functions

// DefaultPodIsPrunable is a default IsPrunableFunc to be used specifically with Pod resources.
// This can be overridden by registering your own IsPrunableFunc via the RegisterIsPrunableFunc method
func DefaultPodIsPrunable(obj client.Object) error {
	pod := obj.(*corev1.Pod)
	if pod.Status.Phase != corev1.PodSucceeded {
		return &Unprunable{
			Obj:    &obj,
			Reason: "Pod has not succeeded",
		}
	}

	return nil
}

// DefaultJobIsPrunable is a default IsPrunableFunc to be used specifically with Job resources.
// This can be overridden by registering your own IsPrunableFunc via the RegisterIsPrunableFunc method
func DefaultJobIsPrunable(obj client.Object) error {

	job := obj.(*batchv1.Job)

	// If the job has completed we can remove it
	if job.Status.CompletionTime == nil {
		return &Unprunable{
			Obj:    &obj,
			Reason: "Job has not completed",
		}
	}

	return nil
}
