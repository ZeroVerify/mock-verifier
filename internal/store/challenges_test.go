package store

import (
	"testing"
	"time"
)

func TestChallengeIsSingleUse(t *testing.T) {
	c := NewChallenges(time.Minute)
	c.Issue("123")
	if !c.Consume("123") {
		t.Fatal("an issued challenge must be accepted once")
	}
	if c.Consume("123") {
		t.Fatal("a challenge must not be accepted twice")
	}
}

func TestUnknownChallengeRejected(t *testing.T) {
	c := NewChallenges(time.Minute)
	if c.Consume("never-issued") {
		t.Fatal("a challenge this verifier never issued must be rejected")
	}
}

func TestExpiredChallengeRejected(t *testing.T) {
	c := NewChallenges(10 * time.Millisecond)
	c.Issue("old")
	time.Sleep(30 * time.Millisecond)
	if c.Consume("old") {
		t.Fatal("an expired challenge must be rejected")
	}
}
