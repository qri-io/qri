<a name="v0.10.0"></a>
# [v0.10.0](https://github.com/qri-io/qri/compare/v0.9.13...v) (2021-05-04)

Welcome to the long awaited `0.10.0` Qri release! We've focused on usability and bug fixes, specifically surrounding massive improvements to saving a dataset, the  HTTP API, and the `lib` package interface. We've got a few new features (step-based transform execution, change reports over the api, progress bars on save, and a new component: Stats) and you should see an obvious change based on the speed, reliability, and usability in Qri, especially when saving a new version of a dataset.

## Massive Improvements to Save performance
We've drastically improved the reliability and scalability of saving a dataset on Qri. Qri uses a bounded block of memory while saving, meaning it will only consume roughly a MAX of 150MB of memory while saving, regardless of how large your dataset is. This means the max size of dataset you can save is no longer tied to your available memory.

We've had to change some underlying functionality to get the scalability we want, to that end we no longer calculate `Structure.Checksum`, we no longer calculate commit messages for datasets over a certain size, and we no longer store all the error values found when validating the body of a dataset.

## API Overhaul
Our biggest change has been a complete overhaul of our API.

We wanted to make our API easier to work with by making it more consistent across endpoints. After a great deal of review & discussion, this overhaul introduces an RPC-style centric API that expects JSON `POST` requests, plus a few `GET` requests we're calling "sugar" endpoints.

The RPC part of our api is an HTTP pass-through to our lib methods. This makes working with qri over HTTP the _same_ as working with Qri as a library. We've spent a lot of time building & organizing qri's `lib` interface, and now all of that same functionality is exposed over HTTP. The intended audience for the RPC API are folks who want to automate qri across process boundaries, and still have very fine grained control. Think "command line over HTTP".

At the same time, however, we didn't want to lose a number of important-to-have endpoints, like being able to `GET` a dataset body via just a URL string, so we've moved all of these into a "sugar" API, and made lots of room to grow. We'll continue to add convenience-oriented endpoints that make it easy to work with Qri. The "sugar" API will be oriented to users who are prioritizing fetching data from Qri to use elsewhere.

We also noticed how quickly our open api spec fell out of date, so we decided to start generating our spec using the code itself. Take a look at our [open api spec](https://github.com/qri-io/qri/blob/master/api/open_api_3.yaml), for a full list of supported JSON endpoints.

Here is our full API spec, supported in this release:

## API Spec

### Sugar

The purpose of the API package is to expose the lib.RPC api *and* add syntatic sugar for mapping RESTful HTTP requests to lib method calls

| endpoint                                                       | HTTP methods    | Lib Method Name                  |
| -------------------------------------------------------------- | --------------- | -------------------------------- |
| "/"                                                            | GET             | api.HealthCheckHandler           |
| "/health"                                                      | GET             | api.HealthCheckHandler           |
| "/qfs/ipfs/{path:.*}"                                          | GET             | qfs.Get                          |
| "/webui"                                                       | GET             | api.WebuiHandler                 |
| /ds/get/{username}/{name}                                      | GET             | api.GetHandler                   |
| /ds/get/{username}/{name}/at/{path}                            | GET             | api.GetHandler                   |
| /ds/get/{username}/{name}/at/{path}/{component}                | GET             | api.GetHandler                   |
| /ds/get/{username}/{name}/at/{path}/body.csv                   | GET             | api.GetHandler                   |


### RPC

The purpose of the lib package is to expose a uniform interface for interacting with a qri instance

| endpoint                                 | Return Type         | Lib Method Name                  |
| ---------------------------------------- | ------------------- | -------------------------------- |
|                                          |                     |                                  |
| Aggregate Endpoints                      |                     |                                  |
| "/list"                                  | []VersionInfo       | collection.List?                 |
| "/sql"                                   | [][]any             | sql.Exec                         |
| "/diff"                                  | Diff                | diff.Diff                        |
| "/changes"                               | ChangeReport        | diff.Changes                     |
|                                          |                     |                                  |
| Access Endpoints                         |                     |                                  |
| "/access/token"                          | JSON Web Token      | access.Token                     |
|                                          |                     |                                  |
| Automation Endpoints                     |                     |                                  |
| "/auto/apply"                            | ApplyResult         | automation.apply                 |
|                                          |                     |                                  |
| Dataset Endpoints                        |                     |                                  |
| "/ds/componentstatus"                    | []Status            | dataset.ComponentStatus          |
| "/ds/get                                 | GetResult           | dataset.Get                      |
| "/ds/activity"                           | []VersionInfo       | dataset.History                  |
| "/ds/rename"                             | VersionInfo         | dataset.Rename                   |
| "/ds/save"                               | dataset.Dataset     | dataset.Save                     |
| "/ds/pull"                               | dataset.Dataset     | dataset.Pull                     |
| "/ds/push"                               | DSRef               | dataset.Push                     |
| "/ds/render"                             | []byte              | dataset.Render                   |
| "/ds/remove"                             | RemoveResponse      | dataset.Remove                   |
| "/ds/validate"                           | ValidateRes         | dataset.Validate                 |
| "/ds/unpack"                             | Dataset             | dataset.Unpack                   |
| "/ds/manifest"                           | Manifest            | dataset.Manifest                 |
| "/ds/manifestmissing"                    | Manifest            | dataset.ManifestMissing          |
| "/ds/daginfo"                            | DagInfo             | dataset.DagInfo	            |
|                                          |                     |                                  |
| Peer Endpoints                           |                     |                                  |
| "/peer"                                  | Profile             | peer.Info                        |
| "/peer/connect"                          | Profile             | peer.Connect                     |
| "/peer/disconnect"                       | Profile             | peer.Disconnect                  |
| "/peer/list"                             | []Profile           | peer.Profiles                    |
|                                          |                     |                                  |
| Profile Endpoints                        |                     |                                  |
| "/profile"                               | Profile             | profile.GetProfile               |
| "/profile/set"                           | Profile             | profile.SetProfile               |
| "/profile/photo"                         | Profile             | profile.ProfilePhoto             |
| "/profile/poster"                        | Profile             | profile.PosterPhoto              |
|                                          |                     |                                  |
| Remote Endpoints                         |                     |                                  |
| "/remote/feeds"                          | Feed                | remote.Feeds                     |
| "/remote/preview"                        | Dataset             | remote.Preview                   |
| "/remote/remove"                         |  -                  | remote.Remove                    |
| "/remote/registry/profile/new"           | Profile             | registry.CreateProfile           |
| "/remote/registry/profile/prove"         | Profile             | registry.ProveProfile            |
| "/remote/search"                         | SearchResult        | remote.Search                    |
|                                          |                     |                                  |
|                                          |                     |                                  |
| Working Directory Endpoints              |                     |                                  |
| "/wd/status"                             | []StatusItem        | fsi.Status                       |
| "/wd/init"                               | DSRef               | fsi.Init                         |
| "/wd/caninitworkdir"                     | --                  | fsi.CanInitworkdir               |
| "/wd/checkout"                           | --                  | fsi.Checkout                     |
| "/wd/restore"                            | --                  | fsi.Restore                      |
| "/wd/write"                              | []StatusItem        | fsi.Write                        |
| "/wd/createlink"                         | VersionInfo         | fsi.CreateLink                   |
| "/wd/unlink"                             | string              | fsi.Unlink                       |
| "/wd/ensureref"                          | --                  | fsi.EnsureRefNotLinked           |
|                                          |                     |                                  |

## Redesigned lib interface
In general, we've streamlined the core functionality and reconciled input params in the `lib` package (which contain the methods and params that power both the api and cmd), so that no matter how you are accessing functionality in qri, whether using `lib` as a package,  using the HTTP API, or using the command line, you can expect consistent inputs and–more importantly–consistent behavior. We're also utilizing our new `dispatch` pattern to replace our old rpc client with the _same_ JSON HTTP API exposed to users. That way all our API's, HTTP or otherwise have the same expectations. If something is broke in one place, it is broken in all places, consequently, when it is fixed in one place it will be fixed in all places!

These changes will also help us in our upcoming challenges to refine expand the notion of identity inside of Qri.

## Stats Component!
We've added a new component: Stats! Stats is a component that contains statistical metadata about the body of a dataset. Stats are now automatically calculated and saved with each new version.

Its purpose is to provide an "at a glance" summary of a dataset, calculating statistics on columns (ok, on "compound data types", but it's much easier to think about column stats). In order to remain fast for very large dataset sizes, we have opted to calculate the stats using probabilistic structures. Stats are an important part of change reports, and allow you to get a sense of what is different in a dataset body without having to examine that body line by line. 

For earlier versions that don't have stats calculated, we've added a `qri stats` command that will calculate the new stats component for you.

## Other features:
### Change report over API
We've added an endpoint to get the change report via the api: `/changes`, laying the groundwork for future User Interfaces that can report on changes between versions.

### Access and Oauth
We're working towards making qri-core able to handle multiple users on the same node (AKA multi-tenancy). In preparation for multi-tenancy, we are adding support for generating JSON web tokens (JWTs) that will help ensure identity on the network. You can generate an access token via the cmd using the `qri access` command, or over the api using the `/access/token` endpoint.

### Apply Command
You can now use the `qri apply` command to "dry run" and de-bug transforms! Use the `--file` flag to apply a new `transform.star` file, use a dataset reference to re-run an already existing transform on an already existing dataset, or use both to apply a new tranform to an already existing dataset. The resulting dataset is output to the terminal. To run the transform and save the results as a new commit, use the `--apply` flag in the `qri save` command.

### Progress bars
Qri has been fine tuned to handle larger datasets faster. Regardless of how fast your dataset is saving, we want to be able to track any progress. Now, when saving a new dataset or a new version of a dataset, the command line will show you a progress bar.

### `qri version`
To help with debugging, the `qri version` command now comes chock full of details, including the qri version, the build time, and the git summary. (developers: you need to use `make build` now instead of `go build`!)

### dependancy updates:
We've also released updated to our dependencies: [dag](https://github.com/qri-io/dag/releases/tag/v0.2.2), [qfs](https://github.com/qri-io/qfs/releases/tag/v0.6.0), [deepdiff](https://github.com/qri-io/deepdiff/releases/tag/v0.2.1), [dataset](https://github.com/qri-io/dataset/releases/tag/v0.3.0). Take a look at those release notes to learn about other bug fixes and stability enhancements that Qri is inheriting.

### BREAKING CHANGES
* **api:** complete API overhaul. Check out the full api spec for all changes(https://github.com/qri-io/qri/issues/1731)

### Bug Fixes

* **api:** `handleRefRoutes` should refer to `username` rather than ([644f706](https://github.com/qri-io/qri/commit/644f706))
* **api:** Allow OPTIONS header so that CORS is available ([e067500](https://github.com/qri-io/qri/commit/e067500))
* **api:** denyRPC only affects RPC, HTTP can still be used ([b6298ce](https://github.com/qri-io/qri/commit/b6298ce))
* **api:** Fix unmarshal bug in api test ([0234f30](https://github.com/qri-io/qri/commit/0234f30))
* **api:** fix vet error ([afbe53e](https://github.com/qri-io/qri/commit/afbe53e))
* **api:** handle OPTIONS requests on refRoute handlers ([395d5ae](https://github.com/qri-io/qri/commit/395d5ae))
* **api:** health & root endpoints use middleware, which handles OPTIONS ([5f421eb](https://github.com/qri-io/qri/commit/5f421eb))
* **apply:** bad API endpoint for apply over HTTP ([c5cc840](https://github.com/qri-io/qri/commit/c5cc840))
* **base.ListDatasets:** support -1 limit to list all datasets ([bd2f831](https://github.com/qri-io/qri/commit/bd2f831))
* **base.SaveDataset:** move logbook write operation back down from lib ([a00f1b8](https://github.com/qri-io/qri/commit/a00f1b8))
* **changes:** "left" side of report should be the previous path ([32b46ff](https://github.com/qri-io/qri/commit/32b46ff))
* **changes:** column renames are properly handled now ([c306e41](https://github.com/qri-io/qri/commit/c306e41))
* **cmd:** nil pointer dereference in `PrintProgressBarsOnEvents` ([b0a2ec0](https://github.com/qri-io/qri/commit/b0a2ec0))
* **dispatch:** Add default source resolver to Attributes ([e074bf4](https://github.com/qri-io/qri/commit/e074bf4))
* **dispatch:** Calls that return only 1 value can work across RPC ([ceefeaa](https://github.com/qri-io/qri/commit/ceefeaa))
* **dispatch:** Comments clarifying Methods and Attributes ([33222c6](https://github.com/qri-io/qri/commit/33222c6))
* **dispatch:** Dispatch for transform. Transformer instead of Service ([24562c5](https://github.com/qri-io/qri/commit/24562c5))
* **dispatch:** Fix code style and get tests passing ([4cdb3f5](https://github.com/qri-io/qri/commit/4cdb3f5))
* **dispatch:** Fix for fsi plumbing commands, to work over http ([8070537](https://github.com/qri-io/qri/commit/8070537))
* **dispatch:** MethodSet interface to get name for dispatch ([9b73d4b](https://github.com/qri-io/qri/commit/9b73d4b))
* **dispatch:** send source over wire and cleanup attr definitions ([8a2ddf8](https://github.com/qri-io/qri/commit/8a2ddf8))
* **dispatch:** Use dispatcher interface for DatasetMethods ([201edda](https://github.com/qri-io/qri/commit/201edda))
* **dispatch:** When registering, compare methods to impl ([c004812](https://github.com/qri-io/qri/commit/c004812))
* **dsfs:** fix adjustments to meta prior to commit message generation ([365cbb9](https://github.com/qri-io/qri/commit/365cbb9))
* **dsfs:** LoadDataset using the mux filesystem. Error if nil ([75215be](https://github.com/qri-io/qri/commit/75215be))
* **dsfs:** remove dataset field computing deadlock ([ef615a9](https://github.com/qri-io/qri/commit/ef615a9))
* **dsfs:** set script paths before generating commit messages ([8174996](https://github.com/qri-io/qri/commit/8174996))
* **fill:** Map keys are case-insensitive, handle maps recursively ([ab27f1b](https://github.com/qri-io/qri/commit/ab27f1b))
* **fsi:** fsi.ReadDir sets the '/fsi' path prefix ([3d64468](https://github.com/qri-io/qri/commit/3d64468))
* **http:** expect json requests to decode, if the body is not empty ([c50ea28](https://github.com/qri-io/qri/commit/c50ea28))
* **http:** fix httpClient error checking ([6f4567e](https://github.com/qri-io/qri/commit/6f4567e))
* **init:** TargetDir for init, can be absolute, is created if needed ([55c9ff6](https://github.com/qri-io/qri/commit/55c9ff6))
* **key:** fix local keystore key.ID encoding, require ID match keys ([a469b6e](https://github.com/qri-io/qri/commit/a469b6e))
* **lib:** Align input params across all lib methods ([7ba26b6](https://github.com/qri-io/qri/commit/7ba26b6))
* **lib:** don't ignore serialization errors when getting full datasets ([64abb7e](https://github.com/qri-io/qri/commit/64abb7e))
* **lib:** Improve context passing and visibility of internal structs ([8f6509b](https://github.com/qri-io/qri/commit/8f6509b))
* **list:** List datasets even if some refs have bad profileIDs ([11b6763](https://github.com/qri-io/qri/commit/11b6763))
* **load:** Fix spec test to exercise LoadDataset ([f978ec7](https://github.com/qri-io/qri/commit/f978ec7))
* **logbook:** commit timestamps overwrite run timestamps in logs ([2ab44d0](https://github.com/qri-io/qri/commit/2ab44d0))
* **logbook:** Logsync validates that ref has correct profileID ([153c4b9](https://github.com/qri-io/qri/commit/153c4b9))
* **logbook:** remove 'stranded' log histories on new dataset creation ([2412a40](https://github.com/qri-io/qri/commit/2412a40))
* **mux:** allow api mux override ([eccdf9b](https://github.com/qri-io/qri/commit/eccdf9b))
* **oas:** add open api spec tests to CI and makefile ([e91b50b](https://github.com/qri-io/qri/commit/e91b50b))
* **p2p:** dag `MissingManifest` sigfaults if there is a nil manifest ([04ec5b2](https://github.com/qri-io/qri/commit/04ec5b2))
* **p2p:** qri bootstrap addrs config migration ([0680097](https://github.com/qri-io/qri/commit/0680097))
* **prove:** Prove command updates local logbook as needed ([b09a2eb](https://github.com/qri-io/qri/commit/b09a2eb))
* **prove:** Store the original KeyID on config.profile ([88214a4](https://github.com/qri-io/qri/commit/88214a4))
* **pull:** Pull uses network resolver. Fixes integration test. ([d049958](https://github.com/qri-io/qri/commit/d049958))
* **remote:** always send progress completion on client push/pull events ([afcb2f8](https://github.com/qri-io/qri/commit/afcb2f8))
* **repo:** Don't use blank path for new repo in tests ([1ec8e74](https://github.com/qri-io/qri/commit/1ec8e74))
* **routes:** skip over endpoints that are DenyHTTP ([ee7d882](https://github.com/qri-io/qri/commit/ee7d882))
* **rpc:** unregister dataset methods ([6a2b213](https://github.com/qri-io/qri/commit/6a2b213))
* **run:** fix unixnano -> *time.Time conversion, clean up transform logging ([c903657](https://github.com/qri-io/qri/commit/c903657))
* **save:** Remove dry-run, recall, return-body from save path ([fac37da](https://github.com/qri-io/qri/commit/fac37da))
* **search:** Dispatch for search ([8570abf](https://github.com/qri-io/qri/commit/8570abf))
* **sql:** Fix sql command for arm build ([#1783](https://github.com/qri-io/qri/issues/1783)) ([2ae1541](https://github.com/qri-io/qri/commit/2ae1541))
* **sql:** Fix sql command for other 32-bit platforms ([704d9fb](https://github.com/qri-io/qri/commit/704d9fb))
* **startf/ds.set_body:** infer structure when ds.set_body is called ([a8a3492](https://github.com/qri-io/qri/commit/a8a3492))
* **stats:** close accumulator to finalize output ([1b7f4f1](https://github.com/qri-io/qri/commit/1b7f4f1))
* **test:** fix api tests to consume refstr ([e03ef3a](https://github.com/qri-io/qri/commit/e03ef3a))
* **token:** `claim` now includes `ProfileID` ([8dd40e4](https://github.com/qri-io/qri/commit/8dd40e4))
* **transform:** don't duplicate transform steps on save ([8ace963](https://github.com/qri-io/qri/commit/8ace963))
* **transform:** don't write to out streams when nil, use updated preview.Create ([28651cb](https://github.com/qri-io/qri/commit/28651cb))
* **version:** add warning when built with 'go install' ([1063a71](https://github.com/qri-io/qri/commit/1063a71))


### Features

* **api:** change report API ([ca16f3c](https://github.com/qri-io/qri/commit/ca16f3c))
* **api:** GiveAPIServer attachs all routes to api ([ea2d4ad](https://github.com/qri-io/qri/commit/ea2d4ad))
* **api:** read oauth tokens to request context ([f024b26](https://github.com/qri-io/qri/commit/f024b26))
* **apply:** Apply command, and --apply flag for save ([c01a4bf](https://github.com/qri-io/qri/commit/c01a4bf))
* **bus:** Subscribe to all, or by ID. "Type" -> "Topic" ([cc139bc](https://github.com/qri-io/qri/commit/cc139bc))
* **cmd:** add access command ([1c56680](https://github.com/qri-io/qri/commit/1c56680))
* **dispatch:** Abs paths on inputs to dispatch methods ([a19efcc](https://github.com/qri-io/qri/commit/a19efcc))
* **dispatch:** Dispatch func can return 1-3 values, 2 being Cursor ([304f7c5](https://github.com/qri-io/qri/commit/304f7c5))
* **dispatch:** Method attributes contain http endpoint and verb ([0036cb7](https://github.com/qri-io/qri/commit/0036cb7))
* **dsfs:** compute & store stats component at save time ([3ff3b75](https://github.com/qri-io/qri/commit/3ff3b75))
* **httpClient:** introducing an httpClient ([#1629](https://github.com/qri-io/qri/issues/1629)) ([8ecde53](https://github.com/qri-io/qri/commit/8ecde53))
* **keystore:** keystore implmenetation ([#1602](https://github.com/qri-io/qri/issues/1602)) ([205165a](https://github.com/qri-io/qri/commit/205165a))
* **lib:** attach active user to scope ([608540a](https://github.com/qri-io/qri/commit/608540a))
* **lib:** Create Auth tokens using inst.Access() ([3be7af2](https://github.com/qri-io/qri/commit/3be7af2))
* **lib:** Dispatch methods call, used by FSI ([afaf06d](https://github.com/qri-io/qri/commit/afaf06d))
* **list:** Add ProfileID restriction option to List ([774fa06](https://github.com/qri-io/qri/commit/774fa06))
* **logbook:** add methods for writing transform run ops ([7d0cb91](https://github.com/qri-io/qri/commit/7d0cb91))
* **profile:** ResolveProfile replaces CanonicalizeProfile ([7bd848b](https://github.com/qri-io/qri/commit/7bd848b))
* **prove:** Prove a new keypair for an account, set original profileID ([6effbea](https://github.com/qri-io/qri/commit/6effbea))
* **run:** run package defines state of a transform run ([8e69e5e](https://github.com/qri-io/qri/commit/8e69e5e))
* **save:** emit save events. Print progress bars on save ([3c979ed](https://github.com/qri-io/qri/commit/3c979ed))
* **save:** recall transform on empty --apply, write run operations ([3949be9](https://github.com/qri-io/qri/commit/3949be9))
* **save:** support custom timestamps on commit ([e8c18fa](https://github.com/qri-io/qri/commit/e8c18fa))
* **sql:** Disable sql command on 32-bit arm to fix compilation ([190b5cb](https://github.com/qri-io/qri/commit/190b5cb))
* **stats:** overhaul stats service interface, implement os stats cache ([2128a0c](https://github.com/qri-io/qri/commit/2128a0c))
* **stats:** stats based on sketch/probabalistic data structures ([f6191c8](https://github.com/qri-io/qri/commit/f6191c8))
* **transform:** Add runID to transform. Publish some events. ([5ceac77](https://github.com/qri-io/qri/commit/5ceac77))
* **transform:** emit DatasetPreview event after startf transform step ([28bb8b0](https://github.com/qri-io/qri/commit/28bb8b0))
* **validate:** Parameters to methods will Validate automatically ([7bb1515](https://github.com/qri-io/qri/commit/7bb1515))
* **version:** add details reported by "qri version" ([e6a0a67](https://github.com/qri-io/qri/commit/e6a0a67))
* **vesrion:** add json format output for version command ([ad4dcc7](https://github.com/qri-io/qri/commit/ad4dcc7))
* **websocket:** publish dataset save events to websocket event connections ([378e922](https://github.com/qri-io/qri/commit/378e922))


### Performance Improvements

* **dsfs:** don't calculate commit descriptions if title and message are set ([f5ec420](https://github.com/qri-io/qri/commit/f5ec420))
* **save:** improve save performance, using bounded memory ([7699f02](https://github.com/qri-io/qri/commit/7699f02))



<a name="v0.9.13"></a>
# [v0.9.13](https://github.com/qri-io/qri/compare/v0.9.12...v0.9.13) (2020-10-12)

Patch v0.9.13 brings improvments to the `validate` command, and lays the groundwork for OAuth within qri core.

### Better Validate command output
Here's a [demo](https://asciinema.org/a/360495)! `qri validate` gets a little smarter this release, printing a cleaner, more readable list of human errors, and now has flags to output validation error data in JSON and CSV formats.

### Bug Fixes

* **`Validate`:** allow validation of FSI dataset w/ no history ([a212dba](https://github.com/qri-io/qri/commit/a212dba499ee35b02c5a21509487a9787e74de9b))
* **api:** Fix api test for health check by ignoring version number ([f0b7518](https://github.com/qri-io/qri/commit/f0b751823e00cba4353b877f13eec62a4335a7ee))
* **api:** Get with body.csv suffix implies all=true ([ed95ce1](https://github.com/qri-io/qri/commit/ed95ce1f615f2e4b7b798af2ea6f425297290520))
* **connect:** only run `DoSetup` if there is not an existing repo ([3642aef](https://github.com/qri-io/qri/commit/3642aefad2a0000ae5d1c90c45ad363c53f2db98)), closes [#1553](https://github.com/qri-io/qri/issues/1553)
* **event bus:** encoded fields use lower-case names in JSON ([a14f28f](https://github.com/qri-io/qri/commit/a14f28f5647b54041c2790a55d77e06125a08d90))
* **lib:** Wrap returned errors in lib/datasets ([64b2890](https://github.com/qri-io/qri/commit/64b28900657fe74e9733b569667c0adc26a950d0))
* **remote:** be more lenient with `ResolveRef` error when trying to remove a dataset ([7ecbc83](https://github.com/qri-io/qri/commit/7ecbc8350f198bb349f704fb92e4c367f120969c))
* **sql:** Truncate unexpected columns during join ([72ec2c4](https://github.com/qri-io/qri/commit/72ec2c4d92024e1d4822ff500cf2b44d610b4f9c))
* **validate:** Don't swallow details about validate body read errors ([e5fdd8e](https://github.com/qri-io/qri/commit/e5fdd8e97a8ac4861aa856a9da08114134370be2))
* **watchfs:** `WatchAllFSIPaths` needs to call `watchPaths` ([58acace](https://github.com/qri-io/qri/commit/58acacef824679b0a36f87093e2a15bb04213a76)), closes [#1554](https://github.com/qri-io/qri/issues/1554)


### Features

* **access:** token creation now supports arbitrary claims ([6d1c4a0](https://github.com/qri-io/qri/commit/6d1c4a0664ac316d570ad0691c545366765d21ee))
* **access:** TokenSource & TokenStore interfaces, implementations ([2006c4f](https://github.com/qri-io/qri/commit/2006c4fbc439baae948b8f8849cb119d566d9674))
* **progress:** enable websocket push pull events ([47bb395](https://github.com/qri-io/qri/commit/47bb395155236573ed8013d3758059dee87d3c37))
* **validate:** tabular output of validation errors on CLI, configure out format ([cee5919](https://github.com/qri-io/qri/commit/cee59191a396c0d6330f56298b39ece403a1ebe5))



# [v0.9.12](https://github.com/qri-io/qri/compare/v0.9.11...v) (2020-09-10)

Patch release 0.9.12 features a number of fixes to various qri features, most aimed at improving general quality-of-life of the tool, and some others that lay the groundwork for future changes.

## HTTP API Changes
Changed the qri api so that the `/get` endpoint gets dataset heads and bodies. `/body` still exists but is now deprecated.

## P2P and Collaboration
A new way to resolve peers and references on the p2p network.
The start of access control added to our remote communication API.
Remotes serve a simple web ui.

## General polish
Fix ref resolution with divergent logbook user data.
Working directories allow case-insensitive filenames.
Improve sql support so that dataset names don't need an explicit table alias.
The `get` command can fetch datasets from cloud.

### Bug Fixes

* **api:** Accept header can be used to download just body bytes ([ed7aaf4](https://github.com/qri-io/qri/commit/ed7aaf474f74f090f432cdc0bae32bfa03877af8))
* **api:** Additional comments for api/datasets functions ([0883afa](https://github.com/qri-io/qri/commit/0883afad1b8addc09518dcc3af29d850d93159be))
* **api:** Preview endpoint can handle IPFS paths ([c404756](https://github.com/qri-io/qri/commit/c40475675786d7a830ef6ab8e1ce877b6b870b4c))
* **cmd:** fix context passing error in test runner ([f76de25](https://github.com/qri-io/qri/commit/f76de25603a9fd616f1cc083663efd1782188d89))
* **fsi:** component filenames are case-insensitive ([9cc6f0e](https://github.com/qri-io/qri/commit/9cc6f0ef44ebda74910380461f517ac89b33e6fb)), closes [#1509](https://github.com/qri-io/qri/issues/1509)
* **get:** get command auto-fetches ([db88c6c](https://github.com/qri-io/qri/commit/db88c6c2911bb385bc5a7443e38b34be1bc1f195))
* **lib:** `oldRemoteClientExisted` causing waitgroup panic ([e2e23fb](https://github.com/qri-io/qri/commit/e2e23fb4b87ea20e2cb445a552b9330eae526812))
* **lib:** bring the node `Offline` when shutting down the instance ([12a9614](https://github.com/qri-io/qri/commit/12a9614fa76ea42da8608e63ddeb989d6b1d2361))
* **lib:** explicitly shutdown `remoteClient` on `inst.Shutdown()` ([8154dc3](https://github.com/qri-io/qri/commit/8154dc3c323919dbb0e3bcbccfc426e5862b82d0))
* **logbook:** Switch merge to use ProfileID, not UserCreateID ([e5c39d8](https://github.com/qri-io/qri/commit/e5c39d849bae9f14a82bd097a3d28bbd8bb4b0ce))
* **remote:** Remote MockClient adds IPFS blocks ([267a073](https://github.com/qri-io/qri/commit/267a073c6138d47380920f2d3ba464be05e54e45))
* **resolve:** Resolve refs after merging logbook by profileID ([be76f29](https://github.com/qri-io/qri/commit/be76f29698add4dc68accf3c713ad44eba5e64f7))


### Features

* **access:** access defines a policy grammer for access control ([63172c7](https://github.com/qri-io/qri/commit/63172c779a314215bdcb4a10df5f6dbd4cc8e11d))
* **api:** qri can serve a simple webui ([3db8daa](https://github.com/qri-io/qri/commit/3db8daa45383c0b15a81badbb6d23b55e9b2f489))
* **lib:** pass remote options when creating `NewInstance` ([d2b8b8f](https://github.com/qri-io/qri/commit/d2b8b8fb45de0003808a43035d91db410fddd877))
* **pull:** set pulling source with remote flag on CLI & API ([0dc9890](https://github.com/qri-io/qri/commit/0dc9890b53f31bb6043312ae34e95a6f632335ce))



<a name="v0.9.11"></a>
# [v0.9.11](https://github.com/qri-io/qri/compare/v0.9.10...v0.9.11) (2020-08-10)

This patch release addresses a critical error in `qri setup`, and removes overly-verbose output when running `qri connect`.

### Bug Fixes

* **lib:** don't panic when resolving without a registry ([2708801](https://github.com/qri-io/qri/commit/2708801))
* **p2p:** some clean up around `qri peers connect` and `upgradeToQriConnection` ([#1489](https://github.com/qri-io/qri/issues/1489)) ([b7bb076](https://github.com/qri-io/qri/commit/b7bb076))
* **setup:** Fix prompt, add a test for --anonymous ([3b2c58a](https://github.com/qri-io/qri/commit/3b2c58a))
* **setup:** Fix setup, add many unit tests ([b88d084](https://github.com/qri-io/qri/commit/b88d084))



<a name="v0.9.10"></a>
# [v0.9.10](https://github.com/qri-io/qri/compare/v0.9.9...v0.9.10) (2020-07-27)

For this release we focused on clarity, reliability, major fixes, and communication (both between qri and the user, and the different working components of qri as well). The bulk of the changes surround the rename of `publish` and `add` to `push` and `pull`, as well as making the commands more reliable, flexible, and transparent.

### `push` is the new `publish` 

Although qri defaults to publishing datasets to our [qri.cloud](https://qri.cloud) website (if you haven't checked it out recently, it's gone through a major facelift & has new features like dataset issues and vastly improved search!), we still give users tools to create their own services that can host data for others. We call these _remotes_ (qri.cloud is technically a very large, very reliable remote). However, we needed a better way to keep track of where a dataset has been "published", and also allow datasets to be published to different locations. 

We weren't able to correctly convey, "hey this dataset has been published to remote A but not remote B", by using a simple boolean published/unpublished paradigm. We also are working toward a system, where you can push to a _peer_ remote or make your dataset _private_ even though it has been sent to live at a public location.

In all these cases, the name `publish` wasn't cutting it, and was confusing users. 

After debating a few new titles in [RFC0030](https://github.com/qri-io/rfcs/blob/master/text/0030-replace_publish_clone_with_push_pull.md), we settled on `push`. It properly conveys what is happening: you are pushing the dataset from your node to a location that will accept and store it. Qri keeps track of where it has been pushed, so it can be pushed to multiple locations.

It also helps that `git` has a `push` command, that fulfills a similar function in software version control, so using the verb `push` in this way has precident. We've also clarified the command help text: only one version of a dataset is pushed at a time.

### `pull` is the new `add`
We decided that, for clarity, if we are renaming `qri publish ` to `qri pull`, we should rename it's mirrored action, `qri add` to `qri pull`. Now it's clear: to send a dataset to another source use `qri push`, to get a dataset from another source use `qri pull`!

### use `get` instead of `export`
`qri export` has been removed. Use `qri get --format zip me/my_dataset` instead. We want more folks to play with `get`, it's a far more powerful version of export, and we had too many folks miss out on `get` because they found `export` first, and it didn't meet their expectations.

### major fix: pushing & pulling historical versions
`qri push` without a specified version will still default to pushing the _latest version_ and `qri pull` without a specified version will still default to pulling _every version_ of the dataset that is available. However, we've added the ability to push or pull a dataset at _specific versions_ by specifying the dataset version's _path_! You can see a list of a dataset's versions and each version's path by using the `qri log` command.

In the past this would error:

```
$ qri publish me/dataset@/ipfs/SpecificVersion
```

With the new push command, this will now work:

```
$ qri push me/dataset@/ipfs/SpecificVersion
```

You can use this to push old versions to a remote, same with `pull`!

### events, websockets & progress
We needed a better way for the different internal qri processes to coordinate. So we beefed up our events and piped the stream of events to a websocket. Now, one qri process can subscribe and get notified about important events that occur in another process. This is also great for users because we can use those events to communicate more information when resource intensive or time consuming actions are running! Check our our progress bars when you `push` and `pull`!

The websocket event API is still a work in progress, but it's a great way to build dynamic functionality on top of qri, using the same events qri uses internally to power things like progress bars and inter-subsystem communication.

### other important changes

- sql now properly handles dashes in dataset names
- migrations now work on machines across multiple mounts. We fixed a bug that was causing the migration to fail. This was most prevalent on Linux.
- the global `--no-prompt` flag will disable all interactive prompts, but now falls back on defaults for each interaction.
- a global `--migrate` flag will auto-run a migration check before continuing with the given command
- the default when we ask the user to run a migration is now `"No"`. In order to auto-run a migration you need the `--migrate` flag, (_not_ the `--no-prompt` flag, but they can both be use together for "run all migrations and don't bother me")
- the `remove` now takes the duties of the `--unpublish` flag. run `qri remove --all --remote=registry me/dataset` instead of `qri publish --unpublish me/dataset`. More verbose? Yes. But you're deleting stuff, so it should be a think-before-you-hit-enter type thing.
- We've made some breaking changes to our API, they're listed below in the YELLY CAPS TEXT below detailing breaking changes

### Bug Fixes

* **cmd:** re-work migration execution ([3bd14f5](https://github.com/qri-io/qri/commit/3bd14f5))
* **docker:** qri must be built from go 1.14 or higher ([99b267e](https://github.com/qri-io/qri/commit/99b267e))
* **json:** Fix typo in json serialization for dataset listing ([151fcb1](https://github.com/qri-io/qri/commit/151fcb1))
* **linux:** migration fix for copying on cross link device ([cd47426](https://github.com/qri-io/qri/commit/cd47426))
* **logbook:** write timestamps on push & delete operations ([079c759](https://github.com/qri-io/qri/commit/079c759))
* **logsync:** remove errorneous error check on logsync delete reqs ([1670305](https://github.com/qri-io/qri/commit/1670305))
* **print:** Write directory to stdout if not using a terminal ([3575349](https://github.com/qri-io/qri/commit/3575349))
* **sql:** properly handle dashes in dataset names ([dbe472c](https://github.com/qri-io/qri/commit/dbe472c))
* **windows:** Fix Windows test build by removing file permissions ([b020767](https://github.com/qri-io/qri/commit/b020767))
* **zip:** Add get format=zip to the root handler ([6c0a00d](https://github.com/qri-io/qri/commit/6c0a00d))


### Code Refactoring

* **api:** watchfs sends over event bus, websocket API is an event stream ([3cc9949](https://github.com/qri-io/qri/commit/3cc9949))
* **clone:** rename clone to pull, use dsrefs in remote, peerIDs in ResolveRef ([51f432d](https://github.com/qri-io/qri/commit/51f432d))


### Features

* **dsref:** add Complete method on dsref.Ref ([6bffa72](https://github.com/qri-io/qri/commit/6bffa72))
* **event:** add p2p events to `events` and event bus to `QriNode` ([#1440](https://github.com/qri-io/qri/issues/1440)) ([c444288](https://github.com/qri-io/qri/commit/c444288))
* **lib:** `OptNoBootstrap` option func for `NewInstance` ([#1436](https://github.com/qri-io/qri/issues/1436)) ([9530c62](https://github.com/qri-io/qri/commit/9530c62))
* **lib:** add OptEventHandler for subscribing to events move WS to instance ([889c175](https://github.com/qri-io/qri/commit/889c175))
* **preview:** preview subcommand ([3303428](https://github.com/qri-io/qri/commit/3303428))
* **push, remove:** publish -> push rename, remove --remote flag, better docs ([c70c04d](https://github.com/qri-io/qri/commit/c70c04d))
* **remote:** redo remote client interface, write client events, progress bars ([01b88fc](https://github.com/qri-io/qri/commit/01b88fc))


### BREAKING CHANGES

* **push, remove:** * HTTP API: /publish endpoint is now /push
* HTTP API: /unpublish is now /remove?remote=registry
* HTTP API: /add is now /pull
* **clone:** * all remote hooks pass dsref.Ref instead of reporef.DatasetRef arguments
* ResolveRef spec now requires public key ID be set as part of reference resolution
* **api:** websocket JSON messages have changed. They're now _always_ an object with "type" and
"data" keys.



<a name="v0.9.9"></a>
# [v0.9.9](https://github.com/qri-io/qri/compare/v0.9.8...v0.9.9) (2020-07-01)

Welcome to Qri 0.9.9! We've got a lot of internal changes that speed up the work you do on Qri everyday, as well as a bunch of new features, and key bug fixes!

## Config Overhaul
We've taken a hard look at our config and wanted to make sure that, not only was every field being used, but also that this config could serve us well as we progress down our roadmap and create future features. 

To that effect, we removed many unused fields, switched to using multiaddresses for all network configuration (replacing any `port` fields), formalized the hierarchy of different configuration sources, and added a new `Filesystems` field. 

This new `Filesystems` field allows users to choose the supported filesystems on which they want Qri to store their data. For example, in the future, when we support s3 storage, this `Filesystems` field is where the user can go to configure the path to the storage, if it's the default save location, etc. More immediately however, exposing the `Filesystems` configuration also allows folks to point to a non-default location for their IPFS storage. This leads directly to our next change: moving the default IPFS repo location.

## Migration
One big change we've been working on behind the scenes is upgrading our IPFS dependency. IPFS recently released version 0.6.0, and that's the version we are now relying on! This was a very important upgrade, as users relying on older versions of IPFS (below 0.5.0) would not be seen by the larger IPFS network. 

We also wanted to move the Qri associated IPFS node off the default `IPFS_PATH` and into a location that advertises a bit more that this is the IPFS node we rely on. And since our new configuration allows users to explicitly set the path to the IPFS repo, if a user prefers to point their repo to the old location, we can still accommodate that. By default, the IPFS node that Qri relies on will now live on the `QRI_PATH`.

Migrations can be rough, so we took the time to ensure that upgrading to the newest version of IPFS, adjusting the Qri config, and moving the IPFS repo onto the `QRI_PATH` would go off without a hitch!

## JSON schema
Qri now relies on a newer draft (draft2019_09) of JSON Schema. Our golang implementation of `jsonschema` now has better support for the spec, equal or better performance depending on the keyword, and the option to extend using your own keywords.

## Removed `Update`
This was a real kill-your-darlings situation! The functionality of `update` - scheduling and running `qri saves` - can be done more reliably using other schedulers/taskmanagers. Our upcoming roadmap expands many Qri features, and we realized we couldn't justify the planning/engineering time to ensure `update` was up to our standards. Rather then letting this feature weigh us down, we realized it would be better to remove `update` and instead point users to docs on how to schedule updates. One day we may revisit updates as a plugin or wrapper.

## Merkledag error
Some users were getting `Merkledag not found` errors when trying to add some popular datasets from Qri Cloud (for example `nyc-transit-data/turnstile_daily_counts_2019`). This should no longer be the case!

## Specific Command Line Features/Changes
- `qri save` - use the `--drop` flag to remove a component from that dataset version
- `qri log` - use the `--local` flag to only get the logs of the dataset that are storied locally
            - use the `--pull` flag to only get the logs of the dataset from the network (explicitly not local)
            - use the `--remote` flag to specify a remote off of which you want to grab that dataset's log. This defaults to the qri cloud registry
- `qri get` - use the `-- zip` flag to export a zip of the dataset

## Specific API Features/Changes
- `/fetch` - removed, use `/history?pull=true` 
- `/history` - use the `local=true` param to only get the logs of a dataset that are stored locally
             - use the `pull=true` param to get the logs of a dataset from the network only (explicitly not local)
             - use the `remote=REMOTE_NAME` to specify a remote off of which you want to grab that dataset's log. This defaults to the qri cloud registry

### Bug Fixes

* **AddDataset:** preserve original path after `ResolveHeadRef` ([ff87926](https://github.com/qri-io/qri/commit/ff87926))
* **dsfs:** Ensure name and peername are not written to IPFS ([654ed64](https://github.com/qri-io/qri/commit/654ed64))
* **dsref:** Change char to character in dsref.parse errors ([83882b0](https://github.com/qri-io/qri/commit/83882b0))
* **dsref:** Improve dsref parse error messages ([2f97a4e](https://github.com/qri-io/qri/commit/2f97a4e))
* **get:** Quick fix to ignore merkledag errors ([14dd03e](https://github.com/qri-io/qri/commit/14dd03e))
* **init:** Derive dataset name from directory base, not whole path ([8dec66b](https://github.com/qri-io/qri/commit/8dec66b))
* **init:** Multiple fixes to init, save, dscache ([a21cc31](https://github.com/qri-io/qri/commit/a21cc31))
* **lib:** rpc calls need to reference renamed struct `PeerMethods` ([4ef8122](https://github.com/qri-io/qri/commit/4ef8122))
* **lib.Get:** add FSIPath when ref.Path is prefixed with "/fsi" ([74fbe4c](https://github.com/qri-io/qri/commit/74fbe4c))
* **logbook:** enforce single-author permissions on write methods ([b8cd838](https://github.com/qri-io/qri/commit/b8cd838))
* **logbook:** Only a single Logbook method, owner check in logsync ([c741ef0](https://github.com/qri-io/qri/commit/c741ef0))
* **logbook:** use sync.Once for rollback functions ([a59725d](https://github.com/qri-io/qri/commit/a59725d))
* **migrate:** don't copy entire home directory if IPFS_PATH isn't set ([393473b](https://github.com/qri-io/qri/commit/393473b))
* **migrate:** wait for repo to shut down after migration ([7aad0c3](https://github.com/qri-io/qri/commit/7aad0c3))
* **naming:** Handle names with upper-case characters ([d7138a1](https://github.com/qri-io/qri/commit/d7138a1))
* **regClient:** ResolveRef now returns registry address ([c2cdf6a](https://github.com/qri-io/qri/commit/c2cdf6a))
* **remove:** When removing foreign dataset, remove the log ([978def4](https://github.com/qri-io/qri/commit/978def4))
* **repo:** Construct temp repo using correct path ([2b1518f](https://github.com/qri-io/qri/commit/2b1518f))
* **rpc:** shut down gracefully after completed RPC call ([044a16a](https://github.com/qri-io/qri/commit/044a16a))
* **windows:** Don't detect terminal type ([92f0613](https://github.com/qri-io/qri/commit/92f0613))
* simplify issue templates ([11c4d57](https://github.com/qri-io/qri/commit/11c4d57)), closes [#1394](https://github.com/qri-io/qri/issues/1394)
* **temp_repo:** cancel context on LoadDataset method ([894134e](https://github.com/qri-io/qri/commit/894134e))
* correct issue template path ([eced3b5](https://github.com/qri-io/qri/commit/eced3b5)), closes [#1362](https://github.com/qri-io/qri/issues/1362)
* **save:** Save change detection with readme works correctly ([df3975e](https://github.com/qri-io/qri/commit/df3975e))


### chore

* remove update command, subsystem, api ([8b55243](https://github.com/qri-io/qri/commit/8b55243))


### Code Refactoring

* **save:** remove publish flag from save ([dc8737b](https://github.com/qri-io/qri/commit/dc8737b))


### Features

* **`lib`, `cmd`:** add wait group and done channel to `lib.Instance` and `cmd.QriOptions` ([69d6021](https://github.com/qri-io/qri/commit/69d6021))
* **cmd:** add migrate flag to connect command ([4a465a9](https://github.com/qri-io/qri/commit/4a465a9))
* **dscache:** Dscache used for fsi init and checkout ([3b47d1d](https://github.com/qri-io/qri/commit/3b47d1d))
* **dscache:** Remove operations update the dscache if it exists ([3212add](https://github.com/qri-io/qri/commit/3212add))
* **dsref:** ParseLoadResolve turns strings into datasets ([0283fc0](https://github.com/qri-io/qri/commit/0283fc0))
* **dsref.Resolve:** add source multiaddr return value ([dec83f9](https://github.com/qri-io/qri/commit/dec83f9))
* **export:** Rewrite zip creation so it closely resembles fsi ([25fc08c](https://github.com/qri-io/qri/commit/25fc08c))
* **export:** With no filename, write zip bytes to stdout or API ([880ee63](https://github.com/qri-io/qri/commit/880ee63))
* **logbook:** write push & pull operations, logbook diagnostic method ([8d1b6c9](https://github.com/qri-io/qri/commit/8d1b6c9))
* **migrate:** remove qri-only IPFS repo on config migration ([d13511c](https://github.com/qri-io/qri/commit/d13511c))
* **migration:** OneToTwo basic implementation ([4159a25](https://github.com/qri-io/qri/commit/4159a25))
* **RefResolver:** introduce RefResolver interface, use in DatasetRequests.Get ([d5ddc8f](https://github.com/qri-io/qri/commit/d5ddc8f))
* **regClient:** add initial registry client resolver ([3092c75](https://github.com/qri-io/qri/commit/3092c75))
* **save:** add --drop flag to remove dataset components when saving ([63a2808](https://github.com/qri-io/qri/commit/63a2808))
* **zip:** Cmd options for `get --format zip`, initID in WriteZip ([bc3b318](https://github.com/qri-io/qri/commit/bc3b318))


### BREAKING CHANGES

* update command and all api endpoints are removed
* removed `/fetch` endpoint - use `/history` instead. `local=true` param ensure that the logbook data is only what you have locally in your logbook


<a name="0.9.8"></a>
# [0.9.8](https://github.com/qri-io/qri/compare/v0.9.7...v0.9.8) (2020-04-20)

0.9.8 is a quick patch release to fix export for a few users who have been having trouble getting certain datasets out of qri.

### Fixed Export
This patch release fixes a problem that was causing some datasets to not export properly while running `qri connect`.

### Naming rules
This patch also clarifies what characters are allowed in a dataset name and a peername. From now on a legal dataset name and username must:
* consist of only lowercase letters, numbers 0-9, the hyphen "-", and the underscore "_".
* start with a letter

Length limits vary between usernames and dataset names, but qri now enforces these rules more consistently. Existing dataset names that violate these rules will continue to work, but will be forced to rename in a future version. New datasets with names that don't match these rules cannot be created.

### Bug Fixes

* **checkout:** checkout fails early if link exists ([b8f697f](https://github.com/qri-io/qri/commit/b8f697f))
* **cmd:** checkout supports specifying a directory ([3406c8a](https://github.com/qri-io/qri/commit/3406c8a))
* **cmd:** completion supports config, structure and peer args, added workdir ([#1268](https://github.com/qri-io/qri/issues/1268)) ([ef62c71](https://github.com/qri-io/qri/commit/ef62c71))
* **cmd:** tty should auto disable color for windows ([abf678d](https://github.com/qri-io/qri/commit/abf678d))
* **export:** handle ok-case of bad viz while connected ([1bb2463](https://github.com/qri-io/qri/commit/1bb2463))
* **log:** Timeout log retrieval so it won't hang ([14d885a](https://github.com/qri-io/qri/commit/14d885a))
* **parse:** Allow hyphens in usernames and dataset names ([c3b616a](https://github.com/qri-io/qri/commit/c3b616a))
* **setup:** Detect colors during init, which fixes setup ([278241c](https://github.com/qri-io/qri/commit/278241c))
* **setup:** Don't crash when running `qri setup` ([e0202f4](https://github.com/qri-io/qri/commit/e0202f4))


### Features

* **name:** Generate valid dataset names, use in multiple places ([45502dc](https://github.com/qri-io/qri/commit/45502dc))
* **save:** File hint, from --file or --body flags, informs commit message ([10403f5](https://github.com/qri-io/qri/commit/10403f5))



<a name="0.9.7"></a>
# [0.9.7](https://github.com/qri-io/qri/compare/v0.9.6...v0.9.7) (2020-04-07)

aka `midnight_blue_sloughi`

Qri CLI v0.9.7 is **huge**. This release adds SQL support, turning Qri into an ever-growing database of open datasets.

If that wasn't enough, we've added tab completion, nicer automatic commit messages, unified our command descriptions, and fixed a whole slew of bugs!

## 📊 Run SQL on datasets

Experimental support for SQL is here! Landing this feature brings qri full circle to the [original whitepaper](http://qri.io/papers/deterministic-querying) we published in 2017. 

We want to live in a world where you can `SELECT * FROM any_qri_dataset`, and we're delighted to say that day is here. 

We have plans to improve & build upon this crucial feature, and are marking it as experimental while we flesh out our SQL implemetation. We'll drop the "experimental" flag when we support a healthy subset of the SQL spec.

We've been talking about SQL a bunch in our community calls:
* [🎦 introducing SQL support](https://youtu.be/_kvwuZbnyV4?t=2030) 
* [🎦 qri as a global SQL database](https://youtu.be/U6FoBaO0tYM?t=1612)
* [🎦 SQL errors & prepping datasets for querying](https://youtu.be/D5zUIS_v0iY?t=242)


## 🚗🏁 Autocomplete 

The name says it all. after following the instructions on `qri generate --help`, type `qri get`, then press tab, and _voilá_, your list of datasets appears for the choosing. This makes working with datasets much easier, requiring you to remember and type less. [🎦 Here's](https://youtu.be/ROkxdM2pRgY?t=145) a demo from our community call.


## 🤝📓 Friendlier Automatic Commit Messages

For a long time Qri has automatically generated commit messages for you if one isn't suppied by analyzing what's changed between versions. This release makes titles that look like this:

```
updated structure, viz, and transform
```

and adds detailed messages that look like this:
```
structure:
    updated schema.items.items.63.title
viz:
    updated scriptPath
transform:
    updated resources./ipfs/QmfQu6qBS3iJEE3ohUnhejb7vh5KwcS5j4pvNxZMi717pU.path
    added scriptBytes
    updated syntaxVersion
```

These automatic messages form a nice textual description of what's changed from version to version. Qri will automatically add these if you don't provide `--title` and/or `--message` values to `qri save`.

## 📙 Uniform CLI help

Finally, a big shout out to one of our biggest open source contributions to date! @Mr0Grog not only contributed a massive cleanup of our command line help text, they also wrote a [style guide](https://github.com/qri-io/qri/blob/master/DEVELOPERS.md#cli-help-style) based on the existing help text for others to follow in the future!

### Bug Fixes

* **base:** permission for files generated on init ([14816f2](https://github.com/qri-io/qri/commit/14816f2))
* **cmd:** added the email flag to 'registry prove' as it required ([#1200](https://github.com/qri-io/qri/issues/1200)) ([996f3de](https://github.com/qri-io/qri/commit/996f3de))
* **cmd:** autocomplete failed to handle search ([#1257](https://github.com/qri-io/qri/issues/1257)) ([e48001d](https://github.com/qri-io/qri/commit/e48001d))
* **cmd:** pass node when not making online request for peers ([#1234](https://github.com/qri-io/qri/issues/1234)) ([2cc7aff](https://github.com/qri-io/qri/commit/2cc7aff))
* **cmd:** properly utilize --no-color and --no-prompt ([ff5bdeb](https://github.com/qri-io/qri/commit/ff5bdeb))
* **cmd:** qri list should match against peername and dataset name ([eb38505](https://github.com/qri-io/qri/commit/eb38505))
* **cmd:** restrict number of args in fsi commands ([4c8e42c](https://github.com/qri-io/qri/commit/4c8e42c))
* **cmd:** signup should provide feedback on success ([855ff8f](https://github.com/qri-io/qri/commit/855ff8f))
* **cmd:** stats command works with an FSI directory ([f6696c1](https://github.com/qri-io/qri/commit/f6696c1)), closes [#1186](https://github.com/qri-io/qri/issues/1186)
* **dry-run:** dry-runs must never add to the refstore ([d1b71c2](https://github.com/qri-io/qri/commit/d1b71c2))
* **dsfs:** fix dag structure, defend against bad DAGs ([73d0f98](https://github.com/qri-io/qri/commit/73d0f98))
* **export:** export even if dag contains missing viz referernce ([44c696d](https://github.com/qri-io/qri/commit/44c696d)), closes [#1161](https://github.com/qri-io/qri/issues/1161)
* **fetch:** fetch should return DatasetLogItems ([#1221](https://github.com/qri-io/qri/issues/1221)) ([842aef5](https://github.com/qri-io/qri/commit/842aef5))
* **fsi:** qri remove --all --force should not fail on low value files ([#1203](https://github.com/qri-io/qri/issues/1203)) ([398e0ac](https://github.com/qri-io/qri/commit/398e0ac))
* **get:** Get with a path will ignore fsi, to get old versions ([2125edd](https://github.com/qri-io/qri/commit/2125edd))
* **profileID:** Properly decode example profileID for tests that use it ([4f55c6c](https://github.com/qri-io/qri/commit/4f55c6c))
* **registry:** Prompt for password without echoing it to terminal ([3906b49](https://github.com/qri-io/qri/commit/3906b49))
* **rpc:** registered json.RawMessage for gob encoding ([#1232](https://github.com/qri-io/qri/issues/1232)) ([0832292](https://github.com/qri-io/qri/commit/0832292))
* **save:** Cannot save new datasets if name contains upper-case characters ([ff68b40](https://github.com/qri-io/qri/commit/ff68b40))
* **save:** If body is too large to diff, compare checksums ([37dc5c7](https://github.com/qri-io/qri/commit/37dc5c7))
* **save:** Inferred name must start with a letter ([86ddca3](https://github.com/qri-io/qri/commit/86ddca3))
* **search:** change type from interface to dataset so rpc can serialize ([#1226](https://github.com/qri-io/qri/issues/1226)) ([ebadaec](https://github.com/qri-io/qri/commit/ebadaec))
* **startf:** script print statements print to stderr ([3b3d6b8](https://github.com/qri-io/qri/commit/3b3d6b8))
* **transform:** Re-open transform after friendly, so it writes to cafs ([6f30ae5](https://github.com/qri-io/qri/commit/6f30ae5))
* **unlink:** Fsi unlink handles more edge-cases, acts more sane. ([945b853](https://github.com/qri-io/qri/commit/945b853))
* **update:** Change default port for update service ([6500a91](https://github.com/qri-io/qri/commit/6500a91))
* **watchfs:** Methods need a mutex to avoid concurrent writes ([2cebeff](https://github.com/qri-io/qri/commit/2cebeff))
* **whatchanged:** Hide command so it doesn't show up in help ([2ea1441](https://github.com/qri-io/qri/commit/2ea1441))


### Features

* **api:** add SQL endpoint ([4f5be38](https://github.com/qri-io/qri/commit/4f5be38))
* **cmd:** added command to generate basic cmd autocompletion scripts ([#1240](https://github.com/qri-io/qri/issues/1240)) ([0845289](https://github.com/qri-io/qri/commit/0845289))
* **cmd:** remove `--blank` option from `export` ([47f8c70](https://github.com/qri-io/qri/commit/47f8c70))
* **dscache:** Dscache can be used for get command ([144ad50](https://github.com/qri-io/qri/commit/144ad50))
* **save:** Friendly commit messages by analyzing head, body, and transform ([eb91b27](https://github.com/qri-io/qri/commit/eb91b27))
* **showcommit:** Split cmd status-at-version into showcommit. ([9dd444e](https://github.com/qri-io/qri/commit/9dd444e))
* **sql:** run SQL SELECT queries on datasets ([7a6d8ae](https://github.com/qri-io/qri/commit/7a6d8ae))


### BREAKING CHANGES

* **cmd:** `qri export --blank` has been removed. (You can still `qri export peer/some_dataset`.)



# [v0.9.6](https://github.com/qri-io/qri/compare/v0.9.5...v0.9.6) (2020-03-05)

This patch release fixes a number of small bugs, mainly in support of our Desktop app, and continues infrastructural improvements in preparation for larger feature releases. These include: our improved diff experience, significantly better filesystem integration, and a new method of dataset name resolution that better handles changes across a peer network.

### Bug Fixes

* **cmd/list:** show username with no datasets to list ([a7cbde6](https://github.com/qri-io/qri/commit/a7cbde674b4eab23ea8b566c7c96f08f0353f220))
* **history:** Include the foreign field in history requests ([52c5cc1](https://github.com/qri-io/qri/commit/52c5cc1e9bb4978cee156a6f583afc3be9709773))


### Features

* **diff:** show context in diffs ([6d6abb2](https://github.com/qri-io/qri/commit/6d6abb279745810668bf40edfe3b4fc622c0d545))
* **dscache:** Fill dscache when saving, using func pointer in logbook. ([438fd03](https://github.com/qri-io/qri/commit/438fd039dd308a8ff072fb32bed3fcc2ddb74aca))



<a name="v0.9.5"></a>
# [v0.9.5](https://github.com/qri-io/qri/compare/v0.9.4...v0.9.5) (2020-02-27)

This patch release is focused on a number of API refactors, and sets the stage for a new subsystem we're working on called dscache. It's a small release, but should help stabilize communication between peer remotes & the registry.


### Bug Fixes

* **api:** inline script bytes for get requests ([4d4794f](https://github.com/qri-io/qri/commit/4d4794f))
* **cli:** remove extraneous logging from `qri registry prove` ([de640a2](https://github.com/qri-io/qri/commit/de640a2)), closes [#1096](https://github.com/qri-io/qri/issues/1096)
* **cmd:** refSelect always writes to stderr ([b88eb0c](https://github.com/qri-io/qri/commit/b88eb0c))
* **dry-run:** fix logbook conflict with dry-run ([4c051b6](https://github.com/qri-io/qri/commit/4c051b6))
* **dscache:** A few dscache improvements ([a7c7d36](https://github.com/qri-io/qri/commit/a7c7d36))
* **dscache:** Add refs in dsref that logbook is missing. More tests. ([2ddd3b6](https://github.com/qri-io/qri/commit/2ddd3b6))
* **dscache:** Alphabetize refs in the dscache. Add tests. ([8f4943c](https://github.com/qri-io/qri/commit/8f4943c))
* **dsref:** Parse dsref from strings. Use for rename and init. ([338e700](https://github.com/qri-io/qri/commit/338e700))
* **lint:** Fix style problems, event name change ([2f90119](https://github.com/qri-io/qri/commit/2f90119))
* **logbook:** addChild should not create duplicate children ([9a41b6b](https://github.com/qri-io/qri/commit/9a41b6b))
* **logbook:** always provide a privateKey to logbook ([ae68ccc](https://github.com/qri-io/qri/commit/ae68ccc))
* **logbook:** use AddChild so oplogs set ancestry ([be633b9](https://github.com/qri-io/qri/commit/be633b9))
* **p2p:** add local to p2ptest muxer ([4937997](https://github.com/qri-io/qri/commit/4937997))
* **profileID:** Disambiguate profileID construction to avoid subtle bugs ([a41d41a](https://github.com/qri-io/qri/commit/a41d41a))
* **publish:** Use Refselect for publish, so FSI works. Use dsref, add tests. ([68dfff3](https://github.com/qri-io/qri/commit/68dfff3))
* **regclient:** update to new search endpoint, fix response ds.Name ([d3fbce4](https://github.com/qri-io/qri/commit/d3fbce4))
* **registry:** fix inaccurate tests covered by catch-all-200 response regserver handler ([1d30975](https://github.com/qri-io/qri/commit/1d30975))
* **remote:** fetching logs populates VersionInfo.Foreign ([e50cd19](https://github.com/qri-io/qri/commit/e50cd19))
* **remote:** Handle error in order to avoid nil pointer ([4a88b52](https://github.com/qri-io/qri/commit/4a88b52))
* **remove:** RemoveEntireDataset is better at cleaning up broken states ([efaf600](https://github.com/qri-io/qri/commit/efaf600))
* **temp registry:** make configuration not interrupt common config ([b618472](https://github.com/qri-io/qri/commit/b618472))


### Features

* **api:** support for dynamic dataset readme rendering ([ef06599](https://github.com/qri-io/qri/commit/ef06599))
* **dscache:** Dscache holds info about datasets based upon logbook ([28811d7](https://github.com/qri-io/qri/commit/28811d7))
* **dscache:** Dscache will get updated by save, if it exists in the repo ([21c5ca7](https://github.com/qri-io/qri/commit/21c5ca7))
* **dscache:** Fill dataset details, filter when listing ([e460a7c](https://github.com/qri-io/qri/commit/e460a7c))
* **event:** package event implements an event bus ([6068f88](https://github.com/qri-io/qri/commit/6068f88))
* **event:** When a folder is checked out, add it to watchfs ([7536221](https://github.com/qri-io/qri/commit/7536221))
* **feed:** add feed methods to lib backed by registry ([53e6eb3](https://github.com/qri-io/qri/commit/53e6eb3))
* **registry:** add preview support to registry fetching ([19f12bd](https://github.com/qri-io/qri/commit/19f12bd))
* **remote:** add FeedsPreCheck and PreviewPreCheck hooks ([97fb2be](https://github.com/qri-io/qri/commit/97fb2be))
* **remote:** ephemeral log fetching with remoteClient.FetchLogs ([1651def](https://github.com/qri-io/qri/commit/1651def))
* **remote:** remote client signs HTTP feed and preview requests ([a742faa](https://github.com/qri-io/qri/commit/a742faa))
* **watchfs:** Watched filesystem folders send dataset names as well ([f7c7bfb](https://github.com/qri-io/qri/commit/f7c7bfb))



# [v0.9.4](https://github.com/qri-io/qri/compare/v0.9.3...v0.9.4) (2020-01-21)

This patch release fixes a number of FSI (file system integration) issues and infrastructure changes that will improve Desktop. These include the restoration of the validate command, handling certain changes to the file system done outside of qri, improved logging, and Windows bug fixes.

### Bug Fixes

* **fsi:** Remove references that have no path and no working directory. ([f04981b](https://github.com/qri-io/qri/commit/f04981bd8dbd5f2c42aa18bf95b80e00d4bbd8e7))
* **log:** Add --log-all flag to enable logging everywhere ([c3c30b1](https://github.com/qri-io/qri/commit/c3c30b1b91d87b8f2aff54ee5f8e8a4a82d8a118))
* **ref:** CanonicalizeDatasetRef sets fsipath even if ref is full. ([2973bfa](https://github.com/qri-io/qri/commit/2973bfaae85b0f1c492f62ba7602f2e25cc27fce))
* **registry:** update peername and dsrefs on registry signup name change ([8bc151c](https://github.com/qri-io/qri/commit/8bc151c434fc09863d2754a2ab3d027a67314214))
* **remove:** Remove foreign datasets while connected, don't hang ([a93470b](https://github.com/qri-io/qri/commit/a93470b958a57b230e28c72721fc8b5c1498616f))
* **test:** API tests are stable and pass even when version changes ([2c401e5](https://github.com/qri-io/qri/commit/2c401e5d1af63a74f1f15e70ca3272c04c9b206d))
* **validate:** Allow either --structure or --schema flag. Add cmd/ tests ([6978113](https://github.com/qri-io/qri/commit/69781137253218d047cfb21750e4f26cfa8a2787))
* **validate:** Improve validate so it works with FSI ([07cde33](https://github.com/qri-io/qri/commit/07cde335259b8248fb68fab2ad5fead4ecc6e048))
* **windows:** Fix "Access is defined." error on Windows when renaming. ([0d11744](https://github.com/qri-io/qri/commit/0d11744e045a3e1abc07868910e9c98acab29f04))


### Features

* **doggo:** Setup flag that only creates a nick and displays it ([a75e2d0](https://github.com/qri-io/qri/commit/a75e2d0885c3ecaac9785d73c72b6e0d1d05ffea))
* **logbook:** add WriteAuthorRename method to logbook ([d4ac19c](https://github.com/qri-io/qri/commit/d4ac19c6c66eaebe1b377f24996627c8054194c7))
* **websocket:** Open websocket server, watch filesystem for events. ([1d8e022](https://github.com/qri-io/qri/commit/1d8e022b4bd858d8cba0ee723175772b6daf7ec6))



# [v0.9.3](https://github.com/qri-io/qri/compare/v0.9.2...v0.9.3) (2019-12-12)

This patch release includes bug fixes and improvements around the working directory, in particular doing better with removing datasets, and generating automatic commit messages. There are also some small feature changes that are mostly laying the groundwork for future features.

### Bug Fixes

* **cmd:** Clean tmp directories in command output, to simplify tests ([d302828](https://github.com/qri-io/qri/commit/d302828fb4be078919d911974190b1e174ba77b4))
* **diff:** Diff with no history returns a reasonable error message ([35207ba](https://github.com/qri-io/qri/commit/35207bae758e4b8e039eff4769eb6f1eec9d1856))
* **fsi:** FSI should use detected schema for reading body ([9fed513](https://github.com/qri-io/qri/commit/9fed513ff893dd0886b592265c6c961dfa37977a))
* **fsi:** When working directory is moved or deleted, update FSI Path ([3c4153b](https://github.com/qri-io/qri/commit/3c4153b1b8ce4b5bb1b847c1bf3dc75828ef6d0c))
* **remove:** Improve error message when removing a dirty dataset ([21adb9b](https://github.com/qri-io/qri/commit/21adb9b20af5d84c9b8d379cb5e23c667b8e5d3c))
* **remove:** Remove works on foreign datasets, even if logbook missing ([fe4598a](https://github.com/qri-io/qri/commit/fe4598abaa9f4427f05ae8a046e9dcc1fc9a318e))
* **search:** Run TestSearchRun in a tmp directory ([633185e](https://github.com/qri-io/qri/commit/633185e59f5df15e1b900ecd4586666e5dad50de))


### Features

* **friendly:** Friendly generator for commit title and message ([29aa674](https://github.com/qri-io/qri/commit/29aa6740b6be494d4ba224ff56e89373bb1324a6))
* **lib:** add OptLogbook to provide custom logbook ([31667b7](https://github.com/qri-io/qri/commit/31667b7c84051b8283f26102777c81e854ac7c93))
* **oplog:** add Logstore & PersonalLogstore interfaces ([0d7f7a5](https://github.com/qri-io/qri/commit/0d7f7a5c94d673498a8ad6ecb51d44578bd87701))
* **refs:** `qri list --raw` will display raw reference data ([d848f46](https://github.com/qri-io/qri/commit/d848f46d24cef35f218cc14e3f7d4c955d0c3377))
* **save:** Flag --new for save ensures that we're saving a new dataset ([22db718](https://github.com/qri-io/qri/commit/22db71875192dae7cccb8e30401c646c6554f66f))



# [v0.9.2](https://github.com/qri-io/qri/compare/v0.9.1...v0.9.2) (2019-11-18)

In this patch release we're fixing a bunch of tiny bugs centered around removing datasets, and adding methods for column statistics

## 📊 Get to know your data with stats
This release adds support for stats calculation. The easiest way to see stats is to `get` 'em:

```
$ qri get stats me/dataset

# getting stats also works in an FSI-linked directory
# so you can drop the dataset reference & just type:
$ cd /path/to/linked/dataset/directory
$ qri get stats
```

In both cases you'll get a JSON document of stats, with one stat aggregating each column in your datsaet. The type of stat created depends on the data type being aggregated. Here's the table of stats calculated so far:

| column data type | stat type  | notes                                                                |
| ---------------- | ---------- | -------------------------------------------------------------------- |
| string           | string     | Calculates a term frequency If there are fewer than 10,000 unique values, fequencies that only occur once aren't listed in frequency map and instead increment a "unique" count. |
| number           | numeric    | Calculates a 10-bucket histogram, as well as min, max, mean, median. |
| boolean          | boolean    | Calculates a true / false / other count                              |
| null             | null       | counts the number of null values                                     |



### Bug Fixes

* **api:** Remove force flag for api ([2acf8d8](https://github.com/qri-io/qri/commit/2acf8d8))
* **fsi:** Cleanup quote usage, name return values ([1fa2f2b](https://github.com/qri-io/qri/commit/1fa2f2b))
* **fsi status:** use byte comparison for transform equality test ([73c02a5](https://github.com/qri-io/qri/commit/73c02a5))
* **init:** Can init linked datasets with json format ([a679155](https://github.com/qri-io/qri/commit/a679155))
* **init:** If init has errors, rollback changes ([bc3eb42](https://github.com/qri-io/qri/commit/bc3eb42))
* **ls:** Don't crash when `list` runs over RPC ([7eedc5d](https://github.com/qri-io/qri/commit/7eedc5d))
* **readme:** Can save a readme component using --file flag ([714c470](https://github.com/qri-io/qri/commit/714c470))
* **remove:** Can remove a dataset if its working directory is missing ([9a5b4d8](https://github.com/qri-io/qri/commit/9a5b4d8))
* **remove:** Comments, cleanups, improve tests for remove ([4c1a8c1](https://github.com/qri-io/qri/commit/4c1a8c1))
* **remove:** Fix multiple problems with remove ([67f5ad5](https://github.com/qri-io/qri/commit/67f5ad5)), closes [#1000](https://github.com/qri-io/qri/issues/1000)
* **remove:** Force flag always removes files. Remove directory too. ([e89a2f6](https://github.com/qri-io/qri/commit/e89a2f6))
* **rename:** Allow rename for dataset with no history. Update .qri-ref ([ebcb676](https://github.com/qri-io/qri/commit/ebcb676))
* **rename:** Simplify some conditionals around renames ([965d78d](https://github.com/qri-io/qri/commit/965d78d))
* **stats:** `int` was not capturing all integer options, add `int32` or `int64` ([e00791f](https://github.com/qri-io/qri/commit/e00791f))
* **stats:** fix numeric median calculation ([24d29b3](https://github.com/qri-io/qri/commit/24d29b3))
* **stats:** gets stats with no structure ([a2046d6](https://github.com/qri-io/qri/commit/a2046d6))
* **stats/cache:** stats must have a cache struct, even if empty ([70279d4](https://github.com/qri-io/qri/commit/70279d4))
* **windows:** Set the .qri repo as hidden on Windows. ([599b977](https://github.com/qri-io/qri/commit/599b977))


### Features

* **`qri stats`:** add stats cmd! outputs unformated json stats ([422ad72](https://github.com/qri-io/qri/commit/422ad72))
* **api.StatsHandler:** add handler and tests for `/stats/` endpoint! ([a103758](https://github.com/qri-io/qri/commit/a103758))
* **config:** add `Stats` config options to the config ([ba85eac](https://github.com/qri-io/qri/commit/ba85eac))
* **fsi transform:** skeletal support for transform scripts via FSI ([98e0cfc](https://github.com/qri-io/qri/commit/98e0cfc))
* **lib:** add `stats.Cache` to the lib instance as `stats` ([a8646f4](https://github.com/qri-io/qri/commit/a8646f4))
* **lib.Stats:** add `lib.Stats` func and tests! ([1210e91](https://github.com/qri-io/qri/commit/1210e91))
* **stats:** add histogram, mean, meadian to numeric stat ([937d145](https://github.com/qri-io/qri/commit/937d145))
* **stats:** introduce stats subsystem ([5dd0ae1](https://github.com/qri-io/qri/commit/5dd0ae1))



# [0.9.1](https://github.com/qri-io/qri/compare/v0.9.0...v0.9.1) (2019-11-05)

This release brings first-class support for Readmes, adds a bunch of stability, and sets the table for exciting collaboration features in the future.

### 📄 Qri now supports readme!
This release brings support for a new dataset component, readmes! Following in a long tradition of readme's in the world of software. Readme's are [markdown](https://daringfireball.net/projects/markdown/) documents for explaining your dataset in human terms.

The easiest way to create a readme is by creating a file called `readme.md` in an FSI-linked directory. Qri will pick up on the file & add it to your dataset. You can see what the rendered HTML version looks like by running `qri render` in an FSI-linked directory.

In the future, we're excited to build out the feature set readme's offer, and think they're a better long-term fit for us than the generic notion of our existing `viz` component. Readme's differ from viz by not allowing generic script inclusion, which allows us to present them in a safer sandbox-like environment. This fits well with our story around transform scripts and the general expectation that scripts Qri interacts with will come with a safer execution properties. 

With this release, support for readme's in [qri.cloud](https://qri.cloud) and [desktop](https://qri.io/desktop) is right around the corner.

Happy note taking!

### 📘 Introducing Logbook

**[video!](https://youtu.be/WBshhfYv740?t=206)**

Until now qri has used stored datasets as it's source of history. Qri keeps commit information in the dataset itself, and creates a log of datasets by having each dataset reference the one before it. Keeping commits in history has a number of advantages:
* all datasets are attributed to the user that made them
* all datasets have an accurate creation timestamp
* all datasets include any notes the author made at the time
* all of these these details are _part of the dataset_, and move with it.

We've gone a long way with this simplistic apporoach, but using datasets as the only source of history has one major limitation: _the history of a dataset is tied to the data itself_. This means you can't uncover the full history of a dataset unless you have _all_ versions of a dataset _stored locally_. Logbook fixes that problem.

Logbook is a _coordination tool_ for talking about who did what, without having to move around the data itself. This means Qri can tell you meaningful things about dataset versions you don't have. This will make syncing faster, and forms the basis for _collaboration_. 

To make use of logbook, all you have to do is... nothing! Logbook is a transparent service that overlays onto traditional Qri commands. You'll see some new details in commands like `qri log` and a few new plumbing commands like `qri fetch` and `qri logbook`, but this feature adds no new requirements to the Qri workflow.

We're most excited about what logbook allows us to do (collaboration!), and can't wait to ship features that will show the benefit of logbook. More fun soon!

### 🏗 Stability Improvements
As always, we're working on stabilizing & improving the way Qri works. We've this release we've focused on bringing stability to three major areas
* filesystem integration (FSI)
* remotes
* diff

### Bug Fixes

* **diff:** Diff implemented using Component interface ([ed88d67](https://github.com/qri-io/qri/commit/ed88d67))
* **fill:** Support setting int64 ([d6dea9a](https://github.com/qri-io/qri/commit/d6dea9a))
* **fsi:** Can get body even if no structure exists. Infer it. ([4cf8132](https://github.com/qri-io/qri/commit/4cf8132))
* **fsi:** Don't output schema for default csv, many related fixes ([9f2c1e9](https://github.com/qri-io/qri/commit/9f2c1e9))
* **fsi:** Rebase this so it works with new component functionality ([e3ba68a](https://github.com/qri-io/qri/commit/e3ba68a))
* **fsi:** Schema is no longer treated as a top-level component ([e8f22c5](https://github.com/qri-io/qri/commit/e8f22c5))
* **fsi init:** fix init writing the wrong dataset name ([0bc6d62](https://github.com/qri-io/qri/commit/0bc6d62))
* **init:** add basic `structure.json` on `init` ([a6f423b](https://github.com/qri-io/qri/commit/a6f423b))
* **log:** fix history being created in reverse order ([c706ef4](https://github.com/qri-io/qri/commit/c706ef4))
* **log:** fix log construction from history, add test ([0567ae9](https://github.com/qri-io/qri/commit/0567ae9))
* **logbook:** logbook cleanup ([525ac78](https://github.com/qri-io/qri/commit/525ac78))
* **publish:** Can only publish / unpublish a head reference ([f834621](https://github.com/qri-io/qri/commit/f834621))
* **readme:** Add readme to dsref, fix style of yaml import ([0f286ef](https://github.com/qri-io/qri/commit/0f286ef))
* **remote:** disambiguate AuthorID & AuthorPubKey use in logsync ([8e95fe1](https://github.com/qri-io/qri/commit/8e95fe1))
* **remote:** remove dataset ([3c38191](https://github.com/qri-io/qri/commit/3c38191))
* **remove:** don't fail if remove encounters an issue traversing qfs history ([f842273](https://github.com/qri-io/qri/commit/f842273)), closes [#989](https://github.com/qri-io/qri/issues/989)
* **rpc:** fix logbook --raw over RPC ([48708b6](https://github.com/qri-io/qri/commit/48708b6))
* incorporate PR feedback fixes ([0970fa0](https://github.com/qri-io/qri/commit/0970fa0))


### Features

* **api:** add /fetch endpoint ([5637f90](https://github.com/qri-io/qri/commit/5637f90))
* **fetch:** fetch pulls logs via logsync ([0f3f2af](https://github.com/qri-io/qri/commit/0f3f2af))
* **log:** add Local field to DatasetLogItem ([f127167](https://github.com/qri-io/qri/commit/f127167))
* **log:** add sizes to log, fix logbook version pagination ([5ab0c0a](https://github.com/qri-io/qri/commit/5ab0c0a))
* **log:** use logbook to show versions ([1c38865](https://github.com/qri-io/qri/commit/1c38865))
* **logbook:** add lib methods & plumbing commands for logbook ([6a9ae8f](https://github.com/qri-io/qri/commit/6a9ae8f))
* **logbook:** add logbook inspection structs & methods ([dd4fb36](https://github.com/qri-io/qri/commit/dd4fb36))
* **logbook:** construct logs from dataset histories ([0a3de84](https://github.com/qri-io/qri/commit/0a3de84))
* **logbook:** log merge combines two logs ([8a3b006](https://github.com/qri-io/qri/commit/8a3b006))
* **logsync:** add p2p logsync handler ([1b0f1c2](https://github.com/qri-io/qri/commit/1b0f1c2))
* **logsync:** syncronize logs over HTTP ([1feb242](https://github.com/qri-io/qri/commit/1feb242))
* **oplog:** `InitOpHash` returns the encoded hash of the initial op of a log ([4898375](https://github.com/qri-io/qri/commit/4898375))
* **readme:** Readme component, prepare for removing viz ([d567a32](https://github.com/qri-io/qri/commit/d567a32))
* **readme:** Render readme component as html ([21c274e](https://github.com/qri-io/qri/commit/21c274e))
* **remote:** add logsync hooks ([70077bd](https://github.com/qri-io/qri/commit/70077bd))
* **remote:** adding logsync to remote ([12fddad](https://github.com/qri-io/qri/commit/12fddad))
* **remote:** embed log hook oplogs in hook context ([412852d](https://github.com/qri-io/qri/commit/412852d))
* **remote:** populate LogPushFinalCheck dataset reference ([a768023](https://github.com/qri-io/qri/commit/a768023))
* **remote:** push & pull logs on publish & add ([16c7b4e](https://github.com/qri-io/qri/commit/16c7b4e))
* **VerifySigParams:** adds `VerifySigParams` & refactors all signature logic ([2c9e3f8](https://github.com/qri-io/qri/commit/2c9e3f8))


### Tests

* **api:** update API tests to reflect breaking change to log ([1b6be4f](https://github.com/qri-io/qri/commit/1b6be4f))


### BREAKING CHANGES

* **api:** log commands and api endpoint return a different data structure now



# [0.9.0](https://github.com/qri-io/qri/compare/v0.9.0-alpha...v0.9.0) (2019-09-26)

0.9.0 makes Qri work like Git! 

# :open_file_folder: File System Integration [(RFC0025)](https://github.com/qri-io/rfcs/blob/master/text/0025-filesystem-integration.md)
This release brings a few new commands into qri. If you're a [git](https://git-scm.com) user, these will look familiar:

```
init        initialize a dataset directory
checkout    checkout creates a linked directory and writes dataset files to that directory
status      Show status of working directory
restore     restore returns part or all of a dataset to a previous state
```

You can now interact with a versioned dataset in similar way you would a git repository. Now creating new versions is as simple as `cd`ing to a linked directory and typing `qri save`.

After a lot of thought & research, we've come to believe that using the filesystem as an interface is a great way to interact with versioned data. Git has been doing this for some time, and we've put thought & care into bringing the aspects of git that work well in this context. 

Running the new `qri init` command will create an _FSI-linked directory_. A new datset will be created in your qri repo, and a hidden file called `.qri-ref` will be created in the folder you've initialized within. When you're linked directory you no longer need to type the name of a dataset to interact with it. `qri get body peername/dataset_name` is just `qri get body` when you're in an FSI-linked directory. You can see which datasets are linked when you `qri list`, it'll show the folder it's linked to.

Unlike git, qri doesn't track _all_ files in a linked folder. Instead it only looks for specific filenames to map to dataset components:

| component | possible filename                                   |
| --------- | ----------------------------------------------------|
| body      | `body.csv`, `body.json`, `body.xlsx`, `body.cbor`   |
| meta      | `meta.json`, `meta.yaml`                            |
| schema    | `schema.json`, `schema.yaml`                        |

We'll be following up with support for transform and viz components shortly. It's still possible to create datasets that _don't_ have a link to the filesystem, and indeed this is still the better way to go for large datasets.

File system integration opens up a whole bunch of opportunities for integration with other tools by dropping back to a common interface: files. Now you can use whatever software you'd like to edit dataset files, and by writing back to that folder with one of these name you're ready to version from the get go. command like `qri status` make it easy to keep track of where you are in your work, and `qri restore` makes it easy to "reset to head".


# :desktop_computer: [Qri Desktop](https://github.com/qri-io/desktop)
This is the first qri release that will be bundled into Qri Desktop, our brand new project for working with datasets. Qri desktop puts a face on qri, We'll be cutting a release of qri desktop shortly. Check it out!

# :cloud: qri.cloud as a new default registry

This release also puts a bunch of work into the registry. We've made the job of a registry smaller, moving much of the behaviour of dataset syncing into _remotes_, which any peer can now become. At the same time, we're hard at work building qri.cloud, our new hosted service for dataset management and collaboration. If you're coming from a prior version of qri, run the following to swich to the new registry:

```
qri config set registry.location https://registry.qri.cloud
```

Now when you `qri publish`, it'll go to qri.cloud. _Lots_ of exciting things coming for qri cloud in the next few months.



### Bug Fixes

* **add:** need to re-initialize add when connecting ([767d2ad](https://github.com/qri-io/qri/commit/767d2ad))
* **api:** remove files field is 'files', not 'delete' ([3017723](https://github.com/qri-io/qri/commit/3017723))
* **fsi:** Add relative directory to fsi body path. Parse body for status. ([4284b5e](https://github.com/qri-io/qri/commit/4284b5e))
* **fsi:** Init does not output a schema.json ([45b770c](https://github.com/qri-io/qri/commit/45b770c))
* **fsi:** Keep detected schema when body exists without a schema ([3ecdbbe](https://github.com/qri-io/qri/commit/3ecdbbe))
* **fsi unlink:** unlink command removes .qri-ref file ([0e6a8fd](https://github.com/qri-io/qri/commit/0e6a8fd))
* **fsi.DeleteFiles:** attempt to remove files, even on early return ([711f733](https://github.com/qri-io/qri/commit/711f733))
* **remove:** removing a dataset with no history should work ([f0ba1a1](https://github.com/qri-io/qri/commit/f0ba1a1))
* **search:** update to new registry api search result format ([1ac83bf](https://github.com/qri-io/qri/commit/1ac83bf))
* **test:** Handle test flakiness by forcing column type to be a string ([1880b17](https://github.com/qri-io/qri/commit/1880b17))


### Features

* **add:** add link flag to lib.Add, CLI, API ([61984f6](https://github.com/qri-io/qri/commit/61984f6))
* **fsi:** add fsimethods.Write to write directly to the linked filesystem ([65cdb12](https://github.com/qri-io/qri/commit/65cdb12))
* **fsi:** Api can set directory to create for init ([90e2115](https://github.com/qri-io/qri/commit/90e2115))
* **fsi:** Init can take a directory name to create for linking ([d62b816](https://github.com/qri-io/qri/commit/d62b816))
* **fsi:** Init may take a --source-body-path to create body file ([20517d9](https://github.com/qri-io/qri/commit/20517d9))
* **log:** book encryption methods, move book into log pkg ([47e8fbd](https://github.com/qri-io/qri/commit/47e8fbd))
* **log:** flatbuffer encoding code & test work ([b88d899](https://github.com/qri-io/qri/commit/b88d899))
* **log:** flatbuffer work, shuffling names, added signatures ([4545d66](https://github.com/qri-io/qri/commit/4545d66))
* **log:** initial CRDT dataset log proof of concept ([eaf4b94](https://github.com/qri-io/qri/commit/eaf4b94))
* **logbook:** add primitive method for getting log bytes ([9a736c2](https://github.com/qri-io/qri/commit/9a736c2))
* **logbook:** frame out API, unexport a bunch of stuff ([fac0bc7](https://github.com/qri-io/qri/commit/fac0bc7))
* **logbook:** initial example test passing ([5deea01](https://github.com/qri-io/qri/commit/5deea01))
* **mock registry:** add support for mock registry config ([7b7b98b](https://github.com/qri-io/qri/commit/7b7b98b))
* **registry:** add signup & prove subcommands ([8262c6d](https://github.com/qri-io/qri/commit/8262c6d))
* **remove:** add files and unlink flags to remove ([4c5a924](https://github.com/qri-io/qri/commit/4c5a924))
* **windows:** Set .qri-ref as hidden in Windows ([6d71d48](https://github.com/qri-io/qri/commit/6d71d48))



# [0.9.0-alpha](https://github.com/qri-io/qri/compare/v0.8.2...v0.9.0-alpha) (2019-09-04)

# Preparing for 0.9.0
We're not quite ready to put the seal-of-approval on 0.9.0, but it's been more than a few months since we cut a release. This alpha-edition splits the difference while we prepare for a full & proper 0.9.0. The forthcoming big ticket item will be _File System Integration_ [(RFC0025)](https://github.com/qri-io/rfcs/blob/master/text/0025-filesystem-integration.md), which dramatically simplifies the story around integrating with a version-controlled dataset.

So while this isn't a proper release, the changelog gives a feel for just how much work is included this go-round. More soon!

### Bug Fixes

* **api:** body requests honor full references ([335d2a4](https://github.com/qri-io/qri/commit/335d2a4))
* **api:** lowercase StatusItem when serialized to JSON ([e1790c9](https://github.com/qri-io/qri/commit/e1790c9))
* **api:** report proper publish status on dataset get ([2242fb7](https://github.com/qri-io/qri/commit/2242fb7))
* **api:** support stored status requests to /dsstatus ([880a7a3](https://github.com/qri-io/qri/commit/880a7a3))
* **api status, body, get:** fsi=true endpoints should not care if there is no history ([0c7649f](https://github.com/qri-io/qri/commit/0c7649f))
* **cfg:** Better error messages when validating config ([6ee7355](https://github.com/qri-io/qri/commit/6ee7355))
* **checkout:** absolutize FSI link creation paths ([64c3b41](https://github.com/qri-io/qri/commit/64c3b41))
* **cmd:** print export info using o.Out ([3bfec29](https://github.com/qri-io/qri/commit/3bfec29))
* **config:** change default registry location to https://registry.qri.cloud ([7d2248a](https://github.com/qri-io/qri/commit/7d2248a))
* **config:** relax repo config validation, make rpc work ([0d16d32](https://github.com/qri-io/qri/commit/0d16d32))
* **connect:** Handle error gracefully when already connected ([4770bde](https://github.com/qri-io/qri/commit/4770bde))
* **diff:** Better error when diffing a dataset with only 1 version ([0a90ab9](https://github.com/qri-io/qri/commit/0a90ab9))
* **diff:** Diff works for linked working directories ([d352e44](https://github.com/qri-io/qri/commit/d352e44))
* **ds:** Update dataset gomod, fix tests for change that omits errCount ([a99727b](https://github.com/qri-io/qri/commit/a99727b))
* **fill:** Error message consistency ([3b5470b](https://github.com/qri-io/qri/commit/3b5470b))
* **fsi:** body.csv files, set structure.format, don't crash in save ([1d83254](https://github.com/qri-io/qri/commit/1d83254))
* **fsi:** canonicalize alias before fetching ([073c640](https://github.com/qri-io/qri/commit/073c640))
* **fsi:** Cleanup error handling and some if statements ([2feb0ab](https://github.com/qri-io/qri/commit/2feb0ab))
* **fsi:** Ensure api does not send 500s for datasets without history ([6193c5f](https://github.com/qri-io/qri/commit/6193c5f))
* **fsi:** Marshal timestamps consistently to fix tests ([4733d9d](https://github.com/qri-io/qri/commit/4733d9d))
* **fsi:** Multiple bug fixes, unit tests. ([f483d95](https://github.com/qri-io/qri/commit/f483d95))
* **fsi:** Proper handling for fsi status with null datasets ([29606ea](https://github.com/qri-io/qri/commit/29606ea))
* **fsi:** serialize link json to lowerCamelCase ([9b8bb8d](https://github.com/qri-io/qri/commit/9b8bb8d))
* **fsi save:** fix fsi not accepting values from API ([8fca975](https://github.com/qri-io/qri/commit/8fca975))
* **lib:** profileMethods need to persist their changes ([ab4296a](https://github.com/qri-io/qri/commit/ab4296a))
* **publish:** sync refstore publication flag to remote publication ([a969b9a](https://github.com/qri-io/qri/commit/a969b9a))
* **regclient.Profile:** Profile struct has field formatting details when marshalling to json ([e3f0238](https://github.com/qri-io/qri/commit/e3f0238))
* **registry:** update repo profile on successful registration ([fcdfc05](https://github.com/qri-io/qri/commit/fcdfc05))
* **repo:** Error for accessing datasets with no history ([aa4dbb6](https://github.com/qri-io/qri/commit/aa4dbb6))
* **restore:** Restore will delete components that didn’t exist in previous version ([7f64f85](https://github.com/qri-io/qri/commit/7f64f85))
* **status:** Parse errors should also be shown for schema ([271b837](https://github.com/qri-io/qri/commit/271b837))
* **status:** Status displays parse errors, instead of bailing out ([6e5cde7](https://github.com/qri-io/qri/commit/6e5cde7))
* **unlinked status:** show status on unlinked datasets ([e7f561b](https://github.com/qri-io/qri/commit/e7f561b))
* **use:** Fix `qri get` with no args bug. Bump version. ([aaf295e](https://github.com/qri-io/qri/commit/aaf295e))
* **use:** Move `use` up to cmd/ from lib/. Delete select from repo/ ([8286730](https://github.com/qri-io/qri/commit/8286730))
* **validate:** Validate works with FSI. Various FSI cleanups. ([64c1ce1](https://github.com/qri-io/qri/commit/64c1ce1))


### Features

* **api:** add checkout endpoint ([18980f9](https://github.com/qri-io/qri/commit/18980f9))
* **api:** add fsilinks endpoint for debugging fsi ([f319452](https://github.com/qri-io/qri/commit/f319452))
* **api:** init fsi endpoint ([7613302](https://github.com/qri-io/qri/commit/7613302))
* **api.FSI:** initial FSI api methods ([63e57a3](https://github.com/qri-io/qri/commit/63e57a3))
* **checkout:** Checkout creates a linked directory from the repo ([be4d3eb](https://github.com/qri-io/qri/commit/be4d3eb))
* **cmd:** add fsi plumbing subcommand ([4d12de8](https://github.com/qri-io/qri/commit/4d12de8))
* **fsi:** add fsi body api handler ([ea7069c](https://github.com/qri-io/qri/commit/ea7069c))
* **fsi:** add save via fsi endpoint ([7ed6439](https://github.com/qri-io/qri/commit/7ed6439))
* **fsi:** link flatbuffer serialization ([71e9ff8](https://github.com/qri-io/qri/commit/71e9ff8))
* **fsi:** Package fsi defines qri file system integration ([0214d2f](https://github.com/qri-io/qri/commit/0214d2f))
* **fsi:** read fsi-linked dataset ([74705ac](https://github.com/qri-io/qri/commit/74705ac))
* **fsi:** Remove GetDatasetRefString, use RefSelect everywhere ([4b1edc6](https://github.com/qri-io/qri/commit/4b1edc6))
* **init:** qri init binds a directory to a dataset selection ([8e5a203](https://github.com/qri-io/qri/commit/8e5a203))
* **ipfs_http:** support for ipfs_http store ([aaf7d0a](https://github.com/qri-io/qri/commit/aaf7d0a))
* **lib:** add extra options to make NewInstance work in test settings ([b67dcff](https://github.com/qri-io/qri/commit/b67dcff))
* **qri-ref:** Cleanup return values, use varName package ([d3b8688](https://github.com/qri-io/qri/commit/d3b8688))
* **qri-ref:** Commands to link a working directory to a dataset ([93c50c2](https://github.com/qri-io/qri/commit/93c50c2))
* **registry:** sync local profile details on successful signup ([d184f98](https://github.com/qri-io/qri/commit/d184f98))
* **remote:** add support for ref removes ([e2a6d3a](https://github.com/qri-io/qri/commit/e2a6d3a))
* **replace-save dataset:** support dataset save without patching from prior verion ([9906727](https://github.com/qri-io/qri/commit/9906727))
* **restore:** Add api for restore. Test for checkout and restore apis. ([d1d9670](https://github.com/qri-io/qri/commit/d1d9670))
* **restore:** Restore command for restoring component files in FSI ([620d95d](https://github.com/qri-io/qri/commit/620d95d))
* **status:** add initial status check func ([e3eb1c1](https://github.com/qri-io/qri/commit/e3eb1c1))
* **status:** FSI status include file mtime. Requires flag for cmdline. ([2ccffd6](https://github.com/qri-io/qri/commit/2ccffd6))
* **status:** report removed files in qri status ([e5c7fde](https://github.com/qri-io/qri/commit/e5c7fde))
* **status:** Status at a specific dataset version, historical changes ([cbc108c](https://github.com/qri-io/qri/commit/cbc108c))
* **status:** Status UI. RefSelect to simplify handling references in cmd ([21f547a](https://github.com/qri-io/qri/commit/21f547a))
* **unpublish:** support both publish & unpublish, expand remote behaviour ([f23f467](https://github.com/qri-io/qri/commit/f23f467))



# [0.8.2](https://github.com/qri-io/qri/compare/v0.8.1...v0.8.2) (2019-06-25)


Version 0.8.2 is a patch release that improves qri's command-line client in numerous small ways

## Webapp
* Multiple problems were fixed around webapp fetching, saving, and pinning. The general end-user experience was also improved by allowing the frontend to show an interstitial screen while the backend was fetching the webapp.

## Stdout / stderr usage
* Many related issues around stdout and stderr usage were fixed, such as missing endlines being added, and prompts meant to always be seen were moved to stderr.

## Misc
* `qri get` gained a flag `--pretty` to pretty-print json.
* `qri save` with the strict flag set will produce an error if it fails to validate
* `qri add` can use a full reference to specify which version to add


### Bug Fixes

* **add:** add specific versions of a dataset using the full reference ([4b1e562](https://github.com/qri-io/qri/commit/4b1e562))
* **config:** `config.Copy()` copies over the path ([2fd3420](https://github.com/qri-io/qri/commit/2fd3420))
* **secrets:** Write secrets warning to stderr instead of stdout ([b43a4b7](https://github.com/qri-io/qri/commit/b43a4b7))
* **unpublish:** Add endline when printing message ([#810](https://github.com/qri-io/qri/issues/810)) ([0ced201](https://github.com/qri-io/qri/commit/0ced201))
* **validate:** Allow validate to run when connected ([#808](https://github.com/qri-io/qri/issues/808)) ([dedc0c5](https://github.com/qri-io/qri/commit/dedc0c5))
* **webapp:** resolve webapp path before passing path to the handler ([f38dcf4](https://github.com/qri-io/qri/commit/f38dcf4))
* **webapp loading:** pin webapp from dweb, send temporary script if loading ([4fb2921](https://github.com/qri-io/qri/commit/4fb2921))


### Code Refactoring

* **cmd:** remove ephemeral `qri connect` flags ([8620aa0](https://github.com/qri-io/qri/commit/8620aa0))


### Features

* **pretty:** Pretty flag for get command ([b6f579b](https://github.com/qri-io/qri/commit/b6f579b))


### BREAKING CHANGES

* **cmd:** removes:

`--disconnect-after`
`--disable-api`
`--disable-rpc`
`--disable-webapp`
`--disable-p2p`
`--read-only`
`--remote-mode`

These can be configured instead by using the `qri config` command

Removing these temporary flags let's us better reason about the state of the config at any time, as well as helps us be confident that at any time we are not saving over our config with false information.



<a name="v0.8.1"></a>
# [v0.8.1](https://github.com/qri-io/qri/compare/v0.8.0...v) (2019-06-11)

This patch release fixes a small-but-critical bug that prevented `qri setup` from working. A few other fixes & bumps made it in, but the main goal was restoring `qri setup` so folks can, you know, set qri up.

### Bug Fixes

* **config:** lib.NewInstance option func must check for nil pointers ([69537ce](https://github.com/qri-io/qri/commit/69537ce))
* **lib/diff:** adjust deepdiff.Diff params ([62a13eb](https://github.com/qri-io/qri/commit/62a13eb))
* **setup:** load plugins before attempting to setup IPFS ([#795](https://github.com/qri-io/qri/issues/795)) ([69c5fda](https://github.com/qri-io/qri/commit/69c5fda))
* **startf:** bump version number to 0.8.1 ([e27466a](https://github.com/qri-io/qri/commit/e27466a))


### Features

* **diff:** return source data in diff response ([d3eae83](https://github.com/qri-io/qri/commit/d3eae83))
* **diff:** return source data in diff response ([d1d2da5](https://github.com/qri-io/qri/commit/d1d2da5))



<a name="0.8.0"></a>
# [0.8.0](https://github.com/qri-io/qri/compare/v0.7.3...v0.8.0) (2019-06-05)

Version 0.8.0 is our best-effort to close out the first set of public features.

## Automatic Updates ([RFC0024](https://github.com/qri-io/rfcs/blob/master/text/0024-scheduled-updates.md))
Qri can now keep your data up to date for you.  0.8.0 overhauls `qri update` into a service that schedules & runs updates in the background on your computer. Qri runs datasets and maintains a log of changes.

### schedule shell scripts
Scheduling datasets that have starlark transforms is the ideal workflow in terms of portability, but a new set of use cases open by adding the capacity to schedule & execute shell scripts within the same cron environment. 

## Starlark changes
We've made two major changes, and one small API-breaking change. Bad news first:

### `ds.set_body` has different optional arguments
`ds.set_body(csv_string, raw=True, data_format="csv")` is now `ds.set_body(csv_string, parse_as="csv")`. We think think this makes more sense, and that the previous API was confusing enough that we needed to completely deprecate it. Any prior transform scripts that used `raw` or `data_format` arguments will need to update.

### new beautiful soup-like HTML package
Our `html` package is difficult to use, and we plan to deprecate it in a future release. In it's place we've introduced `bsoup`, a new package that implements parts of the [beautiful soup 4 api](https://www.crummy.com/software/BeautifulSoup/bs4). It's _much_ easier use, and will be familiar to anyone coming from the world of python.


### the "ds" passed to a transform is now the previous dataset version
The `ds` that's passed to is now the existing dataset, awaiting transformation. For technical reasons, `ds` used to be a blank dataset. In this version we've addressed those issues, which makes examining the current state a dataset possible without any extra `load_dataset` work. This  makes things like append-only datasets a one-liner:

```python
def transform(ds,ctx):
  ds.set_body(ds.get_body().append(["new row"]))
```

### CLI uses '$PAGER' on POSIX systems
Lots of Qri output is, well, long, so we now check for the presence of the `$PAGER` environment variable and use it to show "scrolling" data where appropriate. While we're at it we've cleaned up  output to make things a little more readable. Windows should be unaffected by this change. If you ever want to _avoid_ pagination, I find the easiest way to do so is by piping to `cat`. For example:
```
$ qri ls | cat
```
Happy paging!

### Switch to go modules
Our project has now switched entirely to using go modules. In the process we've deprecated `gx`, the distributed package manager we formerly used to fetch qri dependencies. This should dramatically simplify the process of building Qri from source by bringing dependency management into alignment with idiomatic go practices.

### Dataset Strict mode
`dataset.structure` has a new boolean field: `strict`. If `strict` is `true`, a dataset _must_ pass validation against the specified schema in order to save. When a dataset Dataset is in strict mode, Qri can assume that all data in the body is valid. Being able to make this assumption will allow us to provide additional functionality and performance speedups in the future. If your dataset has no errors, be sure to set `strict` to `true`.



### Bug Fixes

* **`doesCommandExist`:** fix to `exec.Command` ([4319db3](https://github.com/qri-io/qri/commit/4319db3))
* **`printToPager`:** different syntax needed for different systems ([86644a0](https://github.com/qri-io/qri/commit/86644a0))
* **api/export:** `/export/` api endpoint now sends data! ([7718b26](https://github.com/qri-io/qri/commit/7718b26))
* **api/list:** fix/add pagination for api `/list` endpoint ([6187c09](https://github.com/qri-io/qri/commit/6187c09))
* **base:** cron dataset command shouldn't use quotes in args ([d0568dc](https://github.com/qri-io/qri/commit/d0568dc))
* **base.DatasetLog:** fix limit and offset logic ([f7b042d](https://github.com/qri-io/qri/commit/f7b042d))
* **base.ListDataset:** add offset and limit to base.ListDataset ([dab742a](https://github.com/qri-io/qri/commit/dab742a))
* **base/log:** fix error that didn't check if PreviousPath exists before loading it ([748e58b](https://github.com/qri-io/qri/commit/748e58b))
* **canonicalize:** Correct peername when profileID matches ([affc3c3](https://github.com/qri-io/qri/commit/affc3c3))
* **cmd:** don't use FgWhite, breaks light-colored termimals ([32b8793](https://github.com/qri-io/qri/commit/32b8793))
* **cmd:** listing page 2 starts numbering at the offset ([b37d186](https://github.com/qri-io/qri/commit/b37d186))
* **cmd/get:** must catch `DatasetRequest` error ([f200acc](https://github.com/qri-io/qri/commit/f200acc))
* **config:** Check err when parsing config to avoid segfault ([f88e6e6](https://github.com/qri-io/qri/commit/f88e6e6))
* **config:** fix api.Copy not copying all fields ([4f30f9f](https://github.com/qri-io/qri/commit/4f30f9f))
* **config:** make setter actually write file ([12230e3](https://github.com/qri-io/qri/commit/12230e3))
* **connect:** use lib.NewInstance with qri connect ([3fc3de4](https://github.com/qri-io/qri/commit/3fc3de4))
* **cron:** connect cron to log file, actually run returned command in lib ([41c9366](https://github.com/qri-io/qri/commit/41c9366))
* **docs:** Building on windows, rpi, and brew install instructions ([8953591](https://github.com/qri-io/qri/commit/8953591))
* **events:** event PeerIDs serialize properly to strings ([7f4a23e](https://github.com/qri-io/qri/commit/7f4a23e))
* **fill.Struct:** If present, use field tag as the field name ([432da1d](https://github.com/qri-io/qri/commit/432da1d))
* **linux:** Ignore errors from setting rlimit, needs root on linux ([02108a9](https://github.com/qri-io/qri/commit/02108a9))
* **list:** qri list new command-line interface ([daf2b70](https://github.com/qri-io/qri/commit/daf2b70))
* **Makefile:** Require go version 1.11 ([0cc2f9b](https://github.com/qri-io/qri/commit/0cc2f9b))
* **p2p:** Improve comment for IPFSCoreAPI accessor ([aec34c5](https://github.com/qri-io/qri/commit/aec34c5))
* **p2p:** pass missing context to ipfsfs.GoOnline ([43a4b2e](https://github.com/qri-io/qri/commit/43a4b2e))
* **peers:** add default limit to peers, fix test ([fe8f6d8](https://github.com/qri-io/qri/commit/fe8f6d8))
* **ref:** CanonicalizeProfile just handles renames all the time ([be1736b](https://github.com/qri-io/qri/commit/be1736b))
* **render:** supply default 'html' format if none exists ([2b82b41](https://github.com/qri-io/qri/commit/2b82b41))
* **repo/test:** fix rebase mistake, func should be named `NewTestRepoWithHistory` ([a39a6ae](https://github.com/qri-io/qri/commit/a39a6ae))
* **transform:** Pass prev dataset to ExecScript ([6ff666c](https://github.com/qri-io/qri/commit/6ff666c))
* **update:** fix update run missing type assignment, absolutize paths ([11ec50c](https://github.com/qri-io/qri/commit/11ec50c))
* **vet:** fix go vet error ([8f46850](https://github.com/qri-io/qri/commit/8f46850))
* plumb context into network methods ([fb036e3](https://github.com/qri-io/qri/commit/fb036e3))


### Code Refactoring

* **cmd:** change `limit` and `offset` flags to `page` and `page-size` ([62d5da8](https://github.com/qri-io/qri/commit/62d5da8))
* **lib.AbsPath:** deprecate lib.AbsPath, use qfs.AbsPath ([112362a](https://github.com/qri-io/qri/commit/112362a))


### Features

* **/list/:** add `term` param to `/list/` api, for local dataset search ([f66f857](https://github.com/qri-io/qri/commit/f66f857))
* **api:** add update endpoints ([f5f2024](https://github.com/qri-io/qri/commit/f5f2024))
* **base/fill:** Overhaul error handling ([1deb967](https://github.com/qri-io/qri/commit/1deb967))
* **cmd:** add --repo and --ipfs-path flags ([1711f30](https://github.com/qri-io/qri/commit/1711f30))
* **cmd:** overhaul update command ([4218a28](https://github.com/qri-io/qri/commit/4218a28))
* **config:** add update configuration details ([5786d68](https://github.com/qri-io/qri/commit/5786d68))
* **cron:** add cron package for scheduling dataset updates ([7b38164](https://github.com/qri-io/qri/commit/7b38164))
* **cron:** add flatbuffer HTTP api server & client ([50388e5](https://github.com/qri-io/qri/commit/50388e5))
* **cron:** add incrementing job.RunNumber, refactor job field names ([5f6b403](https://github.com/qri-io/qri/commit/5f6b403))
* **cron:** store dataset SaveParams as job options ([df629a7](https://github.com/qri-io/qri/commit/df629a7))
* **cron:** store logs and files of stdout logs ([a9f3c52](https://github.com/qri-io/qri/commit/a9f3c52))
* **cron.FbStore:** store cron jobs as flatbuffers ([755186d](https://github.com/qri-io/qri/commit/755186d))
* **cron.file:** FileJobStore for saving jobs to a backing CBOR file ([9d4de77](https://github.com/qri-io/qri/commit/9d4de77))
* **fill.Struct:** Support interface{} fields, require map string keys ([46166d1](https://github.com/qri-io/qri/commit/46166d1))
* **instance:** instroduce Instance, deprecate global config ([53a8e4b](https://github.com/qri-io/qri/commit/53a8e4b))
* **lib:** sew initial cron service into lib ([5e8a871](https://github.com/qri-io/qri/commit/5e8a871))
* **peers:** fix limit/offset bugs, add sending output to `less` when listing peer cache ([2e623db](https://github.com/qri-io/qri/commit/2e623db))
* **update:** add update log command and api endpoints ([56c4840](https://github.com/qri-io/qri/commit/56c4840))
* **update:** build out update package API, use it ([3004fd8](https://github.com/qri-io/qri/commit/3004fd8))
* **update:** cleanup CLI output ([fa2ebad](https://github.com/qri-io/qri/commit/fa2ebad))
* **update:** detect and report 'no changes' on dataset jobs ([ae63512](https://github.com/qri-io/qri/commit/ae63512))
* **update:** experimental support for multi-repo updates with --use-repo ([9ae8d44](https://github.com/qri-io/qri/commit/9ae8d44))
* **update:** introduce update package & update daemon ([dc6066c](https://github.com/qri-io/qri/commit/dc6066c))
* **update:** support updates via shell scripts ([c94edd8](https://github.com/qri-io/qri/commit/c94edd8))


### BREAKING CHANGES

* **cron:** Field names of cron.Job have been refactored, which will break repo/update/logs.qfb and repo/update/jobs.qfb files. Delete them to fix.. This will only affect users who have been building from source between releases.
* **connect:** "qri connect" no longer has flags for setting port numbers
* **api:** /update endpoint is moved to /update/run
* **cmd:** On the cli, all `limit` and `offset` flags have been changed to `page` and `page-size`.
* **lib.AbsPath:** lib.AbsPath is removed, use github.com/qri-io/qfs.AbsPath instead



<a name="0.7.3"></a>
# [0.7.3](https://github.com/qri-io/qri/compare/v0.7.2...v0.7.3) (2019-04-03)

This release is all about 3 Rs:
* Rendering
* Remotes
* `load_dataset`

This release we've focused on improving dataset visualiing, setting the stage with better defaults and a cleaner API for creating custom viz. We think expressing dataset vizualiations as self-contained `html` makes Qri datasets an order of magnitude more useful, and can't wait for you to try it.

Along with the usual bug fixes, a few nice bonuses have landed, like supplying [multiple `--file` args](https://github.com/qri-io/qri/pull/718) to qri save to combine dataset input files, and `qri get rendered` to show rendered viz. Anyway, on to the big stuff:

### Default Rendering ([RFC0011](https://github.com/qri-io/rfcs/blob/master/text/0011-html_viz.md))
Whenever you create a new dataset version, Qri will now create a default viz component if you don't provide one. Unless run with `--no-render`, Qri will now execute that template, and store the result in a file called `index.html` in your dataset. This makes your dataset _much_ more fun when viewed directly on the d.web, which is outside of Qri entirely.

This is because IPFS HTTP gateways are sensitive to `index.html`. When you use qri to make a dataset, your dataset comes with a self-contained visualization that others can see without downloading Qri at all.

We think this dramatically increases the usefulness of a dataset, and increases the chances that others will want to share & disseminate your work by making your dataset a more-complete offering in the data value chain. These embedded default visualizations drop the time it takes to create a readable dataset to one step.

That being said, we've intentionally made the default visualization rather bland. The reason for this is twofold. First, to keep the file size of the `index.html` small (less than 1KB). Second, we want you to customize it. We'll refine the default template over time, but we hope you'll use viz to tell a story with your data.

Users may understandably want to disable default vizualizations. To achieve this `qri save` and `qri update` have a new flag: `--no-render`. No render will prevent the execution of any viz template. This will save ~1KB per version, at the cost of usability.

### Overhauled HTML Template API ([RFC0011](https://github.com/qri-io/rfcs/blob/master/text/0011-html_viz.md#template-api))
Keeping with the theme of better viz, we've also taken time to overhaul our template API. Given that this is a public API, we took some time to think about what it would mean to try to render Qri templates _outside_ of our go implementation. While no code does this today, we wanted to make sure it would be easier in the future, so we took steps to define an API that generally avoids use of the go templating `.`, instead presenting a `ds` object with json-case accessors. Taking things like this:

```
<h1>{{ .Meta.Title }}</h1>
```
to this: 
```
<h1>{{ ds.meta.title }}</h1>
```

This change brings the template syntax closer to the way we work with datasets in other places (eg: in `dataset.yaml` files and starlark transform scripts), which should help cut down on the mental overhead of working with a dataset in all these locations. We think this up-front work on our template API will make it easier to start writing custom templates. We don't have docs up yet, but the RFC [reference](https://github.com/qri-io/rfcs/blob/master/text/0011-html_viz.md#reference-level-explanation) section outlines the API in detail.

### Experimental Remote Mode ([RFC0022](https://github.com/qri-io/rfcs/blob/master/text/0022-remotes.md))
The registry is nice and all, but we need more ways to push data around. In this release we're launching a new expriment called "remotes" that start into this work. Remotes act as a way for any user of Qri to setup their own server that keeps datasets alive, providing availability and ownership over data within a set of nodes that they control.

Currently we consider this feature "advanced only" as it comes with a number of warnings and some special setup configuration. For more info, [check the RFC](https://github.com/qri-io/rfcs/blob/master/text/0022-remotes.md), and if you're interested in running a remote, [hop on discord](https://discordapp.com/invite/etap8Gb) and say "hey I want to run a remote".


### Starlark `load_dataset` ([RFC0023](https://github.com/qri-io/rfcs/blob/master/text/0023-starlark_load_dataset.md))
We've made a breaking API change in Starlark that deprecates `qri.load_dataset_body`, and introduce a new global function: `load_dataset`. This new API makes it clear that `load_dataset` both loads the dataset and declares it as a dependency of this script. This is an important step toward making datasets a first-class citizen in the qri ecosystem. Here's an example of the new syntax:

```python
load("http.star", "http")

# load a dataset into a variable named "fhv"
fhv = load_dataset("b5/nyc_for_hire_vehicles")

def download(ctx):
  # use the fhv dataset to inform an http request
  vins = ["%s,%s" % (entry['vin'], entry['model_yearl']), for entry in fhv.body()]

  res = http.post("https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVINValuesBatch/", form_body={
    'format': 'JSON', 
    'DATA': vins.join(";")
  })

  return res.json()

def transform(ds, ctx):
  ds.set_body(ctx.download)
```

Users who were previously using `qri.load_dataset_body` will need to update their scripts to use the new syntax. The easiest way to do that is by adding a new version to your dataset history with the updated script:
```
$ qri get transform.script me/dataset > transform.star
# make updates to transform.star file & save
$ qri save --file transform.script me/dataset
```
Three easy steps, and your dataset log tells the story of the upgrade.


### Bug Fixes

* **api:** Listen on localhost for API and RPC ([04a4500](https://github.com/qri-io/qri/commit/04a4500))
* **fill_struct:** Work with non-pointer structs and pointers to non-structs ([0d14bac](https://github.com/qri-io/qri/commit/0d14bac))
* **get:** If format is misspelled, display an error ([b8fcceb](https://github.com/qri-io/qri/commit/b8fcceb))
* **save:** Better error message when saving with wrong file ([75cb06a](https://github.com/qri-io/qri/commit/75cb06a))
* **tests:** Save with viz needs html format, update testdata file ([f7cd486](https://github.com/qri-io/qri/commit/f7cd486))


### Code Refactoring

* **render:** remove limit, offset, all parmeters ([165abce](https://github.com/qri-io/qri/commit/165abce))


### Features

* **cmd/get:** add `rendered` to selector in `qri get` command ([90719a8](https://github.com/qri-io/qri/commit/90719a8))
* **dag info:** add label for rendered viz output ([7157cd7](https://github.com/qri-io/qri/commit/7157cd7))
* **daginfo:** add `qri daginfo` command that returns a dag.Info of a dataset ([e4d9e27](https://github.com/qri-io/qri/commit/e4d9e27))
* **daginfo:** add summary view for `qri daginfo` ([9f9b32f](https://github.com/qri-io/qri/commit/9f9b32f))
* **dagInfo:** add ability to get a subDag at a specific label ([7a07ee8](https://github.com/qri-io/qri/commit/7a07ee8))
* **remote:** Beginning of "remote" mode implementation ([49dbef9](https://github.com/qri-io/qri/commit/49dbef9))
* **remote:** Bump version number, review fixes. ([21a4448](https://github.com/qri-io/qri/commit/21a4448))
* **remote:** Deserialize response from remote using json.Decode ([2d73e84](https://github.com/qri-io/qri/commit/2d73e84))
* **remote:** Parse config using FillStruct. Remotes field. ([a2efbf3](https://github.com/qri-io/qri/commit/a2efbf3))
* **remote:** Retrieve sessionID & diff first, pass both to dsync ([7f3d8bc](https://github.com/qri-io/qri/commit/7f3d8bc))
* **remote:** Send actual dag.Info to remote, perform dsync ([5cec2de](https://github.com/qri-io/qri/commit/5cec2de))
* **remote:** Switch to --remote-accept-size-max flag ([b26cbb9](https://github.com/qri-io/qri/commit/b26cbb9))
* **remotes:** base/fill/PathValue used by Config, to enable Remotes ([4d45dce](https://github.com/qri-io/qri/commit/4d45dce))
* **remotes:** Complete dsync by writing to ds_refs and pinning ([b47a3e4](https://github.com/qri-io/qri/commit/b47a3e4))
* **save:** add `no-render` option to `qri save` command ([a8f36b2](https://github.com/qri-io/qri/commit/a8f36b2))
* **save:** More tests for saving: viz, transform. Update comments. ([91246fe](https://github.com/qri-io/qri/commit/91246fe))
* **save:** Save a dataset with multiple file arguments ([23735b7](https://github.com/qri-io/qri/commit/23735b7))
* **save:** Tests for saving datasets with multiple file arguments ([9fe54bc](https://github.com/qri-io/qri/commit/9fe54bc))


### BREAKING CHANGES

* **render:** qri render limit, offset, all parameters have been removed



<a name="0.7.2"></a>
# [0.7.2](https://github.com/qri-io/qri/compare/v0.7.1...v0.7.2) (2019-03-14)


### Better piping for `qri save`
Minor set of fixes aimed at making the command line work better. The biggest command we've fixed is `qri save --dry-run`. Qri now properly routes diagnostic output to `stderr` (things like the spinner, transform exectution, and completion messages), while sending the _result_ of the dry run to `stdout`. This makes the following much more powerful:

`qri save --dry-run --file transform.star me/dataset > dry_run_output.json`

If save works, the `dry_run_output.json` will now be a valid `json` file representing the result of the dry run, and you'll still see progress output in your terminal window. We'll be working to bring this separation of diagnostic & results output to all commands in the near future.

### Run transforms directly with the `--file` flag
One other small note, if you feed a file that ends in `.star` to `qri save --file`, Qri will assume you mean it to be a transform script, and execute it as such. This cuts out the need for a `dataset.yaml` file, and makes working with transforms a little easier to reason about. If you need to provide configuration details to the transform, you'll still need to create a `dataset.yaml` file & specify your `.star` file there.


### Bug Fixes

* **get:** better defaults for get body. ([5631d08](https://github.com/qri-io/qri/commit/5631d08))
* **stderr:** write save diagnostic output to stderr ([3dac8b8](https://github.com/qri-io/qri/commit/3dac8b8))


### Features

* **ReadDatasetFile:** infer .star & .html files to be bare tf & vz components ([89fcba8](https://github.com/qri-io/qri/commit/89fcba8))


### BREAKING CHANGES

* **get:** cli body command is now removed



<a name="0.7.1"></a>
# [0.7.1](https://github.com/qri-io/qri/compare/v0.7.0...v0.7.1) (2019-03-07)

0.7.1 is on the larger side for a patch release, shipping fixes & features aimed at doing more with datasets. With this release qri is better at comparing data, and more flexible when it comes to both saving & exporting data.

### Overhauled Diff
We've completely overhauled our `diff` command, using a new algorithm that's capable of diffing large volumes of structured data quickly. We've also adjusted the command itself to work with structured data both inside and outside of qri.

Two caveats come with this new feature:
* This code is new. We have tests that show our differ produces _valid_ diff scripts, but sometimes diff doesn't produce diffs that are fun to read. We'd love your feedback on improving the new diff command.
* We haven't really figured out a proper way to _visualize_ these diffs yet.

Caveats aside, new diff is mad decent. Here's a few examples:

```
# get more info on diff:
$ qri diff --help

# diff dataset body against it's previous version:
$ qri diff body me/annual_pop

# diff two dataset meta sections:
$ qri diff meta me/population_2016 me/population_2017

# diff two local json files:
$ qri diff a.json b.json

# diff a json & a csv file:
$ qri diff data.csv b.json

# output diff as json
$ qri diff a.json b.json --format json

# print just a diff summary
$ qri diff me/annual_pop --summary
```

### Revamped Export
We've overhauled our export command to up it's utility. We've also introduced the terms "foreign" and "native" export to refer to the _fidelity_ of an export. This round of improvements has focused on foreign exports, which converts qri datasets into common formats. For example, now you can (finally) express a complete dataset as json:

```
$ qri export --format json me/annual_pop
```

We've also made the role `qri export` plays work better in relation to `qri get`. If you want a _complete_ dataset, use `qri export`. If you want a piece of a dataset (for example, just the body), use `qri dataset`.


### Shorthand Save Files

We've added a way to save changes to datasets that only affect individual components. If you only want to change, say the meta section of a dataset, you can now create a file that only affects the meta component. In the past,  you'd have no choice but to construct a complete dataset document, and only alter the `meta` section. Instead you can now save a file like this:

```json
{
  "qri": "md:0", // this "md:0" value for the "qri" key tells qri this only affects meta
  "title" : "this is a new title",
  "description": "new decription"
}
```

and give that to save:
`$ qri save --file meta.json me/dataset`

Qri will interpret this file and apply the changes to the meta component. This pairs very nicely with `qri get` to make workflows that only affect specific components, you could use this to change a dataset's schema, which lives in the `structure` component:
```
$ qri get structure me/annual_pop  > structure.yaml

# save changes to structure.yaml

$ qri save --file structure.yaml me/annual_pop
```

### Starlark improvements:
We've added a `json` package to starlark, and given `math` a new method `round()`, which is not built into starlark as a language primtive. If you want to round numbers in starlark scripts, do this:

```python
load("math.star", "math")

print(math.round(3.5)) # prints: 4
```

### Fixes
We've also shipped a bunch of little bug fixes that should add up to a better experience. All the changes described above also apply to the qri JSON api as well. Happy versioning!


### Bug Fixes

* **actions.SaveDataset:** delay inferring values until after writing changes ([86f57fa](https://github.com/qri-io/qri/commit/86f57fa))
* **api:** restore api /diff endpoint ([e71fd8e](https://github.com/qri-io/qri/commit/e71fd8e))
* **base.ReadEntries:** Simplify ReadEntries by using PagedReader in render ([4f914a6](https://github.com/qri-io/qri/commit/4f914a6))
* **cli persistent flags:** properly register global flags ([f4f1ed7](https://github.com/qri-io/qri/commit/f4f1ed7)), closes [#506](https://github.com/qri-io/qri/issues/506)
* **export:** Export --zip flag ([8514c76](https://github.com/qri-io/qri/commit/8514c76))
* **export:** Overhaul export command ([f7df3f9](https://github.com/qri-io/qri/commit/f7df3f9))
* **export:** Test cases for export ([12a279f](https://github.com/qri-io/qri/commit/12a279f))
* **export:** Update api zip handler, and cmd integration test ([82cd4cc](https://github.com/qri-io/qri/commit/82cd4cc))
* **get:** Apply format to get when using a selector ([ed1fd31](https://github.com/qri-io/qri/commit/ed1fd31))
* **get:** Get returns 404 if dataset is not found ([291296b](https://github.com/qri-io/qri/commit/291296b))
* **get dataset:** return a datasetRef, not a dataset! ([6e51a31](https://github.com/qri-io/qri/commit/6e51a31))
* **info:** Remove the info command, list does it now ([ef34a05](https://github.com/qri-io/qri/commit/ef34a05))
* **list:** Flag --num-versions shows number of versions of each dataset ([2f5a8b1](https://github.com/qri-io/qri/commit/2f5a8b1))
* **list:** Improve usability of `list` command ([d773363](https://github.com/qri-io/qri/commit/d773363))
* **list:** Rename lib parameter to ShowNumVersions ([7214f93](https://github.com/qri-io/qri/commit/7214f93))


### Features

* **cmd:** add summary flag to diff command ([f793c0f](https://github.com/qri-io/qri/commit/f793c0f))
* **diff:** added diffStat string, support for diffing files ([2d16df5](https://github.com/qri-io/qri/commit/2d16df5))
* **diff:** overhauling diff to use difff ([b825e65](https://github.com/qri-io/qri/commit/b825e65))
* **fill_struct:** Additional tests, cover some edge-cases ([e1454de](https://github.com/qri-io/qri/commit/e1454de))
* **fill_struct:** Fix qri key, multiple small fixes ([71d78e7](https://github.com/qri-io/qri/commit/71d78e7))
* **fill_struct:** Json and yaml deserialization rewrite ([ace2c1e](https://github.com/qri-io/qri/commit/ace2c1e))
* **fill_struct:** Rename to SetArbitrary ([990ad24](https://github.com/qri-io/qri/commit/990ad24))
* **fill_struct:** Support bool, float, slices ([314be2a](https://github.com/qri-io/qri/commit/314be2a))
* **lib.Datasets.Save:** add force flag to skip empty-commit checking ([6fd5b1f](https://github.com/qri-io/qri/commit/6fd5b1f))



<a name="0.7.0"></a>
# [0.7.0](https://github.com/qri-io/qri/compare/v0.6.2...v0.7.0) (2019-02-05)

We've bumped this release to 0.7.0 to reflect some major-league refactoring going on deeper in the stack. The main goal of this release has been to drive stability up, and put the Qri codebase on firmer foundation for refinement.

### Major Stability Improvements
Much of our refactoring has gone into removing & consolidating code, making it easier reason about. The effect of this are fewer unintended interactions between different subsystems, resulting in a version of qri that behaves much better, especially in the corner cases. All of these improvements are made on both the JSON API, the CLI, _and_ while operating over RPC (we use RPC when `qri connect` is running in another terminal).
 
Commands like `qri get` and `qri export` in particular work consistently, and can be used to greater effect. For example, running `qri use` on a dataset let's you drop the repeated typing of a dataset name, and play with `get` to explore the dataset faster:
```
$ qri use me/dataset
$ qri get meta --format yaml
$ qri get body --format json --limit 20 --offset 3
```

It's much to fetch scripts with `get` and write them to a file for local work:
```
$ qri get transform.script > transform.star
```

Speaking of transforms, `qri update` on a local dataset is now true alias for `qri save --recall=tf`. Eliminating the alternate codepath for `qri update` has made update work far better for re-running transforms.

Export now has a far friendlier `--format` flag for getting a dataset document into a common data format. This'll give you a JSON interpretation of your dataset:
```
qri export --format=json me/dataset
```

We're inching closer to our overall goal of building Qri into a series of well-composed packages with a cohesive user interface. Lots more work to do, partly because there's more that we _can_ do now that our code is better composed. We'd encourage you to play with the CLI if you haven't yet taken it for a spin.

### XLSX support
Along with the improved export command, we now have early support for Excel `.xlsx` documents! this now works:
```
qri export --format=xlsx peer/dataset
``` 

this now works too (so long as there's a sheet named "sheet1"), and an existing history in another format like JSON, CSV, or CBOR:
```
qri save --body dataset.xlsx --keep-format peer/dataset
```
Lots of work to do here, but early support for excel is an exciting addition.


### Bug Fixes

* **get:** Datasets whose name contains a field should work with get ([97410fb](https://github.com/qri-io/qri/commit/97410fb))
* **get:** Fix get command using a dotted path ([c130349](https://github.com/qri-io/qri/commit/c130349))
* **get:** Turn `body` into a selector of the `get` command ([7793845](https://github.com/qri-io/qri/commit/7793845))
* **local update:** rework local update to be a type of save ([fa9ca11](https://github.com/qri-io/qri/commit/fa9ca11))
* **save:** Improve error message if new ds has no body or structure ([31332fd](https://github.com/qri-io/qri/commit/31332fd))
* **save:** Infer more values, such as Schema, when appropriate ([1c95539](https://github.com/qri-io/qri/commit/1c95539))


### Features

* **export:** initial foreign export support ([9382483](https://github.com/qri-io/qri/commit/9382483))
* **get:** support getting script file fields ([da8ae46](https://github.com/qri-io/qri/commit/da8ae46))



<a name="0.6.2"></a>
# [0.6.2](https://github.com/qri-io/qri/compare/v0.6.1...v0.6.2) (2019-01-22)

0.6.2 is mainly about squashing bugs, but there are three fun new features worth noting:

### :globe_with_meridians: IPFS HTTP API Support
Given that all Qri nodes currently run IPFS under the hood, it's sad that Qri doesn't expose the full power of IPFS while running. We've fixed that by adding new configuration options to enable the IPFS HTTP API. This lets you use the `ipfs` binary to use IPFS while qri is running. We've added support for toggling IPFS pubsub as well. By default new installations of Qri come with the IPFS api enabled, and pubsub disabled.

To enable both, change your Qri config.yaml `store` to include the new options flags:

```yaml
store:
  type: "ipfs"
  options:
    # when api is true, Qri will use the IPFS config to spin up an HTTP API
    # at whatever port your IPFS repo is configured for
    # (the configuration in IPFS is "Addresses.API", defaults to localhost:5001)
    api: true
    # optional, when true running "qri connect" will also enable IPFS pubsub 
    # as if `ipfs daemon --enable-pubsub-experiment` were run
    pubsub: true
```

With these settings you can run `qri connect` in one terminal, and then run (nearly) any regular IPFS command in another terminal with `ipfs --api /ip4/127.0.0.1/tcp/5001/ [command]`. Having both IPFS & Qri work at once makes life much easier for anyone who needs access to both functionalities at once.

### :world_map: Starlark Geo Support
We're very excited to land initial support for a geospatial module in Starlark. Considering just how much open data is geospatial, we're very excited to play with this :) [Here's an example transform script](https://gist.github.com/b5/7fc378b6ee504e929ab390bca8f038f6) to get started.

### :1234: Remove now supports any number of revisions
Previously, it was only possible to reomve _entire dataset histories_, which is, well, not ideal. We've changed the way remove works so you can now choose how many revisions (dataset versions) to delete. `qri remove --revisions 1 me/dataset_name` is basically an undo button, while `qri remove --all me/dataset_name` will delete the entire dataset. This also applies to the API, by supplying a now-required `revisions` query param.



### Bug Fixes

* **cmd:** Both body and setup runnable during "qri connect" ([0a9c93a](https://github.com/qri-io/qri/commit/0a9c93a))
* **cmd.Export:** handle -o flag on export to get target directory ([0064715](https://github.com/qri-io/qri/commit/0064715))
* **export:** Check for unsupported export flags, recognize --format ([aea27de](https://github.com/qri-io/qri/commit/aea27de))
* **export:** Export can be used while running `qri connect` ([a5a56d1](https://github.com/qri-io/qri/commit/a5a56d1))
* **export:** Flag is --zip instead of --zipped, api format form value ([f370632](https://github.com/qri-io/qri/commit/f370632))
* **export:** Write exported file to pwd where command is run. ([b62f739](https://github.com/qri-io/qri/commit/b62f739))
* **list:** Listing a non-existent profile should not crash ([e00d516](https://github.com/qri-io/qri/commit/e00d516))
* **p2p dataset info:** return not found if dataset doesn't populate ([b55af2a](https://github.com/qri-io/qri/commit/b55af2a))
* **print:** Use int64 in print in order to support Arm6 (Raspberry Pi). ([64fcbc4](https://github.com/qri-io/qri/commit/64fcbc4))
* **remove:** Flag --all as an alias for --revisions=all. More tests ([7e29002](https://github.com/qri-io/qri/commit/7e29002))
* **remove:** Remove requires the --revisions to specify what to delete ([02a0b02](https://github.com/qri-io/qri/commit/02a0b02))
* **Save:** add tests to cmd that test you can save a transform and viz ([#649](https://github.com/qri-io/qri/issues/649)) ([42410a5](https://github.com/qri-io/qri/commit/42410a5))


### Features

* **api:** always include script output in POST /dataset responses ([ee32e44](https://github.com/qri-io/qri/commit/ee32e44))
* **config.Store:** support store options to enable ipfs api & pubsub ([e76b9f8](https://github.com/qri-io/qri/commit/e76b9f8)), closes [#162](https://github.com/qri-io/qri/issues/162) [#658](https://github.com/qri-io/qri/issues/658)



<a name="0.6.1"></a>
# [0.6.1](https://github.com/qri-io/qri/compare/v0.6.0...v0.6.1) (2018-12-12)

For a patch release, 0.6.1 is on the larger side. After cutting the 0.6.0 release, we realized that NAT traversal issues were preventing Qri from working in a consistent, performant way. Basically, peers aren't seeing each other, and despite tools for building & maintaining direct connections with Qri peers, data transfer isn't performing in a way that's comparable with "git+github". To compensate for this, we're focusing on making publishing to the registry consistent & reliable, and using the registry to reduce our reliance on p2p connections. The result is Qri has a fast path that works with the registry for many dataset transfer commands, but still uses p2p whenever possible.

In the coming months we'll continue to iterate these solutions toward each other, making our p2p technology faster & more reliable, while keeping these "fast-paths" for point-to-point dataset transfer.

0.6.1 No breaking changes, but some behaviours are different. The biggie is `qri publish` now takes a few extra steps.

### :racing_car: Faster, dependable Publish & Add
Our big focus has been on making dataset transfer to the registry fast & reliable. The end result is `qri publish` and `qri add` now feels a _lot_ more like `git push` and `git fetch`. We're using our shiny new [dag package](https://github.com/qri-io/dag) to do a transfer of raw IPFS blocks directly. Look for us to expand & improve the use of this (open source) secret sauce for a more performant Qri experience.

### :mag: Search is restored!
`qri search` is back! It works with the registry! Try running `qri search foo` and `qri add` with one of the results!

### Bug Fixes

* **actions.ResolveDatasetRef:** temp fix to keep incomplete ref responses from resolve ([b284228](https://github.com/qri-io/qri/commit/b284228))
* **build:** move gx-dep packages around ([f85b432](https://github.com/qri-io/qri/commit/f85b432))
* **config migrate:** fix crash in migration ([2c4396e](https://github.com/qri-io/qri/commit/2c4396e))
* **config migration:** add config migration to update p2p.QriBootstrapAddrs ([bc9bfcd](https://github.com/qri-io/qri/commit/bc9bfcd))
* **connect output:** suppress RPC error messages, clarify connection message ([934f11c](https://github.com/qri-io/qri/commit/934f11c)), closes [#623](https://github.com/qri-io/qri/issues/623)
* **format:** Detect format change when saving, either error or rewrite ([137e18b](https://github.com/qri-io/qri/commit/137e18b))
* **format:** Tests for ConvertBodyFormat, many small cleanups. ([be4a32c](https://github.com/qri-io/qri/commit/be4a32c))
* **p2p:** bump qriSupportValue from 1 to 100 ([4159818](https://github.com/qri-io/qri/commit/4159818))
* **publish:**  api publish endpoint and CanonicalizeDatasetRef fix ([75077cf](https://github.com/qri-io/qri/commit/75077cf))
* **ResolveRef:** fix improper IDs form registry resolves, cleanup search CLI printing ([115350c](https://github.com/qri-io/qri/commit/115350c))
* **Save:** fix saving dataset with viz stored in cafs ([bc2b00a](https://github.com/qri-io/qri/commit/bc2b00a))
* **secrets:** move secrets out-of-band from dataset ([a258a23](https://github.com/qri-io/qri/commit/a258a23)), closes [#609](https://github.com/qri-io/qri/issues/609)
* **transform:** Fix how secrets and config are passed into transform. ([179f31c](https://github.com/qri-io/qri/commit/179f31c))
* **webapp:** restore webapp by serving compiled app as a directory ([ad5e2be](https://github.com/qri-io/qri/commit/ad5e2be))


### Features

* **api.registry:** create `/registry/list` endpoint that returns a list of datasets avail on the registry ([868bd5d](https://github.com/qri-io/qri/commit/868bd5d))
* **bsync:** initial block-sync sketched out ([6718e69](https://github.com/qri-io/qri/commit/6718e69))
* **bsync:** initial Receivers implementation, HTTP support ([2e986f6](https://github.com/qri-io/qri/commit/2e986f6))
* **bsync:** initial work on bsync ([f3c37c6](https://github.com/qri-io/qri/commit/f3c37c6))
* **cmd registry pin:** add registry pin commands ([2293275](https://github.com/qri-io/qri/commit/2293275))
* **manifest:** generate a manifest for a given reference ([e3e52ac](https://github.com/qri-io/qri/commit/e3e52ac))
* **manifest:** initial DAGInfo, move Completion into manifest ([2c206de](https://github.com/qri-io/qri/commit/2c206de))
* **online:** Set Online flag for peers when using lib.Info. Cleanups. ([4542056](https://github.com/qri-io/qri/commit/4542056)), closes [#577](https://github.com/qri-io/qri/issues/577)
* **RegistryList:** get a list of datasets from a registry ([708c554](https://github.com/qri-io/qri/commit/708c554))
* **RegistryRequests.List:** wrap actions.RegistryList at the lib level ([e18f664](https://github.com/qri-io/qri/commit/e18f664))



<a name="0.6.0"></a>
# [0.6.0](https://github.com/qri-io/qri/compare/v0.5.6...v0.6.0) (2018-11-09)

Version 0.6.0 is a **big** 'ol release. Lots of changes that have taken close to a month to land. Test coverage is up, but expect us to have broken a bunch of things. We'll be rapidly iterating over the coming weeks to make sure everything works the way it should.

This release marks turning a corner for the Qri project as a whole. We have a new look for the frontend, and have rounded out an initial feature set we think will take Qri out of the realm of "experimental" and into the world of dependable, usable code. It's taken a great deal of time, effort, and research. I'm _very_ thankful/proud/excited for all who have contributed to this release, and can't wait to start showing off this newest verion. Here's some of the highlights, with a full changelog below.

### :heart: We've adopted the RFC process
I'm delighted to say Qri's feature development is now driven by a request-for-comments process. You can read about new features we're considering and implementing, as well as make suggestions over at our RFC repo. From this release forward, we'll note the biggest RFCs that have landed in release notes.


### Overhauled, more capable starlark transform functions, renamed everything from "skylark" to "starlark" [(RFC0016)](https://github.com/qri-io/rfcs/blob/master/text/0016-revise_transform_processing.md)
We've overhauled the way transform functions are called to provide greater flexibility between function calls to allow sidecar state to travel from special functions, and unified all transforms around the `transform` special function. here's an example:

```python
load("http.star", "http")
def download(ctx):
  res = http.get("https://api.github.com/")
  return res.json()

def transform(ds, ctx):
  ds.set_body(ctx.download)
```

The passed in dataset is now properly set to the most recent snapshot thanks to our improved transform logic. More on that below.

### `.zip` export & import [(RFC0014)](https://github.com/qri-io/rfcs/blob/master/text/0014-export.md)
The following now works:
```
$ qri export me/my_dataset
exported my_dataset.zip

# on some other qri repo:
$ qri save --file=my_dataset.zip
```
This makes it possible to send around Qri datasets without using the p2p network. It even works when the zip is posted to a url:
```
$ qri save --file=https://dropbox.com/.../my_dataset.zip
```

We think this is a big win for portability. We'll be working on exporting to different file formats in the coming weeks.


### Publish & Update [(RFC0018)](https://github.com/qri-io/rfcs/blob/master/text/0018-publish-update.md)
Qri now gives you control over which datasets of your will be listed for others to see using `qri publish`. This does mean you need to publish a datset before others will see it listed. **This does not mean that data added to Qri is private**. It's better to think of data you've added to qri that isn't published as 'unlisted'. If you gave someone the hash of your dataset, they could add it with `qri add`, but users who list your datasets over p2p won't see your unlisted work. If you want private data, for now Qri isn't the right tool, but now you have more control over what users see when they visit your profile.

We now also have our first real mechanism for automated synchronization: update. Update works on both your own datasets and other people's. Running update on your own dataset will re-run the most recent transform and generate a new dataset version if Qri detects a change. Running update on a peer's dataset will check to see if they're online, and if they are, update will fast-forward your copy of their dataset to the latest version.

### New and Save have merged [(RFC0017)](https://github.com/qri-io/rfcs/blob/master/text/0017-define_dataset_creation.md)
The `new` and `save` commands (and API endpoints) have merged into just `save`. New wasn't doing too much for us, so we're hoping to get down to one keyword for all modifications that aren't re-running transform scripts.

### Deterministic Transforms & Overhauled Dataset Creation [(RFC0020)](https://github.com/qri-io/rfcs/blob/master/text/0020-distingush_manual_vs_scripted_transforms.md)
We've completely overhauled the process of saving a datset, clarifying the mental model by distinguishing between _manual_ and _scripted_ transformations. The process of creating a dataset is now easier to understand and predict. We'll be working in the coming weeks to properly document how this works, but the first form of documentation we've landed are error messages that help clarify when an issue arises.

### Bug Fixes

* **actions.SaveDataset:** ensure both manual & scripted transforms are provided previous dataset ([902183d](https://github.com/qri-io/qri/commit/902183d))
* **actions.SaveDataset:** load transform from store if cafs scriptPath provided ([274fa76](https://github.com/qri-io/qri/commit/274fa76))
* **actions.Update:** properly set PrevPath before saving update ([eafe8b5](https://github.com/qri-io/qri/commit/eafe8b5))
* **bootstrap peers:** fix network not ever contacting bootstrap peers ([f9074b0](https://github.com/qri-io/qri/commit/f9074b0))
* **fs Profile Store:** fix lock pass-by-value error ([a22515a](https://github.com/qri-io/qri/commit/a22515a))
* **fs refstore:** don't store DatasetPod in refstore ([6ab5849](https://github.com/qri-io/qri/commit/6ab5849))
* **lib.AbsPath:** allow directories named 'http' in filepaths ([684d3be](https://github.com/qri-io/qri/commit/684d3be))
* **qri:** Fix usage of dataset and startf ([b754c06](https://github.com/qri-io/qri/commit/b754c06))
* **racy p2p tests:** don't pass locks by value in MemStore ([5e410a3](https://github.com/qri-io/qri/commit/5e410a3))
* **save:** ensure transforms are run when importing a zip ([a35523c](https://github.com/qri-io/qri/commit/a35523c))


### Features

* **actions.UpdateDataset:** support for local dataset updates ([f402956](https://github.com/qri-io/qri/commit/f402956))
* **api /publish:** add publish API endpoint ([502188e](https://github.com/qri-io/qri/commit/502188e))
* **api /publish:** add publish API endpoint ([3d1b4bf](https://github.com/qri-io/qri/commit/3d1b4bf))
* **api published:** GET /publish/ now lists published datasets ([6c8f0b2](https://github.com/qri-io/qri/commit/6c8f0b2))
* **api published:** GET /publish/ now lists published datasets ([9e60af6](https://github.com/qri-io/qri/commit/9e60af6))
* **api update:** add update API endpoint ([cb66b36](https://github.com/qri-io/qri/commit/cb66b36))
* **api.Update:** accept dry_run parameter ([642e7fa](https://github.com/qri-io/qri/commit/642e7fa))
* **base:** defining base package ([1a7cd42](https://github.com/qri-io/qri/commit/1a7cd42))
* **base.LogDiff:** diff logs against a repo's dataset history ([bd7340a](https://github.com/qri-io/qri/commit/bd7340a))
* **cmd publish:** add publish command, publish-on-save flag ([1bfcd92](https://github.com/qri-io/qri/commit/1bfcd92))
* **cmd publish:** add publish command, publish-on-save flag ([327f697](https://github.com/qri-io/qri/commit/327f697))
* **cmd.Save, cmd.Update:** add recall flags to save and update ([083331d](https://github.com/qri-io/qri/commit/083331d))
* **cmd.Update:** add update command ([03c2630](https://github.com/qri-io/qri/commit/03c2630))
* **fsrepo:** add support for storing 'Published' field in fsrepo ([837edeb](https://github.com/qri-io/qri/commit/837edeb))
* **mutal exclusion tf:** error if two transform types affect the same component ([fd439cb](https://github.com/qri-io/qri/commit/fd439cb))
* **p2p update:** initial p2p update ([76e886a](https://github.com/qri-io/qri/commit/76e886a))
* **p2p.test:** add `TestableNode` struct that satisfies the `TestablePeerNode` interface ([2c53a49](https://github.com/qri-io/qri/commit/2c53a49))
* **published:** datasets now have a publish flag ([84019b7](https://github.com/qri-io/qri/commit/84019b7))
* **published:** datasets now have a publish flag ([11db984](https://github.com/qri-io/qri/commit/11db984))
* **rev:** add rev package, add LoadRevs to base ([00ca81c](https://github.com/qri-io/qri/commit/00ca81c))
* **save:** add support for saving from file archives ([00a30d3](https://github.com/qri-io/qri/commit/00a30d3))
* **save:** Don't leak file paths across API. ([57a1c36](https://github.com/qri-io/qri/commit/57a1c36))
* **save:** Merge command `new` into `save`. ([33c4da6](https://github.com/qri-io/qri/commit/33c4da6))
* **save:** Remove new api and cmd. Document plan for actions. ([4919503](https://github.com/qri-io/qri/commit/4919503))
* **starlark:** renamed, reworked starlark syntax & transforms ([4e9c6fd](https://github.com/qri-io/qri/commit/4e9c6fd))
* **zip:** Import a zip file using save on the command-line ([bbb989a](https://github.com/qri-io/qri/commit/bbb989a))



<a name="0.5.6"></a>
# [0.5.6](https://github.com/qri-io/qri/compare/v0.5.5...v0.5.6) (2018-10-10)

### :two_women_holding_hands: Peer Sharing Fixes & Slightly better export

This patch release is focused on making inroads on some long-standing issues with peer discovery. We've added tests & a few fixes that should help peers get connected and stay connected. This is an area of active work that we'll be adding more improvements to in the future.

We've also standardized our export format with a newly-approved [export rfc](https://github.com/qri-io/rfcs/blob/master/text/0014-export.md), which fixes a few issues with the way Qri exports .zip archives. exports should now be complete archives that we'll start using for import in a forthcoming release.

### Bug Fixes

* **export:** Blank yaml file should have correct fields ([15484c0](https://github.com/qri-io/qri/commit/15484c0))
* **new:** Take absolute path to body file before loading it in `new`. ([b9afcd0](https://github.com/qri-io/qri/commit/b9afcd0))
* **p2p:** fix concurrent writes to repo Profile store ([3cf6f5c](https://github.com/qri-io/qri/commit/3cf6f5c))
* **p2p.QriConnectePeers:** properly tag QriPeers on both ends of the ConnManager ([14d473f](https://github.com/qri-io/qri/commit/14d473f))
* **peer sharing:** fix peers not sharing info with each other ([8219bab](https://github.com/qri-io/qri/commit/8219bab)), closes [#510](https://github.com/qri-io/qri/issues/510)


### Features

* **api:** Unpack endpoint gets a zip and sends back its contents ([185c01a](https://github.com/qri-io/qri/commit/185c01a))
* **api:** Wrap unpack response in api envelope ([aa5df03](https://github.com/qri-io/qri/commit/aa5df03))
* **export:** Always export by zip for now, until full RFC is implemented. ([9bc77b0](https://github.com/qri-io/qri/commit/9bc77b0))
* **export:** Test case for unpack endpoint. ([54aa3ee](https://github.com/qri-io/qri/commit/54aa3ee))



<a name="0.5.5"></a>
# [0.5.5](https://github.com/qri-io/qri/compare/v0.5.4...v0.5.5) (2018-10-05)

Version 0.5.5 is a patch release with some small features and a few bugfixes. It's mainly here because @b5 wants
to play with .zip files & regexes in transform scripts.

### :package: Skylark `re` and `zip` packages
We've added two new small, bare-bones packages to skylark to handle common-yet-vital use cases:
`re` brings basic support regular expressions, and `zip` brings read-only capacity to open zip archives.
Both of these are rather utility-oriented, but _very_ importnat when opening & cleaning data.

### :twisted_rightwards_arrows: Upcoming switch from "skylark" to "starlark"
Speaking of skylark, google has landed on a rename for their project, and it'll hence-fourth be named "starlark".
As such we'll be making the switch to this terminology in an upcoming release. Our package names will be changing
from `.sky` to some new file extension, which will be a breaking change for all tranforms that import `.sky` packages.
We'll keep you posted.

### Features

* **RequestDatasetLog:** add back functionality for getting a peer's dataset log over p2p ([813bf0d](https://github.com/qri-io/qri/commit/813bf0d))
* **RequestDatasetLog:** handle non-local datasets, rename lots of vars ([248e02e](https://github.com/qri-io/qri/commit/248e02e))



<a name="0.5.4"></a>
# [0.5.4](https://github.com/qri-io/qri/compare/v0.5.3...v0.5.4) (2018-10-01)

### :wrench: minor patch release
0.5.4 is a _very_ minor release that cleans up a few issues with our API to make the frontend editor work :smile:

### Features

* **ConvertBodyFile:** extracted actions.ConvertBodyFile from LookupBody func ([d8a7f77](https://github.com/qri-io/qri/commit/d8a7f77))



<a name="0.5.3"></a>
# [0.5.3](https://github.com/qri-io/qri/compare/v0.5.2...v0.5.3) (2018-09-24)

### :running_woman: Dry Run
0.5.3 is a minor version bump that introduces a new `--dry-run` flag on `qri new` and `qri save` that'll run new/save _without committing changes_. We've added the same thing on the api using a query param: `dry_run=true`. Dry runs are fun! use 'em to experiment with different input without having to constantly delete stuff.

### Bug Fixes

* **dsgraph:** fix breaking change from dataset pkg ([0954513](https://github.com/qri-io/qri/commit/0954513))


### Features

* **api dry run:** add dry run & moar file options to /new ([52cfa19](https://github.com/qri-io/qri/commit/52cfa19))
* **dry_run flag:** added dry run flag to creating & updating datasets ([ef2d5ca](https://github.com/qri-io/qri/commit/ef2d5ca))
* **p2p local streams:** add local streams to QriNode for local stdio interaction ([53ff3fd](https://github.com/qri-io/qri/commit/53ff3fd))



<a name="0.5.2"></a>
# [0.5.2](https://github.com/qri-io/qri/compare/v0.5.1...v0.5.2) (2018-09-14)


Most of 0.5.2 is under-the-hood changes that make qri work better. We've put a _lot_ of time into our test suite & refactoring our existing code, which will set the stage for dev work to move a little faster in the coming weeks :smile:

### :truck: IPFS repo migration
We've updated our IPFS dependencies from go-ipfs 0.4.15 to 0.4.17. Between those verions the structure of IPFS repositories changed, and as such need to be migrated. If this is your first time installing Qri, you won't have an issue, but if you're coming from Qri 0.5.1 or earlier you'll see a message after upgrading that'll explain you'll need to install go-ipfs to update your repo. We know this is, well, not fun, but it's the safest way to ensure the integrity of your underlying IPFS repo.

### Bug Fixes

* **api.Body:** fix bug that returns error when getting a peers body ([d87ccac](https://github.com/qri-io/qri/commit/d87ccac))
* **api/datasets:** fix error based on [@dustmop](https://github.com/dustmop)'s comment https://github.com/qri-io/qri/pull/533#discussion_r214204240 ([a509627](https://github.com/qri-io/qri/commit/a509627)), closes [/github.com/qri-io/qri/pull/533#discussion_r214204240](https://github.com//github.com/qri-io/qri/pull/533/issues/discussion_r214204240)
* **cmd:** Don't ignore command-line arguments for `get` command. ([d6b49d3](https://github.com/qri-io/qri/commit/d6b49d3))
* **cmd:** Execute commands using RunE instead of Run ([5457ccb](https://github.com/qri-io/qri/commit/5457ccb))
* **cmd:** Operating system-specific calls in their own sourcefile ([4074df6](https://github.com/qri-io/qri/commit/4074df6))
* **cmd:** Set number of open files limit at startup. ([761259f](https://github.com/qri-io/qri/commit/761259f))
* **cmd:** Split old `add` command into `new` and `add`. ([0abc837](https://github.com/qri-io/qri/commit/0abc837))
* **cmd/cmd.go:** have printErr check errs for lib.Error and print correct message ([75a7e85](https://github.com/qri-io/qri/commit/75a7e85))
* **connect --setup:** add Anon flag to make setup work w/o user input ([dcb92ae](https://github.com/qri-io/qri/commit/dcb92ae))
* **CreateDataset:** remove transform creating default viz ([4536dc6](https://github.com/qri-io/qri/commit/4536dc6))
* **get:** Additional documentation about lib/Get method. ([6108ddd](https://github.com/qri-io/qri/commit/6108ddd))
* **get:** Can get another user's repo, as long as node is connected. ([3251d58](https://github.com/qri-io/qri/commit/3251d58))
* **get:** Get completely reworked to function better ([7c0cc8d](https://github.com/qri-io/qri/commit/7c0cc8d)), closes [#519](https://github.com/qri-io/qri/issues/519) [#509](https://github.com/qri-io/qri/issues/509) [#479](https://github.com/qri-io/qri/issues/479) [#397](https://github.com/qri-io/qri/issues/397)
* **LookupBody:** LookupBody should respond with *body* path ([7c1f38f](https://github.com/qri-io/qri/commit/7c1f38f))
* **p2p:** Avoid generating p2p keys in DefaultP2P, for faster tests ([97a1b6e](https://github.com/qri-io/qri/commit/97a1b6e))
* **p2p:** NewQriNode parameter cleanup, and factory for TestableQriNode. ([5652f4d](https://github.com/qri-io/qri/commit/5652f4d))
* **profile:** Tests for ProfilePod.Copy and ProfilePod.SetField ([a1a464c](https://github.com/qri-io/qri/commit/a1a464c))
* **profile:** Using config set profile.something calls SaveProfile. ([7c3e9f6](https://github.com/qri-io/qri/commit/7c3e9f6))


### Features

* **api/open_api:** add both open api 2.0 and open api 3.0 specs ([4faf8ac](https://github.com/qri-io/qri/commit/4faf8ac))
* **cmd loading:** add loading indicator for long-running commands ([d988843](https://github.com/qri-io/qri/commit/d988843))
* **docs:** add script to generate documentation in markdown ([6ca60f4](https://github.com/qri-io/qri/commit/6ca60f4))
* **p2p.ResolveDatasetRef:** resolve dataset names with p2p network ([4fd24c5](https://github.com/qri-io/qri/commit/4fd24c5))
* **profile:** Detect peername renames when listing datasets. ([f1a19ba](https://github.com/qri-io/qri/commit/f1a19ba))
* **ResolveDatasetRef:** new action for resolving dataset references ([00aefc2](https://github.com/qri-io/qri/commit/00aefc2))
* **save:** re-run transform on qri save with no args ([0e906ae](https://github.com/qri-io/qri/commit/0e906ae))



<a name="0.5.1"></a>
# [0.5.1](https://github.com/qri-io/qri/compare/v0.5.1-rc1...v0.5.1) (2018-07-19)

Ok ok so now we have a formal 0.5.1 release. Maybe this should be 0.6.0 given the magnitude of visualizations, but meh, we're calling it a patch.

#### :bar_chart: Delight in Data with HTML-template visualizations
For a little while we've been experimenting with the `qri render` as a way to template data into html. The more we've played with it, the more we've come to rely on it. So much so, that we think templates should become a native component of datasets. For this we've added a new section to the dataset definition called `viz`, which is where you specify a custom template. a `dataset.yaml` file that specifies viz will look something like this (These details are always available with the handy `qri export --blank`):

```
# use viz to provide custom a HTML template of your dataset
# the currently accepted syntax is 'html'
# scriptpath is the path to your template, relative to this file:
# viz:
#   syntax: html
#   scriptpath: template.html
```

A template can only be a single file. So your template can't reference a local "style.css" file for now. _But!_ You're more than welcome to write HTML that grabs external resources, like getting jQuery or D3, or your favourite charting library off a CDN.

When you supply a template, the Dataset itself is handed back to you, and we use golang's `html/template` package to render. Here's a simplified example:

```html
<html>
  <body>
    <h1>{{ .Meta.Title }}</h1>
    <p>{{ .Meta.Description }}</p>
    <ul>
    {{ range .Body }}
      <li>{{ . }}</li>
    {{ end }}
    </ul>
  </body>
</html>
```

This would print the Title & Description from your dataset's metadata. If your dataset was a list of strings, each entry would be printed out as a list item.

#### Moar Better Frontend
We're so into viz, we've completely overhauled the frontend to put custom templates front-and-center for datasets. Be sure to take the new frontend for a spin by visiting `http://localhost:2505` while `qri connect` is running.

We've chosen to invest time in viz because we think it brings an important moment of delight to datasets. Without it, all of this _data_ stuff just feels like work. There's something truly magical about seeing your data rendered in a custom page that makes all the munging worthwhile.


### Bug Fixes

* **get:** fix get command not accepting ds refs ([fc2a2b2](https://github.com/qri-io/qri/commit/fc2a2b2)), closes [#492](https://github.com/qri-io/qri/issues/492)
* **peers:** If user is not connected, show error instead of segfaulting. ([80b1e3e](https://github.com/qri-io/qri/commit/80b1e3e))
* **rpc:** Register types with gob to avoid "reading body EOF" errors ([011f4bd](https://github.com/qri-io/qri/commit/011f4bd))



<a name="0.5.1-rc1"></a>
# [0.5.1-rc1](https://github.com/qri-io/qri/compare/v0.5.0...v0.5.1-rc1) (2018-07-14)

0.5.1 is a release candidate to get a few important details out the door to demo at the wonderful _our networks_ conference in Toronto. we antipate further changes before cutting a proper 0.5.1 release, which we'll properly document. More soon!

### Bug Fixes

* **add:** Work-around for RPC error that breaks `qri add`. ([d9f7805](https://github.com/qri-io/qri/commit/d9f7805))
* **cmd:** Assume "me" peername for save and remove as well. ([1ef627c](https://github.com/qri-io/qri/commit/1ef627c))
* **cmd:** Wrap how DatasetRefs are parsed for better cmd behavior. ([62bf188](https://github.com/qri-io/qri/commit/62bf188))
* **cmd/validate:** fix bug that did not give proper error message when command inputs were incorrect ([07b2d13](https://github.com/qri-io/qri/commit/07b2d13))
* **render:** fix render template render priority ([dffdbb4](https://github.com/qri-io/qri/commit/dffdbb4))
* **render:** restore render tests, small refactoring ([7735669](https://github.com/qri-io/qri/commit/7735669))


### Features

* **`/add`:** can now use dataset.yaml or .json file to add commit/meta/structure/etc info on add ([3cc1917](https://github.com/qri-io/qri/commit/3cc1917))
* **api.Registry:** add groundwork for `/registry/` endpoint ([ec10ddd](https://github.com/qri-io/qri/commit/ec10ddd))
* **api/registry, lib/registry, repo/registry:** finish publish and unpublish endpoints ([841bfb8](https://github.com/qri-io/qri/commit/841bfb8))
* **api/search:** add `/search` endpoint back to api ([25635f1](https://github.com/qri-io/qri/commit/25635f1))
* **api/server:** add RegistryHandler to server ([b675fdd](https://github.com/qri-io/qri/commit/b675fdd))
* **api/server:** version added to `/status` response ([739671e](https://github.com/qri-io/qri/commit/739671e))
* **cmd Validate:** `Validate` func for `validate` cmd ([b9e675b](https://github.com/qri-io/qri/commit/b9e675b))
* **cmd/search:** Validate function to validate user input ([0593058](https://github.com/qri-io/qri/commit/0593058))
* **lib.Error:** new type Error that satisfies the error interface ([207e314](https://github.com/qri-io/qri/commit/207e314))
* **pin,unpin:** add registry actions for pinning and unpinning ([c67df8c](https://github.com/qri-io/qri/commit/c67df8c))
* **TestFactory:** new struct TestFactory that can be used as factory when building tests on individual commands ([79446cd](https://github.com/qri-io/qri/commit/79446cd))
* **useOptions.validate:** add `Validate` function to use command ([d831a51](https://github.com/qri-io/qri/commit/d831a51))
* **viz:** add viz api endpoint, viz refactor ([b5a658e](https://github.com/qri-io/qri/commit/b5a658e))



<a name="0.5.0"></a>
# [0.5.0](https://github.com/qri-io/qri/compare/v0.4.0...v0.5.0) (2018-06-18)

Who needs patch releases!? In version 0.5.0 we're introducing an initial search implementation, a few new commands, rounding out a bunch of features that warrant a minor version bump with some breaking changes. Because of these breaking changes, datasets created with v0.4.0 or earler will need to be re-created to work properly with this version.

#### :mag: Registry Search Alpha
We're still hard at work on getting registries right (more on that in the coming weeks), but for now we've shipped an initial command "qri search", that'll let you search registries for datasets. The way we see search working in the future is leveraging registries to build indexes of published datasets so you can easily search for datasets that have been published. We have a lot of work to do around making sure those datasets are available for download, but feel free to play with the command to get a feel for where we're headed with search.

#### use & get commands
working with the command line, it can get really irritating to constantly re-type the name of a dataset. To help with this, we've added a new command: `qri use`, which takes it's inspiration from database "use" commands that set the current database. `qri use` takes any number of dataset references as arguments, and once a user has set a selection with qri use, they become the default dataset names when no selection is made.

Around qri HQ we've all come to love the ease of working with the `qri config` command. `qri config get` shows either the whole configuration, or you can provide a dot.separated.path to scope down the section of config to show. `qri get` takes this idea an applies it to datasets. `qri get meta.title me/dataset_name` will get the title from metadata, and like the config command, it's output can be switched from YAML (default) to JSON. `qri get` also accepts multiple datasets, which will be combined into 

#### new skylark html module
We're still working on a proper html module for skylark transforms with an API that pythonists will be familiar with, but in the meantime we've added in a basic jquery-like selector syntax for working with HTML documents.

#### "Data" is now "Body"
This is a breaking change we've been hoping to get in sooner-rather-than-later that renames the `Data` field of a dataset to `Body`. From here on in we'll refer to the _body_ of a dataset as it's principle content. We think this language helps show how datasets are like webpages, and cutw down on use of an ambiguous term like "data".

#### Thinking about qri as an importable library
Finally, in what is more of a symbolic change than anything else, we've renamed the `core` package to `lib` to get us to start thinking about qri as an importable library. Next week we'll publish a new project aimed at writing tutorials & docs with an associated test suite built around datasets that uses qri as a library. We hope to use this project to mature the `lib` package for this use case.


### Bug Fixes

* **cmd:** set core.ConfigPath on init ([9e841b8](https://github.com/qri-io/qri/commit/9e841b8))
* **cmd.RPC:** warn when RPC connected ([00ff64f](https://github.com/qri-io/qri/commit/00ff64f))
* **cmd/list:** fix `network` flag error ([e15483b](https://github.com/qri-io/qri/commit/e15483b))
* **config:** set map values with config set command ([edd9e44](https://github.com/qri-io/qri/commit/edd9e44)), closes [#450](https://github.com/qri-io/qri/issues/450)
* **CreateDataset:** user-specified data should override transforms ([7edff80](https://github.com/qri-io/qri/commit/7edff80))
* **CreateDataset:** user-specified data should override transforms ([dea1ac4](https://github.com/qri-io/qri/commit/dea1ac4)), closes [#456](https://github.com/qri-io/qri/issues/456)
* **list peer datasets:** fix to make listing peer datasets work ([f1df720](https://github.com/qri-io/qri/commit/f1df720))
* **MemRepo:** remove privatekey field, properly assign regclient ([594a86c](https://github.com/qri-io/qri/commit/594a86c))
* **save:** qri save without CLI dataset arg ([780d734](https://github.com/qri-io/qri/commit/780d734))


### Code Refactoring

* **dataset.Body:** rename dataset.Data to Body ([d764749](https://github.com/qri-io/qri/commit/d764749)), closes [#422](https://github.com/qri-io/qri/issues/422)
* **export:** remove excessive flags from qri export ([aeb5ee4](https://github.com/qri-io/qri/commit/aeb5ee4))


### Features

* **default to use:** added use-defaults to commands ([dee523e](https://github.com/qri-io/qri/commit/dee523e))
* **export:** export transform.sky when present ([5939668](https://github.com/qri-io/qri/commit/5939668))
* **Reference Selection:** support reference selection ([3cae7bd](https://github.com/qri-io/qri/commit/3cae7bd))
* **Search:** added implementations for core search and commandline search ([698a53b](https://github.com/qri-io/qri/commit/698a53b))
* **use flags:** added list & clear flags to qri use ([3b61f60](https://github.com/qri-io/qri/commit/3b61f60))
* **use, get:** added use and get commands ([6628354](https://github.com/qri-io/qri/commit/6628354))
* **yaml body:** support yaml body with json conversion ([49568e0](https://github.com/qri-io/qri/commit/49568e0))


### BREAKING CHANGES

* **dataset.Body:** this change will break hashes. `dataPath` is now `bodyPath`.
* **export:** option to export specific dataset components has been removed



<a name="0.4.0"></a>
# [0.4.0](https://github.com/qri-io/qri/compare/v0.3.2...v0.4.0) (2018-06-06)

_We're going to start writing proper release notes now, so, uh, here are those notes:_

This release brings a big new feature in the form of our first transformation implementation, and a bunch of refinements that make the experience of working with qri a little easier. 

#### Introducing Skylark Transformations
For months qri has had a planned feature set for embedding the concept of "transformations" directly into datasets. Basically transforms are scripts that auto-generate datasets. We've defined a "transformation" to be a repeatable process that takes zero or more datasets and data sources, and outputs exactly one dataset. By embedding transformations directly into datasets, users can repeat them with a single command, keeping the code that updates a dataset, and it's resulting data in the same place. This opens up a whole new set of uses for qri datasets, making them auditable, repeatable, configurable, and generally _functional_. Using transformations, qri can check to see if your dataset is out of date, and update it for you.

While we've had the _plan_ for transformations for some time now, it's taken us a long time to figure out how to write a first implementation. Because transformations are executable code, security & behavioural expectations are a big concern. We also want to set ourselves up for success by choosing an implementation that will feel familiar to those who do a lot of code-based data munging, while also leaving the door open to things we'd like to do in the future like parallelized execution.

So after a lot of research and a false-start or five, we've decided on a scripting language called _skylark_ as our base implementation, which has grown out of the _bazel_ project at google. This choice might seem strange at first (bazel is a build tool and has nothing to do with data), but skylark has a number of advantages:
* **python-like syntax** - _many_ people working in data science these days write python, we like that.
* **deterministic subset of python** - unlike python, skylark removes properties that reduce introspection into code behaviour. things like `while` loops and recursive functions are omitted, making it possible for qri to infer how a given transformation will behave.
* **parallel execution** - thanks to this deterministic requirement (and lack of global interpreter lock) skylark functions can be executed in parallel. Combined with peer-2-peer networking, we're hoping to advance tranformations toward peer-driven distribed computing. More on that in the coming months.

A tutorial on how to write skylark transformations is forthcoming, we'll post examples to our documentation site when it's ready: https://qri.io/docs

#### dataset.yaml, and more :heart: for the CLI
For a while now we've been thinking about datasets as being a lot like web pages. Web pages have `head`,`meta` and `body` elements. Datasets have `meta`, `structure`, `commit`, and `data`. To us this metaphor helps reason about the elements of a dataset, why they exist, and their function. And just like how webpages are defined in `.html` files, we've updated the CLI to work with `.yaml` files that define dataests. `qri export --blank` will now write a blank dataset file with comments that link to documentation on each section of a dataset. You can edit that file, save it, and run `qri add --dataset=dataset.yaml me/my_dataset` to add the dataset to qri. Ditto for `qri update`.

We'd like to encourage users to think in terms of these `dataset.yaml` files, building up a mental model of each element of a dataset in much the same way we think about HTML page elements. We chose `yaml` over JSON specifically because we can include comments in these files, making it easier to pass them around with tools outside of qri, and we're hoping this'll make it easier to think about datasets moving forward. In futures release we plan to rename the "data" element to "body" to bring this metaphor even closer.

Along with `dataset.yaml`, we've also done a bunch of refactoring & bug fixes to make the CLI generally work better, and look forward to improving on this trend in near-term patch releases. One of the biggest things we'd like to improve upon is providing more meaningful error messages.


### Bug Fixes

* **cmd.list:** `qri list` now displays datasets size in KBs, MBs, etc ([c4805ca](https://github.com/qri-io/qri/commit/c4805ca))
* **cmd/add:** title and message flags work and override the dataset.Commit fields ([0ef9b80](https://github.com/qri-io/qri/commit/0ef9b80))
* **ipfs:** Display better error if IPFS is holding a lock ([dbc1604](https://github.com/qri-io/qri/commit/dbc1604))
* **list:** Show info for datasets even if they don't have meta ([a853e71](https://github.com/qri-io/qri/commit/a853e71))


### Code Refactoring

* **core.Add, core.Save:** refactor input params, folding into a DatasetPod ([e63694f](https://github.com/qri-io/qri/commit/e63694f))


### Features

* **add:** accept --dataset yaml/json file arg to add ([6e04481](https://github.com/qri-io/qri/commit/6e04481))
* **backtrace:** Show error's stack if QRI_BACKTRACE is defined ([88160e2](https://github.com/qri-io/qri/commit/88160e2))
* **cmd.Export:** added --blank flag to export dataset template ([27d13a0](https://github.com/qri-io/qri/commit/27d13a0)), closes [#409](https://github.com/qri-io/qri/issues/409)
* **cmd/export:** specify data format using --data-format flag ([8549d59](https://github.com/qri-io/qri/commit/8549d59))
* **CreateDataset:** write author ProfileID when creating a dataset ([93d4ef1](https://github.com/qri-io/qri/commit/93d4ef1))
* **export:** can now choose dataset/structure/meta format on export ([2863ded](https://github.com/qri-io/qri/commit/2863ded))
* **registry cmd:** add commands for working with registries ([85d6892](https://github.com/qri-io/qri/commit/85d6892))
* **skylark:** Pass previous dataset body to skylark for updates ([64e5b64](https://github.com/qri-io/qri/commit/64e5b64))
* **transform:** execute transformations with skylark language ([f684229](https://github.com/qri-io/qri/commit/f684229))
* **transform secrets:** supply secrets to transforms ([5d8cac9](https://github.com/qri-io/qri/commit/5d8cac9))


### BREAKING CHANGES

* **core.Add, core.Save:** add & save on both API & CLI now accept a "file" which is the full dataset



<a name="0.3.2"></a>
# [0.3.2](https://github.com/qri-io/qri/compare/v0.3.1...v0.3.2) (2018-05-15)


### Bug Fixes

* **core.profile:** fixing bug in SetPosterPhoto and SetProfilePhoto ([8cbed3a](https://github.com/qri-io/qri/commit/8cbed3a))
* **handleDatasetsList:** peer's datasets not returning fully hydrated ([b755554](https://github.com/qri-io/qri/commit/b755554))
* **profile.MemStore:** address race condition ([58ece89](https://github.com/qri-io/qri/commit/58ece89))
* **registry:** make registry post & deletes actually work ([f71eec6](https://github.com/qri-io/qri/commit/f71eec6))
* **SetProfile:** peer should be able to remove name, email, description, homeurl, and twitter ([6f4cc1e](https://github.com/qri-io/qri/commit/6f4cc1e))
* **setup:** fix config not loading if already found ([190f2c7](https://github.com/qri-io/qri/commit/190f2c7))
* **test_repo:** added field `Type: "peer"` to test repo profile ([8478462](https://github.com/qri-io/qri/commit/8478462))


### Features

* **cmd.peers:** merge connections command into peers list command ([a3fe649](https://github.com/qri-io/qri/commit/a3fe649))
* **core.GetProfile:** add naive online indicator if P2P is enabled ([84ccc24](https://github.com/qri-io/qri/commit/84ccc24))
* **p2p:** Tag qri peers in connManager ([2f4b1fd](https://github.com/qri-io/qri/commit/2f4b1fd))
* **peer connections:** single peer connection control ([ab7e035](https://github.com/qri-io/qri/commit/ab7e035))
* **Profile Info, Connections:** verbose profile info, better explicit conn mgmt ([eb01247](https://github.com/qri-io/qri/commit/eb01247))
* **profiles:** transition to online-first display, announce ntwk joins ([f712080](https://github.com/qri-io/qri/commit/f712080))
* **registry.Datasets:** initial dataset registry integration ([58d64c2](https://github.com/qri-io/qri/commit/58d64c2)), closes [#397](https://github.com/qri-io/qri/issues/397)
* **render:** add render command for executing templates against datasets ([607104d](https://github.com/qri-io/qri/commit/607104d))
* **repo:** merge repos' EventLogs using a simple first attempt ([e5997bb](https://github.com/qri-io/qri/commit/e5997bb))
* **repo.Registry:** add registry to repo interface ([9bcb303](https://github.com/qri-io/qri/commit/9bcb303))



<a name="0.3.1"></a>
# [0.3.1](https://github.com/qri-io/qri/compare/v0.3.0...v0.3.1) (2018-04-25)


### Bug Fixes

* **api:** add image/jpeg content type to PosterHandler ([b0e2495](https://github.com/qri-io/qri/commit/b0e2495))
* **config:** `qri config set` on an int field does not actually update ([435c634](https://github.com/qri-io/qri/commit/435c634))
* **connect:** fix errors caused by running connect with setup ([e522925](https://github.com/qri-io/qri/commit/e522925))
* **core.Add:** major cleanup to make add work properly ([e7951d4](https://github.com/qri-io/qri/commit/e7951d4))
* **core.StructuredData:** make StructuredData handle object datasets ([ffd8dd3](https://github.com/qri-io/qri/commit/ffd8dd3))
* **fsrepo.Profiles:** add file lock for peers.json ([f692d55](https://github.com/qri-io/qri/commit/f692d55)), closes [#357](https://github.com/qri-io/qri/issues/357)
* **fsrepo.ProfileStore:** add mutex lock ([2d4d9ee](https://github.com/qri-io/qri/commit/2d4d9ee))
* **p2p:** restore RPC peer-oriented calls, add CLI qri peers connect ([6b07712](https://github.com/qri-io/qri/commit/6b07712))
* **RPC:** bail on commands that currently don't work over RPC ([cefcebd](https://github.com/qri-io/qri/commit/cefcebd))
* **webapp:** replaced hardcoded entryupintupdateaddress ([c221d7d](https://github.com/qri-io/qri/commit/c221d7d))


### Features

* **cmd:** organize commands into groups. ([1059277](https://github.com/qri-io/qri/commit/1059277))
* **config:** add registry config ([1c72892](https://github.com/qri-io/qri/commit/1c72892))
* **config:** made webapp updates lookup configurable ([8999021](https://github.com/qri-io/qri/commit/8999021))
* **registry:** use registry at setup to claim peername ([5b3c2ee](https://github.com/qri-io/qri/commit/5b3c2ee))
* **teardown:** added 'uninstall' option from the CLI & core Teardown method ([428eeb7](https://github.com/qri-io/qri/commit/428eeb7)), closes [#175](https://github.com/qri-io/qri/issues/175)



<a name="0.3.0"></a>
# [0.3.0](https://github.com/qri-io/qri/compare/v0.2.0...v0.3.0) (2018-04-09)


### Bug Fixes

* **api:** fix double api import caused by rebase ([b590776](https://github.com/qri-io/qri/commit/b590776))
* **config:** fix issues created by config overhaul ([501cf9a](https://github.com/qri-io/qri/commit/501cf9a)), closes [#328](https://github.com/qri-io/qri/issues/328) [#329](https://github.com/qri-io/qri/issues/329)
* **dockerfile:** Use unambiguous adduser arguments ([#331](https://github.com/qri-io/qri/issues/331)) ([2061d93](https://github.com/qri-io/qri/commit/2061d93))
* **fsrepo.Peerstore:** fix peerstore writing invalid json ([b76c52b](https://github.com/qri-io/qri/commit/b76c52b))
* **structuredData:** data should return DataPath not dataset Path ([7bae799](https://github.com/qri-io/qri/commit/7bae799))
* **webapp:** hide webapp hash behind vanity url ([64a5a76](https://github.com/qri-io/qri/commit/64a5a76))


### Features

* **actions:** added new repo actions package to encapsulate repo biz logic ([20322e1](https://github.com/qri-io/qri/commit/20322e1))
* **cmd.Config:** added qri config export command ([87fc395](https://github.com/qri-io/qri/commit/87fc395))
* **config:** add basic validation for api config ([c353200](https://github.com/qri-io/qri/commit/c353200))
* **config:** add basic validation for cli config ([0bea261](https://github.com/qri-io/qri/commit/0bea261))
* **config:** add basic validation for logging config ([2e6ccbb](https://github.com/qri-io/qri/commit/2e6ccbb))
* **config:** add basic validation for p2p config ([57b740c](https://github.com/qri-io/qri/commit/57b740c))
* **config:** add basic validation for profile config ([77fe3eb](https://github.com/qri-io/qri/commit/77fe3eb))
* **config:** add basic validation for repo config ([6fb0cda](https://github.com/qri-io/qri/commit/6fb0cda))
* **config:** add basic validation for rpc config ([4ddc2e6](https://github.com/qri-io/qri/commit/4ddc2e6))
* **config:** add basic validation for store config ([a9c341f](https://github.com/qri-io/qri/commit/a9c341f))
* **config:** add basic validation for webapp config ([6d64eb3](https://github.com/qri-io/qri/commit/6d64eb3))
* **config:** add more tests for config validation ([cd8aa2f](https://github.com/qri-io/qri/commit/cd8aa2f))
* **config:** add tests for config validation ([2175e03](https://github.com/qri-io/qri/commit/2175e03))
* **config:** unify qri configuration into single package ([6969ec5](https://github.com/qri-io/qri/commit/6969ec5))
* **p2p:** shiny new peer-2-peer communication library ([7a4e292](https://github.com/qri-io/qri/commit/7a4e292))
* **p2p.AnnounceDatasetChanges:** new message for announcing datasets ([29016e6](https://github.com/qri-io/qri/commit/29016e6))
* **readonly:** add check for readonly & GET in middleware ([92a2e84](https://github.com/qri-io/qri/commit/92a2e84))
* **readonly:** add fields for ReadOnly option ([832f7e3](https://github.com/qri-io/qri/commit/832f7e3))
* **readonly:** add ReadOnly to handlers ([e04f9c1](https://github.com/qri-io/qri/commit/e04f9c1))
* **readonly:** add tests for read-only server ([d7c8732](https://github.com/qri-io/qri/commit/d7c8732))
* **repo.Event:** removed DatasetAnnounce in favor of event logs ([35686fd](https://github.com/qri-io/qri/commit/35686fd))



<a name="0.2.0"></a>
# [0.2.0](https://github.com/qri-io/qri/compare/v0.1.2...v0.2.0) (2018-03-12)


### Bug Fixes

* **api:** update history, add, remove, info to work with hashes ([60f79fd](https://github.com/qri-io/qri/commit/60f79fd)), closes [#222](https://github.com/qri-io/qri/issues/222)
* **CanonicalizeDatatsetRef:** should hydrate datasetRef if given path ([bf12ac4](https://github.com/qri-io/qri/commit/bf12ac4))
* **cmd.Setup:** added back env var to supply IPFS config on setup ([fb19615](https://github.com/qri-io/qri/commit/fb19615))
* **data api endpoint:** need path in response to be able to normalize data on frontend ([dc0d1f0](https://github.com/qri-io/qri/commit/dc0d1f0))
* **DatasetRef:** fixes to datasetRef handling ([134e0f9](https://github.com/qri-io/qri/commit/134e0f9))
* **refstore:** fix bug that allowed ref to save without peername ([4698dab](https://github.com/qri-io/qri/commit/4698dab))
* **Validate:** completely overhaul validation to make it work properly ([8af5653](https://github.com/qri-io/qri/commit/8af5653)), closes [#290](https://github.com/qri-io/qri/issues/290) [#290](https://github.com/qri-io/qri/issues/290)


### Features

* added color diff output that works with global color flags ([d46afac](https://github.com/qri-io/qri/commit/d46afac))
* **private flag** add private flag to cli and api add ([6ffee3d](https://github.com/qri-io/qri/commit/6ffee3d))
* color diff output ([7f38363](https://github.com/qri-io/qri/commit/7f38363))
* **api.List:** allow /list/[peer_id] ([a2771f9](https://github.com/qri-io/qri/commit/a2771f9))
* **CBOR:** added experimental support for CBOR data format ([6ac1c0e](https://github.com/qri-io/qri/commit/6ac1c0e))
* **cmd.data:** added a 'data' command to the CLI to match /data API endpoint ([51d36d2](https://github.com/qri-io/qri/commit/51d36d2))
* **diff:** add `/diff` api endpoint ([91c3d22](https://github.com/qri-io/qri/commit/91c3d22))
* **PeerID:** add PeerID to datasetRef ([005814e](https://github.com/qri-io/qri/commit/005814e))
* **save:** add `/save/` endpoint ([8563835](https://github.com/qri-io/qri/commit/8563835))
* **SelfUpdate:** added check for version being out of date ([0d87261](https://github.com/qri-io/qri/commit/0d87261))
* **webapp:** add local webapp server on port 2505 ([e38951d](https://github.com/qri-io/qri/commit/e38951d))
* **webapp:** fetch webapp hash via dnslink ([101809d](https://github.com/qri-io/qri/commit/101809d))



<a name="0.1.2"></a>
# [0.1.2](https://github.com/qri-io/qri/compare/v0.1.1...v0.1.2) (2018-02-19)


### Bug Fixes

* **api history:** need to use new repo.CanonicalizeDatasetRef function to get correct ref to pass to history ([4ee7ab1](https://github.com/qri-io/qri/commit/4ee7ab1))
* **cmd:** invalid flags no longer emit a weird message ([44657ee](https://github.com/qri-io/qri/commit/44657ee))
* **cmd.Export:** fixed the ordering of path and namespace ([0ca5791](https://github.com/qri-io/qri/commit/0ca5791))
* **cmd.version:** fix incorrect version number ([#262](https://github.com/qri-io/qri/issues/262)) ([6465192](https://github.com/qri-io/qri/commit/6465192))
* **NewFilesRequest, save:** renamed NewMimeMultipartRequest to NewFilesRequest ([35da882](https://github.com/qri-io/qri/commit/35da882))
* **ParseDatasetRef, HTTPPathToQriPath:** fix that allows datasetRefs to parse peername/dataset_name@/hash ([40943ae](https://github.com/qri-io/qri/commit/40943ae))
* **save:** reset the meta and structure paths after assign ([c815bd6](https://github.com/qri-io/qri/commit/c815bd6))
* **SaveRequestParams, initHandler:** fix bug that only let us add a dataset from a file (now can add from a url) ([0e3e908](https://github.com/qri-io/qri/commit/0e3e908))


### Features

* **api tests:** expand tests to include testing responses ([b8a4d06](https://github.com/qri-io/qri/commit/b8a4d06))
* **JSON:** support JSON as a first-class citizen in qri ([#271](https://github.com/qri-io/qri/issues/271)) ([6dee242](https://github.com/qri-io/qri/commit/6dee242))
* **profileSchema:** we now validate the peer profile before saving ([6632613](https://github.com/qri-io/qri/commit/6632613))
* **ValidateProfile:** ValidateProfile reads schema from file ([5c4e987](https://github.com/qri-io/qri/commit/5c4e987))



<a name="0.1.1"></a>
# [0.1.1](https://github.com/qri-io/qri/compare/v0.1.0...v0.1.1) (2018-02-13)


### Bug Fixes

* **api, save:** fix bugs to allow save API endpoint to work ([39d9be5](https://github.com/qri-io/qri/commit/39d9be5))
* updated cmd.diff to be compatible with updates to dsdiff ([201bdda](https://github.com/qri-io/qri/commit/201bdda))
* updated output param of core.Diff to `*map[string]*dsdiff.SubDiff` ([8e1aa39](https://github.com/qri-io/qri/commit/8e1aa39))
* **handleDatasetInfoResponse:** fix bug that did not keep path ([5c6d372](https://github.com/qri-io/qri/commit/5c6d372))


### Features

* **add:** add ability to add structure and metadata to API add endpoint ([722d9ad](https://github.com/qri-io/qri/commit/722d9ad))
* **add:** add ability to add structure via the CLI ([69c6a27](https://github.com/qri-io/qri/commit/69c6a27))
* **add:** cmd.add should print out any validation errors as a warning ([92bd873](https://github.com/qri-io/qri/commit/92bd873))
* **api, history:** get your own dataset's history, or a peer's dataset history using peername/datasetname ([bb321de](https://github.com/qri-io/qri/commit/bb321de))
* **cmd, log:** add ability to get history of peer datasets ([e3bb5ab](https://github.com/qri-io/qri/commit/e3bb5ab))
* **cmd.Add:** added a verbose flag to list specific validaton errors ([6b3de72](https://github.com/qri-io/qri/commit/6b3de72))
* **cmd.export:** added optional namespacing flag ([2103885](https://github.com/qri-io/qri/commit/2103885))
* **cmd.Save:** added validation warnings/options to match cmd.Add ([e77176a](https://github.com/qri-io/qri/commit/e77176a))
* **core, log:** add getRemote to Log, add Node *p2p.QriNode to HistoryRequests struct ([4356a9d](https://github.com/qri-io/qri/commit/4356a9d))
* **Init:** add structure and structureFilename to InitParams, and handle adding structure to InitParams in Init function ([9f8785e](https://github.com/qri-io/qri/commit/9f8785e))
* **p2p, history:** add handling history/log requests for peer's dataset ([a5b5f9b](https://github.com/qri-io/qri/commit/a5b5f9b))
* export to zip file and export directory using dataset name now active ([1533293](https://github.com/qri-io/qri/commit/1533293))
* reference canonicalization ([3f3aca5](https://github.com/qri-io/qri/commit/3f3aca5))



<a name="0.1.0"></a>
# 0.1.0 (2018-02-02)


### Bug Fixes

* added the cmd/diff.go file ([b4139f1](https://github.com/qri-io/qri/commit/b4139f1))
* cleanup bugs introduced by recent changes ([c8e5a57](https://github.com/qri-io/qri/commit/c8e5a57))
* cleanup output of export & dataset commands ([4cf18a0](https://github.com/qri-io/qri/commit/4cf18a0))
* lots of little cleanup ([77ce05c](https://github.com/qri-io/qri/commit/77ce05c))
* lots of little cleanups here & there ([b44c749](https://github.com/qri-io/qri/commit/b44c749))
* lots of nitty-gritty fixes in time for demo. ([bc09dcf](https://github.com/qri-io/qri/commit/bc09dcf))
* more work on settling Transform refactor ([6010e9f](https://github.com/qri-io/qri/commit/6010e9f))
* remove query dedupe for now, it's broken ([a27c274](https://github.com/qri-io/qri/commit/a27c274))
* removed a println that was causing an invalid nil pointer dereference ([43c5123](https://github.com/qri-io/qri/commit/43c5123))
* **core.DatasetHandlers.StructuredData:** fix support for different data formats ([cda0728](https://github.com/qri-io/qri/commit/cda0728))
* restoring tests, cleaning up post-ds.Transform refactor ([f3dea07](https://github.com/qri-io/qri/commit/f3dea07))
* updated handlers functions taking a ListParams to use the OrderBy field ([1bf6ec3](https://github.com/qri-io/qri/commit/1bf6ec3))
* updated pagination to rely on core struct ([cdba451](https://github.com/qri-io/qri/commit/cdba451))
* more pre-demo tweaks ([a90ba04](https://github.com/qri-io/qri/commit/a90ba04))
* **api add:** fix api add endpoint bugs ([e3b2554](https://github.com/qri-io/qri/commit/e3b2554))
* **build:** always fetch dependencies from the network ([510e5ff](https://github.com/qri-io/qri/commit/510e5ff)), closes [#217](https://github.com/qri-io/qri/issues/217)
* **cmd.Add:** cleaned up add command ([875cd6f](https://github.com/qri-io/qri/commit/875cd6f))
* **cmd.queriesCmd:**  fix query listing command ([5ce7d38](https://github.com/qri-io/qri/commit/5ce7d38))
* **core.DatasetRequests.Delete:** restore dataset delete method ([54abed8](https://github.com/qri-io/qri/commit/54abed8)), closes [#89](https://github.com/qri-io/qri/issues/89)
* **core.DatasetRequests.Rename:** prevent name overwrite ([2d44f48](https://github.com/qri-io/qri/commit/2d44f48))
* **core.DatasetRequests.Rename:** repo.DatasetRef return ([cbde8a7](https://github.com/qri-io/qri/commit/cbde8a7))
* **core.History:** fix null datasets in repsonses ([ab1fd7c](https://github.com/qri-io/qri/commit/ab1fd7c))
* **core.PeerRequests.List:** remove current user from peers list ([e8611d0](https://github.com/qri-io/qri/commit/e8611d0)), closes [#115](https://github.com/qri-io/qri/issues/115)
* vendor in missing dsgraph dep, update  deps ([1967864](https://github.com/qri-io/qri/commit/1967864))
* **deps:** fix outdated ipfs gx dep ([6017ea3](https://github.com/qri-io/qri/commit/6017ea3))
* **ListParams:** address zero-indexing error in ListParams ([c317b28](https://github.com/qri-io/qri/commit/c317b28))
* **p2p.Bootstrap:** fixes to bootstrapping ([351c8dd](https://github.com/qri-io/qri/commit/351c8dd))
* **repo.Namestore.PutName:** added test to prevent empty names in repos ([202f33d](https://github.com/qri-io/qri/commit/202f33d))
* **repo.WalkRepoDatasets:** remove data race ([56862e2](https://github.com/qri-io/qri/commit/56862e2))
* **repo/graph.RepoGraph:** fix data race ([7372abc](https://github.com/qri-io/qri/commit/7372abc))


### Code Refactoring

* **core.DatasetRequests.Save:** removed ambiguous Save method ([f1972dc](https://github.com/qri-io/qri/commit/f1972dc))


### Features

* **api.ServeRPC:** serve core methods over RPC ([d4778fe](https://github.com/qri-io/qri/commit/d4778fe))
* **bash completions, DatasetRef:** added compl maker, dataset ref parsing ([036ccee](https://github.com/qri-io/qri/commit/036ccee))
* **ChangeRequest:** add support for change requests to query repositories ([eba76ea](https://github.com/qri-io/qri/commit/eba76ea))
* **cli.export:** added export command ([232b5d9](https://github.com/qri-io/qri/commit/232b5d9))
* **cmd.Config:** re-add support for configuration files ([6305d22](https://github.com/qri-io/qri/commit/6305d22)), closes [#108](https://github.com/qri-io/qri/issues/108)
* **cmd.config, cmd.profile:** added initial profile & config commands ([0e968fd](https://github.com/qri-io/qri/commit/0e968fd)), closes [#192](https://github.com/qri-io/qri/issues/192)
* **cmd.Init:** added init command ([a3530c1](https://github.com/qri-io/qri/commit/a3530c1))
* **cmd.Init:** brand new initialization process ([79371c0](https://github.com/qri-io/qri/commit/79371c0))
* **cmd.Init:** set init args via ENV variables ([878e7cb](https://github.com/qri-io/qri/commit/878e7cb))
* **cmd.Search:** added format to search, fix dockerfile ([314f088](https://github.com/qri-io/qri/commit/314f088))
* **core.DatasetRequests.Rename:** added capacity to rename a dataset ([a5776b3](https://github.com/qri-io/qri/commit/a5776b3))
* **core.HistoryRequests.Log:** deliver history as a log of dataset references ([6ca1839](https://github.com/qri-io/qri/commit/6ca1839))
* **core.Init:** initialize a dataset from a url ([7858ba7](https://github.com/qri-io/qri/commit/7858ba7))
* **core.QueryRequests.DatasetQueries:** first implementation ([a8fd2ec](https://github.com/qri-io/qri/commit/a8fd2ec))
* **core.QueryRequests.Query:** check for previously executed queries ([c3be454](https://github.com/qri-io/qri/commit/c3be454)), closes [#30](https://github.com/qri-io/qri/issues/30)
* **Datasets.List:** list removte peer datasets ([69a5210](https://github.com/qri-io/qri/commit/69a5210))
* **DefaultDatasets:** first cuts on requesting default datasets ([05a9e2f](https://github.com/qri-io/qri/commit/05a9e2f)), closes [#161](https://github.com/qri-io/qri/issues/161)
* **history:** add support for dataset history logs ([f9a3938](https://github.com/qri-io/qri/commit/f9a3938))
* **license:** switch project license to GPLv3 ([66edf29](https://github.com/qri-io/qri/commit/66edf29))
* **makefile:** added make build command ([cc6b26a](https://github.com/qri-io/qri/commit/cc6b26a))
* working on remote dataset retrieval ([d8b4424](https://github.com/qri-io/qri/commit/d8b4424))
* **makefile:** make build now requires gopath ([e74b8a3](https://github.com/qri-io/qri/commit/e74b8a3))
* added *tentative* workaround for diffing data--might need to refactor ([757753c](https://github.com/qri-io/qri/commit/757753c))
* **p2p.Bootstrap:** boostrap off both IPFS and QRI nodes ([4e80265](https://github.com/qri-io/qri/commit/4e80265))
* added *tentative* workaround for diffing data--might need to refactor ([920e66f](https://github.com/qri-io/qri/commit/920e66f))
* added initial validate command ([a647eb6](https://github.com/qri-io/qri/commit/a647eb6))
* **repo.Graph:** initial support for repo graphs ([1a5c9f9](https://github.com/qri-io/qri/commit/1a5c9f9))
* added new dataset request method `Diff` with minimal test ([c308bf0](https://github.com/qri-io/qri/commit/c308bf0))
* added new dataset request method `Diff` with minimal test ([4e7a033](https://github.com/qri-io/qri/commit/4e7a033))
* **p2p.DatasetInfo:** support dataset info over the wire ([821ed03](https://github.com/qri-io/qri/commit/821ed03))
* **p2p.QriConns:** list connected qri peers ([f067ce7](https://github.com/qri-io/qri/commit/f067ce7))
* **profile.Photo,profile.Poster:** hash-based profile images ([c0c1047](https://github.com/qri-io/qri/commit/c0c1047))
* **repo.Repo.Graph:** repos now have a graph method ([bc1f377](https://github.com/qri-io/qri/commit/bc1f377))
* **repo/graph:** added QueriesMap and DataNodes ([a285644](https://github.com/qri-io/qri/commit/a285644))
* **RPC:** change default port, provide RPC listener ([92e42ae](https://github.com/qri-io/qri/commit/92e42ae)), closes [#163](https://github.com/qri-io/qri/issues/163)
* added placeholders for `DatasetRequests.Diff` ([cffa623](https://github.com/qri-io/qri/commit/cffa623))
* added placeholders for `DatasetRequests.Diff` ([44f896b](https://github.com/qri-io/qri/commit/44f896b))
* new export command. ([f36da26](https://github.com/qri-io/qri/commit/f36da26))
* qri repos generate & store their own keys ([961b219](https://github.com/qri-io/qri/commit/961b219))


### BREAKING CHANGES

* **core.DatasetRequests.Save:** all api methods now route through either Init or Update.
This doesn't really matter, as no one was using this api anyway. But hey, it's
good to practice documenting these things



