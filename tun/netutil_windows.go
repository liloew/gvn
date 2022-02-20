package tun

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	VIP string
)

func ConfigAddr(dev Device) error {
	VIP = dev.Ip
	tmpfile, err := ioutil.TempFile("", "ConfigureAddr-*.bat")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File": tmpfile,
		}).Error("ConfigureAddr error when create tmp file")
	}
	logrus.WithFields(logrus.Fields{
		"File": tmpfile.Name(),
	}).Info("bat file to be executed")
	// cidr -> 10.30.20.0/24,172.16.1.1/23 and etc
	ipv4Addr, ipv4Net, err := net.ParseCIDR(VIP)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Parse VIP error")
	}
	vip := ipv4Addr.String()
	mask := ipv4MaskString(ipv4Net.Mask)
	serverVIP, _, _ := net.ParseCIDR(dev.ServerVIP)
	content := fmt.Sprintf("netsh interface ip set address name=%s source=static addr=%s mask=%s gateway=%s gwmetric=1", dev.Name, vip, mask, serverVIP.String())
	for _, route := range dev.Subnets {
		content = fmt.Sprintf("%s\nroute delete %s\nroute add -p %s %s metric 1", content, route, route, vip)
	}
	content = fmt.Sprintf("%s\nroute delete 0.0.0.0 %s\n", content, serverVIP)
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
			"File":  tmpfile.Name(),
		}).Error("AddRoute - Close tmp file error")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"COMMAND": content,
	}).Debug("Execute command")

	if err := RunCommand(tmpfile.Name()); err == nil {
		// remove if success otherwise for debug
		defer os.Remove(tmpfile.Name())
	}

	return nil
}

func UnloadFirewall(dev Device) error {
	return RemoveRoute(dev.Subnets)
}

func RemoveRoute(subnets []string) error {
	logrus.WithFields(logrus.Fields{
		"subnets": subnets,
		"VIP":     VIP,
	}).Debug("Subnets to be removed")
	tmpfile, err := ioutil.TempFile("", "RemoveRoute-*.bat")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File": tmpfile,
		}).Error("RemoveRoute error when create tmp file")
	}
	content := ""
	for _, route := range subnets {
		content = fmt.Sprintf("%s\nroute delete %s\n", content, route)
	}
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"COMMAND": content,
	}).Debug("Execute command")

	if err := tmpfile.Close(); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
			"File":  tmpfile.Name(),
		}).Error("RemoveRoute - Close tmp file error")
		return err
	}

	if err := RunCommand(tmpfile.Name()); err == nil {
		defer os.Remove(tmpfile.Name())
	}
	return nil
}

func AddRoute(subnets []string) error {
	logrus.WithFields(logrus.Fields{
		"subnets": subnets,
		"VIP":     VIP,
	}).Debug("Subnets to be added")
	tmpfile, err := ioutil.TempFile("", "AddRoute-*.bat")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File": tmpfile,
		}).Error("AddRoute error when create tmp file")
	}
	// cidr -> 10.30.20.0/24,172.16.1.1/23 and etc
	ipv4Addr, _, err := net.ParseCIDR(VIP)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Parse VIP error")
	}
	vipWithoutMask := ipv4Addr.String()
	content := ""
	for _, route := range subnets {
		content = fmt.Sprintf("%s\nroute delete %s\nroute add -p %s %s metric 1", content, route, route, vipWithoutMask)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"COMMAND": content,
	}).Debug("Execute command")

	if err := RunCommand(tmpfile.Name()); err == nil {
		defer os.Remove(tmpfile.Name())
	}

	if err := tmpfile.Close(); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
			"File":  tmpfile.Name(),
		}).Error("AddRoute - Close tmp file error")
		return err
	}
	return nil
}

func RefreshRoute(subnets []string) {
	RemoveRoute(subnets)
	AddRoute(subnets)
}
