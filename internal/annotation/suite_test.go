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

package annotation_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestAnnotation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Annotation Suite")
}

func makeCreateEventFor(obj client.Object) event.CreateEvent {
	var e event.CreateEvent
	e.Object = obj
	return e
}

func makeUpdateEventFor(oldObj, newObj client.Object) event.UpdateEvent {
	var e event.UpdateEvent
	e.ObjectOld = oldObj
	e.ObjectNew = newObj
	return e
}

func makeDeleteEventFor(obj client.Object) event.DeleteEvent {
	var e event.DeleteEvent
	e.Object = obj
	return e
}

func makeGenericEventFor(obj client.Object) event.GenericEvent {
	var e event.GenericEvent
	e.Object = obj
	return e
}
