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

/*
Package singleton defines utility functions for implementing the singleton
pattern for an API type. A singleton is an operator for which only one instance
of a custom object ever exists; whether that is per namespace or cluster-wide
depends on the API CustomResourceDefinition's or internal definition's scope.
This object must have a predefined name for efficient lookups and discovery.

The following example demonstrates usage of all utilities defined here, in concert
with controller-runtime libraries.

In a type definitions file 'api/v1alpha1/foo_types.go':

	import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// The single Foo object, in a namespace or cluster-wide, will always have this name.
	const SingletonFooName = "global-foo"

	// Foo is some internally or externally defined API type.
	type Foo struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec FooSpec `json:"spec"`
		...
	}

	type FooSpec struct {
		Bar string `json:"foo"`
	}

In a webhook definitions file 'api/v1alpha1/foo_webhook.go':

	import (
		"github.com/operator-framework/operator-lib/patterns/singleton"
		"sigs.k8s.io/controller-runtime/pkg/webhook"
	)

	var _ webhook.Validator = &Foo{}

	func (r *Foo) ValidateCreate() error {
		if err := singleton.ValidateCreate(r, SingletonFooName); err != nil {
			return err
		}
		// Other validation.
		return nil
	}

	// Since an object's name cannot be updated after being created, no ValidateUpdate()
	// logic for singletons is required.
	func (r *Foo) ValidateUpdate(old runtime.Object) error { ... }

	func (r *Foo) ValidateDelete() error {
		// This is optional since it only logs a debug message.
		if err := singleton.ValidateDelete(r, SingletonFooName); err != nil {
			return err
		}
		// Other validation.
		return nil
	}

In a controller implementation file 'controllers/foo_controller.go':

	import (
		"context"
		ctrl "sigs.k8s.io/controller-runtime"
		"sigs.k8s.io/controller-runtime/pkg/client"
		"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	)

	type FooReconciler struct {
		client.Client
		...
	}

	func (r *FooReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		expectedFoo := NewExpectedFoo()
		updateFn := func() error {
			// Some mutation on expectedFoo if it exists.
			return nil
		}
		// Ensure a Foo with the expected name and desired body always exists
		if err := controllerutil.CreateOrUpdate(ctx, r.Client, expectedFoo, updateFn); err != nil {
			return ctrl.Result{}, err
		}
		...
	}

	func NewExpectedFoo() *v1alpha1.Foo {
		foo := &v1alpha1.Foo{}
		foo.SetName(v1alpha1.SingletonFooName)
		foo.Spec.Bar = "baz"
		return foo
	}

In the main function:

	import (
		"github.com/operator-framework/operator-lib/patterns/singleton"
		ctrl "sigs.k8s.io/controller-runtime"
		"my.project/foo-operator/api/v1alpha1"
		"my.project/foo-operator/controllers"
	)

	func main() {
		mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})

		// Set up the reconciler, which will run the singleton create/update/patch function.
		r := &controllers.FooReconciler{Client: mgr.GetClient()}
		_ = r.SetupWithManager(mgr)

		// foo will be created after leader election has started.
		foo := controllers.NewExpectedFoo()
		_ = mgr.Add(singleton.NewRunnable(mgr.GetClient(), foo))

		// Set up the validating webhook, which will call singleton validators.
		_ = (&v1alpha1.Foo{}).SetupWebhookWithManager(mgr)
		...
	}

*/
package singleton
