package addresstable_test

import (
	"service-discovery-controller/addresstable"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddressTable", func() {
	var (
		table     *addresstable.AddressTable
		fakeClock *fakeclock.FakeClock
	)
	BeforeEach(func() {
		fakeClock = fakeclock.NewFakeClock(time.Now())
		table = addresstable.NewAddressTable(5*time.Second, 1*time.Second, fakeClock)
	})
	Describe("Add", func() {
		It("adds an endpoint", func() {
			table.Add([]string{"foo.com"}, "192.0.0.1")
			Expect(table.Lookup("foo.com.")).To(Equal([]string{"192.0.0.1"}))
		})

		Context("when two hostnames are registered to same ip address", func() {
			It("returns both IPs", func() {
				table.Add([]string{"foo.com", "bar.com"}, "192.0.0.2")
				Expect(table.Lookup("foo.com.")).To(Equal([]string{"192.0.0.2"}))
				Expect(table.Lookup("bar.com.")).To(Equal([]string{"192.0.0.2"}))
			})
		})

		Context("when two different ips are registered to same host name", func() {
			It("returns both IPs", func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
				table.Add([]string{"foo.com"}, "192.0.0.2")
				Expect(table.Lookup("foo.com.")).To(Equal([]string{"192.0.0.1", "192.0.0.2"}))
			})
		})

		Context("when ip address is already registered", func() {
			It("ignores the duplicate ip", func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
				table.Add([]string{"foo.com"}, "192.0.0.1")
				Expect(table.Lookup("foo.com")).To(Equal([]string{"192.0.0.1"}))
			})
		})
	})

	Describe("GetAllAddresses", func() {
		BeforeEach(func() {
			table.Add([]string{"foo.com"}, "192.0.0.1")
			table.Add([]string{"foo.com"}, "192.0.0.2")
			table.Add([]string{"bar.com"}, "192.0.0.4")
		})

		It("returns all addresses", func() {
			Expect(table.GetAllAddresses()).To(Equal(map[string][]string{
				"foo.com.": []string{"192.0.0.1", "192.0.0.2"},
				"bar.com.": []string{"192.0.0.4"},
			}))
		})
	})

	Describe("Remove", func() {
		It("removes an endpoint", func() {
			table.Add([]string{"foo.com"}, "192.0.0.1")
			table.Remove([]string{"foo.com"}, "192.0.0.1")
			Expect(table.Lookup("foo.com")).To(Equal([]string{}))
		})
		Context("when two hostnames are registered to same ip address", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com.", "bar.com"}, "192.0.0.2")
			})
			It("removes both IPs", func() {
				table.Remove([]string{"foo.com", "bar.com."}, "192.0.0.2")

				Expect(table.Lookup("foo.com")).To(Equal([]string{}))
				Expect(table.Lookup("bar.com")).To(Equal([]string{}))
			})
		})

		Context("when removing an IP for an endpoint for a hostname that has multiple endpoints", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com"}, "192.0.0.3")
				table.Add([]string{"foo.com"}, "192.0.0.4")
			})
			It("removes only the IPs", func() {
				table.Remove([]string{"foo.com"}, "192.0.0.3")
				Expect(table.Lookup("foo.com")).To(Equal([]string{"192.0.0.4"}))
			})
		})

		Context("when removing an IP that does not exist", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com"}, "192.0.0.2")
			})
			It("does not panic", func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
				Expect(table.Lookup("foo.com")).To(Equal([]string{"192.0.0.2"}))
			})
		})

		Context("when removing a host that does not exist", func() {
			It("does not panic", func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
				Expect(table.Lookup("foo.com")).To(Equal([]string{}))
			})
		})
	})

	Describe("Lookup", func() {
		It("returns an empty array for an unknown hostname", func() {
			Expect(table.Lookup("foo.com")).To(Equal([]string{}))
		})
		Context("when routes go stale", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
			})
			It("excludes stale routes", func() {
				fakeClock.Increment(4 * time.Second)

				Expect(table.Lookup("foo.com")).To(Equal([]string{"192.0.0.1"}))
				table.Add([]string{"bar.com"}, "192.0.0.2")

				fakeClock.Increment(1001 * time.Millisecond)
				Eventually(func() []string { return table.Lookup("foo.com") }).Should(Equal([]string{}))
				Eventually(func() []string { return table.Lookup("bar.com") }).Should(Equal([]string{"192.0.0.2"}))

			})

		})
	})

	Describe("Concurrency", func() {
		It("handles requests properly", func() {
			go func() {
				table.Add([]string{"foo.com"}, "192.0.0.2")
			}()
			go func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
			}()
			Eventually(func() []string { return table.Lookup("foo.com") }).Should(ConsistOf([]string{
				"192.0.0.1",
				"192.0.0.2",
			}))

			go func() {
				table.Remove([]string{"foo.com"}, "192.0.0.2")
			}()
			go func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
			}()
			Eventually(func() []string { return table.Lookup("foo.com") }).Should(ConsistOf([]string{}))
		})
	})
})
