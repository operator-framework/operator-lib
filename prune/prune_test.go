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
	"time"

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

var appLabels = map[string]string{"app": app}

var _ = Describe("Prune", func() {
	var (
		fakeClient client.Client
		fakeObj    client.Object
		podGVK     schema.GroupVersionKind
		jobGVK     schema.GroupVersionKind
	)
	BeforeEach(func() {
		testScheme, err := createSchemes()
		Expect(err).Should(BeNil())

		fakeClient = crFake.NewClientBuilder().WithScheme(testScheme).Build()
		fakeObj = &corev1.Pod{}

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

				Expect(NewRegistry().IsPrunable(obj)).Should(BeNil())
			})
		})

	})
	Describe("Pruner", func() {
		Describe("NewPruner()", func() {
			It("Should Return a New Pruner Object", func() {
				pruner, err := NewPruner(fakeClient, podGVK, myStrategy)
				Expect(err).Should(BeNil())
				Expect(pruner).ShouldNot(BeNil())
			})

			It("Should Return a New Pruner Object with Custom Configuration", func() {
				namespace := "namespace"
				labels := map[string]string{"app": "churro"}
				pruner, err := NewPruner(fakeClient,
					jobGVK,
					myStrategy,
					WithNamespace(namespace),
					WithLabels(labels))
				Expect(err).Should(BeNil())
				Expect(pruner).ShouldNot(BeNil())
				Expect(&pruner.registry).Should(Equal(DefaultRegistry()))
				Expect(pruner.namespace).Should(Equal(namespace))
				Expect(pruner.labels).Should(Equal(labels))
				Expect(pruner.strategy).ShouldNot(BeNil())
				Expect(pruner.gvk).Should(Equal(jobGVK))
				Expect(pruner.client).Should(Equal(fakeClient))
			})

			It("Should Error if schema.GroupVersionKind Parameter is empty", func() {
				// empty GVK struct
				pruner, err := NewPruner(fakeClient, schema.GroupVersionKind{}, myStrategy)
				Expect(err).ShouldNot(BeNil())
				Expect(err.Error()).Should(Equal("error when creating a new Pruner: gvk parameter can not be empty"))
				Expect(pruner).ShouldNot(BeNil())
			})
		})

		Describe("Prune()", func() {
			Context("Does not return an Error", func() {
				testPruneWithDefaultIsPrunableFunc := func(gvk schema.GroupVersionKind) {
					pruner, err := NewPruner(fakeClient, gvk, myStrategy, WithLabels(appLabels), WithNamespace(namespace))
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					prunedObjects, err := pruner.Prune(context.Background())
					Expect(err).Should(BeNil())
					Expect(len(prunedObjects)).Should(Equal(2))
				}
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

					testPruneWithDefaultIsPrunableFunc(podGVK)

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

					testPruneWithDefaultIsPrunableFunc(jobGVK)

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

					pruner, err := NewPruner(fakeClient, jobGVK, myStrategy, WithLabels(appLabels), WithNamespace(namespace))
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

					dryRunClient := newDryRunClient(fakeClient)
					pruner, err := NewPruner(dryRunClient, podGVK, myStrategy, WithLabels(appLabels), WithNamespace(namespace))
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

					pruner, err := NewPruner(fakeClient, jobGVK, myStrategy, WithLabels(appLabels), WithNamespace(namespace))
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					// IsPrunableFunc that throws Unprunable error
					errorPrunableFunc := func(obj client.Object) error {
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

					pruner, err := NewPruner(fakeClient, jobGVK, myStrategy, WithLabels(appLabels), WithNamespace(namespace))
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					// IsPrunableFunc that throws non Unprunable error
					errorPrunableFunc := func(obj client.Object) error {
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

					pruner, err := NewPruner(fakeClient, jobGVK, prunerStrategy, WithLabels(appLabels), WithNamespace(namespace))
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

					pruner, err := NewPruner(fakeClient, jobGVK, myStrategy, WithLabels(appLabels), WithNamespace(namespace))
					Expect(err).Should(BeNil())
					Expect(pruner).ShouldNot(BeNil())

					// IsPrunableFunc that returns nil but also deletes the object
					// so that it will throw an error when attempting to remove the object
					prunableFunc := func(obj client.Object) error {
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

		Describe("GVK()", func() {
			It("Should return the GVK field in the Pruner", func() {
				pruner, err := NewPruner(fakeClient, podGVK, myStrategy)
				Expect(err).Should(BeNil())
				Expect(pruner).ShouldNot(BeNil())
				Expect(pruner.GVK()).Should(Equal(podGVK))
			})
		})

		Describe("Labels()", func() {
			It("Should return the Labels field in the Pruner", func() {
				pruner, err := NewPruner(fakeClient, podGVK, myStrategy, WithLabels(appLabels))
				Expect(err).Should(BeNil())
				Expect(pruner).ShouldNot(BeNil())
				Expect(pruner.Labels()).Should(Equal(appLabels))
			})
		})

		Describe("Namespace()", func() {
			It("Should return the Namespace field in the Pruner", func() {
				pruner, err := NewPruner(fakeClient, podGVK, myStrategy, WithNamespace(namespace))
				Expect(err).Should(BeNil())
				Expect(pruner).ShouldNot(BeNil())
				Expect(pruner.Namespace()).Should(Equal(namespace))
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
			err := DefaultPodIsPrunable(pod)
			Expect(err).Should(BeNil())
		})

		It("Should Panic When client.Object is not of type 'Pod'", func() {
			// Create an Unstrutcured with GVK where Kind is not 'Pod'
			notPod := &unstructured.Unstructured{}

			defer expectPanic()

			// Run it through DefaultPodIsPrunable
			_ = DefaultPodIsPrunable(notPod)
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
			err := DefaultPodIsPrunable(pod)
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
			err := DefaultJobIsPrunable(job)
			Expect(err).Should(BeNil())
		})

		It("Should Return An Error When Kind Is Not 'Job'", func() {
			// Create an Unstrutcured with GVK where Kind is not 'Job'
			notJob := &unstructured.Unstructured{}

			defer expectPanic()

			// Run it through DefaultJobIsPrunable
			_ = DefaultJobIsPrunable(notJob)
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
			err := DefaultJobIsPrunable(job)
			Expect(err).ShouldNot(BeNil())
			var expectErr *Unprunable
			Expect(errors.As(err, &expectErr)).Should(BeTrue())
			Expect(expectErr.Reason).Should(Equal("Job has not completed"))
			Expect(expectErr.Obj).ShouldNot(BeNil())
			Expect(err.Error()).Should(ContainSubstring(fmt.Sprintf("unable to prune %s: Job has not completed", client.ObjectKeyFromObject(job))))
		})
	})

	Context("NewPruneByCountStrategy", func() {
		resources := createDatedResources()
		It("Should return the 3 oldest resources", func() {
			resourcesToRemove, err := NewPruneByCountStrategy(2)(context.Background(), resources)
			Expect(err).Should(BeNil())
			Expect(resourcesToRemove).Should(Equal(resources[2:]))
		})

		It("Should return nil", func() {
			resourcesToRemove, err := NewPruneByCountStrategy(5)(context.Background(), resources)
			Expect(err).Should(BeNil())
			Expect(resourcesToRemove).Should(BeNil())
		})
	})

	Context("NewPruneByDateStrategy", func() {
		resources := createDatedResources()
		It("Should return 2 resources", func() {
			date := time.Now().Add(time.Hour * time.Duration(2))
			resourcesToRemove, err := NewPruneByDateStrategy(date)(context.Background(), resources)
			Expect(err).Should(BeNil())
			Expect(len(resourcesToRemove)).Should(Equal(2))
		})

		It("Should return 0 resources", func() {
			date := time.Now().Add(time.Hour * time.Duration(24))
			resourcesToRemove, err := NewPruneByDateStrategy(date)(context.Background(), resources)
			Expect(err).Should(BeNil())
			Expect(len(resourcesToRemove)).Should(Equal(0))
		})
	})

})

// TODO(everettraven): Remove once https://github.com/kubernetes-sigs/controller-runtime/pull/1873 is released
//---
type dryRunClient struct {
	client.Client
}

func newDryRunClient(baseClient client.Client) client.Client {
	return dryRunClient{client.NewDryRunClient(baseClient)}
}

// Delete implements a dry run delete, that is currently broken in the latest release.
func (c dryRunClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

//---

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
	// Due to some weirdness in the way the fake client is set up we need to create our
	// Kubernetes objects via the unstructured.Unstructured method
	for i := 0; i < 3; i++ {
		job := &unstructured.Unstructured{}
		job.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "batch/v1",
			"kind":       "Job",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("churro%d", i),
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app": app,
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

// createDatedResources is a helper function to get an array of client.Object that have
// different CreationTimestamps to test the common strategy functions
func createDatedResources() []client.Object {
	var jobs []client.Object
	for i := 0; i < 5; i++ {
		job := &unstructured.Unstructured{}
		job.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "batch/v1",
			"kind":       "Job",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("churro%d", i),
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app": app,
				},
			},
			"status": map[string]interface{}{
				"completionTime": metav1.Now(),
			},
		})
		job.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))
		job.SetCreationTimestamp(metav1.NewTime(time.Now().Add(time.Hour * time.Duration(i))))

		jobs = append(jobs, job)
	}

	return jobs
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
func myIsPrunable(obj client.Object) error {
	return nil
}
