package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewPeersCommand cerates a new `qri peers` cobra command
func NewPeersCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &PeersOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "peers",
		Short: "Commands for working with peers",
		Long: `
The ` + "`peers`" + ` commands allow you to interact with other peers on the Qri network.
In order for these commands to work, you must be running a Qri node. This 
node allows you to communicate on the network. To spin up a Qri node, run
` + "`qri connect`" + ` in a separate terminal. This will connect you to the network, 
until you choose to close the connection by ending the session or closing 
the terminal.`,
		Annotations: map[string]string{
			"group": "network",
		},
	}

	info := &cobra.Command{
		Use:   "info",
		Short: `Get info on a Qri peer`,
		Long: `
The peers info command returns a peer's profile information. The default
format is yaml.

Using the ` + "`--verbose`" + ` flag, you can also view a peer's network information.

You must have ` + "`qri connect`" + ` running in another terminal.`,
		Example: `  show info on a peer named "b5":
  $ qri peers info b5

  show info in json:
  $ qri peers info b5 --format json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Info()
		},
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List known qri peers",
		Long: `
Lists the peers to which your Qri node is connected. 

You must have ` + "`qri connect`" + ` running in another terminal.

To find peers that are not online, but to which your node has previously been 
connected, use the ` + "`--cached`" + ` flag.`,
		Example: `  # spin up a Qri node
  qri connect

  # thenin a separate terminal, to list qri peers:
  qri peers list

  # to ensure you get a cached version of the list:
  qri peers list --cached`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.List()
		},
	}

	connect := &cobra.Command{
		Use:   "connect",
		Short: "Connect to a peer",
		Long: `
Connect to a peer using a peername, peer ID, or multiaddress. Qri will use this name, id, or address
to find a peer to which it has not automatically connected. 

You must have a Qri node running (` + "`qri connect`" + `) in a separate terminal. You will only be able 
to connect to a peer that also has spun up it's own Qri node.

A multiaddress, or multiaddr, is the most specific way to refer to a peer's location, and is therefore
the most sure-fire way to connect to a peer. `,
		Example: `  # spin up a Qri node
  qri connect

  # in a separate terminal, connect to a specific peer
  qri peers connect /ip4/192.168.0.194/tcp/4001/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Connect()
		},
	}

	disconnect := &cobra.Command{
		Use:   "disconnect",
		Short: "Explicitly close a connection to a peer",
		Args:  cobra.MinimumNArgs(1),
		Long: `
Explicitly close a connection to a peer using a peername, peer id, or multiaddress. 

You can close all connections to the Qri network by ending your Qri node session. 

Use the disconnect command when you want to stay connected to the network, but want to 
close your connection to a specific peer. This could be because that connection is hung,
the connection is pulling too many resources, or because you simply no longer need an
explicit connection.  This is not the same as blocking a peer or connection.

Once you close a connection to a peer, you or that peer can immediately open another 
connection.

You must have ` + "`qri connect`" + ` running in another terminal.`,
		Example: `  # disconnect from a peer using a multiaddr
  qri peers disconnect /ip4/192.168.0.194/tcp/4001/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Disconnect()
		},
	}

	info.Flags().BoolVarP(&o.Verbose, "verbose", "v", false, "show verbose profile info")
	info.Flags().StringVarP(&o.Format, "format", "", "yaml", "output format. formats: yaml, json")

	list.Flags().BoolVarP(&o.Cached, "cached", "c", false, "show peers that aren't online, but previously seen")
	list.Flags().StringVarP(&o.Network, "network", "n", "", "specify network to show peers from [ipfs]")
	list.Flags().IntVarP(&o.Limit, "limit", "l", 200, "limit max number of peers to show")
	list.Flags().IntVarP(&o.Offset, "offset", "s", 0, "number of peers to skip during listing")

	cmd.AddCommand(info, list, connect, disconnect)

	return cmd
}

// PeersOptions encapsulates state for the peers command
type PeersOptions struct {
	IOStreams

	Peername string
	Verbose  bool
	Format   string
	Cached   bool
	Network  string
	Limit    int
	Offset   int

	UsingRPC     bool
	PeerRequests *lib.PeerRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *PeersOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		o.Peername = args[0]
	}
	o.UsingRPC = f.RPC() != nil
	o.PeerRequests, err = f.PeerRequests()
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

	res := &config.ProfilePod{}
	if err = o.PeerRequests.Info(p, res); err != nil {
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
	if o.Network == "ipfs" {
		res := []string{}
		if err := o.PeerRequests.ConnectedIPFSPeers(&o.Limit, &res); err != nil {
			return err
		}

		for i, p := range res {
			printSuccess(o.Out, "%d.\t%s", i+1, p)
		}
	} else {

		// if we don't have an RPC client, assume we're not connected
		if o.UsingRPC && !o.Cached {
			printInfo(o.Out, "qri not connected, listing cached peers")
			o.Cached = true
		}

		p := &lib.PeerListParams{
			Limit:  o.Limit,
			Offset: o.Offset,
			Cached: o.Cached,
		}
		res := []*config.ProfilePod{}
		if err = o.PeerRequests.List(p, &res); err != nil {
			return err
		}

		fmt.Fprintln(o.Out, "")
		for i, peer := range res {
			printPeerInfo(o.Out, i, peer)
		}
	}
	return nil
}

// Connect attempts to connect to a peer
func (o *PeersOptions) Connect() (err error) {
	pcpod := lib.NewPeerConnectionParamsPod(o.Peername)
	res := &config.ProfilePod{}
	if err = o.PeerRequests.ConnectToPeer(pcpod, res); err != nil {
		return err
	}

	printSuccess(o.Out, "successfully connected to %s:\n", res.Peername)
	printPeerInfo(o.Out, 0, res)
	return nil
}

// Disconnect attempts to disconnect from a peer
func (o *PeersOptions) Disconnect() (err error) {
	pcpod := lib.NewPeerConnectionParamsPod(o.Peername)
	res := false
	if err = o.PeerRequests.DisconnectFromPeer(pcpod, &res); err != nil {
		return err
	}

	printSuccess(o.Out, "disconnected")
	return nil
}
