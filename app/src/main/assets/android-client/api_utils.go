package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func truncateResp(resp map[string]interface{}) string {
	b, err := json.Marshal(resp)
	if err != nil {
		return fmt.Sprintf("(unmarshallable: %v)", err)
	}
	return truncateStr(string(b), 300)
}

func apiExtractStr(resp map[string]interface{}, keys ...string) (string, error) {
	var cur interface{} = resp
	for _, k := range keys {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("expected map at key %q, got %T", k, cur)
		}
		cur = m[k]
	}
	s, ok := cur.(string)
	if !ok {
		return "", fmt.Errorf("expected string at end of path, got %T", cur)
	}
	return s, nil
}

func apiExtractFloat(resp map[string]interface{}, keys ...string) (float64, error) {
	var cur interface{} = resp
	for _, k := range keys {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("expected map at key %q, got %T", k, cur)
		}
		cur = m[k]
	}
	f, ok := cur.(float64)
	if !ok {
		return 0, fmt.Errorf("expected float64 at end of path, got %T", cur)
	}
	return f, nil
}

func parseTURNAddresses(resp map[string]interface{}) []string {
	turnServer, ok := resp["turn_server"].(map[string]interface{})
	if !ok {
		return nil
	}
	urls, ok := turnServer["urls"].([]interface{})
	if !ok {
		return nil
	}
	var addrs []string
	for i, u := range urls {
		s, ok := u.(string)
		if !ok {
			log.Printf("[TURN] urls[%d] non-string %T, skipping", i, u)
			continue
		}
		clean := strings.Split(s, "?")[0]
		addr := strings.TrimPrefix(strings.TrimPrefix(clean, "turn:"), "turns:")
		log.Printf("[TURN] urls[%d] = %s", i, addr)
		addrs = append(addrs, addr)
	}
	return addrs
}

func parseTURNFromWS2(turnObj map[string]interface{}) (user, pass string, addrs []string) {
	urls, ok := turnObj["urls"].([]interface{})
	if !ok || len(urls) == 0 {
		return
	}
	urlStr, _ := urls[0].(string)
	clean := strings.Split(urlStr, "?")[0]
	server := strings.TrimPrefix(strings.TrimPrefix(clean, "turn:"), "turns:")

	username, _ := turnObj["username"].(string)
	credential, _ := turnObj["credential"].(string)

	if server == "" {
		return
	}
	return username, credential, []string{server}
}
