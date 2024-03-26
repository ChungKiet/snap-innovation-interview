# p2p sample network [support peer discovery using mdns]

This program demonstrates a simple p2p network. 

## How to build this example?

```
go get -v -d ./...

go build
```

## Usage

Use two different terminal windows to run

```
./chat-room 
./chat-room -pd /ip4/127.0.0.1/tcp/52205/p2p/12D3KooWEEqWv4BsDWmrpzn75ojWFwy1N9XSEp3fdZ4uGtdzqij6
./chat-room -pd /ip4/127.0.0.1/tcp/52205/p2p/12D3KooWEEqWv4BsDWmrpzn75ojWFwy1N9XSEp3fdZ4uGtdzqij6,/ip4/127.0.0.1/tcp/52312/p2p/12D3KooWEEqWv4BsDWmrpzn75ojWFwy1N9XSEp3fdZ4uGtdzqij6
```

## Authors
1. kiet.cht
