package lib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/qri/config"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestSetupTeardown(t *testing.T) {
	_, registryServer := regmock.NewMockServer()

	path := filepath.Join(os.TempDir(), "test_lib_setup_teardown")
	cfg1 := config.DefaultConfig()
	cfg1.Profile = nil
	cfg1.Registry = nil
	params := SetupParams{
		QriRepoPath:    path,
		ConfigFilepath: filepath.Join(path, "config.yml"),
		Config:         cfg1,
	}
	if err := Setup(params); err == nil {
		t.Errorf("expected invalid cfg to fail")
	}

	params.Config = config.DefaultConfig()
	params.Config.Registry.Location = registryServer.URL
	params.SetupIPFS = true
	params.IPFSFsPath = path
	params.SetupIPFSConfigData = ipfsCfg
	if err := Setup(params); err != nil {
		t.Error(err.Error())
	}

	err := Teardown(TeardownParams{
		Config:         params.Config,
		ConfigFilepath: params.ConfigFilepath,
		QriRepoPath:    path,
	})

	if err != nil {
		t.Error(err.Error())
	}

}

var ipfsCfg = []byte(`{
  "Identity": {
    "PeerID": "QmUiF6GyKcNt3fbc9pCN72KF5qgneLt3eufVT3tGEBiR9h",
    "PrivKey": "CAASqQkwggSlAgEAAoIBAQD8vTnrl8vClngB1zSSOvlL15tKwXabTdFMTsIXFE1j9TxdksgKinBhT/fvbbt3W4Il8kmhz37ShlUQ8alkr2Uf7/cW6VPD8BdISfzEXc8th5rl0mDxJtKvPkYYbRcBZwUH2fL1KLaaRRYEVq0BItHpldtfaajtD04H+kYFPSdapxuPLNDPBxyaVoX2Yiqi/PmYuK6RJA1od27SqPicVMC7QXKZxNpz3Y2Q8g1+PAc+2uIbbWQ3Ow2a7E37+/Rt2NzlcB7R/n1Lpj0YhwRmNCxuyEmwkgd0VHFIjyrRwoRhFBA88o5Go2nNRb0tc9Iq1eFDpU+HXLx5uZYBKe/JZXIXAgMBAAECggEBAMZj8zdP7I5Odt1bBNVUnaQ/FpNT0bqPFyADIq/jK+yu8DezpHtBuH1qvIChbmp+1mbbDZmKu06eS+AFEqcKVyL+xsKhXTONH3mLOnMaACsJKzoELjyd8PvGslcyKsDbEUPcfa6bytrGKEY3k440uvnUvGLlGckcHnB8sMIkAuRQgSbRXSRmfvnyy/VW7+l4ivy0t5528M4yG6i+xbgJqYO14SAblIV/c66mYfLXd1enW2bijZEtMkS1537EuU5hy/HQoCJ8OGHx9R4DRzIyhJTonmv1Xh97+dvF3c/ibNjkIQyN3z2heivVCLoRTMlD5QIS4p6PCaahPb4s+NgA9YECgYEA/tYaDt1fTLA44J9SeUVisXk7cQ1Hi4y3IyhrtRQ8M53Z3pKIWbw68Qv4ZRqRoPK8NOclb4A/mKnyi52rIu7TL7Dn/p0KS4fazcclAaGae59z7qAlg+RadBruS1kPHAYCrbh5YWfyNgeuUHxAYrMimKAeT5jwrdW4pTQj7cRJp2cCgYEA/eSsRCeUkSbCCG1cTS5YeRp0Zpp1+sGuSsZkmp9eOzz6J9Bk6Df70gOU4AShpn7smZYlGYpmoPtdgLziSORDoTRPWL3rXLWXNVqyTVheGcrdYtMEf8vNCVduOfOYDekBsJ8xABDelkGGXPOiWkF+yFchx1V2d27SmX/1CJ+kodECgYEA05PXFrhdQ0KcNoKQ6vbcthS9cWNhH0+5TYtlwXYHdaN9G/n1Euvg0/joRqkEd+iQsiunPSfxpUKUia5iRCKdXF84foDL52HoHClXZD9UD4eXrWtxOkwBfZxOdGiAzvd+idU7kc/HnWxLIa/HlSq9cpKeF+AXE3z6TM85dVMfA8kCgYEAz/5a5carHjJrOK4mtI/oKOX0P+4AAvpSR6250zYF42+z25QMZnUellEa0F7a8uP9/mCTahYIt47VbdbPZjmh8dlBu4hy3VNiWXJAqb5f8K9RqFkI0YzrHuECSvV1NsgQ+1mesdggEWYCpfltopUPQR6obH1l/LfMTbYWzgbCv1ECgYBFZYJahdVLmSpZvESJ3UbxkoWEgSGfw8u67NvYSeYAvCLtb9T02HBvYk4AdKYVRBu2rclf1I8mr8pGoBq2eb11A5i9gWw+8YcvkRFv6qJcgcuQKUAGqw95xDd4C0OuQxQMdx3LxIzPHz7OvSO67YhzuafN5aaalt9WQKyfitrubQ=="
  },
  "Datastore": {
    "StorageMax": "10GB",
    "StorageGCWatermark": 90,
    "GCPeriod": "1h",
    "Spec": {
      "mounts": [
        {
          "child": {
            "path": "blocks",
            "shardFunc": "/repo/flatfs/shard/v1/next-to-last/2",
            "sync": true,
            "type": "flatfs"
          },
          "mountpoint": "/blocks",
          "prefix": "flatfs.datastore",
          "type": "measure"
        },
        {
          "child": {
            "compression": "none",
            "path": "datastore",
            "type": "levelds"
          },
          "mountpoint": "/",
          "prefix": "leveldb.datastore",
          "type": "measure"
        }
      ],
      "type": "mount"
    },
    "HashOnRead": false,
    "BloomFilterSize": 0
  },
  "Addresses": {
    "Swarm": [
      "/ip4/0.0.0.0/tcp/4001",
      "/ip6/::/tcp/4001"
    ],
    "Announce": [],
    "NoAnnounce": [],
    "API": "/ip4/127.0.0.1/tcp/5001",
    "Gateway": "/ip4/127.0.0.1/tcp/8080"
  },
  "Mounts": {
    "IPFS": "/ipfs",
    "IPNS": "/ipns",
    "FuseAllowOther": false
  },
  "Discovery": {
    "MDNS": {
      "Enabled": true,
      "Interval": 10
    }
  },
  "Ipns": {
    "RepublishPeriod": "",
    "RecordLifetime": "",
    "ResolveCacheSize": 128
  },
  "Bootstrap": [
    "/dnsaddr/bootstrap.libp2p.io/ipfs/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
    "/dnsaddr/bootstrap.libp2p.io/ipfs/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
    "/dnsaddr/bootstrap.libp2p.io/ipfs/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
    "/dnsaddr/bootstrap.libp2p.io/ipfs/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    "/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
    "/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
    "/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
    "/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
    "/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
    "/ip6/2604:a880:1:20::203:d001/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
    "/ip6/2400:6180:0:d0::151:6001/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
    "/ip6/2604:a880:800:10::4a:5001/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
    "/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd"
  ],
  "Gateway": {
    "HTTPHeaders": {
      "Access-Control-Allow-Headers": [
        "X-Requested-With",
        "Range"
      ],
      "Access-Control-Allow-Methods": [
        "GET"
      ],
      "Access-Control-Allow-Origin": [
        "*"
      ]
    },
    "RootRedirect": "",
    "Writable": false,
    "PathPrefixes": []
  },
  "API": {
    "HTTPHeaders": null
  },
  "Swarm": {
    "AddrFilters": null,
    "DisableBandwidthMetrics": false,
    "DisableNatPortMap": false,
    "DisableRelay": false,
    "EnableRelayHop": false,
    "ConnMgr": {
      "Type": "basic",
      "LowWater": 600,
      "HighWater": 900,
      "GracePeriod": "20s"
    }
  },
  "Reprovider": {
    "Interval": "12h",
    "Strategy": "all"
  },
  "Experimental": {
    "FilestoreEnabled": false,
    "ShardingEnabled": false,
    "Libp2pStreamMounting": false
  }
}`)
