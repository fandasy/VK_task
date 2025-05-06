package subpub_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"VK_task/pkg/subpub"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubPub(t *testing.T) {
	t.Run("Subscribe/Publish", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 10})

		var wg sync.WaitGroup
		wg.Add(1)

		sub, err := sp.Subscribe("test", func(msg interface{}) {
			defer wg.Done()
			assert.Equal(t, "hello", msg)
		})
		require.NoError(t, err)
		require.NotNil(t, sub)

		err = sp.Publish("test", "hello")
		require.NoError(t, err)

		wg.Wait()
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 10})

		called := false
		sub, err := sp.Subscribe("test", func(msg interface{}) {
			called = true
		})
		require.NoError(t, err)

		sub.Unsubscribe()
		err = sp.Publish("test", "hello")
		require.Error(t, err)

		time.Sleep(100 * time.Millisecond) // Wait potential delivery
		assert.False(t, called)
	})

	t.Run("Close", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 10})

		handlerStarted := make(chan struct{})

		// Setup slow subscriber
		_, err := sp.Subscribe("slow", func(msg interface{}) {
			close(handlerStarted)
			time.Sleep(500 * time.Millisecond)
		})
		require.NoError(t, err)

		err = sp.Publish("slow", "data")
		require.NoError(t, err)

		start := time.Now()

		// Wait handler goroutine to start
		<-handlerStarted

		// Test graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = sp.Close(ctx)
		assert.NoError(t, err)
		assert.True(t, time.Since(start) >= 500*time.Millisecond)
	})

	t.Run("Close with canceled context", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 10})

		handlerStarted := make(chan struct{})

		// Setup very slow subscriber
		_, err := sp.Subscribe("slow", func(msg interface{}) {
			close(handlerStarted)
			time.Sleep(2 * time.Second)
		})
		require.NoError(t, err)

		err = sp.Publish("slow", "data")
		require.NoError(t, err)

		// Wait handler goroutine to start
		<-handlerStarted

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = sp.Close(ctx)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded))
	})

	t.Run("Publish to non-existent subject", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 10})
		err := sp.Publish("nonexistent", "data")
		assert.Equal(t, subpub.ErrNoSuchSubject, err)
	})

	t.Run("Concurrent operations", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 100})
		var wg sync.WaitGroup

		// Start 10 subscribers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := sp.Subscribe("concurrent", func(msg interface{}) {})
				assert.NoError(t, err)
			}()
		}

		// Publish 100 messages
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				err := sp.Publish("concurrent", i)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()
		assert.NoError(t, sp.Close(context.Background()))
	})

	t.Run("Handler panic recovery", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 10})

		_, err := sp.Subscribe("panic", func(msg interface{}) {
			panic("test panic")
		})
		require.NoError(t, err)

		// Should not crash
		err = sp.Publish("panic", "data")
		require.NoError(t, err)

		// Give time for panic recovery
		time.Sleep(50 * time.Millisecond)
		assert.NoError(t, sp.Close(context.Background()))
	})

	t.Run("Double close", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 10})
		require.NoError(t, sp.Close(context.Background()))
		assert.Equal(t, subpub.ErrSubPubClosed, sp.Close(context.Background()))
	})

	t.Run("Subscribe after close", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 10})
		require.NoError(t, sp.Close(context.Background()))

		_, err := sp.Subscribe("test", func(msg interface{}) {})
		assert.Equal(t, subpub.ErrSubPubClosed, err)
	})

	t.Run("Publish after close", func(t *testing.T) {
		sp := subpub.NewSubPub(subpub.Config{SubjectPuffer: 10})
		require.NoError(t, sp.Close(context.Background()))

		err := sp.Publish("test", "data")
		assert.Equal(t, subpub.ErrSubPubClosed, err)
	})
}
