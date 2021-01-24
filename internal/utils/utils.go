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
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const (
	// SAFileDefaultLocation default location of the service account namespace file
	SAFileDefaultLocation = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	// SAFileLocationEnv is the name of the environment variable that holds the service
	// account file location file. It is not set by default, but setting it allows operator
	// developers set different file location, because the default path may not be accessible
	// on the development environment. If not set, the default path will be used.
	SAFileLocationEnv = "SA_FILE_PATH"

	// OperatorNamespaceEnv the name of the environm,ent variable that holds the namespace.
	// If set, the GetOperatorNamespace method returns its value. If not, the method read the
	// service account file.
	OperatorNamespaceEnv = "OPERATOR_NAMESPACE"
)

// ErrNoNamespace indicates that a namespace could not be found for the current
// environment
var ErrNoNamespace = fmt.Errorf("namespace not found for current environment")

var readSAFile = func() ([]byte, error) {
	saFileLocation, found := os.LookupEnv(SAFileLocationEnv)
	if !found {
		saFileLocation = SAFileDefaultLocation
	}
	return ioutil.ReadFile(saFileLocation)
}

// GetOperatorNamespace returns the namespace the operator should be running in from
// the associated service account secret.
var GetOperatorNamespace = func() (string, error) {
	if ns := strings.TrimSpace(os.Getenv(OperatorNamespaceEnv)); ns != "" {
		return ns, nil
	}

	nsBytes, err := readSAFile()
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoNamespace
		}
		return "", err
	}
	ns := strings.TrimSpace(string(nsBytes))
	return ns, nil
}
