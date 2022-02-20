//go:build !windows
// +build !windows

package tun

import (
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	VIP     string
	devName string
)

func ConfigAddr(dev Device) error {
	VIP = dev.Ip
	devName = dev.Name
	tmpfile, err := ioutil.TempFile("", "ConfigureAddr-*.sh")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File": tmpfile,
		}).Error("ConfigureAddr error when create tmp file")
	}
	logrus.WithFields(logrus.Fields{
		"File": tmpfile.Name(),
	}).Info("shell script to be executed")
	// cidr -> 10.30.20.0/24,172.16.1.1/23 and etc
	// ipv4Addr, ipv4Net, err := net.ParseCIDR(VIP)
	_, ipv4Net, err := net.ParseCIDR(VIP)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Parse VIP error")
	}
	// vip := ipv4Addr.String()
	mask := ipv4MaskString(ipv4Net.Mask)
	serverVIP, _, _ := net.ParseCIDR(dev.ServerVIP)
	// content := fmt.Sprintf("netsh interface ip set address name=%s source=static addr=%s mask=%s gateway=%s gwmetric=1", dev.Name, vip, mask, serverVIP.String())
	content := `export LOGFILE="/tmp/ConfigAddr-$(date +%Y%m%d%H%M%S).log"
export OS="$(uname)"
if [ "${OS}" == "Linux" ]
then
    export IPOPR="$(which ip)"
    export IPTABLES="$(which iptables)"

    "${IPOPR}" link set $INTERFACE up
    echo "${IPOPR} link set $INTERFACE up" >> "${LOGFILE}"
    "${IPOPR}" addr add "${GVN_VIP}" dev $INTERFACE
    echo "${IPOPR} addr add ${GVN_VIP} dev $INTERFACE" >> "${LOGFILE}"
    "${IPOPR}" route add ${GVN_VIP} dev $INTERFACE
    echo "${IPOPR} route add ${GVN_VIP} dev $INTERFACE" >> "${LOGFILE}"
    # TODO: check whether conflict with LAN
    for ROU in ${ROUTES}
    do
       "${IPOPR}" route add ${ROU} dev $INTERFACE
       echo "${IPOPR} route add ${ROU} dev $INTERFACE" >> "${LOGFILE}"
    done

    # CONFIGURE ALL NODE AS ROUTER
    #if [ "${MODE}" == "server" ]
    #then
        "${IPTABLES}" -I INPUT -p tcp -m multiport --dports "${SERVER_PORT}" -j ACCEPT
        echo "${IPTABLES} -I INPUT -p tcp -m multiport --dports ${SERVER_PORT} -j ACCEPT" >> "${LOGFILE}"
        #"${IPTABLES}" -I INPUT -p tcp -m multiport --dports "${ADMIN_SERVER_PORT}" -j ACCEPT
        #echo "${IPTABLES} -I INPUT -p tcp -m multiport --dports ${ADMIN_SERVER_PORT} -j ACCEPT" >> "${LOGFILE}"
    #fi

    "${IPTABLES}" -A INPUT -i $INTERFACE -j ACCEPT
    echo "${IPTABLES} -A INPUT -i $INTERFACE -j ACCEPT" >> "${LOGFILE}"
    "${IPTABLES}" -A OUTPUT -o $INTERFACE -j ACCEPT
    echo "${IPTABLES} -A OUTPUT -o $INTERFACE -j ACCEPT" >> "${LOGFILE}"
    "${IPTABLES}" -I FORWARD -i $INTERFACE -j ACCEPT
    echo "${IPTABLES} -I FORWARD -i $INTERFACE -j ACCEPT" >> "${LOGFILE}"
    "${IPTABLES}" -I FORWARD -o $INTERFACE -j ACCEPT
    echo "${IPTABLES} -I FORWARD -o $INTERFACE -j ACCEPT" >> "${LOGFILE}"
    # CONFIGURE ALL NODE AS ROUTER
    #if [ "${LAN_ZONE}" != "" ]
    #then
        # 当前节点作为中转路由使用
        #"${IPTABLES}" -t nat -A POSTROUTING -s "${LAN_ZONE}" -o "${INTERFACE}" -j MASQUERADE
        #echo "${IPTABLES} -t nat -A POSTROUTING -s ${LAN_ZONE} -o ${INTERFACE} -j MASQUERADE" >> "${LOGFILE}"
        # 所有来自 VIP 网段的报文均可转发到 server 所在的 LAN 网段
        echo 1 > /proc/sys/net/ipv4/ip_forward
        echo "echo 1 > /proc/sys/net/ipv4/ip_forward" >> "${LOGFILE}"
        "${IPTABLES}" -t nat -A POSTROUTING -s "${GVN_VIP}" ! -o "${INTERFACE}" -j MASQUERADE
        echo "${IPTABLES} -t nat -A POSTROUTING -s ${GVN_VIP} ! -o ${INTERFACE} -j MASQUERADE" >> "${LOGFILE}"
    #fi
elif [ "${OS}" == "Darwin" ]
then
    export IPOPR="$(which ifconfig)"
    export ROUTE="$(which route)"
    #"${IPOPR}" $INTERFACE 192.168.123.222 192.168.123.223 up netmask 255.255.255.0
    #"${ROUTE}" add -net 192.168.123.0 192.168.123.223 255.255.255.0
    sudo "${IPOPR}" $INTERFACE "${GVN_VIP}" "${SERVER_VIP}" up netmask "${MASK}"
    echo "sudo ${IPOPR} $INTERFACE ${GVN_VIP} ${SERVER_VIP} up netmask ${MASK}" >> "${LOGFILE}"
    #"${ROUTE}" add -net 192.168.123.0 192.168.123.223 255.255.255.0
    sudo "${ROUTE}" add ${GVN_VIP} -interface "${INTERFACE}"
    echo "sudo ${ROUTE} add ${GVN_VIP} -interface ${INTERFACE}" >> "${LOGFILE}"
    for ROU in ${ROUTES}
    do
        sudo "${ROUTE}" add ${ROU} -interface "${INTERFACE}"
        echo "sudo ${ROUTE} add ${ROU} -interface ${INTERFACE}" >> "${LOGFILE}"
    done
fi`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
			"File":  tmpfile.Name(),
		}).Error("ConfigureAddr - Close tmp file error")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"COMMAND": content,
	}).Debug("Execute command")

	envs := make([]string, 0)
	envs = append(envs, fmt.Sprintf("GVN_VIP=%s", VIP))
	envs = append(envs, fmt.Sprintf("SERVER_VIP=%s", serverVIP))
	envs = append(envs, fmt.Sprintf("MASK=%s", mask))
	envs = append(envs, fmt.Sprintf("ROUTES=%s", strings.Join(dev.Subnets, " ")))
	envs = append(envs, fmt.Sprintf("INTERFACE=%s", dev.Name))
	envs = append(envs, fmt.Sprintf("SERVER_PORT=%d", dev.Port))

	if err := RunCommand(tmpfile.Name(), envs...); err != nil {
		return err
	}
	// remove if success otherwise for debug
	// defer os.Remove(tmpfile.Name())
	return nil
}

func UnloadFirewall(dev Device) error {
	// TODO:
	VIP = dev.Ip
	tmpfile, err := ioutil.TempFile("", "UnloadFirewall-*.sh")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File": tmpfile,
		}).Error("ConfigureAddr error when create tmp file")
	}
	logrus.WithFields(logrus.Fields{
		"File": tmpfile.Name(),
	}).Info("shell script to be executed")
	// cidr -> 10.30.20.0/24,172.16.1.1/23 and etc
	// ipv4Addr, ipv4Net, err := net.ParseCIDR(VIP)
	_, ipv4Net, err := net.ParseCIDR(VIP)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Parse VIP error")
	}
	// vip := ipv4Addr.String()
	mask := ipv4MaskString(ipv4Net.Mask)
	serverVIP, _, _ := net.ParseCIDR(dev.ServerVIP)
	// content := fmt.Sprintf("netsh interface ip set address name=%s source=static addr=%s mask=%s gateway=%s gwmetric=1", dev.Name, vip, mask, serverVIP.String())
	content := `export LOGFILE="/tmp/UnloadFirewall-$(date +%Y%m%d%H%M%S).log"
export OS="$(uname)"
if [ "${OS}" == "Linux" ]
then
    export IPOPR="$(which ip)"
    export IPTABLES="$(which iptables)"

    #echo "${IPOPR} route del ${GVN_IP}" >> "${LOGFILE}"
    #"${IPOPR}" route del "${GVN_IP}"
    for ROU in ${ROUTES}
    do
       echo "${IPOPR} route del ${ROU}" >> "${LOGFILE}"
       "${IPOPR}" route del "${ROU}"
    done

    echo "${IPOPR} addr del ${GVN_IP} dev $INTERFACE"  >> "${LOGFILE}"
    "${IPOPR}" addr del "${GVN_IP}" dev $INTERFACE
    echo "${IPOPR} link set $INTERFACE down" >> "${LOGFILE}"
    "${IPOPR}" link set $INTERFACE down
    #if [ "${MODE}" == "server" ]
    #then
        echo "${IPTABLES} -D INPUT -p tcp -m multiport --dports ${SERVER_PORT} -j ACCEPT" >> "${LOGFILE}"
        "${IPTABLES}" -D INPUT -p tcp -m multiport --dports "${SERVER_PORT}" -j ACCEPT
        #echo "${IPTABLES} -D INPUT -p tcp -m multiport --dports ${ADMIN_SERVER_PORT} -j ACCEPT" >> "${LOGFILE}"
        #"${IPTABLES}" -D INPUT -p tcp -m multiport --dports "${ADMIN_SERVER_PORT}" -j ACCEPT
    fi

    echo "${IPTABLES} -D INPUT -i $INTERFACE -j ACCEPT" >> "${LOGFILE}"
    "${IPTABLES}" -D INPUT -i $INTERFACE -j ACCEPT
    echo "${IPTABLES} -D OUTPUT -o $INTERFACE -j ACCEPT" >> "${LOGFILE}"
    "${IPTABLES}" -D OUTPUT -o $INTERFACE -j ACCEPT
    echo "${IPTABLES} -D FORWARD -i $INTERFACE -j ACCEPT" >> "${LOGFILE}"
    "${IPTABLES}" -D FORWARD -i $INTERFACE -j ACCEPT
    echo "${IPTABLES} -D FORWARD -o $INTERFACE -j ACCEPT" >> "${LOGFILE}"
    "${IPTABLES}" -D FORWARD -o $INTERFACE -j ACCEPT
    #if [ "${LAN_ZONE}" != "" ]
    #then
        # 当前节点作为中转路由使用
        #"${IPTABLES}" -t nat -D POSTROUTING -s "${LAN_ZONE}" -o "${INTERFACE}" -j MASQUERADE
        #echo "${IPTABLES} -t nat -D POSTROUTING -s ${LAN_ZONE} -o ${INTERFACE} -j MASQUERADE" >> "${LOGFILE}"
        echo "${IPTABLES} -t nat -D POSTROUTING -s ${GVN_IP} ! -o ${INTERFACE} -j MASQUERADE" >> "${LOGFILE}"
        "${IPTABLES}" -t nat -D POSTROUTING -s "${GVN_IP}" ! -o "${INTERFACE}" -j MASQUERADE
    #fi
elif [ "${OS}" == "Darwin" ]
    export IPOPR="$(which ifconfig)"
    export ROUTE="$(which route)"
    echo "sudo ${IPOPR} $INTERFACE down" >> "${LOGFILE}"
    sudo "${IPOPR}" $INTERFACE down
    echo "sudo ${ROUTE} delete ${GVN_IP}" >> "${LOGFILE}"
    sudo "${ROUTE}" delete "${GVN_IP}"
    for ROU in ${ROUTES}
    do
        echo "sudo ${ROUTE} delete ${ROU}" >> "${LOGFILE}"
        sudo "${ROUTE}" delete "${ROU}"
    done
then
fi`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
			"File":  tmpfile.Name(),
		}).Error("UnloadFirewall - Close tmp file error")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"COMMAND": content,
	}).Debug("Execute command")

	envs := make([]string, 0)
	envs = append(envs, fmt.Sprintf("GVN_VIP=%s", VIP))
	envs = append(envs, fmt.Sprintf("SERVER_VIP=%s", serverVIP))
	envs = append(envs, fmt.Sprintf("MASK=%s", mask))
	envs = append(envs, fmt.Sprintf("ROUTES=%s", strings.Join(dev.Subnets, " ")))
	envs = append(envs, fmt.Sprintf("INTERFACE=%s", dev.Name))
	envs = append(envs, fmt.Sprintf("SERVER_PORT=%d", dev.Port))

	if err := RunCommand(tmpfile.Name(), envs...); err != nil {
		return err
	}
	// remove if success otherwise for debug
	// defer os.Remove(tmpfile.Name())
	return nil
}

// TODO: should provide TUN device name
func AddRoute(subnets []string) error {
	tmpfile, err := ioutil.TempFile("", "AddRoute-*.sh")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File": tmpfile,
		}).Error("AddRoute error when create tmp file")
	}
	logrus.WithFields(logrus.Fields{
		"File": tmpfile.Name(),
	}).Info("shell script to be executed")
	// cidr -> 10.30.20.0/24,172.16.1.1/23 and etc
	// ipv4Addr, ipv4Net, err := net.ParseCIDR(VIP)
	// _, ipv4Net, err := net.ParseCIDR(VIP)
	// if err != nil {
	// 	logrus.WithFields(logrus.Fields{
	// 		"ERROR": err,
	// 	}).Error("Parse VIP error")
	// }
	// vip := ipv4Addr.String()
	// mask := ipv4MaskString(ipv4Net.Mask)
	// content := fmt.Sprintf("netsh interface ip set address name=%s source=static addr=%s mask=%s gateway=%s gwmetric=1", dev.Name, vip, mask, serverVIP.String())
	content := `export LOGFILE="/tmp/AddRoute-$(date +%Y%m%d%H%M%S).log"
export OS="$(uname)"
if [ "${OS}" == "Linux" ]
then
    export IPOPR="$(which ip)"
    export IPTABLES="$(which iptables)"

    # TODO: check whether conflict with LAN
    for ROU in ${ROUTES}
    do
       echo "${IPOPR} route add ${ROU} dev $INTERFACE" >> "${LOGFILE}"
       "${IPOPR}" route add ${ROU} dev $INTERFACE
    done
elif [ "${OS}" == "Darwin" ]
then
    export IPOPR="$(which ifconfig)"
    export ROUTE="$(which route)"
    for ROU in ${ROUTES}
    do
        echo "sudo ${ROUTE} add ${ROU} -interface ${INTERFACE}" >> "${LOGFILE}"
        sudo "${ROUTE}" add ${ROU} -interface "${INTERFACE}"
    done
fi`
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

	envs := make([]string, 0)
	envs = append(envs, fmt.Sprintf("ROUTES=%s", strings.Join(subnets, " ")))
	// TODO:
	envs = append(envs, fmt.Sprintf("INTERFACE=%s", devName))

	if err := RunCommand(tmpfile.Name(), envs...); err != nil {
		return err
	}
	// remove if success otherwise for debug
	// defer os.Remove(tmpfile.Name())
	return nil
}

func RemoveRoute(subnets []string) error {
	tmpfile, err := ioutil.TempFile("", "RemoveRoute-*.sh")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File": tmpfile,
		}).Error("RemoveRoute error when create tmp file")
	}
	logrus.WithFields(logrus.Fields{
		"File": tmpfile.Name(),
	}).Info("shell script to be executed")
	// cidr -> 10.30.20.0/24,172.16.1.1/23 and etc
	// ipv4Addr, ipv4Net, err := net.ParseCIDR(VIP)
	// _, ipv4Net, err := net.ParseCIDR(VIP)
	// if err != nil {
	// 	logrus.WithFields(logrus.Fields{
	// 		"ERROR": err,
	// 	}).Error("Parse VIP error")
	// }
	// vip := ipv4Addr.String()
	// mask := ipv4MaskString(ipv4Net.Mask)
	// content := fmt.Sprintf("netsh interface ip set address name=%s source=static addr=%s mask=%s gateway=%s gwmetric=1", dev.Name, vip, mask, serverVIP.String())
	content := `export LOGFILE="/tmp/RemoveRoute-$(date +%Y%m%d%H%M%S).log"
export OS="$(uname)"
if [ "${OS}" == "Linux" ]
then
    export IPOPR="$(which ip)"
    export IPTABLES="$(which iptables)"

    # TODO: check whether conflict with LAN
    for ROU in ${ROUTES}
    do
       echo "${IPOPR} route del ${ROU}" >> "${LOGFILE}"
       "${IPOPR}" route del ${ROU}
    done
elif [ "${OS}" == "Darwin" ]
then
    export IPOPR="$(which ifconfig)"
    export ROUTE="$(which route)"
    for ROU in ${ROUTES}
    do
        echo "sudo ${ROUTE} delete ${ROU}" >> "${LOGFILE}"
        sudo "${ROUTE}" delete ${ROU} "
    done
fi`
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

	envs := make([]string, 0)
	envs = append(envs, fmt.Sprintf("ROUTES=%s", strings.Join(subnets, " ")))
	// TODO:
	envs = append(envs, fmt.Sprintf("INTERFACE=%s", devName))

	if err := RunCommand(tmpfile.Name(), envs...); err != nil {
		return err
	}
	// remove if success otherwise for debug
	// defer os.Remove(tmpfile.Name())
	return nil
}

func RefreshRoute(subnets []string) {
	RemoveRoute(subnets)
	AddRoute(subnets)
}
