package main

import (
	"fmt"
	"gitlab.com/flaneurtv/samm/core"
	"gitlab.com/flaneurtv/samm/core/env"
	"gitlab.com/flaneurtv/samm/core/logger"
	"gitlab.com/flaneurtv/samm/core/mqtt"
	"os"
)

func main() {
	log := logger.NewMQTTLogger(os.Stdout, os.Stderr)

	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				log.Log(core.LogLevelError, fmt.Sprintf("uncaughtException: %s", err))
			} else {
				log.Log(core.LogLevelError, fmt.Sprintf("uncaughtException: %s", r))
			}
		}
	}()

	cfg, err := env.NewBridgeConfig(log)
	if err != nil {
		log.Log(core.LogLevelCritical, fmt.Sprintf("can't create config: %s", err))
		os.Exit(1)
	}

	logLevelConsole, _ := core.ParseLogLevel(cfg.LogLevelConsole())
	logLevelRemote, _ := core.ParseLogLevel(cfg.LogLevelRemote())
	log.SetLevels(logLevelConsole, logLevelRemote)

	listenerClientID := fmt.Sprintf("%s_%s_%s_listener", cfg.ServiceName(), cfg.ServiceHost(), cfg.ServiceUUID())
	listener := mqtt.NewMQTTClient(cfg.ListenerURL(), listenerClientID, cfg.ListenerCredentials(), log, func(err error) {
		os.Exit(1)
	})

	var publisher core.MessageBusClient
	if cfg.ListenerURL() != cfg.PublisherURL() || cfg.ListenerCredentials() != cfg.PublisherCredentials() {
		publisherClientID := fmt.Sprintf("%s_%s_%s_publisher", cfg.ServiceName(), cfg.ServiceHost(), cfg.ServiceUUID())
		publisher = mqtt.NewMQTTClient(cfg.PublisherURL(), publisherClientID, cfg.PublisherCredentials(), log, func(err error) {
			os.Exit(1)
		})
	} else {
		publisher = listener
	}

	log.SetClient(publisher, cfg.NamespacePublisher(), cfg.ServiceName(), cfg.ServiceUUID(), cfg.ServiceHost())

	bridge := core.NewBridge(listener, publisher, cfg.NamespaceListener(), cfg.NamespacePublisher(), cfg.Subscriptions(), log)
	done, err := bridge.Start()
	if err != nil {
		log.Log(core.LogLevelCritical, fmt.Sprintf("can't start bridge: %s", err))
		os.Exit(1)
	}

	<-done
}
