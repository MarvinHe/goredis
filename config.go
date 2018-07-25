/**
 * go redis config file
 */
package go_redis

type clientConfig struct {
	maxClients  int
	timeout	    int
	tcpKeepalive int
	clientOutputBufferLimit string
}

type slaveConfig struct {

}

type replicationConfig struct {

}

type Config struct {
	host 	string
	port 	int

	dir     string
	rdbFile string
	logFile string
	logLevel string
	pidFile string
	aofFile string
	appendonly bool

	maxMemory int
	maxMemoryPolicy string
	savePolicy []int
	slowlogTimelimit int
	slowlogMaxLen int

	// crontab
	hz        int

	clientConfig
	slaveConfig
}
