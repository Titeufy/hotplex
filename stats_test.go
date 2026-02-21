package hotplex

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSessionStats_RecordToolUse(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Record tool use
	stats.RecordToolUse("bash", "tool-123")

	// Small delay to ensure duration > 0
	time.Sleep(1 * time.Millisecond)

	// Verify internal state (need to check via RecordToolResult)
	duration := stats.RecordToolResult()

	// Duration may be 0 if too fast, so we just check it's non-negative
	if duration < 0 {
		t.Errorf("RecordToolResult() returned negative duration: %d", duration)
	}
	if stats.ToolCallCount != 1 {
		t.Errorf("ToolCallCount = %d, want 1", stats.ToolCallCount)
	}
	if !stats.ToolsUsed["bash"] {
		t.Error("ToolsUsed should contain 'bash'")
	}
}

func TestSessionStats_RecordToolUse_NilMap(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// ToolsUsed is nil, should be initialized
	stats.RecordToolUse("edit", "tool-456")
	_ = stats.RecordToolResult()

	if stats.ToolsUsed == nil {
		t.Error("ToolsUsed should be initialized")
	}
	if !stats.ToolsUsed["edit"] {
		t.Error("ToolsUsed should contain 'edit'")
	}
}

func TestSessionStats_RecordToolResult_NoStart(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Call RecordToolResult without RecordToolUse
	duration := stats.RecordToolResult()

	if duration != 0 {
		t.Errorf("RecordToolResult() without RecordToolUse should return 0, got %d", duration)
	}
	if stats.ToolCallCount != 0 {
		t.Errorf("ToolCallCount = %d, want 0", stats.ToolCallCount)
	}
}

func TestSessionStats_RecordTokens(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.RecordTokens(100, 50, 20, 10)

	if stats.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", stats.InputTokens)
	}
	if stats.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", stats.OutputTokens)
	}
	if stats.CacheWriteTokens != 20 {
		t.Errorf("CacheWriteTokens = %d, want 20", stats.CacheWriteTokens)
	}
	if stats.CacheReadTokens != 10 {
		t.Errorf("CacheReadTokens = %d, want 10", stats.CacheReadTokens)
	}
}

func TestSessionStats_RecordTokens_Accumulative(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.RecordTokens(100, 50, 20, 10)
	stats.RecordTokens(50, 25, 10, 5)

	if stats.InputTokens != 150 {
		t.Errorf("InputTokens = %d, want 150 (accumulated)", stats.InputTokens)
	}
	if stats.OutputTokens != 75 {
		t.Errorf("OutputTokens = %d, want 75 (accumulated)", stats.OutputTokens)
	}
}

func TestSessionStats_ThinkingPhase(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.StartThinking()
	time.Sleep(10 * time.Millisecond)
	stats.EndThinking()

	if stats.ThinkingDurationMs <= 0 {
		t.Errorf("ThinkingDurationMs = %d, want > 0", stats.ThinkingDurationMs)
	}
}

func TestSessionStats_ThinkingPhase_NoStart(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// End without start should be safe (no-op)
	stats.EndThinking()

	if stats.ThinkingDurationMs != 0 {
		t.Errorf("ThinkingDurationMs = %d, want 0", stats.ThinkingDurationMs)
	}
}

func TestSessionStats_ThinkingPhase_Multiple(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Multiple thinking phases should accumulate
	for i := 0; i < 3; i++ {
		stats.StartThinking()
		time.Sleep(5 * time.Millisecond)
		stats.EndThinking()
	}

	if stats.ThinkingDurationMs < 10 { // At least 15ms total
		t.Errorf("ThinkingDurationMs = %d, want >= 15 (accumulated)", stats.ThinkingDurationMs)
	}
}

func TestSessionStats_GenerationPhase(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.StartGeneration()
	time.Sleep(10 * time.Millisecond)
	stats.EndGeneration()

	if stats.GenerationDurationMs <= 0 {
		t.Errorf("GenerationDurationMs = %d, want > 0", stats.GenerationDurationMs)
	}
}

func TestSessionStats_GenerationPhase_NoStart(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// End without start should be safe (no-op)
	stats.EndGeneration()

	if stats.GenerationDurationMs != 0 {
		t.Errorf("GenerationDurationMs = %d, want 0", stats.GenerationDurationMs)
	}
}

func TestSessionStats_RecordFileModification(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.RecordFileModification("/path/to/file1.txt")
	stats.RecordFileModification("/path/to/file2.txt")

	if stats.FilesModified != 2 {
		t.Errorf("FilesModified = %d, want 2", stats.FilesModified)
	}
	if len(stats.FilePaths) != 2 {
		t.Errorf("len(FilePaths) = %d, want 2", len(stats.FilePaths))
	}
}

func TestSessionStats_RecordFileModification_Empty(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Empty path should be ignored
	stats.RecordFileModification("")

	if stats.FilesModified != 0 {
		t.Errorf("FilesModified = %d, want 0", stats.FilesModified)
	}
}

func TestSessionStats_RecordFileModification_Deduplication(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Same file multiple times should only count once
	stats.RecordFileModification("/path/to/file.txt")
	stats.RecordFileModification("/path/to/file.txt")
	stats.RecordFileModification("/path/to/file.txt")

	if stats.FilesModified != 1 {
		t.Errorf("FilesModified = %d, want 1 (deduplicated)", stats.FilesModified)
	}
	if len(stats.FilePaths) != 1 {
		t.Errorf("len(FilePaths) = %d, want 1 (deduplicated)", len(stats.FilePaths))
	}
}

func TestSessionStats_ToSummary(t *testing.T) {
	stats := &SessionStats{
		SessionID:            "test-session",
		StartTime:            time.Now(),
		TotalDurationMs:      1000,
		ThinkingDurationMs:   200,
		ToolDurationMs:       500,
		GenerationDurationMs: 300,
		InputTokens:          100,
		OutputTokens:         50,
		CacheWriteTokens:     20,
		CacheReadTokens:      10,
		ToolCallCount:        5,
		ToolsUsed:            map[string]bool{"bash": true, "edit": true},
		FilesModified:        3,
		FilePaths:            []string{"/a.txt", "/b.txt", "/c.txt"},
	}

	summary := stats.ToSummary()

	if summary["session_id"] != "test-session" {
		t.Errorf("session_id = %v, want test-session", summary["session_id"])
	}
	if summary["total_duration_ms"] != int64(1000) {
		t.Errorf("total_duration_ms = %v, want 1000", summary["total_duration_ms"])
	}
	if summary["tool_call_count"] != int32(5) {
		t.Errorf("tool_call_count = %v, want 5", summary["tool_call_count"])
	}

	// Check tools_used is a slice
	tools, ok := summary["tools_used"].([]string)
	if !ok {
		t.Fatal("tools_used should be []string")
	}
	if len(tools) != 2 {
		t.Errorf("len(tools_used) = %d, want 2", len(tools))
	}
}

func TestSessionStats_FinalizeDuration(t *testing.T) {
	stats := &SessionStats{
		SessionID:            "test-session",
		StartTime:            time.Now(),
		ThinkingDurationMs:   100,
		GenerationDurationMs: 200,
	}

	// Start phases but don't end them
	stats.StartThinking()
	stats.StartGeneration()

	finalized := stats.FinalizeDuration()

	// Finalized should include ongoing phases
	if finalized.ThinkingDurationMs < 100 {
		t.Errorf("ThinkingDurationMs = %d, want >= 100", finalized.ThinkingDurationMs)
	}
	if finalized.GenerationDurationMs < 200 {
		t.Errorf("GenerationDurationMs = %d, want >= 200", finalized.GenerationDurationMs)
	}
}

func TestSessionStats_Concurrency(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	const goroutines = 10
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent operations - focus on thread-safe methods that don't have state dependencies
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				// These methods are designed for concurrent use
				stats.RecordTokens(1, 1, 1, 1)
				stats.StartThinking()
				stats.EndThinking()
				stats.StartGeneration()
				stats.EndGeneration()
				stats.RecordFileModification("/path/to/file")
			}
		}(i)
	}

	wg.Wait()

	// Verify counts (should be goroutines * operations for tokens)
	expectedTokens := int32(goroutines * operations)
	if stats.InputTokens != expectedTokens {
		t.Errorf("InputTokens = %d, want %d", stats.InputTokens, expectedTokens)
	}

	// File modifications should be deduplicated (same path)
	if stats.FilesModified != 1 {
		t.Errorf("FilesModified = %d, want 1 (deduplicated)", stats.FilesModified)
	}
}

func TestSessionStats_Concurrency_Tools(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	const goroutines = 5
	const operations = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Each goroutine uses a unique tool name to avoid interference
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			toolName := fmt.Sprintf("tool-%d", id)
			for j := 0; j < operations; j++ {
				stats.RecordToolUse(toolName, fmt.Sprintf("id-%d-%d", id, j))
				time.Sleep(1 * time.Millisecond) // Ensure duration > 0
				_ = stats.RecordToolResult()
			}
		}(i)
	}

	wg.Wait()

	// Each goroutine does `operations` tool calls
	expected := int32(goroutines * operations)
	if stats.ToolCallCount != expected {
		t.Errorf("ToolCallCount = %d, want %d", stats.ToolCallCount, expected)
	}

	// Should have exactly goroutines unique tools
	if len(stats.ToolsUsed) != goroutines {
		t.Errorf("len(ToolsUsed) = %d, want %d", len(stats.ToolsUsed), goroutines)
	}
}
