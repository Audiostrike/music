package audiostrike

import (
	"context"
	"encoding/json"
	"flag"
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
)

var (
	restHost         = flag.String("host", defaultRESTHost, "ip/tor address for this audiostrike service")
	restPort         = flag.Int("port", defaultRESTPort, "port where audiostrike protocol is exposed")
	certFilePath     = flag.String("tlscert", defaultTLSCertPath, "file path for tls cert")
	macaroonFilePath = flag.String("macaroon", defaultMacaroonPath, "file path for macaroon")

	lndHost     = flag.String("lnd_host", defaultLndHost, "ip/onion address of lnd")
	lndGrpcPort = flag.Int("lnd_grpc_port", defaultLndGrpcPort, "port where lnd exposes grpc")
)

// ArtServer has this node's state and music directory.
type ArtServer struct {
	artDb       *AustkDb
	httpServer  *http.Server
	lndClient   lnrpc.LightningClient
	lndConn     *grpc.ClientConn
	quitChannel chan bool
}

// NewServer creates an ArtServer instance
func NewServer(serverOpts []grpc.ServerOption) (*ArtServer, error) {
	const logPrefix = "server NewServer "
	serverNameOverride := ""
	tlsCreds, err := credentials.NewClientTLSFromFile(*certFilePath, serverNameOverride)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"lnd credentials NewClientTLSFromFile error: %v\n", err)
		return nil, err
	}

	macaroonData, err := ioutil.ReadFile(*macaroonFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"ReadFile macaroon %v error %v\n", *macaroonFilePath, err)
		return nil, err
	}
	mac := &macaroon.Macaroon{}
	err = mac.UnmarshalBinary(macaroonData)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"UnmarchalBinary macaroon error: %v\n", err)
		return nil, err
	}

	lndOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCreds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(mac)),
	}

	lndGrpcEndpoint := fmt.Sprintf("%v:%d", *lndHost, *lndGrpcPort)
	fmt.Printf(logPrefix+"Dial lnd grpc at %v...", lndGrpcEndpoint)
	lndConn, err := grpc.Dial(lndGrpcEndpoint, lndOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"Dial lnd error: %v\n", err)
		return nil, err
	}
	fmt.Println(".")
	fmt.Print(logPrefix + "NewLightningClient...")
	lndClient := lnrpc.NewLightningClient(lndConn)
	fmt.Println(".")

	ctx := context.Background()
	var signMessageInput lnrpc.SignMessageRequest
	signMessageInput.Msg = []byte("Test message")
	signMessageResult, err := lndClient.SignMessage(ctx, &signMessageInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"SignMessage test error: %v\n", err)
		return nil, err
	}
	fmt.Printf(logPrefix+"Signed test message, signature: %v\n", signMessageResult.Signature)

	s := &ArtServer{
		//grpcServer: grpc.NewServer(serverOpts...),
		httpServer: &http.Server{
			Addr: "localhost",
		},
		lndConn:     lndConn,
		lndClient:   lndClient,
		quitChannel: make(chan bool),
	}
	return s, nil
}

// Start the ArtServer listening for REST requests for artists in the db.
func (s *ArtServer) Start(db *AustkDb) (err error) {
	const logPrefix = "server Start "
	s.artDb = db
	var artists map[string]art.Artist
	var tracks map[string]art.Track
	artists, err = db.SelectAllArtists()
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"db.SelectAllArtists error: %v\n", err)
		return
	}
	fmt.Printf(logPrefix+"%v artists:\n", len(artists))
	for artistID, artist := range artists {
		fmt.Printf("\tArtist id %s: %v\n", artistID, artist)
		tracks, err = db.SelectArtistTracks(artistID)
		if err != nil {
			fmt.Fprintf(os.Stderr, logPrefix+"db.SelectArtistTracks error: %v\n", err)
			return
		}
		for trackID, track := range tracks {
			fmt.Printf("\t\tTrack id: %s: %v\n", trackID, track)
		}
	}

	pubkey, err := s.Pubkey()
	selfPeer, err := db.SelectPeer(pubkey)
	if err == sql.ErrNoRows {
		fmt.Printf(logPrefix+"Inserting peer %v:%v for %v\n", *restHost, *restPort, pubkey)
		err = db.PutPeer(&art.Peer{Pubkey: pubkey, Host: *restHost, Port: uint32(*restPort)})
	} else if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"SelectPeer %v error: %v\n", pubkey, err)
		return
	} else {
		fmt.Printf(logPrefix+"Updating peer %v:%v to %v:%v\n",
			selfPeer.Host, selfPeer.Port,
			*restHost, *restPort)
		err = db.PutPeer(&art.Peer{Pubkey: pubkey, Host: *restHost, Port: uint32(*restPort)})
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"PutPeer %v error: %v\n", pubkey, err)
		return
	}

	// Listen for REST requests and serve in another thread.
	go s.serve()
	fmt.Printf(logPrefix+"serving REST requests on :%d\n", *restPort)

	return
}

func (server *ArtServer) serve() (err error) {
	const logPrefix = "server serve "
	httpRouter := mux.NewRouter()
	httpRouter.HandleFunc("/artist/{id}", server.putArtistHandler).Methods("PUT")
	httpRouter.HandleFunc("/", server.getAllArtHandler).Methods("GET")
	httpRouter.HandleFunc("/art/{artist:[^/]*}/{track:.*}", server.getArtHandler).Methods("GET")
	restAddress := fmt.Sprintf(":%d", *restPort)
	err = http.ListenAndServe(restAddress, httpRouter)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"ListenAndServe error: %v\n", err)
	}

	return
}

func (server *ArtServer) getAllArtHandler(w http.ResponseWriter, req *http.Request) {
	const logPrefix = "server getAllArtHandler "
	// TODO: read any predicates from req to filter results
	// for price (per track, per minute, or per byte),
	// preferred bit rate, or other conditions TBD.
	// Maybe read any follow-back peer URL as well.

	artists, err := server.artDb.SelectAllArtists()
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"SelectAllArtists error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	artistArray := make([]*art.Artist, 0, len(artists))
	trackArray := make([]*art.Track, 0)
	fmt.Println(logPrefix + "Select all artists:")
	for _, artist := range artists {
		fmt.Printf("\tArtist: %v\n", artist)
		artistCopy := artist
		artistArray = append(artistArray, &artistCopy)
		tracks, err := server.artDb.SelectArtistTracks(artist.ArtistId)
		if err != nil {
			fmt.Fprintf(os.Stderr, logPrefix+"SelectAllTracks error: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, track := range tracks {
			fmt.Printf("\tTrack: %v\n", track)
			trackCopy := track
			trackArray = append(trackArray, &trackCopy)
		}
	}
	peers, err := server.artDb.SelectAllPeers()
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"SelectAllPeers error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	peerArray := make([]*art.Peer, 0, len(peers))
	for _, peer := range peers {
		fmt.Printf("\tPeer: %v\n", peer)
		peerCopy := *peer
		peerArray = append(peerArray, &peerCopy)
	}
	reply := art.ArtReply{
		Artists: artistArray,
		Tracks:  trackArray,
		Peers: peerArray,
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
		fmt.Fprintf(os.Stderr, logPrefix+"SignMessage error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Printf(logPrefix+"Signed message %s, signature: %v\n", data, signMessageResult.Signature)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (server *ArtServer) Pubkey() (string, error) {
	ctx := context.Background()
	getInfoRequest := lnrpc.GetInfoRequest{}
	getInfoResponse, err := server.lndClient.GetInfo(ctx, &getInfoRequest)
	if err != nil {
		return "", err
	}
	pubkey := getInfoResponse.IdentityPubkey

	return pubkey, nil
}

// WaitUntilQuitSignal waits for SIGINT (keyboard interrupt Ctrl-C) or for another reason to quit.
func (server *ArtServer) WaitUntilQuitSignal() {
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

// Stop the ArtServer
func (server *ArtServer) Stop() error {
	server.quitChannel <- true
	return nil
}

func (server *ArtServer) putArtistHandler(w http.ResponseWriter, req *http.Request) {
	requestVars := mux.Vars(req)
	decoder := json.NewDecoder(req.Body)
	var artist art.Artist
	err := decoder.Decode(&artist)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode artist from request, error: %v\n", err)
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
		err = server.artDb.PutArtist(&artist)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to put artist into DB, error: %v\n", err)
		}
	}()
}

func (server *ArtServer) getArtHandler(w http.ResponseWriter, req *http.Request) {
	const logPrefix = "server getArtHandler "
	artistId := mux.Vars(req)["artist"]
	trackId := mux.Vars(req)["track"]
	if artistId == "" || trackId == "" {
		fmt.Fprintf(os.Stderr, logPrefix+"expected artist and track but received artist: %s, track: %s\n", artistId, trackId)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Printf(logPrefix+"artist: %v, track: %v\n", artistId, trackId)
	_, err := server.artDb.SelectTrack(artistId, trackId)
	if err != nil {
		// TODO: if requested track isn't in the db, error with 404 Not Found
		fmt.Fprintf(os.Stderr, logPrefix+"failed to select track, error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// TODO: to require payment, check for secret received in receipt for paying invoice
	// If valid secret is not supplied, issue a Lightning invoice.
	// For now, skip the payment mechanics and serve the requested resource immediately.
	filename := fmt.Sprintf("./tracks/%s/%s", artistId, trackId)
	trackData, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"ReadFile %s error: %v\n", filename, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	//data, err := json.Marshal(track)
	fmt.Printf(logPrefix+"data: %s\n", string(trackData))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(trackData)
}

// GetArt returns an ArtReply with all known art
func (s *ArtServer) GetArt(context context.Context, artRequest *art.ArtRequest) (artReply *art.ArtReply, err error) {
	artists, err := s.artDb.SelectAllArtists()
	if err != nil {
		return nil, err
	}
	artistArray := make([]*art.Artist, 0, len(artists))
	albumsArray := make([]*art.Album, 0)
	tracksArray := make([]*art.Track, 0)
	for _, artist := range artists {
		artistArray = append(artistArray, &artist)
		albums, err := s.artDb.SelectArtistAlbums(artist.ArtistId)
		if err != nil {
			return nil, err
		}
		for _, album := range albums {
			albumsArray = append(albumsArray, album)
		}
	}
	peersArray, err := s.artDb.SelectAllPeers()
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
