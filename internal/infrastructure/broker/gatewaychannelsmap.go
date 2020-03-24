package broker

import (
	"io"
	"sync"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
)

// ChannelsMap stores gateway channels
type ChannelsMap map[string]io.ReadWriteCloser

// GatewayChannelsMap keeps channel map with guarding mutex
type GatewayChannelsMap struct {
	mx       sync.Mutex
	channels ChannelsMap
}

// NewGatewayChannelsMap function constructs GatewayChannelsMap structure
func NewGatewayChannelsMap() *GatewayChannelsMap {
	return &GatewayChannelsMap{
		mx:       sync.Mutex{},
		channels: make(ChannelsMap),
	}
}

// Add channel to map securely
func (gc *GatewayChannelsMap) Add(gatewayID string, ch interfaces.Channel) {
	gc.mx.Lock()
	gc.channels[gatewayID] = ch
	gc.mx.Unlock()
}

// Get channel from map
func (gc *GatewayChannelsMap) Get(gatewayID string) io.ReadWriteCloser {
	var ch io.ReadWriteCloser
	gc.mx.Lock()
	ch, _ = gc.channels[gatewayID]
	gc.mx.Unlock()

	return ch
}

// Remove channel from map
func (gc *GatewayChannelsMap) Remove(gatewayID string) {
	gc.mx.Lock()
	delete(gc.channels, gatewayID)
	gc.mx.Unlock()
}

// GetChannels converts map values to slice of channels
func (gc *GatewayChannelsMap) GetChannels() []io.ReadWriteCloser {
	gc.mx.Lock()
	defer gc.mx.Unlock()

	chans := make([]io.ReadWriteCloser, len(gc.channels))
	for _, ch := range gc.channels {
		chans = append(chans, ch)
	}

	return chans
}
