package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/util/workqueue"
)

var _ = Describe("EventHandler", func() {
	var q workqueue.RateLimitingInterface
	var instance EnqueueRequestForAnnotation
	var mapper meta.RESTMapper
	var pod *corev1.Pod
	var podOwner = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "podOwnerNs",
			Name:      "podOwnerName",
		},
	}

	// t := true
	BeforeEach(func() {
		q = controllertest.Queue{Interface: workqueue.New()}
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "biz",
				Name:      "biz",
			},
		}

		err := SetOwnerAnnotation(podOwner, pod, schema.GroupKind{Group: "Pods", Kind: "core"})
		Expect(err).To(BeNil())
		Expect(cfg).NotTo(BeNil())
		mapper, err = apiutil.NewDiscoveryRESTMapper(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(mapper).NotTo(BeNil())
	})

	Describe("EnqueueRequestForAnnotation", func() {
		It("should enqueue a Request with the annotations of the object in the CreateEvent", func() {
			instance = EnqueueRequestForAnnotation{
				Type: schema.GroupKind{
					Group: "Pods",
					Kind:  "core",
				}}

			evt := event.CreateEvent{
				Object: pod,
				Meta:   pod.GetObjectMeta(),
			}

			instance.Create(evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))

		})
	})
})
