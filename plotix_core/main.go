package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"plotix_core/api"
	"plotix_core/core"
	"plotix_core/crypto"
	"plotix_core/discovery"
	"plotix_core/utils"
)

func main() {
	ifaceFlag := flag.String("iface", "", "Network interface name")
	flag.Parse()

	selectedIface := *ifaceFlag

	if selectedIface == "" {
		ifaces, err := utils.GetInterfacesList()
		if err != nil {
			log.Fatalf("[FATAL] Cannot get interfaces: %v", err)
		}

		if len(ifaces) == 0 {
			log.Fatal("[FATAL] No active network interfaces!")
		}

		fmt.Println("=== Plotix - Select network interface ===")
		for i, iface := range ifaces {
			fmt.Printf("[%d] %s (IP: %s)\n", i+1, iface.Name, iface.IP)
		}
		fmt.Print("Enter number: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		index, err := strconv.Atoi(input)
		if err != nil || index < 1 || index > len(ifaces) {
			log.Fatal("[FATAL] Invalid choice.")
		}
		selectedIface = ifaces[index-1].Name
	}

	log.Printf("[BOOT] Interface: %s", selectedIface)

	ident, err := crypto.InitIdentity("keystore.json")
	if err != nil {
		log.Fatalf("[FATAL] Key init error: %v", err)
	}

	log.Printf("[BOOT] PeerID: %s", ident.PeerID)

	state := core.NewNodeState(ident)

	discovery.Start(state, selectedIface)

	server := api.NewServer(state)
	server.Start("8080")
}
