load("time.star", "time")

def transform(ds, ctx):
  body = ds.body
  body.append(str(time.now()))
  ds.body = body
