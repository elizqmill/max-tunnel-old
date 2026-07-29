package main

import (
	"math/rand"
)

type Profile struct {
	UserAgent       string
	SecChUa         string
	SecChUaMobile   string
	SecChUaPlatform string
}

var profileList = []Profile{
	{
		UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
		SecChUa:         `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`,
		SecChUaMobile:   "?0",
		SecChUaPlatform: `"Windows"`,
	},
	{
		UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36",
		SecChUa:         `"Chromium";v="145", "Not-A.Brand";v="99", "Google Chrome";v="145"`,
		SecChUaMobile:   "?0",
		SecChUaPlatform: `"Windows"`,
	},
	{
		UserAgent:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
		SecChUa:         `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`,
		SecChUaMobile:   "?0",
		SecChUaPlatform: `"macOS"`,
	},
	{
		UserAgent:       "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
		SecChUa:         `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`,
		SecChUaMobile:   "?0",
		SecChUaPlatform: `"Linux"`,
	},
	{
		UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
		SecChUa:         `"Firefox";v="132", "Not-A.Brand";v="8", "Mozilla Firefox";v="132"`,
		SecChUaMobile:   "?0",
		SecChUaPlatform: `"Windows"`,
	},
}

var activeFingerprint = "chrome"

func SetActiveFingerprint(fp string) {
	activeFingerprint = fp
}

func GetActiveFingerprint() string {
	return activeFingerprint
}

func getRandomProfile() Profile {
	switch activeFingerprint {
	case "android":
		break
	case "ios":
		break
	case "safari":
		return profileList[2]
	case "firefox":
		break
	default:
		return profileList[rand.Intn(2)]
	}
	return profileList[rand.Intn(2)]
}
