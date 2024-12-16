package main

import (
	"context"

	"github.com/let-light/gomodule"
	"github.com/pingostack/livhub/apps/pms"
	"github.com/pingostack/livhub/apps/whip"
	"github.com/pingostack/livhub/internal/core"
	"github.com/pingostack/livhub/internal/rtc"
	"github.com/sirupsen/logrus"
)

func serv(ctx context.Context) {
	gomodule.RegisterDefaultModules()
	gomodule.RegisterWithName(whip.WhipModule(), "whip")
	gomodule.RegisterWithName(pms.PMSModule(), "pms")
	gomodule.RegisterWithName(core.CoreModule(), "core")
	gomodule.RegisterWithName(rtc.RtcModule(), "webrtc")
	gomodule.Launch(ctx)

	gomodule.Wait()
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("Error during: %v", err)
		}
	}()

	serv(context.Background())
}
