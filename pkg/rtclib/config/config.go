package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pion/webrtc/v4"
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

type OAuthCredential struct {
	MACKey      string `json:"MACKey" mapstructure:"MACKey" yaml:"MACKey,omitempty"`
	AccessToken string `json:"AccessToken" mapstructure:"AccessToken" yaml:"AccessToken,omitempty"`
}

type ICEServer struct {
	URLs           []string `json:"urls" mapstructure:"urls" yaml:"urls,omitempty"`
	Username       string   `json:"username,omitempty" mapstructure:"username,omitempty" yaml:"username,omitempty"`
	Credential     string   `json:"credential,omitempty" mapstructure:"credential,omitempty" yaml:"credential,omitempty"`
	CredentialType string   `json:"credentialType,omitempty" mapstructure:"credentialType,omitempty" yaml:"credentialType,omitempty"`
}

func (ice *ICEServer) ToWebRTCICEServer() (webrtc.ICEServer, error) {
	if ice.CredentialType == "" {
		ice.CredentialType = webrtc.ICECredentialTypePassword.String()
	}

	if ice.CredentialType == webrtc.ICECredentialTypePassword.String() {
		return webrtc.ICEServer{
			URLs:           ice.URLs,
			Username:       ice.Username,
			Credential:     ice.Credential,
			CredentialType: webrtc.ICECredentialTypePassword,
		}, nil
	} else if ice.CredentialType == webrtc.ICECredentialTypeOauth.String() {
		// mackey:(base64 encoded mac key) token:(base64 encoded token)
		mackey := ""
		token := ""

		// Split the credential into mackey and token
		credParts := strings.Split(ice.Credential, " ")
		for _, part := range credParts {
			if strings.HasPrefix(part, "mackey:") {
				mackey = strings.TrimPrefix(part, "mackey:")
			} else if strings.HasPrefix(part, "token:") {
				token = strings.TrimPrefix(part, "token:")
			}
		}

		return webrtc.ICEServer{
			URLs:           ice.URLs,
			Username:       ice.Username,
			Credential:     webrtc.OAuthCredential{MACKey: mackey, AccessToken: token},
			CredentialType: webrtc.ICECredentialTypeOauth,
		}, nil
	}

	return webrtc.ICEServer{}, webrtc.ErrUnknownType
}

func (ice *ICEServer) ToWhipLinkHeader() ([]string, error) {
	iceServer, err := ice.ToWebRTCICEServer()
	if err != nil {
		return nil, err
	}

	if iceServer.CredentialType == webrtc.ICECredentialTypeOauth {
		return nil, fmt.Errorf("OAuth credentials are not supported in WhipLinkHeader")
	}

	links := []string{}
	for _, url := range iceServer.URLs {
		link := fmt.Sprintf(`<%s>; rel="ice-server"`, url)
		if ice.Username != "" {
			link += fmt.Sprintf(`; username="%s"`, ice.Username)
		}

		if ice.Credential != "" {
			link += fmt.Sprintf(`; credential="%s"`, ice.Credential)
		}

		if ice.CredentialType != "" {
			link += fmt.Sprintf(`; credentialType="%s";`, ice.CredentialType)
		}

		links = append(links, link)
	}

	return links, nil
}

type Settings struct {
	UseICELite              bool             `json:"useIceLite,omitempty" yaml:"useIceLite,omitempty" mapstructure:"useIceLite,omitempty"`
	NAT1To1IPs              []string         `json:"nat1to1Ips,omitempty" yaml:"nat1to1Ips,omitempty" mapstructure:"nat1to1Ips,omitempty"`
	AutoGenerateExternalIP  bool             `json:"autoGenerateExternalIp,omitempty" yaml:"autoGenerateExternalIp,omitempty" mapstructure:"autoGenerateExternalIp,omitempty"`
	ICEPortRange            PortRange        `json:"icePortRange,omitempty" yaml:"icePortRange,omitempty" mapstructure:"icePortRange,omitempty"`
	UDPMuxPort              PortRange        `json:"udpMuxPort,omitempty" yaml:"udpMuxPort,omitempty" mapstructure:"udpMuxPort,omitempty"`
	TCPPort                 int              `json:"tcpPort,omitempty" yaml:"tcpPort,omitempty" mapstructure:"tcpPort,omitempty"`
	ICEServers              []ICEServer      `json:"iceServers,omitempty" yaml:"iceServers,omitempty" mapstructure:"iceServers,omitempty"`
	Interfaces              InterfacesConfig `json:"interfaces,omitempty" yaml:"interfaces,omitempty" mapstructure:"interfaces,omitempty"`
	IPs                     IPsConfig        `json:"ips,omitempty" yaml:"ips,omitempty" mapstructure:"ips,omitempty"`
	EnableLoopbackCandidate bool             `json:"enable_loopback_candidate,omitempty" yaml:"enable_loopback_candidate,omitempty" mapstructure:"enable_loopback_candidate,omitempty"`
	UseMDNS                 bool             `json:"useMdns,omitempty" yaml:"useMdns,omitempty" mapstructure:"useMdns,omitempty"`
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
