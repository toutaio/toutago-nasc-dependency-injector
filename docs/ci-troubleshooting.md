# CI Troubleshooting Guide

This guide helps you diagnose and fix common CI failures in the nasc project.

## Quick Reference

| Error | Quick Fix | Details |
|-------|-----------|---------|
| Code not formatted | `gofmt -w .` | [Formatting](#formatting-failures) |
| go vet failed | Check error details | [Static Analysis](#go-vet-failures) |
| Race detected | Review race report | [Race Conditions](#race-detector-failures) |
| Coverage too low | Add tests | [Coverage](#coverage-below-threshold) |
| Build failed | Check compile errors | [Build](#build-failures) |
| Vulnerability found | Update deps | [Security](#security-vulnerabilities) |

## Common Failures

### Formatting Failures

**Error Message:**
```
❌ Code is not formatted. Please run: gofmt -w .
```

**Cause:** Code doesn't follow Go formatting standards.

**Fix:**
```bash
# Format all Go files
gofmt -w .

# Verify formatting
gofmt -l .  # Should return nothing

# Commit changes
git add .
git commit -m "fix: format code"
git push
```

**Prevention:** Set up your editor to run `gofmt` on save.

---

### go vet Failures

**Error Message:**
```
go vet ./...
# Outputs specific issues
```

**Common Issues:**

1. **Printf format issues:**
   ```
   Error: fmt.Println call has arguments but no formatting directives
   ```
   Fix: Use proper formatting or remove extra arguments

2. **Redundant newlines:**
   ```
   Error: fmt.Println arg list ends with redundant newline
   ```
   Fix: Remove `\n` from `fmt.Println()` calls

3. **Unreachable code:**
   ```
   Error: unreachable code
   ```
   Fix: Remove code after return/panic statements

**Fix:**
```bash
# Run locally to see issues
go vet ./...

# Fix reported issues
# Then commit
git add .
git commit -m "fix: address go vet issues"
```

---

### staticcheck Failures

**Error Message:**
```
staticcheck ./...
# Lists specific warnings
```

**Common Issues:**

1. **Unused variables/functions:**
   ```
   Error: this value of x is never used
   ```
   Fix: Remove or use the variable

2. **Deprecated code:**
   ```
   Error: function X is deprecated
   ```
   Fix: Use recommended alternative

3. **Inefficient code:**
   ```
   Error: should use strings.Contains() instead
   ```
   Fix: Apply suggested improvement

**Fix:**
```bash
# Install staticcheck locally
go install honnef.co/go/tools/cmd/staticcheck@latest

# Run it
staticcheck ./...

# Fix issues and commit
```

---

### Race Detector Failures

**Error Message:**
```
WARNING: DATA RACE
Read at 0x... by goroutine X:
  ...
Previous write at 0x... by goroutine Y:
  ...
```

**Cause:** Concurrent access to shared memory without proper synchronization.

**Fix:**

1. **Add proper locking:**
   ```go
   type MyStruct struct {
       mu    sync.RWMutex
       data  map[string]interface{}
   }
   
   func (m *MyStruct) Get(key string) interface{} {
       m.mu.RLock()
       defer m.mu.RUnlock()
       return m.data[key]
   }
   ```

2. **Use channels:**
   ```go
   // Instead of shared variable
   resultChan := make(chan Result)
   go func() {
       resultChan <- computeResult()
   }()
   result := <-resultChan
   ```

3. **Use atomic operations:**
   ```go
   import "sync/atomic"
   
   var counter int64
   atomic.AddInt64(&counter, 1)
   ```

**Test locally:**
```bash
go test -race ./...
```

---

### Coverage Below Threshold

**Error Message:**
```
❌ Coverage 63.2% is below threshold of 80%
```

**Fix:**

1. **Check what's not covered:**
   ```bash
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out  # Opens browser
   ```

2. **Add tests for uncovered code:**
   - Focus on critical paths first
   - Test error cases
   - Add edge case tests

3. **Example test addition:**
   ```go
   func TestMyFunction_ErrorCase(t *testing.T) {
       result, err := MyFunction(nil)
       if err == nil {
           t.Error("Expected error for nil input")
       }
   }
   ```

**Current threshold:** 80% (goal is to increase over time)

---

### Build Failures

**Error Message:**
```
# github.com/toutaio/toutago-nasc-dependency-injector
./myfile.go:42:5: undefined: SomeFunction
```

**Common Causes:**

1. **Import missing:**
   ```go
   import "package/path"  // Add missing import
   ```

2. **Typo in function name:**
   ```go
   container.Bind(...)  // Check spelling
   ```

3. **Platform-specific code:**
   ```go
   // Use build tags if needed
   //go:build windows
   ```

**Fix:**
```bash
# Test build locally
go build ./...

# Check specific platform
GOOS=windows go build ./...
GOOS=darwin go build ./...
```

---

### Security Vulnerabilities

**Error Message:**
```
Vulnerability #1: GO-2024-XXXX
Package: golang.org/x/text
Description: ...
```

**Fix:**

1. **Update vulnerable package:**
   ```bash
   go get golang.org/x/text@latest
   go mod tidy
   ```

2. **If direct dependency:**
   ```bash
   go get package@safe-version
   ```

3. **If indirect dependency:**
   ```bash
   # Update parent package
   go get parent-package@latest
   ```

**Check locally:**
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

---

### go.mod Not Tidy

**Error Message:**
```
❌ go.mod or go.sum is not tidy
```

**Fix:**
```bash
go mod tidy
git add go.mod go.sum
git commit -m "fix: tidy go.mod"
```

**Common Causes:**
- Added/removed dependencies
- Upgraded Go version
- Manually edited go.mod

---

### Test Timeout

**Error Message:**
```
panic: test timed out after 10m0s
```

**Causes:**
- Deadlock in test
- Infinite loop
- Waiting for channel that never receives

**Fix:**

1. **Add timeouts to tests:**
   ```go
   func TestWithTimeout(t *testing.T) {
       ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
       defer cancel()
       
       // Use ctx in test
   }
   ```

2. **Check for deadlocks:**
   - Review channel operations
   - Ensure all goroutines complete
   - Use `select` with `default` or timeout

3. **Run test with verbose output:**
   ```bash
   go test -v -timeout 1m ./...
   ```

---

## Platform-Specific Issues

### Windows Build Failures

**Common Issues:**
- Path separators (`\` vs `/`)
- Line endings (CRLF vs LF)
- Case-sensitive paths

**Fix:**
- Use `filepath.Join()` for paths
- Use Go's `os` package for file operations
- Test on Windows if possible

### macOS Build Failures

**Less common** - usually same fixes as Linux work.

---

## CI Workflow Issues

### Actions Not Triggering

**Check:**
1. Workflow is on `main` branch
2. Pushed to `main` or opened PR
3. YAML syntax is valid

**Fix:**
```bash
# Validate YAML locally
cat .github/workflows/ci.yml | yamllint -
```

### Cache Not Working

**Symptom:** Every run downloads all dependencies.

**Check:**
- `go.sum` exists and is committed
- Cache key includes `go.sum` hash

**Usually resolves itself** after first successful run.

---

## Getting Help

### View CI Logs

1. Go to Actions tab on GitHub
2. Click on failed workflow run
3. Click on failed job
4. Expand failing step to see details

### Debug Locally

```bash
# Run all checks that CI runs
go test -race -timeout 10m ./...
go test -coverprofile=coverage.out ./...
gofmt -l .
go vet ./...
staticcheck ./...
go mod tidy
git diff go.mod go.sum
```

### Ask for Help

If stuck:
1. Include full error message
2. Mention what you've tried
3. Link to failed CI run
4. Open issue on GitHub

---

## Preventive Measures

### Pre-commit Checklist

Before pushing:
```bash
# Format code
gofmt -w .

# Run tests
go test ./...

# Check for races
go test -race ./...

# Lint
go vet ./...

# Tidy modules
go mod tidy
```

### Git Pre-commit Hook

Create `.git/hooks/pre-commit`:
```bash
#!/bin/bash
set -e

echo "Running pre-commit checks..."

# Format
gofmt -w .

# Test
go test ./...

# Vet
go vet ./...

# Tidy
go mod tidy

echo "✅ All checks passed"
```

Make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

---

## Performance Tips

### Speed Up CI Locally

```bash
# Run only changed packages
go test ./nasc ./registry

# Skip slow tests
go test -short ./...

# Run specific test
go test -run TestName ./...
```

### Speed Up CI Pipeline

- Ensure `go.sum` is committed (enables caching)
- Don't run benchmarks on every commit
- Use `fail-fast: false` to see all failures

---

**Last Updated:** 2024-12-28  
**Workflow Version:** 1.0
