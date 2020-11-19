module github.com/operator-framework/operator-lib

go 1.15

require (
<<<<<<< HEAD
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver v0.19.3 // indirect
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	sigs.k8s.io/controller-runtime v0.7.0-alpha.6
=======
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/operator-framework/api v0.3.21-0.20201112205953-820e285e84ae
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/stretchr/testify v1.5.1 // indirect
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.3
	sigs.k8s.io/controller-runtime v0.6.1
>>>>>>> 215831e... Operator Conditions: Add helpers for olm operator conditions
)
