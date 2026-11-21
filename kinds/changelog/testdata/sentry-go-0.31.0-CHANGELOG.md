## 0.31.0

### Breaking Changes

- Remove support for metrics. Read more about the end of the Metrics beta [here](https://sentry.zendesk.com/hc/en-us/articles/26369339769883-Metrics-Beta-Ended-on-October-7th). ([#914](https://github.com/getsentry/sentry-go/pull/914))

- Remove support for profiling. ([#915](https://github.com/getsentry/sentry-go/pull/915))

- Remove `Segment` field from the `User` struct. This field is no longer used in the Sentry product. ([#928](https://github.com/getsentry/sentry-go/pull/928))

- Every integration is now a separate module, reducing the binary size and number of dependencies. Once you update `sentry-go` to latest version, you'll need to `go get` the integration you want to use. For example, if you want to use the `echo` integration, you'll need to run `go get github.com/getsentry/sentry-go/echo` ([#919](github.com/getsentry/sentry-go/pull/919)).

### Features

Add the ability to override `hub` in `context` for integrations that use custom context. ([#931](https://github.com/getsentry/sentry-go/pull/931))

- Add `HubProvider` Hook for `sentrylogrus`, enabling dynamic Sentry hub allocation for each log entry or goroutine. ([#936](https://github.com/getsentry/sentry-go/pull/936))

This change enhances compatibility with Sentry's recommendation of using separate hubs per goroutine. To ensure a separate Sentry hub for each goroutine, configure the `HubProvider` like this:

```go
hook, err := sentrylogrus.New(nil, sentry.ClientOptions{})
if err != nil {
    log.Fatalf("Failed to initialize Sentry hook: %v", err)
}

// Set a custom HubProvider to generate a new hub for each goroutine or log entry
hook.SetHubProvider(func() *sentry.Hub {
    client, _ := sentry.NewClient(sentry.ClientOptions{})
    return sentry.NewHub(client, sentry.NewScope())
})

logrus.AddHook(hook)
```

### Bug Fixes

- Add support for closing worker goroutines started by the `HTTPTranport` to prevent goroutine leaks. ([#894](https://github.com/getsentry/sentry-go/pull/894))

```go
client, _ := sentry.NewClient()
defer client.Close()
```

Worker can be also closed by calling `Close()` method on the `HTTPTransport` instance. `Close` should be called after `Flush` and before terminating the program otherwise some events may be lost.

```go
transport := sentry.NewHTTPTransport()
defer transport.Close()
```

### Misc

- Bump [gin-gonic/gin](https://github.com/gin-gonic/gin) to v1.9.1. ([#946](https://github.com/getsentry/sentry-go/pull/946))

## 0.30.0

The Sentry SDK team is happy to announce the immediate availability of Sentry Go SDK v0.30.0.

### Features

- Add `sentryzerolog` integration ([#857](https://github.com/getsentry/sentry-go/pull/857))
- Add `sentryslog` integration ([#865](https://github.com/getsentry/sentry-go/pull/865))
- Always set Mechanism Type to generic ([#896](https://github.com/getsentry/sentry-go/pull/897))

### Bug Fixes

- Prevent panic in `fasthttp` and `fiber` integration in case a malformed URL has to be parsed ([#912](https://github.com/getsentry/sentry-go/pull/912))

### Misc

Drop support for Go 1.18, 1.19 and 1.20. The currently supported Go versions are the last 3 stable releases: 1.23, 1.22 and 1.21.
