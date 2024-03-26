package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"testing"
)

func TestCommunication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, host2 := createHost(ctx, t), createHost(ctx, t)
	defer host1.Close()
	defer host2.Close()

	notiChan := &discoveryNotifee2{h: host1}
	discovery1 := mdns.NewMdnsService(host1, "example", notiChan)
	defer discovery1.Close()

	host1.SetStreamHandler(protocolID, func(s network.Stream) {
		//go writeCounter(s)
		go readCounter(s)
	})

	notiChan2 := &discoveryNotifee2{h: host2}
	discovery2 := mdns.NewMdnsService(host2, "example", notiChan2)
	defer discovery2.Close()

	host2.SetStreamHandler(protocolID, func(s network.Stream) {
		//go writeCounter(s)
		go readCounter(s)
	})

	//advertiseAndFindPeers(ctx, t, discovery1, discovery2)

	host2Info := host2.Peerstore().PeerInfo(host2.ID())
	fmt.Println("host2 info: ", host2Info)
	err := connectP2pByAddr(host1, fmt.Sprintf("%s/p2p/%s", host2.Addrs()[0], host2.ID()))
	if err != nil {
		t.Fatalf("Failed to connect host1 to host2: %v", err)
	}

	stream, err := host1.NewStream(ctx, host2Info.ID, protocolID)
	if err != nil {
		t.Fatalf("Failed to open stream from host1 to host2: %v", err)
	}
	defer stream.Close()

	msg := uint64(1)
	sendMessage(t, stream, msg)
	receivedMsg := receiveMessage(t, stream)

	if receivedMsg != 1 {
		t.Fatalf("Received unexpected message from host2: got %d, want %d", receivedMsg, msg)
	}
}

func createHost(ctx context.Context, t *testing.T) host.Host {
	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	return host
}

func sendMessage(t *testing.T, stream network.Stream, msg uint64) {
	err := binary.Write(stream, binary.BigEndian, msg)
	if err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}
}

func receiveMessage(t *testing.T, stream network.Stream) uint64 {
	var receivedMsg uint64
	err := binary.Read(stream, binary.BigEndian, &receivedMsg)
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}
	return receivedMsg
}
