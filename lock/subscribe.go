package lock

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrSubscriptionFull     = errors.New("Subscription full")
	ErrSubscriptionNotFound = errors.New("Subscription not found")
)

type Subscription[T any] struct {
	channel chan T
	parent  *Subscribable[T]
}

type Subscribable[T any] struct {
	lock        *Lock
	subscribers map[*Subscription[T]]struct{}
}

func NewSubscribable[T any]() *Subscribable[T] {
	return &Subscribable[T]{
		lock:        NewLock(),
		subscribers: make(map[*Subscription[T]]struct{}),
	}
}

func (s *Subscribable[T]) Publish(ctx context.Context, v T) error {
	if err := s.lock.Lock(ctx); err != nil {
		return err
	}
	defer s.lock.Unlock()

	var errs []string
	for sub := range s.subscribers {
		if err := sub.send(ctx, v); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while publishing: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (s *Subscribable[T]) Subscribe(ctx context.Context) (*Subscription[T], error) {
	if err := s.lock.Lock(ctx); err != nil {
		return nil, err
	}
	defer s.lock.Unlock()

	sub := &Subscription[T]{
		channel: make(chan T, 2),
		parent:  s,
	}
	s.subscribers[sub] = struct{}{}
	return sub, nil
}

func (s *Subscribable[T]) Unsubscribe(ctx context.Context, sub *Subscription[T]) error {
	if err := s.lock.Lock(ctx); err != nil {
		return err
	}
	defer s.lock.Unlock()

	if _, ok := s.subscribers[sub]; !ok {
		return ErrSubscriptionNotFound
	}
	delete(s.subscribers, sub)
	close(sub.channel)
	return nil
}

func (s *Subscription[T]) Unsubscribe(ctx context.Context) error {
	return s.parent.Unsubscribe(ctx, s)
}

func (s *Subscription[T]) send(ctx context.Context, v T) error {
	select {
	case s.channel <- v:
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrSubscriptionFull
	}
	return nil
}
