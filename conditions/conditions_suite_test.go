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

package conditions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "github.com/operator-framework/api/pkg/operators/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Conditions Suite", []Reporter{printer.NewlineReporter{}, printer.NewProwReporter("Conditions Suite")})
}

var testenv *envtest.Environment
var cfg *rest.Config
var sch = runtime.NewScheme()
var err error
var tempDir = fmt.Sprintf("%s_%d", "temp", rand.Int63nRange(0, 1000000))

const (
	olmYAMLURL  = "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.17.0/olm.yaml"
	crdsYAMLURL = "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.17.0/crds.yaml"

	// TODO: Remove this once OLM releases operator conditions CRD set
	condCRDYAML = "https://raw.githubusercontent.com/dinhxuanvu/operator-lifecycle-manager/create-operatorconditions-for-operator/deploy/chart/crds/0000_50_olm_00-operatorconditions.crd.yaml"
)

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	err = getOLMManifests()
	Expect(err).NotTo(HaveOccurred())
	// Add operator apiv1 to scheme
	err = apiv1.AddToScheme(sch)
	Expect(err).NotTo(HaveOccurred())

	testenv = &envtest.Environment{}
	testenv.CRDInstallOptions = envtest.CRDInstallOptions{
		Paths: []string{tempDir},
	}

	cfg, err = testenv.Start()
	Expect(err).NotTo(HaveOccurred())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	// remove tmp folder
	os.RemoveAll(tempDir)
	Expect(err).NotTo(HaveOccurred())
	Expect(testenv.Stop()).To(Succeed())
})

func getOLMManifests() error {
	// create a directory
	cmd := exec.Command("mkdir", tempDir)
	err := cmd.Run()
	if err != nil {
		return err
	}

	// fetch manifests to install olm
	err = getYAML(filepath.Join(tempDir, "olm.yaml"), olmYAMLURL)
	if err != nil {
		return fmt.Errorf("error fetching olm.yaml %v", err)
	}

	err = getYAML(filepath.Join(tempDir, "crds.yaml"), crdsYAMLURL)
	if err != nil {
		return fmt.Errorf("error fetching crds.yaml %v", err)
	}

	err = getYAML(filepath.Join(tempDir, "operatorconditions.crd.yaml"), condCRDYAML)
	if err != nil {
		return fmt.Errorf("error fetching operator conditions crd %v", err)
	}
	return nil
}

func getYAML(file, url string) error {
	cmd := exec.Command("curl", "-sSLo", file, url)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
