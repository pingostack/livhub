package config

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pingostack/livhub/pkg/logger"
	"github.com/pion/stun"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
)

func GetLocalIPAddresses(includeLoopback bool, preferredInterfaces []string) ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	loopBacks := make([]string, 0)
	addresses := make([]string, 0)
	for _, iface := range ifaces {
		if len(preferredInterfaces) != 0 && !slices.Contains(preferredInterfaces, iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch typedAddr := addr.(type) {
			case *net.IPNet:
				ip = typedAddr.IP.To4()
			case *net.IPAddr:
				ip = typedAddr.IP.To4()
			default:
				continue
			}
			if ip == nil {
				continue
			}
			if ip.IsLoopback() {
				loopBacks = append(loopBacks, ip.String())
			} else {
				addresses = append(addresses, ip.String())
			}
		}
	}

	if includeLoopback {
		addresses = append(addresses, loopBacks...)
	}

	if len(addresses) > 0 {
		return addresses, nil
	}
	if len(loopBacks) > 0 {
		return loopBacks, nil
	}
	return nil, fmt.Errorf("could not find local IP address")
}

func GetExternalIP(ctx context.Context, iceServers []ICEServer, localAddr net.Addr) (ip string, err error) {
	logger.Infof("getting external IP, iceServers: %v, localAddr: %s", iceServers, localAddr.String())
	if len(iceServers) == 0 {
		return "", errors.New("STUN servers are required but not defined")
	}

	for _, iceServer := range iceServers {
		if len(iceServer.URLs) == 0 {
			continue
		}
		ip, err = getExternalIP(ctx, iceServer, localAddr)
		if err == nil {
			return
		}
		logger.Warnf("failed to get external IP from %s: %v", iceServer, err)
	}

	return "", errors.Wrap(err, "could not resolve external IP")
}

func getExternalIP(ctx context.Context, iceServer ICEServer, localAddr net.Addr) (string, error) {

	dialer := &net.Dialer{
		LocalAddr: localAddr,
	}
	var conn net.Conn
	var err error
	for _, url := range iceServer.URLs {
		conn, err = dialer.Dial("udp4", url)
		if err != nil {
			continue
		}
	}
	if err != nil {
		return "", errors.Wrap(err, "could not dial STUN server")
	}

	c, err := stun.NewClient(conn)
	if err != nil {
		return "", err
	}
	defer c.Close()

	message, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return "", err
	}

	var stunErr error
	// sufficiently large buffer to not block it
	ipChan := make(chan string, 20)
	err = c.Start(message, func(res stun.Event) {
		if res.Error != nil {
			stunErr = res.Error
			return
		}

		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			stunErr = err
			return
		}
		ip := xorAddr.IP.To4()
		if ip != nil {
			ipChan <- ip.String()
		}
	})
	if err != nil {
		return "", err
	}

	ctx1, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	select {
	case nodeIP := <-ipChan:
		if localAddr == nil {
			return nodeIP, nil
		}
		_ = c.Close()
		return nodeIP, validateExternalIP(ctx1, nodeIP, localAddr.(*net.UDPAddr))
	case <-ctx1.Done():
		msg := "could not determine public IP"
		if stunErr != nil {
			return "", errors.Wrap(stunErr, msg)
		} else {
			return "", fmt.Errorf(msg)
		}
	}
}

// validateExternalIP validates that the external IP is accessible from the outside by listen the local address,
// it will send a magic string to the external IP and check the string is received by the local address.
func validateExternalIP(ctx context.Context, nodeIP string, addr *net.UDPAddr) error {
	srv, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer srv.Close()

	magicString := "9#B8D2Nvg2xg5P$ZRwJ+f)*^Nne6*W3WamGY"

	validCh := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := srv.Read(buf)
			if err != nil {
				return
			}
			if string(buf[:n]) == magicString {
				close(validCh)
				return
			}
		}
	}()

	cli, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP(nodeIP), Port: srv.LocalAddr().(*net.UDPAddr).Port})
	if err != nil {
		return err
	}
	defer cli.Close()

	if _, err = cli.Write([]byte(magicString)); err != nil {
		return err
	}

	ctx1, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	select {
	case <-validCh:
		return nil
	case <-ctx1.Done():
		break
	}
	return fmt.Errorf("could not validate external IP: %s", nodeIP)
}
