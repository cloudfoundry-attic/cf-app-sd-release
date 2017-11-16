package addresstable

import (
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
)

type AddressTable struct {
	addresses          map[string][]entry
	clock              clock.Clock
	stalenessThreshold time.Duration
	mutex              sync.RWMutex
}

type entry struct {
	ip         string
	updateTime time.Time
}

func NewAddressTable(stalenessThreshold, pruningInterval time.Duration, clock clock.Clock) *AddressTable {
	table := &AddressTable{
		addresses:          map[string][]entry{},
		clock:              clock,
		stalenessThreshold: stalenessThreshold,
	}
	table.pruneStaleEntriesOnInterval(pruningInterval)
	return table
}

func (at *AddressTable) Add(hostnames []string, ip string) {
	at.mutex.Lock()
	for _, hostname := range hostnames {
		fqHostname := fqdn(hostname)
		entries := at.entriesForHostname(fqHostname)
		entryIndex := indexOf(entries, ip)
		if entryIndex == -1 {
			at.addresses[fqHostname] = append(entries, entry{ip: ip, updateTime: at.clock.Now()})
		} else {
			at.addresses[fqHostname][entryIndex].updateTime = at.clock.Now()
		}
	}
	at.mutex.Unlock()
}

func (at *AddressTable) Remove(hostnames []string, ip string) {
	at.mutex.Lock()
	for _, hostname := range hostnames {
		fqHostname := fqdn(hostname)
		entries := at.entriesForHostname(fqHostname)
		index := indexOf(entries, ip)
		if index > -1 {
			if len(entries) == 1 {
				delete(at.addresses, fqHostname)
			} else {
				at.addresses[fqHostname] = append(entries[:index], entries[index+1:]...)
			}
		}
	}
	at.mutex.Unlock()
}

func (at *AddressTable) Lookup(hostname string) []string {
	at.mutex.RLock()

	found := at.entriesForHostname(fqdn(hostname))

	at.mutex.RUnlock()

	return entriesToIPs(found)
}

func (at *AddressTable) GetAllAddresses() map[string][]string {
	at.mutex.RLock()

	addresses := map[string][]string{}
	for address, entries := range at.addresses {
		addresses[address] = entriesToIPs(entries)
	}

	at.mutex.RUnlock()

	return addresses
}

func (at *AddressTable) entriesForHostname(hostname string) []entry {
	if existing, ok := at.addresses[hostname]; ok {
		return existing
	} else {
		return []entry{}
	}
}

func entriesToIPs(entries []entry) []string {
	ips := make([]string, len(entries))
	for idx, entry := range entries {
		ips[idx] = entry.ip
	}

	return ips
}

func (at *AddressTable) pruneStaleEntriesOnInterval(pruningInterval time.Duration) {
	ticker := at.clock.NewTicker(pruningInterval)
	go func() {
		defer ticker.Stop()
		for _ = range ticker.C() {
			staleAddresses := at.addressesWithStaleEntriesWithReadLock()
			at.pruneStaleEntriesWithWriteLock(staleAddresses)
		}
	}()
}

func (at *AddressTable) pruneStaleEntriesWithWriteLock(candidateAddresses []string) {
	if len(candidateAddresses) == 0 {
		return
	}

	at.mutex.Lock()
	for _, staleAddr := range candidateAddresses {
		entries, ok := at.addresses[staleAddr]
		if ok {
			freshEntries := []entry{}
			for _, entry := range entries {
				if at.clock.Since(entry.updateTime) <= at.stalenessThreshold {
					freshEntries = append(freshEntries, entry)
				}
			}
			at.addresses[staleAddr] = freshEntries
		}
	}
	at.mutex.Unlock()
}

func (at *AddressTable) addressesWithStaleEntriesWithReadLock() []string {
	staleAddresses := []string{}
	at.mutex.RLock()
	for address, entries := range at.addresses {
		for _, entry := range entries {
			if at.clock.Since(entry.updateTime) > at.stalenessThreshold {
				staleAddresses = append(staleAddresses, address)
				break
			}
		}
	}
	at.mutex.RUnlock()
	return staleAddresses
}

func indexOf(entries []entry, value string) int {
	for idx, entry := range entries {
		if entry.ip == value {
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
