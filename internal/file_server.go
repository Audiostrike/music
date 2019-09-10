package audiostrike

import (
	"bytes"
	"fmt"
	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"os"
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
	artistArtFileRegex  *regexp.Regexp = regexp.MustCompile("^(?P<root>.*)/(?P<ArtistID>[0-9a-z]+)/[.]art$")
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

	err := filepath.Walk(artDirPath, fileServer.readFile)
	if err != nil {
		log.Fatalf(logPrefix+"Failed to read art directory, error: %v", err)
		return nil, err
	}
	return &fileServer, nil
}

func (fileServer *FileServer) readFile(path string, fileInfo os.FileInfo, err error) error {
	const logPrefix = "FileServer readFile "

	if path == fileServer.rootPath {
		return filepath.SkipDir
	}

	// All the files to read are owned by the Artist whose artistID is the file's directory name.
	// Identify that artistID by the name of the directory.
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

	// if this is the [pubkey].pub file
	pubFileMatchGroups := artistPubFileRegex.FindStringSubmatch(path)
	if len(pubFileMatchGroups) == 4 {
		// This is the artist's [pubkey].pub file. Read the publication into art resources.
		pubFileArtistID := pubFileMatchGroups[2]
		if pubFileArtistID != artistID { // sanity check
			return fmt.Errorf("unexpected ArtistID %s in %s for artist %s",
				pubFileArtistID, path, artistID)
		}
		pubkey := pubFileMatchGroups[3]
		log.Printf(logPrefix+"pub file %s for artistId: %s, pubkey: %s}", path, artistID, pubkey)

		publication, err := fileServer.readPublication(artistID, pubkey, path)
		if err != nil {
			log.Fatalf("failed to read artist %s publication %s, error: %v", artistID, path, err)
			return err
		}
		resources, err := read(publication)
		if err != nil {
			log.Printf(logPrefix+"failed to read resources from publication, error: %v", err)
			return err
		}
		err = fileServer.indexResources(resources)
		if err != nil {
			log.Printf(logPrefix+"failed to index resources from publication, error: %v", err)
			return err
		}
		
		log.Printf(logPrefix+"read and index resources from publication at %s", path)
		return nil
	}

	// if this is the .art file
	artFileMatchGroups := artistArtFileRegex.FindStringSubmatch(path)
	if len(artFileMatchGroups) == 4 {
		// This is the artist's .art file. Read its art records.
		artFileArtistID := artFileMatchGroups[2]
		if artFileArtistID != artistID { // sanity check
			return fmt.Errorf("unexpected ArtistID %s in %s for artist %s",
				artFileArtistID, path, artistID)
		}
		pubkey := artFileMatchGroups[3]
		log.Printf(logPrefix+"pub file %s for artistId: %s, pubkey: %s}", path, artistID, pubkey)

		err := fileServer.readArtFile(artistID, path)
		if err != nil {
			log.Fatalf("failed to read artist %s resources %s, error: %v", artistID, path, err)
			return err
		}

		return nil
	}

	// Read it as art.ArtResources
	// ArtistTrackID may be simple alphanumeric identifier
	// or it may optionally include album or other slash-separated hierarchy.
	// Optionally order tracks and albums with numeric prefixes.

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

func (fileServer *FileServer) savePublishedResources(publication *art.ArtistPublication, resources *art.ArtResources) error {
	const logPrefix = "FileServer savePublishedResources "

	// Ensure that this artist has a directory.
	artistDirectory := filepath.Join(fileServer.rootPath, publication.Artist.ArtistId)
	_ = os.MkdirAll(artistDirectory, 0755)

	// Write the publication's serialized resources to the .art file
	artPath := fileServer.artPath(publication.Artist)
	_, err := os.Stat(artPath)
	if os.IsNotExist(err) {
		log.Printf(logPrefix+"publishing %v to %s", resources, artPath)
	} else {
		log.Printf(logPrefix+"republishing %v to %s", resources, artPath)
	}

	err = ioutil.WriteFile(artPath, publication.SerializedArtResources, 0644)
	if err != nil {
		log.Printf(logPrefix+"failed to write resources to %s, error: %v", artPath, err)
		return err
	}

	publishedBytes, err := ioutil.ReadFile(artPath)
	if err != nil {
		log.Printf(logPrefix+"failed to read resources from %s, error: %v", artPath, err)
		return err
	}
	// TODO: Move this verification into a unit test.
	if bytes.Compare(publishedBytes, publication.SerializedArtResources) != 0 {
		log.Fatalf(logPrefix+"mismatched bytes, published at %s: %v, serialized: %v",
			artPath, publishedBytes, publication.SerializedArtResources)
		return fmt.Errorf("bytes on disk out of sync")
	}

	// Write the signed publication to the artist's [pubkey].pub file.
	marshaledPublication, err := proto.Marshal(publication)
	if err != nil {
		log.Printf(logPrefix+"failed to marshal %v, error: %v", publication, err)
		return err
	}

	pubPath := fileServer.publicationPath(publication.Artist)
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
	if err != nil {
		log.Printf(logPrefix+"failed to unmarshal publication, error: %v", err)
		return nil, err
	}

	return &publication, err
}

func (fileServer *FileServer) readArtFile(artistID string, artFilePath string) error {
	const logPrefix = "FileServer readArtFile "

	artData, err := ioutil.ReadFile(artFilePath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile %s, error: %v", artFilePath, err)
		return err
	}

	resources := art.ArtResources{}
	err = proto.Unmarshal(artData, &resources)
	if err != nil {
		return err
	}

	return fileServer.indexResources(&resources)
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

func (fileServer *FileServer) Albums(artistId string) (map[string]*art.Album, error) {
	albums, found := fileServer.albums[artistId]
	if !found {
		// Read album info from file system.
		fmt.Errorf("not implemented")
	}

	return albums, nil
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

// StorePublication saves a file with the published artist details, albums, tracks, and peers.
func (fileServer *FileServer) StorePublication(publication *art.ArtistPublication) error {
	const logPrefix = "fileServer storeToFileSystem "

	artistId := publication.Artist.ArtistId
	previouslyPublishedArtist := fileServer.artists[artistId]
	if previouslyPublishedArtist != nil &&
		previouslyPublishedArtist.Pubkey != publication.Artist.Pubkey &&
		previouslyPublishedArtist.Pubkey != "" {
		log.Printf(logPrefix+"update pubkey for /%s from %s to %s",
			artistId, previouslyPublishedArtist.Pubkey, publication.Artist.Pubkey)
		// TODO: validate that it's safe to replace
	}
	fileServer.artists[artistId] = publication.Artist

	// Read the resources from the publication.
	publishedResources := art.ArtResources{}
	err := proto.Unmarshal(publication.SerializedArtResources, &publishedResources)
	if err != nil {
		log.Fatalf(logPrefix+"failed to deserialized publication %v, error: %v", publication, err)
		return err
	}

	err = fileServer.savePublishedResources(publication, &publishedResources)
	if err != nil {
		return err
	}

	err = fileServer.indexResources(&publishedResources)

	return err
}

// indexResources indexes the resources for fast retrieval
func (fileServer *FileServer) indexResources(resources *art.ArtResources) error {
	const logPrefix = "FileServer indexResources "

	for _, artist := range resources.Artists {
		previouslyStoredArtist := fileServer.artists[artist.ArtistId]
		if previouslyStoredArtist != nil {
			log.Printf(logPrefix+"replacing artist %s details %v with %v",
				artist.ArtistId, previouslyStoredArtist, artist)
		}
		fileServer.artists[artist.ArtistId] = artist
	}
	for _, album := range resources.Albums {
		artistAlbums := fileServer.albums[album.ArtistId]
		if artistAlbums == nil {
			artistAlbums := make(map[string]*art.Album)
			fileServer.albums[album.ArtistId] = artistAlbums
		}
		artistAlbums[album.ArtistAlbumId] = album
	}
	for _, track := range resources.Tracks {
		artistTracks := fileServer.tracks[track.ArtistId]
		if artistTracks == nil {
			artistTracks := make(map[string]*art.Track)
			fileServer.tracks[track.ArtistId] = artistTracks
		}
		artistTracks[track.ArtistTrackId] = track
	}

	for _, peer := range resources.Peers {
		previouslyStoredPeer := fileServer.peers[peer.Pubkey]
		if previouslyStoredPeer != nil {
			log.Printf(logPrefix+"replacing peer %s details %v with %v from resources %v",
				peer.Pubkey, previouslyStoredPeer, peer, resources)
		}
		fileServer.peers[peer.Pubkey] = peer
	}

	return nil
}

func (fileServer *FileServer) StoreArtist(artist *art.Artist) error {
	const logPrefix = "FileServer StoreArtist "

	fileServer.artists[artist.ArtistId] = artist

	return nil
}

func (fileServer *FileServer) StoreAlbum(album *art.Album, publisher Publisher) error {
	const logPrefix = "FileServer StoreAlbum "

	publishingArtist, err := publisher.Artist()
	if err != nil {
		log.Fatalf(logPrefix+"failed to get Artist for publisher %v, error: %v", publisher, err)
		return err
	}

	albumArtist, err := fileServer.Artist(album.ArtistId)
	if err != nil {
		log.Fatalf(logPrefix+"failed to get artist %s for album %v, error: %v",
			album.ArtistId, album, err)
		return err
	}

	if publishingArtist.Pubkey != albumArtist.Pubkey {
		log.Printf(logPrefix+"skip StoreAlbum %v because publishing pubkey %v does not match album artist pubkey %s, error: %v",
			album, publishingArtist.Pubkey, albumArtist.Pubkey, err)
		return err
	}

	if album.ArtistId == "" || album.ArtistAlbumId == "" {
		log.Fatalf(logPrefix+"malformed album %v", album)
	}

	artistAlbums := fileServer.albums[album.ArtistId]
	if artistAlbums == nil {
		artistAlbums = make(map[string]*art.Album)
		fileServer.albums[album.ArtistId] = artistAlbums
	}

	artistAlbums[album.ArtistAlbumId] = album
	log.Printf(logPrefix+"stored album %v for publishing artist %v", album, publishingArtist)

	return nil
}

// StorePeer stores the peer in the in-memory database
// and saves it to the publisher's artist directory file system.
func (fileServer *FileServer) StorePeer(peer *art.Peer, publisher Publisher) error {
	const logPrefix = "FileServer StorePeer "

	publishingArtist, err := publisher.Artist()
	if err != nil {
		log.Fatalf(logPrefix+"failed to get Artist for publisher %v, error: %v", publisher, err)
		return err
	}

	log.Printf("FileServer StorePeer %v for publishing artist %v", peer, publishingArtist)
	if publishingArtist.Pubkey == peer.Pubkey {
		fileServer.peers[peer.Pubkey] = peer
	} else {
		log.Printf(logPrefix+"skip StorePeer %v because pubkey does not match artist %v, error: %v",
			peer, publishingArtist, err)
		return err
	}
	return nil
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
func (fileServer *FileServer) StoreTrack(track *art.Track, publisher Publisher) error {
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

	return nil
}

func (fileServer *FileServer) StoreTrackPayload(track *art.Track, payload []byte) error {
	const logPrefix = "FileServer StoreTrackPayload "

	filename := fileServer.mp3Filename(track)
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

func (fileServer *FileServer) TrackFilePath(track *art.Track) string {
	return fileServer.mp3Filename(track)
}

func (fileServer *FileServer) artPath(artist *art.Artist) string {
	return filepath.Join(fileServer.rootPath, artist.ArtistId, ".art")
}

func (fileServer *FileServer) publicationPath(artist *art.Artist) string {
	// TODO: validate params.
	return filepath.Join(fileServer.rootPath, artist.ArtistId, artist.Pubkey+".pub")
}

func (fileServer *FileServer) mp3Filename(track *art.Track) (filename string) {
	// TODO: sanitize filepath so peer cannot write outside the base path dir sandbox.
	return filepath.Join(fileServer.rootPath, track.ArtistId, track.ArtistTrackId+".mp3")
}
