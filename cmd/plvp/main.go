package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc/grpclog"

	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/plvp"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

var (
	defaultConfig = "./plvp.config"
	configPath    string
	versionNo     string
	vFlag         bool
	pidFile       string
	lockFile      string
)

var conf = plvp.NewConfig()

func init() {
	config.SetEnvPrefix("REVTR")
	if configPath == "" {
		config.AddConfigPath(defaultConfig)
	} else {
		config.AddConfigPath(configPath)
	}
	flag.BoolVar(&vFlag, "version", false,
		"Prints the current version")
	flag.StringVar(conf.Local.Addr, "a", ":65000",
		"The address to run the local service on")
	flag.BoolVar(conf.Local.CloseStdDesc, "d", false,
		"Close std file descripters")
	flag.BoolVar(conf.Local.AutoConnect, "auto-connect", false,
		"Autoconnect to 0.0.0.0 and will use port 55000")
	flag.StringVar(conf.Local.PProfAddr, "pprof-addr", ":55557",
		"The address to use for pperf")
	flag.StringVar(conf.Local.Host, "host", "plcontroller.revtr.ccs.neu.edu",
		"The url for the plcontroller service")
	flag.IntVar(conf.Local.Port, "p", 4380,
		"The port the controller service is listening on")
	flag.BoolVar(conf.Local.StartScamp, "start-scamper", true,
		"Determines if scamper starts or not.")
	flag.StringVar(conf.Scamper.BinPath, "b", "/usr/local/bin/scamper",
		"The path to the scamper binary")
	flag.StringVar(conf.Scamper.Port, "scamper-port", "4381",
		"The port scamper will try to connect to.")
	flag.StringVar(conf.Scamper.Host, "scamper-host", "plcontroller.revtr.ccs.neu.edu",
		"The host that the sc_remoted process is running, should most likely match the host arg")
	grpclog.SetLogger(log.GetLogger())
}

func main() {
	go sigHandle()
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Errorf("Failed to parse config: %v", err)
		exit(1)
	}
	if vFlag {
		fmt.Println(versionNo)
		exit(0)
	}
	_, err = os.Stat(lockFile)
	if err == nil {
		log.Debug("Lockfile exists")
		exit(1)
	} else {
		_, err = os.Create(lockFile)
		if err != nil {
			log.Error(err)
			exit(1)
		}
	}
	util.CloseStdFiles(*conf.Local.CloseStdDesc)
	err = <-plvp.Start(conf)
	if err != nil {
		log.Errorf("PLVP Start returned with error: %v", err)
		exit(1)
	}
}

func exit(status int) {
	os.Remove(pidFile)
	os.Exit(status)
}

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		log.Infof("Got signal: %v", sig)
		os.Remove(lockFile)
		plvp.HandleSig(sig)
		exit(1)
	}
}
