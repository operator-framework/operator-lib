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

package predicate

import (
	"github.com/operator-framework/operator-lib/internal/annotation"

	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NewPause returns a predicate that filters out objects with a truthy "paused" annotation.
// When an annotation with key string key is present on an object and has a truthy value, ex. "true",
// the watch constructed with this predicate will not pass events for that object to the event handler.
// Key string key must be a valid annotation key.
func NewPause(key string) (predicate.Predicate, error) {
	return annotation.NewFalsyPredicate(key, annotation.Options{Log: log})
}
