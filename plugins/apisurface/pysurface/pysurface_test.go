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

// The class index: classes named as a base, with their bases and every name
// their bodies define; private modules and names included.
func TestReadClasses(t *testing.T) {
	c := ReadClasses([]File{
		{"acme/_resource.py", "class _Base:\n    def list(self): pass\n\nclass SyncAPIResource(_Base, Generic[T], metaclass=M):\n    _client: Client\n    create = _create\n    if TYPE_CHECKING:\n        def _hidden(self): pass\n    else:\n        pass\n    def _create(self, x):\n        y = 1\n        return y\n"},
		{"acme/completions.py", "from ._resource import SyncAPIResource\n\nclass Completions(base.SyncAPIResource):\n    pass\n"},
	})
	got, _ := json.Marshal(c)
	want := `{"SyncAPIResource":{"bases":["Generic","_Base"],"members":["_client","_create","_hidden","create"]},"_Base":{"members":["list"]}}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if b := Bases("class X(a.B[T], C, metaclass=M): / def __init__(self, a=(1, 2))"); strings.Join(b, ",") != "B,C" {
		t.Errorf("bases %v", b)
	}
}
