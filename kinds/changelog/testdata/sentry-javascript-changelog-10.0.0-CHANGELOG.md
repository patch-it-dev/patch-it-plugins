## 10.0.0

Version `10.0.0` marks a release of the Sentry JavaScript SDKs that contains breaking changes. The goal of this release is to primarily upgrade the underlying OpenTelemetry dependencies to v2 with minimal breaking changes.

### How To Upgrade

Please carefully read through the migration guide in the Sentry docs on how to upgrade from version 9 to version 10. Make sure to select your specific platform/framework in the top left corner: https://docs.sentry.io/platforms/javascript/migration/v9-to-v10/

A comprehensive migration guide outlining all changes can be found within the Sentry JavaScript SDK Repository: https://github.com/getsentry/sentry-javascript/blob/develop/MIGRATION.md

### Breaking Changes

- feat!: Bump to OpenTelemetry v2 ([#16872](https://github.com/getsentry/sentry-javascript/pull/16872))
- feat(browser)!: Remove FID web vital collection ([#17076](https://github.com/getsentry/sentry-javascript/pull/17076))
- feat(core)!: Remove `BaseClient` ([#17071](https://github.com/getsentry/sentry-javascript/pull/17071))
- feat(core)!: Remove `enableLogs` and `beforeSendLog` experimental options ([#17063](https://github.com/getsentry/sentry-javascript/pull/17063))
- feat(core)!: Remove `hasTracingEnabled` ([#17072](https://github.com/getsentry/sentry-javascript/pull/17072))
- feat(core)!: Remove deprecated logger ([#17061](https://github.com/getsentry/sentry-javascript/pull/17061))
- feat(replay)!: Promote `_experiments.autoFlushOnFeedback` option as default ([#17220](https://github.com/getsentry/sentry-javascript/pull/17220))
- chore(deps)!: Bump bundler plugins to v4 ([#17089](https://github.com/getsentry/sentry-javascript/pull/17089))

### Other Changes

- feat(astro): Implement Request Route Parametrization for Astro 5 ([#17105](https://github.com/getsentry/sentry-javascript/pull/17105))
- feat(astro): Parametrize routes on client-side ([#17133](https://github.com/getsentry/sentry-javascript/pull/17133))
- feat(aws): Add `SentryNodeServerlessSDKv10` v10 AWS Lambda Layer ([#17069](https://github.com/getsentry/sentry-javascript/pull/17069))
- feat(aws): Create unified lambda layer for ESM and CJS ([#17012](https://github.com/getsentry/sentry-javascript/pull/17012))
- feat(aws): Detect SDK source for AWS Lambda layer ([#17128](https://github.com/getsentry/sentry-javascript/pull/17128))
- feat(core): Add missing openai tool calls attributes ([#17226](https://github.com/getsentry/sentry-javascript/pull/17226))
- feat(core): Add shared `flushIfServerless` function ([#17177](https://github.com/getsentry/sentry-javascript/pull/17177))
- feat(core): Implement `strictTraceContinuation` ([#16313](https://github.com/getsentry/sentry-javascript/pull/16313))
- feat(core): MCP server instrumentation without breaking Miniflare ([#16817](https://github.com/getsentry/sentry-javascript/pull/16817))
- feat(deps): bump @prisma/instrumentation from 6.11.1 to 6.12.0 ([#17117](https://github.com/getsentry/sentry-javascript/pull/17117))
- feat(meta): Unify detection of serverless environments and add Cloud Run ([#17168](https://github.com/getsentry/sentry-javascript/pull/17168))
- feat(nestjs): Switch to OTel core instrumentation ([#17068](https://github.com/getsentry/sentry-javascript/pull/17068))
- feat(node-native): Upgrade `@sentry-internal/node-native-stacktrace` to `0.2.2` ([#17207](https://github.com/getsentry/sentry-javascript/pull/17207))
- feat(node): Add `shouldHandleError` option to `fastifyIntegration` ([#16845](https://github.com/getsentry/sentry-javascript/pull/16845))
- feat(node): Add firebase integration ([#16719](https://github.com/getsentry/sentry-javascript/pull/16719))
- feat(node): Instrument stream responses for openai ([#17110](https://github.com/getsentry/sentry-javascript/pull/17110))
- feat(react-router): Add `createSentryHandleError` ([#17235](https://github.com/getsentry/sentry-javascript/pull/17235))
- feat(react-router): Automatically flush on serverless for loaders/actions ([#17234](https://github.com/getsentry/sentry-javascript/pull/17234))
- feat(react-router): Automatically flush on Vercel for request handlers ([#17232](https://github.com/getsentry/sentry-javascript/pull/17232))
- fix(astro): Construct parametrized route during runtime ([#17190](https://github.com/getsentry/sentry-javascript/pull/17190))
- fix(aws): Add layer build output to nx cache ([#17148](https://github.com/getsentry/sentry-javascript/pull/17148))
- fix(aws): Fix path to packages directory ([#17112](https://github.com/getsentry/sentry-javascript/pull/17112))
- fix(aws): Resolve all Sentry packages to local versions in layer build ([#17106](https://github.com/getsentry/sentry-javascript/pull/17106))
- fix(aws): Use file link in dependency version ([#17111](https://github.com/getsentry/sentry-javascript/pull/17111))
- fix(cloudflare): Allow non uuid workflow instance IDs ([#17121](https://github.com/getsentry/sentry-javascript/pull/17121))
- fix(cloudflare): Avoid turning DurableObject sync methods into async ([#17184](https://github.com/getsentry/sentry-javascript/pull/17184))
- fix(core): Fix OpenAI SDK private field access by binding non-instrumented fns ([#17163](https://github.com/getsentry/sentry-javascript/pull/17163))
- fix(core): Fix operation name for openai responses API ([#17206](https://github.com/getsentry/sentry-javascript/pull/17206))
- fix(core): Update ai.response.object to gen_ai.response.object ([#17153](https://github.com/getsentry/sentry-javascript/pull/17153))
- fix(nextjs): Flush in route handlers ([#17223](https://github.com/getsentry/sentry-javascript/pull/17223))
- fix(nextjs): Handle async params in url extraction ([#17162](https://github.com/getsentry/sentry-javascript/pull/17162))
- fix(nextjs): Update stackframe calls for next v15.5 ([#17156](https://github.com/getsentry/sentry-javascript/pull/17156))
- fix(node): Add mechanism to `fastifyIntegration` error handler ([#17208](https://github.com/getsentry/sentry-javascript/pull/17208))
- fix(node): Ensure tool errors for `vercelAiIntegration` have correct trace connected ([#17132](https://github.com/getsentry/sentry-javascript/pull/17132))
- fix(node): Fix exports for openai instrumentation ([#17238](https://github.com/getsentry/sentry-javascript/pull/17238))
- fix(node): Handle stack traces with data URI filenames ([#17218](https://github.com/getsentry/sentry-javascript/pull/17218))
- fix(react): Memoize wrapped component to prevent rerenders ([#17230](https://github.com/getsentry/sentry-javascript/pull/17230))
- fix(remix): Ensure source maps upload fails silently if Sentry CLI fails ([#17082](https://github.com/getsentry/sentry-javascript/pull/17082))
- fix(replay): Fix re-sampled sessions after a click ([#17008](https://github.com/getsentry/sentry-javascript/pull/17008))
- fix(svelte): Do not insert preprocess code in script module in Svelte 5 ([#17114](https://github.com/getsentry/sentry-javascript/pull/17114))
- fix(sveltekit): Align error status filtering and mechanism in `handleErrorWithSentry` ([#17157](https://github.com/getsentry/sentry-javascript/pull/17157))

Work in this release was contributed by @richardjelinek-fastest. Thank you for your contribution!

## 9.40.0

### Important Changes

- **feat(browser): Add debugId sync APIs between web worker and main thread ([#16981](https://github.com/getsentry/sentry-javascript/pull/16981))**

This release adds two Browser SDK APIs to let the main thread know about debugIds of worker files:

- `webWorkerIntegration({worker})` to be used in the main thread
- `registerWebWorker({self})` to be used in the web worker

```js
// main.js
Sentry.init({...})

const worker = new MyWorker(...);

Sentry.addIntegration(Sentry.webWorkerIntegration({ worker }));

worker.addEventListener('message', e => {...});
```

```js
// worker.js
Sentry.registerWebWorker({ self });

self.postMessage(...);
```

- **feat(core): Deprecate logger in favor of debug ([#17040](https://github.com/getsentry/sentry-javascript/pull/17040))**

The internal SDK `logger` export from `@sentry/core` has been deprecated in favor of the `debug` export. `debug` only exposes `log`, `warn`, and `error` methods but is otherwise identical to `logger`. Note that this deprecation does not affect the `logger` export from other packages (like `@sentry/browser` or `@sentry/node`) which is used for Sentry Logging.

```js
import { logger, debug } from '@sentry/core';

// before
logger.info('This is an info message');

// after
debug.log('This is an info message');
```

- **feat(node): Add OpenAI integration ([#17022](https://github.com/getsentry/sentry-javascript/pull/17022))**

This release adds official support for instrumenting OpenAI SDK calls in with Sentry tracing, following OpenTelemetry semantic conventions for Generative AI. It instruments:

- `client.chat.completions.create()` - For chat-based completions
- `client.responses.create()` - For the responses API

```js
// The integration respects your `sendDefaultPii` option, but you can override the behavior in the integration options

Sentry.init({
  dsn: '__DSN__',
  integrations: [
    Sentry.openAIIntegration({
      recordInputs: true, // Force recording prompts
      recordOutputs: true, // Force recording responses
    }),
  ],
});
```

### Other Changes

- feat(node-core): Expand `@opentelemetry/instrumentation` range to cover `0.203.0` ([#17043](https://github.com/getsentry/sentry-javascript/pull/17043))
- fix(cloudflare): Ensure errors get captured from durable objects ([#16838](https://github.com/getsentry/sentry-javascript/pull/16838))
- fix(sveltekit): Ensure server errors from streamed responses are sent ([#17044](https://github.com/getsentry/sentry-javascript/pull/17044))

Work in this release was contributed by @0xbad0c0d3 and @tommy-gilligan. Thank you for your contributions!
