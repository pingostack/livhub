package core

import (
	"context"

	"github.com/let-light/gomodule"
	"github.com/pingostack/neon/internal/httpserv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var coreModule *core

type CoreSettings struct {
	httpserv.HttpParams `json:"http" mapstructure:"http"`
}

type core struct {
	gomodule.DefaultModule
	ctx         context.Context
	preSettings CoreSettings
	settings    *CoreSettings
	logger      *logrus.Entry
}

func init() {
	coreModule = &core{
		logger: logrus.WithField("module", "core"),
	}
}

func CoreModule() *core {
	return coreModule
}

func (core *core) InitModule(ctx context.Context, _ *gomodule.Manager) (interface{}, error) {
	core.ctx = ctx
	return &core.preSettings, nil
}

func (core *core) InitCommand() ([]*cobra.Command, error) {

	return nil, nil
}

func (core *core) ConfigChanged() {
	if core.settings == nil {
		core.settings = &core.preSettings
	}
}

func (core *core) ModuleRun() {
	core.close()
}

func (core *core) close() {
	core.logger.Info("core closing")
}
