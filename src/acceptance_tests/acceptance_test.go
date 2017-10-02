package acceptance_tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Acceptance", func() {

	It("happycase", func() {
		println("acceptance tests says hi")
		Expect(true).To(BeTrue())
	})

})
