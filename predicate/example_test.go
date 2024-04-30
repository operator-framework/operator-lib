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

package predicate_test

import (
	"context"
	"os"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/operator-framework/operator-lib/predicate"
)

// This example applies the Pause predicate to all incoming Pod events on a Pod controller.
func ExampleNewPause() {
	cfg, err := config.GetConfig()
	if err != nil {
		os.Exit(1)
	}

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		os.Exit(1)
	}

	var r reconcile.Func = func(_ context.Context, _ reconcile.Request) (reconcile.Result, error) {
		// Your reconcile logic would go here. No paused Pod events would trigger reconciliation.
		return reconcile.Result{}, nil
	}

	// Filter out Pods with the "my.app/paused: true" annotation.
	pause, err := predicate.NewPause("my.app/paused")
	if err != nil {
		os.Exit(1)
	}
	pred := builder.WithPredicates(pause)
	if err := builder.ControllerManagedBy(mgr).For(&corev1.Pod{}, pred).Complete(r); err != nil {
		os.Exit(1)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		os.Exit(1)
	}
}
