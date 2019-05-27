package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	audiostrike "github.com/audiostrike/music/internal"
	art "github.com/audiostrike/music/pkg/art"
	"google.golang.org/grpc"
)

var (
	artistId       string
	initDb         bool
	addMp3FileName string
	playMp3        bool
	runAsDaemon    bool
	peerAddress    string
	torProxy       string
)

func main() {
	const logPrefix = "austk main "
	cfg, err := audiostrike.LoadConfig()
	if err != nil {
		log.Fatalf(logPrefix+"LoadConfig error: %v", err)
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
	cfg.DbUser = *dbUserFlag
	cfg.DbPassword = *dbPasswordFlag
	initDb = *initDbFlag
	artistId = *artistIdFlag
	addMp3FileName = *addMp3Flag
	playMp3 = *playMp3Flag
	peerAddress = *peerFlag
	runAsDaemon = *runAsDaemonFlag
	torProxy = *torProxyFlag

	if initDb {
		err := audiostrike.InitializeDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
		if err != nil {
			log.Fatalf(logPrefix+"InitializeDb error: %v", err)
		}
	}

	db, err := audiostrike.OpenDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
	if err != nil {
		log.Fatalf(logPrefix+"Failed to open database, error: %v", err)
	}

	if addMp3FileName != "" {
		mp3, err := addMp3File(addMp3FileName, db)
		if err != nil {
			log.Fatalf(logPrefix+"addMp3File error: %v", err)
		}
		fmt.Printf(logPrefix+"addMp3File %s ok\n", addMp3FileName)

		if playMp3 {
			mp3.PlayAndWait()
		}
	}

	if peerAddress != "" {
		client, err := audiostrike.NewClient(torProxy, peerAddress)
		if err != nil {
			log.Fatalf(logPrefix+"NewClient via torProxy %v to peerAddress %v, error: %v",
				torProxy, peerAddress, err)
		}
		defer client.CloseConnection()
		reply, err := client.GetAllArtByTor() //GetAllArtByGrpc()
		if err != nil {
			log.Fatalf(logPrefix+"GetAllArt from %v error: %v", peerAddress, err)
		}
		fmt.Printf("Received reply: %v", reply)
		err = importArtReply(reply, db, client)
		if err != nil {
			log.Fatalf(logPrefix+"importArtReply error: %v", err)
		}
	}

	if runAsDaemon {
		fmt.Println(logPrefix + "Starting Audiostrike server...")
		server, err := startServer(artistId, db)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer server.Stop()

		// Execution will stop in this function until server quits from SIGINT etc.
		server.WaitUntilQuitSignal()
	}
}

func importArtReply(artReply *art.ArtReply, db *audiostrike.AustkDb, client *audiostrike.Client) (err error) {
	const logPrefix = "austk importArtReply "
	var errors []error
	artists := make(map[string]art.Artist)
	for _, artist := range artReply.Artists {
		artists[artist.ArtistId] = *artist
		err = db.PutArtist(artist)
		if err != nil {
			fmt.Fprintf(os.Stderr, logPrefix+"db.PutArtist error: %v\n", err)
			errors = append(errors, err)
		}
	}
	for _, track := range artReply.Tracks {
		_, err = db.PutTrack(*track)
		if err != nil {
			fmt.Fprintf(os.Stderr, logPrefix+"db.PutTrack error: %v\n", err)
			errors = append(errors, err)
			continue
		}

		if playMp3 {
			var peer *art.Peer
			trackArtist, err := db.SelectArtist(track.ArtistId)
			if err != nil {
				fmt.Fprintf(os.Stderr, logPrefix+"db.SelectArtist error: %v\n", err)
				errors = append(errors, err)
				continue
			}
			for i := range artReply.Peers {
				fmt.Printf(logPrefix+"compare peer %v with track artist pubkey %v\n",
					artReply.Peers[i].Pubkey, trackArtist.Pubkey)
				if artReply.Peers[i].Pubkey == trackArtist.Pubkey {
					peer = artReply.Peers[i]
					fmt.Printf(logPrefix+
						"Peer %v with pubkey %v matches artist %v\n",
						peer.Host, peer.Pubkey, trackArtist.ArtistId)
					break
				}
			}
			if peer == nil {
				errors = append(errors,
					fmt.Errorf("no peer owns remote .mp3 %s/%s",
						track.ArtistId, track.ArtistTrackId))
				continue
			}
			// TODO: sanitize filepath so peer cannot write outside ./tracks/ dir sandbox.
			filename := fmt.Sprintf("./tracks/%s/%s", track.ArtistId, track.ArtistTrackId)
			replyBytes, err := client.GetArtByTor(track.ArtistId, track.ArtistTrackId)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					logPrefix+"GetArtByTor %v/%v, error: %v\n",
					track.ArtistId, track.ArtistTrackId, err)
				errors = append(errors, err)
				continue
			}
			dirname := fmt.Sprintf("./tracks")
			err = os.Mkdir(dirname, 0777)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					logPrefix+"Mkdir ./tracks error: %v\n", err)
			}
			dirname = fmt.Sprintf("./tracks/%s", track.ArtistId)
			err = os.Mkdir(dirname, 0777)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					logPrefix+"Mkdir ./tracks/%s error: %v\n",
					track.ArtistId, err)
			}
			if track.ArtistAlbumId != "" {
				dirname = fmt.Sprintf("./tracks/%s/%s", track.ArtistId, track.ArtistAlbumId)
				err = os.Mkdir(dirname, 0777)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						logPrefix+"Mkdir ./tracks/%s/%s error: %v\n",
						track.ArtistId, track.ArtistAlbumId, err)
				}
			}
			err = ioutil.WriteFile(filename, replyBytes, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, logPrefix+"WriteFile error: %v\n", err)
				errors = append(errors, err)
				continue
			}
			// TODO: play the .mp3 file
			errors = append(errors, fmt.Errorf("not yet implemented to play remote .mp3 file"))
		}
	}
	if len(errors) > 0 {
		// return the first error
		err = errors[0]
	}
	return
}

func addMp3File(addMp3FileName string, db *audiostrike.AustkDb) (mp3 *audiostrike.Mp3, err error) {
	const logPrefix = "austk importMp3File "
	mp3, err = audiostrike.OpenMp3ToRead(addMp3FileName)
	if err != nil {
		return
	}

	artistName := mp3.ArtistName()
	artistID := nameToId(artistName)
	var artistPubkey string
	dbArtist, err := db.SelectArtist(artistID)
	if err == sql.ErrNoRows {
		artistPubkey = ""
	} else if err != nil {
		return
	} else {
		artistPubkey = dbArtist.Pubkey
	}
	trackTitle := mp3.Title()
	albumTitle, isInAlbum := mp3.AlbumTitle()
	fmt.Printf(logPrefix+"file: %v\n\tTitle: %v\n\tArtist: %v\n\tAlbum: %v\n\tTags: %v\n",
		addMp3FileName, trackTitle, artistName, albumTitle, mp3.Tags)
	err = db.PutArtist(&art.Artist{
		ArtistId: artistID,
		Name:     artistName,
		Pubkey:   artistPubkey,
	})
	if err != nil {
		return
	}

	var artistTrackID string
	var artistAlbumID string
	trackTitleID := nameToId(trackTitle)
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
	_, err = db.PutTrack(art.Track{
		ArtistId:      artistID,
		ArtistTrackId: artistTrackID,
		Title:         trackTitle,
		ArtistAlbumId: artistAlbumID,
	})
	return
}

func nameToId(name string) string {
	// TODO: strip other whitespace, punctuation, etc.
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}

func startServer(artistID string, db *audiostrike.AustkDb) (s *audiostrike.ArtServer, err error) {
	const logPrefix = "austk startServer "
	opts := [...]grpc.ServerOption{}

	s, err = audiostrike.NewServer(opts[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"NewServer error: %v\n", err)
		return
	}
	fmt.Printf(logPrefix+"select artist %v\n", artistID)
	artist, err := db.SelectArtist(artistID)
	if err != nil {
		return
	}
	artist.Pubkey, err = s.Pubkey()
	if err != nil {
		return
	}
	err = db.PutArtist(artist)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"PutArtist Pubkey %v, error: %v\n", artist.Pubkey, err)
		return
	}
	fmt.Printf(logPrefix+"PutArtist Pubkey %v ok\n", artist.Pubkey)
	err = s.Start(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"Start error: %v\n", err)
	}
	return
}
