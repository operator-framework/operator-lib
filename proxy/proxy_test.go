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

	. "github.com/onsi/ginkgo"
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
		// TODO(asmacdo) seems like a bad idea to mess with the env
		BeforeEach(func() {
			os.Setenv("HTTPS_PROXY", "https_proxy_test")
			// 	os.Setenv("HTTP_PROXY", "http_proxy_test")
			// 	os.Setenv("NO_PROXY", "no_proxy_test")
		})
		It("returns a slice of environment variables", func() {
			envVars := ReadProxyVarsFromEnv()
			Expect(len(envVars)).To(Equal(6))
		})
		It("creates upper and lower case environment variables with the same value", func() {
			envVars := ReadProxyVarsFromEnv()
			// dumb, err := checkValueFromEnvObj("HTTPS_PROXY", envVars)
			// Expect(err).To(BeNil())
			// Expect(dumb).To(Equal("https_proxy_test"))

			// Kinda dumb test if they are all empty string
			for _, envName := range ProxyEnvNames {
				upperValue, err := checkValueFromEnvObj(envName, envVars)
				Expect(err).To(BeNil())
				lowerValue, err := checkValueFromEnvObj(strings.ToLower(envName), envVars)
				Expect(err).To(BeNil())
				Expect(upperValue).To(Equal(lowerValue))
			}
		})
	})
})
