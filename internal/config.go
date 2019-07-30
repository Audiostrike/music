package audiostrike

import (
	"bufio"
	"flag"
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

// Config for art server.
// These settings are specified in defaults, a config file, or the command line.
// (Config file not yet implemented)
type Config struct {
	ArtistId       string `long:"artist"`
	ConfigFilename string `long:"config"`
	DbName         string `long:"dbname"`
	DbUser         string `long:"dbuser"`
	DbPassword     string `long:"dbpass"`
	AddMp3Filename string
	Mp3Dir         string
	TorProxy       string `long:"torproxy"`
	PeerAddress    string
	Pubkey         string `long:"pubkey"`
	RestHost       string `long:"host"`
	RestPort       int    `long:"port"`
	ListenOn       string // ip address and port to listen, e.g. 0.0.0.0:53545
	CertFilePath   string `long:"tlscert"`
	MacaroonPath   string `long:"macaroon"`
	LndHost        string `long:"lndhost"`
	LndGrpcPort    int    `long:"lndport"`

	InitDb      bool
	PlayMp3     bool
	RunAsDaemon bool `long:"daemon"`

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

	var configFilenameFlag = flag.String("config", cfg.ConfigFilename, "config file")
	flag.String("artist", cfg.ArtistId, "artist id for publishing tracks")
	flag.String("dbname", cfg.DbName, "mysql db name")
	flag.String("dbuser", cfg.DbUser, "mysql db username")
	flag.String("dbpass", cfg.DbPassword, "mysql db password")

	flag.Bool("daemon", false, "run as daemon until quit signal (e.g. SIGINT)")
	var peerFlag = flag.String("peer", "", "audiostrike server peer to connect")
	flag.String("torproxy", cfg.TorProxy, "onion-routing proxy")

	flag.String("host", defaultRESTHost, "ip/tor address for this audiostrike service")
	flag.Int("port", defaultRESTPort, "port where audiostrike protocol is exposed")
	flag.String("tlscert", defaultTLSCertPath, "file path for tls cert")
	flag.String("lndhost", defaultLndHost, "ip/onion address of lnd")
	flag.Int("lndport", defaultLndGrpcPort, "port where lnd exposes grpc")
	flag.String("macaroon", defaultMacaroonPath, "file path for macaroon")

	var initDbFlag = flag.Bool("dbinit", false, "initialize the database (first use only)")
	var addMp3Flag = flag.String("add", "", "mp3 file to add, e.g. -add=1.YourTrackToServe.mp3")
	var playMp3Flag = flag.Bool("play", false, "play imported mp3 file (requires -file)")

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
	err = flags.IniParse(*configFilenameFlag, cfg)
	if err != nil {
		log.Fatalf(logPrefix+"Error parsing config: %v", err)
	}
	flags.Parse(cfg)

	// The artist should configure ArtistId by specifying the `artist` flag in austk.config,
	// or in an alternate config file specified by -config, or by command-line flag `-artist`.
	if cfg.ArtistId == "" {
		// Artist id is not configured or specified so ask the artist for an id.
		fmt.Printf(
			"Please specify your artist id.\n" +
				"Use your public name/identity spelled in lowercase " +
				" with no punctuation or spaces (for example, alicetheartist): ")
		inputArtistId, err := userInputReader.ReadString('\n')
		if err != nil {
			log.Printf(logPrefix+"Error reading ArtistId from stdin: %v", err)
			return cfg, err
		}
		artistId := strings.Replace(inputArtistId, "\n", "", 1)
		artistId = strings.ReplaceAll(artistId, " ", "")
		// TODO: strip other whitespace, punctuation, etc.
		artistId = strings.ToLower(artistId)
		if artistId == "" {
			log.Fatalf(logPrefix + "No artist id. Specify your artist id to publish your music.")
		}
		cfg.ArtistId = artistId
	}

	cfg.InitDb = *initDbFlag
	cfg.AddMp3Filename = *addMp3Flag
	cfg.PlayMp3 = *playMp3Flag
	cfg.PeerAddress = *peerFlag

	return cfg, err
}

func getDefaultConfig() *Config {
	return &Config{
		ConfigFilename: defaultConfFilename,
		DbName:         defaultDbName,
		DbUser:         defaultDbUser,
		Mp3Dir:         defaultMp3Dir,
		TorProxy:       defaultTorProxy,
	}
}
