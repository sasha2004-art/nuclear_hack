//go:build windows

package utils

import (
	"log"
	"os"
	"os/exec"
)

func SetupFirewallRules() {
	exePath, _ := os.Executable()

	log.Println("[BOOT] Настройка Брандмауэра Windows...")

	runNetsh("advfirewall", "firewall", "add", "rule",
		"name=Plotix_App", "dir=in", "action=allow",
		"program="+exePath, "enable=yes", "profile=any")

	runNetsh("advfirewall", "firewall", "add", "rule",
		"name=Plotix_Discovery", "dir=in", "action=allow",
		"protocol=UDP", "localport=9999", "profile=any")

	runNetsh("advfirewall", "firewall", "add", "rule",
		"name=Plotix_Transport", "dir=in", "action=allow",
		"protocol=TCP", "localport=10000", "profile=any")
}

func runNetsh(args ...string) {
	cmd := exec.Command("netsh", args...)
	_ = cmd.Run()
}
