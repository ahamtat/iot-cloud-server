package broker

import (
	"io"
	"sync"

	"github.com/AcroManiac/iot-cloud-server/internal/domain/interfaces"
)

type GatewayChannelsMap struct {
	mx       sync.Mutex
	channels map[string]io.ReadWriteCloser
}

func (gc *GatewayChannelsMap) Add(gatewayId string, ch interfaces.Channel) {
	gc.mx.Lock()
	gc.channels[gatewayId] = ch
	gc.mx.Unlock()
}

func (gc *GatewayChannelsMap) Get(gatewayId string) io.ReadWriteCloser {
	var ch io.ReadWriteCloser
	gc.mx.Lock()
	ch = gc.channels[gatewayId]
	gc.mx.Unlock()

	return ch
}

func (gc *GatewayChannelsMap) Remove(gatewayId string) {
	gc.mx.Lock()
	delete(gc.channels, gatewayId)
	gc.mx.Unlock()
}

func (gc *GatewayChannelsMap) GetChannels() []io.ReadWriteCloser {
	gc.mx.Lock()
	defer gc.mx.Unlock()

	chans := make([]io.ReadWriteCloser, len(gc.channels))
	for _, ch := range gc.channels {
		chans = append(chans, ch)
	}

	return chans
}
