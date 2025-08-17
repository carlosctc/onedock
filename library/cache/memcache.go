package cache

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aichy126/igo/context"
	"github.com/aichy126/onedock/utils"
	cache "github.com/patrickmn/go-cache"
)

// MemCache
type MemCache struct {
	mem *cache.Cache
}

// NewMemCache
func NewMemCache() *MemCache {
	Memcache := cache.New(5*time.Minute, 10*time.Minute)
	return &MemCache{
		mem: Memcache,
	}
}

func (s *MemCache) Get(ctx context.IContext, key string, value interface{}) error {
	item, has := s.mem.Get(key)
	if !has {
		return fmt.Errorf("information does not exist")
	}
	str, ok := item.(string)
	if !ok {
		return fmt.Errorf("format error")
	}
	json.Unmarshal([]byte(str), value)
	return nil
}

func (s *MemCache) Set(ctx context.IContext, key string, value interface{}, redisTime int) error {
	str, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.SetString(ctx, key, string(str), redisTime)
}

func (s *MemCache) GetString(ctx context.IContext, key string) (string, error) {
	value, has := s.mem.Get(key)
	if !has {
		return "", fmt.Errorf("information does not exist")
	}
	v, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("information abnormality")
	}
	return v, nil
}

func (s *MemCache) SetString(ctx context.IContext, key, value string, redisTime int) error {
	var nTTL time.Duration
	if redisTime > 0 {
		nTTL, _ = time.ParseDuration(fmt.Sprintf("%ds", redisTime))
	} else {
		nTTL = cache.NoExpiration
	}
	s.mem.Set(key, value, nTTL)
	return nil
}

func (s *MemCache) GetInt64(ctx context.IContext, key string) (int64, error) {
	value, has := s.mem.Get(key)
	if !has {
		return 0, fmt.Errorf("information does not exist")
	}
	v, ok := value.(string)
	if !ok {
		return 0, fmt.Errorf("information abnormality")
	}
	num := utils.StringToInt64(v)
	return num, nil
}

func (s *MemCache) SetInt64(ctx context.IContext, key string, value int64, redisTime int) error {
	var nTTL time.Duration
	if redisTime > 0 {
		nTTL, _ = time.ParseDuration(fmt.Sprintf("%ds", redisTime))
	} else {
		nTTL = cache.NoExpiration
	}
	str := fmt.Sprintf("%d", value)
	s.mem.Set(key, str, nTTL)
	return nil
}

func (s *MemCache) Del(ctx context.IContext, Key string) error {
	s.mem.Delete(Key)
	return nil
}
