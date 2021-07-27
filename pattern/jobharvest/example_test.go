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

package jobharvest_test

import (
	"context"
	"fmt"
	"os"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/operator-framework/operator-lib/pattern/jobharvest"
)

// This example creates a HarvesterController controller with controller-runtime libraries
// managed by a manager.Manager to extract Job logs in various situations.
func ExampleNewControllerCtrl() {
	checkErr := func(err error) {
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	cfg := config.GetConfigOrDie()

	// Labels that, when applied to a Job, will trigger harvester reconciles
	// on updates. Make these labels unique to your operator by using the
	// operator's package name.
	jobLabels := labels.Set{"job-harvester": "foo-operator"}

	// **VERY IMPORTANT** - Many, many controllers use Jobs for various tasks,
	// so there will likely be many in some state at a given time. This means
	// your cache will hydrate with each Job in the set of watched namespaces,
	// burdening your node unnecessarily. Make sure your cache is set to only
	// watch Jobs with the harvester's label applied.
	opts := manager.Options{
		NewCache: cache.BuilderWithOptions(cache.Options{
			SelectorsByObject: cache.SelectorsByObject{
				&batchv1.Job{}: {Label: jobLabels.AsSelector()},
			},
		}),
	}
	mgr, err := manager.New(cfg, opts)
	checkErr(err)

	createLogFile := func(name string) (*os.File, error) {
		return os.OpenFile("/var/log/"+name, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	}

	// Jobs known beforehand.
	fooJobFile, err := createLogFile("job-foo.log")
	checkErr(err)
	defer fooJobFile.Close()
	harvesterOpts := []*jobharvest.HarvesterOptions{
		{Name: "job-foo", LogWriter: jobharvest.WriteLogsTo(fooJobFile)},
	}

	// Create the Job harvester controller with one harvester per option
	// and add it to the manager.
	hf, err := jobharvest.NewControllerCtrl(kubernetes.NewForConfigOrDie(cfg), mgr, harvesterOpts...)
	checkErr(err)

	// Example reconciler function that uses the HarvesterFactory in several ways.
	var r reconcile.Func = func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
		// 1. Existing Job harvesting, in which a Job is created on each reconcile
		// if one does not exist that has the same name every time.
		// This Job is harvested by the "job-foo" harvester by direct name matching.
		systemJob := newJob("job-foo", "operator-system")
		for k, v := range jobLabels {
			systemJob.Labels[k] = v
		}
		if err := mgr.GetClient().Create(ctx, systemJob); err != nil && !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}

		pod := &v1.Pod{}
		if err := mgr.GetClient().Get(ctx, req.NamespacedName, pod); err != nil {
			return reconcile.Result{}, err
		}
		if needsJob(pod) {
			// 2. Dynamic Job harvesting, where the Job's name might not be known beforehand
			// but an existing harvester needs to be re-used.
			job := newJobForPod(pod)
			for k, v := range jobLabels {
				job.Labels[k] = v
			}
			// Re-use the existing "job-foo" harvester to write logs to the same location
			// by setting a registration annotation with the registered harvester's name.
			job.Annotations[jobharvest.HarvesterRegistrationKey] = "job-foo"
			if err := mgr.GetClient().Create(ctx, job); err != nil {
				return reconcile.Result{}, err
			}
		}

		// 3. Constructing harvesters on the fly, where a new harvester is
		// needed when some condition is true. For example,
		// this controller also owns some services that require inspection
		// with a Job if have bad LB port configurations, and a new Job per
		// service created is required, each with its own harvester so logs
		// are written to unique destinations.
		svcList := &v1.ServiceList{}
		if err := mgr.GetClient().List(ctx, svcList, client.InNamespace(req.Namespace)); err != nil {
			return reconcile.Result{}, err
		}
		lbIngressHasPortError := func(ingresses []v1.LoadBalancerIngress) bool {
			for _, ingressStatus := range ingresses {
				for _, portStatus := range ingressStatus.Ports {
					if portStatus.Error != nil {
						return true
					}
				}
			}
			return false
		}
		erroredSvcNames := []string{}
		for _, svc := range svcList.Items {
			if lbIngressHasPortError(svc.Status.LoadBalancer.Ingress) {
				erroredSvcNames = append(erroredSvcNames, svc.Name)
			}
		}
		// Create the harvester first so the job will definitely be cleaned up when done.
		jobName := fmt.Sprintf("service-job-%d", time.Now().UnixNano())
		jobFile, err := createLogFile(jobName + ".log")
		if err != nil {
			return reconcile.Result{}, err
		}
		hrvOpts := &jobharvest.HarvesterOptions{
			Name:      jobName,
			LogWriter: jobharvest.WriteLogsTo(jobFile),
			// This job is a one-off, so delete the harvester and
			// clean up the file when done.
			RunOnce:  true,
			Cleanups: []func() error{jobFile.Close},
		}
		if _, err := hf.Create(hrvOpts); err != nil {
			log.FromContext(ctx).Error(err, "failed to create job harvester")
			return reconcile.Result{}, nil
		}
		svcJob := newServicesJob(jobName, "operator-system", erroredSvcNames)
		if err := mgr.GetClient().Create(ctx, svcJob); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	checkErr(builder.ControllerManagedBy(mgr).For(&v1.Pod{}).Complete(r))

	// **VERY IMPORTANT** - Register the job harvest webhook, which adds a finalizer
	// to jobs in case spec.ttlSecondsAfterFinished is set. Setting this field is
	// good practice since it results in that job being garbage collected after completion.
	// Prior to that however, the job harvester needs to read each container's logs
	// and so prevents their deletion with this finalizer. Add the following comment
	// to your main.go or a controller Go file to generate a webhook config for your project:
	//
	// +kubebuilder:webhook:path=/job-harvester,mutating=true,failurePolicy=fail,groups="batch",resources=jobs,verbs=create;update,versions=v1,name=job-harvester.my.domain.io
	//
	mgr.GetWebhookServer().Register("/job-harvester", &webhook.Admission{Handler: &jobharvest.Webhook{}})

	checkErr(mgr.Start(signals.SetupSignalHandler()))
}

func needsJob(pod *v1.Pod) bool { return true }
func newJobForPod(pod *v1.Pod) *batchv1.Job {
	return newJob("job-"+pod.Name, pod.Namespace)
}
func newServicesJob(name, namespace string, _ []string) *batchv1.Job {
	return newJob(name, namespace)
}
func newJob(name, namespace string) *batchv1.Job {
	job := &batchv1.Job{}
	job.Name = name
	job.Namespace = namespace
	job.Labels = make(map[string]string)
	job.Annotations = make(map[string]string)
	return job
}
