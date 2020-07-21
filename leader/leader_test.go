package leader

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestBecome(t *testing.T) {
	err := Become(context.TODO(), "testOperator", "foobar")
	if err == nil {
		t.Fatal("expected it to fail")
	}
}

func TestGetOperatorNamespace(t *testing.T) {
	namespaceDir = "/tmp/namespace"

	testCases := []struct {
		name        string
		expected    string
		expectedErr bool
	}{
		{
			name:        "no namespace available",
			expectedErr: true,
		},
	}

	// test
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := getOperatorNamespace()
			if tc.expectedErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			}
			if actual != tc.expected {
				t.Fatalf("expected %v received %v", tc.expected, actual)
			}
		})
	}
}

func TestIsPodEvicted(t *testing.T) {
	testCases := []struct {
		name     string
		pod      corev1.Pod
		expected bool
	}{
		{
			name:     "Evicted pod returns true",
			expected: true,
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase:  corev1.PodFailed,
					Reason: "Evicted",
				},
			},
		},
		{
			name:     "Failed pod but not evicted returns false",
			expected: false,
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase:  corev1.PodFailed,
					Reason: "Don't know",
				},
			},
		},
		{
			name:     "Succeeded pod returns false",
			expected: false,
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
		},
		{
			name:     "Invalid reason for pod returns false",
			expected: false,
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase:  corev1.PodSucceeded,
					Reason: "Evicted",
				},
			},
		},
	}

	// test
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := isPodEvicted(tc.pod)
			if actual != tc.expected {
				t.Fatalf("expected %v received %v", tc.expected, actual)
			}
		})
	}

}
