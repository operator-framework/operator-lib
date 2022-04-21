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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crFake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const namespace = "default"
const app = "churro"

var _ = Describe("Prune", func() {
	var (
		fakeClient   client.Client
		fakeObj      client.Object
		prunerConfig PrunerOption
		podGVK       schema.GroupVersionKind
		jobGVK       schema.GroupVersionKind
	)
	BeforeEach(func() {
		testScheme, err := createSchemes()
		Expect(err).Should(BeNil())

		fakeClient = crFake.NewClientBuilder().WithScheme(testScheme).Build()
		fakeObj = &corev1.Pod{}

		// Create our function to configure our pruner
		prunerConfig = func(p *Pruner) {
			// Create the labels we want to select with
			labels := make(map[string]string)
			labels["app"] = app

			p.labels = labels
			p.namespace = namespace
		}

		podGVK = corev1.SchemeGroupVersion.WithKind("Pod")
		jobGVK = batchv1.SchemeGroupVersion.WithKind("Job")
	})

	Describe("Unprunable", func() {
		Describe("Error()", func() {
			It("Should Return a String Representation of Unprunable", func() {
				unpruneable := Unprunable{
					Obj:    &fakeObj,
					Reason: "TestReason",
				}
				Expect(unpruneable.Error()).To(Equal(fmt.Sprintf("unable to prune %s: %s", client.ObjectKeyFromObject(fakeObj), unpruneable.Reason)))
			})
		})
	})

	Describe("Registry", func() {
		Describe("NewRegistry()", func() {
			It("Should Return a New Registry Object", func() {
				registry := NewRegistry()
				Expect(registry).ShouldNot(BeNil())
			})
		})

		Describe("RegisterIsPrunableFunc()", func() {
			It("Should Add an Entry to Registry Prunables Map", func() {
				registry := NewRegistry()
				Expect(registry).ShouldNot(BeNil())

				registry.RegisterIsPrunableFunc(podGVK, myIsPrunable)
				Expect(registry.prunables).Should(HaveKey(podGVK))
			})
		})

		Describe("IsPrunable()", func() {
			It("Should Return 'nil' if object GVK is not found in Prunables Map", func() {
				obj := &unstructured.Unstructured{}
				obj.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "group",
					Version: "v1",
					Kind:    "NotReal",
				})

				Expect(NewRegistry().IsPrunable(obj, logr.Logger{})).Should(BeNil())
			})
		})

	})
	Describe("Pruner", func() {
		Describe("NewPruner()", func() {
			Context("Successful", func() {
				It("Should Return a New Pruner Object", func() {
					pruner, err := NewPruner(fakeClient, podGVK, myStrategy)
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())
				})

				It("Should Return a New Pruner Object with Custom Configuration", func() {
					namespace := "namespace"
					labels := map[string]string{"app": "churro"}
					logger := &logr.Logger{}
					pruner, err := NewPruner(fakeClient,
						jobGVK,
						myStrategy,
						WithNamespace(namespace),
						WithLabels(labels),
						WithLogger(*logger))

					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())
					Expect(&pruner.registry).Should(Equal(DefaultRegistry()))
					Expect(pruner.namespace).Should(Equal(namespace))
					Expect(pruner.labels).Should(Equal(labels))
					Expect(&pruner.logger).Should(Equal(logger))
					Expect(pruner.strategy).ShouldNot(BeNil())
					Expect(pruner.gvk).Should(Equal(jobGVK))
					Expect(pruner.client).Should(Equal(fakeClient))
				})
			})

			Context("Errors", func() {
				errorString := "error creating a new Pruner: explicit parameters cannot be nil or contain empty values"

				It("Should Error if client.Client Parameter is nil", func() {
					pruner, err := NewPruner(nil, podGVK, myStrategy)
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(Equal(errorString))
					Expect(pruner).ShouldNot(BeNil())
				})

				It("Should Error if schema.GroupVersionKind Parameter fields have empty values", func() {
					// empty GVK struct
					pruner, err := NewPruner(fakeClient, schema.GroupVersionKind{}, myStrategy)
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(Equal(errorString))
					Expect(pruner).ShouldNot(BeNil())

					// empty Version
					pruner, err = NewPruner(fakeClient, schema.GroupVersionKind{Group: "group", Kind: "kind"}, myStrategy)
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(Equal(errorString))
					Expect(pruner).ShouldNot(BeNil())

					// empty Kind
					pruner, err = NewPruner(fakeClient, schema.GroupVersionKind{Group: "group", Version: "version"}, myStrategy)
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(Equal(errorString))
					Expect(pruner).ShouldNot(BeNil())
				})

				It("Should Error if StrategyFunc parameter is nil", func() {
					pruner, err := NewPruner(fakeClient, podGVK, nil)
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(Equal(errorString))
					Expect(pruner).ShouldNot(BeNil())
				})
			})
		})

		Describe("Prune()", func() {
			Context("Does not return an Error", func() {
				It("Should Prune Pods with Default IsPrunableFunc", func() {
					// Create the test resources - in this case Pods
					err := createTestPods(fakeClient)
					Expect(err).Should(BeNil())

					// Make sure the pod resources are properly created
					pods := &unstructured.UnstructuredList{}
					pods.SetGroupVersionKind(podGVK)
					err = fakeClient.List(context.Background(), pods)
					Expect(err).Should(BeNil())
					Expect(len(pods.Items)).Should(Equal(3))

					pruner, err := NewPruner(fakeClient, podGVK, myStrategy, prunerConfig)
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					prunedObjects, err := pruner.Prune(context.Background())
					Expect(err).Should(BeNil())
					Expect(len(prunedObjects)).Should(Equal(2))

					// Get a list of the Pods to make sure we have pruned the ones we expected
					err = fakeClient.List(context.Background(), pods)
					Expect(err).Should(BeNil())
					Expect(len(pods.Items)).Should(Equal(1))
				})

				It("Should Prune Jobs with Default IsPrunableFunc", func() {
					// Create the test resources - in this case Jobs
					err := createTestJobs(fakeClient)
					Expect(err).Should(BeNil())

					// Make sure the job resources are properly created
					jobs := &unstructured.UnstructuredList{}
					jobs.SetGroupVersionKind(jobGVK)
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(3))

					pruner, err := NewPruner(fakeClient, jobGVK, myStrategy, prunerConfig)
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					prunedObjects, err := pruner.Prune(context.Background())
					Expect(err).Should(BeNil())
					Expect(len(prunedObjects)).Should(Equal(2))

					// Get a list of the job to make sure we have pruned the ones we expected
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(1))
				})

				It("Should Remove Resource When Using a Custom IsPrunableFunc", func() {
					// Create the test resources - in this case Jobs
					err := createTestJobs(fakeClient)
					Expect(err).Should(BeNil())

					// Make sure the job resources are properly created
					jobs := &unstructured.UnstructuredList{}
					jobs.SetGroupVersionKind(jobGVK)
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(3))

					pruner, err := NewPruner(fakeClient, jobGVK, myStrategy, prunerConfig)
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					// Register our custom IsPrunableFunc
					RegisterIsPrunableFunc(jobGVK, myIsPrunable)

					prunedObjects, err := pruner.Prune(context.Background())
					Expect(err).Should(BeNil())
					Expect(len(prunedObjects)).Should(Equal(2))

					// Get a list of the jobs to make sure we have pruned the ones we expected
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(1))
				})

				It("Should Not Prune Resources when using a DryRunClient", func() {
					// Create the test resources - in this case Pods
					err := createTestPods(fakeClient)
					Expect(err).Should(BeNil())

					// Make sure the pod resources are properly created
					pods := &unstructured.UnstructuredList{}
					pods.SetGroupVersionKind(podGVK)
					err = fakeClient.List(context.Background(), pods)
					Expect(err).Should(BeNil())
					Expect(len(pods.Items)).Should(Equal(3))

					dryRunClient := client.NewDryRunClient(fakeClient)
					pruner, err := NewPruner(dryRunClient, podGVK, myStrategy, prunerConfig)
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					prunedObjects, err := pruner.Prune(context.Background())
					Expect(err).Should(BeNil())
					Expect(len(prunedObjects)).Should(Equal(2))

					// Get a list of the Pods to make sure we haven't pruned any
					err = fakeClient.List(context.Background(), pods)
					Expect(err).Should(BeNil())
					Expect(len(pods.Items)).Should(Equal(3))
				})

				It("Should Skip Pruning a Resource If IsPrunable Returns an Error of Type Unprunable", func() {
					// Create the test resources - in this case Jobs
					err := createTestJobs(fakeClient)
					Expect(err).Should(BeNil())

					// Make sure the job resources are properly created
					jobs := &unstructured.UnstructuredList{}
					jobs.SetGroupVersionKind(jobGVK)
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(3))

					pruner, err := NewPruner(fakeClient, jobGVK, myStrategy, prunerConfig)
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					// IsPrunableFunc that throws Unprunable error
					errorPrunableFunc := func(obj client.Object, logger logr.Logger) error {
						return &Unprunable{
							Obj:    &obj,
							Reason: "TEST",
						}
					}

					// Register our custom IsPrunableFunc
					RegisterIsPrunableFunc(jobGVK, errorPrunableFunc)

					prunedObjects, err := pruner.Prune(context.Background())
					Expect(err).Should(BeNil())
					Expect(len(prunedObjects)).Should(Equal(0))

					// Get a list of the jobs to make sure we have pruned the ones we expected
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(3))
				})

			})
			Context("Returns an Error", func() {
				It("Should Return an Error if IsPrunableFunc Returns an Error That is not of Type Unprunable", func() {
					// Create the test resources - in this case Jobs
					err := createTestJobs(fakeClient)
					Expect(err).Should(BeNil())

					// Make sure the job resources are properly created
					jobs := &unstructured.UnstructuredList{}
					jobs.SetGroupVersionKind(jobGVK)
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(3))

					pruner, err := NewPruner(fakeClient, jobGVK, myStrategy, prunerConfig)
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					// IsPrunableFunc that throws non Unprunable error
					errorPrunableFunc := func(obj client.Object, logger logr.Logger) error {
						return fmt.Errorf("TEST")
					}

					// Register our custom IsPrunableFunc
					RegisterIsPrunableFunc(jobGVK, errorPrunableFunc)

					prunedObjects, err := pruner.Prune(context.Background())
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(Equal("TEST"))
					Expect(len(prunedObjects)).Should(Equal(0))

					// Get a list of the jobs to make sure we have pruned the ones we expected
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(3))
				})

				It("Should Return An Error If Strategy Function Returns An Error", func() {
					// Create the test resources - in this case Jobs
					err := createTestJobs(fakeClient)
					Expect(err).Should(BeNil())

					// Make sure the job resources are properly created
					jobs := &unstructured.UnstructuredList{}
					jobs.SetGroupVersionKind(jobGVK)
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(3))

					// strategy that will return an error
					prunerStrategy := func(ctx context.Context, objs []client.Object) ([]client.Object, error) {
						return nil, fmt.Errorf("TESTERROR")
					}

					pruner, err := NewPruner(fakeClient, jobGVK, prunerStrategy, prunerConfig)
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					// Register our custom IsPrunableFunc
					RegisterIsPrunableFunc(jobGVK, myIsPrunable)

					prunedObjects, err := pruner.Prune(context.Background())
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(Equal("error determining prunable objects: TESTERROR"))
					Expect(prunedObjects).Should(BeNil())

					// Get a list of the jobs to make sure we have pruned the ones we expected
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(3))
				})

				It("Should Return an Error if it can not Prune a Resource", func() {
					// Create the test resources - in this case Jobs
					err := createTestJobs(fakeClient)
					Expect(err).Should(BeNil())

					// Make sure the job resources are properly created
					jobs := &unstructured.UnstructuredList{}
					jobs.SetGroupVersionKind(jobGVK)
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(3))

					pruner, err := NewPruner(fakeClient, jobGVK, myStrategy, prunerConfig)
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					// IsPrunableFunc that returns nil but also deletes the object
					// so that it will throw an error when attempting to remove the object
					prunableFunc := func(obj client.Object, logger logr.Logger) error {
						_ = fakeClient.Delete(context.TODO(), obj, &client.DeleteOptions{})
						return nil
					}

					// Register our custom IsPrunableFunc
					RegisterIsPrunableFunc(jobGVK, prunableFunc)

					prunedObjects, err := pruner.Prune(context.Background())
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(ContainSubstring("error pruning object: jobs.batch \"churro1\" not found"))
					Expect(len(prunedObjects)).Should(Equal(0))

					// Get a list of the jobs to make sure we have pruned the ones we expected
					err = fakeClient.List(context.Background(), jobs)
					Expect(err).Should(BeNil())
					Expect(len(jobs.Items)).Should(Equal(0))
				})

			})
		})
	})

	Context("DefaultPodIsPrunable", func() {
		It("Should Return 'nil' When Criteria Is Met", func() {
			// Create a Pod Object
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      app,
					Namespace: namespace,
					Labels:    map[string]string{"app": app},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			}
			pod.SetGroupVersionKind(podGVK)

			// Run it through DefaultPodIsPrunable
			err := DefaultPodIsPrunable(pod, logr.Logger{})
			Expect(err).Should(BeNil())
		})

		It("Should Panic When client.Object is not of type 'Pod'", func() {
			// Create an Unstrutcured with GVK where Kind is not 'Pod'
			notPod := &unstructured.Unstructured{}

			defer expectPanic()

			// Run it through DefaultPodIsPrunable
			_ = DefaultPodIsPrunable(notPod, logr.Logger{})
		})

		It("Should Return An Error When Kind Is 'Pod' But Phase Is Not 'Succeeded'", func() {
			// Create a Pod Object
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      app,
					Namespace: namespace,
					Labels:    map[string]string{"app": app},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}
			pod.SetGroupVersionKind(podGVK)

			// Run it through DefaultPodIsPrunable
			err := DefaultPodIsPrunable(pod, logr.Logger{})
			Expect(err).ShouldNot(BeNil())
			var expectErr *Unprunable
			Expect(errors.As(err, &expectErr)).Should(BeTrue())
			Expect(expectErr.Reason).Should(Equal("Pod has not succeeded"))
			Expect(expectErr.Obj).ShouldNot(BeNil())
			Expect(err.Error()).Should(Equal(fmt.Sprintf("unable to prune %s: Pod has not succeeded", client.ObjectKeyFromObject(pod))))
		})
	})

	Context("DefaultJobIsPrunable", func() {
		It("Should Return 'nil' When Criteria Is Met", func() {
			// Create a Job Object
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      app,
					Namespace: namespace,
					Labels:    map[string]string{"app": app},
				},
				Status: batchv1.JobStatus{
					CompletionTime: &metav1.Time{Time: metav1.Now().Time},
				},
			}
			job.SetGroupVersionKind(jobGVK)

			// Run it through DefaultJobIsPrunable
			err := DefaultJobIsPrunable(job, logr.Logger{})
			Expect(err).Should(BeNil())
		})

		It("Should Return An Error When Kind Is Not 'Job'", func() {
			// Create an Unstrutcured with GVK where Kind is not 'Job'
			notJob := &unstructured.Unstructured{}

			defer expectPanic()

			// Run it through DefaultJobIsPrunable
			_ = DefaultJobIsPrunable(notJob, logr.Logger{})
		})

		It("Should Return An Error When Kind Is 'Job' But 'CompletionTime' is 'nil'", func() {
			// Create a Job Object
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      app,
					Namespace: namespace,
					Labels:    map[string]string{"app": app},
				},
				Status: batchv1.JobStatus{
					CompletionTime: nil,
				},
			}
			job.SetGroupVersionKind(jobGVK)

			// Run it through DefaultJobIsPrunable
			err := DefaultJobIsPrunable(job, logr.Logger{})
			Expect(err).ShouldNot(BeNil())
			var expectErr *Unprunable
			Expect(errors.As(err, &expectErr)).Should(BeTrue())
			Expect(expectErr.Reason).Should(Equal("Job has not completed"))
			Expect(expectErr.Obj).ShouldNot(BeNil())
			Expect(err.Error()).Should(ContainSubstring(fmt.Sprintf("unable to prune %s: Job has not completed", client.ObjectKeyFromObject(job))))
		})
	})

})

// create 3 pods and 3 jobs with different start times (now, 2 days old, 4 days old)
func createTestPods(client client.Client) error {
	// some defaults
	ns := namespace
	appLabel := app

	// Due to some weirdness in the way the fake client is set up we need to create our
	// Kubernetes objects via the unstructured.Unstructured method
	for i := 0; i < 3; i++ {
		pod := &unstructured.Unstructured{}
		pod.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "core/v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("churro%d", i),
				"namespace": ns,
				"labels": map[string]interface{}{
					"app": appLabel,
				},
			},
			"status": map[string]interface{}{
				"phase": "Succeeded",
			},
		})
		pod.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))

		err := client.Create(context.Background(), pod)
		if err != nil {
			return err
		}
	}

	return nil
}

// create 3 pods and 3 jobs with different start times (now, 2 days old, 4 days old)
func createTestJobs(client client.Client) error {
	// some defaults
	ns := namespace
	appLabel := app

	// Due to some weirdness in the way the fake client is set up we need to create our
	// Kubernetes objects via the unstructured.Unstructured method
	for i := 0; i < 3; i++ {
		job := &unstructured.Unstructured{}
		job.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "batch/v1",
			"kind":       "Job",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("churro%d", i),
				"namespace": ns,
				"labels": map[string]interface{}{
					"app": appLabel,
				},
			},
			"status": map[string]interface{}{
				"completionTime": metav1.Now(),
			},
		})
		job.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))

		err := client.Create(context.Background(), job)
		if err != nil {
			return err
		}
	}

	return nil
}

// createSchemes is a helper function to set up the schemes needed to run
// our tests utilizing controller-runtime's fake client
func createSchemes() (*runtime.Scheme, error) {
	corev1SchemeBuilder := &scheme.Builder{GroupVersion: corev1.SchemeGroupVersion}
	corev1SchemeBuilder.Register(&corev1.Pod{}, &corev1.PodList{})

	batchv1SchemeBuilder := &scheme.Builder{GroupVersion: batchv1.SchemeGroupVersion}
	batchv1SchemeBuilder.Register(&batchv1.Job{}, &batchv1.JobList{})

	outScheme := runtime.NewScheme()

	err := corev1SchemeBuilder.AddToScheme(outScheme)
	if err != nil {
		return nil, err
	}

	err = batchv1SchemeBuilder.AddToScheme(outScheme)
	if err != nil {
		return nil, err
	}

	return outScheme, nil
}

// myStrategy shows how you can write your own strategy
// In this example it simply removes a resource if it has
// the name 'churro1' or 'churro2'
func myStrategy(ctx context.Context, objs []client.Object) ([]client.Object, error) {
	var objsToRemove []client.Object

	for _, obj := range objs {
		// If the object has name churro1 or churro2 get rid of it
		if obj.GetName() == "churro1" || obj.GetName() == "churro2" {
			objsToRemove = append(objsToRemove, obj)
		}
	}

	return objsToRemove, nil
}

// expectPanic is a helper function for testing functions that are expected to panic
// when used it should be used with a defer statement before the function
// that is expected to panic is called
func expectPanic() {
	r := recover()
	Expect(r).ShouldNot(BeNil())
}

// myIsPrunable shows how you can write your own IsPrunableFunc
// In this example it simply removes all resources
func myIsPrunable(obj client.Object, logger logr.Logger) error {
	return nil
}
