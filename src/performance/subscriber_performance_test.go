package performance_test

import (
	. "github.com/onsi/ginkgo"
	"fmt"
	"github.com/nats-io/nats/bench"
	"sync"
	"log"
	"flag"
	"time"
	"github.com/nats-io/nats"
	"strings"
)

var _ = Describe("SubscriberPerformance", func() {
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
		natsBenchmark := bench.NewBenchmark("foo", 1, config.NumPublisher)

		// Now Publishers
		startwg.Add(config.NumPublisher)
		pubCounts := bench.MsgsPerClient(config.NumMessages, config.NumPublisher)
		for i := 0; i < config.NumPublisher; i++ {
			go runPublisher(natsBenchmark, &startwg, opts, pubCounts[i], msgSize)
		}

		startwg.Wait()
		natsBenchmark.Close()
		fmt.Fprintln(GinkgoWriter, natsBenchmark.Report())

		Fail("hello")
	}, 1)

})

func runPublisher(benchmark *bench.Benchmark, startwg *sync.WaitGroup, opts nats.Options, numMsgs int, msgSize int) {
	nc, err := opts.Connect()
	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}
	defer nc.Close()
	startwg.Done()

	args := flag.Args()
	subj := args[0]
	var msg []byte
	if msgSize > 0 {
		msg = make([]byte, msgSize)
	}

	start := time.Now()

	for i := 0; i < numMsgs; i++ {
		nc.Publish(subj, msg)
	}
	nc.Flush()
	benchmark.AddPubSample(bench.NewSample(numMsgs, msgSize, start, time.Now(), nc))
}
