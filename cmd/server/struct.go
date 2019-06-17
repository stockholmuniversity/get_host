package main

import (
	"sync"
	"time"

	"github.com/miekg/dns"
)

// cache is the structure for the dns cache, mutex and meta information regarding the cache.
type cache struct {
	data         map[string][]dns.RR // data is the dns cache
	soas         []dns.SOA           // soas is domains/subdomains the cache will include
	sync.RWMutex                     // RWMutex is read/write lock
	age          time.Time           // age is the age of the cache.
	startTime    time.Time           /// startTime is the time the server started
	// TODO hits int: Number of questions the server have got.
}

// Age returns the age of the cache. It should never get older than TTL from the config.
func (c cache) Age() time.Duration {
	c.RLock()
	t := time.Since(c.age)
	c.RUnlock()
	return t.Truncate(time.Second)
}

// Len returns the lenth (size) of the cache. How many dns records it holds.
func (c *cache) Len() int {
	c.RLock()
	n := len(c.data)
	c.RUnlock()
	return n
}

// uptime return uptime since start.
func (c cache) uptime() time.Duration {
	t := time.Since(c.startTime)
	return t.Truncate(time.Second)
}
