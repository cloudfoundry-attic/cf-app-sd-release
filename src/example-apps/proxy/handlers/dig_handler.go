package handlers

import (
	"net"
	"net/http"
	"strings"
	"encoding/json"
)

type DigHandler struct {
}

func (h *DigHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/dig/")
	destination = strings.Split(destination, ":")[0]

	ips, err := net.LookupIP(destination)
	if err != nil {
		handleError(err, destination, resp)
	}

	var ip4s []string

	for _, ip := range ips {
		ip4s = append(ip4s, ip.To4().String())
	}

	ip4Json, err := json.Marshal(ip4s)
	if err != nil {
		handleError(err, destination, resp)
	}

	resp.Write(ip4Json)
}
