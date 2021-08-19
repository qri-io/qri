def transform(ds, ctx):
  body = ds.body
  for row in body:
    row[1] = row[1] + 1
  ds.body = body

