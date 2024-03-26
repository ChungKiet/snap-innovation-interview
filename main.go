package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	protocolID  = "/example/1.0.0"
	serviceName = "meetme"
	encryptKey  = "abcdabcdabcdabcd" // sample encrypt key with length 16
	delimeter   = ","
	HOST        = "/ip4/127.0.0.1/tcp/0"
)

// connectP2pByAddr: connect p2p address when init new node
func connectP2pByAddr(host host.Host, peerAddrs string) error {
	for _, addr := range strings.Split(peerAddrs, delimeter) {
		peerMA, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return err
		}
		peerAddrInfo, err := peer.AddrInfoFromP2pAddr(peerMA)
		if err != nil {
			return err
		}

		// Connect to the node at the given address.
		if err := host.Connect(context.Background(), *peerAddrInfo); err != nil {
			return err
		}
		fmt.Println("Connected to", peerAddrInfo.String())

		// Open a stream with the given peer.
		s, err := host.NewStream(context.Background(), peerAddrInfo.ID, protocolID)
		if err != nil {
			return err
		}

		// Start the write and read threads.
		go writeCounter(s)
		go readCounter(s)
	}

	return nil
}

// connectP2p: connect to other node
func connectP2p(host host.Host, peerAddrInfo *peer.AddrInfo) error {
	// Connect to the node at the given address.
	if err := host.Connect(context.Background(), *peerAddrInfo); err != nil {
		return err
	}
	fmt.Println("Connected to", peerAddrInfo.String())

	// Open a stream with the given peer.
	s, err := host.NewStream(context.Background(), peerAddrInfo.ID, protocolID)
	if err != nil {
		return err
	}

	// Start the write and read threads.
	go writeCounter(s)
	go readCounter(s)

	return nil
}

func main() {
	// Add -peer-address flag, include many address concat by ","
	peerAddrs := flag.String("pd", "", "peer address")
	flag.Parse()
	fmt.Println("Peer addr: ", strings.Split(*peerAddrs, delimeter))

	// Create the libp2p host.
	host, err := libp2p.New(libp2p.ListenAddrStrings(HOST))
	if err != nil {
		panic(err)
	}
	defer host.Close()

	// Print this node's addresses and ID
	fmt.Println("Addresses:", host.Addrs())
	fmt.Println("ID:", host.ID())

	host.SetStreamHandler(protocolID, func(s network.Stream) {
		go writeCounter(s)
		go readCounter(s)
	})

	// Setup peer discovery.
	peerChan := &discoveryNotifee2{h: host}
	discoveryService := mdns.NewMdnsService(
		host,
		serviceName,
		peerChan,
	)
	if err != nil {
		panic(err)
	}
	defer discoveryService.Close()

	if peerAddrs != nil && len(*peerAddrs) > 0 {
		err = connectP2pByAddr(host, *peerAddrs)
		if err != nil {
			fmt.Println("Error when connect to addr: ", err.Error(), "peer_addr", *peerAddrs)
		}
	}

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh
}

// increase counter and send, it easy to check when connect to new peer
func writeCounter(s network.Stream) {
	var counter uint64
	for {
		<-time.After(time.Second)
		counter++

		// Encrypt counter value before sending
		encryptedCounter := encrypt(counter)
		if err := binary.Write(s, binary.BigEndian, encryptedCounter); err != nil {
			fmt.Println("Err when write msg: ", err.Error())
		}
	}
}

func readCounter(s network.Stream) {
	for {
		var encryptedCounter uint64

		// Read encrypted counter value
		if err := binary.Read(s, binary.BigEndian, &encryptedCounter); err != nil {
			fmt.Println("Err when read msg: ", err.Error())
		}

		// Decrypt the counter value
		counter := decrypt(encryptedCounter)
		fmt.Printf("Received %d from %s\n", counter, s.ID())
	}
}

func encrypt(plainText uint64) uint64 {
	// Create a new AES cipher using a random key
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		fmt.Println("Err when encrypt msg: ", err.Error())
		return 0
	}

	// Create a counter mode stream
	stream := cipher.NewCTR(block, make([]byte, aes.BlockSize))

	// Encrypt the plaintext
	cipherText := make([]byte, binary.MaxVarintLen64)
	binary.BigEndian.PutUint64(cipherText, plainText)
	stream.XORKeyStream(cipherText, cipherText)

	return binary.BigEndian.Uint64(cipherText)
}

func decrypt(cipherText uint64) uint64 {
	// Create a new AES cipher using a random key
	block, err := aes.NewCipher([]byte(encryptKey))
	if err != nil {
		fmt.Println("Err when decrypt msg: ", err.Error())
		return 0
	}

	// Create a counter mode stream
	stream := cipher.NewCTR(block, make([]byte, aes.BlockSize))

	// Decrypt the ciphertext
	plainTextBytes := make([]byte, binary.MaxVarintLen64)
	binary.BigEndian.PutUint64(plainTextBytes, cipherText)
	stream.XORKeyStream(plainTextBytes, plainTextBytes)

	return binary.BigEndian.Uint64(plainTextBytes)
}

type discoveryNotifee2 struct {
	h host.Host
}

// HandlePeerFound handle when found new peer
func (n *discoveryNotifee2) HandlePeerFound(peerInfo peer.AddrInfo) {
	fmt.Println("found peer", peerInfo.String())
	err := connectP2p(n.h, &peerInfo)
	if err != nil {
		fmt.Println("Connection failed:", err)
	}
	showConnections(n.h)
}

// show connection when connect to new peer
func showConnections(host host.Host) {
	conns := host.Network().Conns()
	for _, conn := range conns {
		fmt.Printf("Connection from %s to %s\n", conn.LocalPeer(), conn.RemotePeer())
	}
}
