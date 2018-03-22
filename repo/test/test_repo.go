package test

import (
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/actions"
	"github.com/qri-io/qri/repo/profile"
)

// base64-encoded Test Private Key, decoded in init
// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
var (
	testPk  = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)
	privKey crypto.PrivKey

	testPeerProfile = &profile.Profile{
		Peername: "peer",
		ID:       "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
	}
)

func init() {
	data, err := base64.StdEncoding.DecodeString(string(testPk))
	if err != nil {
		panic(err)
	}
	testPk = data

	privKey, err = crypto.UnmarshalPrivateKey(testPk)
	if err != nil {
		panic(fmt.Errorf("error unmarshaling private key: %s", err.Error()))
		return
	}
}

// NewTestRepo generates a repository usable for testing purposes
func NewTestRepo() (mr repo.Repo, err error) {
	datasets := []string{"movies", "cities", "counter", "craigslist"}

	ms := cafs.NewMapstore()
	mr, err = repo.NewMemRepo(testPeerProfile, ms, repo.MemProfiles{})
	if err != nil {
		return
	}

	mr.SetPrivateKey(privKey)
	act := actions.Dataset{mr}

	gopath := os.Getenv("GOPATH")
	for _, k := range datasets {

		tc, err := dstest.NewTestCaseFromDir(fmt.Sprintf("%s/src/github.com/qri-io/qri/repo/test/testdata/%s", gopath, k))
		if err != nil {
			return nil, err
		}

		datafile := cafs.NewMemfileBytes(tc.DataFilename, tc.Data)

		if _, err = act.CreateDataset(tc.Name, tc.Input, datafile, true); err != nil {
			return nil, fmt.Errorf("%s error creating dataset: %s", k, err.Error())
		}
	}

	return
}

func pkgPath(paths ...string) string {
	gp := os.Getenv("GOPATH")
	return filepath.Join(append([]string{gp, "src/github.com/qri-io/qri/repo/test"}, paths...)...)
}

// NewMemRepoFromDir reads a director of testCases and calls createDataset
// on each case with the given privatekey, yeilding a repo where the peer with
// this pk has created each dataset in question
func NewMemRepoFromDir(path string) (repo.Repo, crypto.PrivKey, error) {
	pro, pk, err := ReadRepoConfig(filepath.Join(path, "config.yaml"))
	if err != nil {
		return nil, pk, err
	}

	ms := cafs.NewMapstore()
	mr, err := repo.NewMemRepo(pro, ms, repo.MemProfiles{})
	if err != nil {
		return mr, pk, err
	}
	mr.SetPrivateKey(pk)
	act := actions.Dataset{mr}

	tc, err := dstest.LoadTestCases(path)
	if err != nil {
		return mr, pk, err
	}

	for _, c := range tc {
		if _, err := act.CreateDataset(c.Name, c.Input, c.DataFile(), true); err != nil {
			return mr, pk, err
		}
	}

	return mr, pk, nil
}

// ReadRepoConfig loads configuration data from a .yaml file
func ReadRepoConfig(path string) (pro *profile.Profile, pk crypto.PrivKey, err error) {
	cfgData, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("error reading config file: %s", err.Error())
		return
	}

	cfg := &struct {
		PrivateKey string
		Profile    *profile.Profile
	}{}
	if err = yaml.Unmarshal(cfgData, &cfg); err != nil {
		err = fmt.Errorf("error unmarshaling yaml data: %s", err.Error())
		return
	}

	pro = cfg.Profile

	pkdata, err := base64.StdEncoding.DecodeString(cfg.PrivateKey)
	if err != nil {
		err = fmt.Errorf("invalde privatekey: %s", err.Error())
		return
	}

	pk, err = crypto.UnmarshalPrivateKey(pkdata)
	if err != nil {
		err = fmt.Errorf("error unmarshaling privatekey: %s", err.Error())
		return
	}

	return
}

// BadDataFile is a bunch of bad CSV data
var BadDataFile = cafs.NewMemfileBytes("bad_csv_file.csv", []byte(`
asdlkfasd,,
fm as
f;lajsmf 
a
's;f a'
sdlfj asdf`))

// BadDataFormatFile has weird line lengths
var BadDataFormatFile = cafs.NewMemfileBytes("abc.csv", []byte(`
"colA","colB","colC","colD"
1,2,3,4
1,2,3`))

// BadStructureFile has double-named columns
var BadStructureFile = cafs.NewMemfileBytes("badStructure.csv", []byte(`
colA, colB, colB, colC
1,2,3,4
1,2,3,4`))

// JobsByAutomationFile is real, valid data
var JobsByAutomationFile = cafs.NewMemfileBytes("jobs_ranked_by_automation_probability.csv", []byte(`rank,probability_of_automation,soc_code,job_title
702,"0.99","41-9041","Telemarketers"
701,"0.99","23-2093","Title Examiners, Abstractors, and Searchers"
700,"0.99","51-6051","Sewers, Hand"
699,"0.99","15-2091","Mathematical Technicians"
698,"0.99","13-2053","Insurance Underwriters"
697,"0.99","49-9064","Watch Repairers"
696,"0.99","43-5011","Cargo and Freight Agents"
695,"0.99","13-2082","Tax Preparers"
694,"0.99","51-9151","Photographic Process Workers and Processing Machine Operators"
693,"0.99","43-4141","New Accounts Clerks"
692,"0.99","25-4031","Library Technicians"
691,"0.99","43-9021","Data Entry Keyers"
690,"0.98","51-2093","Timing Device Assemblers and Adjusters"
689,"0.98","43-9041","Insurance Claims and Policy Processing Clerks"
688,"0.98","43-4011","Brokerage Clerks"
687,"0.98","43-4151","Order Clerks"
686,"0.98","13-2072","Loan Officers"
685,"0.98","13-1032","Insurance Appraisers, Auto Damage"
684,"0.98","27-2023","Umpires, Referees, and Other Sports Officials"
683,"0.98","43-3071","Tellers"
682,"0.98","51-9194","Etchers and Engravers"
681,"0.98","51-9111","Packaging and Filling Machine Operators and Tenders"
680,"0.98","43-3061","Procurement Clerks"
679,"0.98","43-5071","Shipping, Receiving, and Traffic Clerks"
678,"0.98","51-4035","Milling and Planing Machine Setters, Operators, and Tenders, Metal and Plastic"
677,"0.98","13-2041","Credit Analysts"
676,"0.98","41-2022","Parts Salespersons"
675,"0.98","13-1031","Claims Adjusters, Examiners, and Investigators"
674,"0.98","53-3031","Driver/Sales Workers"
673,"0.98","27-4013","Radio Operators"
`))
