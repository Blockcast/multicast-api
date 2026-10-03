package api

import (
	sync2 "sync"
	"testing"
)

// Size reads sendCases under mu, so every write to sendCases must hold mu too.
// Two writers did not: init, which the first Subscribe runs, and Send when it
// takes an unsubscribed channel off removeSub. Run under -race; the reader
// polls throughout both writes.
func TestFeedOfSizeDoesNotRaceInitOrSendRemoval(t *testing.T) {
	var feed FeedOf[int]
	stop := make(chan struct{})
	polling := make(chan struct{})
	var poller sync2.WaitGroup
	poller.Add(1)
	go func() {
		defer poller.Done()
		_ = feed.Size()
		close(polling)
		for {
			select {
			case <-stop:
				return
			default:
				_ = feed.Size()
			}
		}
	}()
	<-polling

	// The first Subscribe runs init, which seeds sendCases.
	blocked := make(chan int) // never read: Send stays blocked on it
	sub := feed.Subscribe(blocked)
	delivered := make(chan int)
	feed.Subscribe(delivered)

	sent := make(chan int, 1)
	go func() { sent <- feed.Send(1) }()
	// Receiving on delivered proves Send holds sendLock; it is still blocked
	// on the other channel, so Unsubscribe must hand that channel to Send over
	// removeSub, and Send deletes it from sendCases.
	<-delivered
	sub.Unsubscribe()
	if n := <-sent; n != 1 {
		t.Fatalf("Send delivered to %d subscribers, want 1", n)
	}

	close(stop)
	poller.Wait()
}

// Size counts the inbox plus sendCases, and sendCases holds FeedOf's internal
// removeSub case once the feed is initialized. Callers rely on these values
// (multicast's receiver accepts every object while recvAll.Size() != 0, and its
// test helper subtracts the internal case), so a locking fix must keep them.
func TestFeedOfSizeCountsTheInternalCaseOnceInitialized(t *testing.T) {
	var feed FeedOf[int]
	if got := feed.Size(); got != 0 {
		t.Fatalf("never-initialized feed: Size() = %d, want 0", got)
	}
	ch := make(chan int, 1)
	sub := feed.Subscribe(ch)
	if got := feed.Size(); got != 2 {
		t.Fatalf("one subscriber in the inbox: Size() = %d, want 2", got)
	}
	if n := feed.Send(7); n != 1 {
		t.Fatalf("Send delivered to %d subscribers, want 1", n)
	}
	if got := feed.Size(); got != 2 {
		t.Fatalf("one subscriber in sendCases: Size() = %d, want 2", got)
	}
	sub.Unsubscribe()
	if got := feed.Size(); got != 1 {
		t.Fatalf("no subscribers after init: Size() = %d, want 1", got)
	}
}
