package performance_test

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-top/util"
	"github.com/nats-io/nats/bench"
	"github.com/nats-io/gnatsd/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NatsPerformance", func() {

	Measure(fmt.Sprintf("NATS subscriptions when publishing %d messages", config.NumMessages), func(b Benchmarker) {
		By("building a benchmark of subscribers listening on service-discovery.register")
		benchMarkNatsSubMap := collectNatsTop("service-discovery.register")


		By("publish messages onto service-discovery.register")
		natsBenchmark := runNatsBencmarker()
		generateBenchmarkGinkgoReport(b, natsBenchmark)

		By("building an updated benchmark of subscribers listening on service-discovery.register")
		natsSubMap := collectNatsTop("service-discovery.register")

		By("Making sure service-discovery.register subscribers received every published message", func() {
			for key, benchMarkVal := range benchMarkNatsSubMap {
				Expect(int(natsSubMap[key].OutMsgs - benchMarkVal.OutMsgs)).To(Equal(config.NumMessages), fmt.Sprintf("Benchmark: %+v \\n \\n Got %+v", benchMarkVal, natsSubMap[key]))
			}
		})

	}, 1)

})

func runNatsBencmarker() *bench.Benchmark {
	opts := nats.GetDefaultOptions()
	opts.Servers = strings.Split(config.NatsURL, ",")
	opts.User = config.NatsUsername
	opts.Password = config.NatsPassword
	for i, s := range opts.Servers {
		opts.Servers[i] = "nats://" + strings.Trim(s, " ") + ":" + strconv.Itoa(config.NatsPort)
	}

	var startwg sync.WaitGroup
	natsBenchmark := bench.NewBenchmark("SDC Nats Benchmark", 0, config.NumPublisher)
	startwg.Add(config.NumPublisher)
	pubCounts := bench.MsgsPerClient(config.NumMessages, config.NumPublisher)
	for _, pubCount := range pubCounts {
		go runPublisher("service-discovery.register", natsBenchmark, &startwg, opts, pubCount, 1024)
	}
	startwg.Wait()
	natsBenchmark.Close()
	return natsBenchmark
}

func generateBenchmarkGinkgoReport(b Benchmarker, bm *bench.Benchmark) {
	if bm.Pubs.HasSamples() {
		if len(bm.Pubs.Samples) > 1 {
			b.RecordValue("PubStats", float64(bm.Pubs.Rate()), "msgs/sec")
			for i, stat := range bm.Pubs.Samples {
				b.RecordValue("Pub", float64(stat.MsgCnt), fmt.Sprintf("subscriber # %d", i))
			}
			b.RecordValue("min", float64(bm.Pubs.MinRate()))
			b.RecordValue("avg", float64(bm.Pubs.AvgRate()))
			b.RecordValue("max", float64(bm.Pubs.MaxRate()))
			b.RecordValue("stddev", float64(bm.Pubs.StdDev()))
		}
	}

}

func collectNatsTop(subscriber string) map[uint64]server.ConnInfo {
	serviceDiscoverySubs := map[uint64]server.ConnInfo{}

	timeoutChan := time.After(10 * time.Second)
	natsTopEngine := toputils.NewEngine("localhost", 8822, 1000, 1)
	natsTopEngine.DisplaySubs = true
	natsTopEngine.SetupHTTP()

	go func() {
		defer GinkgoRecover()
		Expect(natsTopEngine.MonitorStats()).To(Succeed())
	}()

	for {
		select {
		case stats := <-natsTopEngine.StatsCh:
			for _, statsConn := range stats.Connz.Conns {
				if strings.Contains(strings.Join(statsConn.Subs, ","), subscriber) {
					serviceDiscoverySubs[statsConn.Cid] = server.ConnInfo(statsConn)
				}
			}
		case <-timeoutChan:
			close(natsTopEngine.ShutdownCh)
			return serviceDiscoverySubs
		}
	}
}

func runPublisher(subject string, benchmark *bench.Benchmark, startwg *sync.WaitGroup, opts nats.Options, numMsgs int, msgSize int) {
	defer GinkgoRecover()
	defer startwg.Done()
	nc, err := opts.Connect()

	Expect(err).NotTo(HaveOccurred())
	defer nc.Close()

	var msg []byte
	if msgSize > 0 {
		msg = make([]byte, msgSize)
	}

	start := time.Now()

	for i := 0; i < numMsgs; i++ {
		nc.Publish(subject, msg)
	}
	nc.Flush()
	benchmark.AddPubSample(bench.NewSample(numMsgs, msgSize, start, time.Now(), nc))
}
