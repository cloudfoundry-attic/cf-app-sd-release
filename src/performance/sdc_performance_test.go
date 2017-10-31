package performance_test

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nats-io/nats"
	"github.com/nats-io/nats/bench"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("NatsClientPerformance", func() {
	var msgSize = 1024

	Measure(fmt.Sprintf("NATS CPU when publishing %s messages", config.NumMessages), func() {
		opts := nats.GetDefaultOptions()
		opts.Servers = strings.Split(config.NatsURL, ",")
		opts.User = config.NatsUsername
		opts.Password = config.NatsPassword
		for i, s := range opts.Servers {
			opts.Servers[i] = strings.Trim(s, " ")
		}

		var startwg sync.WaitGroup
		natsBenchmark := bench.NewBenchmark("foo", 0, config.NumPublisher)

		// // Now Publishers
		// startwg.Add(config.NumPublisher)
		// pubCounts := bench.MsgsPerClient(config.NumMessages, config.NumPublisher)
		// for i := 0; i < config.NumPublisher; i++ {
		// 	go runPublisher(natsBenchmark, &startwg, opts, pubCounts[i], msgSize)
		// }
		//
		// startwg.Wait()
		// natsBenchmark.Close()
		// fmt.Fprintln(GinkgoWriter, natsBenchmark.Report())
		//
		// Fail("hello")
	}, 1)

})
