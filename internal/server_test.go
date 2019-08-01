package audiostrike

import (
	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

// func TestMain(m *testing.M) {
// 	// Initialize....
// 	// Run the test.
// 	os.Exit(m.Run())
// }

func TestGetAllArt(t *testing.T) {
	cfg := &Config{
		DbName:       "music",
		DbUser:       defaultDbUser,
		DbPassword:   "2jV@.UXg2$1f",
		CertFilePath: "/home/rod/.lnd/tls.cert",
		MacaroonPath: "/home/rod/.lnd/data/chain/bitcoin/mainnet/admin.macaroon",
		LndHost:      "127.0.0.1",
		LndGrpcPort:  10009,
	}
	db, err := OpenDb(cfg.DbName, cfg.DbUser, cfg.DbPassword)
	if err != nil {
		t.Errorf("Failed to open db, error: %v", err)
		return
	}
	artServer, err := NewServer(cfg, db)
	if err != nil {
		t.Errorf("Failed to connect to music DB, error %v", err)
	}
	testHttpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		artServer.getAllArtHandler(w, req)
	}))

	artReply := art.ArtReply{}
	response, err := http.Get(testHttpServer.URL)
	bytes, err := ioutil.ReadAll(response.Body)
	err = proto.Unmarshal(bytes, &artReply)

	// Verify that the server handled the request successfully.
	if response.StatusCode != 200 {
		t.Errorf("expected success but got %d", response.StatusCode)
	}

	// Verify that the one test artist and her music was served.
	if len(artReply.Artists) != 1 {
		t.Errorf("expected 1 artist but got %d in reply: %v", len(artReply.Artists), artReply)
	}

	// TODO: configure db for test with known artist pubkey etc.
	// replyArtist := artReply.Artists[0]
	// if replyArtist.Pubkey != cfg.Pubkey {
	// 	t.Errorf("expected artist with pubkey %s but got %s in reply: %v",
	// 		cfg.Pubkey, replyArtist.Pubkey, artReply)
	// }
	defer testHttpServer.Close()
}
