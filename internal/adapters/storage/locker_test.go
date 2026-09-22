package storage

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileLockerSerializes(t *testing.T) {
	locker := NewFileLocker(filepath.Join(t.TempDir(), ".lock"))
	var counter int64
	var concurrent int64
	done := make(chan struct{})

	for i := 0; i < 4; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			err := locker.WithWriteLock(context.Background(), func() error {
				if atomic.AddInt64(&concurrent, 1) != 1 {
					t.Error("lock is not exclusive")
				}
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt64(&counter, 1)
				atomic.AddInt64(&concurrent, -1)
				return nil
			})
			if err != nil {
				t.Errorf("WithWriteLock: %v", err)
			}
		}()
	}
	for i := 0; i < 4; i++ {
		<-done
	}
	if counter != 4 {
		t.Fatalf("counter = %d, want 4", counter)
	}
}

func TestFileLockerTimesOut(t *testing.T) {
	locker := NewFileLocker(filepath.Join(t.TempDir(), ".lock"))
	hold := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = locker.WithWriteLock(context.Background(), func() error {
			close(hold)
			<-release
			return nil
		})
	}()
	<-hold
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := locker.WithWriteLock(ctx, func() error { return nil })
	close(release)
	if err == nil {
		t.Fatal("expected timeout error while lock is held")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
}
