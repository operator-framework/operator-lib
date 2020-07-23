package leader

import (
	"context"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Leader", func() {

	Describe("Become", func() {
		It("should return an error when POD_NAME is not set", func() {
			os.Unsetenv("POD_NAME")
			err := Become(context.TODO(), "leader-test")
			Expect(err).ShouldNot(BeNil())
		})
		// TODO: write a test to ensure Become works
	})
	Describe("isPodEvicted", func() {
		var (
			leaderPod *corev1.Pod
		)
		BeforeEach(func() {
			leaderPod = &corev1.Pod{}
		})
		It("should return false with an empty status", func() {
			Expect(isPodEvicted(*leaderPod)).To(Equal(false))
		})
		It("should return false if reason is incorrect", func() {
			leaderPod.Status.Phase = corev1.PodFailed
			leaderPod.Status.Reason = "invalid"
			Expect(isPodEvicted(*leaderPod)).To(Equal(false))
		})
		It("should return false if pod is in the wrong phase", func() {
			leaderPod.Status.Phase = corev1.PodRunning
			Expect(isPodEvicted(*leaderPod)).To(Equal(false))
		})
		It("should return true when Phase and Reason are set", func() {
			leaderPod.Status.Phase = corev1.PodFailed
			leaderPod.Status.Reason = "Evicted"
			Expect(isPodEvicted(*leaderPod)).To(Equal(true))
		})
	})
	Describe("getOperatorNamespace", func() {
		It("should return error when namespace not found", func() {
			namespace, err := getOperatorNamespace()
			Expect(err).To(Equal(ErrNoNamespace))
			Expect(namespace).To(Equal(""))
		})
		It("should return namespace", func() {

			nsFile, err := setupNamespace("testnamespace")
			if err != nil {
				Fail(err.Error())
			}
			defer os.Remove(nsFile.Name())
			readNamespace = func() ([]byte, error) {
				return ioutil.ReadFile(nsFile.Name())
			}

			// test
			namespace, err := getOperatorNamespace()
			Expect(err).Should(BeNil())
			Expect(namespace).To(Equal("testnamespace"))
		})
		It("should trim whitespace from namespace", func() {

			nsFile, err := setupNamespace("   testnamespace	   ")
			if err != nil {
				Fail(err.Error())
			}
			defer os.Remove(nsFile.Name())
			readNamespace = func() ([]byte, error) {
				return ioutil.ReadFile(nsFile.Name())
			}

			// test
			namespace, err := getOperatorNamespace()
			Expect(err).Should(BeNil())
			Expect(namespace).To(Equal("testnamespace"))
		})
	})
	Describe("myOwnerRef", func() {
		var (
			client crclient.Client
		)
		BeforeEach(func() {
			client = fake.NewFakeClient(
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mypod",
						Namespace: "testns",
					},
				},
			)
		})
		It("should return an error when POD_NAME is not set", func() {
			os.Unsetenv("POD_NAME")
			_, err := myOwnerRef(context.TODO(), client, "")
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error if no pod is found", func() {
			os.Setenv("POD_NAME", "thisisnotthepodyourelookingfor")
			_, err := myOwnerRef(context.TODO(), client, "")
			Expect(err).ShouldNot(BeNil())
		})
		It("should return the owner reference without error", func() {
			os.Setenv("POD_NAME", "mypod")
			owner, err := myOwnerRef(context.TODO(), client, "testns")
			Expect(err).Should(BeNil())
			Expect(owner.APIVersion).To(Equal("v1"))
			Expect(owner.Kind).To(Equal("Pod"))
			Expect(owner.Name).To(Equal("mypod"))
		})
	})
	Describe("getPod", func() {
		var (
			client crclient.Client
		)
		BeforeEach(func() {
			client = fake.NewFakeClient(
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mypod",
						Namespace: "testns",
					},
				},
			)
		})
		It("should return an error when POD_NAME is not set", func() {
			os.Unsetenv("POD_NAME")
			_, err := getPod(context.TODO(), nil, "")
			Expect(err).ShouldNot(BeNil())
		})
		It("should return an error if no pod is found", func() {
			os.Setenv("POD_NAME", "thisisnotthepodyourelookingfor")
			_, err := getPod(context.TODO(), client, "")
			Expect(err).ShouldNot(BeNil())
		})
		It("should return the pod with the given name", func() {
			os.Setenv("POD_NAME", "mypod")
			pod, err := getPod(context.TODO(), client, "testns")
			Expect(err).Should(BeNil())
			Expect(pod).ShouldNot(BeNil())
			Expect(pod.TypeMeta.APIVersion).To(Equal("v1"))
			Expect(pod.TypeMeta.Kind).To(Equal("Pod"))
		})
	})
})

func setupNamespace(ns string) (*os.File, error) {
	nsFile, err := ioutil.TempFile("/tmp", "operator-ns-test")
	if err != nil {
		return nil, err
	}
	if _, err := nsFile.Write([]byte(ns)); err != nil {
		return nil, err
	}
	return nsFile, nil
}
