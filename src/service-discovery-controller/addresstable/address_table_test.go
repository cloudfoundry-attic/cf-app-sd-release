package addresstable_test

import (
	"service-discovery-controller/addresstable"
	"sync"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddressTable", func() {
	var (
		table              *addresstable.AddressTable
		fakeClock          *fakeclock.FakeClock
		stalenessThreshold time.Duration
		pruningInterval    time.Duration
	)
	BeforeEach(func() {
		fakeClock = fakeclock.NewFakeClock(time.Now())
		stalenessThreshold = 5 * time.Second
		pruningInterval = 1 * time.Second
		table = addresstable.NewAddressTable(stalenessThreshold, pruningInterval, fakeClock)
	})
	AfterEach(func() {
		table.Shutdown()
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
				table.Add([]string{"stale.com"}, "192.0.0.1")
				table.Add([]string{"fresh.updated.com"}, "192.0.0.2")

				fakeClock.Increment(stalenessThreshold - 1*time.Second)

				By("adding/updating routes to make them fresh", func() {
					table.Add([]string{"fresh.updated.com"}, "192.0.0.2")
					table.Add([]string{"fresh.just.added.com"}, "192.0.0.3")
				})

				fakeClock.Increment(1001 * time.Millisecond)
			})
			It("excludes stale routes", func() {
				Eventually(func() []string { return table.Lookup("stale.com") }).Should(Equal([]string{}))
				Eventually(func() []string { return table.Lookup("fresh.updated.com") }).Should(Equal([]string{"192.0.0.2"}))
				Eventually(func() []string { return table.Lookup("fresh.just.added.com") }).Should(Equal([]string{"192.0.0.3"}))
			})
		})
	})

	Describe("Shutdown", func() {
		It("stops pruning", func() {
			table.Add([]string{"foo.com"}, "192.0.0.1")
			table.Shutdown()
			fakeClock.Increment(stalenessThreshold + time.Second)
			Consistently(func() []string { return table.Lookup("foo.com") }).Should(Equal([]string{"192.0.0.1"}))
			Expect(fakeClock.WatcherCount()).To(Equal(0))
		})
	})

	Describe("Concurrency", func() {
		It("handles requests properly", func() {
			var wg sync.WaitGroup
			wg.Add(3)
			go func() {
				table.Add([]string{"foo.com"}, "192.0.0.2")
				wg.Done()
			}()
			go func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
				wg.Done()
			}()
			go func() {
				fakeClock.Increment(stalenessThreshold - time.Second)
				wg.Done()
			}()
			Eventually(func() []string { return table.Lookup("foo.com") }).Should(ConsistOf([]string{
				"192.0.0.1",
				"192.0.0.2",
			}))
			wg.Wait()

			wg.Add(3)
			table.Add([]string{"foo.com"}, "192.0.0.2")
			go func() {
				fakeClock.Increment(stalenessThreshold - time.Second)
				wg.Done()
			}()
			go func() {
				table.Remove([]string{"foo.com"}, "192.0.0.2")
				wg.Done()
			}()
			go func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
				wg.Done()
			}()
			Eventually(func() []string { return table.Lookup("foo.com") }).Should(ConsistOf([]string{}))
			wg.Wait()
		})
	})
})
