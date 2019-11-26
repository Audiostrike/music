package audiostrike

import (
	"github.com/golang/protobuf/proto"
	"testing"

	art "github.com/audiostrike/music/pkg/art"
	"log"
)

const (
	rootPath = "testart"
)

var mockArtist = art.Artist{
	ArtistId: mockArtistID,
	Name:     "Alice McTester",
	Pubkey:   string(mockPubkey)}

type MockPublisher struct{}

func (s *MockPublisher) Artist() (*art.Artist, error) {
	return &mockArtist, nil
}

func (s *MockPublisher) Pubkey() (Pubkey, error) {
	return mockPubkey, nil
}

func (s *MockPublisher) Sign(resources *art.ArtResources) (*art.ArtistPublication, error) {
	marshaledResources, err := proto.Marshal(resources)
	if err != nil {
		log.Printf("Marshal %v, error: %v", resources, err)
		return nil, err
	}
	return &art.ArtistPublication{
		Artist:                 &mockArtist,
		Signature:              "mock signature",
		SerializedArtResources: marshaledResources,
	}, nil
}

func (s *MockPublisher) Verify(publication *art.ArtistPublication) (*art.ArtResources, error) {
	return &art.ArtResources{}, nil
}

var mockPublisher MockPublisher

func TestSaveAndLoadFromPub(t *testing.T) {
	savingFileServer, err := NewFileServer(rootPath)
	if err != nil {
		t.Errorf("failed to instantiate file server at %s, error: %v", rootPath, err)
	}

	mockTrack := art.Track{
		ArtistId:      mockArtistID,
		ArtistTrackId: mockTrackID,
		Title:         "Test Track",
	}
	resources := &art.ArtResources{
		Artists: []*art.Artist{&mockArtist},
		Tracks:  []*art.Track{&mockTrack},
	}
	publication, err := mockPublisher.Sign(resources)
	if err != nil {
		t.Errorf("failed to sign resources %v, error: %v", resources, err)
	}

	err = savingFileServer.StorePublication(publication)
	if err != nil {
		t.Errorf("failed to store publication %v, error: %v", publication, err)
	}

	readingFileServer, err := NewFileServer(rootPath)
	if err != nil {
		t.Errorf("failed to instantiate file server at %s, error: %v", rootPath, err)
	}

	tracks, err := readingFileServer.Tracks(mockArtistID)
	if err != nil {
		t.Errorf("failed to get tracks for artist %s from %s, error: %v", mockArtistID, rootPath, err)
	}
	if len(tracks) == 0 {
		t.Errorf("loaded 0 tracks from %s for %s", rootPath, mockArtistID)
	}

}

// TODO: test that TestNameToID is used for all IDs created from external input.

// TestNameToID verifies that TitleToID converts the given name to lower case, strips white space and punctuation,
// leaving only a lower-case string of alphabetic characters, numbers, dashes, and periods.
func TestNameToID(t *testing.T) {
	name := "Alice the-dash-and.dot.Artist"
	id := NameToID(name)
	expected := "alicethe-dash-and.dot.artist"
	if id != expected {
		t.Errorf("Failed to normalized name: \"%s\" to expected: %s, actual: %s", name, expected, id)
	}
}

// TestTitleToHierarchy verifies that TitleToHierarchy converts the given title to lower case,
// strips white space and punctuation except for slashes, etc.
// leaving only a slash-separated series of lower-case strings of alphabetic characters, numbers, dashes, and periods.
func TestTitleToHierarchy(t *testing.T) {
	// TODO: use TitleToHierarchy wherever externally specified strings titles are used to create an ArtistAlbumID, ArtistTrackID, etc.

	title := "Test 'Container' / Sub-container \"quoted\" / Item.1"
	hierarchy := TitleToHierarchy(title)
	expected := "testcontainer/sub-containerquoted/item.1"
	if hierarchy != expected {
		t.Errorf("Failed to normalize title: \"%s\" to expected hierarchy: %s, actual: %s", title, expected, hierarchy)
	}
}

// TestStoreTrack verifies that a Track sent to StoreTrack is retrieved by its unique ArtistId and ArtistTrackId.
func TestStoreTrack(t *testing.T) {
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
		err = fileServer.StoreTrack(&testTrack)
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
		Title:            "Some Track"})
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

	err = fileServer.StoreArtist(&mockArtist)
	if err != nil {
		t.Errorf("StoreArtist %v failed for publisher %v, error: %v", mockArtist, mockPublisher, err)
	}

	err = fileServer.StorePeer(&art.Peer{Pubkey: string(mockPubkey)})
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
	if Pubkey(fetchedPeer.Pubkey) != mockPubkey {
		t.Errorf("Peer fetched had the wrong pubkey (%s), expected %s",
			fetchedPeer.Pubkey, mockPubkey)
	}
}
