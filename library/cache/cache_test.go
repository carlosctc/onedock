package cache

import (
	"flag"
	"testing"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/context"
	"github.com/davecgh/go-spew/spew"
)

var ctx context.IContext

func Init() {
	confPath := flag.String("config", "../../config.toml", "configure file")
	flag.Parse()

	igo.App = igo.NewApp(*confPath)
	ctx = context.NewContext()
}

type Activity struct {
	ID     string `xorm:"not null pk autoincr BIGINT(20) id"`
	UserID string `xorm:"not null index BIGINT(20) user_id"`
}

func TestMemGet(t *testing.T) {
	Init()
	cache := NewMemCache()
	key := "memtest"
	Data := new(Activity)
	Data.ID = "12345"
	err := cache.Set(ctx, key, Data, 60)
	spew.Dump("===Set===", key, Data, err)

	getData := new(Activity)
	err = cache.Get(ctx, key, getData)
	spew.Dump("===Get===", key, getData, err)

}

func TestRedisGet(t *testing.T) {
	Init()
	cache := NewRedisCache()
	key := "memtest"
	Data := new(Activity)
	Data.ID = "12345"
	err := cache.Set(ctx, key, Data, 60)
	spew.Dump("===Set===", key, Data, err)
	getData := new(Activity)
	err = cache.Get(ctx, key, getData)
	spew.Dump("===Get===", key, getData, err)

}
