package tun

var (
	VIP string
)

func ConfigAddr(dev Device) error {
	return nil
}

func UnloadFirewall(dev Device) error {
	// TODO:
	return nil
}

func UnloadFirewall(dev Device) error {
	// TODO:
	return nil
}

func RemoveRoute(subnets []string, vip string) {

}

func AddRoute(subnets []string, vip string) {

}

func RefreshRoute(subnets []string) {
	RemoveRoute(subnets)
	AddRoute(subnets)
}
