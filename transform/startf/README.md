# Qri Starlark Transformation Syntax

Qri ("query") is about datasets. Transformations are repeatable scripts for generating a dataset. [Starlark](https://github.com/google/starlark-go/blob/master/doc/spec.md) is a scripting language from Google that feels a lot like python. This package implements starlark as a _transformation syntax_. Starlark tranformations are about as close as one can get to the full power of a programming language as a transformation syntax. Often you need this degree of control to generate a dataset.

Typical examples of a starlark transformation include:
* combining paginated calls to an API into a single dataset
* downloading unstructured structured data from the internet to extract
* pulling raw data off the web & turning it into a datset

We're excited about starlark for a few reasons:
* **python syntax** - _many_ people working in data science these days write python, we like that, starlark likes that. dope.
* **deterministic subset of python** - unlike python, starlark removes properties that reduce introspection into code behaviour. things like `while` loops and recursive functions are omitted, making it possible for qri to infer how a given transformation will behave.
* **parallel execution** - thanks to this deterministic requirement (and lack of global interpreter lock) starlark functions can be executed in parallel. Combined with peer-2-peer networking, we're hoping to advance tranformations toward peer-driven distribed computing. More on that in the coming months.


## Getting started
If you're mainly interested in learning how to write starlark transformations, our [documentation](https://qri.io/docs) is a better place to start. If you're interested in contributing to the way starlark transformations work, this is the place!

The easiest way to see starlark transformations in action is to use [qri](https://github.com/qri-io/qri). This `startf` package powers all the starlark stuff in qri. Assuming you have the [go programming language](https://golang.org/) the following should work from a terminal:

<!--
docrun:
  pass: true
-->
```shell
# get this package
$ go get github.com/qri-io/startf

# navigate to package
$ cd $GOPATH/src/github.com/qri-io/startf
```

# run tests

<!--
docrun:
  pass: true
-->
```
$ go test ./...
```

Often the next steps are to install [qri](https://github.com/qri-io/qri), mess with this `startf` package, then rebuild qri with your changes to see them in action within qri itself.

## Starlark Special Functions

_Special Functions_ are the core of a starlark transform script. Here's an example of a simple data function that sets the body of a dataset to a constant:

<!--
docrun:
  test:
    call: transform(ds, ctx)
    actual: ds.get_meta()
    expect: {"hello": "world", "qri": "md:0"}
-->
```python
def transform(ds,ctx):
  ds.set_meta("hello","world")
```

Here's something slightly more complicated (but still very contrived) that modifies a dataset by adding up the length of all of the elements in a dataset body

<!--
docrun:
  test:
    setup: ds.set_body(["a","b","c"])
    call: transform(ds, ctx)
    actual: ds.get_body()
    expect: [{"total": 3.0}]
-->
```python
def transform(ds, ctx):
  body = ds.get_body()
  if body != None:
    count = 0
    for entry in body:
      count += len(entry)
  ds.set_body([{"total": count}])
```

Starlark special functions have a few rules on top of starlark itself:
* special functions *always* accept a _transformation context_ (the `ctx` arg)
* When you define a data function, qri calls it for you
* All special functions are optional (you don't _need_ to define them), except `transform`. transform is required.
* Special functions are always called in the same order

Another import special function is `download`, which allows access to the `http` package:

<!--
docrun:
  test:
    webproxy:
      url: http://example.com/data.json
      response: {"data":[4,5,6]}
    call: download(ctx)
    actual: ctx.download
    expect: {"data":[4.0,5.0,6.0]}
  save:
    filename: transform.star
-->
```python
load("http.star", "http")

def download(ctx):
  data = http.get("http://example.com/data.json")
  return data
```

The result of this special function can be accessed using `ctx.download`:

<!--
docrun:
  test:
    setup: ctx.download = ["test"]
    call: transform(ds, ctx)
    actual: ds.get_body()
    expect: ["test"]
  save:
    filename: transform.star
    append: true
-->
```python
def transform(ds, ctx):
  ds.set_body(ctx.download)
```

More docs on the provide API is coming soon.

## Running a transform

Let's say the above function is saved as `transform.star`. You can run it to create a new dataset by using:

<!--
docrun:
  pass: true
  # TODO: Run this command in a sandbox, using the transform.star created above.
-->
```
qri save --file=transform.star me/dataset_name
```

Or, you can add more details by creating a dataset file (saved as `dataset.yaml`, for example) with additional structure:

<!--
docrun:
  pass: true
  # TODO: Save this file to use in the command below.
-->
```
name: dataset_name
transform:
  scriptpath: transform.star
meta:
  title: My awesome dataset
```

Then invoke qri:

<!--
docrun:
  pass: true
  # TODO: Run this command in a sandbox, using the dataset.yaml created above.
-->
```
qri save --file=dataset.yaml
```

Fun! More info over on our [docs site](https://qri.io/docs)

** **
