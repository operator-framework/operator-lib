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

package conditions

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv2 "github.com/operator-framework/api/pkg/operators/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	conditionFoo apiv2.ConditionType = "conditionFoo"
	conditionBar apiv2.ConditionType = "conditionBar"
)

var _ = Describe("NewCondition", func() {
	var cl client.Client
	BeforeEach(func() {
		sch := runtime.NewScheme()
		err := apiv2.AddToScheme(sch)
		Expect(err).NotTo(HaveOccurred())
		cl = fake.NewClientBuilder().WithScheme(sch).Build()
	})

	testNewCondition(func(ct apiv2.ConditionType) (Condition, error) {
		return NewCondition(cl, ct)
	})
})

var _ = Describe("GetNamespacedName", func() {
	testGetNamespacedName(GetNamespacedName)
})

var _ = Describe("InClusterFactory", func() {
	var cl client.Client
	var f InClusterFactory

	BeforeEach(func() {
		sch := runtime.NewScheme()
		err := apiv2.AddToScheme(sch)
		Expect(err).NotTo(HaveOccurred())
		cl = fake.NewClientBuilder().WithScheme(sch).Build()
		f = InClusterFactory{cl}
	})

	Describe("NewCondition", func() {
		testNewCondition(f.NewCondition)
	})

	Describe("GetNamespacedName", func() {
		testGetNamespacedName(f.GetNamespacedName)
	})
})

func testNewCondition(fn func(apiv2.ConditionType) (Condition, error)) {
	It("should create a new condition", func() {
		err := os.Setenv(operatorCondEnvVar, "test-operator-condition")
		Expect(err).NotTo(HaveOccurred())
		readNamespace = func() (string, error) {
			return "default", nil
		}

		c, err := fn(conditionFoo)
		Expect(err).NotTo(HaveOccurred())
		Expect(c).NotTo(BeNil())
	})

	It("should error when namespacedName cannot be found", func() {
		err := os.Unsetenv(operatorCondEnvVar)
		Expect(err).NotTo(HaveOccurred())

		c, err := fn(conditionFoo)
		Expect(err).To(HaveOccurred())
		Expect(c).To(BeNil())
	})
}

func testGetNamespacedName(fn func() (*types.NamespacedName, error)) {
	It("should error when name of the operator condition cannot be found", func() {
		err := os.Unsetenv(operatorCondEnvVar)
		Expect(err).NotTo(HaveOccurred())

		objKey, err := fn()
		Expect(err).To(HaveOccurred())
		Expect(objKey).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("could not determine operator condition name"))
	})

	It("should error when object namespace cannot be found", func() {
		err := os.Setenv(operatorCondEnvVar, "test")
		Expect(err).NotTo(HaveOccurred())

		readNamespace = func() (string, error) {
			return "", os.ErrNotExist
		}

		objKey, err := fn()
		Expect(err).To(HaveOccurred())
		Expect(objKey).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("get operator condition namespace: file does not exist"))
	})

	It("should return the right namespaced name from SA namespace file", func() {
		err := os.Setenv(operatorCondEnvVar, "test")
		Expect(err).NotTo(HaveOccurred())

		readNamespace = func() (string, error) {
			return "testns", nil
		}
		objKey, err := fn()
		Expect(err).NotTo(HaveOccurred())
		Expect(objKey).NotTo(BeNil())
		Expect(objKey.Name).To(BeEquivalentTo("test"))
		Expect(objKey.Namespace).To(BeEquivalentTo("testns"))
	})
}

func deleteCondition(ctx context.Context, client client.Client, obj client.Object) {
	err := client.Delete(ctx, obj)
	Expect(err).NotTo(HaveOccurred())
}
