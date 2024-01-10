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

package proxy

import (
	"errors"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func checkValueFromEnvObj(name string, envVars []corev1.EnvVar) (string, error) {
	for i := range envVars {
		if envVars[i].Name == name {
			return envVars[i].Value, nil
		}
	}
	return "", errors.New("empty name")
}

var _ = Describe("Retrieving", func() {
	Describe("proxy environment variables", func() {
		It("returns a slice of environment variables that were set", func() {
			os.Setenv("HTTPS_PROXY", "https_proxy_test")
			os.Setenv("HTTP_PROXY", "http_proxy_test")
			os.Setenv("NO_PROXY", "no_proxy_test")
			envVars := ReadProxyVarsFromEnv()
			Expect(envVars).To(HaveLen(6))
		})
		It("does not return unset variables", func() {
			envVars := ReadProxyVarsFromEnv()
			Expect(envVars).To(BeEmpty())
		})

		It("creates upper and lower case environment variables with the same value", func() {
			os.Setenv("HTTPS_PROXY", "https_proxy_test")
			os.Setenv("HTTP_PROXY", "http_proxy_test")
			os.Setenv("NO_PROXY", "no_proxy_test")
			envVars := ReadProxyVarsFromEnv()

			for _, envName := range ProxyEnvNames {
				upperValue, err := checkValueFromEnvObj(envName, envVars)
				Expect(err).ToNot(HaveOccurred())
				lowerValue, err := checkValueFromEnvObj(strings.ToLower(envName), envVars)
				Expect(err).ToNot(HaveOccurred())
				Expect(upperValue).To(Equal(lowerValue))
			}
		})
		AfterEach(func() {
			_ = os.Unsetenv("HTTPS_PROXY")
			_ = os.Unsetenv("HTTP_PROXY")
			_ = os.Unsetenv("NO_PROXY")
		})
	})
})
