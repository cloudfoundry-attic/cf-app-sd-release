package addresstable

import "sync"

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
		ips := at.ipsForHostname(hostname)
		if indexOf(ips, ip) == -1 {
			ips = append(ips, ip)
			at.addresses[hostname] = ips
		}
	}
	at.mutex.Unlock()
}

func (at *AddressTable) Remove(hostnames []string, ip string) {
	at.mutex.Lock()
	for _, hostname := range hostnames {
		ips := at.ipsForHostname(hostname)
		index := indexOf(ips, ip)
		if index > -1 {
			at.addresses[hostname] = append(ips[:index], ips[index+1:]...)
		}
	}
	at.mutex.Unlock()
}

func (at *AddressTable) Lookup(hostname string) []string {
	at.mutex.RLock()
	ips := at.ipsForHostname(hostname)
	at.mutex.RUnlock()
	return ips
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
