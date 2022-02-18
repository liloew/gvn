package tun

import (
	"github.com/sirupsen/logrus"
	"github.com/songgao/packets/ethernet"
	tun "golang.zx2c4.com/wireguard/tun"
)

type Device struct {
	Name string
	// CIDR
	Ip string
	// Mask    string
	Mtu     int
	Subnets []string
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
	// TODO: Config ip and firewall
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
