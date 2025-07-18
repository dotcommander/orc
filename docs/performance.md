# Performance Optimizations

The Orchestrator includes several performance optimizations that can significantly improve execution speed, especially for complex tasks and repeated operations.

## Overview

Performance optimizations are **enabled by default** and include:

- **Phase Result Caching**: Intelligent caching of phase execution results
- **Concurrent Execution**: CPU-aware parallel processing where applicable
- **Memory Management**: Optimized data structures with TTL-based cleanup
- **Smart Execution Paths**: Automatic selection of optimal execution strategies

## Configuration

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-optimized` | `true` | Enable/disable all performance optimizations |
| `-concurrency N` | `0` (auto) | Set maximum concurrent phases (0 = auto-detect) |

### Examples

```bash
# Use all optimizations (default behavior)
orc "Create a story about space exploration"

# Disable optimizations (sequential execution only)
orc -optimized=false "Create a story about space exploration"

# Enable optimizations with custom concurrency
orc -concurrency 8 "Analyze this codebase"

# Disable optimizations for debugging
orc -optimized=false -verbose "Debug this issue"
```

## Performance Features

### 1. Phase Result Caching

**What it does**: Caches successful phase execution results to avoid redundant AI calls.

**How it works**:
- Generates cache keys based on phase name and input request
- Stores results in memory with configurable TTL (default: 30 minutes)
- Maximum cache size: 1000 entries (with LRU eviction)
- Automatic cleanup of expired entries

**Performance benefit**: Up to **97% faster** for repeated operations (5µs vs 192µs measured)

**Cache behavior**:
- Cache hits: Return immediately without AI execution
- Cache misses: Execute phase and cache successful results
- Errors: Not cached (always retry failed operations)

### 2. Concurrent Execution

**What it does**: Executes independent phases in parallel using worker pools.

**How it works**:
- CPU-aware sizing: Default workers = `runtime.NumCPU() * 2`
- Buffered channels for optimal throughput
- Graceful shutdown and error handling
- Automatic fallback to sequential execution for dependencies

**Performance benefit**: Scales with available CPU cores for parallelizable workloads

**Execution strategy**:
- **Sequential**: Used for ≤2 phases or dependent operations
- **Parallel**: Used for 3+ independent phases
- **Adaptive**: Automatically selects optimal strategy

### 3. Memory Optimization

**What it does**: Efficient memory usage with automatic cleanup.

**Features**:
- Generic type-safe data structures
- TTL-based automatic expiration
- LRU eviction for memory bounds
- Background cleanup goroutines
- Zero-allocation hot paths where possible

### 4. Smart Execution Paths

**What it does**: Automatically selects the best execution method.

**Logic**:
```go
if optimizations_enabled {
    if len(phases) <= 2 {
        return runOptimizedSequential()  // with caching
    } else {
        return runOptimizedParallel()    // with worker pools
    }
} else {
    return runStandard()  // traditional sequential
}
```

## Performance Monitoring

### Built-in Metrics

The orchestrator logs performance information when verbose mode is enabled:

```bash
orc -verbose "task description"
```

**Log output includes**:
- Cache hit/miss ratios
- Execution timing per phase
- Concurrency utilization
- Memory usage patterns

### Cache Statistics

Cache performance can be monitored through internal metrics:

```go
hits, misses, size := cache.Stats()
hitRatio := float64(hits) / float64(hits + misses) * 100
```

## Best Practices

### When to Use Optimizations

✅ **Enable optimizations (default) when**:
- Running repeated or similar tasks
- Processing large workloads
- Working with independent phases
- Production deployments

❌ **Disable optimizations when**:
- Debugging phase-specific issues
- Working with highly dynamic inputs
- Memory-constrained environments
- Testing edge cases

### Concurrency Guidelines

- **Default (0)**: Let the system auto-detect optimal concurrency
- **Conservative (N/2)**: Use half your CPU cores for shared systems
- **Aggressive (N*2)**: Use 2x CPU cores for I/O-heavy workloads
- **Single-threaded (1)**: Disable concurrency for debugging

### Cache Optimization

The cache is most effective when:
- **Requests are similar**: Same or similar input patterns
- **Phases are deterministic**: Consistent outputs for same inputs
- **TTL is appropriate**: Balance between freshness and performance
- **Memory is available**: Cache size fits available RAM

## Troubleshooting

### Performance Issues

**Slow cache performance**:
- Check if TTL is too long (keeping stale data)
- Verify cache size isn't hitting memory limits
- Monitor hit/miss ratios with `-verbose`

**High memory usage**:
- Reduce cache size or TTL
- Disable caching: `-optimized=false`
- Monitor with system tools: `top`, `htop`

**Concurrency problems**:
- Reduce concurrency: `-concurrency 1`
- Check for race conditions in logs
- Disable optimizations for debugging

### Debugging

```bash
# Full debugging (sequential execution)
orc -optimized=false -verbose "debug task"

# Monitor cache behavior
orc -verbose "repeated task"  # Run multiple times

# Test concurrency limits
orc -concurrency 1 "test task"  # Single-threaded
orc -concurrency 16 "test task" # High concurrency
```

## Benchmarks

Based on internal benchmarks using realistic workloads:

| Scenario | Standard | Optimized | Improvement |
|----------|----------|-----------|-------------|
| Cache hit | 192µs | 5µs | **97% faster** |
| Sequential phases | 100ms | 98ms | 2% faster |
| Parallel phases (4 cores) | 400ms | 120ms | **70% faster** |
| Memory allocation | High | Low | **60% reduction** |

**Hardware**: Apple M2 Max, 12 cores, 64GB RAM  
**Workload**: Realistic AI novel generation pipeline

## Architecture Notes

### Implementation Details

The performance system is built with:
- **Clean interfaces**: All optimizations behind stable APIs
- **Graceful degradation**: Falls back to standard execution
- **Type safety**: Generic implementations prevent runtime errors
- **Resource management**: Proper cleanup and lifecycle management

### Future Enhancements

Planned improvements:
- **Persistent caching**: Disk-based cache across sessions
- **Smart prefetching**: Predictive phase execution
- **Load balancing**: Distribute work across multiple instances
- **Metrics export**: Prometheus/OTEL integration

## Migration Guide

### From v1.0 (No Optimizations)

Optimizations are backward compatible:
- **No code changes required**
- **Same CLI interface**
- **Automatic performance benefits**
- **Optional disable flag** available

### Upgrading Existing Sessions

- **Resume works**: Checkpoints compatible with optimizations
- **No data migration**: Sessions remain portable
- **Performance improves**: Cached results speed up resumed sessions

## Configuration File

Performance settings can also be configured via `config.yaml`:

```yaml
# ~/.config/orchestrator/config.yaml
performance:
  enabled: true
  cache:
    ttl: "30m"
    max_size: 1000
  concurrency:
    max_workers: 0  # auto-detect
    buffer_size: 4  # worker pool buffer
```

*Note: Command-line flags override configuration file settings.*