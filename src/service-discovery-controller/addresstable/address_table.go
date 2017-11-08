package addresstable

import (
	"sync"
)

type AddressTable struct {
	addresses map[string][]string
	mutex     sync.RWMutex
}

func NewAddressTable() *AddressTable {
	return &AddressTable{
		addresses: map[string][]string{},
	}
}

func (at *AddressTable) Add(hostnames []string, ip string) {
	at.mutex.Lock()
	for _, hostname := range hostnames {
		fqHostname := fqdn(hostname)
		ips := at.ipsForHostname(fqHostname)
		if indexOf(ips, ip) == -1 {
			ips = append(ips, ip)
			at.addresses[fqHostname] = ips
		}
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

func (at *AddressTable) Lookup(hostname string) []string {
	at.mutex.RLock()

	found := at.ipsForHostname(fqdn(hostname))
	foundCopy := make([]string, len(found))
	copy(foundCopy, found)

	at.mutex.RUnlock()

	return foundCopy
}

func (at *AddressTable) GetAllAddresses() map[string][]string {
	at.mutex.RLock()

	addresses := at.addresses

	at.mutex.RUnlock()

	return addresses
}

func (at *AddressTable) ipsForHostname(hostname string) []string {
	if existing, ok := at.addresses[hostname]; ok {
		return existing
	} else {
		return []string{}
	}
}

func indexOf(strings []string, value string) int {
	for idx, str := range strings {
		if str == value {
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
