package mbus_test

import (
	"service-discovery-controller/mbus"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsRecorder", func() {
	It("should return the highest value since the last time it was asked", func() {
		currentSystemTime := time.Unix(0, 150)
		fakeClock := fakeclock.NewFakeClock(currentSystemTime)
		recorder := &mbus.MetricsRecorder{
			Clock: fakeClock,
		}

		recorder.RecordMessageTransitTime(120)
		recorder.RecordMessageTransitTime(130)
		recorder.RecordMessageTransitTime(125)

		time, err := recorder.GetMaxSinceLastInterval()
		Expect(err).NotTo(HaveOccurred())
		Expect(time).To(Equal(float64(30)))
	})
})
