package cron

import (
	"fmt"
	"github.com/dinp/server/g"
	"log"
	"time"
)

var (
	Domains         = make(map[string]string)
	DomainsToUpdate = make(map[string]bool)
)

func SyncDomain() {
	duration := time.Duration(g.Config().Interval) * time.Second
	for {
		syncDomain()
		time.Sleep(duration)
	}
}

func syncDomain() {
	_sql := "select domain, app_name from domain where app_id <> 0"
	rows, err := g.DB.Query(_sql)
	if err != nil {
		log.Printf("[ERROR] exec %s fail: %s", _sql, err)
		return
	}

	needUpdateRedis := false

	for rows.Next() {
		var domain, appName string
		err = rows.Scan(&domain, &appName)
		if err != nil {
			log.Printf("[ERROR] %s scan fail: %s", _sql, err)
			return
		}

		name, existent := Domains[domain]
		if !existent || name != appName {
			Domains[domain] = appName
			DomainsToUpdate[domain] = true
			needUpdateRedis = true
		}
	}

	if !needUpdateRedis {
		return
	}

	rc := g.RedisConnPool.Get()
	defer rc.Close()

	err = rc.Send("MULTI")
	if err != nil {
		log.Printf("[ERROR] rc.Do(\"MULTI\") fail: %v", err)
		return
	}

	rsPrefix := g.Config().Redis.RsPrefix
	cnamePrefix := g.Config().Redis.CNamePrefix
	domain := g.Config().Domain
	debug := g.Config().Debug

	for d, toUp := range DomainsToUpdate {
		if !toUp {
			continue
		}

		uriKey := fmt.Sprintf("%s%s.%s", rsPrefix, Domains[d], domain)
		cname := fmt.Sprintf("%s%s", cnamePrefix, d)
		if debug {
			log.Printf("[Redis] SET %s %s", cname, uriKey)
		}
		rc.Send("SET", cname, uriKey)
		DomainsToUpdate[d] = false
	}

	_, err = rc.Do("EXEC")
	if err != nil {
		log.Printf("[ERROR] rc.Do(\"EXEC\") fail: %v", err)
	}

}
