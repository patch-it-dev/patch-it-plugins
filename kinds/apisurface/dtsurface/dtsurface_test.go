package dtsurface

import (
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/pysurface"
)

const pkgJSON = `{"name": "acme", "types": "./index.d.ts",
  "exports": {
    ".": {"import": {"types": "./index.d.mts", "default": "./index.mjs"}, "require": {"types": "./index.d.ts", "default": "./index.js"}},
    "./resources/*": {"import": "./resources/*.mjs", "require": "./resources/*.js"},
    "./resources/*.js": {"default": "./resources/*.js"},
    "./_shims/*": {"default": "./_shims/*.js"},
    "./index": {"default": "./index.js"},
    "./package.json": "./package.json"
  }}`

// index.d.mts is what the "." entry reads (the import condition first).
const indexDTS = `/** The client. */
export { Acme as default } from "./client.mjs";
export { Acme, type ClientOptions } from "./client.mjs";
export * from "./errors.mjs";
export * from "@acme/core";
export { helper } from "@acme/helpers";
`

const clientDTS = `import * as API from "./resources/index.mjs";
import { Chat } from "./resources/chat.mjs";
export interface ClientOptions { apiKey?: string; timeout?: number }
export declare class Acme {
    #private;
    constructor({ apiKey, ...opts }?: ClientOptions);
    chat: API.Chat;
    private secret;
    protected base(): void;
    /** @internal */
    debug(x: string): void;
    get baseURL(): string;
    set baseURL(v: string);
    static readonly VERSION: string;
    withOptions(options: Partial<ClientOptions>): this;
    request<T>(this: Acme, path: string, init?: { method?: "GET" | "POST"; body?: Record<string, Array<Array<number>>> }): Promise<T>;
    stream(path: string): AsyncIterable<Uint8Array>;
    stream(path: string, signal: AbortSignal): AsyncIterable<Uint8Array>;
}
export declare namespace Acme {
    export { Chat as Chat };
    export type RequestOptions = { retries?: number };
}
`

const chatDTS = `import { APIResource } from "../core/resource.mjs";
export declare class Chat extends APIResource {
    create(body: ChatCreateParams, options?: RequestOptions): Promise<ChatCompletion>;
    list(query?: ListParams | null, ...rest: unknown[]): Promise<ChatCompletion[]>;
}
export declare namespace Chat {
    export { type ChatCompletion as ChatCompletion };
}
export interface ChatCompletion { id: string }
export type ChatCreateParams = { model: string; messages: Array<{ role: string }> }
export declare function toFile(value: Blob | string, name?: string | null): Promise<File>
`

const errorsDTS = `export declare class AcmeError extends Error {}
export declare class APIError<TStatus extends number | undefined = number | undefined> extends AcmeError {
    readonly status: TStatus;
    constructor(status: TStatus, error: object | undefined, message: string | undefined);
}
export declare const NotFoundError: typeof APIError;
export declare enum Role { User = "user", Assistant = "assistant" }
declare function internalOnly(): void;
`

func readAcme(t *testing.T) pysurface.Surface {
	t.Helper()
	s, _, _ := Read("acme", []File{
		{"package.json", pkgJSON},
		{"index.d.mts", indexDTS},
		{"index.d.ts", "export {};\n"},
		{"client.d.mts", clientDTS},
		{"resources/chat.d.mts", chatDTS},
		{"resources/index.d.mts", `export { Chat, type ChatCompletion } from "./chat.mjs";` + "\n"},
		{"errors.d.mts", errorsDTS},
		{"core/resource.d.mts", "export declare class APIResource { protected _client: unknown }\n"},
		{"_shims/node.d.mts", "export declare function shim(): void;\n"},
	})
	return s
}

func keys(s pysurface.Surface) []string {
	var out []string
	for k := range s {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// Every entry point's exports and what its classes and namespaces hold;
// private and protected members, @internal ones, the setter, private
// entry points (_shims), types inside a namespace, and what another
// package exports wholesale are left out.
func TestRead(t *testing.T) {
	s := readAcme(t)
	want := []string{
		"acme.APIError", "acme.APIError.status", "acme.Acme", "acme.Acme.Chat", "acme.Acme.Chat.create", "acme.Acme.Chat.list",
		"acme.Acme.VERSION", "acme.Acme.baseURL", "acme.Acme.chat", "acme.Acme.request", "acme.Acme.stream", "acme.Acme.withOptions",
		"acme.AcmeError", "acme.ClientOptions", "acme.NotFoundError", "acme.Role", "acme.default", "acme.helper",
		"acme/resources/chat.Chat", "acme/resources/chat.Chat.create", "acme/resources/chat.Chat.list",
		"acme/resources/chat.ChatCompletion", "acme/resources/chat.ChatCreateParams", "acme/resources/chat.toFile",
	}
	if got := keys(s); !slices.Equal(got, want) {
		t.Errorf("got\n %s\nwant\n %s", strings.Join(got, "\n "), strings.Join(want, "\n "))
	}
	c := s["acme.Acme"]
	if c.Kind != "class" || c.File != "client.d.mts" || c.Line != 4 || len(c.Params) != 1 || !c.Params[0].HasDef {
		t.Errorf("Acme: %+v", c)
	}
	if r := s["acme.Acme.request"]; len(r.Params) != 2 || r.Params[0].Name != "path" || r.Params[0].HasDef || !r.Params[1].HasDef {
		t.Errorf("request (this left out, init optional): %+v", r)
	}
	if st := s["acme.Acme.stream"]; !st.Overloaded || st.Params != nil {
		t.Errorf("stream: %+v", st)
	}
	if l := s["acme.Acme.Chat.list"]; len(l.Params) != 2 || l.Params[1].Kind != "args" {
		t.Errorf("list: %+v", l)
	}
	if f := s["acme/resources/chat.toFile"]; f.Kind != "function" || len(f.Params) != 2 || !f.Params[1].HasDef || f.Line != 11 {
		t.Errorf("toFile (no ; at the end): %+v", f)
	}
	if h := s["acme.helper"]; h.Kind != "export" || !strings.Contains(h.Signature, `from "@acme/helpers"`) {
		t.Errorf("helper: %+v", h)
	}
	if e := s["acme.APIError"]; len(e.Params) != 3 || e.Signature != "class APIError<TStatus extends number | undefined = number | undefined> extends AcmeError / constructor(status: TStatus, error: object | undefined, message: string | undefined)" {
		t.Errorf("APIError: %+v", e)
	}
}

// What each entry point exports, Star when it re-exports another package
// wholesale; the classes with their bases.
func TestBindingsAndClasses(t *testing.T) {
	_, c, b := Read("acme", []File{
		{"package.json", pkgJSON},
		{"index.d.mts", indexDTS},
		{"client.d.mts", clientDTS},
		{"errors.d.mts", errorsDTS},
		{"resources/chat.d.mts", chatDTS},
		{"core/resource.d.mts", "export declare class APIResource { protected _client: unknown }\n"},
	})
	if m := b["acme"]; !m.Star || !slices.Contains(m.Names, "AcmeError") || !slices.Contains(m.Names, "default") {
		t.Errorf("acme binds %+v", m)
	}
	if ch := c["Chat"]; !slices.Equal(ch.Bases, []string{"APIResource"}) || !slices.Equal(ch.Members, []string{"create", "list"}) {
		t.Errorf("Chat: %+v", ch)
	}
}

// A declaration file of an older style: `export =` a namespace merged with
// a function, no package.json exports, members of an ambient namespace
// exported without the keyword.
func TestExportAssignment(t *testing.T) {
	s, _, _ := Read("old", []File{
		{"package.json", `{"name": "old", "typings": "types/index.d.ts"}`},
		{"types/index.d.ts", "declare function old(a: string): old.Result;\ndeclare namespace old {\n  interface Result { ok: boolean }\n  function parse(s: string, strict?: boolean): Result\n  class Parser { parse(s: string): Result }\n}\nexport = old;\n"},
	})
	want := []string{"old.Parser", "old.Parser.parse", "old.Result", "old.parse"}
	if got := keys(s); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// The tokenizer: comments, strings with escapes, template literal types,
// and ">>" closing two generics.
func TestLex(t *testing.T) {
	var got []string
	for _, tk := range lex("/** doc */ type A<T> = `x${\"}\"}y` | 'q\\'' // c\nArray<Array<T>>;") {
		got = append(got, tk.text)
	}
	want := []string{"type", "A", "<", "T", ">", "=", "`x${\"}\"}y`", "|", "q'", "Array", "<", "Array", "<", "T", ">", ">", ";", ""}
	if !slices.Equal(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// A release is data: a cut-off or malformed file is read as far as it goes,
// never in a loop and never a panic.
func TestATruncatedFileEnds(t *testing.T) {
	src := clientDTS + chatDTS + errorsDTS
	for i := 0; i < len(src); i += 7 {
		Read("acme", []File{{"package.json", `{"name": "acme", "types": "index.d.ts"}`}, {"index.d.ts", src[:i]}})
	}
	Read("acme", []File{{"index.d.ts", "export { a as } from; class { constructor( ; namespace . { export * as from"}})
}
