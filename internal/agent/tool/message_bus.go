package tool

import "cybertown/internal/agent/teams"

var MessageBus *teams.MessageBus

func SetMessageBus(bus *teams.MessageBus) {
	MessageBus = bus
}
