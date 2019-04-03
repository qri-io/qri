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



