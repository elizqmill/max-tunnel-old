package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type streamCredentialsCache struct {
	user   string
	pass   string
	urls   []string
	expiry time.Time
}

var credentialsStore = struct {
	mu     sync.RWMutex
	caches map[int]*streamCredentialsCache
}{caches: make(map[int]*streamCredentialsCache)}

var globalLastFetchTime time.Time
var globalFetchMu sync.Mutex
var globalCaptchaLockout atomic.Int64

func getCacheID(streamID int) int {
	return streamID / 9
}

func getStreamCache(streamID int) *streamCredentialsCache {
	credentialsStore.mu.RLock()
	cache, exists := credentialsStore.caches[getCacheID(streamID)]
	credentialsStore.mu.RUnlock()
	if exists && time.Now().Before(cache.expiry) {
		return cache
	}
	return nil
}

func setStreamCache(streamID int, user, pass string, urls []string) {
	credentialsStore.mu.Lock()
	defer credentialsStore.mu.Unlock()
	cacheID := getCacheID(streamID)
	credentialsStore.caches[cacheID] = &streamCredentialsCache{
		user:   user,
		pass:   pass,
		urls:   urls,
		expiry: time.Now().Add(25 * time.Minute),
	}
}

func cloneStringSlice(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func clearCachedCreds(streamID int) {
	credentialsStore.mu.Lock()
	defer credentialsStore.mu.Unlock()
	delete(credentialsStore.caches, getCacheID(streamID))
}

func GetCreds(ctx context.Context, link string, streamID int) (string, string, []string, error) {
	if cache := getStreamCache(streamID); cache != nil {
		log.Printf("[STREAM %d] [CREDS] Using cached credentials", streamID)
		return cache.user, cache.pass, cloneStringSlice(cache.urls), nil
	}

	globalFetchMu.Lock()
	minInterval := 2*time.Second + time.Duration(rand.Intn(2000))*time.Millisecond
	elapsed := time.Since(globalLastFetchTime)
	if !globalLastFetchTime.IsZero() && elapsed < minInterval {
		wait := minInterval - elapsed
		log.Printf("[STREAM %d] [CREDS] Throttling: waiting %v", streamID, wait.Truncate(time.Millisecond))
		select {
		case <-ctx.Done():
			globalFetchMu.Unlock()
			return "", "", nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	defer func() {
		globalLastFetchTime = time.Now()
		globalFetchMu.Unlock()
	}()

	if time.Now().Unix() < globalCaptchaLockout.Load() {
		return "", "", nil, fmt.Errorf("CAPTCHA_WAIT_REQUIRED: global lockout active")
	}

	user, pass, urls, err := getMaxCreds(ctx, link, streamID)
	if err != nil {
		if strings.Contains(err.Error(), "call_unavailable") {
			return "", "", nil, err
		}
		log.Printf("[STREAM %d] [CREDS] Max path failed: %v", streamID, err)
		return "", "", nil, err
	}

	setStreamCache(streamID, user, pass, urls)
	return user, pass, cloneStringSlice(urls), nil
}
