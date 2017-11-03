package performance_test

import (
	"fmt"
	"strings"

	"github.com/nats-io/nats"
	. "github.com/onsi/ginkgo"
)

var _ = PDescribe("NatsClientPerformance", func() {
	// var msgSize = 1024

	Measure(fmt.Sprintf("NATS CPU when publishing %s messages", config.NumMessages), func() {
		opts := nats.GetDefaultOptions()
		opts.Servers = strings.Split(config.NatsURL, ",")
		opts.User = config.NatsUsername
		opts.Password = config.NatsPassword
		for i, s := range opts.Servers {
			opts.Servers[i] = strings.Trim(s, " ")
		}

		// runRoutePopulator := func(nats, backendHost string, backendPort int, appDomain, appName string, numRoutes int) *gexec.Session {
		// 	routePopulatorCommand := exec.Command(httpRoutePopulatorPath,
		// 		"-nats", nats,
		// 		"-backendHost", backendHost,
		// 		"-backendPort", strconv.Itoa(backendPort),
		// 		"-appDomain", appDomain,
		// 		"-appName", appName,
		// 		"-numRoutes", strconv.Itoa(numRoutes),
		// 	)
		// 	session, err := gexec.Start(routePopulatorCommand, GinkgoWriter, GinkgoWriter)
		// 	Expect(err).ToNot(HaveOccurred())
		// 	return session
		// }

		// run the route populator with 1000 routes
		// check that the SDC received 1000 routes
		// hitting the /route endpoint and getting length

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
