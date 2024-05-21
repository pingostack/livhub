package pms

import (
	"context"
	"time"

	"github.com/let-light/gomodule"
	"github.com/pingostack/neon/internal/httpserv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var pmsModule *pms

type ISignalServer interface {
	Start() error
	Close() error
}

type WebRTCConfig struct {
	KeyFrameInterval time.Time `json:"keyFrameInterval" mapstructure:"keyFrameInterval"`
}

type PMSSettings struct {
	httpserv.HttpParams `json:"http" mapstructure:"http"`
}

type pms struct {
	gomodule.DefaultModule
	ctx         context.Context
	preSettings PMSSettings
	settings    *PMSSettings
	logger      *logrus.Entry
	serv        ISignalServer
}

func init() {
	pmsModule = &pms{
		logger: logrus.WithField("module", "pms"),
	}
}

func PMSModule() *pms {
	return pmsModule
}

func (pms *pms) InitModule(ctx context.Context, _ *gomodule.Manager) (interface{}, error) {
	pms.ctx = ctx
	return &pms.preSettings, nil
}

func (pms *pms) InitCommand() ([]*cobra.Command, error) {

	return nil, nil
}

func (pms *pms) ConfigChanged() {
	if pms.settings == nil {
		pms.settings = &pms.preSettings
	}
}

func (pms *pms) ModuleRun() {
	pms.serv = NewSignalServer(pms.ctx, pms.settings.HttpParams, pms.logger)
	if err := pms.serv.Start(); err != nil {
		pms.logger.Errorf("pms start error: %v", err)
		panic(errors.Wrap(err, "pms start error"))
	}

	<-pms.ctx.Done()
	pms.close()
}

func (pms *pms) close() {
	pms.logger.Info("pms closing")
	pms.serv.Close()
}
