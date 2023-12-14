package config

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultUDPMuxPort      = PortRange("8899")
	readBufferSize         = 50
	minUDPBufferSize       = 5_000_000
	writeBufferSizeInBytes = 4 * 1024 * 1024
	defaultUDPBufferSize   = 16_777_216
)

var defaultStunServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun2.l.google.com:19302",
	"stun3.l.google.com:19302",
	"stun4.l.google.com:19302",
	"stun.ekiga.net",
	"stun.ideasip.com",
	"stun.schlund.de",
	"stun.stunprotocol.org:3478",
	"stun.voiparound.com",
	"stun.voipbuster.com",
	"stun.voipstunt.com",
	"stun.voxgratia.org",
	"stun.services.mozilla.com",
}

type ICEConfig struct {
	ICEDisconnectedTimeout   time.Duration `json:"iceDisconnectedTimeout" mapstructure:"iceDisconnectedTimeout" p:"iceDisconnectedTimeout"`
	ICEFailedTimeout         time.Duration `json:"iceFailedTimeout" mapstructure:"iceFailedTimeout" p:"iceFailedTimeout"`
	ICEKeepaliveInterval     time.Duration `json:"iceKeepaliveInterval" mapstructure:"iceKeepaliveInterval" p:"iceKeepaliveInterval"`
	MinTcpICEConnectTimeout  time.Duration `json:"minTcpICEConnectTimeout" mapstructure:"minTcpICEConnectTimeout" p:"minTcpICEConnectTimeout"`
	MaxTcpICEConnectTimeout  time.Duration `json:"maxTcpICEConnectTimeout" mapstructure:"maxTcpICEConnectTimeout" p:"maxTcpICEConnectTimeout"`
	ShortConnectionThreshold time.Duration `json:"shortConnectionThreshold" mapstructure:"shortConnectionThreshold" p:"shortConnectionThreshold"`
}

type CodecConfig struct {
	Mime     string `json:"mime,omitempty" yaml:"mime,omitempty" mapstructure:"mime,omitempty"`
	FmtpLine string `json:"fmtp_line,omitempty" yaml:"fmtp_line,omitempty" mapstructure:"fmtp_line,omitempty"`
}

type IPsConfig struct {
	Includes []string `json:"includes,omitempty" yaml:"includes,omitempty" mapstructure:"includes,omitempty"`
	Excludes []string `json:"excludes,omitempty" yaml:"excludes,omitempty" mapstructure:"excludes,omitempty"`
}

type BatchIOConfig struct {
	BatchSize        int           `yaml:"batch_size,omitempty"`
	MaxFlushInterval time.Duration `yaml:"max_flush_interval,omitempty"`
}

type InterfacesConfig struct {
	Includes []string `yaml:"includes,omitempty"`
	Excludes []string `yaml:"excludes,omitempty"`
}

// PortRange is a range of ports, eg 10000-20000 or 10000
type PortRange string

func (pr *PortRange) Valid() bool {
	regex := regexp.MustCompile(`^\d{1,5}-\d{1,5}$`)
	if !regex.MatchString(pr.String()) {
		return false
	}

	startPort := pr.StartPort()
	endPort := pr.EndPort()

	if startPort <= 0 || startPort > endPort {
		return false
	}

	return true
}

func (pr *PortRange) String() string {
	return string(*pr)
}

func (pr *PortRange) StartPort() int {
	values := strings.Split(pr.String(), "-")
	startPort, _ := strconv.Atoi(values[0])
	return startPort
}

func (pr *PortRange) EndPort() int {
	values := strings.Split(pr.String(), "-")
	if len(values) == 1 {
		return pr.StartPort()
	}
	endPort, _ := strconv.Atoi(values[1])
	return endPort
}

func (pr *PortRange) Validate() {
	if pr.Valid() {
		return
	}

	*pr = defaultUDPMuxPort
}

func (pr *PortRange) ToSlice() []int {
	start := pr.StartPort()
	end := pr.EndPort()

	ret := []int{}
	for i := start; i <= end; i++ {
		ret = append(ret, i)
	}

	return ret
}

type Settings struct {
	UseICELite              bool             `json:"use_ice_lite,omitempty" yaml:"use_ice_lite,omitempty" mapstructure:"use_ice_lite,omitempty"`
	NAT1To1IPs              []string         `json:"nat_1to1_ips,omitempty" yaml:"nat_1to1_ips,omitempty" mapstructure:"nat_1to1_ips,omitempty"`
	AutoGenerateExternalIP  bool             `json:"auto_generate_external_ip,omitempty" yaml:"auto_generate_external_ip,omitempty" mapstructure:"auto_generate_external_ip,omitempty"`
	ICEPortRange            PortRange        `json:"ice_port_range,omitempty" yaml:"ice_port_range,omitempty" mapstructure:"ice_port_range,omitempty"`
	UDPMuxPort              PortRange        `json:"udp_mux_port,omitempty" yaml:"udp_mux_port,omitempty" mapstructure:"udp_mux_port,omitempty"`
	TCPPort                 int              `json:"tcp_port,omitempty" yaml:"tcp_port,omitempty" mapstructure:"tcp_port,omitempty"`
	STUNServers             []string         `json:"stun_servers,omitempty" yaml:"stun_servers,omitempty" mapstructure:"stun_servers,omitempty"`
	Interfaces              InterfacesConfig `json:"interfaces,omitempty" yaml:"interfaces,omitempty" mapstructure:"interfaces,omitempty"`
	IPs                     IPsConfig        `json:"ips,omitempty" yaml:"ips,omitempty" mapstructure:"ips,omitempty"`
	EnableLoopbackCandidate bool             `json:"enable_loopback_candidate,omitempty" yaml:"enable_loopback_candidate,omitempty" mapstructure:"enable_loopback_candidate,omitempty"`
	UseMDNS                 bool             `json:"use_mdns,omitempty" yaml:"use_mdns,omitempty" mapstructure:"use_mdns,omitempty"`
	BatchIO                 BatchIOConfig    `json:"batch_io,omitempty" yaml:"batch_io,omitempty" mapstructure:"batch_io,omitempty"`
	ForceTCP                bool             `json:"force_tcp,omitempty" yaml:"force_tcp,omitempty" mapstructure:"force_tcp,omitempty"`
	ICEConfig               ICEConfig        `json:"ice_config,omitempty" yaml:"ice_config,omitempty" mapstructure:"ice_config,omitempty"`
}

func (settings *Settings) Validate() error {
	if !settings.ICEPortRange.Valid() && !settings.UDPMuxPort.Valid() {
		settings.UDPMuxPort.Validate()
	}

	return nil
}
