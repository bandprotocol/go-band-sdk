package subscriber

import (
	"fmt"
	"sync"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/patrickmn/go-cache"
)

type Subscriber struct {
	client client.Client
	logger logging.Logger

	cache      *cache.Cache
	eventInfos sync.Map
	conf       Config
}

// NewSubscriber creates a new subscriber object.
func NewSubscriber(client client.Client, logger logging.Logger, conf Config) *Subscriber {
	cache := cache.New(conf.ExpirationDuration, conf.CleanupInterval)

	return &Subscriber{
		client: client,
		logger: logger,
		cache:  cache,
		conf:   conf,
	}
}

// Subscribe subscribes events from client chain with the given query.
func (s *Subscriber) Subscribe(name, query string, getUIDFromEventFn func(e ctypes.ResultEvent) (string, error)) (*EventInfo, error) {
	if _, found := s.eventInfos.Load(name); found {
		return nil, fmt.Errorf("event already subscribed; name %s", name)
	}

	sc, err := s.client.Subscribe(name, query)
	if err != nil {
		return nil, err
	}

	stopChs := make([]chan struct{}, 0, len(sc.ChannelInfos))
	aggCh := make(chan ctypes.ResultEvent, len(sc.ChannelInfos))

	for _, info := range sc.ChannelInfos {
		stopCh := make(chan struct{})
		stopChs = append(stopChs, stopCh)
		go s.listenEvents(info.EventCh, aggCh, stopCh, name, getUIDFromEventFn)
	}

	eventInfo := EventInfo{
		Name:    name,
		Query:   query,
		StopChs: stopChs,
		OutCh:   aggCh,
	}
	s.eventInfos.Store(name, eventInfo)

	return &eventInfo, nil
}

// Unsubscribe unsubscribes events from client chain.
func (s *Subscriber) Unsubscribe(name string) error {
	v, found := s.eventInfos.Load(name)
	if !found {
		return fmt.Errorf("event is not subscribed; name %s", name)
	}

	info, ok := v.(EventInfo)
	if !ok {
		return fmt.Errorf("event info is invalid; name %s: %v", name, v)
	}

	for _, stopCh := range info.StopChs {
		stopCh <- struct{}{}
	}

	s.eventInfos.Delete(name)

	return nil
}

func (s *Subscriber) listenEvents(
	in <-chan ctypes.ResultEvent,
	out chan<- ctypes.ResultEvent,
	stopCh <-chan struct{},
	evName string,
	getUIDFromEventFn func(e ctypes.ResultEvent) (string, error),
) {
	for {
		select {
		case msg := <-in:
			uid, err := getUIDFromEventFn(msg)
			if err != nil {
				s.logger.Error("Failed to get UID from event", "error", err)
				continue
			}

			key := evName + uid
			// Check if the event is already in the cache
			if _, found := s.cache.Get(key); found {
				s.logger.Debug("Event already in cache", "uid: %s", uid)
				continue
			}

			s.logger.Debug("New event", "uid: %s", uid)
			s.cache.Add(key, struct{}{}, cache.DefaultExpiration)
			out <- msg
		case <-stopCh:
			break
		}
	}
}
