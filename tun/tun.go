package tun

import (
	"fmt"
	"runtime"

	tun "github.com/liloew/wireguard-go/tun"
	"github.com/sirupsen/logrus"
	"github.com/songgao/packets/ethernet"
)

type Device struct {
	Name string
	// CIDR
	Ip string
	// Mask    string
	Mtu       int
	Subnets   []string
	ServerVIP string
	// iptables
	Port uint
}

var (
	device tun.Device
)

func NewTun(dev Device) {
	ifce, err := tun.CreateTUN(dev.Name, dev.Mtu, true)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Panic("Create TUN error")
	}
	if err := ConfigAddr(dev); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Panic("Configure TUN error")
	}
	device = ifce
}

func Read(frame []byte) (int, error) {
	if runtime.GOOS == "darwin" {
		return device.Read(frame, 4)
	}
	return device.Read(frame, 0)
}

func Write(frame ethernet.Frame) (int, error) {
	if runtime.GOOS == "darwin" {
		return device.Write(frame, 4)
	}
	return device.Write(frame, 0)
}

func Close(dev Device) error {
	if err := UnloadFirewall(dev); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Unload ip/firewall rules error")
		return err
	}
	if err := device.Close(); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Close TUN device error")
		return err
	}
	return nil
}

func ipv4MaskString(m []byte) string {
	if len(m) != 4 {
		logrus.Panic("ipv4Mask: len must be 4 bytes")
	}

	return fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])
}
