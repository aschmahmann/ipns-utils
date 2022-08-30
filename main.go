package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	crypto_pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"os"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multihash"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipns"
	ipns_pb "github.com/ipfs/go-ipns/pb"

	psr "github.com/libp2p/go-libp2p-pubsub-router"

	"github.com/urfave/cli/v2"
)

func main() {
	var ipnsKey, topic string
	var cidVersion int

	app := &cli.App{
		Name: "ipns-utils",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "create a IPNS records",
				Subcommands: []*cli.Command{
					{
						Name:  "id",
						Usage: "create an IPNS identifier",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Required: false,
								Name:     "output-base",
								Value:    "",
								Usage:    "multibase name or prefix character, none means no encoding",
							},
							&cli.StringFlag{
								Required: false,
								Name:     "type",
								Value:    "ed25519",
								Usage:    "type of the key to create",
							},
							&cli.IntFlag{
								Required: false,
								Name:     "size",
								Value:    -1,
								Usage:    "size of the key to generate (only valid to be set for RSA keys which defaults to 2048)",
							},
						},
						Action: func(c *cli.Context) error {
							return createIPNSID(c.String("type"), c.Int("size"), c.String("output-base"))
						},
					},
					{
						Name:      "record",
						Usage:     "record <value>",
						UsageText: "create an IPNS record",
						Flags: []cli.Flag{
							&cli.PathFlag{
								Required: false,
								Name:     "key-file",
								Value:    "",
								Usage:    "The path to the private key",
							},
							&cli.PathFlag{
								Required: false,
								Name:     "key-encoded",
								Value:    "",
								Usage:    "multibase encoded private key",
							},
							&cli.StringFlag{
								Required: false,
								Name:     "output-base",
								Value:    "",
								Usage:    "multibase name or prefix character, none means no encoding",
							},
							&cli.DurationFlag{
								Required: false,
								Name:     "ttl",
								Value:    0,
							},
							&cli.TimestampFlag{
								Required:    false,
								Name:        "eol",
								Layout:      "2006-01-02T15:04:05",
								DefaultText: "End of life for the record, in UTC. Time format is 2006-01-02T15:04:05. Defaults to 24 hours from now",
							},
							&cli.DurationFlag{
								Required:    false,
								Name:        "lifetime",
								DefaultText: "An alternative to eol. Defines how long from now a record should be valid for (e.g. 30s, -10m, 24.5h). Defaults to 24 hours",
							},
							&cli.Int64Flag{
								Required: false,
								Name:     "seqno",
								Value:    0,
							},
							&cli.StringFlag{
								Required: false,
								Name:     "value",
								Value:    "/ipfs/bafkqaaa",
							},
						},
						Action: func(c *cli.Context) error {
							seqno := c.Int64("seqno")
							ttl := c.Duration("ttl")
							eol := c.Timestamp("eol")
							const lifetimeStr = "lifetime"
							lifetime := c.Duration(lifetimeStr)

							if c.IsSet(lifetimeStr) && eol != nil {
								return errors.New("cannot define lifetime and eol on a record, choose one")
							}

							if !c.IsSet(lifetimeStr) && eol == nil {
								eolTime := time.Now().Add(time.Hour * 24)
								eol = &eolTime
							} else if c.IsSet(lifetimeStr) {
								eolTime := time.Now().Add(lifetime)
								eol = &eolTime
							}

							value := c.String("value")
							keyFile := c.Path("key-file")
							keyEncoded := c.String("key-encoded")

							var key crypto.PrivKey
							if keyFile != "" && keyEncoded != "" {
								return errors.New("cannot pass a key file and encoded key")
							} else if keyFile == "" && keyEncoded == "" {
								return errors.New("no key specified, specify a key file or encoded key")
							} else if keyFile != "" {
								keyBytes, err := os.ReadFile(keyFile)
								if err != nil {
									return err
								}
								priv, err := crypto.UnmarshalPrivateKey(keyBytes)
								if err != nil {
									return err
								}
								key = priv
							} else {
								_, keyBytes, err := multibase.Decode(keyEncoded)
								if err != nil {
									return err
								}
								priv, err := crypto.UnmarshalPrivateKey(keyBytes)
								if err != nil {
									return err
								}
								key = priv
							}

							return createIPNSRecord(seqno, ttl, *eol, value, key, c.String("output-base"))
						},
					},
				},
			},
			{
				Name: "parse",
				Subcommands: []*cli.Command{
					{
						Name:      "record",
						Usage:     "record <record>",
						UsageText: "parse an IPNS record. The public key, if present, is multibase encoded",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Required: false,
								Name:     "input-type",
								Value:    "bytes",
								Usage:    "record input type, may be: bytes, multibase, or path",
							},
						},
						Action: func(c *cli.Context) error {
							recordInput := c.Args().First()
							inputType := c.Path("input-type")
							var recordBytes []byte
							var err error
							switch inputType {
							case "bytes":
								recordBytes = []byte(recordInput)
							case "multibase":
								_, recordBytes, err = multibase.Decode(recordInput)
								if err != nil {
									return err
								}
							case "path":
								recordBytes, err = os.ReadFile(recordInput)
								if err != nil {
									return err
								}
							default:
								return errors.New("must pass either a record file or encoded record to parse")
							}

							return parseIPNSRecord(recordBytes)
						},
					},
					{
						Name:      "key",
						Usage:     "key <key>",
						UsageText: "parse the encoded libp2p key format used with IPNS. The key material is multibase encoded",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Required: false,
								Name:     "input-type",
								Value:    "bytes",
								Usage:    "record input type, may be: bytes, multibase, or path",
							},
							&cli.BoolFlag{
								Required: false,
								Name:     "private-key",
								Value:    true,
							},
						},
						Action: func(c *cli.Context) error {
							keyInput := c.Args().First()
							inputType := c.Path("input-type")
							var keyBytes []byte
							var err error
							switch inputType {
							case "bytes":
								keyBytes = []byte(keyInput)
							case "multibase":
								_, keyBytes, err = multibase.Decode(keyInput)
								if err != nil {
									return err
								}
							case "path":
								keyBytes, err = os.ReadFile(keyInput)
								if err != nil {
									return err
								}
							default:
								return errors.New("must pass either a record file or encoded record to parse")
							}

							return parselibp2pkey(keyBytes, c.Bool("private-key"))
						},
					},
				},
			},
			{
				Name:    "pubsub",
				Aliases: []string{"p"},
				Usage:   "IPNS over PubSub utilities",
				Subcommands: []*cli.Command{
					{
						Name:    "get-topic",
						Aliases: []string{"t"},
						Usage:   "get pubsub topic name from key",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Required:    true,
								Name:        "key",
								Aliases:     []string{"k"},
								Usage:       "The CIDv0 or CIDv1 representations of an IPNS Key",
								Destination: &ipnsKey,
							},
						},
						Action: func(c *cli.Context) error {
							topic, err := getPubSubTopic(ipnsKey)
							if err != nil {
								return err
							}
							fmt.Println(topic)
							return nil
						},
					},
					{
						Name:    "get-key",
						Usage:   "get IPNS key from pubsub topic",
						Aliases: []string{"k"},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Required:    true,
								Name:        "topic",
								Aliases:     []string{"t"},
								Usage:       "The CIDv0 or CIDv1 representations of an IPNS Key",
								Destination: &topic,
							},
							&cli.IntFlag{
								Required:    false,
								Name:        "format",
								Aliases:     []string{"f"},
								Value:       0,
								Usage:       "Output as CIDv0 or CIDv1",
								Destination: &cidVersion,
							},
						},
						Action: func(c *cli.Context) error {
							key, err := getIPNSKey(topic, cidVersion)
							if err != nil {
								return err
							}
							fmt.Println(key)
							return nil
						},
					},
					{
						Name:    "get-dht-key-from-topic",
						Usage:   "get the rendezvous DHT key from the pubsub topic",
						Aliases: []string{"dkt"},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Required:    false,
								Name:        "topic",
								Aliases:     []string{"t"},
								Usage:       "The CIDv0 or CIDv1 representations of an IPNS Key",
								Destination: &topic,
							},
						},
						Action: func(c *cli.Context) error {
							key, err := getDHTRendezvousKey(topic)
							if err != nil {
								return err
							}
							fmt.Println(key)
							return nil
						},
					},
					{
						Name:    "get-dht-key-from-key",
						Usage:   "get the rendezvous DHT key from the IPNS key",
						Aliases: []string{"dkk"},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Required:    true,
								Name:        "key",
								Aliases:     []string{"k"},
								Usage:       "The CIDv0 or CIDv1 representations of an IPNS Key",
								Destination: &ipnsKey,
							},
						},
						Action: func(c *cli.Context) error {
							topic, err := getPubSubTopic(ipnsKey)
							if err != nil {
								return err
							}
							key, err := getDHTRendezvousKey(topic)
							if err != nil {
								return err
							}
							fmt.Println(key)
							return nil
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

func createIPNSID(keyType string, keyLen int, outputBase string) error {
	var priv crypto.PrivKey
	var pub crypto.PubKey

	switch keyType {
	case "rsa":
		rsaLen := keyLen
		if keyLen <= 0 {
			rsaLen = 2048
		}

		var err error
		priv, pub, err = crypto.GenerateKeyPairWithReader(crypto.RSA, rsaLen, rand.Reader)
		if err != nil {
			return err
		}
	case "ed25519":
		var err error
		priv, pub, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return err
		}
	case "secp256k1":
		var err error
		priv, pub, err = crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			return err
		}
	case "ecdsa":
		var err error
		priv, pub, err = crypto.GenerateECDSAKeyPair(rand.Reader)
		if err != nil {
			return err
		}
	default:
		return crypto.ErrBadKeyType
	}

	privKeyBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return err
	}

	recPkHash, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(os.Stderr, "identfier: %s\n", peer.ToCid(recPkHash)); err != nil {
		return err
	}

	if outputBase != "" {
		enc, err := multibase.EncoderByName(outputBase)
		if err != nil {
			return err
		}
		fmt.Printf(enc.Encode(privKeyBytes))
		return nil
	}
	_, err = os.Stdout.Write(privKeyBytes)
	return nil
}

func createIPNSRecord(seqno int64, ttl time.Duration, eol time.Time, value string, privKey crypto.PrivKey, outputBase string) error {
	rec, err := ipns.Create(privKey, []byte(value), uint64(seqno), eol, ttl)
	if err != nil {
		return err
	}

	pub := privKey.GetPublic()
	if err := ipns.EmbedPublicKey(pub, rec); err != nil {
		return err
	}

	recBytes, err := rec.Marshal()
	if err != nil {
		return err
	}

	if outputBase != "" {
		enc, err := multibase.EncoderByName(outputBase)
		if err != nil {
			return err
		}
		fmt.Println(enc.Encode(recBytes))
		return nil
	}
	_, err = os.Stdout.Write(recBytes)
	return err
}

func parseIPNSRecord(data []byte) error {
	rec := &ipns_pb.IpnsEntry{}
	if err := rec.Unmarshal(data); err != nil {
		return err
	}

	eol, err := ipns.GetEOL(rec)
	if err != nil {
		return err
	}

	var ttl time.Duration
	if rec.Ttl != nil {
		ttl = time.Duration(*rec.Ttl)
	}

	pubKeyString := ""

	if len(rec.PubKey) > 0 {
		pubKeyString, err = multibase.Encode(multibase.Base16, rec.PubKey)
		if err != nil {
			return err
		}
	}

	fmt.Printf(`
{
    "Value": "%s",
    "SequenceNumber" : %d,
    "EOL" : "%v",
    "TTL" : "%v",
    "PubKey" : "%s"
}

`, rec.Value, *rec.Sequence, eol, ttl, pubKeyString,
	)
	return nil
}

func parselibp2pkey(data []byte, isPrivateKey bool) error {
	var keyType crypto_pb.KeyType
	var keyMaterial []byte

	if isPrivateKey {
		privKey, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			return err
		}

		keyType = privKey.Type()

		keyMaterial, err = privKey.Raw()
		if err != nil {
			return err
		}
	} else {
		pubKey, err := crypto.UnmarshalPublicKey(data)
		if err != nil {
			return err
		}

		keyType = pubKey.Type()

		keyMaterial, err = pubKey.Raw()
		if err != nil {
			return err
		}
	}

	keyMaterialString, err := multibase.Encode(multibase.Base16, keyMaterial)
	if err != nil {
		return err
	}

	fmt.Printf(`
{
	"Private Key" : %t,
	"Key Type": "%s",
	"Key Material" : "%s",
}

`, isPrivateKey, keyType, keyMaterialString,
	)
	return nil
}

func getPubSubTopic(ipnsKey string) (string, error) {
	c, err := cid.Decode(ipnsKey)
	if err != nil {
		return "", err
	}

	switch c.Version() {
	case 0:
		key := "/ipns/" + c.KeyString()
		return psr.KeyToTopic(key), nil
	case 1:
		key := "/ipns/" + string(c.Hash())
		return psr.KeyToTopic(key), nil
	default:
		return "", fmt.Errorf("IPNS key has unsupported CID version %d", c.Version())
	}
}

func getIPNSKey(topic string, cidVersion int) (string, error) {
	topic = topic[len("/record/"):]
	decoded, err := base64.RawURLEncoding.DecodeString(topic)
	if err != nil {
		return "", err
	}

	decoded = decoded[len("/ipns/"):]
	c, err := cid.Cast(decoded)
	if err != nil {
		return "", err
	}

	switch cidVersion {
	case 0:
		return c.String(), nil
	case 1:
		c = cid.NewCidV1(cid.Libp2pKey, c.Hash())
		return c.String(), nil
	default:
		return "", fmt.Errorf("could not output IPNS Key as unsupported CID version %d", cidVersion)
	}
}

func getDHTRendezvousKey(topic string) (string, error) {
	keybytes, err := multihash.Sum([]byte("floodsub:"+topic), multihash.SHA2_256, -1)
	if err != nil {
		return "", err
	}

	c := cid.NewCidV1(cid.Raw, keybytes)
	return c.String(), nil
}
