package xerr

import "github.com/joomcode/errorx"

var (
	Namespace = errorx.NewNamespace("agent-coder")
	Config    = Namespace.NewType("config")
	Infra     = Namespace.NewType("infra")
	Startup   = Namespace.NewType("startup")
)
