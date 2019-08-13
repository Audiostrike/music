package audiostrike

import (
	"os"
	"time"

	"github.com/faiface/beep"
	faifacemp3 "github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	mikkyangid3 "github.com/mikkyang/id3-go"
	"log"
	"path/filepath"
)

// Mp3 exposes the Tags (mp3 metadata) and bytes of a given .mp3 file.
type Mp3 struct {
	file             *os.File
	buffer           []byte
	length           int
	position         int
	Tags             map[string]string
	playbackFinished chan bool
}

// OpenMp3ToRead opens an mp3 file to read its data and tags (metadata)
func OpenMp3ToRead(fileName string) (mp3 *Mp3, err error) {
	// Read the file.
	var file *os.File
	file, err = os.Open(fileName)
	if err != nil {
		return
	}

	// Read the mp3 tags.
	var id3File *mikkyangid3.File
	id3File, err = mikkyangid3.OpenForRead(fileName)
	if err != nil {
		return
	}
	defer id3File.Close()
	var tags map[string]string
	tags, err = parseTags(id3File)
	if err != nil {
		return
	}

	// Return the Mp3 struct with the file and mp3 tags.
	mp3 = &Mp3{
		file: file,
		Tags: tags,
	}
	return
}

func BuildMp3Filename(rootDirPath string, artistID string, artistTrackID string) (filename string) {
	// TODO: make base path configurable, defaulting to ./tracks/
	// TODO: sanitize filepath so peer cannot write outside the base path dir sandbox.
	return filepath.Join(rootDirPath, artistID, artistTrackID+".mp3")
}

func parseTags(file *mikkyangid3.File) (map[string]string, error) {
	tags := map[string]string{
		"Artist": file.Artist(),
		"Album":  file.Album(),
		"Title":  file.Title(),
	}
	return tags, nil
}

func (mp3 *Mp3) ArtistName() string {
	return mp3.Tags["Artist"]
}

func (mp3 *Mp3) AlbumTitle() (string, bool) {
	albumTitle := mp3.Tags["Album"]
	return albumTitle, albumTitle != ""
}

func (mp3 *Mp3) Title() string {
	return mp3.Tags["Title"]
}

// ReadBytes returns the raw data from the .mp3 file.
func (mp3 *Mp3) ReadBytes() ([]byte, error) {
	// If buffer already has the bytes, return them.
	if mp3.buffer != nil {
		return mp3.buffer, nil
	}

	// Otherwise read the bytes from the file into buffer.
	fileInfo, err := mp3.file.Stat()
	if err == nil {
		mp3.buffer = make([]byte, fileInfo.Size())
		_, err = mp3.file.Read(mp3.buffer)
	}
	return mp3.buffer, err
}

func (mp3 *Mp3) PlayAndWait() error {
	trackStreamer, format, err := faifacemp3.Decode(mp3.file)
	if err != nil {
		log.Printf("Failed to decode mp3, error: %v", err)
		return err
	}
	defer trackStreamer.Close()
	mp3.playbackFinished = make(chan bool)
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/5))
	speaker.Play(beep.Seq(trackStreamer, beep.Callback(func() {
		mp3.playbackFinished <- true
	})))

	<-mp3.playbackFinished

	return nil
}
