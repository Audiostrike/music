package audiostrike

import (
	//    "fmt"
	"net"
	"os"
	//    "path"
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
)

var (
	defaultDir    = defaultAppDir()
	defaultMp3Dir = filepath.Join(defaultDir, defaultMp3DirName)
)

// Config for art server
type Config struct {
	ArtistId      string
	DbName        string
	DbUser        string
	DbPassword    string
	Mp3Dir        string
	TorProxy      string
	ListenOn      string // ip address and port to listen, e.g. 0.0.0.0:53545
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
	case "windows":
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			appData = os.Getenv("APPDATA")
		}
		if appData != "" {
			return filepath.Join(appData, "AUDIOSTRIKE")
		}

	case "darwin": // MacOS
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
