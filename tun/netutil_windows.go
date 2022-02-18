package tun

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
)

var (
	VIP string
)

func ConfigAddr(dev Device) error {
	tmpfile, err := ioutil.TempFile("", "ConfigureAddr-*.bat")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File": tmpfile,
		}).Error("AddRoute error when create tmp file")
	}
	// DEBUG
	// defer os.Remove(tmpfile.Name())
	logrus.WithFields(logrus.Fields{
		"File": tmpfile.Name(),
	}).Info("bat file to be executed")
	// cidr -> 10.30.20.0/24,172.16.1.1/23 and etc
	ipv4Addr, ipv4Net, err := net.ParseCIDR(dev.Ip)
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

	RunCommand(tmpfile.Name())

	return nil
}

func RemoveRoute(subnets []string) {

}

func AddRoute(subnets []string, vip string) error {
	VIP = vip
	logrus.WithFields(logrus.Fields{
		"subnets": subnets,
		"VIP":     vip,
	}).Debug("Subnets to be added")
	tmpfile, err := ioutil.TempFile("", "AddRoute-*.sh")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File": tmpfile,
		}).Error("AddRoute error when create tmp file")
	}
	defer os.Remove(tmpfile.Name())
	// cidr -> 10.30.20.0/24,172.16.1.1/23 and etc
	content := ""
	for _, route := range subnets {
		// "${IPOPR}" route add ${ROU} dev $INTERFACE
		// content = fmt.Sprintf("%s\n${IPOPR} route add %s dev $INTERFACE", content, route)
		// 不在此处更新, 此处逻辑在客户端
		// // 更新路由表
		// IPTable.AddByString(route, vip)
		content = fmt.Sprintf("%s\nroute delete %s\nroute add -p %s %%GVN_IP%% metric 1", content, route, route)
	}
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"COMMAND": content,
	}).Debug("Execute command")

	RunCommand(tmpfile.Name())

	if err := tmpfile.Close(); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
			"File":  tmpfile.Name(),
		}).Error("AddRoute - Close tmp file error")
		return err
	}
	return nil
}

func RefreshRoute(subnet []string) {

}

func RunCommand(filepath string) error {
	// ipv4Addr, ipv4Net, err := net.ParseCIDR(VIP)
	// if err != nil {
	// 	logrus.WithFields(logrus.Fields{
	// 		"ERROR":   err,
	// 		"ip":      VIP,
	// 		"IPv4Net": ipv4Net,
	// 	}).Error("Does not provide VIP")
	// }
	cmd := exec.Command(filepath)
	// 传入环境变量
	cmd.Env = os.Environ()
	// cmd.Env = append(cmd.Env,
	// 	"MODE="+viper.GetString("gvn.mode"),
	// 	"INTERFACE="+viper.GetString("gvn.device.name"),
	// 	"GVN_IP="+ipv4Addr.String(),
	// 	"IP_COMMAND="+viper.GetString("gvn.device.commands.ip"),
	// 	"IPTABLES_COMMAND="+viper.GetString("gvn.device.commands.iptables"),
	// )
	// 计算掩码
	if err := cmd.Run(); err != nil {
		output, _ := cmd.Output()
		logrus.WithFields(logrus.Fields{
			"script": filepath,
			"ERROR":  err,
			"OUTPUT": output,
		}).Error("Execute failed")
		return err
	} else {
		output, _ := cmd.Output()
		logrus.WithFields(logrus.Fields{
			"OUTPUT": output,
		}).Debug("Execute success")
	}
	return nil
}
