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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

/*
	-------------------------------------------
	New Auto Pruning API Implementation
	-------------------------------------------
*/

// Pruner is an object that runs a prune job.
type Pruner struct {
	Registry

	// Client is the k8s client that will be used
	Client client.Client

	// DryRun indicates whether or not we should actually perform pruning or just return the list of pruneable objects
	// true = Just check, don't prune
	// false (default) = Prune
	DryRun bool

	// GVK is the type of objects to prune.
	// It defaults to Pod
	GVK schema.GroupVersionKind

	// Strategy is the function used to determine a list of resources that are pruneable
	Strategy StrategyFunc

	// Labels is a map of the labels to use for label matching when looking for resources
	Labels map[string]string

	// Namespace is the namespace to use when looking for resources
	Namespace string

	// Logger is the logger to use when running pruning functionality
	Logger logr.Logger
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
type StrategyFunc func(ctx context.Context, objs []client.Object) ([]client.ObjectKey, error)

// IsPrunableFunc is a function that checks the data of an object to see whether or not it is safe to prune it.
// It should return `nil` if it is safe to prune, `Unprunable` if it is unsafe, or another error.
// It should safely assert the object is the expected type, otherwise it might panic.
type IsPrunableFunc func(obj client.Object) error

// PrunerOption configures the pruner.
type PrunerOption func(p *Pruner)

// SetRegistry can be used to set the Registry field when configuring a Pruner
func SetRegistry(registry Registry) PrunerOption {
	return func(p *Pruner) {
		p.Registry = registry
	}
}

// DryRun can be used to set the DryRun field to true when configuring a Pruner
func DryRun() PrunerOption {
	return func(p *Pruner) {
		p.DryRun = true
	}
}

// SetStrategy can be used to set the Strategy field when configuring a Pruner
func SetStrategy(strategy StrategyFunc) PrunerOption {
	return func(p *Pruner) {
		p.Strategy = strategy
	}
}

// Namespace can be used to set the Namespace field when configuring a Pruner
func Namespace(namespace string) PrunerOption {
	return func(p *Pruner) {
		p.Namespace = namespace
	}
}

// SetLogger can be used to set the Logger field when configuring a Pruner
func SetLogger(logger logr.Logger) PrunerOption {
	return func(p *Pruner) {
		p.Logger = logger
	}
}

// Labels can be used to set the Labels field when configuring a Pruner
func Labels(labels map[string]string) PrunerOption {
	return func(p *Pruner) {
		p.Labels = labels
	}
}

// GVK can be used to set the GVK field when configuring a Pruner
func GVK(gvk schema.GroupVersionKind) PrunerOption {
	return func(p *Pruner) {
		p.GVK = gvk
	}
}

// NewPruner returns a pruner that uses the given strategy to prune objects.
func NewPruner(prunerClient client.Client, opts ...PrunerOption) Pruner {
	podGVK := corev1.SchemeGroupVersion.WithKind("Pod")

	jobGVK := batchv1.SchemeGroupVersion.WithKind("Job")

	pruner := Pruner{
		Registry: defaultRegistry,
		Client:   prunerClient,
		DryRun:   false,
		Logger:   Logger(context.Background(), Pruner{}),
		GVK:      podGVK,
	}

	// Populate the default IsPrunableFunc(s)
	RegisterIsPrunableFunc(podGVK, DefaultPodIsPrunable)

	RegisterIsPrunableFunc(jobGVK, DefaultJobIsPrunable)

	for _, opt := range opts {
		opt(&pruner)
	}

	return pruner
}

// Prune runs the pruner.
func (p Pruner) Prune(ctx context.Context) ([]client.ObjectKey, error) {
	var objs []client.Object
	p.Logger.Info("Starting the pruning process...")
	listOpts := client.ListOptions{
		LabelSelector: labels.Set(p.Labels).AsSelector(),
		Namespace:     p.Namespace,
	}

	var unstructuredObjs unstructured.UnstructuredList
	unstructuredObjs.SetGroupVersionKind(p.GVK)
	if err := p.Client.List(ctx, &unstructuredObjs, &listOpts); err != nil {
		p.Logger.Error(err, "failed to get a list of resources for pruning", "Labels", p.Labels, "Namespace", p.Namespace)
		return nil, fmt.Errorf("failed to get list of objects -- ERROR -- %s", err)
	}

	for _, unsObj := range unstructuredObjs.Items {
		obj, err := convert(p.Client, p.GVK, &unsObj)
		if err != nil {
			return nil, err
		}

		if err := p.IsPrunable(obj); isUnprunable(err) {
			continue
		} else if err != nil {
			return nil, err
		}

		objs = append(objs, obj)
	}

	objsToPrune, err := p.Strategy(ctx, objs)
	if err != nil {
		p.Logger.Error(err, "failed to get a list of resources to prune from Strategy")
		return nil, fmt.Errorf("failed when running Strategy -- ERROR -- %s", err)
	}

	if p.DryRun {
		// print out objects
		return objsToPrune, nil
	}

	// Prune the resources
	for _, obj := range objsToPrune {
		// Prune
		prunableObj := &unstructured.Unstructured{}
		prunableObj.SetName(obj.Name)
		prunableObj.SetNamespace(obj.Namespace)
		prunableObj.SetGroupVersionKind(p.GVK)

		if err = p.Client.Delete(ctx, prunableObj); err != nil {
			p.Logger.Error(err, "failed to prune resource", "Resource", obj)
			return nil, fmt.Errorf("failed to prune object -- ERROR -- %s", err)
		}
	}

	return objsToPrune, nil
}

/*
	-------------------------------------------
	New Auto Pruning API Implementation
	-------------------------------------------
*/

// Logger returns a logger from the context using logr method or Config.Log if none is found
// controller-runtime automatically provides a logger in context.Context during Reconcile calls.
// Note that there is no compile time check whether a logger can be retrieved by either way.
// keysAndValues allow to add fields to the logs, cf logr documentation.
func Logger(ctx context.Context, pruner Pruner, keysAndValues ...interface{}) logr.Logger {
	var log logr.Logger
	if pruner.Logger != (logr.Logger{}) {
		log = pruner.Logger
	} else {
		log = ctrllog.FromContext(ctx)
	}
	return log.WithValues(keysAndValues...)
}

func isUnprunable(target error) bool {
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
