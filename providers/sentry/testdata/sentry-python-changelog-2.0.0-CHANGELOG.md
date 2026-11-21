## 2.0.0

This is the first major update in a *long* time! 

We dropped support for some ancient languages and frameworks (Yes, Python 2.7 is no longer supported). Additionally we refactored a big part of the foundation of the SDK (how data inside the SDK is handled).

We hope you like it!

For a shorter version of what you need to do, to upgrade to Sentry SDK 2.0 see: https://docs.sentry.io/platforms/python/migration/1.x-to-2.x 

### New Features

- Additional integrations will now be activated automatically if the SDK detects the respective package is installed: Ariadne, ARQ, asyncpg, Chalice, clickhouse-driver, GQL, Graphene, huey, Loguru, PyMongo, Quart, Starlite, Strawberry.
- Added new API for custom instrumentation: `new_scope`, `isolation_scope`. See the [Deprecated](#deprecated) section to see how they map to the existing APIs.

### Changed
(These changes are all backwards-incompatible. **Breaking Change** (if you are just skimming for that phrase))

- The Pyramid integration will not capture errors that might happen in `authenticated_userid()` in a custom `AuthenticationPolicy` class.
- The method `need_code_loation` of the `MetricsAggregator` was renamed to `need_code_location`.
- The `BackgroundWorker` thread used to process events was renamed from `raven-sentry.BackgroundWorker` to `sentry-sdk.BackgroundWorker`.
- The `reraise` function was moved from `sentry_sdk._compat` to `sentry_sdk.utils`.
- The `_ScopeManager` was moved from `sentry_sdk.hub` to `sentry_sdk.scope`.
- Moved the contents of `tracing_utils_py3.py` to `tracing_utils.py`. The `start_child_span_decorator` is now in `sentry_sdk.tracing_utils`.
- The actual implementation of `get_current_span` was moved to `sentry_sdk.tracing_utils`. `sentry_sdk.get_current_span` is still accessible as part of the top-level API.
- `sentry_sdk.tracing_utils.add_query_source()`: Removed the `hub` parameter. It is not necessary anymore.
- `sentry_sdk.tracing_utils.record_sql_queries()`: Removed the `hub` parameter. It is not necessary anymore.
- `sentry_sdk.tracing_utils.get_current_span()` does now take a `scope` instead of a `hub` as parameter.
- `sentry_sdk.tracing_utils.should_propagate_trace()` now takes a `Client` instead of a `Hub` as first parameter.
- `sentry_sdk.utils.is_sentry_url()` now takes a `Client` instead of a `Hub` as first parameter.
- `sentry_sdk.utils._get_contextvars` does not return a tuple with three values, but a tuple with two values. The `copy_context` was removed.
- If you create a transaction manually and later mutate the transaction in a `configure_scope` block this does not work anymore. Here is a recipe on how to change your code to make it work:
    Your existing implementation:
    ```python
    transaction = sentry_sdk.transaction(...)

    # later in the code execution:

    with sentry_sdk.configure_scope() as scope:
        scope.set_transaction_name("new-transaction-name")
    ```

    needs to be changed to this:
    ```python
    transaction = sentry_sdk.transaction(...)

    # later in the code execution:

    scope = sentry_sdk.Scope.get_current_scope()
    scope.set_transaction_name("new-transaction-name")
    ```
- The classes listed in the table below are now abstract base classes. Therefore, they can no longer be instantiated. Subclasses can only be instantiated if they implement all of the abstract methods.
  <details>
    <summary><b>Show table</b></summary>

  | Class                                 | Abstract methods                       |
  | ------------------------------------- | -------------------------------------- |
  | `sentry_sdk.integrations.Integration` | `setup_once`                           |
  | `sentry_sdk.metrics.Metric`           | `add`, `serialize_value`, and `weight` |
  | `sentry_sdk.profiler.Scheduler`       | `setup` and `teardown`                 |
  | `sentry_sdk.transport.Transport`      | `capture_envelope`                     |

    </details>

### Removed
(These changes are all backwards-incompatible. **Breaking Change** (if you are just skimming for that phrase))

- Removed support for Python 2 and Python 3.5. The SDK now requires at least Python 3.6.
- Removed support for Celery 3.\*.
- Removed support for Django 1.8, 1.9, 1.10.
- Removed support for Flask 0.\*.
- Removed support for gRPC < 1.39.
- Removed support for Tornado < 6.
- Removed `last_event_id()` top level API. The last event ID is still returned by `capture_event()`, `capture_exception()` and `capture_message()` but the top level API `sentry_sdk.last_event_id()` has been removed.
- Removed support for sending events to the `/store` endpoint. Everything is now sent to the `/envelope` endpoint. If you're on SaaS you don't have to worry about this, but if you're running Sentry yourself you'll need version `20.6.0` or higher of self-hosted Sentry.
- The deprecated `with_locals` configuration option was removed. Use `include_local_variables` instead. See https://docs.sentry.io/platforms/python/configuration/options/#include-local-variables.
- The deprecated `request_bodies` configuration option was removed. Use `max_request_body_size`. See https://docs.sentry.io/platforms/python/configuration/options/#max-request-body-size.
- Removed support for `user.segment`. It was also removed from the trace header as well as from the dynamic sampling context.
- Removed support for the `install` method for custom integrations. Please use `setup_once` instead.
- Removed `sentry_sdk.tracing.Span.new_span`. Use `sentry_sdk.tracing.Span.start_child` instead.
- Removed `sentry_sdk.tracing.Transaction.new_span`. Use `sentry_sdk.tracing.Transaction.start_child` instead.
- Removed support for creating transactions via `sentry_sdk.tracing.Span(transaction=...)`. To create a transaction, please use `sentry_sdk.tracing.Transaction(name=...)`.
- Removed `sentry_sdk.utils.Auth.store_api_url`.
- `sentry_sdk.utils.Auth.get_api_url`'s now accepts a `sentry_sdk.consts.EndpointType` enum instead of a string as its only parameter. We recommend omitting this argument when calling the function, since the parameter's default value is the only possible `sentry_sdk.consts.EndpointType` value. The parameter exists for future compatibility.
- Removed `tracing_utils_py2.py`. The `start_child_span_decorator` is now in `sentry_sdk.tracing_utils`.
- Removed the `sentry_sdk.profiler.Scheduler.stop_profiling` method. Any calls to this method can simply be removed, since this was a no-op method.

### Deprecated

- Using the `Hub` directly as well as using hub-based APIs has been deprecated. Where available, use [the top-level API instead](sentry_sdk/api.py); otherwise use the [scope API](sentry_sdk/scope.py) or the [client API](sentry_sdk/client.py).

  Before:

  ```python
  with hub.start_span(...):
      # do something
  ```

  After:

  ```python
  import sentry_sdk

  with sentry_sdk.start_span(...):
      # do something
  ```

- Hub cloning is deprecated.

  Before:

  ```python
  with Hub(Hub.current) as hub:
      # do something with the cloned hub
  ```

  After:

  ```python
  import sentry_sdk

  with sentry_sdk.isolation_scope() as scope:
      # do something with the forked scope
  ```

- `configure_scope` is deprecated. Use the new isolation scope directly via `Scope.get_isolation_scope()` instead.

  Before:

  ```python
  with configure_scope() as scope:
      # do something with `scope`
  ```

  After:

  ```python
  from sentry_sdk.scope import Scope

  scope = Scope.get_isolation_scope()
  # do something with `scope`
  ```

- `push_scope` is deprecated. Use the new `new_scope` context manager to fork the necessary scopes.

  Before:

  ```python
  with push_scope() as scope:
      # do something with `scope`
  ```

  After:

  ```python
  import sentry_sdk

  with sentry_sdk.new_scope() as scope:
      # do something with `scope`
  ```

- Accessing the client via the hub has been deprecated. Use the top-level `sentry_sdk.get_client()` to get the current client.
- `profiler_mode` and `profiles_sample_rate` have been deprecated as `_experiments` options. Use them as top level options instead:
  ```python
  sentry_sdk.init(
      ...,
      profiler_mode="thread",
      profiles_sample_rate=1.0,
  )
  ```
- Deprecated `sentry_sdk.transport.Transport.capture_event`. Please use `sentry_sdk.transport.Transport.capture_envelope`, instead.
- Passing a function to `sentry_sdk.init`'s `transport` keyword argument has been deprecated. If you wish to provide a custom transport, please pass a `sentry_sdk.transport.Transport` instance or a subclass.
- The parameter `propagate_hub` in `ThreadingIntegration()` was deprecated and renamed to `propagate_scope`.

## 1.45.0

This is the final 1.x release for the forseeable future. Development will continue on the 2.x release line. The first 2.x version will be available in the next few weeks.

### Various fixes & improvements

- Allow to upsert monitors (#2929) by @sentrivana

  It's now possible to provide `monitor_config` to the `monitor` decorator/context manager directly:

  ```python
  from sentry_sdk.crons import monitor

  # All keys except `schedule` are optional
  monitor_config = {
      "schedule": {"type": "crontab", "value": "0 0 * * *"},
      "timezone": "Europe/Vienna",
      "checkin_margin": 10,
      "max_runtime": 10,
      "failure_issue_threshold": 5,
      "recovery_threshold": 5,
  }
  
  @monitor(monitor_slug='<monitor-slug>', monitor_config=monitor_config)
  def tell_the_world():
      print('My scheduled task...')
  ```

  Check out [the cron docs](https://docs.sentry.io/platforms/python/crons/) for details.

- Add Django `signals_denylist` to filter signals that are attached to by `signals_spans` (#2758) by @lieryan

  If you want to exclude some Django signals from performance tracking, you can use the new `signals_denylist` Django option:

  ```python
  import django.db.models.signals
  import sentry_sdk
  
  sentry_sdk.init(
      ...
      integrations=[
          DjangoIntegration(
              ...
              signals_denylist=[
                  django.db.models.signals.pre_init, 
                  django.db.models.signals.post_init,
              ],
          ),
      ],
  )
  ```

- `increment` for metrics (#2588) by @mitsuhiko

  `increment` and `inc` are equivalent, so you can pick whichever you like more.

- Add `value`, `unit` to `before_emit_metric` (#2958) by @sentrivana

  If you add a custom `before_emit_metric`, it'll now accept 4 arguments (the `key`, `value`, `unit` and `tags`) instead of just `key` and `tags`.

  ```python
  def before_emit(key, value, unit, tags):
      if key == "removed-metric":
          return False
      tags["extra"] = "foo"
      del tags["release"]
      return True
  
  sentry_sdk.init(
      ...
      _experiments={
          "before_emit_metric": before_emit,
      }
  )
  ```

- Remove experimental metric summary options (#2957) by @sentrivana

  The `_experiments` options `metrics_summary_sample_rate` and `should_summarize_metric` have been removed.

- New normalization rules for metric keys, names, units, tags (#2946) by @sentrivana
- Change `data_category` from `statsd` to `metric_bucket` (#2954) by @cleptric
- Accessing `__mro__` might throw a `ValueError` (#2952) by @sentrivana
- Suppress prompt spawned by subprocess when using `pythonw` (#2936) by @collinbanko
- Handle `None` in GraphQL query #2715 (#2762) by @czyber
- Do not send "quiet" Sanic exceptions to Sentry (#2821) by @hamedsh
- Implement `metric_bucket` rate limits (#2933) by @cleptric
- Fix type hints for `monitor` decorator (#2944) by @szokeasaurusrex
- Remove deprecated `typing` imports in crons (#2945) by @szokeasaurusrex
- Make `monitor_config` a `TypedDict` (#2931) by @sentrivana
- Add `devenv-requirements.txt` and update env setup instructions (#2761) by @arr-ee
- Bump `types-protobuf` from `4.24.0.20240311` to `4.24.0.20240408` (#2941) by @dependabot
- Disable Codecov check run annotations (#2537) by @eliatcodecov
