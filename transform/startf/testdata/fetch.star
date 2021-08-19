load("http.star", "http")
load("qri.star", "qri")

def download(ctx):
  res = http.get(test_server_url)
  return res.json()['foo']

def transform(ds, ctx):
  # TODO(dustmop): Fix this, decide how Dataframes get data from `download`
  ds.body = [['ctx.download']]
