package audiostrike

import (
	"fmt"
	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	mockArtistId string = "alicetheartist"
	mockPubkey   string = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef50"
	mockTrackId  string = "testtrack"
)

var cfg *Config = &Config{
	ArtDir:       "testart",
	CertFilePath: "tls.cert",
	MacaroonPath: "test.macaroon",
	LndHost:      "127.0.0.1",
	LndGrpcPort:  10009,
}

type MockArtServer struct {
	artists  map[string]*art.Artist
	albums   map[string]map[string]*art.Album
	peers    map[string]*art.Peer
	tracks   map[string]map[string]*art.Track
	payloads map[string]map[string][]byte
}

func (s *MockArtServer) Artists() (map[string]*art.Artist, error) {
	return s.artists, nil
}

func (s *MockArtServer) Artist(artistId string) (*art.Artist, error) {
	return s.artists[artistId], nil
}

func (s *MockArtServer) StoreAlbum(album *art.Album) error {
	s.albums[album.ArtistId][album.ArtistAlbumId] = album
	return nil
}

func (s *MockArtServer) Albums(artistId string) (map[string]*art.Album, error) {
	return s.albums[artistId], nil
}

func (s *MockArtServer) Peer(pubkey string) (*art.Peer, error) {
	for _, peer := range s.peers {
		if peer.Pubkey == pubkey {
			return peer, nil
		}
	}
	return nil, ErrPeerNotFound
}

func (s *MockArtServer) Peers() (map[string]*art.Peer, error) {
	return s.peers, nil
}

func (s *MockArtServer) StoreArtist(artist *art.Artist) error {
	return fmt.Errorf("not implemented")
}

func (s *MockArtServer) StorePeer(peer *art.Peer) error {
	for i, old := range s.peers {
		if old.Pubkey == peer.Pubkey {
			s.peers[i] = peer
			return nil
		}
	}
	s.peers[peer.Pubkey] = peer
	return nil
}

func (s *MockArtServer) Track(artistId string, trackId string) (*art.Track, error) {
	if artistId == mockArtistId && trackId == mockTrackId {
		return &art.Track{
			ArtistId:      mockArtistId,
			ArtistTrackId: mockTrackId,
		}, nil
	} else {
		return nil, ErrArtNotFound
	}
}

func (s *MockArtServer) TrackFilePath(artistID string, artistTrackID string) string {
	return BuildMp3Filename(cfg.ArtDir, artistID, artistTrackID)
}

func (s *MockArtServer) StoreTrack(track *art.Track) error {
	s.tracks[track.ArtistId][track.ArtistTrackId] = track
	return nil
}

func (s *MockArtServer) StoreTrackPayload(artistId string, artistTrackId string, payload []byte) error {
	s.payloads[artistId][artistTrackId] = payload
	return nil
}

func (s *MockArtServer) Tracks(artistId string) (map[string]*art.Track, error) {
	return s.tracks[artistId], nil
}

var mockArtServer MockArtServer = MockArtServer{
	artists: map[string]*art.Artist{
		mockArtistId: &art.Artist{
			ArtistId: mockArtistId,
			Pubkey:   mockPubkey,
		},
	},
	peers: map[string]*art.Peer{},
	tracks: map[string]map[string]*art.Track{
		mockArtistId: map[string]*art.Track{
			mockTrackId: &art.Track{
				ArtistId:      mockArtistId,
				ArtistTrackId: mockTrackId,
			},
		},
	},
}

// TestGetAllArt tests that AustkServer's getAllArtHandler returns art from the given ArtServer.
func TestGetAllArt(t *testing.T) {
	austkServer, err := NewAustkServer(cfg, &mockArtServer)
	if err != nil {
		t.Errorf("Failed to connect to music DB, error %v", err)
	}

	// Start an httptest server to record and test the reply of austkServer's handlerFunc.
	testHttpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		austkServer.getAllArtHandler(w, req)
	}))
	defer testHttpServer.Close()

	// Get and deserialize the ArtReply to verify that austkServer published the expected art.
	artReply := art.ArtReply{}
	response, err := http.Get(testHttpServer.URL)
	bytes, err := ioutil.ReadAll(response.Body)
	err = proto.Unmarshal(bytes, &artReply)

	// Verify that the server handled the request successfully.
	if response.StatusCode != 200 {
		t.Errorf("expected success but got %d", response.StatusCode)
	}

	// Verify that the one test artist and her music was served.
	if len(artReply.Artists) != 1 {
		t.Errorf("expected 1 artist but got %d in reply: %v", len(artReply.Artists), artReply)
	}
	replyArtist := artReply.Artists[0]
	if replyArtist.Pubkey != mockPubkey {
		t.Errorf("expected artist with mock pubkey %s but got %s in reply: %v",
			mockPubkey, replyArtist.Pubkey, artReply)
	}
	if replyArtist.ArtistId != mockArtistId {
		t.Errorf("expected artist with id %s but got %s in reply: %v",
			mockArtistId, replyArtist.ArtistId, artReply)
	}

	if len(artReply.Tracks) != 1 {
		t.Errorf("expected 1 track but got %d in reply: %v", len(artReply.Tracks), artReply)
	}
	replyTrack := artReply.Tracks[0]
	if replyTrack.ArtistId != mockArtistId {
		t.Errorf("expected track with artist id %s but got %s in reply: %v",
			mockArtistId, replyTrack.ArtistId, artReply)
	}
}

// TestGetArt
func TestGetArt(t *testing.T) {
	austkServer, err := NewAustkServer(cfg, &mockArtServer)
	if err != nil {
		t.Errorf("Failed to connect to music DB, error %v", err)
	}

	// Start an httptest server to record and test the reply of austkServer's handlerFunc.
	testRouter := mux.NewRouter()
	testRouter.HandleFunc("/art/{artist:[^/]*}/{track:.*}",
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			austkServer.getArtHandler(w, req)
		})).Methods("GET")
	testHttpServer := httptest.NewServer(testRouter)
	defer testHttpServer.Close()

	// Get the reply to verify that austkServer published the expected art.
	artRequestUrl := fmt.Sprintf("%s/art/%s/%s", testHttpServer.URL, mockArtistId, mockTrackId)
	t.Logf("request url %s", artRequestUrl)
	response, err := http.Get(artRequestUrl)
	bytes, err := ioutil.ReadAll(response.Body)

	// Verify that the server handled the request successfully.
	if response.StatusCode != 200 {
		t.Errorf("expected success but got %d", response.StatusCode)
	}

	// Verify that the one test artist and her music was served.
	if len(bytes) != 7412458 {
		t.Errorf("expected 7412458 byte track but got %d bytes in reply", len(bytes))
	}
}

// Verify that the server publishes itself as the Peer with its Pubkey.
func TestPeersForServerPubkey(t *testing.T) {
	cfg := &Config{
		CertFilePath: "tls.cert",
		MacaroonPath: "test.macaroon",
		LndHost:      "127.0.0.1",
		LndGrpcPort:  10009,
	}
	austkServer, err := NewAustkServer(cfg, &mockArtServer)
	if err != nil {
		t.Errorf("Failed to connect to music DB, error %v", err)
	}

	// Verify that the server starts and has a pubkey from lnd.
	err = austkServer.Start()
	if err != nil {
		t.Errorf("Failed to Start austkServer, error: %v", err)
	}
	lndPubkey, err := austkServer.Pubkey()
	if err != nil {
		t.Errorf("lndPubkey error: %v", err)
	}

	// Start an httptest server to record and test the reply of austkServer's handlerFunc.
	testHttpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		austkServer.getAllArtHandler(w, req)
	}))
	defer testHttpServer.Close()

	// Get and deserialize the ArtReply to verify that austkServer published the expected peer.
	artReply := art.ArtReply{}
	response, err := http.Get(testHttpServer.URL)
	bytes, err := ioutil.ReadAll(response.Body)
	err = proto.Unmarshal(bytes, &artReply)

	// Verify that the server handled the request successfully.
	if response.StatusCode != 200 {
		t.Errorf("expected success but got %d", response.StatusCode)
	}

	// Verify that the one test artist and her music was served.
	if len(artReply.Peers) != 1 {
		t.Errorf("expected 1 peer but got %d in reply: %v", len(artReply.Peers), artReply)
	}
	replyPeer := artReply.Peers[0]
	if replyPeer.Pubkey != lndPubkey {
		t.Errorf("expected peer with pubkey %s but got %s in reply: %v",
			lndPubkey, replyPeer.Pubkey, artReply)
	}
}
