package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewPeersCommand cerates a new `qri peers` cobra command
func NewPeersCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &PeersOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "peers",
		Short: "commands for working with peers",
		Long: `The ` + "`peers`" + ` commands interact with other peers on the Qri network. In
order for these commands to work, you must be running a Qri node, which is
responsible for peer-to-peer communication. To spin up a Qri node, run
` + "`qri connect`" + ` in a separate terminal. This connects you to the network
until you choose to close the connection by ending the session or closing 
the terminal.`,
		Annotations: map[string]string{
			"group": "network",
		},
	}

	info := &cobra.Command{
		Use:   "info PEER",
		Short: `get info on a Qri peer`,
		Long: `The peers info command returns a peer's profile information. The default
format is yaml.

Using the ` + "`--verbose`" + ` flag, you can also view a peer's network information.

You must have ` + "`qri connect`" + ` running in another terminal.`,
		Example: `  # Show info on a peer named "b5":
  $ qri peers info b5

  # Show info in json:
  $ qri peers info b5 --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Info()
		},
	}

	info.Flags().BoolVarP(&o.Verbose, "verbose", "v", false, "show verbose profile info")
	info.Flags().StringVarP(&o.Format, "format", "", "yaml", "output format. formats: yaml, json")

	list := &cobra.Command{
		Use:   "list",
		Short: "list known qri peers",
		Long: `Lists the peers to which your Qri node is connected. 

You must have ` + "`qri connect`" + ` running in another terminal.

To find peers that are not online, but to which your node has previously been 
connected, use the ` + "`--cached`" + ` flag.`,
		Example: `  # Spin up a Qri node:
  $ qri connect

  # Then in a separate terminal, to list qri peers:
  $ qri peers list

  # To ensure you get a cached version of the list:
  $ qri peers list --cached`,
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.List()
		},
	}

	list.Flags().BoolVarP(&o.Cached, "cached", "c", false, "show peers that aren't online, but previously seen")
	list.Flags().StringVarP(&o.Network, "network", "n", "", "specify network to show peers from (qri|ipfs) (defaults to qri)")
	list.Flags().StringVarP(&o.Format, "format", "", "", "output format. formats: simple")
	list.Flags().IntVar(&o.Offset, "offset", 0, "number of peers to skip from the results, default 0")
	list.Flags().IntVar(&o.Limit, "limit", 200, "max number of peers to show, default 200")

	connect := &cobra.Command{
		Use:   "connect (NAME|ADDRESS)",
		Short: "connect to a peer",
		Long: `Connect to a peer using a peername, peer ID, or multiaddress. Qri will use this name, id, or address
to find a peer to which it has not automatically connected. 

You must have a Qri node running (` + "`qri connect`" + `) in a separate terminal. You will only be able 
to connect to a peer that also has spun up its own Qri node.

A multiaddress, or multiaddr, is the most specific way to refer to a peer's location, and is therefore
the most sure-fire way to connect to a peer.`,
		Example: `  # Spin up a Qri node:
  $ qri connect

  # In a separate terminal, connect to a specific peer:
  $ qri peers connect /ip4/192.168.0.194/tcp/4001/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Connect()
		},
	}

	disconnect := &cobra.Command{
		Use:   "disconnect (NAME|ADDRESS)",
		Short: "explicitly close a connection to a peer",
		Args:  cobra.ExactArgs(1),
		Long: `Explicitly close a connection to a peer using a peername, peer id, or multiaddress. 

You can close all connections to the Qri network by ending your Qri node session. 

Use the disconnect command when you want to stay connected to the network, but want to 
close your connection to a specific peer. This could be because that connection is hung,
the connection is pulling too many resources, or because you simply no longer need an
explicit connection.  This is not the same as blocking a peer or connection.

Once you close a connection to a peer, you or that peer can immediately open another 
connection.

You must have ` + "`qri connect`" + ` running in another terminal.`,
		Example: `  # Disconnect from a peer using a multiaddr:
  $ qri peers disconnect /ip4/192.168.0.194/tcp/4001/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Disconnect()
		},
	}

	cmd.AddCommand(info, list, connect, disconnect)

	return cmd
}

// PeersOptions encapsulates state for the peers command
type PeersOptions struct {
	ioes.IOStreams

	Peername string
	Verbose  bool
	Format   string
	Cached   bool
	Network  string
	Offset   int
	Limit    int

	UsingRPC bool
	Instance *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *PeersOptions) Complete(f Factory, args []string) (err error) {
	if o.Instance, err = f.Instance(); err != nil {
		return err
	}
	if len(args) > 0 {
		o.Peername = args[0]
	}
	o.UsingRPC = f.HTTPClient() != nil
	return
}

// Info gets peer info
func (o *PeersOptions) Info() (err error) {
	if !(o.Format == "yaml" || o.Format == "json") {
		return fmt.Errorf("format must be either `yaml` or `json`")
	}

	var data []byte

	p := &lib.PeerInfoParams{
		Peername: o.Peername,
		Verbose:  o.Verbose,
	}

	ctx := context.TODO()
	res, err := o.Instance.Peer().Info(ctx, p)
	if err != nil {
		return err
	}

	switch o.Format {
	case "json":
		if data, err = json.MarshalIndent(res, "", "  "); err != nil {
			return err
		}
	case "yaml":
		if data, err = yaml.Marshal(res); err != nil {
			return err
		}
	}

	printInfo(o.Out, "\n"+string(data)+"\n")
	return
}

// List shows a list of peers
func (o *PeersOptions) List() (err error) {

	res := []*config.ProfilePod{}
	ctx := context.TODO()

	if o.Network == "ipfs" {
		params := &lib.ConnectionsParams{List: params.List{Limit: o.Limit}}
		res, err = o.Instance.Peer().ConnectedQriProfiles(ctx, params)
		if err != nil {
			return err
		}
	} else {
		// if we don't have an RPC client, assume we're not connected
		if !o.UsingRPC && !o.Cached {
			printInfo(o.ErrOut, "qri not connected, listing cached peers")
			o.Cached = true
		}

		p := &lib.PeerListParams{
			List: params.List{
				Limit:  o.Limit,
				Offset: o.Offset,
			},
			Cached: o.Cached,
		}
		res, err = o.Instance.Peer().List(ctx, p)
		if err != nil {
			return err
		}
	}

	items := make([]fmt.Stringer, len(res))
	peerNames := make([]string, len(res))
	for i, p := range res {
		items[i] = peerStringer(*p)
		peerNames[i] = p.Peername
	}

	if o.Format == "simple" {
		printlnStringItems(o.Out, peerNames)
	} else {
		printItems(o.Out, items, o.Offset)
	}
	return
}

// Connect attempts to connect to a peer
func (o *PeersOptions) Connect() (err error) {
	pcpod := lib.NewConnectParamsPod(o.Peername)
	ctx := context.TODO()
	res, err := o.Instance.Peer().Connect(ctx, pcpod)
	if err != nil {
		return err
	}

	printSuccess(o.Out, "successfully connected to %s:\n", res.Peername)
	peer := peerStringer(*res)
	fmt.Fprint(o.Out, peer.String())
	return nil
}

// Disconnect attempts to disconnect from a peer
func (o *PeersOptions) Disconnect() (err error) {
	pcpod := lib.NewConnectParamsPod(o.Peername)
	ctx := context.TODO()
	if err = o.Instance.Peer().Disconnect(ctx, pcpod); err != nil {
		return err
	}

	printSuccess(o.Out, "disconnected")
	return nil
}
