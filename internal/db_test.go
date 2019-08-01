package audiostrike

import (
	"fmt"
	"os"
	"testing"

	art "github.com/audiostrike/music/pkg/art"
	_ "github.com/go-sql-driver/mysql"
)

const (
	dbName         = "test_austk"
	dbUser         = defaultDbUser
	dbPassword     = ""
	torProxy       = defaultTorProxy
)

func TestMain(m *testing.M) {
	// Get the db config.
	var err error
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config, error: %v\n", err)
		os.Exit(1)
	}

	// Initialize the db if it cannot be opened normally.
	_, err = OpenDb(dbName, dbUser, dbPassword)
	if err != nil {
		err := InitializeDb(dbName, dbUser, dbPassword)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize test DB, error: %v\n", err)
		}
	}

	// Run the test.
	os.Exit(m.Run())
}

func TestPutTrack(t *testing.T) {
	db, err := OpenDb(dbName, dbUser, dbPassword)
	if err != nil {
		t.Errorf("Failed to connect to music DB, error %v", err)
	}
	err = db.Ping()
	if err != nil {
		t.Errorf("Failed to connect to music DB, error %v", err)
	}
	defer db.Close()
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
		err := db.PutTrack(&testTrack)
		if err != nil {
			t.Errorf("PutAlbumTrack failed to put track %s for %s",
				testTrack.ArtistTrackId, testTrack.ArtistId)
		}
		selectedTrack, err := db.SelectTrack(testArtistId, testTrack.ArtistTrackId)
		if err != nil ||
			selectedTrack.AlbumTrackNumber != testTrack.AlbumTrackNumber ||
			selectedTrack.Title != testTrack.Title {
			t.Errorf("PutAlbutTrack failed to select track %v, error: %v, found: %v",
				testTrack, err, selectedTrack)
		}
	}

	// Cleanup
	db.DeleteAlbum(testArtistId, testAlbumId)
}

func TestDeleteAlbum(t *testing.T) {
	db, err := OpenDb(dbName, dbUser, dbPassword)
	if err != nil {
		t.Errorf("Failed to open music DB, error: %v", err)
	}
	defer db.Close()
	const testArtistId = "tester"
	const testAlbumId = "deleting-album"
	err = db.PutAlbum(&art.Album{
		ArtistId: testArtistId,
		ArtistAlbumId: testAlbumId,
		Title:    "Test Album to Delete"})
	if err != nil {
		t.Errorf("PutAlbum error: %v", err)
	}
	err = db.PutTrack(&art.Track{
		ArtistId:         testArtistId,
		ArtistAlbumId:    testAlbumId,
		AlbumTrackNumber: 1,
		ArtistTrackId:    testAlbumId + "-1.SomethingToDelete",
		Title:            "Something to Delete"})
	if err != nil {
		t.Errorf("PutTrack failed to put track for %s", testAlbumId)
	}
	tracks, err := db.SelectAlbumTracks(testArtistId, testAlbumId)
	if err != nil || len(tracks) == 0 {
		t.Errorf("TestDeleteAlbum failed to setup album with a track")
	}
	err = db.DeleteAlbum(testArtistId, testAlbumId)
	if err != nil {
		t.Errorf("DeleteAlbum error %v", err)
	}
	tracks, err = db.SelectAlbumTracks(testArtistId, testAlbumId)
	if err != nil || len(tracks) != 0 {
		t.Errorf("TestDeleteAlbum failed to delete album %v", testAlbumId)
	}
}

func TestGetAlbumTracks(t *testing.T) {
	// Test album with known id
	db, err := OpenDb(dbName, dbUser, dbPassword)
	if err != nil {
		t.Errorf("Failed to open music DB, error: %v", err)
	}
	err = db.Ping()
	if err != nil {
		t.Errorf("Failed to ping music DB, error %v", err)
	}
	defer db.Close()

	const (
		testArtistId = "tester"
		testAlbumId  = "test-get-album-tracks-album-id"
	)

	err = db.PutTrack(&art.Track{
		ArtistId:         testArtistId,
		ArtistAlbumId:    testAlbumId,
		AlbumTrackNumber: 1,
		ArtistTrackId:    testAlbumId + "-1.SomethingToGet",
		Title:            "Something to Get"})
	if err != nil {
		t.Errorf("PutTrack failed to put track for \"%s\"", testAlbumId)
	}

	// Check the db for the track.
	tracks, err := db.SelectAlbumTracks(testArtistId, testAlbumId)
	if err != nil || len(tracks) == 0 {
		t.Errorf("GetAlbumTracks failed to get tracks for \"%s\", got only %v", testAlbumId, tracks)
	}

	// Cleanup
	db.DeleteAlbum(testArtistId, testAlbumId)
}
