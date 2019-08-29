package main

import (
	"fmt"
	"log"
	"strings"

	audiostrike "github.com/audiostrike/music/internal"
	art "github.com/audiostrike/music/pkg/art"
	flags "github.com/jessevdk/go-flags"
	"path/filepath"
)

// main runs austk with config from command line, austk.config file, or defaults. `-help` for help:
//
//     go/src/github.com/audiostrike/music$ ./austk -help
//
// Setup your computer to run `austk` to serve music with the steps at
// https://github.com/audiostrike/music/wiki/austk-node-setup
// bitcoind may take several days for initial block download to sync to bitcoin mainnet blockchain.
//
// Use `-artist {id}` to set the id as a simple lower-case name, no spaces or punctuation:
//
//     go/src/github.com/audiostrike/music$ ./austk -artist aliceinchains
//
// The node setup steps create a mysql db user for `austk` to use.
// Specify that mysql username with `-dbuser {username}` and password with `-dbpass {password}`.
// On first run, also initialize the database with `-dbinit`:
//
//     go/src/github.com/audiostrike/music$ ./austk -artist aliceinchains
//     -dbuser examplemysqlusername -dbpass 3x4mpl3mysqlp455w0rd -dbinit
//
// Add mp3 files to the art directory with `-add {filepath}`:
//
//     go/src/github.com/audiostrike/music$ ./austk -artist aliceinchains
//     -add /media/recordings/dirt/would.mp3
//
// To serve added tracks, run as a daemon with the `-daemon` flag.
// Publish your austk node's tor address with `-host {address}`.
// Connect securely with your `lnd` through `-macaroon` and `-tlscert`.
//
//     go/src/github.com/audiostrike/music$ ./austk -artist aliceinchains
//     -dbuser examplemysqlusername -dbpass 3x4mpl3mysqlp455w0rd
//     -macaroon ~/.lnd/data/chain/bitcoin/mainnet/admin.macaroon -tlscert ~/.lnd/tls.cert
//     -host 45o4k7vt75tgh4zwbkxl5ec6ccagaulr273piugh3tt2cfmcawzeiwqd.onion -daemon
//
func main() {
	const logPrefix = "austk main "

	cfg, err := audiostrike.LoadConfig()
	if err != nil {
		isShowingHelp := (err.(*flags.Error).Type == flags.ErrHelp)
		if isShowingHelp {
			return
		}
		log.Fatalf(logPrefix+"LoadConfig error: %v", err)
	}

	if cfg.InitDb {
		err := audiostrike.InitializeDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
		if err != nil {
			log.Fatalf(logPrefix+"InitializeDb error: %v", err)
		}
	}

	localStorage, err := audiostrike.NewFileServer(cfg.ArtDir)
	if err != nil {
		log.Fatalf(logPrefix+"Failed to open database, error: %v", err)
	}

	if cfg.PeerAddress != "" {
		client, err := audiostrike.NewClient(cfg.TorProxy, cfg.PeerAddress)
		if err != nil {
			log.Fatalf(logPrefix+"NewClient via torProxy %v to peerAddress %v, error: %v",
				cfg.TorProxy, cfg.PeerAddress, err)
		}

		_, err = client.SyncFromPeer(localStorage)
		if err != nil {
			log.Fatalf(logPrefix+"SyncFromPeer error: %v", err)
		}

		client.CloseConnection()
	}

	austkServer, err := injectLnd(cfg, localStorage)
	if err != nil {
		if cfg.AddMp3Filename != "" || cfg.RunAsDaemon {
			log.Fatalf(logPrefix+"Failed to connect to lightning network, error: %v", err)
		} else {
			log.Printf(logPrefix+"failed to  connect to lightning network, error: %v", err)
		}
	}

	if cfg.AddMp3Filename != "" {
		mp3, err := storeMp3File(cfg.AddMp3Filename, localStorage, austkServer)
		if err != nil {
			log.Fatalf(logPrefix+"storeMp3File error: %v", err)
		}
		log.Printf(logPrefix+"storeMp3File %s ok", cfg.AddMp3Filename)

		if cfg.PlayMp3 {
			mp3.PlayAndWait()
		}
	}

	if cfg.RunAsDaemon {
		log.Println(logPrefix + "Starting Audiostrike server...")
		err = startServer(cfg, localStorage, austkServer)
		if err != nil {
			log.Fatalf(logPrefix+"startServer daemon error: %v", err)
		}
		defer austkServer.Stop()

		cfg.Pubkey, err = austkServer.Pubkey()
		if err != nil {
			log.Fatalf(logPrefix+"error getting server pubkey: %v", err)
		}
	}

	peers, err := localStorage.Peers()
	if err != nil {
		log.Printf(logPrefix+"failed to get Peers from localStorage, error: %v", err)
	}
	for _, peer := range peers {
		if peer.Pubkey == cfg.Pubkey {
			log.Printf(logPrefix+"skip sync from self pubkey %v", peer)
			continue // to next peer since we sync'ed this one above.
			// Could simplify by adding entry above and sync'ing here.
		}
		log.Printf(logPrefix+"sync from peer %v", peer)
		peerAddress := fmt.Sprintf("%s:%d", peer.Host, peer.Port)
		client, err := audiostrike.NewClient(cfg.TorProxy, peerAddress)
		if err != nil {
			log.Fatalf(logPrefix+"NewClient via torProxy %v to peerAddress %v, error: %v",
				cfg.TorProxy, peer.Host, err)
		}

		resources, err := client.SyncFromPeer(localStorage)
		if err != nil {
			// Log misbehaving peer but continue with other peers.
			log.Printf(logPrefix+"SyncFromPeer error: %v", err)
			continue
		}
		tracks := resources.Tracks

		if cfg.PlayMp3 {
			log.Printf("playing tracks...")
			err = client.DownloadTracks(tracks, localStorage)
			if err != nil {
				log.Fatalf(logPrefix+"DownloadTracks error: %v", err)
			}
			err = playTracks(tracks, cfg.ArtDir)
		} else {
			log.Printf("will not play tracks")
		}

		client.CloseConnection()
	}

	if cfg.RunAsDaemon {
		// Execution will stop in this function until server quits from SIGINT etc.
		austkServer.WaitUntilQuitSignal()
	}
}

// playTracks opens the mp3 files of the given tracks, plays each in series, and waits for playback to finish.
// It is used to test mp3 files added for the artist or downloaded from other artists.
func playTracks(tracks []*art.Track, rootDirPath string) error {
	const logPrefix = "client playTracks "

	for _, track := range tracks {
		mp3FilePath := filepath.Join(rootDirPath, track.ArtistId, track.ArtistTrackId+".mp3")
		mp3, err := audiostrike.OpenMp3ToRead(mp3FilePath)
		if err != nil {
			log.Fatalf(logPrefix+"OpenMp3ToRead %v, error: %v", track, err)
			return err
		}
		mp3.PlayAndWait()
	}
	return nil
}

// storeMp3File reads mp3 tags from the file named filename
// and stores an art record for the track, for the artist, and for the album if relevant.
// This lets the austk node host the mp3 track for the artist and collect payments to download/stream it.
func storeMp3File(filename string, localStorage audiostrike.ArtServer, publisher audiostrike.Publisher) (*audiostrike.Mp3, error) {
	const logPrefix = "austk storeMp3File "

	mp3, err := audiostrike.OpenMp3ToRead(filename)
	if err != nil {
		return nil, err
	}

	artistName := mp3.ArtistName()
	artistID := nameToId(artistName)

	var artistTrackID string
	trackTitle := mp3.Title()

	albumTitle, isInAlbum := mp3.AlbumTitle()
	var artistAlbumID string
	trackTitleID := nameToId(trackTitle)
	log.Printf(logPrefix+"file: %v\n\tTitle: %v\n\tArtist: %v\n\tAlbum: %v\n\tTags: %v",
		filename, trackTitle, artistName, albumTitle, mp3.Tags)
	if isInAlbum {
		artistAlbumID = nameToId(albumTitle)
		err = localStorage.StoreAlbum(&art.Album{
			ArtistId:      artistID,
			ArtistAlbumId: artistAlbumID,
			Title:         albumTitle,
		})
		artistTrackID = fmt.Sprintf("%v/%v", artistAlbumID, trackTitleID)
	} else {
		artistAlbumID = ""
		artistTrackID = trackTitleID
	}

	// Store the artist and the track
	artist := &art.Artist{
		ArtistId: artistID,
		Name:     artistName,
	}
	err = localStorage.StoreArtist(artist, publisher)
	if err != nil {
		log.Printf(logPrefix+"StoreArtist %v, error: %v", artist, err)
		return nil, err
	}
	track := &art.Track{
		ArtistId:      artistID,
		ArtistTrackId: artistTrackID,
		Title:         trackTitle,
		ArtistAlbumId: artistAlbumID,
	}
	err = localStorage.StoreTrack(track, publisher)
	if err != nil {
		log.Printf(logPrefix+"StoreTrack %v %v, error: %v", track, err)
		return nil, err
	}

	trackPayload, err := mp3.ReadBytes()
	if err != nil {
		log.Printf(logPrefix+"ReadBytes error: %v", err)
		return nil, err
	}

	err = localStorage.StoreTrackPayload(track, trackPayload)
	if err != nil {
		log.Printf(logPrefix+"StoreTrackPayload for %s/%s with %d bytes, error: %v",
			track.ArtistId, track.ArtistTrackId, len(trackPayload), err)
		return nil, err
	}

	return mp3, nil
}

// nameToId converts the name or title of an artist, album, or track
// into a case-insensitive id usable for urls, filenames, etc.
func nameToId(name string) string {
	// TODO: strip other whitespace, punctuation, etc.
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}

// startServer sets the configured artist to use the configured lnd for signing and selling music.
// and starts running as a daemon
// until SIGINT (ctrl-c or `kill`) is received.
func startServer(cfg *audiostrike.Config, localStorage audiostrike.ArtServer, austkServer *audiostrike.AustkServer) error {
	const logPrefix = "austk startServer "

	// Set the pubkey for artistID to this server's pubkey (from lnd).
	pubkey, err := austkServer.Pubkey()
	if err != nil {
		log.Fatalf(logPrefix+"s.Pubkey error: %v", err)
		return err
	}
	artist, err := localStorage.Artist(cfg.ArtistId)
	if err != nil {
		log.Fatalf(logPrefix+"Failed to get artist %s from local storage, error: %v",
			cfg.ArtistId, err)
		return err
	}
	artist.Pubkey = pubkey
	err = localStorage.StoreArtist(artist, austkServer)
	if err != nil {
		log.Fatalf(logPrefix+"StoreArtist %v, error: %v", artist, err)
		return err
	}

	err = austkServer.Start()
	if err != nil {
		log.Fatalf(logPrefix+"Start error: %v", err)
		return err
	}
	return nil
}
