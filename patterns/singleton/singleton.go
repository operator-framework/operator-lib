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

package singleton

import (
	"context"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ manager.Runnable = runnable{}
var _ manager.LeaderElectionRunnable = runnable{}

type runnable struct {
	objs []client.Object
	c    client.Client
}

// NewRunnable returns a manager.Runnable that requires leader election to
// create all objs using c. This runnable should be added to a manager.Manager
// with Manager.Add(runnable).
//
//	const singletonFooName = "global-foo"
//
//	func main() {
//
//		mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
//		if err != nil {
//			os.Exit(1)
//		}
//
//		// Some internally or externally defined API type.
//		foo := &foosv1alpha1.Foo{}
//		foo.SetName(singletonFooName)
//		foo.Spec.Bar = "baz"
//		// foo will be created after leader election has started.
//		mgr.Add(singleton.NewRunnable(mgr.GetClient(), foo))
//
//	}
func NewRunnable(c client.Client, objs ...client.Object) manager.Runnable {
	return runnable{c: c, objs: objs}
}

func (r runnable) NeedLeaderElection() bool { return true }

// TODO(estroz): parallelize
func (r runnable) Start(ctx context.Context) error {
	for _, obj := range r.objs {
		if err := r.c.Create(ctx, obj); err != nil {
			return err
		}
	}

	for _, obj := range r.objs {
		// Blocking here is fine because this method is not run in a controller.
		if err := waitForCreate(ctx, r.c, obj); err != nil {
			return err
		}
	}

	return nil
}

func waitForCreate(ctx context.Context, c client.Client, obj client.Object) error {
	key := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}

	return wait.PollImmediateUntil(200*time.Millisecond, func() (bool, error) {
		err := c.Get(ctx, key, obj)
		return err == nil, err
	}, ctx.Done())
}

// ConstraintViolationError is returned when the singleton constraint is violated cluster-wide.
type ConstraintViolationError struct {
	schema.GroupVersionKind
	// ExpectedName is the expected name of the singleton object.
	ExpectedName string
	// ViolatingNames is a list of objects with any name != ExpectedName.
	ViolatingNames []string
}

func (e ConstraintViolationError) Error() string {
	return fmt.Sprintf("expected the set of objects of type %s to contain only %q, found %q",
		e.GroupVersionKind, e.ExpectedName, e.ViolatingNames)
}

// CheckViolations returns an error if the cluster-wide state has violated the singleton constraint.
// Use this function within your controller's reconcile loop.
func CheckViolations(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, objectName string) error {
	list := unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	err := c.List(ctx, &list, client.InNamespace(""))
	if err != nil {
		return err
	}

	gr := schema.GroupResource{Group: gvk.Group, Resource: gvk.Kind}
	if len(list.Items) == 0 {
		return apierrors.NewNotFound(gr, objectName)
	}

	var violatingNames []string
	for _, obj := range list.Items {
		if existingName := obj.GetName(); existingName != objectName {
			violatingNames = append(violatingNames, existingName)
		}
	}
	if len(violatingNames) > 0 {
		return ConstraintViolationError{
			GroupVersionKind: gvk,
			ExpectedName:     objectName,
			ViolatingNames:   violatingNames,
		}
	}

	return nil
}

// EnforceConstraint will delete all non-singleton objects if CheckViolations() returns an error.
// Use this function within your controller's reconcile loop.
func EnforceConstraint(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, objectName string, deleteOpts ...client.DeleteOption) error {
	sverr := &ConstraintViolationError{}
	if err := CheckViolations(ctx, c, gvk, objectName); err == nil || !errors.As(err, &sverr) {
		return err
	}

	for _, violatingName := range sverr.ViolatingNames {
		// Attempt to use the typed cache.
		robj, err := c.Scheme().New(gvk)
		if err != nil {
			return err
		}
		obj, ok := robj.(client.Object)
		if !ok {
			obj = &unstructured.Unstructured{}
			obj.GetObjectKind().SetGroupVersionKind(gvk)
		}
		obj.SetName(violatingName)
		if err := c.Delete(ctx, obj, deleteOpts...); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
