package metrics_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-lib/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"strings"
)

var _ = Describe("OperatorHealth", func() {
	Context("NewOperatorHealth", func() {
		It("should create a new OperatorHealth", func() {
			m := metrics.NewOperatorHealth("test-operator")
			Expect(testutil.CollectAndCompare(m, strings.NewReader(expectedUnknown),
				"operator_lib_operator_healthy",
				"operator_lib_operator_degraded",
				"operator_lib_operator_unhealthy",
				"operator_lib_operator_health_unknown")).To(Succeed())
		})
	})

	Context("Register", func() {
		It("should register the OperatorHealth with the provided registry", func() {
			reg := prometheus.NewRegistry()
			m := metrics.NewOperatorHealth("test-operator")
			err := m.Register(reg)
			Expect(err).NotTo(HaveOccurred())

			Expect(testutil.CollectAndCompare(m, strings.NewReader(expectedUnknown),
				"operator_lib_operator_healthy",
				"operator_lib_operator_degraded",
				"operator_lib_operator_unhealthy",
				"operator_lib_operator_health_unknown")).To(Succeed())
		})
	})

	Context("Set", func() {
		var m *metrics.OperatorHealth
		BeforeEach(func() {
			m = metrics.NewOperatorHealth("test-operator")
		})

		It("should set the operator health metric to healthy", func() {
			Expect(m.Set(metrics.OperatorHealthHealthy)).To(Succeed())
			Expect(testutil.CollectAndCompare(m, strings.NewReader(expectedHealthy),
				"operator_lib_operator_healthy",
				"operator_lib_operator_degraded",
				"operator_lib_operator_unhealthy",
				"operator_lib_operator_health_unknown")).To(Succeed())
		})
		It("should set the operator health metric to degraded", func() {
			Expect(m.Set(metrics.OperatorHealthDegraded)).To(Succeed())
			Expect(testutil.CollectAndCompare(m, strings.NewReader(expectedDegraded),
				"operator_lib_operator_healthy",
				"operator_lib_operator_degraded",
				"operator_lib_operator_unhealthy",
				"operator_lib_operator_health_unknown")).To(Succeed())
		})
		It("should set the operator health metric to unhealthy", func() {
			Expect(m.Set(metrics.OperatorHealthUnhealthy)).To(Succeed())
			Expect(testutil.CollectAndCompare(m, strings.NewReader(expectedUnhealthy),
				"operator_lib_operator_healthy",
				"operator_lib_operator_degraded",
				"operator_lib_operator_unhealthy",
				"operator_lib_operator_health_unknown")).To(Succeed())
		})
		It("should set the operator health metric to unknown", func() {
			Expect(m.Set(metrics.OperatorHealthUnknown)).To(Succeed())
			Expect(testutil.CollectAndCompare(m, strings.NewReader(expectedUnknown),
				"operator_lib_operator_healthy",
				"operator_lib_operator_degraded",
				"operator_lib_operator_unhealthy",
				"operator_lib_operator_health_unknown")).To(Succeed())
		})
	})
})

var (
	expectedHealthy = `# HELP operator_lib_operator_healthy Whether the operator is healthy. Value 1 means healthy, 0 means not healthy.
# TYPE operator_lib_operator_healthy gauge
operator_lib_operator_healthy{operator_name="test-operator"} 1
# HELP operator_lib_operator_degraded Whether the operator is degraded. Value 1 means degraded, 0 means not degraded.
# TYPE operator_lib_operator_degraded gauge
operator_lib_operator_degraded{operator_name="test-operator"} 0
# HELP operator_lib_operator_unhealthy Whether the operator is unhealthy. Value 1 means unhealthy, 0 means not unhealthy.
# TYPE operator_lib_operator_unhealthy gauge
operator_lib_operator_unhealthy{operator_name="test-operator"} 0
# HELP operator_lib_operator_health_unknown Whether the operator health is unknown. Value 1 means unknown, 0 means not unknown.
# TYPE operator_lib_operator_health_unknown gauge
operator_lib_operator_health_unknown{operator_name="test-operator"} 0
`
	expectedDegraded = `# HELP operator_lib_operator_healthy Whether the operator is healthy. Value 1 means healthy, 0 means not healthy.
# TYPE operator_lib_operator_healthy gauge
operator_lib_operator_healthy{operator_name="test-operator"} 0
# HELP operator_lib_operator_degraded Whether the operator is degraded. Value 1 means degraded, 0 means not degraded.
# TYPE operator_lib_operator_degraded gauge
operator_lib_operator_degraded{operator_name="test-operator"} 1
# HELP operator_lib_operator_unhealthy Whether the operator is unhealthy. Value 1 means unhealthy, 0 means not unhealthy.
# TYPE operator_lib_operator_unhealthy gauge
operator_lib_operator_unhealthy{operator_name="test-operator"} 0
# HELP operator_lib_operator_health_unknown Whether the operator health is unknown. Value 1 means unknown, 0 means not unknown.
# TYPE operator_lib_operator_health_unknown gauge
operator_lib_operator_health_unknown{operator_name="test-operator"} 0
`

	expectedUnhealthy = `# HELP operator_lib_operator_healthy Whether the operator is healthy. Value 1 means healthy, 0 means not healthy.
# TYPE operator_lib_operator_healthy gauge
operator_lib_operator_healthy{operator_name="test-operator"} 0
# HELP operator_lib_operator_degraded Whether the operator is degraded. Value 1 means degraded, 0 means not degraded.
# TYPE operator_lib_operator_degraded gauge
operator_lib_operator_degraded{operator_name="test-operator"} 0
# HELP operator_lib_operator_unhealthy Whether the operator is unhealthy. Value 1 means unhealthy, 0 means not unhealthy.
# TYPE operator_lib_operator_unhealthy gauge
operator_lib_operator_unhealthy{operator_name="test-operator"} 1
# HELP operator_lib_operator_health_unknown Whether the operator health is unknown. Value 1 means unknown, 0 means not unknown.
# TYPE operator_lib_operator_health_unknown gauge
operator_lib_operator_health_unknown{operator_name="test-operator"} 0
`

	expectedUnknown = `# HELP operator_lib_operator_healthy Whether the operator is healthy. Value 1 means healthy, 0 means not healthy.
# TYPE operator_lib_operator_healthy gauge
operator_lib_operator_healthy{operator_name="test-operator"} 0
# HELP operator_lib_operator_degraded Whether the operator is degraded. Value 1 means degraded, 0 means not degraded.
# TYPE operator_lib_operator_degraded gauge
operator_lib_operator_degraded{operator_name="test-operator"} 0
# HELP operator_lib_operator_unhealthy Whether the operator is unhealthy. Value 1 means unhealthy, 0 means not unhealthy.
# TYPE operator_lib_operator_unhealthy gauge
operator_lib_operator_unhealthy{operator_name="test-operator"} 0
# HELP operator_lib_operator_health_unknown Whether the operator health is unknown. Value 1 means unknown, 0 means not unknown.
# TYPE operator_lib_operator_health_unknown gauge
operator_lib_operator_health_unknown{operator_name="test-operator"} 1
`
)
