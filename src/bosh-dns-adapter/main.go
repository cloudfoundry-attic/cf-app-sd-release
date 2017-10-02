package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, os.Interrupt)

	l, err := net.Listen("tcp", "127.0.0.1:8053")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Address (127.0.0.1:8053) not available")
		os.Exit(1)
	}

	go func() {
		http.Serve(l, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			dnsType := req.URL.Query().Get("type")
			name := req.URL.Query().Get("name")

			if dnsType == "" {
				writeBadRequestResponse(resp, name, "0")
				return
			}
			if name == "" {
				writeBadRequestResponse(resp, name, dnsType)
				return
			}

			writeSuccessfulResponse(resp, name, dnsType)
		}))

	}()

	fmt.Println("Server Started")
	select {
	case <-signalChannel:
		fmt.Println("Shutting bosh-dns-adapter down")
		return
	}
}
func writeBadRequestResponse(resp http.ResponseWriter, name string, dnsType string) {
	resp.WriteHeader(http.StatusBadRequest)
	writeResponse(resp, 2, name, dnsType)
}

func writeSuccessfulResponse(resp http.ResponseWriter, name string, dnsType string) {
	resp.WriteHeader(http.StatusOK)
	writeResponse(resp, 0, name, dnsType)
}

func writeResponse(resp http.ResponseWriter, status int, name string, dnsType string) {
	_, err := resp.Write([]byte(fmt.Sprintf(`{
											"Status": %d,
											"TC": false,
											"RD": false,
											"RA": false,
											"AD": false,
											"CD": false,
											"Question":
											[
												{
													"name": "%s",
													"type": %s
												}
											],
											"Answer":
											[
												{
													"name": "app-id.internal.local.",
													"type": 1,
													"TTL":  0,
													"data": "192.168.0.1"
												}
											],
											"Additional": [ ],
											"edns_client_subnet": "0.0.0.0/0"}`, status, name, dnsType)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing to http response body") // not tested
	}
}
