// Copyright 2020 The Operator-SDK Authors
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

package utils

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers test", func() {
	Describe("GetOperatorNamespace", func() {
		var origReadSAFile = readSAFile
		AfterEach(func() {
			readSAFile = origReadSAFile
		})
		const testNamespace = "testnamespace"
		It("should return error when namespace not found", func() {
			readSAFile = func() ([]byte, error) {
				return nil, os.ErrNotExist
			}
			namespace, err := GetOperatorNamespace()
			Expect(err).To(Equal(ErrNoNamespace))
			Expect(namespace).To(Equal(""))
		})
		It("should return namespace", func() {
			readSAFile = func() ([]byte, error) {
				return []byte(testNamespace), nil
			}

			// test
			namespace, err := GetOperatorNamespace()
			Expect(err).Should(BeNil())
			Expect(namespace).To(Equal(testNamespace))
		})
		It("should trim whitespace from namespace", func() {
			readSAFile = func() ([]byte, error) {
				return []byte("   " + testNamespace + "    "), nil
			}

			// test
			namespace, err := GetOperatorNamespace()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(namespace).To(Equal(testNamespace))
		})
		Context("read namespace from environment variable", func() {
			var originalVal string
			JustBeforeEach(func() {
				originalVal = os.Getenv(OperatorNamespaceEnv)
			})
			JustAfterEach(func() {
				err := os.Setenv(OperatorNamespaceEnv, originalVal)
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("should return the env var value, if set", func() {
				err := os.Setenv(OperatorNamespaceEnv, testNamespace)
				Expect(err).ShouldNot(HaveOccurred())

				namespace, err := GetOperatorNamespace()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(namespace).To(Equal(testNamespace))
			})
			It("should trim spaces from the namespace", func() {
				err := os.Setenv(OperatorNamespaceEnv, "   "+testNamespace+"   ")
				Expect(err).ShouldNot(HaveOccurred())

				namespace, err := GetOperatorNamespace()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(namespace).To(Equal(testNamespace))
			})
			It("should return the namespace from a file if not the env var is not set", func() {
				readSAFile = func() ([]byte, error) {
					return []byte("namespace-from-file"), nil
				}
				err := os.Unsetenv(OperatorNamespaceEnv)
				Expect(err).ShouldNot(HaveOccurred())

				namespace, err := GetOperatorNamespace()
				Expect(err).Should(BeNil())
				Expect(namespace).To(Equal("namespace-from-file"))

			})
			It("should return the namespace from a file if not the env var is only spaces", func() {
				readSAFile = func() ([]byte, error) {
					return []byte("namespace-from-file"), nil
				}
				err := os.Setenv(OperatorNamespaceEnv, "   ")
				Expect(err).ShouldNot(HaveOccurred())

				namespace, err := GetOperatorNamespace()
				Expect(err).Should(BeNil())
				Expect(namespace).To(Equal("namespace-from-file"))
			})
		})
		Context("read namespace from non standard location", func() {
			var originalVal string
			JustBeforeEach(func() {
				originalVal = os.Getenv(SAFileLocationEnv)
			})
			JustAfterEach(func() {
				err := os.Setenv(SAFileLocationEnv, originalVal)
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("should return the env var value, if set", func() {
				err := os.Setenv(SAFileLocationEnv, getTestFilesDir()+"namespace")
				Expect(err).ShouldNot(HaveOccurred())

				namespace, err := GetOperatorNamespace()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(namespace).To(Equal(testNamespace))
			})
			It("should trim spaces from the namespace", func() {
				err := os.Setenv(SAFileLocationEnv, getTestFilesDir()+"namespaceWithSpaces")
				Expect(err).ShouldNot(HaveOccurred())

				namespace, err := GetOperatorNamespace()
				Expect(err).ShouldNot(HaveOccurred())

				Expect(namespace).To(Equal(testNamespace))
			})
			It("should return error if the file is not exists", func() {
				err := os.Setenv(SAFileLocationEnv, getTestFilesDir()+"notExists")
				Expect(err).ShouldNot(HaveOccurred())

				namespace, err := GetOperatorNamespace()
				Expect(err).Should(HaveOccurred())
				Expect(err).Should(Equal(ErrNoNamespace))

				Expect(namespace).Should(BeEmpty())
			})
		})
	})

})

// return the path to the test files directory
func getTestFilesDir() string {
	const (
		packageUnderTestPath = "internal/utils"
		testFileDir          = "/testfiles/"
	)

	wd, err := os.Getwd()
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	// if running form internal/utils/
	if strings.HasSuffix(wd, packageUnderTestPath) {
		return wd + testFileDir
	}

	// if running from repository root
	return packageUnderTestPath + testFileDir
}
