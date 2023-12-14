package config

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/pingostack/neon/pkg/logger"
	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3"
)

type WebRTCConfig struct {
	Configuration  webrtc.Configuration
	SettingEngine  webrtc.SettingEngine
	UDPMux         ice.UDPMux
	TCPMuxListener *net.TCPListener
	NAT1To1IPs     []string
	UseMDNS        bool
}

func NewWebRTCConfig(settings *Settings) (*WebRTCConfig, error) {
	c := webrtc.Configuration{
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	}
	se := webrtc.SettingEngine{
		LoggerFactory: logger.NewPionLoggerFactory(nil),
	}

	var ifFilter func(string) bool
	if len(settings.Interfaces.Includes) != 0 || len(settings.Interfaces.Excludes) != 0 {
		ifFilter = InterfaceFilterFromConf(settings.Interfaces)
		se.SetInterfaceFilter(ifFilter)
	}

	var ipFilter func(net.IP) bool
	if len(settings.IPs.Includes) != 0 || len(settings.IPs.Excludes) != 0 {
		filter, err := IPFilterFromConf(settings.IPs)
		if err != nil {
			return nil, err
		}
		ipFilter = filter
		se.SetIPFilter(filter)
	}

	if !settings.UseMDNS {
		se.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)
	}

	var nat1to1IPs []string

	if settings.AutoGenerateExternalIP {
		externalNat1to1IPs, localNat1to1IPs, err := getNAT1to1IPsForConf(settings, ipFilter)
		if err != nil {
			return nil, err
		}
		nat1to1IPs = append(nat1to1IPs, externalNat1to1IPs...)
		nat1to1IPs = append(nat1to1IPs, localNat1to1IPs...)
		nat1to1IPs = append(nat1to1IPs, settings.NAT1To1IPs...)
		natMapping := make(map[string]string)
		for _, ip := range nat1to1IPs {
			values := strings.Split(ip, "/")
			if len(values) != 1 && len(values) != 2 {
				continue
			}
			if len(values) == 1 {
				natMapping[values[0]] = values[0]
			} else {
				natMapping[values[0]] = values[1]
			}
		}

		nat1to1IPs = []string{}
		for external, local := range natMapping {
			nat1to1IPs = append(nat1to1IPs, fmt.Sprintf("%s/%s", external, local))
		}

		logger.Infof("auto generated nat1to1 ips: %v", nat1to1IPs)
		se.SetNAT1To1IPs(nat1to1IPs, webrtc.ICECandidateTypeHost)
	} else if len(settings.NAT1To1IPs) > 0 {
		nat1to1IPs = settings.NAT1To1IPs
		se.SetNAT1To1IPs(nat1to1IPs, webrtc.ICECandidateTypeHost)
	} else {
		if len(settings.STUNServers) > 0 {
			c.ICEServers = []webrtc.ICEServer{iceServerForStunServers(settings.STUNServers)}
		} else {
			c.ICEServers = []webrtc.ICEServer{iceServerForStunServers(defaultStunServers)}
		}
	}

	if settings.UseICELite {
		se.SetLite(true)
	}

	var udpMux ice.UDPMux
	var err error
	networkTypes := make([]webrtc.NetworkType, 0, 4)

	if !settings.ForceTCP {
		networkTypes = append(networkTypes,
			webrtc.NetworkTypeUDP4, webrtc.NetworkTypeUDP6,
		)
		if settings.ICEPortRange.Valid() {
			if err := se.SetEphemeralUDPPortRange(uint16(settings.ICEPortRange.StartPort()), uint16(settings.ICEPortRange.EndPort())); err != nil {
				return nil, err
			}
		} else if settings.UDPMuxPort.Valid() {
			udpMux, err = getICEUDPMux(settings, &se, ipFilter, ifFilter)
			if err != nil {
				return nil, err
			}
			se.SetICEUDPMux(udpMux)
		}
	}

	// use TCP mux when it'se set
	var tcpListener *net.TCPListener
	if settings.TCPPort != 0 {
		networkTypes = append(networkTypes,
			webrtc.NetworkTypeTCP4, webrtc.NetworkTypeTCP6,
		)
		tcpListener, err = net.ListenTCP("tcp", &net.TCPAddr{
			Port: int(settings.TCPPort),
		})
		if err != nil {
			return nil, err
		}

		tcpMux := ice.NewTCPMuxDefault(ice.TCPMuxParams{
			Logger:          se.LoggerFactory.NewLogger("tcp_mux"),
			Listener:        tcpListener,
			ReadBufferSize:  readBufferSize,
			WriteBufferSize: writeBufferSizeInBytes,
		})

		se.SetICETCPMux(tcpMux)
	}

	if len(networkTypes) == 0 {
		return nil, errors.New("TCP is forced but not configured")
	}
	se.SetNetworkTypes(networkTypes)

	if settings.EnableLoopbackCandidate {
		se.SetIncludeLoopbackCandidate(true)
	}

	return &WebRTCConfig{
		Configuration:  c,
		SettingEngine:  se,
		UDPMux:         udpMux,
		TCPMuxListener: tcpListener,
		NAT1To1IPs:     nat1to1IPs,
		UseMDNS:        settings.UseMDNS,
	}, nil
}

func iceServerForStunServers(servers []string) webrtc.ICEServer {
	iceServer := webrtc.ICEServer{}
	for _, stunServer := range servers {
		iceServer.URLs = append(iceServer.URLs, fmt.Sprintf("stun:%s", stunServer))
	}
	return iceServer
}

func getNAT1to1IPsForConf(settings *Settings, ipFilter func(net.IP) bool) ([]string, []string, error) {
	stunServers := settings.STUNServers
	if len(stunServers) == 0 {
		stunServers = defaultStunServers
	}
	localIPs, err := GetLocalIPAddresses(settings.EnableLoopbackCandidate, nil)
	if err != nil {
		return nil, nil, err
	}
	type ipmapping struct {
		externalIP string
		localIP    string
	}
	addrCh := make(chan ipmapping, len(localIPs))

	var udpPorts []int
	portRangeStart, portRangeEnd := uint16(settings.ICEPortRange.StartPort()), uint16(settings.ICEPortRange.EndPort())
	if portRangeStart != 0 && portRangeEnd != 0 {
		for i := 0; i < 5; i++ {
			udpPorts = append(udpPorts, rand.Intn(int(portRangeEnd-portRangeStart))+int(portRangeStart))
		}
	} else if settings.UDPMuxPort.Valid() {
		udpPorts = append(udpPorts, settings.UDPMuxPort.StartPort())
	} else {
		udpPorts = append(udpPorts, 0)
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	for _, ip := range localIPs {
		if ipFilter != nil && !ipFilter(net.ParseIP(ip)) {
			continue
		}

		wg.Add(1)
		go func(localIP string) {
			defer wg.Done()
			for _, port := range udpPorts {
				addr, err := GetExternalIP(ctx, stunServers, &net.UDPAddr{IP: net.ParseIP(localIP), Port: port})
				if err != nil {
					if strings.Contains(err.Error(), "address already in use") {
						logger.Debug("failed to get external ip, address already in use", "local", localIP, "port", port)
						continue
					}
					logger.Info("failed to get external ip", "local", localIP, "err", err)
					return
				}
				addrCh <- ipmapping{externalIP: addr, localIP: localIP}
				return
			}
		}(ip)
	}

	var firstResolved bool
	natMapping := make(map[string]string)
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

done:
	for {
		select {
		case mapping := <-addrCh:
			if !firstResolved {
				firstResolved = true
				timeout.Reset(1 * time.Second)
			}
			if _, ok := natMapping[mapping.externalIP]; !ok {
				natMapping[mapping.externalIP] = mapping.localIP
			}

		case <-timeout.C:
			break done
		}
	}
	cancel()
	wg.Wait()

	localNat1to1IPs := []string{}
	// mapping unresolved local ip to itself
	for _, local := range localIPs {
		var found bool
		for _, localIPMapping := range natMapping {
			if local == localIPMapping {
				found = true
				break
			}
		}
		if !found {
			localNat1to1IPs = append(localNat1to1IPs, fmt.Sprintf("%s/%s", local, local))
		}
	}

	externalNat1to1IPs := make([]string, 0, len(natMapping))
	for external, local := range natMapping {
		externalNat1to1IPs = append(externalNat1to1IPs, fmt.Sprintf("%s/%s", external, local))
	}

	return externalNat1to1IPs, localNat1to1IPs, nil
}

func InterfaceFilterFromConf(ifs InterfacesConfig) func(string) bool {
	includes := ifs.Includes
	excludes := ifs.Excludes
	return func(se string) bool {
		// filter by include interfaces
		if len(includes) > 0 {
			for _, iface := range includes {
				if iface == se {
					return true
				}
			}
			return false
		}

		// filter by exclude interfaces
		if len(excludes) > 0 {
			for _, iface := range excludes {
				if iface == se {
					return false
				}
			}
		}
		return true
	}
}

func IPFilterFromConf(ips IPsConfig) (func(ip net.IP) bool, error) {
	var ipnets [2][]*net.IPNet
	var err error
	for i, ips := range [][]string{ips.Includes, ips.Excludes} {
		ipnets[i], err = func(fromIPs []string) ([]*net.IPNet, error) {
			var toNets []*net.IPNet
			for _, ip := range fromIPs {
				_, ipnet, err := net.ParseCIDR(ip)
				if err != nil {
					return nil, err
				}
				toNets = append(toNets, ipnet)
			}
			return toNets, nil
		}(ips)

		if err != nil {
			return nil, err
		}
	}

	includes, excludes := ipnets[0], ipnets[1]

	return func(ip net.IP) bool {
		if len(includes) > 0 {
			for _, ipn := range includes {
				if ipn.Contains(ip) {
					return true
				}
			}
			return false
		}

		if len(excludes) > 0 {
			for _, ipn := range excludes {
				if ipn.Contains(ip) {
					return false
				}
			}
		}
		return true
	}, nil
}
