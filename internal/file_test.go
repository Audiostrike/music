package audiostrike

import (
	"testing"

	art "github.com/audiostrike/music/pkg/art"
)

const (
	rootPath         = "testart"
)

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

	// Check the db for the track.
	tracks, err := fileServer.AlbumTracks(testArtistId, testAlbumId)
	if err != nil || len(tracks) == 0 {
		t.Errorf("GetAlbumTracks failed to get tracks for \"%s\", got only %v", testAlbumId, tracks)
	}
}
