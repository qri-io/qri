# config

The config encapsulates qri configuration options & details. configuration is generally stored as a .yaml file, or provided at CLI runtime via command a line argument. 

``` shell
# to get, for example, your profile information
$ qri config get profile

# to set, for example, your api port to 4444
$ qri config set api.port 4444

# to disable rpc connections
$ qri config set rpc.enabled false
```

Objects and arrays can be indexed into using dot notation, eg as shown above to get or set the api port use `api.port`
To get the first element (which is at index 0) in the p2p.qribootstrapaddrs array: 
`qri config get p2p.qribootstrapaddrs.0`

Here is a quick reference of all configurable fields:
[profile](#profile) *object*
  [id](#id) *string*
  [privkey](#privkey) *string*
  [peername](#peername) *string*
  [created](#created) *string*
  [updated](#updated) *string*
  [type](#type) *string*
  [email](#email) *string*
  [name](#name) *string*
  [description](#description) *string*
  [homeurl](#homeurl) *string*
  [color](#color) *string*
  [thumb](#color) *string*
  [profile](#profile-photo) *ipfs hash*
  [poster](#poster-photo) *ipfs hash*
  [twitter](#twitter) *string*
[repo](#repo)
  [middleware](#middleware) *array*
  [type](#repo-type) *string*
[store](#store) *object*
  [type](#store-type) *string*
[p2p](#p2p) *object*
  [enabled](#p2p-enabled) *bool*
  [peerid](#peerid) *base58 hash*
  [pubkey](#pubkey) *string* 
  [privkey](#p2p-privkey) *string*
  [port](#p2p-port) *string*
  [addrs](#addrs) *array*
  [qribootstrapaddrs](#qribootstrapaddrs) *array*
  [profilereplication](#profilereplication) *bool*
  [boostrapaddrs](#bootstrapaddrs) *array*
[cli](#cli) *object*
  [colorizeoutput](#colorizeoutput) *bool*
[api](#api) *object*
  [enabled](#api-enabled) *bool*
  [port](#api-port) *string*
  [readonly](#readonly) *bool*
  [urlroot](#urlroot) *string*
  [tls](#tls) *string*
  [proxyforcehttps](#proxyforcehttps) *string*
  [allowedorigins](#allowedorigins) *array*
[webapp](#webapp) *object*
  [enabled](#webapp-enabled) *bool*
  [port](#webapp-port) *string*
  [analyticstoken](#analyticstoken) *string*
  [scripts](#scripts) *array*
[rpc](#rpc) *object*
  [enabled](#rpc-enabled) *bool*
  [port](#rpc-port) *string*
[logging](#logging) *object*
  [levels](#levels) *object*
    [qriapi](#qriapi) *string*

-----
## Profile
-----
Your profile contains some hairy stuff you shouldn't change once it is set initially. 

**We strongly recommend you don't change your privkey, ID, and peername.**

Created and cpdated are the timestamps that your profile was created and updated. These will be set automatically, we suggest you don't change these either.

Type, email, name, description, homeurl, color, twitter, and profile and poster photo we strongly encourage you to update!

-----
### ID
*Profile ID*
Your id is your identity on Qri and it is set when you first run `qri setup` or `qri connect --setup`. Your datasets, your qri nodes, your profile, your identity to your peers are all tied to this profile id. Changing this is bad news bears and will break everything. 

**DO NOT CHANGE**

-----
### privkey
*private key*
Your private key is generated when you first run `qri setup` or `qri connect --setup`. 

**DO NOT PUBLISH THIS ANYWHERE**. 

Your private key is a form of security. If anyone else has your private key, they can pretend to be you. Also bad news bears to change this.

-----
### peername
Your moniker on qri. The name that is associated with your profile and datasets. 

Let's say your peername is lunalovegood7 (for some reason), and a dataset of yours called best_harry_potter_quotes_ranked.
```
# a peer would get your profile info by:
$ qri peers lunalovegood74

# and get info on your super important dataset using
$ qri info lunalovegood7/best_harry_potter_quotes_ranked
```

Your peername is set when you first run `qri setup` (or `qri setup --peername lunalovegood7`).

The peername will be mutable in the future, but for now changing your peername is as bad as changing your profile id. 

**Do not change your peername after setup**

-----
### created
Date and time timestamp when the qri profile was created on setup. We recommend you do not change this field.

-----
### updated
Date and time timestamp when the qri profile was last updated. We recommend you do not change this field as it should auto update on any profile change. (this auto update feature might not be set yet)

-----
### type
*peer or organization*
Qri profiles can be associated with a single person or an organization. 

**Input options** (*string*): `peer`, `organization`

**Commands:**
```
$ qri config get profile.type

$ qri config set profile.type peer
```

-----
### email
An email address to reach you. If other qri folks can reach you, it will greatly strengthen their trust in yoru datasets.

**Input options** (*string*): valid email address

**Commands:**
```
$ qri config get profile.email

$ qri config set profile.email example@example.com
```

-----
### name
Your name or organizations name.

**Input options** (*string*): max 255 character length

**Commands:**
```
$ qri config get profile.name

$ qri config set profile.name Jane Doe
```

-----
### description
A little bio about you or your organization. Can help other users understand the types of data you are interested in.

**Input options** (*string*): max 255 character length

**Commands:**
```
$ qri config get profile.description 

$ qri config set profile.description "Hi my name is Jane Doe and I am a researcher in the field of loving Harry Potter. I am interested data surrounding the behavior of those who love Harry Potter as much as I do"
```

-----
### homeurl
You or your organization's website. 

**Input options** (*string*): valid url

**Commands:**
```
$ qri config get profile.homeurl

$ qri config set profile.homeurl https://harrypotterlover.com
```

-----
### color
The theme color your prefer when viewing Qri using the webapp. This will expand, but for now the only option is 'default'

**Input options** (*string*): `default`

**Commands:**
```
$ qri config get profile.color

$ qri config set profile.color default
```

-----
### thumb
*thumbnail photo*
Your thumbnail photo is auto generated using the profile photo uploaded. We recommend not setting this yourself.

-----
### profile photo
Upload a profile photo using a filepath, url, or ipfs hash. This photo is used on the Qri webapp.

**Input options** (*string*): valid url, valid filepath, or valid ipfs hash

**Commands:**
```
$ qri config get profile.profile

$ qri config set profile.profile ~/Documents/pictures/headshot.jpeg
```

-----
### poster photo
Upload a poster photo (the backdrop to your profile). This photo is used on the Qri webapp.

**Input options** (*string*): valid url, valid filepath, or valid ipfs hash

**Commands:**
```
$ qri config get profile.poster

$ qri config set profile.poster http://www.imgur.com/pic_of_sunset_i_took_one_time.jpeg
```

-----
### twitter
*twitter handle*
You or your organization's twitter handle. No need to include the `@` symbol. 

**Input options** (*string*): valid twitter handle (max length 15)

**Commands:**
```
$ qri config get profile.twitter

$ qri config set profile.twitter lunalovegood7
```
