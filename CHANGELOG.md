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
* updated cmd.diff to be compatible with updates to datasetDiffer ([201bdda](https://github.com/qri-io/qri/commit/201bdda))
* updated output param of core.Diff to `*map[string]*datasetDiffer.SubDiff` ([8e1aa39](https://github.com/qri-io/qri/commit/8e1aa39))
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
* **core.QueryRequests.Query:** check for previously exectued queries ([c3be454](https://github.com/qri-io/qri/commit/c3be454)), closes [#30](https://github.com/qri-io/qri/issues/30)
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



