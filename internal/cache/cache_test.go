package cache

import (
	"os"
	"testing"
	"time"
)

func TestKey_Deterministic(t *testing.T) {
	k1 := Key("openai", "explain this")
	k2 := Key("openai", "explain this")
	if k1 != k2 {
		t.Fatalf("Key not deterministic: %q vs %q", k1, k2)
	}
}

func TestKey_UniqueByProvider(t *testing.T) {
	if Key("openai", "x") == Key("ollama", "x") {
		t.Fatal("different providers should produce different keys")
	}
}

func TestKey_Length(t *testing.T) {
	k := Key("openai", "prompt")
	if len(k) != 32 {
		t.Fatalf("expected 32 chars, got %d", len(k))
	}
}

func TestSetGet_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	key := Key("test", "roundtrip")
	want := "hello world response"

	if err := Set(key, want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok := Get(key, 0)
	if !ok {
		t.Fatal("Get returned false after Set")
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGet_Miss(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, ok := Get("nonexistent-key", 0)
	if ok {
		t.Fatal("expected cache miss, got hit")
	}
}

func TestGet_Expired(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	key := Key("test", "expiry")
	if err := Set(key, "content"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// Backdate the file modification time so the entry appears old.
	path := Dir() + "/" + key + ".gz"
	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	_, ok := Get(key, 5*time.Minute)
	if ok {
		t.Fatal("expected expired cache miss, got hit")
	}
}

func TestGet_NotExpiredWithinWindow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	key := Key("test", "not-expired")
	if err := Set(key, "fresh"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	_, ok := Get(key, 1*time.Hour)
	if !ok {
		t.Fatal("expected cache hit within TTL window")
	}
}

func TestSet_AtomicOnLargeContent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	key := Key("test", "large")
	large := make([]byte, 512*1024) // 512 KB
	for i := range large {
		large[i] = byte('A' + (i % 26))
	}
	if err := Set(key, string(large)); err != nil {
		t.Fatalf("Set large content: %v", err)
	}
	got, ok := Get(key, 0)
	if !ok {
		t.Fatal("Get returned false for large content")
	}
	if got != string(large) {
		t.Fatal("large content mismatch after round-trip")
	}
}

func TestClear(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	key := Key("test", "clear")
	if err := Set(key, "to be cleared"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	_, ok := Get(key, 0)
	if ok {
		t.Fatal("expected cache miss after Clear")
	}
}
