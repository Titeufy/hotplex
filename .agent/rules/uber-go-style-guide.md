# Uber Go Style Guide - HotPlex Reference

This document summarizes the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) with HotPlex-specific context. All AI agents working on HotPlex must follow these guidelines.

---

## Concurrency & Safety

### 1. Zero-Value Mutexes Are Valid

The zero-value of `sync.Mutex` and `sync.RWMutex` is valid, so you almost never need a pointer to a mutex.

```go
// ❌ Bad
mu := new(sync.Mutex)
mu.Lock()

// ✅ Good
var mu sync.Mutex
mu.Lock()
```

**For HotPlex**: Don't embed mutex in exported structs — use named fields.

```go
// ❌ Bad - embedded mutex
type SMap struct {
    sync.Mutex
    data map[string]string
}

// ✅ Good - named field
type SMap struct {
    mu   sync.Mutex
    data map[string]string
}
```

---

### 2. Copy Slices and Maps at Boundaries

Slices and maps contain pointers to underlying data. Always copy them when passing across API boundaries to prevent external mutation.

```go
// ❌ Bad — caller can modify internal state
func (d *Driver) SetTrips(trips []Trip) {
    d.trips = trips
}

// ✅ Good — defensive copy
func (d *Driver) SetTrips(trips []Trip) {
    d.trips = make([]Trip, len(trips))
    copy(d.trips, trips)
}
```

**For HotPlex**: Critical for `SessionPool` — always deep-copy maps/slices returned to callers.

---

### 3. Defer for Clean Up

Use `defer` to clean up resources such as files and locks. Place immediately after acquisition.

```go
// ❌ Bad — easy to miss unlocks due to multiple returns
p.Lock()
if p.count < 10 {
    p.Unlock()
    return p.count
}
p.count++
newCount := p.count
p.Unlock()
return newCount

// ✅ Good — defer ensures cleanup
p.Lock()
defer p.Unlock()

if p.count < 10 {
    return p.count
}
p.count++
return p.count
```

**For HotPlex**: Use `defer` for mutex unlocking, file closures, and connection cleanup.

---

### 4. Channel Size is One or None

Channels should be unbuffered (size 0) or buffered with size 1. Larger buffers require explicit justification.

```go
// ❌ Bad — why 64?
c := make(chan int, 64)

// ✅ Good
c := make(chan int, 1)  // or
c := make(chan int)      // unbuffered
```

---

### 5. Don't Fire-and-Forget Goroutines

Never launch goroutines without a plan for their termination. Always use `sync.WaitGroup` or `context` for coordination.

```go
// ❌ Bad — no termination plan
go func() {
    for {
        select {
        case <-ch:
            // do work
        }
    }
}()

// ✅ Good — with context for cancellation
func startWorker(ctx context.Context, ch chan Work) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case w := <-ch:
                process(w)
            }
        }
    }()
}
```

**For HotPlex**: HotPlex sessions spawn CLI processes — always ensure proper cleanup via `sync.WaitGroup` or `context`.

---

### 6. Use `go.uber.org/atomic`

The `sync/atomic` package uses raw types, making it easy to forget atomic operations. Use `go.uber.org/atomic` for type safety.

```go
// ❌ Bad — race condition!
type foo struct {
    running int32
}

func (f *foo) isRunning() bool {
    return f.running == 1  // non-atomic read
}

// ✅ Good
type foo struct {
    running atomic.Bool
}

func (f *foo) isRunning() bool {
    return f.running.Load()
}
```

**For HotPlex**: Use `atomic.Bool`, `atomic.Int64` for simple counters and flags in session management.

---

## Error Handling

### 7. Don't Panic

Panics cause cascading failures. Always return errors and let callers decide how to handle them.

```go
// ❌ Bad
func run(args []string) {
    if len(args) == 0 {
        panic("an argument is required")
    }
}

// ✅ Good
func run(args []string) error {
    if len(args) == 0 {
        return errors.New("an argument is required")
    }
    return nil
}
```

**For HotPlex**: Never `panic()` in core engine — return errors instead. Only panic for truly unrecoverable initialization errors.

---

### 8. Error Types

Use static errors for comparison and custom types for dynamic context.

```go
// Static error — use var
var ErrNotFound = errors.New("not found")

// Dynamic error — use custom type
type NotFoundError struct {
    File string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("file %q not found", e.File)
}
```

**For HotPlex**: Define top-level errors for session states, CLI errors, and WAF violations.

---

### 9. Error Wrapping with `%w`

Use `%w` for error chaining to preserve the error chain.

```go
// ✅ Good
if err != nil {
    return fmt.Errorf("open config: %w", err)
}

// Then use errors.Is / errors.As for checking
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

---

### 10. Handle Errors Once

Don't log AND return the same error — let callers handle it. Only log if you're recovering gracefully.

```go
// ❌ Bad — double handling
if err != nil {
    log.Printf("failed: %v", err)
    return err
}

// ✅ Good — wrap and return
if err != nil {
    return fmt.Errorf("get user %q: %w", id, err)
}

// ✅ Good — degrade gracefully (log + recover)
if err := emitMetrics(); err != nil {
    log.Printf("could not emit metrics: %v", err) // non-fatal
    // continue execution
}
```

---

### 11. Handle Type Assertion Failures

Single-value type assertion panics on failure. Always use the comma-ok idiom.

```go
// ❌ Bad — panics if not string
t := i.(string)

// ✅ Good
t, ok := i.(string)
if !ok {
    return fmt.Errorf("unexpected type %T", i)
}
```

---

### 11.1 Log Type Assertion Failures in Integration Code (HotPlex)

When performing interface-based type assertions in integration/wiring code, **always log failures** for debugging.

```go
// ❌ Bad — silent failure, impossible to debug
if engineSupport, ok := adapter.(base.EngineSupport); ok {
    engineSupport.SetEngine(eng)
}

// ✅ Good — logs help identify interface mismatch issues
if engineSupport, ok := adapter.(base.EngineSupport); ok {
    engineSupport.SetEngine(eng)
    logger.Debug("Engine injected", "platform", platform)
} else {
    logger.Warn("Adapter does not implement EngineSupport",
        "platform", platform,
        "adapter_type", fmt.Sprintf("%T", adapter))
}
```

**Common assertion points requiring logs:**
- `setup.go`: EngineSupport, WebhookProvider
- `manager.go`: MessageOperations, SessionOperations
- Any `adapter.(Interface)` pattern

**Rationale**: Issue #99 showed that silent type assertion failures are invisible until runtime symptoms appear. Logs enable rapid diagnosis.

## Code Quality

### 12. Verify Interface Compliance at Compile Time

Use compile-time checks to ensure types implement interfaces correctly. **MANDATORY for ALL interfaces.**

```go
// ✅ Good — compile-time verification
type Handler struct{}

var _ http.Handler = (*Handler)(nil)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}
```

**For HotPlex (MANDATORY)**:

1. **Every interface implementation MUST have compile-time verification:**

```go
// chatapps/slack/adapter.go
var _ base.ChatAdapter = (*Adapter)(nil)
var _ base.EngineSupport = (*Adapter)(nil)
var _ base.MessageOperations = (*Adapter)(nil)
```

2. **New interface definition = immediate compliance check required:**

```go
// When defining new interface:
type EngineSupport interface {
    SetEngine(eng *engine.Engine)
}

// Immediately add check in implementing type:
var _ EngineSupport = (*Adapter)(nil)  // ← Mandatory
```

3. **Self-check after refactoring:**
   - Did I define a new interface? → Add compliance check
   - Did I modify an interface signature? → Verify all implementations compile

**Rationale**: Issue #99 cost hours of debugging because `EngineSupport` interface signature (`any`) didn't match implementation (`*engine.Engine`). Compile-time check would have caught this instantly.

---

### 13. No Pointers to Interfaces

You rarely need a pointer to an interface. Pass interfaces as values; the underlying data can still be a pointer.

```go
// ❌ Bad
func (h *Handler) Process(i *io.Reader) {}

// ✅ Good — interface as value
func (h Handler) Process(i io.Reader) {}
```

---

### 14. Avoid Mutable Globals — Use Dependency Injection

Don't use global variables; inject dependencies instead. Enables testability.

```go
// ❌ Bad
var _timeNow = time.Now

func sign(msg string) string {
    return signWithTime(msg, _timeNow())
}

// ✅ Good
type signer struct {
    now func() time.Time
}

func newSigner() *signer {
    return &signer{now: time.Now}
}
```

**For HotPlex**: Pass logger, config, and providers as dependencies — don't use global singletons.

---

### 15. Avoid Embedding Types in Public Structs

Embedded types leak implementation details and limit future changes. Use named fields instead.

```go
// ❌ Bad
type ConcreteList struct {
    *AbstractList  // leaks implementation
}

// ✅ Good
type ConcreteList struct {
    list *AbstractList
}

func (l *ConcreteList) Add(e Entity) {
    l.list.Add(e)
}
```

**For HotPlex**: Avoid embedding `sync.Mutex` or other types in exported session structs.

---

### 16. Use `time.Duration` for Time

Never use raw `int` for time. Use `time.Time` for instants, `time.Duration` for intervals.

```go
// ❌ Bad
func poll(delay int) {
    time.Sleep(time.Duration(delay) * time.Millisecond)
}

// ✅ Good
func poll(delay time.Duration) {
    time.Sleep(delay)
}

// Usage
poll(10 * time.Second)
```

**For HotPlex**: Use `time.Duration` for timeouts, session TTLs, and retry delays.

---

### 17. Reduce Nesting

Early returns reduce nesting and improve readability.

```go
// ❌ Bad — deep nesting
if a {
    if b {
        if c {
            doSomething()
        }
    }
}

// ✅ Good — early returns
if !a {
    return
}
if !b {
    return
}
if !c {
    return
}
doSomething()
```

---

### 18. Consistency

If you're doing something one way in one file, do it the same way throughout the codebase. **Consistency > personal preference.**

---

## Quick Reference Summary

| Category | Key Guidelines |
|----------|---------------|
| **Concurrency** | Zero-value mutexes • Defer cleanup • Channel size 0/1 • No fire-and-forget goroutines • Use `go.uber.org/atomic` |
| **Errors** | Never panic • Static errors via `var` • Wrap with `%w` • Handle errors once • Safe type assertions |
| **Quality** | Verify interface compliance (**MANDATORY**) • Log type assertion failures • No pointers to interfaces • Dependency injection • Use `time.Duration` • Consistency |

---

## HotPlex-Specific Checklist

Before submitting code, verify:

- [ ] New interface has `var _ Interface = (*Impl)(nil)` check
- [ ] Type assertions in integration code have failure logs
- [ ] No silent `adapter.(Interface)` without logging the `ok=false` case
