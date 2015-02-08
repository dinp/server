package cron

import (
	"github.com/dinp/server/g"
	"time"
)

func CheckStale() {
	duration := time.Duration(g.Config().Interval) * time.Second
	for {
		time.Sleep(duration)
		checkStale()
	}
}

func checkStale() {
	now := time.Now().Unix()
	before := now - 3*int64(g.Config().Interval)
	g.DeleteStaleNode(before)
	g.RealState.DeleteStale(before)
}
