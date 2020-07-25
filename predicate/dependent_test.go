package predicate

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("DependentPredicate", func() {
	var (
		pred DependentPredicate
	)

	Describe("Create", func() {
		It("returns false", func() {
			e := makeCreateEventFor(&unstructured.Unstructured{})
			Expect(pred.Create(e)).To(BeFalse())
		})
	})

	Describe("Update", func() {
		var oldObj, newObj *unstructured.Unstructured
		When("objects are equal", func() {
			BeforeEach(func() {
				oldObj = &unstructured.Unstructured{}
				oldObj.SetUID("A")
				newObj = &unstructured.Unstructured{}
				newObj.SetUID("A")
			})

			It("should return false", func() {
				e := makeUpdateEventFor(oldObj, newObj)
				Expect(pred.Update(e)).To(BeFalse())
			})

			When("except status is different", func() {
				BeforeEach(func() {
					newObj.Object["status"] = "foo"
				})
				It("should return false", func() {
					e := makeUpdateEventFor(oldObj, newObj)
					Expect(pred.Update(e)).To(BeFalse())
				})
			})

			When("except resource version is different", func() {
				BeforeEach(func() {
					newObj.SetResourceVersion("bar")
				})
				It("should return false", func() {
					e := makeUpdateEventFor(oldObj, newObj)
					Expect(pred.Update(e)).To(BeFalse())
				})
			})
		})

		When("objects are different", func() {
			BeforeEach(func() {
				oldObj = &unstructured.Unstructured{}
				oldObj.SetUID("A")
				newObj = &unstructured.Unstructured{}
				newObj.SetUID("B")
			})

			It("should return true", func() {
				e := makeUpdateEventFor(oldObj, newObj)
				Expect(pred.Update(e)).To(BeTrue())
			})
		})
	})

	Describe("Delete", func() {
		It("returns true", func() {
			e := makeDeleteEventFor(&unstructured.Unstructured{})
			Expect(pred.Delete(e)).To(BeTrue())
		})
	})

	Describe("Generic", func() {
		It("returns false", func() {
			e := makeGenericEventFor(&unstructured.Unstructured{})
			Expect(pred.Generic(e)).To(BeFalse())
		})
	})
})
