package pysurface

import (
	"encoding/json"
	"strings"
	"testing"
)

const initPy = `"""Acme."""
from ._client import Client, AsyncClient as Async
from .version import __version__
from . import types
from .helpers import *
from os import path

__all__ = ["Client", "Async", "helper", "VERSION"]
VERSION = "1"
`

const clientPy = `import typing
from typing import overload

class Client(Base):
    """A client.

    # not a comment
    """

    def __init__(self, api_key: str | None = None, *, timeout: float = 600.0, **kwargs) -> None:
        self._x = "a # b"

    def create(
        self,
        model: str,  # the model
        /,
        n: int = 1,
        *,
        stream: bool = False,
        extra=("a", "b,c"),
    ) -> "Result":
        def inner(z):
            pass
        return None

    @property
    def models(self) -> int:
        return 1

    def _private(self, x):
        pass

    @overload
    def get(self, a: int) -> int: ...
    @overload
    def get(self, a: str) -> str: ...
    def get(self, a):
        return a

    class Nested:
        def deep(self, q): pass

if typing.TYPE_CHECKING:
    class AsyncClient: pass
else:
    class AsyncClient(Client): \
        pass

def _hidden(): pass
`

const helpersPy = `__all__ = ["helper"]

async def helper(x, *args, y=None, **kw):
    pass

def not_exported(): pass
`

func TestRead(t *testing.T) {
	s := Read([]File{
		{"acme/__init__.py", initPy},
		{"acme/_client.py", clientPy},
		{"acme/helpers.py", helpersPy},
		{"acme/tests/test_x.py", "def test_a(): pass\n"},
	})
	var keys []string
	for k := range s {
		keys = append(keys, k)
	}
	want := map[string]bool{
		"acme.Client": true, "acme.Client.create": true, "acme.Client.models": true, "acme.Client.get": true,
		"acme.Async": true, "acme.helper": true,
		"acme.helpers.helper": true,
	}
	if len(s) != len(want) {
		b, _ := json.MarshalIndent(s, "", " ")
		t.Fatalf("got %d symbols %v, want %v\n%s", len(s), keys, want, b)
	}
	for k := range want {
		if _, ok := s[k]; !ok {
			t.Errorf("missing %s (have %v)", k, keys)
		}
	}
	c := s["acme.Client"]
	if c.File != "acme/_client.py" || c.Line != 4 || len(c.Params) != 4 || c.Params[1].Name != "api_key" || c.Params[1].Default != "None" || c.Params[2].Kind != "keyword" || c.Params[3].Kind != "kwargs" {
		t.Errorf("Client: %+v", c)
	}
	cr := s["acme.Client.create"]
	if cr.Line != 13 || len(cr.Params) != 5 || cr.Params[1].Kind != "positional" || cr.Params[2].Kind != "either" || cr.Params[3].Kind != "keyword" || cr.Params[4].Default != `("a", "b,c")` {
		t.Errorf("create: %+v", cr)
	}
	if g := s["acme.Client.get"]; !g.Overloaded || g.Params != nil || g.Line != 37 {
		t.Errorf("get: %+v", g)
	}
	if h := s["acme.helper"]; h.Params[1].Kind != "args" || h.Params[2].Kind != "keyword" {
		t.Errorf("helper: %+v", h)
	}
}

// The class index: every module-level class, with its bases and every name
// its body defines; private modules and names included.
func TestReadClasses(t *testing.T) {
	c := ReadClasses([]File{
		{"acme/_resource.py", "class _Base:\n    def list(self): pass\n\nclass SyncAPIResource(_Base, Generic[T], metaclass=M):\n    _client: Client\n    create = _create\n    if TYPE_CHECKING:\n        def _hidden(self): pass\n    else:\n        pass\n    def _create(self, x):\n        y = 1\n        return y\n"},
		{"acme/completions.py", "from ._resource import SyncAPIResource\n\nclass Completions(base.SyncAPIResource):\n    models = _models\n"},
	})
	got, _ := json.Marshal(c)
	want := `{"Completions":{"bases":["SyncAPIResource"],"members":["models"]},"SyncAPIResource":{"bases":["Generic","_Base"],"members":["_client","_create","_hidden","create"]},"_Base":{"members":["list"]}}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if b := Bases("class X(a.B[T], C, metaclass=M): / def __init__(self, a=(1, 2))"); strings.Join(b, ",") != "B,C" {
		t.Errorf("bases %v", b)
	}
}

// A property's setter isn't a method of its own: it doesn't replace the
// getter, and a property's parameters (self, then the value a setter takes)
// aren't compared: callers never pass them.
func TestAPropertyHasNoParametersToCompare(t *testing.T) {
	s := Read([]File{{"acme/span.py", "class Span:\n    @property\n    def name(self) -> str:\n        return self._name\n\n    @name.setter\n    def name(self, value: str) -> None:\n        self._name = value\n\n    @functools.cached_property\n    def size(self):\n        return 1\n"}})
	n, ok := s["acme.span.Span.name"]
	if !ok || n.Params != nil || n.Line != 3 {
		t.Errorf("name: %+v", n)
	}
	if sz := s["acme.span.Span.size"]; sz.Params != nil {
		t.Errorf("size: %+v", sz)
	}
}

// A module that star-imports another module of the package re-exports its
// public names, __init__ or not: `from . import *` makes them importable.
func TestAStarImportReExportsAnywhere(t *testing.T) {
	s := Read([]File{
		{"acme/__init__.py", "from ._sync.client import Client\n"},
		{"acme/_sync/client.py", "class Client:\n    def __init__(self, url):\n        pass\n"},
		{"acme/client.py", "import acme as _package\n\nfrom . import *\n\n__all__ = _package.__all__ + (\"Extra\",)\n"},
	})
	if _, ok := s["acme.client.Client"]; !ok {
		t.Errorf("acme.client.Client missing: %v", s)
	}
}

// The names each public module binds at its top level, whatever they are:
// what lets a name that is no longer a class or function the package
// defines (now an alias, or imported from elsewhere) read as still there.
func TestReadBindings(t *testing.T) {
	b := ReadBindings([]File{
		{"acme/__init__.py", "from ._exceptions import GatewayTimeoutError, DeadlineExceededError\n"},
		{"acme/_exceptions.py", "class GatewayTimeoutError(Exception): pass\nDeadlineExceededError = GatewayTimeoutError\n"},
		{"acme/interactions.py", "import typing as t\nimport os\nfrom ._gen.types import *\nfrom ._gen.webhooks import Webhook, WebhookInput as WebhookCreateParams\nif t.TYPE_CHECKING:\n    Alias: t.TypeAlias = int\ndef f(): pass\nclass C:\n    inner = 1\n"},
	})
	got, _ := json.Marshal(b)
	want := `{"acme":{"names":["DeadlineExceededError","GatewayTimeoutError"]},"acme.interactions":{"names":["Alias","C","Webhook","WebhookCreateParams","f","os","t"],"star":true}}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// A module under a package named internal is the package's own business, as
// one starting with "_" is (ddtrace.internal, ddtrace.contrib.internal), and
// so is vendored code (ddtrace.vendor).
func TestAnInternalModuleIsPrivate(t *testing.T) {
	s := Read([]File{
		{"acme/internal/util.py", "def helper(): pass\n"},
		{"acme/vendor/jsonpath/__init__.py", "class Child: pass\n"},
		{"acme/contrib/internal/grpc.py", "def patch(): pass\n"},
		{"acme/contrib/grpc.py", "def patch(): pass\n"},
	})
	if len(s) != 1 {
		t.Errorf("got %v, want only acme.contrib.grpc.patch", s)
	}
	if _, ok := s["acme.contrib.grpc.patch"]; !ok {
		t.Errorf("missing acme.contrib.grpc.patch: %v", s)
	}
}

// A module with a module-level __getattr__ (PEP 562) can hand out any name,
// lazily: ddtrace 4.13.0's appsec.ai_guard forwards its old names to their
// new home that way. Its names can't be listed: Star.
func TestAModuleGetattrBindsAnyName(t *testing.T) {
	b := ReadBindings([]File{{"acme/old.py", "import typing\n\n_PUBLIC = frozenset({\"Client\"})\n\ndef __getattr__(name: str) -> typing.Any:\n    return 1\n"}})
	if !b["acme.old"].Star {
		t.Errorf("got %+v, want Star", b["acme.old"])
	}
}

// A package's __init__ re-exports what it imports from its own modules;
// what it imports from elsewhere in the package (a logger from
// ddtrace.internal into ddtrace.appsec.ai_guard.integrations) is its own
// business, as in any other module, unless __all__ names it.
func TestAnInitReExportsOnlyFromItsOwnModules(t *testing.T) {
	s := Read([]File{
		{"acme/__init__.py", ""},
		{"acme/_log.py", "def get_logger(): pass\n"},
		{"acme/sub/__init__.py", "from .._log import get_logger\nfrom ._impl import Client\nfrom acme.sub._impl import Other\n"},
		{"acme/sub/_impl.py", "class Client: pass\nclass Other: pass\n"},
		{"acme/listed/__init__.py", "from .._log import get_logger\n__all__ = [\"get_logger\"]\n"},
	})
	for _, k := range []string{"acme.sub.Client", "acme.sub.Other", "acme.listed.get_logger"} {
		if _, ok := s[k]; !ok {
			t.Errorf("missing %s", k)
		}
	}
	if _, ok := s["acme.sub.get_logger"]; ok {
		t.Errorf("acme.sub.get_logger is the module's own import, not a re-export")
	}
}
