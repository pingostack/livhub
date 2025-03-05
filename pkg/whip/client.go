package whip

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pion/webrtc/v4"
)

// Client represents a WHIP/WHEP client
type Client struct {
	httpClient  *http.Client
	endpoint    string
	resourceURL string
}

// NewClient creates a new WHIP/WHEP client
func NewClient(endpoint string) *Client {
	return &Client{
		httpClient: &http.Client{},
		endpoint:   strings.TrimSuffix(endpoint, "/"),
	}
}

// Publish starts a WHIP publishing session
func (c *Client) Publish(pc *webrtc.PeerConnection) error {
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer: %v", err)
	}

	if err = pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description: %v", err)
	}

	// Send the offer to the WHIP endpoint
	resp, err := c.sendOffer(offer.SDP, c.endpoint)
	if err != nil {
		return fmt.Errorf("failed to send offer: %v", err)
	}
	defer resp.Body.Close()

	// Handle the response
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Store the resource URL for later use
	c.resourceURL = resp.Header.Get("Location")

	// Read and set the answer
	answer, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read answer: %v", err)
	}

	if err = pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  string(answer),
	}); err != nil {
		return fmt.Errorf("failed to set remote description: %v", err)
	}

	return nil
}

// Subscribe starts a WHEP subscription session
func (c *Client) Subscribe(pc *webrtc.PeerConnection, streamID string) error {
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer: %v", err)
	}

	// Add stream ID to the SDP
	offer.SDP = addStreamIDToSDP(offer.SDP, streamID)

	if err = pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description: %v", err)
	}

	// Send the offer to the WHEP endpoint
	resp, err := c.sendOffer(offer.SDP, c.endpoint)
	if err != nil {
		return fmt.Errorf("failed to send offer: %v", err)
	}
	defer resp.Body.Close()

	// Handle the response
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Store the resource URL for later use
	c.resourceURL = resp.Header.Get("Location")

	// Read and set the answer
	answer, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read answer: %v", err)
	}

	if err = pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  string(answer),
	}); err != nil {
		return fmt.Errorf("failed to set remote description: %v", err)
	}

	return nil
}

// Close terminates the WHIP/WHEP session
func (c *Client) Close() error {
	if c.resourceURL == "" {
		return nil
	}

	req, err := http.NewRequest(http.MethodDelete, c.resourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper functions

func (c *Client) sendOffer(sdp, endpoint string) (*http.Response, error) {
	body := bytes.NewReader([]byte(sdp))
	req, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/sdp")
	return c.httpClient.Do(req)
}

func addStreamIDToSDP(sdp, streamID string) string {
	// Add stream ID to the first media section
	lines := strings.Split(sdp, "\r\n")
	newLines := make([]string, 0, len(lines)+1)

	foundMedia := false
	for _, line := range lines {
		newLines = append(newLines, line)
		if !foundMedia && strings.HasPrefix(line, "m=") {
			foundMedia = true
			newLines = append(newLines, fmt.Sprintf("a=mid:%s", streamID))
		}
	}

	return strings.Join(newLines, "\r\n")
}
