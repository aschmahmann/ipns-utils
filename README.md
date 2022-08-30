# ipns-utils
Some utilities for working with IPNS records

## Record creation

Problem: Have you ever tried to run `ipfs dht put` and realized you don't have any way of creating an IPNS record?

Solution: Run `ipns-utils create record` and it will output a record for you.
If you need a new key to work with `ipns-utils create key` will give you a key.

## Record parsing

Problem: You run `ipfs dht get /ipns/Qmxyz... > ipns_record` and get a file, but you have no easy way of seeing what's inside of it

Solution: Run `ipns-utils parse record` and choose your input type as a file path, bytes, or a multibase encoded string.
If you want to parse private or public key information `ipns-utils parse key` will do it for you.

## PubSub topics

Problem: The IPNS keys that people tend to interact with look like `QmXMuMWm6k3CD3sHV824H2BT1ugcHKF6Tm13ZVM8RhGTB7` (base58 CIDv0 representation) or `bafzbeiegbnjh5uopd5vc22tgkz6chf7a6ala3x5e47vnhv5sq5bzo46tri` (base32 CIDv1 with libp2p-key codec representation), while IPNS over PubSub topics look like `/record/L2lwbnMvEiCGC1J-0c8fai1qZlZ8I5fg8BYN36Tn6tPXsodDl3PTig`. This makes it very difficult to be able to know which IPNS topics you are subscribed to, or to debug the raw pubsub channel to view IPNS updates.

Solution:

`ipns-utils pubsub get-key --topic topicID [--format cidValue]` will convert a pubsub topic into an IPNS key. For example `ipns-utils pubsub get-key --topic /record/L2lwbnMvEiCGC1J-0c8fai1qZlZ8I5fg8BYN36Tn6tPXsodDl3PTig` will return `QmXMuMWm6k3CD3sHV824H2BT1ugcHKF6Tm13ZVM8RhGTB7` and passing the `--format 1` flag will return `bafzbeiegbnjh5uopd5vc22tgkz6chf7a6ala3x5e47vnhv5sq5bzo46tri`.

`ipns utils pubsub get-topic --key key` will convert an IPNS key into a pubsub topic. For example both `ipns utils pubsub get-topic --key QmXMuMWm6k3CD3sHV824H2BT1ugcHKF6Tm13ZVM8RhGTB7` and `ipns utils pubsub get-topic --key bafzbeiegbnjh5uopd5vc22tgkz6chf7a6ala3x5e47vnhv5sq5bzo46tri` output `/record/L2lwbnMvEiCGC1J-0c8fai1qZlZ8I5fg8BYN36Tn6tPXsodDl3PTig`

## Notes

Some command line flags have shortened aliases if/when you get tired of typing out `get-topic` over and over again, but you'll see those on the command line 😃.

This is, so far, a very basic tool for working with IPNS records in a way which has been useful to the author. If you have suggestions or PRs please feel free to add.
