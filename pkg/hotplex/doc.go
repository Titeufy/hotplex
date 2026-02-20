// Package hotplex provides the core SDK for managing persistent, hot-multiplexed
// Large Language Model (LLM) CLI agent sessions (e.g. Claude Code).
//
// It resolves the "cold start" latency issue by maintaining a pool of long-lived
// execution environments (sandboxes). The SDK uses a persistent marker system
// to allow seamless session recovery across process restarts and crashes.
//
// Security is enforced via an integrated Web Application Firewall (WAF) that
// inspects commands before execution. HotPlex also provides real-time streaming
// events, comprehensive token usage tracking, and automated cost reporting.
package hotplex
