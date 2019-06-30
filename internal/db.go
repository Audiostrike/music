package audiostrike

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	// This loads the mysql driver that implements sql.DB below. The namespace is discarded.
	_ "github.com/go-sql-driver/mysql"

	art "github.com/audiostrike/music/pkg/art" // protobuf for artists, albums, peers, tracks
)

// AustkDb persists art data to a sql database and retrieves saved art
type AustkDb struct {
	sqlDb *sql.DB
}

func InitializeDb(sqlDbName string, sqlDbUser string, sqlDbPassword string) error {
	dataSource := fmt.Sprintf("%s:%s@tcp(localhost)/%s", sqlDbUser, sqlDbPassword, sqlDbName)
	sqlDb, err := sql.Open("mysql", dataSource)
	if err != nil {
		return err
	}
	initDbTx, err := sqlDb.Begin()
	if err != nil {
		return err
	}
	_, err = initDbTx.Exec(
		"CREATE TABLE `artist` (" +
			"`artist_id` varchar(32) NOT NULL," +
			"`name` varchar(64) NOT NULL," +
			"`pubkey` char(33) DEFAULT NULL," +
			" PRIMARY KEY (`artist_id`)" +
			")")
	if err != nil {
		initDbTx.Rollback()
		return err
	}
	_, err = initDbTx.Exec(
		"CREATE TABLE `album` (" +
			"`artist_id` varchar(32) NOT NULL," +
			"`artist_album_id` varchar(32) NOT NULL," +
			"`title` varchar(64) NOT NULL," +
			" PRIMARY KEY (`artist_id`, `artist_album_id`)" +
			")")
	if err != nil {
		initDbTx.Rollback()
		return err
	}
	_, err = initDbTx.Exec(
		"CREATE TABLE `track` (" +
			"`artist_id` varchar(32) NOT NULL," +
			"`artist_track_id` varchar(32) NOT NULL," +
			"`title` varchar(64) NOT NULL," +
			"`artist_album_id` varchar(32) DEFAULT NULL," +
			"`album_track_num` tinyint(4) DEFAULT NULL," +
			" PRIMARY KEY (`artist_id`, `artist_track_id`)" +
			")")
	if err != nil {
		initDbTx.Rollback()
		return err
	}
	_, err = initDbTx.Exec(
		"CREATE TABLE `peer` (" +
			"`pubkey` char(33) NOT NULL," +
			"`host` varchar(56) NOT NULL," +
			"`port` smallint(5) unsigned NOT NULL," +
			" PRIMARY KEY (`pubkey`)" +
			")")
	if err != nil {
		initDbTx.Rollback()
		return err
	}
	err = initDbTx.Commit()
	return err
}

// OpenDb instantiates an ArtDb with an open connection to the local sql db.
func OpenDb(sqlDbName string, sqlDbUser string, sqlDbPassword string) (*AustkDb, error) {
	dataSource := fmt.Sprintf("%s:%s@tcp(localhost)/%s", sqlDbUser, sqlDbPassword, sqlDbName)
	sqlDb, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Opened connection to db %v.\n", dataSource)
	db := &AustkDb{sqlDb: sqlDb}
	err = db.verifyReady()
	return db, err
}

// Ping the sql database to verify the connection
func (db *AustkDb) Ping() error {
	return db.sqlDb.Ping()
}

// Close the sqlDb and wait for started queries to finish
func (db *AustkDb) Close() error {
	return db.sqlDb.Close()
}

var ErrNoSchema = errors.New("AustkDb db has no schema")

func (db *AustkDb) hasSchema() (bool, error) {
	tables, err := db.sqlDb.Query("SHOW TABLES")
	if err != nil {
		return false, err
	}
	if tables.Next() {
		return true, nil
	}
	return false, ErrNoSchema
}

func (db *AustkDb) verifyReady() error {
	if db.sqlDb == nil {
		return fmt.Errorf("no sql")
	}
	err := db.sqlDb.Ping()
	if err != nil {
		return err
	}
	_, err = db.hasSchema()

	return err
}

// SelectAllArtists returns all artists from the sqlDb
func (db *AustkDb) SelectAllArtists() (artists map[string]art.Artist, err error) {
	const logPrefix = "db SelectAllArtists "
	err = db.verifyReady()
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"db not ready, error: %v\n", err)
		return nil, err
	}
	artistRows, err := db.sqlDb.Query(
		"SELECT `artist_id`, `name`, `pubkey` FROM `artist`")
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"Query error: %v\n", err)
		return
	}
	defer artistRows.Close()
	artists = make(map[string]art.Artist)
	var (
		artistID string
		name     string
		pubkey   string
	)
	for artistRows.Next() {
		err = artistRows.Scan(&artistID, &name, &pubkey)
		if err != nil {
			fmt.Fprintf(os.Stderr, logPrefix+"Scan error: %v\n", err)
			return
		}
		artists[artistID] = art.Artist{
			ArtistId: artistID,
			Name:     name,
			Pubkey:   pubkey}
	}
	return
}

// SelectPeer returns the peer from the sqlDb with the given pubkey. ErrNoRows is returned if the peer is not yet in the database.
func (db *AustkDb) SelectPeer(pubkey string) (peer *art.Peer, err error) {
	const logPrefix = "db SelectPeer "
	err = db.verifyReady()
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"db not ready, error: %v\n", err)
		return nil, err
	}
	peerRow := db.sqlDb.QueryRow(
		"SELECT `host`, `port` FROM `peer` WHERE `pubkey` = ?", pubkey[:33])
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"QueryRow error: %v\n", err)
		return
	}
	var (
		host string
		port uint32
	)
	err = peerRow.Scan(&host, &port)
	if err == nil {
		peer = &art.Peer{
			Pubkey: pubkey,
			Host:   host,
			Port:   port}
	} else if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"Scan error: %v\n", err)
	}
	return
}

// SelectArtist returns the artist from the sqlDb with the given artistID.
func (db *AustkDb) SelectArtist(artistID string) (artist *art.Artist, err error) {
	const logPrefix = "db SelectArtist "
	err = db.verifyReady()
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"db not ready, error: %v\n", err)
		return nil, err
	}
	artistRow := db.sqlDb.QueryRow(
		"SELECT `name`, `pubkey` FROM `artist` WHERE `artist_id` = ?", artistID)
	if err != nil {
		return artist, err
	}
	var (
		name   string
		pubkey string
	)
	err = artistRow.Scan(&name, &pubkey)
	if err != nil {
		return
	}
	artist = &art.Artist{
		ArtistId: artistID,
		Name:     name,
		Pubkey:   pubkey}
	return
}

// SelectArtistTracks reads all tracks for the specified artist from the db.
func (db *AustkDb) SelectArtistTracks(artistID string) (tracks map[string]art.Track, err error) {
	trackRows, err := db.sqlDb.Query(
		"SELECT `artist_track_id`, `title`, `artist_album_id`, `album_track_num`"+
			" FROM `track`"+
			" WHERE `artist_id` = ?",
		artistID)
	if err != nil {
		return
	}
	defer trackRows.Close()
	tracks = make(map[string]art.Track)
	var (
		trackID       string
		title         string
		artistAlbumID string
		albumTrackNum uint
	)
	for trackRows.Next() {
		err = trackRows.Scan(&trackID, &title, &artistAlbumID, &albumTrackNum)
		if err != nil {
			return
		}
		tracks[trackID] = art.Track{
			ArtistId:         artistID,
			ArtistTrackId:    trackID,
			Title:            title,
			ArtistAlbumId:    artistAlbumID,
			AlbumTrackNumber: uint32(albumTrackNum)}
	}
	return
}

// SelectAlbumTracks returns the db tracks from the album with the given albumID
func (db *AustkDb) SelectAlbumTracks(artistID string, artistAlbumID string) (tracks map[string]art.Track, err error) {
	trackRows, err := db.sqlDb.Query(
		"SELECT `artist_track_id`, `title`, `album_track_num`"+
			" FROM `track`"+
			" WHERE `artist_id` = ? AND `artist_album_id` = ?",
		artistID, artistAlbumID)
	if err != nil {
		return
	}
	defer trackRows.Close()
	tracks = make(map[string]art.Track)
	var (
		trackID       string
		name          string
		albumTrackNum uint
	)
	for trackRows.Next() {
		err = trackRows.Scan(&trackID, &name, &albumTrackNum)
		if err != nil {
			return
		}
		tracks[trackID] = art.Track{
			Title:            name,
			ArtistId:         artistID,
			ArtistAlbumId:    artistAlbumID,
			AlbumTrackNumber: uint32(albumTrackNum)}
	}
	return
}

// PutPeer INSERTs or UPDATEs the peer with the given pubkey.
func (db *AustkDb) PutPeer(peer *art.Peer) (err error) {
	const logPrefix = "db PutPeer "
	existingPeer, err := db.SelectPeer(peer.Pubkey)
	if err == sql.ErrNoRows {
		_, err = db.sqlDb.Exec(
			"INSERT `peer`(`pubkey`, `host`, `port`)"+
				" VALUES(?, ?, ?)",
			peer.Pubkey, peer.Host, peer.Port)
		if err != nil {
			fmt.Fprintf(os.Stderr, logPrefix+"sqlDb.Exec error: %v\n", err)
			return
		}
	} else if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"SelectPeer error: %v\n", err)
		return
	} else {
		fmt.Printf(logPrefix+"Update peer %v from %v:%d to %v:%d\n",
			peer.Pubkey,
			existingPeer.Host, existingPeer.Port,
			peer.Host, peer.Port)
		_, err = db.sqlDb.Exec(
			"UPDATE `peer`"+
				" SET `host` = ?, `port` = ?"+
				" WHERE `pubkey` = ?",
			peer.Host, peer.Port, peer.Pubkey[:33])
		if err != nil {
			fmt.Fprintf(os.Stderr, logPrefix+"sqlDb.Exec error: %v\n", err)
			return
		}
		updatedPeer, err := db.SelectPeer(peer.Pubkey)
		if err != nil {
			fmt.Fprintf(os.Stderr, logPrefix+"SelectPeer error: %v\n", err)
			return err
		}
		fmt.Printf(logPrefix+"updated to host %v\n", updatedPeer.Host)
	}
	return
}

// PutArtist INSERTs or UPDATEs the specified artist using artist_id as the unique key.
func (db *AustkDb) PutArtist(artist *art.Artist) (err error) {
	const logPrefix = "db PutArtist "
	artists, err := db.SelectAllArtists()
	if err != nil {
		return
	}
	_, isReplacing := artists[artist.ArtistId]
	if isReplacing {
		_, err = db.sqlDb.Exec(
			"UPDATE `artist`"+
				" SET `name` = ?, `pubkey` = ?"+
				" WHERE `artist_id` = ?",
			artist.Name, artist.Pubkey, artist.ArtistId)
	} else {
		_, err = db.sqlDb.Exec(
			"INSERT `artist`(`artist_id`, `name`, `pubkey`)"+
				" VALUES(?, ?, ?)",
			artist.ArtistId, artist.Name, artist.Pubkey)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"Exec sql error: %v\n", err)
		return
	}
	artists[artist.ArtistId] = *artist
	return
}

// PutAlbum inserts or updates the specified album using its artist_id and artist_album_id as the unique key
func (db *AustkDb) PutAlbum(album art.Album) (err error) {
	_, err = db.SelectAlbum(album.ArtistId, album.ArtistAlbumId)
	if err == sql.ErrNoRows {
		db.sqlDb.Exec("INSERT `album`(`artist_id`, `artist_album_id`, `title`)"+
			" VALUES(?, ?, ?)",
			album.ArtistId, album.ArtistAlbumId, album.Title)
	} else if err != nil {
		return
	} else {
		db.sqlDb.Exec("UPDATE `album`"+
			" SET `title` = ?"+
			" WHERE `artist_id` = ? AND `artist_album_id` = ?",
			album.Title, album.ArtistId, album.ArtistAlbumId)
	}
	return
}

// PutTrack inserts or updates the specified track using its id as the unique key
func (db *AustkDb) PutTrack(track art.Track) (albumTrackCount int, err error) {
	albumTracks, err := db.SelectAlbumTracks(track.ArtistId, track.ArtistAlbumId)
	if err != nil {
		return
	}
	_, isReplacing := albumTracks[track.ArtistTrackId]
	if isReplacing {
		db.sqlDb.Exec("UPDATE `track`"+
			"SET `title` = ?, `artist_album_id` = ?, `album_track_num` = ?"+
			"WHERE `artist_id` = ? AND `artist_track_id` = ?",
			track.Title, track.ArtistAlbumId, track.AlbumTrackNumber,
			track.ArtistId, track.ArtistTrackId)
	} else {
		_, err := db.sqlDb.Exec(
			"INSERT `track`(`artist_id`, `artist_track_id`,"+
				" `title`, `artist_album_id`, `album_track_num`)"+
				" VALUES(?, ?, ?, ?, ?)",
			track.ArtistId, track.ArtistTrackId,
			track.Title, track.ArtistAlbumId, len(albumTracks)+1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "AustkDb PutTrack failed INSERT, error: %v\n", err)
			return len(albumTracks), err
		}
	}
	albumTracks[track.ArtistTrackId] = track
	albumTrackCount = len(albumTracks)
	return
}

// DeleteAlbum deletes the tracks and album with the specified artistID and albumID from the track and album tables.
func (db *AustkDb) DeleteAlbum(artistID string, albumID string) (err error) {
	// TODO: check whether request is authorized to delete existing track.
	// Each request can carry an auth token signed by the artist whose
	// public key was used to issue the track.
	result, err := db.sqlDb.Exec(
		"DELETE FROM `track`"+
			" WHERE `artist_id` = ? AND `artist_album_id` = ?",
		artistID, albumID)
	if err != nil {
		fmt.Printf("AustkDb DeleteAlbum failed DELETE from album %v, error: %v\n", albumID, err)
		return err
	}
	result, err = db.sqlDb.Exec(
		"DELETE FROM `album`"+
			" WHERE `artist_id` = ? AND `artist_album_id` = ?",
		artistID, albumID)
	albumDeletedCount, err := result.RowsAffected()
	if albumDeletedCount == 0 {
		err = fmt.Errorf("AustkDb DeleteAlbum album not found, id: %s", albumID)
	}

	return err
}

// SelectAlbum returns the metadata of the db album for the given artist with the given albumID
func (db *AustkDb) SelectAlbum(artistID string, albumID string) (*art.Album, error) {
	albumRow := db.sqlDb.QueryRow(
		"SELECT `title`"+
			" FROM `album`"+
			" WHERE `artist_id` = ? AND `artist_album_id` = ?",
		artistID, albumID)
	var title string
	err := albumRow.Scan(&title)
	if err != nil {
		return nil, err
	}

	return &art.Album{ArtistId: artistID, ArtistAlbumId: albumID, Title: title}, nil
}

// SelectAlbum returns the metadata of the db album for the given artist with the given albumID
func (db *AustkDb) SelectArtistAlbums(artistID string) (map[string]*art.Album, error) {
	albumRows, err := db.sqlDb.Query(
		"SELECT `artist_album_id`, `title`"+
			" FROM `album`"+
			" WHERE `artist_id` = ?",
		artistID)
	if err != nil {
		return nil, err
	}
	defer albumRows.Close()
	albums := make(map[string]*art.Album)
	var (
		artistAlbumID string
		title         string
	)
	for albumRows.Next() {
		err = albumRows.Scan(&artistAlbumID, &title)
		if err != nil {
			return nil, err
		}
		albums[artistAlbumID] = &art.Album{
			ArtistId:      artistID,
			ArtistAlbumId: artistAlbumID,
			Title:         title,
		}
	}

	return albums, nil
}

func (db *AustkDb) SelectAllPeers() ([]*art.Peer, error) {
	const logPrefix = "db SelectAllPeers "
	peerRows, err := db.sqlDb.Query(
		"SELECT `host`, `port`, `pubkey` FROM `peer`")
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"Query error: %v\n", err)
		return nil, err
	}
	defer peerRows.Close()
	peers := make([]*art.Peer, 0)
	var (
		host   string
		port   uint32
		pubkey string
	)
	for peerRows.Next() {
		err = peerRows.Scan(&host, &port, &pubkey)
		if err != nil {
			return nil, err
		}
		peers = append(peers, &art.Peer{
			Host:   host,
			Port:   port,
			Pubkey: pubkey,
		})
	}

	return peers, nil
}

// SelectTrack returns the metadata of the db track for the given artist with the given trackID
func (db *AustkDb) SelectTrack(artistID string, artistTrackID string) (track art.Track, err error) {
	trackRow := db.sqlDb.QueryRow(
		"SELECT `title`, `artist_album_id`, `album_track_num`"+
			" FROM `track`"+
			" WHERE `artist_id` = ? AND `artist_track_id` = ?",
		artistID, artistTrackID)
	var (
		title         string
		artistAlbumID string
		albumTrackNum uint
	)
	err = trackRow.Scan(&title, &artistAlbumID, &albumTrackNum)
	if err != nil {
		return
	}
	track = art.Track{
		ArtistId:         artistID,
		ArtistTrackId:    artistTrackID,
		Title:            title,
		ArtistAlbumId:    artistAlbumID,
		AlbumTrackNumber: uint32(albumTrackNum),
	}
	return
}
