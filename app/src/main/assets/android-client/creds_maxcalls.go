package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tlsclient "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type maxCallsFailureKind string

const (
	maxCallsFailureSkipped  maxCallsFailureKind = "skipped"
	maxCallsFailureSetup    maxCallsFailureKind = "setup"
	maxCallsFailureNetwork  maxCallsFailureKind = "network"
	maxCallsFailureDecode   maxCallsFailureKind = "decode"
	maxCallsFailureMaxAPI   maxCallsFailureKind = "max_api"
	maxCallsFailureCall     maxCallsFailureKind = "call_unavailable"
	maxCallsFailureWS2      maxCallsFailureKind = "ws2"
	maxCallsFailureParse    maxCallsFailureKind = "parse"
)

type maxCallsFailure struct {
	Step string
	Kind maxCallsFailureKind
	Err  error
}

func (e *maxCallsFailure) Error() string {
	if e == nil {
		return "maxcalls failure"
	}
	if e.Err == nil {
		return fmt.Sprintf("step=%s kind=%s", e.Step, e.Kind)
	}
	return fmt.Sprintf("step=%s kind=%s: %v", e.Step, e.Kind, e.Err)
}

func (e *maxCallsFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newMaxCallsFailure(step string, kind maxCallsFailureKind, err error) error {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}
	return &maxCallsFailure{Step: step, Kind: kind, Err: err}
}

func describeMaxCallsFailure(err error) string {
	if err == nil {
		return ""
	}
	var failure *maxCallsFailure
	if errors.As(err, &failure) {
		return failure.Error()
	}
	return err.Error()
}

const (
	maxAPIVersion  = 270
	maxAppKey      = "CGPGAGLGDIHBABABA"
	maxWS2Host     = "wss://videowebrtc.okcdn.ru"
)

var maxProfile = Profile{
	UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
	SecChUa:         `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`,
	SecChUaMobile:   "?0",
	SecChUaPlatform: `"Windows"`,
}

func getMaxCredsViaVKCallsPath(ctx context.Context, link string, streamID int) (string, string, []string, error) {
	return getMaxCredsViaMaxCallsPath(ctx, link, streamID)
}

func getMaxCredsViaMaxCallsPath(ctx context.Context, link string, streamID int) (string, string, []string, error) {
	if os.Getenv("MAX_SKIP_CALLS") == "1" {
		return "", "", nil, newMaxCallsFailure("preflight", maxCallsFailureSkipped, fmt.Errorf("disabled by MAX_SKIP_CALLS=1"))
	}

	deviceID := uuid.New().String()
	profile := maxProfile

	client, err := tlsclient.NewHttpClient(tlsclient.NewNoopLogger(),
		tlsclient.WithTimeoutSeconds(20),
		tlsclient.WithClientProfile(profiles.Chrome_146),
		tlsclient.WithCookieJar(tlsclient.NewCookieJar()),
	)
	if err != nil {
		return "", "", nil, newMaxCallsFailure("setup", maxCallsFailureSetup, fmt.Errorf("create tls client: %w", err))
	}

	log.Printf("[STREAM %d] [MaxCalls] device_id=%s | TLS=Chrome_146", streamID, deviceID)

	doRequest := func(step string, url string) (map[string]interface{}, error) {
		req, err := fhttp.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(nil))
		if err != nil {
			return nil, newMaxCallsFailure(step, maxCallsFailureSetup, fmt.Errorf("create request: %w", err))
		}
		req.Header.Set("User-Agent", profile.UserAgent)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
		req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
		req.Header.Set("Origin", "https://max.ru")

		httpResp, err := client.Do(req)
		if err != nil {
			return nil, newMaxCallsFailure(step, maxCallsFailureNetwork, fmt.Errorf("request failed: %w", err))
		}
		defer func() {
			if closeErr := httpResp.Body.Close(); closeErr != nil {
				log.Printf("close response body: %s", closeErr)
			}
		}()

		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, newMaxCallsFailure(step, maxCallsFailureNetwork, fmt.Errorf("read response: %w", err))
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, newMaxCallsFailure(step, maxCallsFailureDecode, fmt.Errorf("unmarshal JSON: %w (body: %s)", err, truncateStr(string(body), 200)))
		}
		return resp, nil
	}

	vkAPIBase := "https://api.vk.me"
	vkVersion := "5.276"

	step1URL := fmt.Sprintf(
		"%s/method/auth.getAnonymToken?v=%s&client_id=8093730&link=%s&device_id=%s&anonymName=Guest&lang=en",
		vkAPIBase, vkVersion,
		neturl.QueryEscape("https://max.ru/joincall/"+link),
		deviceID,
	)
	resp1, err := doRequest("step1 auth.getAnonymToken", step1URL)
	if err != nil {
		return "", "", nil, err
	}
	anonymToken, err := apiExtractStr(resp1, "response", "token")
	if err != nil {
		return "", "", nil, newMaxCallsFailure("step1", maxCallsFailureParse, fmt.Errorf("parse token: %w", err))
	}
	log.Printf("[STREAM %d] [MaxCalls] step1 OK", streamID)

	step2URL := fmt.Sprintf(
		"%s/method/messages.getCallPreview?v=%s&anonymous_token=%s&device_id=%s&extended=1&fields=first_name,last_name,photo_200&lang=en&link=%s",
		vkAPIBase, vkVersion,
		neturl.QueryEscape(anonymToken),
		deviceID,
		neturl.QueryEscape("https://max.ru/joincall/"+link),
	)
	resp2, err := doRequest("step2 messages.getCallPreview", step2URL)
	if err != nil {
		return "", "", nil, err
	}
	if apiErr := vkCallsAPIError(resp2); apiErr != nil {
		return "", "", nil, newMaxCallsFailure("step2", maxCallsFailureMaxAPI, apiErr)
	}
	userIDFloat, err := apiExtractFloat(resp2, "response", "user_id")
	if err != nil {
		return "", "", nil, newMaxCallsFailure("step2", maxCallsFailureParse, fmt.Errorf("parse user_id: %w", err))
	}
	userIDStr := fmt.Sprintf("%.0f", userIDFloat)
	secret, err := apiExtractStr(resp2, "response", "secret")
	if err != nil {
		return "", "", nil, newMaxCallsFailure("step2", maxCallsFailureParse, fmt.Errorf("parse secret: %w", err))
	}
	log.Printf("[STREAM %d] [MaxCalls] step2 OK, user_id=%s", streamID, userIDStr)

	step3URL := fmt.Sprintf(
		"%s/method/messages.getAnonymCallToken?v=%s&anonymous_token=%s&device_id=%s&link=%s&name=Guest&user_id=%s&secret=%s&lang=en",
		vkAPIBase, vkVersion,
		neturl.QueryEscape(anonymToken),
		deviceID,
		neturl.QueryEscape("https://max.ru/joincall/"+link),
		userIDStr,
		neturl.QueryEscape(secret),
	)
	resp3, err := doRequest("step3 messages.getAnonymCallToken", step3URL)
	if err != nil {
		return "", "", nil, err
	}
	if apiErr := vkCallsAPIError(resp3); apiErr != nil {
		return "", "", nil, newMaxCallsFailure("step3", maxCallsFailureMaxAPI, apiErr)
	}
	okAnonymToken, err := apiExtractStr(resp3, "response", "token")
	if err != nil {
		return "", "", nil, newMaxCallsFailure("step3", maxCallsFailureParse, fmt.Errorf("parse token: %w", err))
	}
	log.Printf("[STREAM %d] [MaxCalls] step3 OK", streamID)

	okDeviceID := uuid.New().String()
	step4URL := "https://calls.okcdn.ru/fb.do?session_data=" +
		neturl.QueryEscape(fmt.Sprintf(
			`{"version":2,"device_id":"%s","client_version":"1.0.1"}`, okDeviceID,
		)) +
		"&method=auth.anonymLogin&format=JSON&application_key=CGMMEJLGDIHBABABA"
	resp4, err := doRequest("step4 auth.anonymLogin", step4URL)
	if err != nil {
		return "", "", nil, err
	}
	sessionKey, err := apiExtractStr(resp4, "session_key")
	if err != nil {
		return "", "", nil, newMaxCallsFailure("step4", maxCallsFailureParse, fmt.Errorf("parse session_key: %w", err))
	}
	log.Printf("[STREAM %d] [MaxCalls] step4 OK", streamID)

	step5URL := fmt.Sprintf(
		"https://calls.okcdn.ru/fb.do?joinLink=%s&isVideo=false&protocolVersion=5&anonymToken=%s&method=vchat.joinConversationByLink&format=JSON&application_key=CGMMEJLGDIHBABABA&session_key=%s",
		link, okAnonymToken, sessionKey,
	)
	resp5, err := doRequest("step5 vchat.joinConversationByLink", step5URL)
	if err != nil {
		return "", "", nil, err
	}
	if okErr := vkCallsOKError(resp5); okErr != nil {
		return "", "", nil, newMaxCallsFailure("step5", maxCallsFailureMaxAPI, fmt.Errorf("%w", okErr))
	}

	user, err := apiExtractStr(resp5, "turn_server", "username")
	if err != nil {
		return "", "", nil, newMaxCallsFailure("step5", maxCallsFailureParse, fmt.Errorf("parse username: %w", err))
	}
	pass, err := apiExtractStr(resp5, "turn_server", "credential")
	if err != nil {
		return "", "", nil, newMaxCallsFailure("step5", maxCallsFailureParse, fmt.Errorf("parse credential: %w", err))
	}
	addrs := parseTURNAddresses(resp5)
	if len(addrs) == 0 {
		return "", "", nil, newMaxCallsFailure("step5", maxCallsFailureParse, fmt.Errorf("turn_server.urls empty"))
	}

	log.Printf("[STREAM %d] [MaxCalls] SUCCESS, TURN urls=%d", streamID, len(addrs))
	return user, pass, addrs, nil
}

func getMaxCredsViaWS2Path(ctx context.Context, link string, streamID int) (string, string, []string, error) {
	conversationID := uuid.New().String()
	userID := uuid.New().String()

	wsURL := fmt.Sprintf("%s/ws2?token=%s&userId=%s&conversationId=%s&version=5&platform=ANDROID&clientType=ONE_ME&clientAppKey=%s",
		maxWS2Host,
		neturl.QueryEscape(link),
		neturl.QueryEscape(userID),
		neturl.QueryEscape(conversationID),
		maxAppKey,
	)

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	hdr := http.Header{}
	hdr.Set("Origin", "https://max.ru")
	hdr.Set("User-Agent", maxProfile.UserAgent)

	conn, _, err := dialer.DialContext(ctx, wsURL, hdr)
	if err != nil {
		return "", "", nil, newMaxCallsFailure("ws2_dial", maxCallsFailureWS2, fmt.Errorf("ws2 dial: %w", err))
	}
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return "", "", nil, newMaxCallsFailure("ws2_deadline", maxCallsFailureWS2, err)
	}

	acceptMsg := map[string]interface{}{
		"command":        "accept-call",
		"sequence":       0,
		"conversationId": conversationID,
		"participantId":  userID,
	}
	if err := conn.WriteJSON(acceptMsg); err != nil {
		return "", "", nil, newMaxCallsFailure("ws2_accept", maxCallsFailureWS2, fmt.Errorf("send accept-call: %w", err))
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				conn.WriteMessage(websocket.TextMessage, []byte("ping"))
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return "", "", nil, newMaxCallsFailure("ws2_read", maxCallsFailureWS2, fmt.Errorf("ws2 read: %w", err))
		}

		if string(msg) == "pong" {
			continue
		}

		var wsMsg map[string]interface{}
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			continue
		}

		notif, hasNotif := wsMsg["notification"].(string)
		if hasNotif && notif == "connection" {
			if cp, ok := wsMsg["conversationParams"].(map[string]interface{}); ok {
				if turn, ok := cp["turn"].(map[string]interface{}); ok {
					user, pass, addrs := parseTURNFromWS2(turn)
					if user != "" {
						log.Printf("[STREAM %d] [MaxWS2] SUCCESS, TURN: %s", streamID, addrs[0])
						return user, pass, addrs, nil
					}
				}
			}
		}

		if hasNotif && (notif == "hungup" || notif == "closed-conversation") {
			return "", "", nil, newMaxCallsFailure("ws2_call_ended", maxCallsFailureCall, fmt.Errorf("call ended before TURN creds"))
		}
	}
}

func getMaxCreds(ctx context.Context, hash string, streamID int) (string, string, []string, error) {
	if getVKAuthMode() == "maxcalls" {
		user, pass, addrs, err := getMaxCredsViaMaxCallsPath(ctx, hash, streamID)
		if err == nil {
			return user, pass, addrs, nil
		}
		log.Printf("[CREDS] Max Calls path failed (%s): %v, trying WS2", describeMaxCallsFailure(err), err)
		user, pass, addrs, err = getMaxCredsViaWS2Path(ctx, hash, streamID)
		if err == nil {
			return user, pass, addrs, nil
		}
		log.Printf("[CREDS] Max WS2 path failed (%s): %v, falling back to VK", describeMaxCallsFailure(err), err)
	}
	return getVkCredsCached(ctx, hash, streamID)
}
