package codex

import (
	"strings"
	"testing"
	"time"
)

func TestDecodeThreadListUsesCodexNativeNames(t *testing.T) {
	t.Parallel()

	stream := strings.NewReader(strings.Join([]string{
		`{"id":0,"result":{"userAgent":"code_remote/0.155.1"}}`,
		`{"method":"remoteControl/status/changed","params":{"status":"disabled"}}`,
		`{"id":1,"result":{"data":[{"id":"019abcde-1234-7000-8000-123456789abc","name":"重构移动端 UI","preview":"请重新设计当前界面","createdAt":1789950000,"updatedAt":1789950300},{"id":"019abcde-1234-7000-8000-abcdefabcdef","preview":"修复触摸滚动","createdAt":1789940000,"updatedAt":1789940100}],"nextCursor":null}}`,
	}, "\n"))

	threads, err := decodeThreadList(stream, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 2 {
		t.Fatalf("threads = %#v, want 2", threads)
	}
	if threads[0].Name != "重构移动端 UI" || threads[0].Preview != "请重新设计当前界面" {
		t.Fatalf("first thread = %#v, want Codex native name and preview", threads[0])
	}
	if want := time.Unix(1789950300, 0).UTC(); !threads[0].UpdatedAt.Equal(want) {
		t.Fatalf("updated at = %v, want %v", threads[0].UpdatedAt, want)
	}
	if threads[1].DisplayName() != "修复触摸滚动" {
		t.Fatalf("unnamed DisplayName() = %q, want preview fallback", threads[1].DisplayName())
	}
}

func TestDecodeThreadListReturnsRPCError(t *testing.T) {
	t.Parallel()

	_, err := decodeThreadList(strings.NewReader(`{"id":1,"error":{"code":-32602,"message":"bad cwd"}}`), 1)
	if err == nil || !strings.Contains(err.Error(), "bad cwd") {
		t.Fatalf("decodeThreadList error = %v, want RPC message", err)
	}
}
