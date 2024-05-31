package pms_feature

import "github.com/let-light/gomodule"

type Feature interface {
	gomodule.IModule
}

func Type() interface{} {
	return (*Feature)(nil)
}
