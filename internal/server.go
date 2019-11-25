package audiostrike

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/lightningnetwork/lnd/lntypes"
)

const (
	HttpHeaderPaymentHash     = "Payment-Hash"
	HttpHeaderPaymentPreimage = "Payment-Preimage"
)

var (
	ErrArtNotFound      = errors.New("ArtServer has no such art")
	ErrInvalidSignature = errors.New("Signature is not valid for the artist pubkey and message")
	ErrPeerNotFound     = errors.New("AustkServer has no such peer")
	ErrInvoiceNotFound  = errors.New("Invoice not found")
)

type Pubkey string

// AustkServer hosts publishingArtist's art for http/tor clients who might pay the lightning node for it.
type AustkServer struct {
	config      *Config
	artServer   ArtServer // interface for storing art in a file system, database, or test mock
	httpServer  *http.Server
	publisher   Publisher
	quitChannel chan bool
}

// ArtServer is a repository to store/serve music and related data for this austk node.
// Implementations may use a database, file system, test fixture, etc.
type ArtServer interface {
	// Get and store artist info.
	StoreArtist(artist *art.Artist) error
	Artists() (map[string]*art.Artist, error)
	Artist(artistId string) (*art.Artist, error)

	// Album: an artist's optional track container to name and sequence tracks
	StoreAlbum(*art.Album) error
	Albums(artistId string) (map[string]*art.Album, error)

	// Get and store Track info.
	StoreTrack(*art.Track) error
	StoreTrackPayload(*art.Track, []byte) error
	Tracks(artistID string) (map[string]*art.Track, error)
	Track(artistID string, artistTrackID string) (*art.Track, error)
	TrackFilePath(track *art.Track) string

	// Get and store an Invoice.
	StoreInvoice(*art.Invoice) error
	Invoice(paymentHash *lntypes.Hash) (*art.Invoice, error)

	// Get and store network info.
	StorePeer(*art.Peer) error
	Peers() (map[Pubkey]*art.Peer, error)
	Peer(Pubkey) (*art.Peer, error)

	StorePublication(*art.ArtistPublication) error
}

type Publisher interface {
	Artist() (*art.Artist, error)
	Pubkey() (Pubkey, error)
	Publish(*art.ArtResources) (*art.ArtistPublication, error)
	NewInvoice(tracks []*art.Track, amount int32, unit art.Bolt11AmountMultiplier) (*art.Invoice, error)
	Invoice(paymentHash *lntypes.Hash) (*art.Invoice, error)
}

var invalidIDRegex = regexp.MustCompile("[^a-z0-9.-]")

// NameToId converts the name or title of an artist, album, or track
// into a case-insensitive id usable for urls, filenames, etc.
func NameToID(name string) string {
	lowerCaseName := strings.ToLower(name)
	// strip whitespace, punctuation, etc. and leave just a lower-case string of letters, numbers, periods, and dashes.
	return invalidIDRegex.ReplaceAllString(lowerCaseName, "")
}

var invalidHierarchyRegex = regexp.MustCompile("[^/a-z0-9.-]")

func TitleToHierarchy(title string) string {
	lowerCaseTitle := strings.ToLower(title)
	return invalidHierarchyRegex.ReplaceAllString(lowerCaseTitle, "")
}

// Artist gets the Artist publishing from this server.
func (server *AustkServer) Artist() (*art.Artist, error) {
	return server.publisher.Artist()
}

// func (server *AustkServer) Pubkey() (Pubkey, error) {
// 	return server.publisher.Pubkey()
// }

func (server *AustkServer) Sign(resources *art.ArtResources) (*art.ArtistPublication, error) {
	const logPrefix = "AustkServer Sign "

	publication, err := server.publisher.Publish(resources)
	if err != nil {
		log.Fatalf(logPrefix+"lnd is not operational. SignMessage error: %v", err)
		return nil, err
	}
	publishingArtist, err := server.Artist()
	if err != nil {
		log.Fatalf(logPrefix+"failed to get publishing artist, error: %v", err)
		return nil, err
	}
	if publication == nil {
		log.Fatalf(logPrefix+"No publication to sign resources %v", resources)
		return nil, ErrArtNotFound
	} else if publication.Artist == nil {
		log.Fatalf(logPrefix+"No publication artist to sign resources %v", resources)
		return nil, ErrArtNotFound
	}
	if publishingArtist == nil {
		log.Fatalf(logPrefix + "server has no publishingArtist")
		return nil, ErrArtNotFound
	}
	if publishingArtist.Pubkey != publication.Artist.Pubkey ||
		publishingArtist.ArtistId != publication.Artist.ArtistId {
		return nil, fmt.Errorf(
			"lightning node signed with pubkey %s for artist %s but expected pubkey %s for %s",
			publication.Artist.Pubkey, publication.Artist.ArtistId,
			publishingArtist.Pubkey, publishingArtist.ArtistId)
	}

	return publication, nil
}

// NewAustkServer creates a new network Server to serve the configured artist's art.
func NewAustkServer(cfg *Config, localStorage ArtServer, publisher Publisher) (*AustkServer, error) {
	const logPrefix = "server NewAustkServer "

	server := &AustkServer{
		artServer:   localStorage,
		config:      cfg,
		httpServer:  &http.Server{Addr: "localhost"},
		publisher:   publisher,
		quitChannel: make(chan bool),
	}

	return server, nil
}

// Start the AustkServer listening for REST austk requests for art.
func (s *AustkServer) Start() error {
	const logPrefix = "AustkServer Start "

	s.debugPrintInventory()

	// Publish this austk node as the Peer with this server's Pubkey.
	pubkey, err := s.publisher.Pubkey()
	if err != nil {
		log.Printf(logPrefix+"Pubkey retrieval error: %v", err)
		return err
	}
	log.Printf(logPrefix+"start with pubkey %s", pubkey)
	restHost := s.RestHost()
	restPort := s.RestPort()

	selfPeer, err := s.artServer.Peer(pubkey)
	if err == ErrPeerNotFound {
		log.Printf(logPrefix+"artServer has no peer with this publisher's pubkey %s", pubkey)
		selfPeer = &art.Peer{Pubkey: string(pubkey), Host: restHost, Port: restPort}
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
	err = s.artServer.StorePeer(selfPeer)
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
	httpRouter.HandleFunc("/art/{artist:[^/]*}/{track:.*}", server.getTrackHandler).Methods("GET")
	httpRouter.HandleFunc("/invoice/{artist:[^/]*}/{track:.*}", server.getTrackInvoiceHandler).Methods("GET")
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

	resources, err := CollectResources(server.artServer)
	if err != nil {
		log.Printf(logPrefix+"collectResources error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	publication, err := server.Sign(resources)
	if err != nil {
		log.Printf(logPrefix+"failed to Sign resources %v, error: %v", resources, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf(logPrefix+"Signed resources %v, publication: %v", resources, publication)
	responseData, err := proto.Marshal(publication)
	if err != nil {
		log.Printf(logPrefix+"Marshal %v, error: %v", publication, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(responseData)
}

func CollectResources(artServer ArtServer) (*art.ArtResources, error) {
	const logPrefix = "server collectResources "

	artists, err := artServer.Artists()
	if err != nil {
		log.Printf(logPrefix+"SelectAllArtists error: %v", err)
		return nil, err
	}
	log.Printf(logPrefix+"found %d artists", len(artists))
	artistArray := make([]*art.Artist, 0, len(artists))
	trackArray := make([]*art.Track, 0)
	log.Println(logPrefix + "Select all artists:")
	for _, artist := range artists {
		log.Printf("\tArtist: %v", artist)
		artistArray = append(artistArray, artist)
		tracks, err := artServer.Tracks(artist.ArtistId)
		if err != nil {
			log.Printf(logPrefix+"artServer.Tracks error: %v", err)
			return nil, err
		}
		for _, track := range tracks {
			log.Printf("\tTrack: %v", track)
			trackArray = append(trackArray, track)
		}
	}
	peers, err := artServer.Peers()
	if err != nil {
		log.Printf(logPrefix+"SelectAllPeers error: %v", err)
		return nil, err
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

	return &resources, nil
}

func read(publication *art.ArtistPublication) (*art.ArtResources, error) {
	const logPrefix = "server readPublication "

	artResources := art.ArtResources{}
	err := proto.Unmarshal(publication.SerializedArtResources, &artResources)
	if err != nil {
		log.Printf(logPrefix+"Unmarshal error: %v", err)
		return nil, err
	}
	return &artResources, nil
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
func (server *AustkServer) getTrackHandler(w http.ResponseWriter, req *http.Request) {
	const logPrefix = "(*AustkServer) getTrackHandler "

	artistID := mux.Vars(req)["artist"]
	artistTrackID := mux.Vars(req)["track"]
	if artistID == "" || artistTrackID == "" {
		log.Printf(logPrefix+"expected artist and track but received artist: %s, track: %s",
			artistID, artistTrackID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	paymentPreimageHex := req.Header.Get(HttpHeaderPaymentPreimage)
	if paymentPreimageHex == "" {
		log.Printf(logPrefix+"request lacks header "+HttpHeaderPaymentPreimage+", headers: %v",
			req.Header)
		w.WriteHeader(http.StatusPaymentRequired) // 402 Payment Required
		w.Write([]byte("payment req'd"))
		return
	}

	paymentPreimageBytes, err := hex.DecodeString(paymentPreimageHex)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400 Bad Request
		w.Write([]byte(HttpHeaderPaymentPreimage + " must be hex encoded payment preimage"))
		return
	}
	log.Printf(logPrefix+"artist: %s, track: %s, paymentPreimageHex: %s", artistID, artistTrackID, paymentPreimageHex)
	paymentHashArray := sha256.Sum256(paymentPreimageBytes)
	paymentHash, err := lntypes.MakeHash(paymentHashArray[:])
	if err != nil {
		log.Printf(logPrefix+"failed to hash preimage, error: %v", err)
		w.WriteHeader(http.StatusInternalServerError) // 500 Internal Server Error
		return
	}
	paymentHashHex := hex.EncodeToString(paymentHash[:])

	invoice, err := server.artServer.Invoice(&paymentHash)
	if err != nil {
		log.Printf(logPrefix+"invoice #%s for %s/%s not found, error: %v",
			paymentHash, artistID, artistTrackID, err)
		w.Header().Set(HttpHeaderPaymentHash, paymentHashHex)
		w.WriteHeader(http.StatusNotFound) // 404 Not Found
		return
	}

	track, err := server.artServer.Track(artistID, artistTrackID)
	if err != nil {
		// Requested track was not found
		log.Printf(logPrefix+"failed to select track, error: %v", err)
		w.WriteHeader(http.StatusNotFound) // 404 Not Found
		w.Write([]byte(fmt.Sprintf("No such track: %s/%s", artistID, artistTrackID)))
		return
	}

	// Validate that the paid invoice is for the requested track.
	err = assertHasTrack(invoice, track)
	if err != nil {
		message := fmt.Sprintf("Invoice with payment hash %s is not for track %s/%s", paymentHashHex, artistID, artistTrackID)
		log.Printf(logPrefix+"Forbidden: %s", message)
		w.Header().Set(HttpHeaderPaymentHash, paymentHashHex)
		w.WriteHeader(http.StatusForbidden) // 403 Forbidden
		w.Write([]byte(message))
		return
	}

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
		w.WriteHeader(http.StatusInternalServerError) // 500 Internal Server Error
		return
	}

	log.Printf(logPrefix+"serving track as %d bytes of data", len(trackData))
	w.WriteHeader(http.StatusOK) // 200 OK
	w.Write(trackData)
}

// TODO: move to invoice.go
//func (invoice *art.Invoice) assertHasTrack(track *art.Track) error {
func assertHasTrack(invoice *art.Invoice, track *art.Track) error {
	for _, invoiceTrack := range invoice.Tracks {
		if track.ArtistId == invoiceTrack.ArtistId &&
			track.ArtistTrackId == invoiceTrack.ArtistTrackId {
			// invoice has track with same id, no error
			return nil
		}
	}
	return ErrArtNotFound
}

// getTrackInvoiceHandler handles requests to get a specified track by a specified artist.
func (server *AustkServer) getTrackInvoiceHandler(w http.ResponseWriter, req *http.Request) {
	const logPrefix = "(*AustkServer) getTrackInvoiceHandler "

	artistID := mux.Vars(req)["artist"]
	artistTrackID := mux.Vars(req)["track"]
	if artistID == "" || artistTrackID == "" {
		log.Printf(logPrefix+"expected artist and track but received artist: %s, track: %s",
			artistID, artistTrackID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	track, err := server.artServer.Track(artistID, artistTrackID)
	if err != nil {
		log.Printf(logPrefix+"failed to get track %s/%s, error: %v",
			artistID, artistTrackID, err)
		w.WriteHeader(http.StatusNotFound) // 404 Not Found
		return
	}

	trackFilePath := server.artServer.TrackFilePath(track)
	log.Printf(logPrefix+"artist: %s, track: %s, path: %s",
		artistID, artistTrackID, trackFilePath)

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
	// Hash trackData to compose a Lightning invoice
	// that commits to the track's metadata and its trackData hash.
	trackHash := sha256.Sum256(trackData)
	trackPath := filepath.Join(track.ArtistId, track.ArtistTrackId)
	trackInvoiceMemo := fmt.Sprintf("%s#%x", trackPath, trackHash[0:32])
	log.Printf(logPrefix+"invoice memo: %s", trackInvoiceMemo)
	maxBitRateRequestVar := mux.Vars(req)["maxBitRate"]
	var maxBitRateKbps int32
	if maxBitRateRequestVar == "" {
		maxBitRateKbps = 0
	} else {
		maxBitRateKbps64, err := strconv.ParseInt(maxBitRateRequestVar, 10, 32)
		if err != nil {
			log.Printf(logPrefix+"invalid parameter maxBitRate:\"%s\", error: %v",
				maxBitRateRequestVar, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		maxBitRateKbps = int32(maxBitRateKbps64)
	}
	// TODO: configure price in bits
	priceInBits := int32(1)
	trackForInvoice := *track
	trackForInvoice.Presentations = make([]*art.TrackPresentation, 0, len(track.Presentations))
	for _, presentation := range track.Presentations {
		if maxBitRateKbps == 0 || presentation.BitRateKbps <= maxBitRateKbps {
			trackForInvoice.Presentations = append(trackForInvoice.Presentations, presentation)
		}
	}

	invoice, err := server.publisher.NewInvoice(
		[]*art.Track{&trackForInvoice},
		priceInBits, art.Bolt11AmountMultiplier_BITCOIN_BIT,
	)
	if err != nil {
		log.Printf(logPrefix+"Failed to add lightning invoice for track memo: %s, error: %v",
			trackInvoiceMemo, err)
		w.WriteHeader(http.StatusInternalServerError) // 500 Internal Server Error
		return
	}

	// save preimage to local storage for fast lookup without hashing
	err = server.artServer.StoreInvoice(invoice)
	if err != nil {
		log.Printf(logPrefix+"Failed to store invoice, error: %v", err)
		w.WriteHeader(http.StatusInternalServerError) // 500 Internal Server error
		return
	}
	log.Printf(logPrefix+"serving invoice: %v", invoice)

	invoiceBytes, err := proto.Marshal(invoice)
	if err != nil {
		log.Printf(logPrefix+"Failed to send invoice, error: %v", err)
		w.WriteHeader(http.StatusInternalServerError) // 500 Internal Server Error
		return
	}

	w.WriteHeader(http.StatusOK) // 200 OK
	w.Write(invoiceBytes)
}
