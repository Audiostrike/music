package audiostrike

import (
	"testing"

	art "github.com/audiostrike/music/pkg/art"
)

const (
	rootPath     = "testart"
	testArtistId = "tester"
)

var mockArtist = art.Artist{
	ArtistId: testArtistId,
	Name:     "Artist McTester",
	Pubkey:   mockPubkey}

type MockPublisher struct{}

func (s *MockPublisher) Artist() (*art.Artist, error) {
	return &mockArtist, nil
}

func (s *MockPublisher) Sign(resources *art.ArtResources) (*art.ArtistPublication, error) {
	return &art.ArtistPublication{}, nil
}

func (s *MockPublisher) Verify(publication *art.ArtistPublication) (*art.ArtResources, error) {
	return &art.ArtResources{}, nil
}

var mockPublisher MockPublisher

func TestTrack(t *testing.T) {
	fileServer, err := NewFileServer(rootPath)
	if err != nil {
		t.Errorf("Failed to instantiate file server on %s, error %v", rootPath, err)
	}
	const testArtistId = "tester"
	const testAlbumId = "putting-tracks"
	testTracks := []art.Track{
		art.Track{
			ArtistId:         testArtistId,
			ArtistAlbumId:    testAlbumId,
			AlbumTrackNumber: 1,
			ArtistTrackId:    testAlbumId + "-1.TestPut1",
			Title:            "Test Put 1"},
		art.Track{
			ArtistId:         testArtistId,
			ArtistAlbumId:    testAlbumId,
			AlbumTrackNumber: 2,
			ArtistTrackId:    testAlbumId + "-2.TestPut2",
			Title:            "Test Put 2"},
		art.Track{
			ArtistId:         testArtistId,
			ArtistAlbumId:    testAlbumId,
			AlbumTrackNumber: 3,
			ArtistTrackId:    testAlbumId + "-3.TestPut3",
			Title:            "Test Put 3"},
	}
	for _, testTrack := range testTracks {
		err = fileServer.StoreTrack(&testTrack, &mockPublisher)
		if err != nil {
			t.Errorf("Failed to store track %v, error: %v", testTrack, err)
		}
		selectedTrack, err := fileServer.Track(testArtistId, testTrack.ArtistTrackId)
		if err != nil || selectedTrack == nil ||
			selectedTrack.AlbumTrackNumber != testTrack.AlbumTrackNumber ||
			selectedTrack.Title != testTrack.Title {
			t.Errorf("Track failed to select track %v, error: %v, found: %v",
				testTrack, err, selectedTrack)
		}
	}
}

func TestAlbumFiles(t *testing.T) {
	// Test album with known id
	fileServer, err := NewFileServer(rootPath)
	if err != nil {
		t.Errorf("NewFileServer(%s), error: %v", rootPath, err)
	}

	const (
		testArtistId = "tester"
		testAlbumId  = "test-album-files-album-id"
	)

	err = fileServer.StoreTrack(&art.Track{
		ArtistId:         testArtistId,
		ArtistAlbumId:    testAlbumId,
		AlbumTrackNumber: 1,
		ArtistTrackId:    testAlbumId + "-1.SomeTrack",
		Title:            "Some Track"}, &mockPublisher)
	if err != nil {
		t.Errorf("StoreTrack failed  for %s/%s", testArtistId, testAlbumId)
	}

	// Check the file server for the album track.
	tracks, err := fileServer.AlbumTracks(testArtistId, testAlbumId)
	if err != nil || len(tracks) == 0 {
		t.Errorf("AlbumTracks failed to get tracks for \"%s\", got only %v", testAlbumId, tracks)
	}
	fetchedTrack1 := tracks[1]
	if fetchedTrack1.ArtistAlbumId != testAlbumId {
		t.Errorf("First album track fetched had the wrong album id (%s), expected %s",
			testAlbumId, fetchedTrack1.ArtistAlbumId)
	}
}

func TestPeers(t *testing.T) {
	// Test peer with known id
	fileServer, err := NewFileServer(rootPath)
	if err != nil {
		t.Errorf("NewFileServer(%s), error: %v", rootPath, err)
	}

	err = fileServer.StoreArtist(&mockArtist, &mockPublisher)
	if err != nil {
		t.Errorf("StoreArtist failed for %s with pubkey %s, error: %v", testArtistId, mockPubkey, err)
	}

	err = fileServer.StorePeer(&art.Peer{Pubkey: mockPubkey}, &mockPublisher)
	if err != nil {
		t.Errorf("StorePeer failed for pubkey %s, error: %v", mockPubkey, err)
	}

	// Check the file server for the album track.
	peers, err := fileServer.Peers()
	if err != nil || len(peers) == 0 {
		t.Errorf("Peers failed to get added peer, error: %v", err)
	}
	fetchedPeer := peers[mockPubkey]
	if fetchedPeer == nil {
		t.Errorf("Peer failed to fetch for pubkey %s", mockPubkey)
	}
	if fetchedPeer.Pubkey != mockPubkey {
		t.Errorf("Peer fetched had the wrong pubkey (%s), expected %s",
			fetchedPeer.Pubkey, mockPubkey)
	}
}
