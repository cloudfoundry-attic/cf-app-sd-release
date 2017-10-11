package mbus_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	gnatsd "github.com/nats-io/gnatsd/test"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/nats"
)

func TestMbus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mbus Suite")
}


// RunDefaultServer will run a server on the default port.
func RunDefaultServer() *server.Server {
	return RunServerOnPort(nats.DefaultPort)
}

// RunServerOnPort will run a server on the given port.
func RunServerOnPort(port int) *server.Server {
	opts := gnatsd.DefaultTestOptions
	opts.Port = port
	return RunServerWithOptions(opts)
}

// RunServerWithOptions will run a server with the given options.
func RunServerWithOptions(opts server.Options) *server.Server {
	return gnatsd.RunServer(&opts)
}

// RunServerWithConfig will run a server with the given configuration file.
func RunServerWithConfig(configFile string) (*server.Server, *server.Options) {
	return gnatsd.RunServerWithConfig(configFile)
}