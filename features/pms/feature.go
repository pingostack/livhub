package pms_feature

import "github.com/im-pingo/gomodule"

type Feature interface {
	gomodule.IModule
}

func Type() interface{} {
	return (*Feature)(nil)
}
