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
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	RegisterIsPrunableFunc(corev1.SchemeGroupVersion.WithKind("Pod"), DefaultPodIsPrunable)

	RegisterIsPrunableFunc(batchv1.SchemeGroupVersion.WithKind("Job"), DefaultJobIsPrunable)
}

// Pruner is an object that runs a prune job.
type Pruner struct {
	registry Registry

	// client is the controller-runtime client that will be used
	// To perform a dry run, use the controller-runtime DryRunClient
	client client.Client

	// gvk is the type of objects to prune.
	// It defaults to Pod
	gvk schema.GroupVersionKind

	// strategy is the function used to determine a list of resources that are pruneable
	strategy StrategyFunc

	// labels is a map of the labels to use for label matching when looking for resources
	labels map[string]string

	// namespace is the namespace to use when looking for resources
	namespace string
}

// Unprunable indicates that it is not allowed to prune a specific object.
type Unprunable struct {
	Obj    *client.Object
	Reason string
}

// Error returns a string representation of an `Unprunable` error.
func (e *Unprunable) Error() string {
	return fmt.Sprintf("unable to prune %s: %s", client.ObjectKeyFromObject(*e.Obj), e.Reason)
}

// StrategyFunc takes a list of resources and returns the subset to prune.
type StrategyFunc func(ctx context.Context, objs []client.Object) ([]client.Object, error)

// IsPrunableFunc is a function that checks the data of an object to see whether or not it is safe to prune it.
// It should return `nil` if it is safe to prune, `Unprunable` if it is unsafe, or another error.
// It should safely assert the object is the expected type, otherwise it might panic.
type IsPrunableFunc func(obj client.Object) error

// PrunerOption configures the pruner.
type PrunerOption func(p *Pruner)

// WithNamespace can be used to set the Namespace field when configuring a Pruner
func WithNamespace(namespace string) PrunerOption {
	return func(p *Pruner) {
		p.namespace = namespace
	}
}

// WithLabels can be used to set the Labels field when configuring a Pruner
func WithLabels(labels map[string]string) PrunerOption {
	return func(p *Pruner) {
		p.labels = labels
	}
}

// GVK returns the schema.GroupVersionKind that the Pruner has set
func (p Pruner) GVK() schema.GroupVersionKind {
	return p.gvk
}

// Labels returns the labels that the Pruner is using to find resources to prune
func (p Pruner) Labels() map[string]string {
	return p.labels
}

// Namespace returns the namespace that the Pruner is using to find resources to prune
func (p Pruner) Namespace() string {
	return p.namespace
}

// NewPruner returns a pruner that uses the given strategy to prune objects that have the given GVK
func NewPruner(prunerClient client.Client, gvk schema.GroupVersionKind, strategy StrategyFunc, opts ...PrunerOption) (*Pruner, error) {
	if gvk.Empty() {
		return nil, fmt.Errorf("error when creating a new Pruner: gvk parameter can not be empty")
	}

	pruner := Pruner{
		registry: defaultRegistry,
		client:   prunerClient,
		gvk:      gvk,
		strategy: strategy,
	}

	for _, opt := range opts {
		opt(&pruner)
	}

	return &pruner, nil
}

// Prune runs the pruner.
func (p Pruner) Prune(ctx context.Context) ([]client.Object, error) {
	listOpts := client.ListOptions{
		LabelSelector: labels.Set(p.labels).AsSelector(),
		Namespace:     p.namespace,
	}

	var unstructuredObjs unstructured.UnstructuredList
	unstructuredObjs.SetGroupVersionKind(p.gvk)
	if err := p.client.List(ctx, &unstructuredObjs, &listOpts); err != nil {
		return nil, fmt.Errorf("error getting a list of resources: %w", err)
	}

	objs := make([]client.Object, 0, len(unstructuredObjs.Items))

	for i := range unstructuredObjs.Items {
		unsObj := unstructuredObjs.Items[i]
		obj, err := convert(p.client, p.gvk, &unsObj)
		if err != nil {
			return nil, err
		}

		if err := p.registry.IsPrunable(obj); IsUnprunable(err) {
			continue
		} else if err != nil {
			return nil, err
		}

		objs = append(objs, obj)
	}

	objsToPrune, err := p.strategy(ctx, objs)
	if err != nil {
		return nil, fmt.Errorf("error determining prunable objects: %w", err)
	}

	// Prune the resources
	for _, obj := range objsToPrune {
		if err = p.client.Delete(ctx, obj); err != nil {
			return nil, fmt.Errorf("error pruning object: %w", err)
		}
	}

	return objsToPrune, nil
}

// IsUnprunable checks if a given error is that of Unprunable.
// Returns true if the given error is of type Unprunable, and false if it is not
func IsUnprunable(target error) bool {
	var unprunable *Unprunable
	return errors.As(target, &unprunable)
}

func convert(c client.Client, gvk schema.GroupVersionKind, obj client.Object) (client.Object, error) {
	obj2, err := c.Scheme().New(gvk)
	if err != nil {
		return nil, err
	}
	objConverted := obj2.(client.Object)
	if err := c.Scheme().Convert(obj, objConverted, nil); err != nil {
		return nil, err
	}

	objConverted.GetObjectKind().SetGroupVersionKind(gvk)

	return objConverted, nil
}
