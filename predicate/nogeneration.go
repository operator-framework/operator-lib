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

package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var _ predicate.Predicate = NoGenerationPredicate{}

// NoGenerationPredicate implements a update predicate function for objects with no Generation value, like a Pod.
//
// This predicate will allow update events on objects that never have their metadata.generation field updated
// by the system, i.e. do not respect Generation semantics:
// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
// This allows a controller to update objects that may have had their spec changed but, because the object does
// not use a generation, will inform on that change in some other manner.
//
// This predicate can be useful by itself, but is intended to be used in conjunction with
// sigs.k8s.io/controller-runtime/pkg/predicate.GenerationChangedPredicate to allow update events on all potentially
// changed objects, those that respect Generation semantics or those that do not:
//
//	import (
//		corev1 "k8s.io/api/core/v1"
//		appsv1 "k8s.io/api/apps/v1"
//		ctrl "sigs.k8s.io/controller-runtime"
//		"sigs.k8s.io/controller-runtime/pkg/event"
//		ctrlpredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
//		libpredicate "github.com/operator-framework/operator-lib/predicate"
//
//		"github.com/example/my-operator/api/v1alpha1"
//	)
//
//	func (r *MyTypeReconciler) SetupWithManager(mgr ctrl.Manager) error {
//		return ctrl.NewControllerManagedBy(mgr).
//			For(&v1alpha1.MyType{}).
//			Owns(&corev1.Pod{}).				// Does not respect Generation.
//			Owns(&appsv1.Deployment{}).	// Respects Generation.
//			WithEventFilter(ctrlpredicate.Or(ctrlpredicate.GenerationChangedPredicate{}, libpredicate.NoGenerationPredicate{})).
//			Complete(r)
//	}
type NoGenerationPredicate struct {
	predicate.Funcs
}

// Update implements the default UpdateEvent filter for validating absence Generation.
func (NoGenerationPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		log.V(1).Info("Update event has no old runtime object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		log.V(1).Info("Update event has no new runtime object for update", "event", e)
		return false
	}
	// Since generation is monotonically increasing, the new generation will always be greater than the old
	// iff the object respects generations.
	return e.ObjectNew.GetGeneration() == e.ObjectOld.GetGeneration() && e.ObjectNew.GetGeneration() == 0
}
