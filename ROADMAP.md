# Roadmap

## Schedule
 * Inject Storage to schedule (no storage implementation on spine to avoid backward compatibility issues. Save in memory by default)
 * Add schedule/distributed implementation

## Cache
 * Finish cache/groupcache implementation

## Hystrix
 * Implement hystrix in context

```go
context.Go // Async
context.Do // Sync

context.Go("foo_command", func() error {
	// talk to other services
	return nil
}, func(err error) error {
	// do this when services are down
	return nil
})
```

## Disco
 * Finish serf adapter

## BG
 * Implement groups with pool of go-routines (e.g. map.update - max 4)

## Admin
 * Add admin package to monitor the app, circuit breakers, drain, ...