package audiostrike

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"errors"
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

// ArtServer is a repository to store/serve music and related data for this austk node.
// Implementations may use a database, file system, test fixture, etc.
type ArtServer interface {
	// Get and store artist info.
	StoreArtist(artist *art.Artist, publisher Publisher) (*art.ArtResources, error)
	Artists() (map[string]*art.Artist, error)
	Artist(artistId string) (*art.Artist, error)

	// Album: an artist's optional track container to name and sequence tracks
	StoreAlbum(album *art.Album) error
	Albums(artistId string) (map[string]*art.Album, error)

	// Get and store Track info.
	StoreTrack(track *art.Track, publisher Publisher) error
	StoreTrackPayload(track *art.Track, bytes []byte) error
	Tracks(artistID string) (map[string]*art.Track, error)
	Track(artistID string, artistTrackID string) (*art.Track, error)
	TrackFilePath(track *art.Track) string

	// Get and store network info.
	StorePeer(peer *art.Peer, publisher Publisher) error
	Peers() (map[string]*art.Peer, error)
	Peer(pubkey string) (*art.Peer, error)
}

type Publisher interface {
	Artist() (*art.Artist, error)
	Sign(*art.ArtResources) (*art.ArtistPublication, error)
	Verify(*art.ArtistPublication) (*art.ArtResources, error)
	VerifyArtist(*art.Artist) error
}

// Artist gets the Artist publishing interface
func (server *AustkServer) Artist() (*art.Artist, error) {
	return server.artServer.Artist(server.config.ArtistId)
}

func (server *AustkServer) Sign(resources *art.ArtResources) (*art.ArtistPublication, error) {
	const logPrefix = "AustkServer Sign "

	log.Printf(logPrefix + "sign %v", resources)
	ctx := context.Background()
	serializedResources, err := proto.Marshal(resources)
	if err != nil {
		return nil, err
	}
	signMessageInput := lnrpc.SignMessageRequest{Msg: serializedResources}
	signMessageResponse, err := server.lightningClient.SignMessage(ctx, &signMessageInput)
	if err != nil {
		log.Fatalf(logPrefix+"lnd is not operational. SignMessage error: %v", err)
		return nil, err
	}
	publishingArtist, err := server.Artist()
	if err != nil {
		log.Fatalf(logPrefix+"failed to get publishing artist, error: %v", err)
		return nil, err
	}
	
	return &art.ArtistPublication{
		Artist: publishingArtist,
		Signature: signMessageResponse.Signature,
		SerializedArtResources: serializedResources,
	}, nil
}

func (server *AustkServer) Verify(*art.ArtistPublication) (*art.ArtResources, error) {
	return nil, fmt.Errorf("Verify not implemented")
}

func (server *AustkServer) VerifyArtist(artist *art.Artist) error {
	if proto.Equal(artist, server.publishingArtist) {
		return nil
	} else {
		return ErrArtNotFound
	}
}

var (
	ErrArtNotFound  = errors.New("ArtServer has no such art")
	ErrPeerNotFound = errors.New("AustkServer has no such peer")
)

// Server hosts the configured artist's art for http/tor clients who might pay the lnd node.
type AustkServer struct {
	publishingArtist *art.Artist
	config           *Config
	artServer        ArtServer // interface for storing art in a file system, database, or test mock
	httpServer       *http.Server
	lightningClient  lnrpc.LightningClient
	quitChannel      chan bool
}

// NewAustkServer creates a new network Server to serve the configured artist's art.
func NewAustkServer(cfg *Config, artServer ArtServer, lightningClient lnrpc.LightningClient) (*AustkServer, error) {
	const logPrefix = "server NewServer "

	// Test lightningClient to ensure that we can sign messages with it.
	ctx := context.Background()
	var signMessageInput lnrpc.SignMessageRequest
	signMessageInput.Msg = []byte("Test message to ensure lnd is operational")
	_, err := lightningClient.SignMessage(ctx, &signMessageInput)
	if err != nil {
		log.Fatalf(logPrefix+"lnd is not operational. SignMessage error: %v", err)
		return nil, err
	}

	return &AustkServer{
		artServer:       artServer,
		config:          cfg,
		httpServer:      &http.Server{Addr: "localhost"},
		lightningClient: lightningClient,
		quitChannel:     make(chan bool),
	}, nil
}

func NewLightningClient(cfg *Config) (lnrpc.LightningClient, error) {
	const logPrefix = "server newLndClient "

	// Get the TLS credentials for the lnd server.
	// The second paramater here is serverNameOverride, set to ""
	// except to override the virtual host name of authority in test requests.
	lndTlsCreds, err := credentials.NewClientTLSFromFile(cfg.CertFilePath, "")
	if err != nil {
		log.Printf(logPrefix+"lnd credentials NewClientTLSFromFile error: %v", err)
		return nil, err
	}

	// Get the macaroon for lnd grpc requests.
	// This macaroon must should support creating invoices and signing messages.
	macaroonData, err := ioutil.ReadFile(cfg.MacaroonPath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile macaroon %v error %v\n", cfg.MacaroonPath, err)
		return nil, err
	}
	lndMacaroon := &macaroon.Macaroon{}
	err = lndMacaroon.UnmarshalBinary(macaroonData)
	if err != nil {
		log.Printf(logPrefix+"UnmarchalBinary macaroon error: %v\n", err)
		return nil, err
	}

	lndOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(lndTlsCreds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(lndMacaroon)),
	}

	lndGrpcEndpoint := fmt.Sprintf("%v:%d", cfg.LndHost, cfg.LndGrpcPort)
	log.Printf(logPrefix+"Dial lnd grpc at %v...", lndGrpcEndpoint)
	lndConn, err := grpc.Dial(lndGrpcEndpoint, lndOpts...)
	if err != nil {
		log.Printf(logPrefix+"Dial lnd error: %v", err)
		return nil, err
	}
	lndClient := lnrpc.NewLightningClient(lndConn)

	return lndClient, err
}

// Start the AustkServer listening for REST austk requests for art.
func (s *AustkServer) Start() error {
	const logPrefix = "AustkServer Start "

	s.debugPrintInventory()

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
	err = s.artServer.StorePeer(selfPeer, s)
	if err != nil {
		log.Printf(logPrefix+"PutPeer %v error: %v", pubkey, err)
		return err
	}

	// Listen for REST requests and serve in another thread.
	go s.serve()
	log.Printf(logPrefix+"serving REST requests for %s on %s:%d", pubkey, restHost, restPort)

	return err
}

func (s *AustkServer) debugPrintInventory() {
	const logPrefix = "server debugPrintInventory "

	artists, err := s.artServer.Artists()
	if err != nil {
		log.Printf(logPrefix+"artServer.Artists error: %v", err)
		return
	}
	log.Printf(logPrefix+"%v artists:", len(artists))
	for artistID, artist := range artists {
		log.Printf(logPrefix+"\tArtist id %s: %v", artistID, artist)
		tracks, err := s.artServer.Tracks(artistID)
		if err != nil {
			log.Printf(logPrefix+"artServer.Tracks error: %v", err)
		}
		for trackID, track := range tracks {
			log.Printf("\t\tTrack id: %s: %v", trackID, track)
		}
	}
}

// serve starts listening for and handling requests to austk endpoints.
func (server *AustkServer) serve() (err error) {
	const logPrefix = "server serve "
	httpRouter := mux.NewRouter()
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

	log.Printf(logPrefix+"configured artist: %s", server.config.ArtistId)
	publishingArtist, err := server.artServer.Artist(server.config.ArtistId)
	if err != nil {
		log.Printf(logPrefix+"Failed to retrieve configured artist %s, error: %v", server.config.ArtistId, err)
	}

	artists, err := server.artServer.Artists()
	if err != nil {
		log.Printf(logPrefix+"SelectAllArtists error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf(logPrefix+"found %d artists", len(artists))
	artistArray := make([]*art.Artist, 0, len(artists))
	trackArray := make([]*art.Track, 0)
	log.Println(logPrefix + "Select all artists:")
	for _, artist := range artists {
		log.Printf("\tArtist: %v", artist)
		artistArray = append(artistArray, artist)
		tracks, err := server.artServer.Tracks(artist.ArtistId)
		if err != nil {
			log.Printf(logPrefix+"artServer.Tracks error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, track := range tracks {
			log.Printf("\tTrack: %v", track)
			trackArray = append(trackArray, track)
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
		peerArray = append(peerArray, peer)
	}
	resources := art.ArtResources{
		Artists: artistArray,
		Tracks:  trackArray,
		Peers:   peerArray,
	}

	ctx := context.Background()
	var signMessageInput lnrpc.SignMessageRequest
	marshaledResources, err := proto.Marshal(&resources)
	if err != nil {
		log.Printf(logPrefix+"Marshal %v, error: %v", resources, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	signMessageInput.Msg = marshaledResources
	signMessageResult, err := server.lightningClient.SignMessage(ctx, &signMessageInput)
	if err != nil {
		log.Printf(logPrefix+"SignMessage error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	publicationSignature := signMessageResult.Signature
	log.Printf(logPrefix+"Signed message %v, signature: %v", resources, publicationSignature)

	publication := art.ArtistPublication{
		Artist:                 publishingArtist,
		Signature:              publicationSignature,
		SerializedArtResources: marshaledResources,
	}
	responseData, err := proto.Marshal(&publication)
	if err != nil {
		log.Printf(logPrefix+"Marshal %v, error: %v", publication, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(responseData)
}

func ValidatePublication(publication *art.ArtistPublication) (*art.ArtResources, error) {
	const logPrefix = "server ValidatePublication "

	artResources := art.ArtResources{}
	err := proto.Unmarshal(publication.SerializedArtResources, &artResources)
	if err != nil {
		log.Printf(logPrefix+"Unmarshal error: %v", err)
		return nil, err
	}
	return &artResources, nil
}

// Pubkey returns the pubkey for the lnd server,
// which clients can use to authenticate publications from this node.
func (server *AustkServer) Pubkey() (string, error) {
	ctx := context.Background()
	getInfoRequest := lnrpc.GetInfoRequest{}
	getInfoResponse, err := server.lightningClient.GetInfo(ctx, &getInfoRequest)
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

// getArtHandler handles requests to get a specified track by a specified artist.
func (server *AustkServer) getArtHandler(w http.ResponseWriter, req *http.Request) {
	const logPrefix = "server getArtHandler "

	artistID := mux.Vars(req)["artist"]
	artistTrackID := mux.Vars(req)["track"]
	if artistID == "" || artistTrackID == "" {
		log.Printf(logPrefix+"expected artist and track but received artist: %s, track: %s",
			artistID, artistTrackID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf(logPrefix+"artist: %v, track: %v", artistID, artistTrackID)

	track, err := server.artServer.Track(artistID, artistTrackID)
	if err != nil {
		// TODO: if requested track isn't found, error with 404 Not Found
		log.Printf(logPrefix+"failed to select track, error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: to require payment, check for secret received in receipt for paying invoice
	// If valid secret is not supplied, issue a Lightning invoice.
	// For now, skip the payment mechanics and serve the requested resource immediately.
	trackFilePath := server.artServer.TrackFilePath(track)
	mp3, err := OpenMp3ToRead(trackFilePath)
	if err != nil {
		log.Printf(logPrefix+"OpenMp3ForTrackToRead %s %s, error: %v", artistID, artistTrackID, err)
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
