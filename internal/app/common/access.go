package common

import (
	"errors"

	"github.com/spf13/cobra"
)

func FromCommand(cmd *cobra.Command) (*AppContext, error) {
	v := cmd.Context().Value(ContextKeyApp)
	app, ok := v.(*AppContext)
	if !ok || app == nil {
		return nil, errors.New("application context is not initialized")
	}
	return app, nil
}
