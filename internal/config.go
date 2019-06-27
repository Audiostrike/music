package audiostrike

import (
	//    "fmt"
	"net"
	"os"
	//    "path"
	"flag"
	"os/user"
	"path/filepath"
	"runtime"
)

const (
	defaultDbName       = "music"
	defaultDbUser       = "audiostrike"
	defaultDbPassword   = "ChangeThisToThePasswordForYourDbUser"
	defaultRESTHost     = "localhost"
	defaultRESTPort     = 53545 // 0xd129 from Unicode symbol 0x1d129 for multi-measure rest
	defaultRPCPort      = 53308 // 0xd03c from Unicode symbol 0x1d03c for Byzantine musical symbol rapisma
	defaultMp3DirName   = "mp3"
	defaultTorProxy     = "socks5://127.0.0.1:9050"
	defaultTLSCertPath  = "./tls.cert"
	defaultMacaroonPath = "./admin.macaroon"
	defaultLndHost      = "127.0.0.1" // "27oxo32rz47oiokfmlnt6ig7qmp6xtq7hgbq67pypfonxs7ubvsualid.onion"
	defaultLndGrpcPort  = 10009

	osMacOS   = "darwin"
	osWindows = "windows"
)

var (
	defaultDir    = defaultAppDir()
	defaultMp3Dir = filepath.Join(defaultDir, defaultMp3DirName)
)

// Config for art server
type Config struct {
	ArtistId       string
	DbName         string
	DbUser         string
	DbPassword     string
	AddMp3FileName string
	Mp3Dir         string
	TorProxy       string
	PeerAddress    string
	ListenOn       string // ip address and port to listen, e.g. 0.0.0.0:53545

	InitDb      bool
	PlayMp3     bool
	RunAsDaemon bool

	Listeners     []net.Addr
	RESTListeners []net.Addr
	RPCListeners  []net.Addr
	ExternalIPs   []net.Addr
}

func defaultAppDir() string {
	var homeDir string
	usr, err := user.Current()
	if err == nil {
		homeDir = usr.HomeDir
	}
	if err != nil || homeDir == "" {
		homeDir = os.Getenv("HOME")
	}
	switch runtime.GOOS {
	case osWindows:
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			appData = os.Getenv("APPDATA")
		}
		if appData != "" {
			return filepath.Join(appData, "AUDIOSTRIKE")
		}

	case osMacOS:
		if homeDir != "" {
			return filepath.Join(homeDir, "Library",
				"Application Support", "Audiostrike")
		}

	default:
		if homeDir != "" {
			return filepath.Join(homeDir, ".audiostrike")
		}
	}

	// Fall back to the current directory if all else fails.
	return "."
}

// LoadConfig will read config values from a file and use defaults for any unset values.
// This proto version just returns the default config.
func LoadConfig() (cfg *Config, err error) {
	cfg, err = getDefaultConfig()
	usr, err := user.Current()
	if err == nil {
		cfg.ArtistId = usr.Username
	}
	var (
		dbNameFlag      = flag.String("dbname", cfg.DbName, "mysql db name")
		dbUserFlag      = flag.String("dbuser", cfg.DbUser, "mysql db username")
		dbPasswordFlag  = flag.String("dbpass", cfg.DbPassword, "mysql db password")
		initDbFlag      = flag.Bool("dbinit", false, "initialize the database (first use only)")
		artistIdFlag    = flag.String("artist", cfg.ArtistId, "artist id for publishing tracks")
		addMp3Flag      = flag.String("add", "", "mp3 file to add, e.g. -add=1.YourTrackToServe.mp3")
		playMp3Flag     = flag.Bool("play", false, "play imported mp3 file (requires -file)")
		runAsDaemonFlag = flag.Bool("daemon", false, "run as daemon until quit signal (e.g. SIGINT)")
		peerFlag        = flag.String("peer", "", "audiostrike server peer to connect")
		torProxyFlag    = flag.String("torproxy", cfg.TorProxy, "onion-routing proxy")
	)
	flag.Parse()

	if *dbNameFlag != "" {
		cfg.DbName = *dbNameFlag
	}
	if *dbUserFlag != "" {
		cfg.DbUser = *dbUserFlag
	}
	if *dbPasswordFlag != "" {
		cfg.DbPassword = *dbPasswordFlag
	}

	if *artistIdFlag != "" {
		cfg.ArtistId = *artistIdFlag
	}

	cfg.InitDb = *initDbFlag
	cfg.AddMp3FileName = *addMp3Flag
	cfg.PlayMp3 = *playMp3Flag
	cfg.PeerAddress = *peerFlag
	cfg.RunAsDaemon = *runAsDaemonFlag
	cfg.TorProxy = *torProxyFlag

	return
}

func getDefaultConfig() (cfg *Config, err error) {
	defaultCfg := Config{
		DbName:     defaultDbName,
		DbUser:     defaultDbUser,
		DbPassword: defaultDbPassword,
		Mp3Dir:     defaultMp3Dir,
		TorProxy:   defaultTorProxy,
	}
	cfg = &defaultCfg
	return
}
