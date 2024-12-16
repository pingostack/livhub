package core

import (
	"context"

	"github.com/let-light/gomodule"
	feature_core "github.com/pingostack/livhub/features/core"
	"github.com/pingostack/livhub/internal/core/router"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var coreModule *core

// type NamespaceInfo struct {
// 	Name    string   `json:"name" mapstructure:"name"`
// 	Domains []string `json:"domain" mapstructure:"domain"`
// }

type CoreSettings struct {
	//httpserv.HttpParams `json:"http" mapstructure:"http"`
	Namespaces router.NSManagerParams `json:"namespaces" mapstructure:"namespaces"`
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
	defaultServ = NewServ(core.ctx, core.settings.Namespaces)
}

func (core *core) Type() interface{} {
	return feature_core.Type()
}
