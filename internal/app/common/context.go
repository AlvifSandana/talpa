package common

import "talpa/internal/infra/logging"

type contextKey string

const ContextKeyApp contextKey = "appctx"

type GlobalOptions struct {
	DryRun  bool
	Debug   bool
	Yes     bool
	JSON    bool
	NoOpLog bool
}

type AppContext struct {
	Options   GlobalOptions
	Whitelist []string
	Logger    logging.Logger
}
