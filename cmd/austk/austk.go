package main

import (
	"fmt"
	"log"
	"strings"

	audiostrike "github.com/audiostrike/music/internal"
	art "github.com/audiostrike/music/pkg/art"
	"google.golang.org/grpc"
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
		log.Fatalf(logPrefix+"LoadConfig error: %v", err)
	}

	if cfg.InitDb {
		err := audiostrike.InitializeDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
		if err != nil {
			log.Fatalf(logPrefix+"InitializeDb error: %v", err)
		}
	}

	db, err := audiostrike.OpenDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
	if err != nil {
		log.Fatalf(logPrefix+"Failed to open database, error: %v", err)
	}

	if cfg.AddMp3Filename != "" {
		mp3, err := addMp3File(cfg.AddMp3Filename, db)
		if err != nil {
			log.Fatalf(logPrefix+"addMp3File error: %v", err)
		}
		log.Printf(logPrefix+"addMp3File %s ok", cfg.AddMp3Filename)

		if cfg.PlayMp3 {
			mp3.PlayAndWait()
		}
	}

	if cfg.PeerAddress != "" {
		client, err := audiostrike.NewClient(cfg.TorProxy, cfg.PeerAddress)
		if err != nil {
			log.Fatalf(logPrefix+"NewClient via torProxy %v to peerAddress %v, error: %v",
				cfg.TorProxy, cfg.PeerAddress, err)
		}
		defer client.CloseConnection()

		tracks, err := client.SyncFromPeer(db)
		if err != nil {
			log.Fatalf(logPrefix+"SyncFromPeer error: %v", err)
		}

		if cfg.PlayMp3 {
			err = client.DownloadTracks(tracks, db)
			if err != nil {
				log.Fatalf(logPrefix+"DownloadTracks error: %v", err)
			}
			err = playTracks(tracks)
		}
	}

	// TODO: sync with each peer from DB

	if cfg.RunAsDaemon {
		log.Println(logPrefix + "Starting Audiostrike server...")
		server, err := startServer(cfg.ArtistId, db)
		if err != nil {
			log.Fatalf(logPrefix+"startServer daemon error: %v", err)
		}
		defer server.Stop()

		// Execution will stop in this function until server quits from SIGINT etc.
		server.WaitUntilQuitSignal()
	}
}

// playTracks opens the mp3 files of the given tracks, plays each in series, and waits for playback to finish.
// It is used to test mp3 files added for the artist or downloaded from other artists.
func playTracks(tracks []*art.Track) error {
	const logPrefix = "client playTracks "

	for _, track := range tracks {
		mp3, err := audiostrike.OpenMp3ForTrackToRead(track.ArtistId, track.ArtistTrackId)
		if err != nil {
			log.Fatalf(logPrefix+"OpenMp3ToRead %v, error: %v", track, err)
			return err
		}
		mp3.PlayAndWait()
	}
	return nil
}

// addMp3File reads mp3 tags from the file named filename
// and adds/updates a db record for the track, for the artist, and for the album if relevant.
// This lets the austk node host the mp3 track for the artist and collect payments to download/stream it.
func addMp3File(filename string, db *audiostrike.AustkDb) (*audiostrike.Mp3, error) {
	const logPrefix = "austk addMp3File "

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
		err = db.PutAlbum(art.Album{
			ArtistId:      artistID,
			ArtistAlbumId: artistAlbumID,
			Title:         albumTitle,
		})
		artistTrackID = fmt.Sprintf("%v/%v", artistAlbumID, trackTitleID)
	} else {
		artistAlbumID = ""
		artistTrackID = trackTitleID
	}

	artist := &art.Artist{
		ArtistId: artistID,
		Name:     artistName,
	}
	track := &art.Track{
		ArtistId:      artistID,
		ArtistTrackId: artistTrackID,
		Title:         trackTitle,
		ArtistAlbumId: artistAlbumID,
	}
	err = db.AddArtistAndTrack(artist, track)
	if err != nil {
		log.Printf(logPrefix+"AddArtistAndTrack %v %v, error: %v", artist, track, err)
		return nil, err
	}

	err = mp3.SaveForTrack(track.ArtistId, track.ArtistTrackId)
	if err != nil {
		log.Printf(logPrefix+"SaveForTrack %s %s, error: %v",
			track.ArtistId, track.ArtistTrackId, err)
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

// startServer sets the configured artist to use the lnd server and starts running as a daemon
// until SIGINT (ctrl-c or `kill`) is received.
func startServer(artistID string, db *audiostrike.AustkDb) (s *audiostrike.ArtServer, err error) {
	const logPrefix = "austk startServer "

	opts := [...]grpc.ServerOption{}
	s, err = audiostrike.NewServer(opts[:])
	if err != nil {
		log.Printf(logPrefix+"NewServer error: %v", err)
		return
	}

	// Set the pubkey for artistID to this server's pubkey (from lnd).
	pubkey, err := s.Pubkey()
	if err != nil {
		log.Printf(logPrefix+"s.Pubkey error: %v", err)
		return
	}
	err = db.UpdateArtistPubkey(artistID, pubkey)
	if err != nil {
		log.Printf(logPrefix+"db.SetPubkeyForArtist %v, error: %v", artistID, err)
		return
	}

	err = s.Start(db)
	if err != nil {
		log.Printf(logPrefix+"Start error: %v", err)
	}
	return
}
