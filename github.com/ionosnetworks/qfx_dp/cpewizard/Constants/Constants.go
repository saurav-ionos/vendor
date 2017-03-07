package Constants

var LogCmds = []string{
	"link",
	"addr",
	"rule",
	"route",
}
var EthernetMapping map[string]string
var NonEditableLogicalEth string
var NonEditablePhysicalEth string
var CurrentLogSoftLink string

const (
	Eth0                   = "eth0"
	Eth1                   = "eth1"
	Eth2                   = "eth2"
	Eth3                   = "eth3"
	Eth4                   = "eth4"
	Eth5                   = "eth5"
	PingResponseCount      = "5"
	PingCheckIP            = "8.8.8.8"
	IPDhcp                 = "dhcp"
	IPStatic               = "static"
	NetWorkAddr            = "golang.org"
	NetWorkPort            = "80"
	PingCommand            = "ping"
	IPCommand              = "ip"
	EthToolCommand         = "ethtool"
	StateUP                = "UP"
	StateDOWN              = "DOWN"
	LanStatusUPString      = "state UP"
	LanStatusDownString    = "state DOWN"
	CpeIdFile              = "/etc/ionos-cpeid.conf"
	CpeVersionFile         = "/etc/ionos-cpe-version.conf"
	PingHostUnreachable    = "Destination Host Unreachable"
	PingNetworkUnreachable = "Network is unreachable"
	ConfigFile             = "./config.json"
	InterfaceFileFolder    = "/etc/network/interfaces.d"
	InterfaceBkpFolder     = "/var/ionos/network"
	//LogFileSoftLink        = "/var/www/cpewizard/logs.tar.gz"
	LogFileSoftLink = "/var/www/cpewizard/ionos-"
	LogFileLocation = "/var/log/ionos/log.tar.gz"
	//LogFileLocation = "/var/log/ionos/ionos.logs"
	LogFileName       = "logs.tar.gz"
	CmdsLogFile       = "/tmp/cmds.log"
	LogFileLevel      = "Debug"
	DataBaseLoc       = "DataBase/DBFiles"
	TokenDur          = 1800
	TimeFormat        = "Mon Jan 2 15:04:05 IST 2016"
	EthMapFile        = "/var/ionos/eth_to_port_mapping"
	LogInfoFile       = "/var/ionos/cpelogfiles"
	UserConfigFile    = "/var/ionos/userconf"
	SoftLinkFolder    = "/var/www/cpewizard"
	DnsResolvHost     = "ionos.com"
	MultiLinkFile     = "/var/ionos/ifc-mp.conf"
	MultiLinkModified = "/var/ionos/ifc-mp.conf.modified"
	ReStartFwdScript  = "/var/ionos/restart_fwder.sh"
	CpeRootDir        = "/usr/local/ica"
)
