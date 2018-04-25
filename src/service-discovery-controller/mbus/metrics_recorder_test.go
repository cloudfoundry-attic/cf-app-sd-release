package mbus_test

import (
	"service-discovery-controller/mbus"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsRecorder", func() {
	var (
		recorder *mbus.MetricsRecorder
	)

	BeforeEach(func() {
		currentSystemTime := time.Unix(0, 150)
		fakeClock := fakeclock.NewFakeClock(currentSystemTime)
		recorder = &mbus.MetricsRecorder{
			Clock: fakeClock,
		}
	})

	It("should return the highest value since the last time it was asked", func() {
		recorder.RecordMessageTransitTime(120)
		recorder.RecordMessageTransitTime(130)
		recorder.RecordMessageTransitTime(125)

		time, err := recorder.GetMaxSinceLastInterval()
		Expect(err).NotTo(HaveOccurred())
		Expect(time).To(Equal(float64(30)))
	})

	It("should not record zero unix times", func() {
		recorder.RecordMessageTransitTime(0)

		time, err := recorder.GetMaxSinceLastInterval()
		Expect(err).NotTo(HaveOccurred())
		Expect(time).To(Equal(float64(0)))
	})
})
