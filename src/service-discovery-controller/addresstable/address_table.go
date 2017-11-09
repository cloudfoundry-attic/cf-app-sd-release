package addresstable

import (
	"sync"
	"time"
)

type IPAndTime struct {
	IP        string
	Timestamp time.Time
}

type AddressTable struct {
	addresses map[string][]IPAndTime
	mutex     sync.RWMutex
}

func NewAddressTable() *AddressTable {
	return &AddressTable{
		addresses: map[string][]IPAndTime{},
	}
}

func (at *AddressTable) Add(hostnames []string, ip string) {
	at.mutex.Lock()
	for _, hostname := range hostnames {
		fqHostname := fqdn(hostname)
		ips := at.ipsForHostname(fqHostname)
		index := indexOf(ips, ip)
		if index == -1 {
			ips = append(ips, IPAndTime{
				IP:        ip,
				Timestamp: time.Now(),
			})
		} else {
			ips[index].Timestamp = time.Now()
		}
		at.addresses[fqHostname] = ips
	}
	at.mutex.Unlock()
}

func (at *AddressTable) Remove(hostnames []string, ip string) {
	at.mutex.Lock()
	for _, hostname := range hostnames {
		fqHostname := fqdn(hostname)
		ips := at.ipsForHostname(fqHostname)
		index := indexOf(ips, ip)
		if index > -1 {
			at.addresses[fqHostname] = append(ips[:index], ips[index+1:]...)
		}
	}
	at.mutex.Unlock()
}

func (at *AddressTable) Lookup(hostname string) []IPAndTime {
	at.mutex.RLock()

	found := at.ipsForHostname(fqdn(hostname))
	foundCopy := make([]IPAndTime, len(found))
	copy(foundCopy, found)

	at.mutex.RUnlock()

	return foundCopy
}

func (at *AddressTable) GetAllAddresses() map[string][]string {
	at.mutex.RLock()

	addresses := at.addresses

	returnAddresses := map[string][]string{}
	for hostname, ipAndTimes := range addresses {
		ips := []string{}
		for _, ipAndTime := range ipAndTimes {
			ips = append(ips, ipAndTime.IP)
		}
		returnAddresses[hostname] = ips
	}

	at.mutex.RUnlock()

	return returnAddresses
}

func (at *AddressTable) ipsForHostname(hostname string) []IPAndTime {
	if existing, ok := at.addresses[hostname]; ok {
		return existing
	} else {
		return []IPAndTime{}
	}
}

func indexOf(ipAndTimes []IPAndTime, value string) int {
	for idx, ipAndTime := range ipAndTimes {
		if ipAndTime.IP == value {
			return idx
		}
	}
	return -1
}

func isFqdn(s string) bool {
	l := len(s)
	if l == 0 {
		return false
	}
	return s[l-1] == '.'
}

func fqdn(s string) string {
	if isFqdn(s) {
		return s
	}
	return s + "."
}
