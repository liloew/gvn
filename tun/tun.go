package tun

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/songgao/packets/ethernet"
	tun "golang.zx2c4.com/wireguard/tun"
)

type Device struct {
	Name string
	// CIDR
	Ip string
	// Mask    string
	Mtu       int
	Subnets   []string
	ServerVIP string
}

var (
	device tun.Device
)

func NewTun(dev Device) {
	ifce, err := tun.CreateTUN(dev.Name, dev.Mtu)
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
	n, err := device.Read(frame, 0)
	return n, err
}

func Write(frame ethernet.Frame) (int, error) {
	n, err := device.Write(frame, 0)
	return n, err
}

func Close() {
	if err := device.Close(); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Close TUN device errro")
	}
}

func ipv4MaskString(m []byte) string {
	if len(m) != 4 {
		logrus.Panic("ipv4Mask: len must be 4 bytes")
	}

	return fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])
}
