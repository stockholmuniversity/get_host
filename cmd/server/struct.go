package main

import (
	"sync"
	"time"

	"github.com/miekg/dns"
)

type cache struct {
	data map[string][]dns.RR
	soas []dns.SOA
	sync.RWMutex
	age time.Time
	// TODO hits int: Number of questions the server have got.
}

func (c cache) Age() time.Duration {
	c.RLock()
	t := time.Since(c.age)
	c.RUnlock()
	return t.Truncate(time.Second)
}

func (c *cache) Len() int {
	c.RLock()
	n := len(c.data)
	c.RUnlock()
	return n
}
