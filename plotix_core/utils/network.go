package utils

import (
	"fmt"
	"net"
)

type InterfaceInfo struct {
	Name string
	IP   string
}

func GetInterfacesList() ([]InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var list []InterfaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				list = append(list, InterfaceInfo{
					Name: iface.Name,
					IP:   ipnet.IP.String(),
				})
				break
			}
		}
	}
	return list, nil
}

func GetInterfaceByName(name string) (*net.Interface, net.IP, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, nil, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			return iface, ipnet.IP, nil
		}
	}

	return nil, nil, fmt.Errorf("interface %s has no IPv4 address", name)
}
