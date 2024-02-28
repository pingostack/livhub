package config

import (
	"net"
	"runtime"

	"github.com/livekit/mediatransportutil/pkg/transport"
	"github.com/pingostack/neon/pkg/logger"
	"github.com/pion/ice/v3"
	"github.com/pion/webrtc/v4"
)

func getICEUDPMux(settings *Settings, se *webrtc.SettingEngine, ipFilter func(net.IP) bool, ifFilter func(string) bool) (udpMux ice.UDPMux, err error) {
	opts := []transport.UDPMuxFromPortOption{
		transport.UDPMuxFromPortWithReadBufferSize(defaultUDPBufferSize),
		transport.UDPMuxFromPortWithWriteBufferSize(defaultUDPBufferSize),
		transport.UDPMuxFromPortWithLogger(se.LoggerFactory.NewLogger("udp_mux")),
	}
	if settings.EnableLoopbackCandidate {
		opts = append(opts, transport.UDPMuxFromPortWithLoopback())
	}
	if ipFilter != nil {
		opts = append(opts, transport.UDPMuxFromPortWithIPFilter(ipFilter))
	}
	if ifFilter != nil {
		opts = append(opts, transport.UDPMuxFromPortWithInterfaceFilter(ifFilter))
	}
	if settings.BatchIO.BatchSize > 0 {
		opts = append(opts, transport.UDPMuxFromPortWithBatchWrite(settings.BatchIO.BatchSize, settings.BatchIO.MaxFlushInterval))
	}
	availablePorts := settings.UDPMuxPort.ToSlice()

	ports := make([]int, 0, len(availablePorts))
	for i := 0; i < runtime.NumCPU() && i < len(availablePorts); i++ {
		ports = append(ports, availablePorts[i])
	}

	muxes, err := transport.CreateUDPMuxesFromPorts(ports, opts...)
	if err != nil {
		return nil, err
	}

	udpMux = transport.NewMultiPortsUDPMux(muxes...)

	logger.Infof("using udp mux ports: %v", ports)
	return udpMux, nil
}
