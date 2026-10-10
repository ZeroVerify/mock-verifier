package store

import (
	"sync"
	"time"
)

// Challenges remembers the nonces this verifier handed out. A proof is only accepted for a nonce that was issued
// here, once, and recently; otherwise a proof made for another verifier could be replayed.
type Challenges struct {
	mu     sync.Mutex
	issued map[string]time.Time
	ttl    time.Duration
}

func NewChallenges(ttl time.Duration) *Challenges {
	return &Challenges{issued: make(map[string]time.Time), ttl: ttl}
}

func (c *Challenges) Issue(challenge string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, t := range c.issued {
		if now.Sub(t) > c.ttl {
			delete(c.issued, k)
		}
	}
	c.issued[challenge] = now
}

// Consume reports whether the challenge was issued and still fresh, and forgets it so it cannot be used twice.
func (c *Challenges) Consume(challenge string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.issued[challenge]
	if !ok {
		return false
	}
	delete(c.issued, challenge)
	return time.Since(t) <= c.ttl
}
