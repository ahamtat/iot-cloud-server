package interfaces

import "io"

// Channel interface for data exchange between IoT-gateway and cloud server
type Channel interface {
	io.ReadWriteCloser
	Start()
	Stop()
}
