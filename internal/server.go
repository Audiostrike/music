package audiostrike

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"database/sql"
	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
	"log"
)

// ArtServer stores art data for this austk node.
// Implementations may use a database, file system, test fixture, etc.
type ArtServer interface {
	Albums(artistId string) (map[string]art.Album, error)
	Artists() (map[string]art.Artist, error)
	Tracks(artistID string) (map[string]art.Track, error)
	Peer(pubkey string) (*art.Peer, error)
	Peers() ([]*art.Peer, error)
	SetArtist(artist *art.Artist) error
	SetPeer(peer *art.Peer) error
	Track(artistId string, trackId string) (*art.Track, error)
}

// Server hosts the configured artist's art for http/tor clients who might pay the lnd node.
type Server struct {
	config      *Config
	artServer   ArtServer // interface for storing art in a file system, database, or test mock
	httpServer  *http.Server
	lndClient   lnrpc.LightningClient
	lndConn     *grpc.ClientConn
	quitChannel chan bool
}

// NewServer creates a new network Server to serve the configured artist's art.
func NewServer(cfg *Config, db *AustkDb) (*Server, error) {
	const logPrefix = "server NewServer "
	serverNameOverride := ""
	tlsCreds, err := credentials.NewClientTLSFromFile(cfg.CertFilePath, serverNameOverride)
	if err != nil {
		log.Printf(logPrefix+"lnd credentials NewClientTLSFromFile error: %v", err)
		return nil, err
	}

	macaroonData, err := ioutil.ReadFile(cfg.MacaroonPath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile macaroon %v error %v\n", cfg.MacaroonPath, err)
		return nil, err
	}
	mac := &macaroon.Macaroon{}
	err = mac.UnmarshalBinary(macaroonData)
	if err != nil {
		log.Printf(logPrefix+"UnmarchalBinary macaroon error: %v\n", err)
		return nil, err
	}

	lndOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCreds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(mac)),
	}

	lndGrpcEndpoint := fmt.Sprintf("%v:%d", cfg.LndHost, cfg.LndGrpcPort)
	log.Printf(logPrefix+"Dial lnd grpc at %v...", lndGrpcEndpoint)
	lndConn, err := grpc.Dial(lndGrpcEndpoint, lndOpts...)
	if err != nil {
		log.Printf(logPrefix+"Dial lnd error: %v", err)
		return nil, err
	}
	lndClient := lnrpc.NewLightningClient(lndConn)

	ctx := context.Background()
	var signMessageInput lnrpc.SignMessageRequest
	signMessageInput.Msg = []byte("Test message")
	signMessageResult, err := lndClient.SignMessage(ctx, &signMessageInput)
	if err != nil {
		log.Printf(logPrefix+"SignMessage test error: %v", err)
		return nil, err
	}
	log.Printf(logPrefix+"Signed test message, signature: %v", signMessageResult.Signature)

	log.Printf("artDb: %v", db)
	dbServer := newDbServer(db)
	return &Server{
		artServer: dbServer,
		config:    cfg,
		httpServer: &http.Server{
			Addr: "localhost",
		},
		lndConn:     lndConn,
		lndClient:   lndClient,
		quitChannel: make(chan bool),
	}, nil
}

type dbServer struct {
	db *AustkDb
}

func (dbServer dbServer) Albums(artistId string) (map[string]art.Album, error) {
	return dbServer.db.SelectArtistAlbums(artistId)
}

func (dbServer dbServer) Artists() (map[string]art.Artist, error) {
	return dbServer.db.SelectAllArtists()
}

func (dbServer dbServer) Tracks(artistID string) (map[string]art.Track, error) {
	return dbServer.db.SelectArtistTracks(artistID)
}

func (dbServer dbServer) Peer(pubkey string) (*art.Peer, error) {
	return dbServer.db.SelectPeer(pubkey)
}

func (dbServer dbServer) Peers() ([]*art.Peer, error) {
	return dbServer.db.SelectAllPeers()
}

func (dbServer dbServer) SetArtist(artist *art.Artist) error {
	return dbServer.db.PutArtist(artist)
}

func (dbServer dbServer) SetPeer(peer *art.Peer) error {
	return dbServer.db.PutPeer(peer)
}

func (dbServer dbServer) Track(artistId string, trackId string) (*art.Track, error) {
	return dbServer.db.SelectTrack(artistId, trackId)
}

func newDbServer(db *AustkDb) dbServer {
	return dbServer{db: db}
}

// Start the Server listening for REST requests for artists in the db.
func (s *Server) Start() error {
	const logPrefix = "server Start "

	artists, err := s.artServer.Artists()
	if err != nil {
		log.Printf(logPrefix+"db.SelectAllArtists error: %v", err)
		return err
	}
	log.Printf(logPrefix+"%v artists:", len(artists))
	for artistID, artist := range artists {
		log.Printf(logPrefix+"\tArtist id %s: %v", artistID, artist)
		tracks, err := s.artServer.Tracks(artistID)
		if err != nil {
			log.Printf(logPrefix+"db.SelectArtistTracks error: %v", err)
			return err
		}
		for trackID, track := range tracks {
			log.Printf("\t\tTrack id: %s: %v", trackID, track)
		}
	}

	pubkey, err := s.Pubkey()
	if err != nil {
		log.Printf(logPrefix+"Pubkey retrieval error: %v", err)
		return err
	}
	restHost := s.RestHost()
	restPort := s.RestPort()

	selfPeer, err := s.artServer.Peer(pubkey)
	if err == sql.ErrNoRows {
		selfPeer = &art.Peer{Pubkey: pubkey, Host: restHost, Port: restPort}
	} else if err != nil {
		log.Printf(logPrefix+"SelectPeer %v error: %v", pubkey, err)
		return err
	} else {
		if selfPeer.Host != restHost {
			log.Printf(logPrefix+"Update self peer %s host from %s to %s",
				pubkey, selfPeer.Host, restHost)
			selfPeer.Host = restHost
		}
		if selfPeer.Port != restPort {
			log.Printf(logPrefix+"Update self peer %s port from %d to %d",
				pubkey, selfPeer.Port, restPort)
			selfPeer.Port = restPort
		}
	}
	err = s.artServer.SetPeer(selfPeer)
	if err != nil {
		log.Printf(logPrefix+"PutPeer %v error: %v", pubkey, err)
		return err
	}

	// Listen for REST requests and serve in another thread.
	go s.serve()
	log.Printf(logPrefix+"serving REST requests for %s on %s:%d", pubkey, restHost, restPort)

	return err
}

func (server *Server) serve() (err error) {
	const logPrefix = "server serve "
	httpRouter := mux.NewRouter()
	httpRouter.HandleFunc("/artist/{id}", server.putArtistHandler).Methods("PUT")
	httpRouter.HandleFunc("/", server.getAllArtHandler).Methods("GET")
	httpRouter.HandleFunc("/art/{artist:[^/]*}/{track:.*}", server.getArtHandler).Methods("GET")
	restAddress := fmt.Sprintf(":%d", server.config.RestPort)
	err = http.ListenAndServe(restAddress, httpRouter)
	if err != nil {
		log.Printf(logPrefix+"ListenAndServe error: %v", err)
	}

	return
}

func (server *Server) getAllArtHandler(w http.ResponseWriter, req *http.Request) {
	const logPrefix = "server getAllArtHandler "
	// TODO: read any predicates from req to filter results
	// for price (per track, per minute, or per byte),
	// preferred bit rate, or other conditions TBD.
	// Maybe read any follow-back peer URL as well.

	artists, err := server.artServer.Artists()
	if err != nil {
		log.Printf(logPrefix+"SelectAllArtists error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	artistArray := make([]*art.Artist, 0, len(artists))
	trackArray := make([]*art.Track, 0)
	log.Println(logPrefix + "Select all artists:")
	for _, artist := range artists {
		log.Printf("\tArtist: %v", artist)
		artistCopy := artist
		artistArray = append(artistArray, &artistCopy)
		tracks, err := server.artServer.Tracks(artist.ArtistId)
		if err != nil {
			log.Printf(logPrefix+"SelectAllTracks error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, track := range tracks {
			log.Printf("\tTrack: %v", track)
			trackCopy := track
			trackArray = append(trackArray, &trackCopy)
		}
	}
	peers, err := server.artServer.Peers()
	if err != nil {
		log.Printf(logPrefix+"SelectAllPeers error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	peerArray := make([]*art.Peer, 0, len(peers))
	for _, peer := range peers {
		log.Printf("\tPeer: %v", peer)
		peerCopy := *peer
		peerArray = append(peerArray, &peerCopy)
	}
	reply := art.ArtReply{
		Artists: artistArray,
		Tracks:  trackArray,
		Peers:   peerArray,
	}

	ctx := context.Background()
	var signMessageInput lnrpc.SignMessageRequest
	data, err := proto.Marshal(&reply)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	signMessageInput.Msg = []byte(data)
	signMessageResult, err := server.lndClient.SignMessage(ctx, &signMessageInput)
	if err != nil {
		log.Printf(logPrefix+"SignMessage error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf(logPrefix+"Signed message %s, signature: %v", data, signMessageResult.Signature)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (server *Server) Pubkey() (string, error) {
	ctx := context.Background()
	getInfoRequest := lnrpc.GetInfoRequest{}
	getInfoResponse, err := server.lndClient.GetInfo(ctx, &getInfoRequest)
	if err != nil {
		return "", err
	}
	pubkey := getInfoResponse.IdentityPubkey

	return pubkey, nil
}

func (server *Server) RestHost() string {
	return server.config.RestHost
}

func (server *Server) RestPort() uint32 {
	return uint32(server.config.RestPort)
}

// WaitUntilQuitSignal waits for SIGINT (keyboard interrupt Ctrl-C) or for another reason to quit.
func (server *Server) WaitUntilQuitSignal() {
	// Run a new thread to watch for a signal to quit.
	go func() {
		sigintKeyboardInterruptChannel := make(chan os.Signal, 1)

		for {
			select {
			case <-sigintKeyboardInterruptChannel:
				server.quitChannel <- true
			}
		}
	}()

	// Wait until the thread above signals through server.quitChannel
	<-server.quitChannel
}

// Stop the Server
func (server *Server) Stop() error {
	server.quitChannel <- true
	return nil
}

func (server *Server) putArtistHandler(w http.ResponseWriter, req *http.Request) {
	requestVars := mux.Vars(req)
	decoder := json.NewDecoder(req.Body)
	var artist art.Artist
	err := decoder.Decode(&artist)
	if err != nil {
		log.Printf("Failed to decode artist from request, error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		// Should intelligible peers be notified, muted, or disconnected?
		return
	}
	if artist.ArtistId != requestVars["artist"] {
		// Should intelligible peers be notified, muted, or disconnected?
		return
	}
	w.WriteHeader(http.StatusNotImplemented)
	go func() {
		err = server.artServer.SetArtist(&artist)
		if err != nil {
			log.Printf("Failed to put artist into DB, error: %v", err)
		}
	}()
}

func (server *Server) getArtHandler(w http.ResponseWriter, req *http.Request) {
	const logPrefix = "server getArtHandler "

	artistId := mux.Vars(req)["artist"]
	trackId := mux.Vars(req)["track"]
	if artistId == "" || trackId == "" {
		log.Printf(logPrefix+"expected artist and track but received artist: %s, track: %s", artistId, trackId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf(logPrefix+"artist: %v, track: %v", artistId, trackId)

	_, err := server.artServer.Track(artistId, trackId)
	if err != nil {
		// TODO: if requested track isn't in the db, error with 404 Not Found
		log.Printf(logPrefix+"failed to select track, error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: to require payment, check for secret received in receipt for paying invoice
	// If valid secret is not supplied, issue a Lightning invoice.
	// For now, skip the payment mechanics and serve the requested resource immediately.
	mp3, err := OpenMp3ForTrackToRead(artistId, trackId)
	if err != nil {
		log.Printf(logPrefix+"OpenMp3ForTrackToRead %s %s, error: %v", artistId, trackId, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	trackData, err := mp3.ReadBytes()
	if err != nil {
		log.Printf(logPrefix+"mp3.ReadBytes error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf(logPrefix+"serving track as %d bytes of data", len(trackData))
	w.WriteHeader(http.StatusOK)
	w.Write(trackData)
}

// GetArt returns an ArtReply with all known art
func (s *Server) GetArt(context context.Context, artRequest *art.ArtRequest) (artReply *art.ArtReply, err error) {
	artists, err := s.artServer.Artists()
	if err != nil {
		return nil, err
	}
	artistArray := make([]*art.Artist, 0, len(artists))
	albumsArray := make([]*art.Album, 0)
	tracksArray := make([]*art.Track, 0)
	for _, artist := range artists {
		artistArray = append(artistArray, &artist)
		albums, err := s.artServer.Albums(artist.ArtistId)
		if err != nil {
			return nil, err
		}
		for _, album := range albums {
			albumsArray = append(albumsArray, &album)
		}
	}
	peersArray, err := s.artServer.Peers()
	if err != nil {
		return nil, err
	}
	reply := &art.ArtReply{
		Artists: artistArray,
		Albums:  albumsArray,
		Tracks:  tracksArray,
		Peers:   peersArray,
	}
	return reply, err
}
