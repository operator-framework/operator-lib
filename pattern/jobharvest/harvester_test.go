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
	"bytes"
	"context"
	"encoding/json"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("Harvester", func() {
	Context("controller-runtime", func() {
		const (
			podName1 = "foo-1234"
			ns       = "default"
		)

		var (
			ctx       context.Context
			logStderr *bytes.Buffer
			podLogBuf *bytes.Buffer
			lw        LogWriter

			trueP = new(bool)
		)

		BeforeEach(func() {
			*trueP = true
			ctx = context.TODO()
			logStderr = &bytes.Buffer{}
			podLogBuf = &bytes.Buffer{}
			lw = WriteLogsTo(podLogBuf)
		})

		It("skips a suspended job", func() {
			job := newJob("foo", ns)
			job.Spec.Suspend = trueP
			h := newFakeHarvesterCtrl("foo", lw, logStderr, job)

			By("running the harvester")
			Expect(h.Run(ctx, job)).To(Succeed())
		})

		It("skips an incomplete job", func() {
			job := newJob("foo", ns)
			job.Status.CompletionTime = nil
			h := newFakeHarvesterCtrl("foo", lw, logStderr, job)

			By("running the harvester")
			Expect(h.Run(ctx, job)).To(Succeed())
		})

		It("harvests a job's single pod", func() {
			job := newJob("foo", ns)
			parentLabels := labels.Set{"id": "1234-5678-9123"}
			// This is usually set by the Job controller.
			job.Spec.Selector = v1.SetAsLabelSelector(parentLabels)
			pod := newPod(podName1, ns, parentLabels, corev1.ContainerStatus{Name: "runner"})
			h := newFakeHarvesterCtrl("foo", lw, logStderr, job, pod)

			By("verifying ttlSecondsAfterFinished is unset")
			Expect(job.Spec.TTLSecondsAfterFinished).To(BeNil())

			By("running the harvester")
			Expect(h.Run(ctx, job)).To(Succeed())

			By("checking pod logs")
			Expect(podLogBuf.String()).To(Equal("fake logs"))

			By("setting ttlSecondsAfterFinished to 0")
			updatedJob := &batchv1.Job{}
			Expect(h.ctrlClient.Get(ctx, client.ObjectKeyFromObject(job), updatedJob)).To(Succeed())
			zero := new(int32)
			Expect(updatedJob.Spec.TTLSecondsAfterFinished).To(Equal(zero))
		})

		It("removing all finalizers", func() {
			job := newJob("foo", ns)
			WithFinalizers(job)
			// Set some other finalizer so we know the harvester isn't removing them all.
			const otherFinalizer = "other/finalizer"
			job.Finalizers = append(job.Finalizers, otherFinalizer)
			parentLabels := labels.Set{"id": "1234-5678-9123"}
			// This is usually set by the Job controller.
			job.Spec.Selector = v1.SetAsLabelSelector(parentLabels)
			pod := newPod(podName1, ns, parentLabels, corev1.ContainerStatus{Name: "runner"})
			pod.Finalizers = append(pod.Finalizers, jobFinalizer)
			h := newFakeHarvesterCtrl("foo", lw, logStderr, job, pod)
			// Simulate controller caller log setup.
			h.logger = h.logger.WithValues("jobName", job.Name, "jobNamespace", job.Namespace)

			By("verifying ttlSecondsAfterFinished is unset")
			Expect(job.Spec.TTLSecondsAfterFinished).To(BeNil())

			By("running the harvester")
			Expect(h.Run(ctx, job)).To(Succeed())

			By("checking for pod finalizer removal")
			updatedPod := &corev1.Pod{}
			Expect(h.ctrlClient.Get(ctx, client.ObjectKeyFromObject(pod), updatedPod)).To(Succeed())
			Expect(updatedPod.Finalizers).To(HaveLen(0))

			By("checking for job finalizer removal")
			updatedJob := &batchv1.Job{}
			Expect(h.ctrlClient.Get(ctx, client.ObjectKeyFromObject(job), updatedJob)).To(Succeed())
			Expect(updatedJob.Finalizers).To(Equal([]string{otherFinalizer}))

			By("checking pod logs")
			Expect(podLogBuf.String()).To(Equal("fake logs"))

			By("setting ttlSecondsAfterFinished to 0")
			zero := new(int32)
			Expect(job.Spec.TTLSecondsAfterFinished).To(Equal(zero))
		})

		It("stream two pod's logs", func() {
			job := newJob("foo", ns)
			parentLabels := labels.Set{"id": "1234-5678-9123"}
			// This is usually set by the Job controller.
			job.Spec.Selector = v1.SetAsLabelSelector(parentLabels)
			pod1 := newPod(podName1, ns, parentLabels, corev1.ContainerStatus{Name: "runner"}, corev1.ContainerStatus{Name: "do-er"})
			pod2 := newPod("foo-4567", ns, parentLabels, corev1.ContainerStatus{Name: "runner"}, corev1.ContainerStatus{Name: "do-er"})
			h := newFakeHarvesterCtrl("foo", lw, logStderr, job, pod1, pod2)

			By("running the harvester")
			Expect(h.Run(ctx, job)).To(Succeed())

			By("decoding Run logs")
			logs := decodeLogs(logStderr)

			By("checking pod iterator logs")
			lls := []logLine{
				{PodName: podName1, PodNamespace: ns, Container: "runner", Msg: "streaming logs"},
				{PodName: podName1, PodNamespace: ns, Container: "do-er", Msg: "streaming logs"},
				{PodName: "foo-4567", PodNamespace: ns, Container: "runner", Msg: "streaming logs"},
				{PodName: "foo-4567", PodNamespace: ns, Container: "do-er", Msg: "streaming logs"},
			}
			Expect(logs).To(ContainElements(lls))

			By("checking pod logs")
			Expect(podLogBuf.String()).To(Equal("fake logsfake logsfake logsfake logs"))
		})

		It("fails to find job in the same namespace different name", func() {
			otherJob := newJob("foo", ns)
			h := newFakeHarvesterCtrl("bar", lw, logStderr, otherJob)

			By("running the harvester")
			Expect(h.Run(ctx, otherJob)).To(Succeed())

			By("checking pod logs")
			Expect(podLogBuf.Len()).To(Equal(0))
		})

		It("fails to find job in a different namespace same name", func() {
			otherJob := newJob("foo", "other")
			h := newFakeHarvesterCtrl("foo", lw, logStderr, otherJob)

			By("running the harvester")
			Expect(h.Run(ctx, otherJob)).To(Succeed())

			By("checking pod logs")
			Expect(podLogBuf.Len()).To(Equal(0))
		})
	})
})

func newJob(name, namespace string) *batchv1.Job { //nolint:unparam
	job := &batchv1.Job{}
	job.Name = name
	job.Namespace = namespace
	now := v1.Now()
	job.Status.CompletionTime = &now
	return job
}

func newPod(name, namespace string, lbls labels.Set, ctrStatuses ...corev1.ContainerStatus) *corev1.Pod { //nolint:unparam
	pod := &corev1.Pod{}
	pod.Name = name
	pod.Namespace = namespace
	pod.Labels = lbls
	pod.Status.ContainerStatuses = ctrStatuses
	return pod
}

func newFakeHarvesterCtrl(name string, lw LogWriter, logStderr io.Writer, objs ...client.Object) *harvester {
	h := &harvester{name: name}
	h.logger = zap.New(zap.WriteTo(logStderr), zap.Level(zapcore.DebugLevel))
	h.lw = lw
	runtimeObjs := make([]runtime.Object, len(objs))
	for i := range objs {
		runtimeObjs[i] = objs[i]
	}
	h.k8sClient = k8sfake.NewSimpleClientset(runtimeObjs...)
	h.ctrlClient = ctrlfake.NewClientBuilder().WithObjects(objs...).Build()
	return h
}

type logLine struct {
	JobName      string
	JobNamespace string
	PodName      string
	PodNamespace string
	Container    string
	Msg          string
	Error        string
}

func decodeLogs(r io.Reader) (logs []logLine) {
	dec := json.NewDecoder(r)
	for dec.More() {
		var ll logLine
		ExpectWithOffset(1, dec.Decode(&ll)).To(Succeed())
		logs = append(logs, ll)
	}
	return logs
}
