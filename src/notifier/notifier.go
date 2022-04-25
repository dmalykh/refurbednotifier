package notifier

import (
	"bufio"
	"container/list"
	"context"
	"github.com/dmalykh/refurbedsender/sender"
	"io"
	"log"
	"reflect"
	"sync"
	"time"
)

type SendFunc func(ctx context.Context, message []byte) error

// NewNotifier returns configured notifier that sends messager via sender.Sender evert interval time with maxSize of messages stacked in queue
func NewNotifier(s sender.Sender, maxSize int, interval time.Duration) *Notifier {
	return &Notifier{
		sender:   s,
		interval: interval,
		maxSize:  maxSize,
	}
}

type Notifier struct {
	sender   sender.Sender
	interval time.Duration
	maxSize  int
	lock     sync.Mutex
}

func (n *Notifier) Run(ctx context.Context, input io.Reader, f SendFunc) {
	var send = make(chan []byte)
	go func() {
		var queue = new(list.List)
		defer queue.Init()
		n.schedule(ctx, send, queue, f)
	}()
	n.scan(ctx, input, send)
	close(send)
}

func (n *Notifier) scan(ctx context.Context, input io.Reader, send chan []byte) {
	var scanner = bufio.NewScanner(input)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case send <- scanner.Bytes():
			continue
		}
	}
	select {
	case <-ctx.Done():
		return
	}
}

// Schedule adds data from send channel to queue and send all from queue with time interval via f
func (n *Notifier) schedule(ctx context.Context, send chan []byte, queue *list.List, f SendFunc) {
	var ticker = time.NewTicker(n.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			queue.Init()

			return
		case m := <-send:
			n.lock.Lock()
			queue.PushBack(m)
			n.lock.Unlock()
		case <-ticker.C:
			n.release(ctx, queue, f)
		}
	}
}

// Send messages from queue
func (n *Notifier) release(ctx context.Context, queue *list.List, f SendFunc) {
	for {
		select {
		case <-ctx.Done():
			queue.Init()
			return
		default:
			func() {
				element := queue.Front()
				if element == nil {
					return
				}
				n.lock.Lock()
				defer func() {
					queue.Remove(element)
					n.lock.Unlock()
				}()
				msg, ok := element.Value.([]byte)
				if !ok {
					log.Fatalf(`Unknown element value in queue: %s`, reflect.TypeOf(element.Value))
				}
				_ = f(ctx, msg) // Work with error in your custom func
			}()
		}
	}
}

// Send message t
func (n *Notifier) Send(ctx context.Context, message []byte) error {
	return n.sender.Send(ctx, sender.NewMessage(message))
}
