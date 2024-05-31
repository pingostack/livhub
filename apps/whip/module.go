package whip

import (
	"context"

	"github.com/let-light/gomodule"
	feature_whip "github.com/pingostack/neon/features/whip"
	"github.com/pingostack/neon/internal/httpserv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var whipModule *whip

type ISignalServer interface {
	Start() error
	Close() error
}

type WhipSettings struct {
	httpserv.HttpParams `json:"http" mapstructure:"http"`
}

type whip struct {
	gomodule.DefaultModule
	ctx         context.Context
	preSettings WhipSettings
	settings    *WhipSettings
	logger      *logrus.Entry
	serv        ISignalServer
}

func init() {
	whipModule = &whip{
		logger: logrus.WithField("module", "whip"),
	}
}

func WhipModule() *whip {
	return whipModule
}

func (whip *whip) InitModule(ctx context.Context, _ *gomodule.Manager) (interface{}, error) {
	whip.ctx = ctx
	return &whip.preSettings, nil
}

func (whip *whip) InitCommand() ([]*cobra.Command, error) {

	return nil, nil
}

func (whip *whip) ConfigChanged() {
	if whip.settings == nil {
		whip.settings = &whip.preSettings
	}
}

func (whip *whip) ModuleRun() {
	whip.serv = NewSignalServer(whip.ctx, whip.settings.HttpParams, whip.logger)
	if err := whip.serv.Start(); err != nil {
		whip.logger.Errorf("whip start error: %v", err)
		return
	}

	<-whip.ctx.Done()
	whip.close()
}

func (whip *whip) Type() interface{} {
	return feature_whip.Type()
}

func (whip *whip) close() {
	whip.logger.Info("whip closing")
	whip.serv.Close()
}
