package audiostrike

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"log"
)

const (
	mockPubkey   string = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef50"
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
	hasher := sha256.New()
	sum := hasher.Sum(in.Msg)
	log.Printf("SignMessage msg: %v, sum: %x", in.Msg, sum)
	//hash := sha256.Sum256(nil)
	if bytes.Equal(sum, hasher.Sum([]byte("Test message to ensure lnd is operational"))) {
		return &lnrpc.SignMessageResponse{Signature: "test signature"}, nil
	} else if bytes.Equal(sum, []byte{0x0a, 0x54, 0x0a, 0x0e, 0x61, 0x6c, 0x69, 0x63, 0x65, 0x74, 0x68, 0x65, 0x61, 0x72, 0x74, 0x69, 0x73, 0x74, 0x1a, 0x42, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x35, 0x30, 0x1a, 0x1b, 0x0a, 0x0e, 0x61, 0x6c, 0x69, 0x63, 0x65, 0x74, 0x68, 0x65, 0x61, 0x72, 0x74, 0x69, 0x73, 0x74, 0x1a, 0x09, 0x74, 0x65, 0x73, 0x74, 0x74, 0x72, 0x61, 0x63, 0x6b, 0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55}) {
		return &lnrpc.SignMessageResponse{Signature: "test sig 2"}, nil
	} else if bytes.Equal(sum, []byte{10, 84, 10, 14, 97, 108, 105, 99, 101, 116, 104, 101, 97, 114, 116, 105, 115, 116, 26, 66, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 53, 48, 26, 27, 10, 14, 97, 108, 105, 99, 101, 116, 104, 101, 97, 114, 116, 105, 115, 116, 26, 9, 116, 101, 115, 116, 116, 114, 97, 99, 107, 34, 68, 10, 66, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 97, 98, 99, 100, 101, 102, 53, 48, 227, 176, 196, 66, 152, 252, 28, 20, 154, 251, 244, 200, 153, 111, 185, 36, 39, 174, 65, 228, 100, 155, 147, 76, 164, 149, 153, 27, 120, 82, 184, 85}) {
		return &lnrpc.SignMessageResponse{Signature: "test sig 3"}, nil
	}
	return nil, fmt.Errorf("Mock lnd client does not implement SignMessage for input msg: %s (sum: %v)", string(in.Msg), sum)
}
func (c MockLightningClient) VerifyMessage(ctx context.Context, in *lnrpc.VerifyMessageRequest, opts ...grpc.CallOption) (*lnrpc.VerifyMessageResponse, error) {
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
	return &lnrpc.GetInfoResponse{IdentityPubkey: mockPubkey}, nil
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
