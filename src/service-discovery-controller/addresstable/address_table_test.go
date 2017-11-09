package addresstable_test

import (
	"service-discovery-controller/addresstable"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddressTable", func() {
	var (
		table *addresstable.AddressTable
	)
	BeforeEach(func() {
		table = addresstable.NewAddressTable()
	})
	Describe("Add", func() {
		It("adds an endpoint", func() {
			table.Add([]string{"foo.com"}, "192.0.0.1")
			result := table.Lookup("foo.com.")
			Expect(len(result)).To(Equal(1))
			Expect(result[0].IP).To(Equal("192.0.0.1"))
			Expect(result[0].Timestamp.String()).NotTo(Equal("0001-01-01 00:00:00 +0000 UTC"))
		})

		Context("when two hostnames are registered to same ip address", func() {
			It("returns both IPs", func() {
				table.Add([]string{"foo.com", "bar.com"}, "192.0.0.2")
				fooResult := table.Lookup("foo.com.")
				barResult := table.Lookup("bar.com.")
				Expect(len(fooResult)).To(Equal(1))
				Expect(len(barResult)).To(Equal(1))
				Expect(fooResult[0].IP).To(Equal("192.0.0.2"))
				Expect(fooResult[0].Timestamp.String()).NotTo(Equal("0001-01-01 00:00:00 +0000 UTC"))
				Expect(barResult[0].IP).To(Equal("192.0.0.2"))
				Expect(barResult[0].Timestamp.String()).NotTo(Equal("0001-01-01 00:00:00 +0000 UTC"))
			})
		})

		Context("when two different ips are registered to same host name", func() {
			It("returns both IPs", func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
				table.Add([]string{"foo.com"}, "192.0.0.2")
				ipAndTimes := table.Lookup("foo.com.")
				Expect(len(ipAndTimes)).To(Equal(2))
				ips := []string{}
				for _, ipAndTime := range ipAndTimes {
					ips = append(ips, ipAndTime.IP)
					Expect(ipAndTime.Timestamp.String()).NotTo(Equal("0001-01-01 00:00:00 +0000 UTC"))
				}
				Expect(ips).To(Equal([]string{"192.0.0.1", "192.0.0.2"}))
			})
		})

		Context("when ip address is already registered", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
			})

			It("ignores the duplicate ip", func() {
				table.Add([]string{"foo.com"}, "192.0.0.1")
				result := table.Lookup("foo.com.")
				Expect(len(result)).To(Equal(1))
				Expect(result[0].IP).To(Equal("192.0.0.1"))
				Expect(result[0].Timestamp.String()).NotTo(Equal("0001-01-01 00:00:00 +0000 UTC"))
			})

			It("updates the timestamp", func() {
				result := table.Lookup("foo.com.")
				Expect(len(result)).To(Equal(1))
				oldTimestamp := result[0].Timestamp

				table.Add([]string{"foo.com"}, "192.0.0.1")
				result = table.Lookup("foo.com.")
				Expect(len(result)).To(Equal(1))
				newTimestamp := result[0].Timestamp

				Expect(oldTimestamp).NotTo(Equal(newTimestamp))
				Expect(newTimestamp.After(oldTimestamp)).To(BeTrue())
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
			Expect(table.Lookup("foo.com")).To(Equal([]addresstable.IPAndTime{}))
		})
		Context("when two hostnames are registered to same ip address", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com.", "bar.com"}, "192.0.0.2")
			})
			It("removes both IPs", func() {
				table.Remove([]string{"foo.com", "bar.com."}, "192.0.0.2")

				Expect(table.Lookup("foo.com")).To(Equal([]addresstable.IPAndTime{}))
				Expect(table.Lookup("bar.com")).To(Equal([]addresstable.IPAndTime{}))
			})
		})

		Context("when removing an IP that does not exist", func() {
			BeforeEach(func() {
				table.Add([]string{"foo.com"}, "192.0.0.2")
			})
			It("does not panic", func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
				result := table.Lookup("foo.com")
				Expect(len(result)).To(Equal(1))
				Expect(result[0].IP).To(Equal("192.0.0.2"))
				Expect(result[0].Timestamp.String()).NotTo(Equal("0001-01-01 00:00:00 +0000 UTC"))
			})
		})

		Context("when removing a host that does not exist", func() {
			It("does not panic", func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
				Expect(table.Lookup("foo.com")).To(Equal([]addresstable.IPAndTime{}))
			})
		})
	})

	Describe("Lookup", func() {
		It("returns an empty array for an unknown hostname", func() {
			Expect(table.Lookup("foo.com")).To(Equal([]addresstable.IPAndTime{}))
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
			Eventually(func() []string {
				ipAndTimes := table.Lookup("foo.com")
				ips := []string{}
				for _, ipAndTime := range ipAndTimes {
					ips = append(ips, ipAndTime.IP)
				}
				return ips
			}).Should(ConsistOf([]string{
				"192.0.0.1",
				"192.0.0.2",
			}))

			go func() {
				table.Remove([]string{"foo.com"}, "192.0.0.2")
			}()
			go func() {
				table.Remove([]string{"foo.com"}, "192.0.0.1")
			}()
			Eventually(func() []string {
				ipAndTimes := table.Lookup("foo.com")
				ips := []string{}
				for _, ipAndTime := range ipAndTimes {
					ips = append(ips, ipAndTime.IP)
				}
				return ips
			}).Should(ConsistOf([]string{}))
		})
	})
})
