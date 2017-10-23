# go-libp2p Q2 roadmap

- [ ] websockets transport to communicate with js-ipfs
- [ ] line switching for libp2p (relay)
  - [ ] add config opt-in for relaying traffic
  - [ ] add 'find indirect connections' logic to dht
- [ ] Pubsub
  - [ ] ship initial implementation to libp2p
  - [ ] ipfs 'flare' in go-ipfs
- [ ] NAT Traversal 
  - [ ] Goal: No "I can't connect to my other node" issues for two weeks
  - [ ] address discovery issue ipfs/#2509 ipfs/#2413
  - [ ] testbed to simulate NAT scenarios
  - [ ] command to clear dial backoff ipfs/#2456
- [ ] libp2p connection closing
  - [ ] figure out when we want to close connections
  - [ ] select 'good' number of connections while idle (based on available resources?)
- [ ] fix mocknet issues #31 #32
