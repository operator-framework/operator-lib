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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestPredicate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Predicate Suite")
}

func makeCreateEventFor(obj controllerutil.Object) (e event.CreateEvent) {
	e.Object, e.Meta = obj, obj
	return e
}

func makeUpdateEventFor(old, new controllerutil.Object) (e event.UpdateEvent) {
	e.ObjectOld, e.MetaOld = old, old
	e.ObjectNew, e.MetaNew = new, new
	return e
}

func makeDeleteEventFor(obj controllerutil.Object) (e event.DeleteEvent) {
	e.Object, e.Meta = obj, obj
	return e
}

func makeGenericEventFor(obj controllerutil.Object) (e event.GenericEvent) {
	e.Object, e.Meta = obj, obj
	return e
}
