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

package jobharvest

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	logger = ctrllog.Log.WithName(controllerName)

	// harvestController implements Reconcile.
	_ reconcile.Reconciler = &harvestController{}
)

// NewControllerCtrl returns a controller with a Harvester registered for each opt in opts.
//
// **VERY IMPORTANT** - Many, many controllers use Jobs for various tasks, so there will likely
// be many in some state at a given time. This means your cache will hydrate with each Job
// in the set of watched namespaces, burdening your node unnecessarily.
// Make sure your cache is set to only watch Jobs with the harvester's label applied,
// and that these labels unique to your operator by using the operator's package name:
//
//  myJobLabels := labels.Set{"job-harvester": "foo-operator"}
//  opts := manager.Options{
//    NewCache: cache.BuilderWithOptions(cache.Options{
//      SelectorsByObject: cache.SelectorsByObject{
//        &batchv1.Job{}: {Label: myJobLabels.AsSelector()},
//      },
//    }),
//  }
//
func NewControllerCtrl(k8sClient kubernetes.Interface, mgr manager.Manager, opts ...*HarvesterOptions) (HarvestController, error) {
	hc := &harvestController{
		k8sClient:  k8sClient,
		ctrlClient: mgr.GetClient(),
		hrvs:       make(harvesters),
	}

	for _, opt := range opts {
		if _, err := hc.Create(opt); err != nil {
			return nil, err
		}
	}

	ctrlOpts := controller.Options{
		Reconciler: hc,
		Log:        logger,
	}
	c, err := controller.New(controllerName, mgr, ctrlOpts)
	if err != nil {
		return nil, err
	}

	if err := c.Watch(
		&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestForObject{},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				job, newIsJob := e.ObjectNew.(*batchv1.Job)
				if newIsJob {
					if job.Spec.Suspend != nil && *job.Spec.Suspend {
						logger.V(1).Info("job suspended", "jobName", job.Name, "jobNamespace", job.Namespace)
						return false
					}
					if job.Status.CompletionTime == nil {
						logger.V(1).Info("job not completed", "jobName", job.Name, "jobNamespace", job.Namespace)
						return false
					}
				}
				return newIsJob
			},
			CreateFunc:  func(event.CreateEvent) bool { return false },
			DeleteFunc:  func(event.DeleteEvent) bool { return false },
			GenericFunc: func(event.GenericEvent) bool { return false },
		},
	); err != nil {
		return nil, err
	}

	return hc, nil
}

// Reconcile implements reconcile.Reconciler on harvestController.
func (hc *harvestController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	jobLog := ctrllog.FromContext(ctx).WithValues("jobName", req.Name, "jobNamespace", req.Namespace)

	job := &batchv1.Job{}
	if err := hc.ctrlClient.Get(ctx, req.NamespacedName, job); err != nil {
		if errors.IsNotFound(err) {
			jobLog.V(1).Info("harvester not found; are you sure your controller is constrained to watching only labeled Jobs?")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	h, has := hc.hrvs.get(job)
	if !has {
		jobLog.Error(fmt.Errorf("harvester not registered"), "")
		return reconcile.Result{}, nil
	}

	h.logger = jobLog
	if err := h.runCtrl(ctx, job); err != nil {
		jobLog.Error(err, "harvester run failed")
		// QUESTION(estroz): requeue and risk duplicating log read (usually log engines will deduplicate)
		// or just let the error slide? Probably requeue or somehow guarantee more Job deletion attempts.
		return reconcile.Result{Requeue: true}, nil
	}

	if h.runOnce {
		if err := hc.Remove(job); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
