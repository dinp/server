package g

import (
	"github.com/garyburd/redigo/redis"
	"time"
)

var RedisConnPool *redis.Pool

func InitRedisConnPool() {
	RedisConnPool = &redis.Pool{
		MaxIdle:     Config().Redis.MaxIdle,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", Config().Redis.Dsn)
		},
	}
}
