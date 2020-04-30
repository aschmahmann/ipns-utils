package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipns"

	psr "github.com/libp2p/go-libp2p-pubsub-router"

	"github.com/urfave/cli/v2"
)

func main() {
	var outputDir, ipnsKey, topic string
	var cidVersion int

	app := &cli.App{
		Name: "ipns-utils",
		Commands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"c"},
				Usage:   "create an IPNS record",
				Flags: []cli.Flag{
					&cli.PathFlag{
						Required:    false,
						Name:        "output",
						Aliases:     []string{"o"},
						Value:       "",
						Usage:       "The directory to output the record to",
						Destination: &outputDir,
					},
				},
				Action: func(c *cli.Context) error {
					return createIPNSRecord(outputDir)
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
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

func createIPNSRecord(outputDir string) error {
	priv, pub, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return err
	}

	rec, err := ipns.Create(priv, []byte("/test/data"), 0, time.Now().Add(time.Hour*1000))
	if err != nil {
		return err
	}

	if err := ipns.EmbedPublicKey(pub, rec); err != nil {
		return err
	}

	recBytes, err := rec.Marshal()
	if err != nil {
		return err
	}

	recPkHash, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return err
	}

	recPath := path.Join(outputDir, recPkHash.String())
	if err := writeFile(recPath, recBytes); err != nil {
		return err
	}

	fmt.Println(recPath)
	return nil
}

func writeFile(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}
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
