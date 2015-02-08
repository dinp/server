package cron

import (
	"fmt"
	"github.com/dinp/common/model"
	"github.com/dinp/server/g"
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
)

func SyncRoutes() {
	duration := time.Duration(g.Config().Interval) * time.Second
	for {
		syncRoutes()
		time.Sleep(duration)
	}
}

func syncRoutes() {
	realNames := g.RealState.Keys()

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	err := rc.Send("MULTI")
	if err != nil {
		log.Printf("[ERROR] rc.Do(\"MULTI\") fail: %v", err)
		return
	}

	for _, name := range realNames {
		sa, _ := g.RealState.GetSafeApp(name)
		if !sa.IsNeedUpdateRouter() {
			continue
		}
		_syncOneApp(rc, name, sa)
	}

	_, err = rc.Do("EXEC")
	if err != nil {
		log.Printf("[ERROR] rc.Do(\"EXEC\") fail: %v", err)
	}
}

func _syncOneApp(rc redis.Conn, appName string, app *model.SafeApp) error {
	uriKey := fmt.Sprintf("%s%s.%s", g.Config().Redis.RsPrefix, appName, g.Config().Domain)

	debug := g.Config().Debug
	if debug {
		log.Printf("[Redis] DEL %s", uriKey)
	}

	err := rc.Send("DEL", uriKey)
	if err != nil {
		log.Printf("[ERROR] DEL %s fail: %v", uriKey, err)
		return err
	}

	cs := app.Containers()
	if len(cs) == 0 {
		app.NeedUpdateRouter(false)
		return nil
	}

	args := []interface{}{uriKey}
	for _, c := range cs {
		args = append(args, fmt.Sprintf("%s:%d", c.Ip, c.Ports[0].PublicPort))
	}

	if debug {
		log.Printf("[Redis] LPUSH %v", args)
	}

	err = rc.Send("LPUSH", args...)
	if err != nil {
		log.Printf("[ERROR] LPUSH %v fail: %v", args, err)
	} else {
		app.NeedUpdateRouter(false)
	}

	return err
}
