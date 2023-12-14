package main

import (
	"context"

	"github.com/let-light/gomodule"
	"github.com/pingostack/neon/apps/pms"
	"github.com/pingostack/neon/apps/whip"
	"github.com/pingostack/neon/internal/core"
	"github.com/sirupsen/logrus"
)

func serv(ctx context.Context) {
	gomodule.RegisterDefaultModules()
	gomodule.RegisterWithName(whip.WhipModule(), "whip")
	gomodule.RegisterWithName(pms.PMSModule(), "pms")
	gomodule.RegisterWithName(core.CoreModule(), "core")
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
