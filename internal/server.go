package audiostrike

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
	"log"
	"errors"
)

// ArtServer is a repository to store/serve music and related data for this austk node.
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

var (
	ErrArtNotFound = errors.New("ArtServer has no such art")
	ErrPeerNotFound = errors.New("AustkServer has no such peer")
)

// Server hosts the configured artist's art for http/tor clients who might pay the lnd node.
type AustkServer struct {
	config      *Config
	artServer   ArtServer // interface for storing art in a file system, database, or test mock
	httpServer  *http.Server
	lndClient   lnrpc.LightningClient
	lndConn     *grpc.ClientConn
	quitChannel chan bool
}

// NewAustkServer creates a new network Server to serve the configured artist's art.
func NewAustkServer(cfg *Config, artServer ArtServer) (*AustkServer, error) {
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

	return &AustkServer{
		artServer: artServer,
		config:    cfg,
		httpServer: &http.Server{
			Addr: "localhost",
		},
		lndConn:     lndConn,
		lndClient:   lndClient,
		quitChannel: make(chan bool),
	}, nil
}

// Start the AustkServer listening for REST austk requests for art.
func (s *AustkServer) Start() error {
	const logPrefix = "AustkServer Start "

	artists, err := s.artServer.Artists()
	if err != nil {
		log.Printf(logPrefix+"artServer.Artists error: %v", err)
		return err
	}
	log.Printf(logPrefix+"%v artists:", len(artists))
	for artistID, artist := range artists {
		log.Printf(logPrefix+"\tArtist id %s: %v", artistID, artist)
		tracks, err := s.artServer.Tracks(artistID)
		if err != nil {
			log.Printf(logPrefix+"artServer.Tracks error: %v", err)
			return err
		}
		for trackID, track := range tracks {
			log.Printf("\t\tTrack id: %s: %v", trackID, track)
		}
	}

	// Publish this austk node as the Peer with this server's Pubkey.
	pubkey, err := s.Pubkey()
	if err != nil {
		log.Printf(logPrefix+"Pubkey retrieval error: %v", err)
		return err
	}
	restHost := s.RestHost()
	restPort := s.RestPort()

	selfPeer, err := s.artServer.Peer(pubkey)
	if err == ErrPeerNotFound {
		selfPeer = &art.Peer{Pubkey: pubkey, Host: restHost, Port: restPort}
		log.Printf(logPrefix+"insert selfPeer: %v", selfPeer)
	} else if err != nil {
		log.Printf(logPrefix+"artServer.Peer(%v) error: %v", pubkey, err)
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

// serve starts listening for and handling requests to austk endpoints.
func (server *AustkServer) serve() (err error) {
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

// getAllArtHandler handles a request to get all the art from the ArtService.
func (server *AustkServer) getAllArtHandler(w http.ResponseWriter, req *http.Request) {
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
			log.Printf(logPrefix+"artServer.Tracks error: %v", err)
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

// Pubkey returns the pubkey for the lnd server,
// which clients can use to authenticate publications from this node.
func (server *AustkServer) Pubkey() (string, error) {
	ctx := context.Background()
	getInfoRequest := lnrpc.GetInfoRequest{}
	getInfoResponse, err := server.lndClient.GetInfo(ctx, &getInfoRequest)
	if err != nil {
		return "", err
	}
	pubkey := getInfoResponse.IdentityPubkey

	return pubkey, nil
}

// RestHost returns the tor address or ip address of this austk node.
func (server *AustkServer) RestHost() string {
	return server.config.RestHost
}

// RestPort returns the tcp/ip port where this austk node listens for requests and serves art.
func (server *AustkServer) RestPort() uint32 {
	return uint32(server.config.RestPort)
}

// WaitUntilQuitSignal waits for SIGINT (keyboard interrupt Ctrl-C) or for another reason to quit.
func (server *AustkServer) WaitUntilQuitSignal() {
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
func (server *AustkServer) Stop() error {
	server.quitChannel <- true
	return nil
}

// putArtistHandler handles a request to put an artist into the ArtService.
func (server *AustkServer) putArtistHandler(w http.ResponseWriter, req *http.Request) {
	const logPrefix = "AustkServer putArtistHandler "
	
	requestVars := mux.Vars(req)
	decoder := json.NewDecoder(req.Body)
	var artist art.Artist
	err := decoder.Decode(&artist)
	if err != nil {
		log.Printf(logPrefix+"Failed to decode artist from request, error: %v", err)
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
			log.Printf(logPrefix+"Failed to put artist, error: %v", err)
		}
	}()
}

// getArtHandler handles requests to get a specified track by a specified artist.
func (server *AustkServer) getArtHandler(w http.ResponseWriter, req *http.Request) {
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
		// TODO: if requested track isn't found, error with 404 Not Found
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
