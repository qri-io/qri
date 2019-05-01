package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	util "github.com/datatogether/api/apiutil"
	"github.com/ghodss/yaml"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewPeersCommand cerates a new `qri peers` cobra command
func NewPeersCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
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
	// TODO (ramfox): when we determine the best way to order and paginate peers, restore!
	// list.Flags().IntVar(&o.PageSize, "page-size", 200, "max page size number of peers to show, default 200")
	// list.Flags().IntVar(&o.Page, "page", 1, "page number of peers, default 1")

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
	PageSize int
	Page     int

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

type peer config.ProfilePod

func (p peer) String() string {
	w := &bytes.Buffer{}
	if p.Online {
		fmt.Fprintf(w, "%s | %s\n", p.Peername, "online")
	} else {
		fmt.Fprintf(w, "%s\n", p.Peername)
	}
	fmt.Fprintf(w, "profile ID: %s\n", p.ID)
	if len(p.NetworkAddrs) > 0 {
		fmt.Fprintf(w, "address:    %s\n", p.NetworkAddrs[0])
	}
	fmt.Fprintln(w, "")
	return w.String()
}

type stringer string

func (s stringer) String() string {
	return string(s) + "\n"
}

// List shows a list of peers
func (o *PeersOptions) List() (err error) {

	// convert Page and PageSize to Limit and Offset
	page := util.NewPage(o.Page, o.PageSize)

	var items []fmt.Stringer

	if o.Network == "ipfs" {
		res := []string{}
		limit := page.Limit()
		if err := o.PeerRequests.ConnectedIPFSPeers(&limit, &res); err != nil {
			return err
		}

		items = make([]fmt.Stringer, len(res))
		for i, p := range res {
			items[i] = stringer(p)
		}
	} else {
		// if we don't have an RPC client, assume we're not connected
		if !o.UsingRPC && !o.Cached {
			printInfo(o.Out, "qri not connected, listing cached peers")
			o.Cached = true
		}

		p := &lib.PeerListParams{
			Limit:  page.Limit(),
			Offset: page.Offset(),
			Cached: o.Cached,
		}
		res := []*config.ProfilePod{}
		if err = o.PeerRequests.List(p, &res); err != nil {
			return err
		}

		items = make([]fmt.Stringer, len(res))
		for i, p := range res {
			items[i] = peer(*p)
		}
	}

	return printItems(o.Out, items)
}

func printItems(w io.Writer, items []fmt.Stringer) error {
	// TODO (ramfox): This is POSIX specific, need to expand!
	envPager := os.Getenv("PAGER")
	if envPager == "" {
		envPager = "less"
	}

	buf := &bytes.Buffer{}
	pager := exec.Command(envPager)
	pager.Stdin = buf
	pager.Stdout = w

	prefix := []byte("    ")
	for i, item := range items {
		buf.WriteString(fmtItem(i+1, item.String(), prefix))
	}

	return pager.Run()
}

func fmtItem(i int, item string, prefix []byte) string {
	var res []byte
	bol := true
	b := []byte(item)
	d := []byte(fmt.Sprintf("%d", i))
	prefix1 := append(d, prefix[len(d):]...)
	for i, c := range b {
		if bol && c != '\n' {
			if i == 0 {
				res = append(res, prefix1...)
			} else {
				res = append(res, prefix...)
			}
		}
		res = append(res, c)
		bol = c == '\n'
	}
	return string(res)
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
