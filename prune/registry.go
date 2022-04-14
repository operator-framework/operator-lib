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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Registry is used to register a mapping of GroupVersionKind to an IsPrunableFunc
type Registry struct {
	// prunables is a map of GVK to an IsPrunableFunc
	prunables map[schema.GroupVersionKind]IsPrunableFunc
}

// NewRegistry creates a new Registry
func NewRegistry() *Registry {
	return new(Registry)
}

// DefaultRegistry is a default Registry configuration
func DefaultRegistry() *Registry {
	return &defaultRegistry
}

var defaultRegistry Registry

// RegisterIsPrunableFunc registers a function to check whether it is safe to prune a resource of a certain type.
func (r *Registry) RegisterIsPrunableFunc(gvk schema.GroupVersionKind, isPrunable IsPrunableFunc) {
	if r.prunables == nil {
		r.prunables = make(map[schema.GroupVersionKind]IsPrunableFunc)
	}

	r.prunables[gvk] = isPrunable
}

// IsPrunable checks if an object is prunable
func (r *Registry) IsPrunable(obj client.Object) error {
	isPrunable, ok := r.prunables[obj.GetObjectKind().GroupVersionKind()]
	if !ok {
		return nil
	}

	return isPrunable(obj)
}

// RegisterIsPrunableFunc registers a function to check whether it is safe to prune a resource of a certain type.
func RegisterIsPrunableFunc(gvk schema.GroupVersionKind, isPrunable IsPrunableFunc) {
	DefaultRegistry().RegisterIsPrunableFunc(gvk, isPrunable)
}

// IsPrunable checks if an object is prunable
func IsPrunable(obj client.Object) error {
	return DefaultRegistry().IsPrunable(obj)
}
