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

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Harvester harvests a job when run.
type Harvester interface {
	Run(context.Context, *batchv1.Job) error
}

// harvester harvests jobs.
type harvester struct {
	name       string
	logger     logr.Logger
	k8sClient  kubernetes.Interface
	ctrlClient client.Client
	lw         LogWriter
}

// Run streams logs of all containers in job if job is complete.
func (h *harvester) Run(ctx context.Context, job *batchv1.Job) error {

	if shouldSkip(job) {
		return nil
	}

	if h.ctrlClient != nil {
		return h.runCtrl(ctx, job)
	}
	return h.runClientGo(ctx, job)
}

func shouldSkip(job *batchv1.Job) bool {
	// Skip suspended jobs, since they will not spawn pods.
	suspended := job.Spec.Suspend != nil && *job.Spec.Suspend
	// To avoid duplicate streamed logs, skip jobs until they are complete
	// and their containers finish running.
	notComplete := job.Status.CompletionTime == nil

	return suspended || notComplete
}

// runCtrl streams logs of all containers in job if job is complete
// using a controller-runtime client.
func (h *harvester) runCtrl(ctx context.Context, job *batchv1.Job) error {

	// QUESTION(estroz): handle job.Spec.ManualSelector?
	sel, err := metav1.LabelSelectorAsSelector(job.Spec.Selector)
	if err != nil {
		return err
	}
	podList := &corev1.PodList{}
	podListOpts := []client.ListOption{
		client.MatchingLabelsSelector{Selector: sel},
		client.InNamespace(job.Namespace),
	}
	if err := h.ctrlClient.List(ctx, podList, podListOpts...); err != nil {
		return err
	}

	if err := streamPodLogs(ctx, h.k8sClient, podList, h.lw, h.logger); err != nil {
		return err
	}

	var errs []error
	for _, pod := range podList.Items {
		podLog := h.logger.WithValues("podName", pod.Name, "podNamespace", pod.Namespace)

		if controllerutil.ContainsFinalizer(&pod, jobFinalizer) {
			podLog.V(1).Info("removing finalizer")

			controllerutil.RemoveFinalizer(&pod, jobFinalizer)
			if err := h.ctrlClient.Update(ctx, &pod); err != nil {
				podLog.Error(err, "updating pod to remove finalizer")
				errs = append(errs, err)
				continue
			}
		}
	}

	if controllerutil.ContainsFinalizer(job, jobFinalizer) {
		h.logger.V(1).Info("removing finalizer")

		controllerutil.RemoveFinalizer(job, jobFinalizer)
		if err := h.ctrlClient.Update(ctx, job); err != nil {
			h.logger.Error(err, "updating job to remove finalizer")
			return utilerrors.NewAggregate(append(errs, err))
		}
	}

	if job.Spec.TTLSecondsAfterFinished == nil {
		h.logger.V(1).Info("deleting job, ttlSecondsAfterFinished unset")

		// Set TTL to 0 to have the TTL controller delete the Job and its Pods.
		pt := types.StrategicMergePatchType
		p := []byte(`{"spec":{"ttlSecondsAfterFinished":0}}`)
		opts := []client.PatchOption{
			client.FieldOwner(controllerName),
		}
		if err := h.ctrlClient.Patch(ctx, job, client.RawPatch(pt, p), opts...); err != nil {
			h.logger.Error(err, "delete job")
			return utilerrors.NewAggregate(append(errs, err))
		}
	}

	return utilerrors.NewAggregate(errs)
}

// runCtrl streams logs of all containers in job if job is complete
// using a client-go client.
func (h *harvester) runClientGo(ctx context.Context, job *batchv1.Job) error {

	// QUESTION(estroz): handle job.Spec.ManualSelector?
	sel, err := metav1.LabelSelectorAsSelector(job.Spec.Selector)
	if err != nil {
		return err
	}
	opts := metav1.ListOptions{
		LabelSelector: sel.String(),
	}
	podList, err := h.k8sClient.CoreV1().Pods(job.Namespace).List(ctx, opts)
	if err != nil {
		return err
	}

	if err := streamPodLogs(ctx, h.k8sClient, podList, h.lw, h.logger); err != nil {
		return err
	}

	var errs []error
	for _, pod := range podList.Items {
		podLog := h.logger.WithValues("podName", pod.Name, "podNamespace", pod.Namespace)

		if controllerutil.ContainsFinalizer(&pod, jobFinalizer) {
			podLog.V(1).Info("removing finalizer")

			controllerutil.RemoveFinalizer(&pod, jobFinalizer)
			if _, err = h.k8sClient.CoreV1().Pods(pod.Namespace).Update(ctx, &pod, metav1.UpdateOptions{}); err != nil {
				podLog.Error(err, "updating pod to remove finalizer")
				errs = append(errs, err)
				continue
			}
		}
	}

	if controllerutil.ContainsFinalizer(job, jobFinalizer) {
		h.logger.V(1).Info("removing finalizer")

		controllerutil.RemoveFinalizer(job, jobFinalizer)
		if job, err = h.k8sClient.BatchV1().Jobs(job.Namespace).Update(ctx, job, metav1.UpdateOptions{}); err != nil {
			h.logger.Error(err, "updating job to remove finalizer")
			return utilerrors.NewAggregate(append(errs, err))
		}
	}

	if job.Spec.TTLSecondsAfterFinished == nil {
		h.logger.V(1).Info("deleting job, ttlSecondsAfterFinished unset")

		// Set TTL to 0 to have the TTL controller delete the Job and its Pods.
		pt := types.MergePatchType
		p := []byte(`{"spec":{"ttlSecondsAfterFinished":0}}`)
		opts := metav1.PatchOptions{
			FieldManager: controllerName,
		}
		if _, err = h.k8sClient.BatchV1().Jobs(job.Namespace).Patch(ctx, job.Name, pt, p, opts); err != nil {
			h.logger.Error(err, "delete job")
			return utilerrors.NewAggregate(append(errs, err))
		}
	}

	return utilerrors.NewAggregate(errs)
}

func streamPodLogs(ctx context.Context, k8sClient kubernetes.Interface, podList *corev1.PodList, lw LogWriter, l logr.Logger) error {

	var errs []error
	for _, pod := range podList.Items {
		podLog := l.WithValues("podName", pod.Name, "podNamespace", pod.Namespace)
		podLog.V(1).Info("found pod")

		for _, containerStatus := range pod.Status.ContainerStatuses {

			err := func() error {

				ctrLog := podLog.WithValues("container", containerStatus.Name)
				ctrLog.V(1).Info("streaming logs")

				logOpts := corev1.PodLogOptions{
					Container:  containerStatus.Name,
					Follow:     true,
					Timestamps: true,
				}
				rc, err := k8sClient.CoreV1().Pods(pod.GetNamespace()).GetLogs(pod.GetName(), &logOpts).Stream(ctx)
				if err != nil {
					ctrLog.Error(err, "stream logs")
					return err
				}
				defer func() {
					if err := rc.Close(); err != nil {
						ctrLog.Error(err, "close stream")
					}
				}()

				if err := lw.WriteLogs(ctx, rc, pod, containerStatus.Name); err != nil {
					ctrLog.Error(err, "read logs")
					return err
				}

				return nil
			}()
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}

	return utilerrors.NewAggregate(errs)
}
