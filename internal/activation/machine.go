package activation

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"runtime"
	"strings"
)

func MachineID() (string, error) {
	macs, err := macAddresses()
	if err != nil {
		return "", err
	}

	parts := []string{
		strings.Join(macs, ","),
		runtime.GOOS,
		runtime.GOARCH,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:]), nil
}

func macAddresses() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var macs []string
	for _, i := range ifaces {
		if i.Flags&net.FlagLoopback != 0 {
			continue
		}
		hw := i.HardwareAddr.String()
		if hw == "" {
			continue
		}
		macs = append(macs, hw)
	}
	return macs, nil
}
