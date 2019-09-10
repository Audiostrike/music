package audiostrike

import (
	"bufio"
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	defaultConfFilename = "austk.conf"
	defaultDbName       = "austk"
	defaultDbUser       = "audiostrike"
	defaultRESTHost     = "localhost"
	defaultRESTPort     = 53545 // 0xd129 from Unicode symbol 0x1d129 for multi-measure rest
	defaultRPCPort      = 53308 // 0xd03c from Unicode symbol 0x1d03c for Byzantine musical symbol rapisma
	defaultArtDirName   = "art"
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
	defaultArtDir = filepath.Join(defaultDir, defaultArtDirName)
)

// Config for austk server.
// These settings are specified in defaults, a config file, or the command line.
// (Config file not yet implemented)
type Config struct {
	ArtistID       string `long:"artist" description:"artist id for publishing tracks"`
	ArtistName string `long:"name" description:"artist name with proper case, punctuation, spacing, etc."`
	ConfigFilename string `long:"config" description:"config file"`
	DbName         string `long:"dbname" description:"mysql db name"`
	DbUser         string `long:"dbuser" description:"mysql db username"`
	DbPassword     string `long:"dbpass" description:"mysql db password"`
	AddMp3Filename string `long:"add" description:"mp3 file to add"`
	ArtDir         string `long:"dir" description:"directory storing music art/artist/album/track"`
	TorProxy       string `long:"torproxy" description:"onion-routing proxy"`
	PeerAddress    string `long:"peer" description:"audiostrike server peer to connect"`
	Pubkey         string `long:"pubkey"`
	RestHost       string `long:"host" description:"ip/tor address for this audiostrike service"`
	RestPort       int    `long:"port" description:"port where audiostrike protocol is exposed"`
	ListenOn       string // ip address and port to listen, e.g. 0.0.0.0:53545
	CertFilePath   string `long:"tlscert" description:"file path for tls cert"`
	MacaroonPath   string `long:"macaroon" description:"file path for macaroon"`
	LndHost        string `long:"lndhost" description:"ip/onion address of lnd"`
	LndGrpcPort    int    `long:"lndport" description:"port where lnd exposes grpc"`

	InitDb      bool `long:"dbinit" description:"initialize the database (first use only)"`
	PlayMp3     bool `long:"play" description:"play imported mp3 file (requires -file)"`
	RunAsDaemon bool `long:"daemon" description:"run as daemon until quit signal (e.g. SIGINT)"`

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

// LoadConfig reads each config value from command line or config file or defaults.
func LoadConfig() (*Config, error) {
	const logPrefix = "config LoadConfig "

	cfg := getDefaultConfig()

	userInputReader := bufio.NewReader(os.Stdin)

	// Parse command line initially to define flags and check for an alternate config file.
	// Then parse the config file, then override any settings with command-line args.
	_, err := flags.Parse(cfg)
	if err != nil {
		isShowingHelp := (err.(*flags.Error).Type == flags.ErrHelp)
		if isShowingHelp {
			return cfg, err
		}
		log.Fatalf(logPrefix+"Error parsing flags: %v\n", err)
	}
	err = flags.IniParse(cfg.ConfigFilename, cfg)
	if err != nil {
		log.Fatalf(logPrefix+"Error parsing config: %v", err)
	}
	flags.Parse(cfg)

	// The artist should configure ArtistId by specifying the `artist` flag in austk.config,
	// or in an alternate config file specified by -config, or by command-line flag `-artist`.
	if cfg.ArtistID == "" {
		// Artist id is not configured or specified so ask the artist for an id.
		fmt.Printf(
			"Please specify your artist id.\n" +
				"Use your public name/identity spelled in lowercase " +
				" with no punctuation or spaces (for example, alicetheartist): ")
		inputArtistID, err := userInputReader.ReadString('\n')
		if err != nil {
			log.Printf(logPrefix+"Error reading ArtistId from stdin: %v", err)
			return cfg, err
		}
		artistID := strings.Replace(inputArtistID, "\n", "", 1)
		artistID = strings.ReplaceAll(artistID, " ", "")
		// TODO: strip other whitespace, punctuation, etc.
		artistID = strings.ToLower(artistID)
		if artistID == "" {
			log.Fatalf(logPrefix + "No artist id. Specify your artist id to publish your music.")
		}
		cfg.ArtistID = artistID
	}

	return cfg, err
}

func getDefaultConfig() *Config {
	return &Config{
		ConfigFilename: defaultConfFilename,
		DbName:         defaultDbName,
		DbUser:         defaultDbUser,
		ArtDir:         defaultArtDir,
		TorProxy:       defaultTorProxy,
		RestHost:       defaultRESTHost,
		RestPort:       defaultRESTPort,
	}
}
