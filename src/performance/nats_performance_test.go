package performance_test

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	nats "github.com/nats-io/go-nats"
	"github.com/nats-io/nats/bench"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("NatsPerformance", func() {
	var msgSize = 1024

	Measure(fmt.Sprintf("NATS subscriptions when publishing %d messages", config.NumMessages), func(b Benchmarker) {
		fmt.Printf("%+v", config)
		opts := nats.GetDefaultOptions()
		opts.Servers = strings.Split(config.NatsURL, ",")
		opts.User = config.NatsUsername
		opts.Password = config.NatsPassword
		for i, s := range opts.Servers {
			opts.Servers[i] = "nats://" + strings.Trim(s, " ") + ":" + strconv.Itoa(config.NatsPort)
		}

		fmt.Printf("servers are " + opts.Servers[0])
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

		// get subscription info
		cmd := exec.Command("curl", fmt.Sprintf("http://%s:%d/subscriptionsz", config.NatsURL, config.NatsMonitoringPort))

		out, err := cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred())
		fmt.Println(string(out))

		cmd = exec.Command("curl", fmt.Sprintf("http://%s:%d/connz", config.NatsURL, config.NatsMonitoringPort))

		out, err = cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred())
		fmt.Println(string(out))
		// resp, err := http.Get(fmt.Sprintf("%s:%d/subscriptionsz", config.NatsURL, config.NatsMonitoringPort))
		// Expect(err).ToNot(HaveOccurred())
		// respBody, err := ioutil.ReadAll(resp.Body)
		// Expect(err).ToNot(HaveOccurred())
		// fmt.Fprintln(GinkgoWriter, respBody)

		Fail("hello")
	}, 1)

})

func runPublisher(benchmark *bench.Benchmark, startwg *sync.WaitGroup, opts nats.Options, numMsgs int, msgSize int) {
	nc, err := opts.Connect()
	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}
	defer nc.Close()

	// args := flag.Args()
	// subj := args[0]
	subj := "service-discovery.register"
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
	startwg.Done()
}
