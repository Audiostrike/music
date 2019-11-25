package audiostrike

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"log"

	art "github.com/audiostrike/music/pkg/art"
)

const (
	mockPubkey Pubkey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef50"
)

type MockLightningClient struct {
}

func (c MockLightningClient) WalletBalance(ctx context.Context, in *lnrpc.WalletBalanceRequest, opts ...grpc.CallOption) (*lnrpc.WalletBalanceResponse, error) {
	return nil, fmt.Errorf("WalletBalance not implemented")
}
func (c MockLightningClient) ChannelBalance(ctx context.Context, in *lnrpc.ChannelBalanceRequest, opts ...grpc.CallOption) (*lnrpc.ChannelBalanceResponse, error) {
	return nil, fmt.Errorf("ChannelBalance not implemented")
}
func (c MockLightningClient) GetTransactions(ctx context.Context, in *lnrpc.GetTransactionsRequest, opts ...grpc.CallOption) (*lnrpc.TransactionDetails, error) {
	return nil, fmt.Errorf("GetTransactions not implemented")
}
func (c MockLightningClient) EstimateFee(ctx context.Context, in *lnrpc.EstimateFeeRequest, opts ...grpc.CallOption) (*lnrpc.EstimateFeeResponse, error) {
	return nil, fmt.Errorf("EstimateFee not implemented")
}
func (c MockLightningClient) SendCoins(ctx context.Context, in *lnrpc.SendCoinsRequest, opts ...grpc.CallOption) (*lnrpc.SendCoinsResponse, error) {
	return nil, fmt.Errorf("SendCoins not implemented")
}
func (c MockLightningClient) ListUnspent(ctx context.Context, in *lnrpc.ListUnspentRequest, opts ...grpc.CallOption) (*lnrpc.ListUnspentResponse, error) {
	return nil, fmt.Errorf("ListUnspent not implemented")
}
func (c MockLightningClient) SubscribeTransactions(ctx context.Context, in *lnrpc.GetTransactionsRequest, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeTransactionsClient, error) {
	return nil, fmt.Errorf("SubscribeTransactions not implemented")
}
func (c MockLightningClient) SendMany(ctx context.Context, in *lnrpc.SendManyRequest, opts ...grpc.CallOption) (*lnrpc.SendManyResponse, error) {
	return nil, fmt.Errorf("SendMany not implemented")
}
func (c MockLightningClient) NewAddress(ctx context.Context, in *lnrpc.NewAddressRequest, opts ...grpc.CallOption) (*lnrpc.NewAddressResponse, error) {
	return nil, fmt.Errorf("NewAddress not implemented")
}
func (c MockLightningClient) SignMessage(ctx context.Context, in *lnrpc.SignMessageRequest, opts ...grpc.CallOption) (*lnrpc.SignMessageResponse, error) {
	sum := sha256.Sum256(in.Msg)
	if sum == sha256.Sum256([]byte("Test message to ensure lnd is operational")) {
		return &lnrpc.SignMessageResponse{Signature: "test signature"}, nil
	} else if sum == [32]byte{76, 26, 34, 184, 199, 139, 58, 86, 156, 230, 49, 148, 181, 191, 98, 208, 243, 101, 199, 217, 61, 32, 138, 199, 66, 142, 152, 52, 113, 203, 208, 171} {
		return &lnrpc.SignMessageResponse{Signature: "test sig 2"}, nil
	} else if sum == [32]byte{15, 139, 163, 22, 5, 209, 7, 12, 64, 101, 25, 174, 93, 121, 62, 213, 222, 198, 26, 52, 168, 170, 125, 223, 215, 156, 74, 24, 188, 9, 88, 134} {
		return &lnrpc.SignMessageResponse{Signature: "test sig 3"}, nil
	}
	return nil, fmt.Errorf("Mock lnd client does not implement SignMessage for input msg: %s (sum: %v)", string(in.Msg), sum)
}
func (c MockLightningClient) VerifyMessage(ctx context.Context, in *lnrpc.VerifyMessageRequest, opts ...grpc.CallOption) (*lnrpc.VerifyMessageResponse, error) {
	if in.Signature == "dh7xh9aw4ce6zhwpczg5qce6xfxkfcyj8cf91j719bgmcks3i7kyhrwiywrhzk5tk7a6d8x3xauppjz6thzzdwbyq8ffzj3p614ko3op" {
		return &lnrpc.VerifyMessageResponse{Valid: true, Pubkey: "036f709187264df770bd453270a95b579595a42cd89eab2ea437dfd537048a7250"}, nil
	}
	return nil, fmt.Errorf("VerifyMessage not implemented")
}
func (c MockLightningClient) ConnectPeer(ctx context.Context, in *lnrpc.ConnectPeerRequest, opts ...grpc.CallOption) (*lnrpc.ConnectPeerResponse, error) {
	return nil, fmt.Errorf("ConnectPeer not implemented")
}
func (c MockLightningClient) DisconnectPeer(ctx context.Context, in *lnrpc.DisconnectPeerRequest, opts ...grpc.CallOption) (*lnrpc.DisconnectPeerResponse, error) {
	return nil, fmt.Errorf("DisconnectPeer not implemented")
}
func (c MockLightningClient) ListPeers(ctx context.Context, in *lnrpc.ListPeersRequest, opts ...grpc.CallOption) (*lnrpc.ListPeersResponse, error) {
	return nil, fmt.Errorf("ListPeers not implemented")
}
func (c MockLightningClient) GetInfo(ctx context.Context, in *lnrpc.GetInfoRequest, opts ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	return &lnrpc.GetInfoResponse{IdentityPubkey: string(mockPubkey)}, nil
}
func (c MockLightningClient) PendingChannels(ctx context.Context, in *lnrpc.PendingChannelsRequest, opts ...grpc.CallOption) (*lnrpc.PendingChannelsResponse, error) {
	return nil, fmt.Errorf("PendingChanels not implemented")
}
func (c MockLightningClient) ListChannels(ctx context.Context, in *lnrpc.ListChannelsRequest, opts ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	return nil, fmt.Errorf("ListChannels not implemented")
}
func (c MockLightningClient) SubscribeChannelEvents(ctx context.Context, in *lnrpc.ChannelEventSubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelEventsClient, error) {
	return nil, fmt.Errorf("SubscribeChannelEvents not implemented")
}
func (c MockLightningClient) ClosedChannels(ctx context.Context, in *lnrpc.ClosedChannelsRequest, opts ...grpc.CallOption) (*lnrpc.ClosedChannelsResponse, error) {
	return nil, fmt.Errorf("ClosedChannels not implemented")
}
func (c MockLightningClient) OpenChannelSync(ctx context.Context, in *lnrpc.OpenChannelRequest, opts ...grpc.CallOption) (*lnrpc.ChannelPoint, error) {
	return nil, fmt.Errorf("OpenChannelSync not implemented")
}
func (c MockLightningClient) OpenChannel(ctx context.Context, in *lnrpc.OpenChannelRequest, opts ...grpc.CallOption) (lnrpc.Lightning_OpenChannelClient, error) {
	return nil, fmt.Errorf("OpenChannel not implemented")
}
func (c MockLightningClient) CloseChannel(ctx context.Context, in *lnrpc.CloseChannelRequest, opts ...grpc.CallOption) (lnrpc.Lightning_CloseChannelClient, error) {
	return nil, fmt.Errorf("CloseChannel not implemented")
}
func (c MockLightningClient) AbandonChannel(ctx context.Context, in *lnrpc.AbandonChannelRequest, opts ...grpc.CallOption) (*lnrpc.AbandonChannelResponse, error) {
	return nil, fmt.Errorf("AbandonChannel not implemented")
}
func (c MockLightningClient) SendPayment(ctx context.Context, opts ...grpc.CallOption) (lnrpc.Lightning_SendPaymentClient, error) {
	return nil, fmt.Errorf("SendPayment not implemented")
}
func (c MockLightningClient) SendPaymentSync(ctx context.Context, in *lnrpc.SendRequest, opts ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	return nil, fmt.Errorf("SendPaymentSync not implemented")
}
func (c MockLightningClient) SendToRoute(ctx context.Context, opts ...grpc.CallOption) (lnrpc.Lightning_SendToRouteClient, error) {
	return nil, fmt.Errorf("SendToRoute not implemented")
}
func (c MockLightningClient) SendToRouteSync(ctx context.Context, in *lnrpc.SendToRouteRequest, opts ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	return nil, fmt.Errorf("SendToRouteSync not implemented")
}
func (c MockLightningClient) AddInvoice(ctx context.Context, in *lnrpc.Invoice, opts ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	return nil, fmt.Errorf("AddInvoice not implemented")
}
func (c MockLightningClient) ListInvoices(ctx context.Context, in *lnrpc.ListInvoiceRequest, opts ...grpc.CallOption) (*lnrpc.ListInvoiceResponse, error) {
	return nil, fmt.Errorf("ListInvoices not implemented")
}
func (c MockLightningClient) LookupInvoice(ctx context.Context, in *lnrpc.PaymentHash, opts ...grpc.CallOption) (*lnrpc.Invoice, error) {
	return nil, fmt.Errorf("LookupInvoice not implemented")
}
func (c MockLightningClient) SubscribeInvoices(ctx context.Context, in *lnrpc.InvoiceSubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeInvoicesClient, error) {
	return nil, fmt.Errorf("SubscribeInvoices not implemented")
}
func (c MockLightningClient) DecodePayReq(ctx context.Context, in *lnrpc.PayReqString, opts ...grpc.CallOption) (*lnrpc.PayReq, error) {
	return nil, fmt.Errorf("DecodePayReq not implemented")
}
func (c MockLightningClient) ListPayments(ctx context.Context, in *lnrpc.ListPaymentsRequest, opts ...grpc.CallOption) (*lnrpc.ListPaymentsResponse, error) {
	return nil, fmt.Errorf("ListPayments not implemented")
}
func (c MockLightningClient) DeleteAllPayments(ctx context.Context, in *lnrpc.DeleteAllPaymentsRequest, opts ...grpc.CallOption) (*lnrpc.DeleteAllPaymentsResponse, error) {
	return nil, fmt.Errorf("DeleteAllPayments not implemented")
}
func (c MockLightningClient) DescribeGraph(ctx context.Context, in *lnrpc.ChannelGraphRequest, opts ...grpc.CallOption) (*lnrpc.ChannelGraph, error) {
	return nil, fmt.Errorf("DescribeGraph not implemented")
}
func (c MockLightningClient) GetChanInfo(ctx context.Context, in *lnrpc.ChanInfoRequest, opts ...grpc.CallOption) (*lnrpc.ChannelEdge, error) {
	return nil, fmt.Errorf("GetChanInfo not implemented")
}
func (c MockLightningClient) GetNodeInfo(ctx context.Context, in *lnrpc.NodeInfoRequest, opts ...grpc.CallOption) (*lnrpc.NodeInfo, error) {
	return nil, fmt.Errorf("GetNodeInfo not implemented")
}
func (c MockLightningClient) QueryRoutes(ctx context.Context, in *lnrpc.QueryRoutesRequest, opts ...grpc.CallOption) (*lnrpc.QueryRoutesResponse, error) {
	return nil, fmt.Errorf("QueryRoutes not implemented")
}
func (c MockLightningClient) GetNetworkInfo(ctx context.Context, in *lnrpc.NetworkInfoRequest, opts ...grpc.CallOption) (*lnrpc.NetworkInfo, error) {
	return nil, fmt.Errorf("GetNetworkInfo not implemented")
}
func (c MockLightningClient) StopDaemon(ctx context.Context, in *lnrpc.StopRequest, opts ...grpc.CallOption) (*lnrpc.StopResponse, error) {
	return nil, fmt.Errorf("StopDaemon not implemented")
}
func (c MockLightningClient) SubscribeChannelGraph(ctx context.Context, in *lnrpc.GraphTopologySubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelGraphClient, error) {
	return nil, fmt.Errorf("SubscribeChannelGraph not implemented")
}
func (c MockLightningClient) DebugLevel(ctx context.Context, in *lnrpc.DebugLevelRequest, opts ...grpc.CallOption) (*lnrpc.DebugLevelResponse, error) {
	return nil, fmt.Errorf("DebugLevel not implemented")
}
func (c MockLightningClient) FeeReport(ctx context.Context, in *lnrpc.FeeReportRequest, opts ...grpc.CallOption) (*lnrpc.FeeReportResponse, error) {
	return nil, fmt.Errorf("FeeReport not implemented")
}
func (c MockLightningClient) UpdateChannelPolicy(ctx context.Context, in *lnrpc.PolicyUpdateRequest, opts ...grpc.CallOption) (*lnrpc.PolicyUpdateResponse, error) {
	return nil, fmt.Errorf("UpdateChannelPolicy not implemented")
}
func (c MockLightningClient) ForwardingHistory(ctx context.Context, in *lnrpc.ForwardingHistoryRequest, opts ...grpc.CallOption) (*lnrpc.ForwardingHistoryResponse, error) {
	return nil, fmt.Errorf("ForwardingHistory not implemented")
}
func (c MockLightningClient) ExportChannelBackup(ctx context.Context, in *lnrpc.ExportChannelBackupRequest, opts ...grpc.CallOption) (*lnrpc.ChannelBackup, error) {
	return nil, fmt.Errorf("ExportChannelBackup not implemented")
}
func (c MockLightningClient) ExportAllChannelBackups(ctx context.Context, in *lnrpc.ChanBackupExportRequest, opts ...grpc.CallOption) (*lnrpc.ChanBackupSnapshot, error) {
	return nil, fmt.Errorf("ExportAllChannelBackups not implemented")
}
func (c MockLightningClient) VerifyChanBackup(ctx context.Context, in *lnrpc.ChanBackupSnapshot, opts ...grpc.CallOption) (*lnrpc.VerifyChanBackupResponse, error) {
	return nil, fmt.Errorf("VerifyChanBackup not implemented")
}
func (c MockLightningClient) RestoreChannelBackups(ctx context.Context, in *lnrpc.RestoreChanBackupRequest, opts ...grpc.CallOption) (*lnrpc.RestoreBackupResponse, error) {
	return nil, fmt.Errorf("RestoreChannelBackups not implemented")
}
func (c MockLightningClient) SubscribeChannelBackups(ctx context.Context, in *lnrpc.ChannelBackupSubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelBackupsClient, error) {
	return nil, fmt.Errorf("SubscribeChannelBackups not implemented")
}

func (c MockLightningClient) ChannelAcceptor(ctx context.Context, opts ...grpc.CallOption) (lnrpc.Lightning_ChannelAcceptorClient, error) {
	return nil, fmt.Errorf("ChannelAcceptor not implemented")
}


func NewMockLightningPublisher(cfg *Config, localStorage ArtServer) (*LightningPublisher, error) {
	const logPrefix = "NewMockLightningPublisher "

	lndClient := MockLightningClient{}

	publishingArtist, err := localStorage.Artist(cfg.ArtistID)
	if err == ErrArtNotFound {
		// The configured artist is not yet stored, so store the artist.
		publishingArtist = &art.Artist{ArtistId: cfg.ArtistID, Name: cfg.ArtistName}
		err = localStorage.StoreArtist(publishingArtist)
		if err != nil {
			log.Fatalf(logPrefix+"failed to store artist %v, error: %v",
				publishingArtist, err)
			return nil, err
		}
		log.Printf(logPrefix+"stored %v", publishingArtist)
	} else if err != nil {
		log.Fatalf(logPrefix+"failed to get artist %s from storage, error: %v", cfg.ArtistID, err)
		return nil, ErrArtNotFound
	}

	return &LightningPublisher{
		lightningClient:  lndClient,
		publishingArtist: publishingArtist,
	}, nil
}
