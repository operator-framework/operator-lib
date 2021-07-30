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
	"encoding/json"
	"net/http"

	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const jobFinalizer = "job-harvester.operatorframework.io/finalizer"

// WithFinalizers sets finalizer on job and its pod template.
func WithFinalizers(job *batchv1.Job) {
	controllerutil.AddFinalizer(job, jobFinalizer)
	for _, f := range job.Spec.Template.Finalizers {
		if f == jobFinalizer {
			return
		}
	}
	job.Spec.Template.Finalizers = append(job.Spec.Template.Finalizers, jobFinalizer)
}

var (
	_ admission.DecoderInjector = &Webhook{}
	_ admission.Handler         = &Webhook{}
)

// Webhook adds jobFinalizer to Jobs and their Pod template.
type Webhook struct {
	// SetTTL controls if the webhook sets Job.Spec.TTLSecondsAfterFinished = 0
	// on the Job so it is automatically garbage collected once finished.
	// Defaults to true.
	SetTTL *bool

	decoder *admission.Decoder
}

// Handle adds jobFinalizer to every incoming Job.
func (w *Webhook) Handle(ctx context.Context, req admission.Request) admission.Response {

	job := &batchv1.Job{}
	if err := w.decoder.Decode(req, job); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	WithFinalizers(job)
	// Set TTLSecondsAfterFinished by default or if explicitly specified.
	if w.SetTTL == nil || *w.SetTTL {
		job.Spec.TTLSecondsAfterFinished = new(int32)
	}

	b, err := json.Marshal(job)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, b)
}

// InjectDecoder injects the decoder.
func (w *Webhook) InjectDecoder(d *admission.Decoder) error {
	w.decoder = d
	return nil
}
