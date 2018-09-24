package core

import (
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"strings"
)

type Bridge struct {
	listener           MessageBusClient
	publisher          MessageBusClient
	namespaceListener  string
	namespacePublisher string
	subscriptions      []string
	logger             Logger
}

func NewBridge(listener, publisher MessageBusClient, namespaceListener, namespacePublisher string, subscriptions []string, logger Logger) *Bridge {
	return &Bridge{
		listener:           listener,
		publisher:          publisher,
		namespaceListener:  namespaceListener,
		namespacePublisher: namespacePublisher,
		subscriptions:      subscriptions,
		logger:             logger,
	}
}

func (b *Bridge) Start() (<-chan struct{}, error) {
	err := b.listener.Connect()
	if err != nil {
		return nil, fmt.Errorf("can't connect: %s", err)
	}

	if b.publisher != b.listener {
		err := b.publisher.Connect()
		if err != nil {
			return nil, fmt.Errorf("can't connect: %s", err)
		}
	}

	inputMessages, err := b.listener.Subscribe(b.subscriptions)
	if err != nil {
		return nil, fmt.Errorf("can't subscribe: %s", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		for inpMsg := range inputMessages {
			inpTopic := gjson.Get(inpMsg, "topic").String()
			topic, msg := inpTopic, inpMsg
			if b.namespacePublisher != b.namespaceListener {
				topic = strings.Replace(inpTopic, b.namespaceListener+"/", b.namespacePublisher+"/", 1)
				msg, _ = sjson.Set(inpMsg, "topic", topic)
			}

			err := b.publisher.Publish(topic, msg)
			if err != nil {
				b.logger.Error("can't publish", err)
			}
		}
	}()

	return done, nil
}
