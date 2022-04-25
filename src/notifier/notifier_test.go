package notifier

import (
	"container/list"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNotifier_Scan(t *testing.T) {
	var n = Notifier{}
	var check = "qwertyasdfghzxcvbn"

	var send = make(chan []byte)
	var got int32

	var wg = new(sync.WaitGroup)
	ctx, cancel := context.WithCancel(context.TODO())
	var r = strings.NewReader(strings.Join(strings.Split(check, ``), "\n"))
	go n.scan(ctx, r, send)
	wg.Add(len(check))
	go func() {
		for _ = range send {
			atomic.AddInt32(&got, 1)
			wg.Done()
		}
	}()
	wg.Wait()
	cancel()
	assert.Equal(t, len(check), int(got))
}

func TestNotifier_ScanContextDone(t *testing.T) {
	var n = Notifier{}
	var check = "qwertyasdfghzxcvbn"
	var want = 8
	ctx, cancel := context.WithCancel(context.TODO())

	var send = make(chan []byte)
	var got int32

	var wg = new(sync.WaitGroup)
	go func() {
		for _ = range send {
			wg.Add(1)
			atomic.AddInt32(&got, 1)
			if atomic.LoadInt32(&got) == int32(want) {
				cancel()
			}
			wg.Done()
		}
	}()

	var r = strings.NewReader(strings.Join(strings.Split(check, ``), "\n"))
	n.scan(ctx, r, send)
	wg.Wait()
	close(send)
	assert.Equal(t, want, int(atomic.LoadInt32(&got)))
}

func TestNotifier_ScheduleAddedToQueue(t *testing.T) {
	var n = Notifier{
		interval: 10 * time.Millisecond,
		maxSize:  2,
	}

	var queue = new(list.List)
	var send = make(chan []byte)
	var ctx = context.TODO()
	var called int32
	var wg sync.WaitGroup
	go n.schedule(ctx, send, queue, func(ctx context.Context, message []byte) error {
		atomic.AddInt32(&called, 1)
		wg.Done()
		return nil
	})

	for i := 0; i < 5; i++ {
		wg.Add(1)
		send <- []byte(fmt.Sprintf(`Iter %d`, i))
	}
	wg.Wait()
	assert.Equal(t, int32(5), atomic.LoadInt32(&called))
}
