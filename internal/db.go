package audiostrike

import (
	"database/sql"
	"errors"
	"fmt"

	// This loads the mysql driver that implements sql.DB below. The namespace is discarded.
	_ "github.com/go-sql-driver/mysql"

	art "github.com/audiostrike/music/pkg/art" // protobuf for artists, albums, peers, tracks
	"log"
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
			"`pubkey` char(66) DEFAULT NULL," +
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
			"`pubkey` char(66) NOT NULL," +
			"`host` varchar(62) NOT NULL," +
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
	const logPrefix = "db OpenDb "

	dataSource := fmt.Sprintf("%s:%s@tcp(localhost)/%s", sqlDbUser, sqlDbPassword, sqlDbName)
	sqlDb, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, err
	}
	log.Printf(logPrefix+"Opened connection to db %v", dataSource)

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
		log.Printf(logPrefix+"db not ready, error: %v", err)
		return nil, err
	}
	artistRows, err := db.sqlDb.Query(
		"SELECT `artist_id`, `name`, `pubkey` FROM `artist`")
	if err != nil {
		log.Printf(logPrefix+"Query error: %v", err)
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
			log.Printf(logPrefix+"row Scan error: %v", err)
			return
		}
		artists[artistID] = art.Artist{
			ArtistId: artistID,
			Name:     name,
			Pubkey:   pubkey}
	}
	return
}

// SelectPeer returns the peer from the sqlDb with the given pubkey.
// ErrNoRows is returned if the peer is not yet in the database.
func (db *AustkDb) SelectPeer(pubkey string) (peer *art.Peer, err error) {
	const logPrefix = "db SelectPeer "

	err = db.verifyReady()
	if err != nil {
		log.Printf(logPrefix+"db not ready, error: %v", err)
		return nil, err
	}

	peerRow := db.sqlDb.QueryRow(
		"SELECT `host`, `port` FROM `peer` WHERE `pubkey` = ?", pubkey)
	if err != nil {
		log.Printf(logPrefix+"QueryRow error: %v", err)
		return nil, err
	}

	var (
		host string
		port uint32
	)
	err = peerRow.Scan(&host, &port)
	if err != nil {
		log.Printf(logPrefix+"row Scan error: %v", err)
		return nil, err
	}

	peer = &art.Peer{
		Pubkey: pubkey,
		Host:   host,
		Port:   port}
	return peer, nil
}

// SelectArtist returns the artist from the sqlDb with the given artistID.
func (db *AustkDb) SelectArtist(artistID string) (artist *art.Artist, err error) {
	const logPrefix = "db SelectArtist "

	err = db.verifyReady()
	if err != nil {
		log.Printf(logPrefix+"db not ready, error: %v", err)
		return nil, err
	}

	artistRow := db.sqlDb.QueryRow(
		"SELECT `name`, `pubkey` FROM `artist` WHERE `artist_id` = ?", artistID)
	if err != nil {
		return nil, err
	}

	var (
		name   string
		pubkey string
	)
	err = artistRow.Scan(&name, &pubkey)
	if err != nil {
		return nil, err
	}

	artist = &art.Artist{
		ArtistId: artistID,
		Name:     name,
		Pubkey:   pubkey}
	return artist, nil
}

// SelectArtistTracks gets all tracks from the specified artist from the db.
//
// Tracks are organized by artist. A given ArtistTrackId is only unique for a given ArtistId.
// Tracks optionally use ArtistAlbumID to define albums for the given artist.
// If so, they should have an AlbumTrackNumber for the track sequence on that album.
// For example, Alice in Chains could run an austk node and add .mp3 files that they own/produced:
//  Alice_in_Chains/We_Die_Young_(Single).mp3
//  Alice_in_Chains/Man_in_the_Box_(Single).mp3
//  Alice_in_Chains/Facelift/01.We_Die_Young.mp3
//  Alice_in_Chains/Facelift/02.Man_in_the_Box.mp3
//  Alice_in_Chains/Facelift/03.Sea_of_Sorrow.mp3
//  ...
//  Alice_in_Chains/Facelift/12.Real_Thing.mp3
//  Alice_in_Chains/Dirt/01.Them_Bones.mp3
//  ...
//  Alice_in_Chains/Dirt/13.Would.mp3
//
// Alice in Chains could add their mp3 files to their austk node to publish 11 tracks as
// owned/hosted by and payable to Alice in Chains' node:
// [
//  {ArtistID:"aliceinchains", ArtistTrackID:"wedieyoung", Title:"We Die Young"},
//  {ArtistID:"aliceinchains", ArtistTrackID:"maninthebox", Title:"Man in the Box"},
//  {ArtistID:"aliceinchains", ArtistTrackID:"facelift/wedieyoung", Title:"We Die Young", AlbumTrackNumber:1},
//  {ArtistID:"aliceinchains", ArtistTrackID:"facelift/maninthebox", Title:"Man in the Box", AlbumTrackNumber:2},
//  {ArtistID:"aliceinchains", ArtistTrackID:"facelift/seaofsorrow", Title:"Sea of Sorrow", AlbumTrackNumber:3},
//  ...
//  {ArtistID:"aliceinchains", ArtistTrackID:"facelift/realthing", Title:"Real Thing", AlbumTrackNumber:12},
//  {ArtistID:"aliceinchains", ArtistTrackID:"dirt/thembones", Title:"Them Bones", AlbumTrackNumber:1},
//  ...
//  {ArtistID:"aliceinchains", ArtistTrackID:"dirt/would", Title:"Would?", AlbumTrackNumber:13},
// ]
//
// Tracks that are in albums have ArtistTrackId prefixed with ArtistAlbumId and a slash, e.g. "dirt/would".
// The url or file path for a track has ArtistId + '/' + ArtistTrackId.
// For example, the above tracks would be hsoted in url paths like these:
//   http.../aliceinchains/wedieyoung
//   http.../aliceinchains/maninthebox
//   http.../aliceinchains/facelift/wedieyoung
//   ...
//   http.../aliceinchains/dirt/would
//
// The austk node run by Alice in Chains would copy those .mp3 files to paths like these:
//   ./tracks/aliceinchains/wedieyoung.mp3
//   ./tracks/aliceinchains/maninthebox.mp3
//   ./tracks/aliceinchains/facelift/wedieyoung.mp3
//   ...
//   ./tracks/aliceinchains/dirt/would.mp3
//
// Alice in Chains' austk node would then serve those .mp3 files or streams of them
// to any fan whose Audiostrike client presents proof of payment of an invoice
// issued by the austk node hosting the requested tracks.
//
// For tracks in albums, the ArtistAlbumId is stored in the mysql database as a distinct field
// for the purpose of normalizing the album details into the albums table.
// The album identifier is also present in the ArtistTrackId in order to accomodate multiple "releases"
// of a given title and alternative hierarchies of music for a given artist.
// For example, an artist may release multiple episodes (each as a track) in each season of a series.
// Such tracks could have ArtistTrackId and AlbumTrackNumber values like these:
//   [
//    {ArtistId:"stephanlivera",
//     ArtistTrackId:"slp/bitcoin2019/connerfromknechtbitcoinlightningwatchtowers",
//     AlbumTrackNumber:81},
//    {ArtistId:"stephanlivera",
//     ArtistTrackId:"slp/bitcoin2019/sergejkotliarbuildingbitcoinscirculareconomy",
//     AlbumTrackNumber:80},
//    ...
//    {ArtistId:"stephanlivera",
//     ArtistTrackId:"slp/bitcoin2018/bitcoinassoundmoneywithsaifedeanammous",
//     AlbumTrackNumber:1},
//   ]
//
// An Audiostrike client could then give the fan the option to play an episode, a whole season,
// or the whole podcast in order of the AlbumTrackNumber values.
//
// Music artists could use additional slashes for additional organization containers/hierarchies.
// For example, a music artist may cut an episode into multiple tracks, each episode then has multiple tracks,
// each season having multiple episodes, and each series having multiple seasons.
//
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
func (db *AustkDb) SelectAlbumTracks(artistID string, artistAlbumID string) (tracks map[string]*art.Track, err error) {
	trackRows, err := db.sqlDb.Query(
		"SELECT `artist_track_id`, `title`, `album_track_num`"+
			" FROM `track`"+
			" WHERE `artist_id` = ? AND `artist_album_id` = ?",
		artistID, artistAlbumID)
	if err != nil {
		return
	}
	defer trackRows.Close()
	tracks = make(map[string]*art.Track)
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
		tracks[trackID] = &art.Track{
			Title:            name,
			ArtistId:         artistID,
			ArtistAlbumId:    artistAlbumID,
			AlbumTrackNumber: uint32(albumTrackNum)}
	}
	return
}

// PutPeer INSERTs or UPDATEs the peer with the given pubkey.
func (db *AustkDb) PutPeer(peer *art.Peer) error {
	const logPrefix = "db PutPeer "

	existingPeer, err := db.SelectPeer(peer.Pubkey)
	if err == sql.ErrNoRows {
		_, err = db.sqlDb.Exec(
			"INSERT `peer`(`pubkey`, `host`, `port`)"+
				" VALUES(?, ?, ?)",
			peer.Pubkey, peer.Host, peer.Port)
		if err != nil {
			log.Printf(logPrefix+"sqlDb.Exec error: %v", err)
			return err
		}
		// Successfully inserted the new peer, so return early.
		return nil
	} else if err != nil {
		log.Printf(logPrefix+"SelectPeer error: %v", err)
		return err
	}

	if existingPeer.Host != peer.Host || existingPeer.Port != peer.Port {
		_, err = db.sqlDb.Exec(
			"UPDATE `peer`"+
				" SET `host` = ?, `port` = ?"+
				" WHERE `pubkey` = ?",
			peer.Host, peer.Port, peer.Pubkey)
		if err != nil {
			log.Printf(logPrefix+"sqlDb.Exec error: %v", err)
			return err
		}

		updatedPeer, err := db.SelectPeer(peer.Pubkey)
		if err != nil {
			log.Printf(logPrefix+"SelectPeer error: %v", err)
			return err
		}
		log.Printf(logPrefix+"Updated peer %v from %v:%d to %v:%d",
			peer.Pubkey,
			existingPeer.Host, existingPeer.Port,
			updatedPeer.Host, updatedPeer.Port)
	}
	
	return nil
}

// PutArtist INSERTs or UPDATEs the specified artist using artist.id as the unique key..
func (db *AustkDb) PutArtist(artist *art.Artist) error {
	const logPrefix = "db PutArtist "

	artists, err := db.SelectAllArtists()
	if err != nil {
		return err
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
		log.Printf(logPrefix+"Exec sql error: %v\n", err)
		return err
	}

	return nil
}

// UpdateArtistPubkey sets the pubkey of the artist record for the given artistID.
//
// This lets the artist publish music verifiably from that pubkey such that fans
// can confidently pay this artist's austk node for the tracks
// if the fan knows that the artist owns that pubkey.
//
// This INSERTs the artist if artistID is not yet in the artist db table.
// If the artist already is in the db, PutArtist UPDATEs the record.
// Otherwise, this throws an error that the artist is not found in the db.
func (db *AustkDb) UpdateArtistPubkey(artistID string, pubkey string) error {
	const logPrefix = "austk setPubkeyForArtist "

	artist, err := db.SelectArtist(artistID)
	if err != nil {
		log.Printf(logPrefix+"selectArtist %v, error: %v", artistID, err)
		return err
	}

	artist.Pubkey = pubkey

	err = db.PutArtist(artist)
	if err != nil {
		log.Printf(logPrefix+"PutArtist Pubkey %v, error: %v", artist.Pubkey, err)
	}
	return err
}

// PutAlbum INSERTs or UPDATEs the specified album using its artist_id and artist_album_id as the unique key
func (db *AustkDb) PutAlbum(album *art.Album) (err error) {
	_, err = db.SelectAlbum(album.ArtistId, album.ArtistAlbumId)
	if err == sql.ErrNoRows {
		_, err = db.sqlDb.Exec("INSERT `album`(`artist_id`, `artist_album_id`, `title`)"+
			" VALUES(?, ?, ?)",
			album.ArtistId, album.ArtistAlbumId, album.Title)
	} else if err != nil {
		return
	} else {
		_, err = db.sqlDb.Exec("UPDATE `album`"+
			" SET `title` = ?"+
			" WHERE `artist_id` = ? AND `artist_album_id` = ?",
			album.Title, album.ArtistId, album.ArtistAlbumId)
	}
	return
}

// AddArtistAndTrack puts the db records for the given artist and track.
// It preserves any existing artist record's pubkey, failing if the new artist pubkey is different.
func (db *AustkDb) AddArtistAndTrack(artist *art.Artist, track *art.Track) error {
	const logPrefix = "db PutTrackForArtist "

	dbArtist, err := db.SelectArtist(artist.ArtistId)
	// If artist is already in the db, keep the Pubkey.
	// Ignore ErrNoRows. Artist is not yet been in the db table, so no pubkey is known to keep.
	if err == nil {
		// Keep the Pubkey from the db.
		// TODO: prompt user or otherwise resolve conflicting pubkey for this artist.
		if artist.Pubkey != "" && artist.Pubkey != dbArtist.Pubkey {
			err = fmt.Errorf("Artist %s already has pubkey %s",
				artist.ArtistId, dbArtist.Pubkey)
			log.Printf(logPrefix+"Reject pubkey update to %s, error: %v", artist.Pubkey, err)
			return err
		}
		artist.Pubkey = dbArtist.Pubkey
	} else if err != sql.ErrNoRows {
		log.Printf(logPrefix+"Failed so select artist %s, error: %v", artist.ArtistId, err)
		return err
	}
	err = db.PutArtist(artist)
	if err != nil {
		log.Printf(logPrefix+"PutArtist %v, error: %v", artist, err)
		return err
	}

	err = db.PutTrack(track)
	return err
}

// PutTrack INSERTs or UPDATEs the track with the ArtistId and ArtistAlbumId as a composite key.
func (db *AustkDb) PutTrack(track *art.Track) (err error) {
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
			log.Printf("Exec INSERT track failed, error: %v", err)
			return err
		}
	}
	return nil
}

// DeleteAlbum DELETEs the tracks and album with the specified artistID and albumID
// from the track and album tables.
func (db *AustkDb) DeleteAlbum(artistID string, albumID string) error {
	// TODO: check whether request is authorized to delete existing track.
	// Each request can carry an auth token signed by the artist whose
	// public key was used to issue the track.
	result, err := db.sqlDb.Exec(
		"DELETE FROM `track`"+
			" WHERE `artist_id` = ? AND `artist_album_id` = ?",
		artistID, albumID)
	if err != nil {
		log.Printf("Exec DELETE track from artist %v on album %v, error: %v",
			artistID, albumID, err)
		return err
	}

	result, err = db.sqlDb.Exec(
		"DELETE FROM `album`"+
			" WHERE `artist_id` = ? AND `artist_album_id` = ?",
		artistID, albumID)
	albumDeletedCount, err := result.RowsAffected()
	if albumDeletedCount == 0 {
		return fmt.Errorf("AustkDb DeleteAlbum album not found, id: %s", albumID)
	}

	return nil
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
func (db *AustkDb) SelectArtistAlbums(artistID string) (map[string]art.Album, error) {
	albumRows, err := db.sqlDb.Query(
		"SELECT `artist_album_id`, `title`"+
			" FROM `album`"+
			" WHERE `artist_id` = ?",
		artistID)
	if err != nil {
		return nil, err
	}
	defer albumRows.Close()
	albums := make(map[string]art.Album)
	var (
		artistAlbumID string
		title         string
	)
	for albumRows.Next() {
		err = albumRows.Scan(&artistAlbumID, &title)
		if err != nil {
			return nil, err
		}
		albums[artistAlbumID] = art.Album{
			ArtistId:      artistID,
			ArtistAlbumId: artistAlbumID,
			Title:         title,
		}
	}

	return albums, nil
}

// SelectAllPeers selects an array of all the Peer records.
func (db *AustkDb) SelectAllPeers() ([]*art.Peer, error) {
	const logPrefix = "db SelectAllPeers "
	peerRows, err := db.sqlDb.Query(
		"SELECT `host`, `port`, `pubkey` FROM `peer`")
	if err != nil {
		log.Printf(logPrefix+"Query error: %v", err)
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
func (db *AustkDb) SelectTrack(artistID string, artistTrackID string) (*art.Track, error) {
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
	err := trackRow.Scan(&title, &artistAlbumID, &albumTrackNum)
	if err != nil {
		return nil, err
	}
	return &art.Track{
		ArtistId:         artistID,
		ArtistTrackId:    artistTrackID,
		Title:            title,
		ArtistAlbumId:    artistAlbumID,
		AlbumTrackNumber: uint32(albumTrackNum),
	}, nil
}
