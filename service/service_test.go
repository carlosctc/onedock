package service

import (
	"flag"
	"testing"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/context"
	"github.com/davecgh/go-spew/spew"
)

var ctx context.IContext

func Init() {
	confPath := flag.String("config", "../config.toml", "configure file")
	flag.Parse()
	igo.App = igo.NewApp(*confPath)
	ctx = context.NewContext()
}

// AddUser
func TestGetContainerMapping(t *testing.T) {
	Init()
	s := NewService()
	list, err := s.GetContainerMapping(ctx, 9203)
	if err != nil {
		t.Error(err)
	}
	spew.Dump(list)
}
