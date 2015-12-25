// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_redis.go

package xmandb

import (
	"errors"
	"time"

	"github.com/garyburd/redigo/redis"
)

var (
	DefaultKey string = "XMANRedis"
)

var (
	ErrNoConnKey = errors.New("redis has no conn key.")
)

type RedisController struct {
	p        *redis.Pool
	key      string
	conninfo string
	dbNum    int
}

func NewRedisController() *RedisController {
	return &RedisController{key: DefaultKey}
}

func (rc *RedisController) InitRedis(key, conninfo string, dbNum int) error {
	if conninfo == "" {
		return ErrNoConnKey
	}
	if key == "" {
		key = DefaultKey
	}
	rc.key = key
	rc.conninfo = conninfo
	rc.dbNum = dbNum
	rc.p = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 180 * time.Second,
		Dial: func() (c redis.Conn, err error) {
			c, err = redis.Dial("tcp", rc.conninfo)
			_, selectErr := c.Do("SELECT", rc.dbNum)
			if selectErr != nil {
				c.Close()
				return nil, selectErr
			}
			return
		},
	}

	c := rc.p.Get()
	defer c.Close()

	return c.Err()
}

func (rc *RedisController) Close() {
	if rc.p != nil {
		rc.p.Close()
	}
}

func (rc *RedisController) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	c := rc.p.Get()
	defer c.Close()

	return c.Do(commandName, args...)
}

// Get cache from redis.
func (rc *RedisController) Get(key string) interface{} {
	if v, err := rc.do("GET", key); err == nil {
		return v
	}
	return nil
}

// put cache to redis.
func (rc *RedisController) Put(key string, val interface{}, timeout int64) error {
	var err error
	if _, err = rc.do("SETEX", key, timeout, val); err != nil {
		return err
	}

	if _, err = rc.do("HSET", rc.key, key, true); err != nil {
		return err
	}
	return err
}

// delete cache in redis.
func (rc *RedisController) Delete(key string) error {
	var err error
	if _, err = rc.do("DEL", key); err != nil {
		return err
	}
	_, err = rc.do("HDEL", rc.key, key)
	return err
}

// check cache's existence in redis.
func (rc *RedisController) IsExist(key string) bool {
	v, err := redis.Bool(rc.do("EXISTS", key))
	if err != nil {
		return false
	}
	if v == false {
		if _, err = rc.do("HDEL", rc.key, key); err != nil {
			return false
		}
	}
	return v
}

// increase counter in redis.
func (rc *RedisController) Incr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, 1))
	return err
}

// decrease counter in redis.
func (rc *RedisController) Decr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, -1))
	return err
}

// clean all cache in redis. delete this redis collection.
func (rc *RedisController) ClearAll() error {
	cachedKeys, err := redis.Strings(rc.do("HKEYS", rc.key))
	if err != nil {
		return err
	}
	for _, str := range cachedKeys {
		if _, err = rc.do("DEL", str); err != nil {
			return err
		}
	}
	_, err = rc.do("DEL", rc.key)
	return err
}
