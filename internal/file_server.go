package audiostrike

import (
	"bytes"
	"fmt"
	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

type FileServer struct {
	rootPath string
	// peers indexed by pubkey
	peers map[string]*art.Peer
	// artists indexed by ArtistId
	artists map[string]*art.Artist
	// albums indexed by ArtistId then by ArtistAlbumId
	albums map[string]map[string]*art.Album
	// tracks indexed by ArtistId then by ArtistTrackId
	tracks map[string]map[string]*art.Track
	// tracks indexed by ArtistId then by ArtistAlbumId then by AlbumTrackNumber
	albumTracks map[string]map[string]map[uint32]*art.Track
}

var (
	artistFileRegex     *regexp.Regexp = regexp.MustCompile("^(?P<root>.*)/(?P<ArtistID>[0-9a-z]+)/(?P<file>.*)")
	artistPubFileRegex  *regexp.Regexp = regexp.MustCompile("^(?P<root>.*)/(?P<ArtistID>[0-9a-z]+)/(?P<Pubkey>[0-9a-f]+)[.]pub$")
	artistTrackMp3Regex *regexp.Regexp = regexp.MustCompile("^(?P<root>.*)/(?P<ArtistID>[0-9a-z]+)/(?P<ArtistTrackID>[0-9a-z/]+)[.]mp3$")
)

// NewFileServer creates a new FileServer to save and serve art in sudirectories of artDirPath.
func NewFileServer(artDirPath string) (*FileServer, error) {
	const logPrefix = "NewFileServer "

	fileServer := FileServer{
		rootPath:    artDirPath,
		artists:     make(map[string]*art.Artist),
		tracks:      make(map[string]map[string]*art.Track),
		albums:      make(map[string]map[string]*art.Album),
		albumTracks: make(map[string]map[string]map[uint32]*art.Track),
		peers:       make(map[string]*art.Peer),
	}

	err := filepath.Walk(artDirPath, fileServer.readArtFile)
	if err != nil {
		log.Fatalf(logPrefix+"Failed to read art directory, error: %v", err)
		return nil, err
	}
	return &fileServer, nil
}

func (fileServer *FileServer) readArtFile(path string, fileInfo os.FileInfo, err error) error {
	const logPrefix = "FileServer readArtFile "

	if path == fileServer.rootPath {
		return filepath.SkipDir
	}

	// Read the .art file as art.ArtResources
	// ArtistTrackID may be simple alphanumeric identifier
	// or it may optionally include album or other slash-separated hierarchy.
	// Optionally order tracks and albums with numeric prefixes.

	// All the records to read are associated with an artist.
	// First identify that artist by the name of the directory
	artistFileMatchGroups := artistFileRegex.FindStringSubmatch(path)
	if len(artistFileMatchGroups) < 4 {
		log.Printf(logPrefix+"skip non-artist file: %s", path)
		return filepath.SkipDir
	}
	prefix := artistFileMatchGroups[1]
	if prefix != fileServer.rootPath {
		log.Printf(logPrefix+"unexpected read attempt on %s instead of root %s",
			prefix, fileServer.rootPath)
		return filepath.SkipDir
	}
	artistID := artistFileMatchGroups[2]

	// Then check wither this is the .pub file with metadata signed by the artist's pubkey.
	pubFileMatchGroups := artistPubFileRegex.FindStringSubmatch(path)
	if len(pubFileMatchGroups) == 4 {
		// This is the artist's .pub file.
		// Read the art records from that file and validate the signature against the pubkey.
		pubFileArtistID := pubFileMatchGroups[2]
		if pubFileArtistID != artistID { // sanity check
			return fmt.Errorf("unexpected ArtistID %s in %s for artist %s",
				pubFileArtistID, path, artistID)
		}
		pubkey := pubFileMatchGroups[3]
		log.Printf(logPrefix+"pub file %s for artistId: %s, pubkey: %s}",
			path, artistID, pubkey)
		publication, err := fileServer.readPublication(artistID, pubkey, path)
		if err != nil {
			log.Fatalf("failed to read artist %s publication %s, error: %v", artistID, path, err)
			return err
		}
		artResources, err := ValidatePublication(publication)

		log.Printf("validated publication as art resources: %v", artResources)
		return nil
	}

	// Finally, check whether this is an .mp3 file published by the artist.
	artistTrackMp3MatchGroups := artistTrackMp3Regex.FindStringSubmatch(path)
	if len(artistTrackMp3MatchGroups) == 4 {
		// mp3 file
		trackID := artistTrackMp3MatchGroups[3]
		log.Printf(logPrefix+"matched mp3 %s as track ID %s for %s",
			path, trackID, artistID)
	} else {
		return fmt.Errorf("Unknown file type: %s", path)
	}
	return nil
}

// store saves a file with the given artist's details, albums, tracks, and peers.
func (fileServer *FileServer) storeAllArtists() (*art.ArtResources, error) {
	const logPrefix = "fileServer storeAllArtists "

	artists := make([]*art.Artist, 0, len(fileServer.artists))
	albums := make([]*art.Album, 0)
	tracks := make([]*art.Track, len(fileServer.tracks))
	for _, artist := range fileServer.artists {
		artists = append(artists, artist)
		for _, album := range fileServer.albums[artist.ArtistId] {
			albums = append(albums, album)
		}
		for _, track := range fileServer.tracks[artist.ArtistId] {
			tracks = append(tracks, track)
		}
	}

	peers := make([]*art.Peer, 0, len(fileServer.peers))
	for _, peer := range fileServer.peers {
		peers = append(peers, peer)
	}

	resources := art.ArtResources{
		Artists: artists,
		Albums:  albums,
		Tracks:  tracks,
		Peers:   peers,
	}
	return &resources, fmt.Errorf(logPrefix + "not implemented")
}

func (fileServer *FileServer) publish(publishingArtist *art.Artist, resources *art.ArtResources, signer Signer) error {
	const logPrefix = "FileServer publish "

	publication, err := signer.Sign(resources)
	if err != nil {
		log.Printf(logPrefix+"failed to sign resources %v, error: %v", resources, err)
		return err
	}

	artPath := fileServer.artPath(publishingArtist)
	_, err = os.Stat(artPath)
	if os.IsNotExist(err) {
		containerDirectory := filepath.Join(fileServer.rootPath, publishingArtist.ArtistId)
		_ = os.MkdirAll(containerDirectory, 0755)
		err = ioutil.WriteFile(artPath, publication.SerializedArtResources, 0644)
		if err != nil {
			log.Printf(logPrefix+"failed to write resources to %s, error: %v", artPath, err)
			return err
		}
	}
	publishedBytes, err := ioutil.ReadFile(artPath)
	if err != nil {
		log.Printf(logPrefix+"failed to read resources from %s, error: %v", artPath, err)
		return err
	}
	if bytes.Compare(publishedBytes, publication.SerializedArtResources) != 0 {
		log.Fatalf(logPrefix + "mismatched bytes")
		return fmt.Errorf("bytes on disk out of sync")
	}

	marshaledPublication, err := proto.Marshal(publication)
	if err != nil {
		log.Printf(logPrefix+"failed to marshal %v, error: %v", publication, err)
		return err
	}

	pubPath := fileServer.publicationPath(publishingArtist)
	err = ioutil.WriteFile(pubPath, marshaledPublication, 0644)
	if err != nil {
		log.Printf(logPrefix+"failed to write publication to %s, error: %v", pubPath, err)
		return err
	}
	return nil
}

func (fileServer *FileServer) readPublication(artistID string, pubkey string, publicationPath string) (*art.ArtistPublication, error) {
	const logPrefix = "FileServer readPublication "

	publishedData, err := ioutil.ReadFile(publicationPath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile %s error: %v", publicationPath, err)
		return nil, err
	}

	publication := art.ArtistPublication{}
	err = proto.Unmarshal(publishedData, &publication)
	// TODO: validate that the pubkey was used for the signature or reject this artist pub file

	// TODO: validate that the publication includes the publishing artist.
	// for _, artist := range art.Artists {
	// 	fileServer.artists[artistID] = &artist
	// }
	return &publication, nil
}

func (fileServer *FileServer) readPeerFile(artDirPath string, pubkey string) error {
	const logPrefix = "FileServer readPeerFile "

	//pubkeyAtHostColonPortRegex, err := regexp.Compile("^([0-9a-f]{66})@([0-9a-z\\-]+):[0-9]+$")
	peerFilePath := path.Join(artDirPath, pubkey)
	peerData, err := ioutil.ReadFile(peerFilePath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile %s error: %v", peerFilePath, err)
		return err
	}
	peer := art.Peer{}
	fileServer.peers[pubkey] = &peer
	//peer := &fileServer.peers[pubkey]
	err = proto.Unmarshal(peerData, fileServer.peers[pubkey])
	return err
}

func (fileServer *FileServer) StoreAlbum(album *art.Album) error {
	return fmt.Errorf("not implemented")
}

func (fileServer *FileServer) Albums(artistId string) (map[string]*art.Album, error) {
	albums, found := fileServer.albums[artistId]
	if !found {
		// Read album info from file system.
		fmt.Errorf("not implemented")
	}

	return albums, nil
}

// storeArtist saves a file with the given artist's details, albums, tracks, and peers.
func (fileServer *FileServer) StoreArtist(artist *art.Artist, signer Signer) (*art.ArtResources, error) {
	const logPrefix = "fileServer storeToFileSystem "

	publishedArtist := fileServer.artists[artist.ArtistId]
	if publishedArtist != nil &&
		publishedArtist.Pubkey != artist.Pubkey &&
		publishedArtist.Pubkey != "" {
		log.Printf(logPrefix+"update pubkey for %s from %s to %s",
			artist.ArtistId, publishedArtist.Pubkey, artist.Pubkey)
		// TODO: validate that it's safe to replace
	}
	fileServer.artists[artist.ArtistId] = artist

	artists := []*art.Artist{artist}
	albums := make([]*art.Album, 0)
	tracks := make([]*art.Track, len(fileServer.tracks))
	for _, album := range fileServer.albums[artist.ArtistId] {
		albums = append(albums, album)
	}
	for _, track := range fileServer.tracks[artist.ArtistId] {
		tracks = append(tracks, track)
	}

	peers := make([]*art.Peer, 0, len(fileServer.peers))
	for _, peer := range fileServer.peers {
		peers = append(peers, peer)
	}

	resources := art.ArtResources{
		Artists: artists,
		Albums:  albums,
		Tracks:  tracks,
		Peers:   peers,
	}

	err := fileServer.publish(artist, &resources, signer)

	return &resources, err
}

func (fileServer *FileServer) Artists() (map[string]*art.Artist, error) {
	return fileServer.artists, nil
}

func (fileServer *FileServer) Artist(artistID string) (*art.Artist, error) {
	artist := fileServer.artists[artistID]
	if artist == nil {
		return nil, ErrArtNotFound
	}
	return artist, nil
}

func (fileServer *FileServer) Tracks(artistID string) (map[string]*art.Track, error) {
	return fileServer.tracks[artistID], nil
}

func (fileServer *FileServer) AlbumTracks(artistID string, albumID string) (map[uint32]*art.Track, error) {
	albumTracksForArtist := fileServer.albumTracks[artistID]
	if albumTracksForArtist == nil {
		return nil, ErrArtNotFound
	}
	tracksForArtistAlbum := albumTracksForArtist[albumID]
	if tracksForArtistAlbum == nil {
		return nil, ErrArtNotFound
	}
	return tracksForArtistAlbum, nil
}

// StorePeer stores the peer in the in-memory database and defers saving to the file system.
func (fileServer *FileServer) StorePeer(artist *art.Artist, peer *art.Peer, signer Signer) error {
	// find the artist with the same pubkey
	log.Printf("FileServer StorePeer artist pubkey %s, peer pubkey %s", artist.Pubkey, peer.Pubkey)
	if artist.Pubkey == peer.Pubkey {
		fileServer.peers[peer.Pubkey] = peer
		_, err := fileServer.StoreArtist(artist, signer)
		if err != nil {
			log.Printf("fileServer StorePeer failed to store artist %s, error: %v",
				artist.ArtistId, err)
			return err
		}
		return nil
	}
	log.Printf("FileServer StorePeer should this create an artist record?")
	return ErrArtNotFound
}

func (fileServer *FileServer) Peer(pubkey string) (*art.Peer, error) {
	peer := fileServer.peers[pubkey]
	if peer == nil {
		return nil, ErrPeerNotFound
	}
	return peer, nil
}

func (fileServer *FileServer) Peers() (map[string]*art.Peer, error) {
	return fileServer.peers, nil
}

// StoreTrack stores track in the in-memory database
// and eventually "publishes" (flushes) all resources to the file system.
func (fileServer *FileServer) StoreTrack(track *art.Track, signer Signer) error {
	const logPrefix = "FileServer StoreTrack "

	tracksForArtist := fileServer.tracks[track.ArtistId]
	if tracksForArtist == nil {
		tracksForArtist = make(map[string]*art.Track)
		fileServer.tracks[track.ArtistId] = tracksForArtist
	}
	tracksForArtist[track.ArtistTrackId] = track
	if track.ArtistAlbumId != "" || track.AlbumTrackNumber > 0 {
		albumTracksForArtist := fileServer.albumTracks[track.ArtistId]
		if albumTracksForArtist == nil {
			albumTracksForArtist = make(map[string]map[uint32]*art.Track)
			fileServer.albumTracks[track.ArtistId] = albumTracksForArtist
		}
		tracksInArtistAlbum := albumTracksForArtist[track.ArtistAlbumId]
		if tracksInArtistAlbum == nil {
			tracksInArtistAlbum = make(map[uint32]*art.Track)
			albumTracksForArtist[track.ArtistAlbumId] = tracksInArtistAlbum
		}
		tracksInArtistAlbum[track.AlbumTrackNumber] = track
	}

	// If we know this track's artist's pubkey, asynchronously record the artist's publication.
	artist := fileServer.artists[track.ArtistId]
	if artist != nil && artist.Pubkey != "" {
		_, err := fileServer.StoreArtist(artist, signer)
		if err != nil {
			log.Printf(logPrefix+"filed to store artist %s, error: %v", artist.ArtistId, err)
			return err
		}
	}
	return nil
}

func (fileServer *FileServer) StoreTrackPayload(artistID string, artistTrackID string, payload []byte) error {
	const logPrefix = "FileServer StoreTrackPayload "

	filename := fileServer.mp3Filename(artistID, artistTrackID)
	containerDirectory := filepath.Dir(filename)

	err := os.MkdirAll(containerDirectory, 0755)
	if err != nil {
		log.Printf(logPrefix+"Failed to make directory %s, error: %v", containerDirectory, err)
		return err
	}

	err = ioutil.WriteFile(filename, payload, 0644)
	return err
}

func (fileServer *FileServer) Track(artistID string, trackID string) (*art.Track, error) {
	return fileServer.tracks[artistID][trackID], nil
}

func (fileServer *FileServer) TrackFilePath(artistID string, artistTrackID string) string {
	return fileServer.mp3Filename(artistID, artistTrackID)
}

func (fileServer *FileServer) artPath(artist *art.Artist) string {
	return filepath.Join(fileServer.rootPath, artist.ArtistId, ".art")
}

func (fileServer *FileServer) publicationPath(artist *art.Artist) string {
	// TODO: validate params.
	return filepath.Join(fileServer.rootPath, artist.ArtistId, artist.Pubkey+".pub")
}

func (fileServer *FileServer) mp3Filename(artistID string, artistTrackID string) (filename string) {
	// TODO: sanitize filepath so peer cannot write outside the base path dir sandbox.
	return filepath.Join(fileServer.rootPath, artistID, artistTrackID+".mp3")
}
