package cache

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/cache"
	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
)

// RedisCache
type RedisCache struct {
	Redis *cache.Redis
}

// NewRedisCache
func NewRedisCache() *RedisCache {
	redis, err := igo.App.Cache.Get("default")
	if err != nil {
		panic(fmt.Errorf("redis get error:%s", err.Error()))
	}
	return &RedisCache{
		Redis: redis,
	}
}

func (R *RedisCache) Colation(ctx context.IContext, Colation string, redisTimeBySecond int) bool {
	_, err := R.Redis.Get(ctx, Colation).Result()
	if err != nil {
		//放行
		nTTL, _ := time.ParseDuration(fmt.Sprintf("%ds", redisTimeBySecond))
		R.Redis.Set(ctx, Colation, true, nTTL)
		return true
	}
	//拦截
	return false
}

func (R *RedisCache) KeyAdd(ctx context.IContext, rediskey string, redisTime int) error {
	_, err := R.Redis.Incr(ctx, rediskey).Result()
	nTTL, _ := time.ParseDuration(fmt.Sprintf("%dh", redisTime))
	R.Redis.Expire(ctx, rediskey, nTTL)
	return err
}

// SetString 秒级缓存
func (R *RedisCache) SetString(ctx context.IContext, rediskey string, redisvalue interface{}, redisTime int) (string, error) {
	return R.Redis.Set(ctx, rediskey, redisvalue, R.redisTime(redisTime)).Result()
}

func (R *RedisCache) GetString(ctx context.IContext, rediskey string) (string, error) {
	return R.Redis.Get(ctx, rediskey).Result()
}

func (R *RedisCache) Set(ctx context.IContext, rediskey string, redisvalue interface{}, redisTime int) error {
	str, err := json.Marshal(redisvalue)
	if err != nil {
		return err
	}
	_, err = R.Redis.Set(ctx, rediskey, string(str), R.redisTime(redisTime)).Result()
	return err
}

func (R *RedisCache) Get(ctx context.IContext, rediskey string, redisvalue interface{}) error {
	str, err := R.Redis.Get(ctx, rediskey).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(str), redisvalue)
}

func (R *RedisCache) Del(ctx context.IContext, rediskey string) (int64, error) {
	return R.Redis.Del(ctx, rediskey).Result()
}

func (R *RedisCache) redisTime(redisTime int) time.Duration {
	var nTTL time.Duration
	if redisTime > 0 {
		nTTL, _ = time.ParseDuration(fmt.Sprintf("%ds", redisTime))
	} else if redisTime < 0 {
		nTTL = time.Duration(0)

	} else {
		nTTL, _ = time.ParseDuration(fmt.Sprintf("%ds", 360))
	}
	return nTTL
}

func (R *RedisCache) SetINCRBYBySecond(ctx context.IContext, rediskey string, setNum int64, redisTime int) (int64, error) {
	var nTTL time.Duration
	if redisTime > 0 {
		nTTL, _ = time.ParseDuration(fmt.Sprintf("%ds", redisTime))
	} else {
		nTTL, _ = time.ParseDuration(fmt.Sprintf("%ds", 360))
	}
	num, err := R.Redis.IncrBy(ctx, rediskey, setNum).Result()
	if err != nil {
		return 0, err
	}
	R.Redis.Expire(ctx, rediskey, nTTL)
	return num, nil
}

// HSet 写入hash 增加有效期 小时为单位 redisTime为0 则不设置有效期
func (R *RedisCache) HSet(ctx context.IContext, rediskey string, redisfield string, redisvalue interface{}, redisTime int) {
	_, err := R.Redis.HSet(ctx, rediskey, redisfield, redisvalue).Result()
	if err != nil {
		log.Error("Redis", log.Any("Error", err))
	}
	//有效期
	if redisTime > 0 {
		ttl := R.Redis.TTL(ctx, rediskey)
		nTTL, _ := time.ParseDuration(fmt.Sprintf("%dh", redisTime))
		if ttl.Val() < nTTL {
			R.Redis.Expire(ctx, rediskey, nTTL)
		}
	}
}

// Llen 列队长度
func (R *RedisCache) Llen(ctx context.IContext, key string) (int64, error) {
	num, err := R.Redis.LLen(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return num, nil
}

// Lpush
func (R *RedisCache) Lpush(ctx context.IContext, key string, v interface{}) (int64, error) {
	vstr, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}
	num, err := R.Redis.LPush(ctx, key, string(vstr)).Result()
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (R *RedisCache) LpushString(ctx context.IContext, key string, v string) (int64, error) {
	num, err := R.Redis.LPush(ctx, key, v).Result()
	if err != nil {
		return 0, err
	}
	return num, nil
}

// Rpop
func (R *RedisCache) Rpop(ctx context.IContext, key string, v interface{}) error {
	vstr, err := R.Redis.RPop(ctx, key).Result()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(vstr), v)
	if err != nil {
		return err
	}
	return nil
}

func (R *RedisCache) RpopString(ctx context.IContext, key string) (string, error) {
	vstr, err := R.Redis.RPop(ctx, key).Result()
	if err != nil {
		return vstr, err
	}

	return vstr, nil
}

func (R *RedisCache) HGetAll(ctx context.IContext, key string) (map[string]string, error) {
	data, err := R.Redis.HGetAll(ctx, key).Result()
	if err != nil {
		return data, err
	}

	return data, nil
}

// HashAdd 哈希计数器
func (R *RedisCache) HashAdd(ctx context.IContext, key string, field string, incr int64, redisTime int) error {
	_, err := R.Redis.HIncrBy(ctx, key, field, incr).Result()
	if err != nil {
		return err
	}
	if redisTime > 0 {
		ttl := R.Redis.TTL(ctx, key)
		nTTL, _ := time.ParseDuration(fmt.Sprintf("%dh", redisTime))
		if ttl.Val() < nTTL {
			R.Redis.Expire(ctx, key, nTTL)
		}
	}
	return nil
}
