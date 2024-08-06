package args

import (
	"fmt"

	"github.com/hertz-contrib/swagger-generate/thrift-gen-rpc-swagger/utils"
)

type Arguments struct {
	OutputDir string
	HostPort  string
}

func (a *Arguments) Unpack(args []string) error {
	err := utils.UnpackArgs(args, a)
	if err != nil {
		return fmt.Errorf("unpack argument failed: %s", err)
	}
	return nil
}
