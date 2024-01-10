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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers test", func() {
	Describe("GetOperatorNamespace", func() {
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
				return []byte("testnamespace"), nil
			}

			// test
			namespace, err := GetOperatorNamespace()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(namespace).To(Equal("testnamespace"))
		})
		It("should trim whitespace from namespace", func() {
			readSAFile = func() ([]byte, error) {
				return []byte("   testnamespace    "), nil
			}

			// test
			namespace, err := GetOperatorNamespace()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(namespace).To(Equal("testnamespace"))
		})
	})

})
