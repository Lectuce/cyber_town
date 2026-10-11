package tool

import "cybertown/internal/agent/cron"

var CronManager *cron.Manager

func SetCronManager(m *cron.Manager) {
	CronManager = m
}
