load("time.star", "time")

def transform(ds, ctx):
  ds.set_body([str(time.now())])