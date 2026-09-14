package tasks

import (
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/m-butterfield/mattbutterfield.com/app/data"
	"net"
)

var (
	ds data.Store
)

func Run(port string) error {
	vips.LoggingSettings(nil, vips.LogLevelWarning)
	if err := vips.Startup(nil); err != nil {
		return err
	}
	defer vips.Shutdown()

	var err error
	if ds, err = data.Connect(); err != nil {
		return err
	}
	r, err := router()
	if err != nil {
		return err
	}
	return r.Run(net.JoinHostPort("", port))
}
