// +build regtest

package audiostrike

import (
	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"testing"
)

// TODO: split this into integration tests using regtest lightning node and unit tests using mock lightning
// These unit tests verify conformance to expectated Lightning behavior simulated by a mock Lightning nnode.
// These integration tests verify expectations about Lightning behavior on regtest to test integration.
func TestValidatePublication(t *testing.T) {
	lightningNode, err := NewLightningNode(cfg, &mockArtServer)
	if err != nil {
		t.Errorf("failed to instantiate lightning node, error: %v", err)
	}
	resources := art.ArtResources{
		Artists: []*art.Artist{&mockArtist},
	}
	testMarshaledResources, err := proto.Marshal(&resources)
	testSignature := "dh7xh9aw4ce6zhwpczg5qce6xfxkfcyj8cf91j719bgmcks3i7kyhrwiywrhzk5tk7a6d8x3xauppjz6thzzdwbyq8ffzj3p614ko3op"
	publication := art.ArtistPublication{
		Artist:                 &mockArtist,
		Signature:              testSignature,
		SerializedArtResources: testMarshaledResources,
	}
	validatedResources, err := lightningNode.ValidatePublication(&publication)
	if err != nil {
		t.Errorf("failed to validate publication, error: %v", err)
	} else if len(validatedResources.Artists) != 1 {
		t.Errorf("expected 1 artist in validated resources but found %d", len(validatedResources.Artists))
	}
}

// TestSign ensures that lightning can sign art resources into artist publications.
func TestSign(t *testing.T) {
	lightningNode, err := NewLightningNode(cfg, &mockArtServer)
	if err != nil {
		t.Errorf("failed to instantiate lightning node, error: %v", err)
	}
	resources := art.ArtResources{
		Artists: []*art.Artist{&mockArtist},
	}
	_, err = lightningNode.Sign(&resources)
	if err != nil {
		t.Errorf("lightning node is not operational. Sign error: %v", err)
	}

	publishingArtist, err := mockArtServer.Artist(cfg.ArtistID)
	if err != nil {
		t.Errorf("failed to get artist %s, error: %v", cfg.ArtistID, err)
	}
	if publishingArtist.Pubkey != mockPubkey {
		t.Errorf("Unexpected pubkey %s", publishingArtist.Pubkey)
	}
	server, err := NewAustkServer(cfg, &mockArtServer, lightningNode)
	if err != nil {
		t.Errorf("Failed to retrieve configured artist %s, error: %v",
			cfg.ArtistID, err)
	}
	if server == nil {
		t.Error("no austk server returned")
	}
}
