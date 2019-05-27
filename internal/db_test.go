package audiostrike

import (
	"flag"
	"fmt"
	"os"
	"testing"

	art "github.com/audiostrike/music/pkg/art"
	_ "github.com/go-sql-driver/mysql"
)

var cfg *Config

func TestMain(m *testing.M) {
	// Get the db config.
	var err error
	cfg, err = LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config, error: %v\n", err)
		os.Exit(1)
	}
	var (
		dbName     = flag.String("dbname", cfg.DbName, "DB name (default: \"music\")")
		dbUser     = flag.String("dbuser", cfg.DbUser, "DB username (default: \"artist\")")
		dbPassword = flag.String("dbpass", cfg.DbPassword, "database password")
	)
	flag.Parse()
	cfg.DbName = *dbName
	cfg.DbUser = *dbUser
	cfg.DbPassword = *dbPassword

	// Initialize the db if it cannot be opened normally.
	_, err = OpenDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
	if err != nil {
		err := InitializeDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize test DB, error: %v\n", err)
		}
	}

	// Run the test.
	os.Exit(m.Run())
}

func TestPutTrack(t *testing.T) {
	db, err := OpenDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
	if err != nil {
		t.Errorf("Failed to connect to music DB, error %v", err)
	}
	err = db.Ping()
	if err != nil {
		t.Errorf("Failed to connect to music DB, error %v", err)
	}
	defer db.Close()
	albumID := "test-put-track-album-id"
	testTrackId := albumID + "-1.TestPut1"
	count, err := db.PutTrack(art.Track{
		Album:            albumID,
		AlbumTrackNumber: 1,
		Id:               testTrackId,
		Title:            "Test Put 1"})
	if err != nil || count == 0 {
		t.Errorf("PutAlbumTrack failed to put track for \"%s\", error: %v", albumID, err)
	}
	testTrackId = albumID + "-2.TestPut2"
	count, err = db.PutTrack(art.Track{
		Album:            albumID,
		AlbumTrackNumber: 2,
		Id:               testTrackId,
		Title:            "Test Put 2"})
	if err != nil || count == 0 {
		t.Errorf("PutAlbumTrack failed to put track for \"%s\"", albumID)
	}
	testTrackId = albumID + "-3.TestPut3"
	count, err = db.PutTrack(art.Track{
		Album:            albumID,
		AlbumTrackNumber: 3,
		Id:               testTrackId,
		Title:            "Test Put 3"})
	if err != nil || count == 0 {
		t.Errorf("PutAlbumTrack failed to put track for \"%s\"", albumID)
	}
	if count != 3 {
		t.Errorf("PutAlbumTrack failed to put 3 tracks for \"%s\", found %v", albumID, count)
	}
	testTrack, err := db.SelectTrack(testTrackId)
	if err != nil || testTrack.AlbumTrackNumber != 3 {
		t.Errorf("PutAlbutTrack failed to select track 3, error: %v, testTrack: %v", err, testTrack)
	}
}

func TestDeleteAlbum(t *testing.T) {
	db, err := OpenDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
	if err != nil {
		t.Errorf("Failed to open music DB, error: %v", err)
	}
	defer db.Close()
	albumID := "test-delete-album-album-id"
	count, err := db.PutTrack(art.Track{
		Album:            albumID,
		AlbumTrackNumber: 1,
		Id:               albumID + "-1.SomethingToDelete",
		Title:            "Something to Delete"})
	if err != nil || count == 0 {
		t.Errorf("PutAlbumTrack failed to put track for \"%s\"", albumID)
	}
	tracks, err := db.SelectAlbumTracks(albumID)
	if err != nil || len(tracks) == 0 {
		t.Errorf("TestDeleteAlbum failed to setup album with a track")
	}
	db.DeleteAlbum(albumID)
	tracks, err = db.SelectAlbumTracks(albumID)
	if err != nil || len(tracks) != 0 {
		t.Errorf("TestDeleteAlbum failed to delete album %v", albumID)
	}
}

func TestGetAlbumTracks(t *testing.T) {
	// Test album with known id
	db, err := OpenDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
	if err != nil {
		t.Errorf("Failed to open music DB, error: %v", err)
	}
	err = db.Ping()
	if err != nil {
		t.Errorf("Failed to ping music DB, error %v", err)
	}
	defer db.Close()
	albumID := "test-get-album-tracks-album-id"
	count, err := db.PutTrack(art.Track{
		Album:            albumID,
		AlbumTrackNumber: 1,
		Id:               albumID + "-1.SomethingToGet",
		Title:            "Something to Get"})
	if err != nil || count == 0 {
		t.Errorf("PutTrack failed to put track for \"%s\"", albumID)
	}
	tracks, err := db.SelectAlbumTracks(albumID)
	if err != nil || len(tracks) == 0 {
		t.Errorf("GetAlbumTracks failed to get tracks for \"%s\", got only %v", albumID, tracks)
	}
}
