# Set a body containing the length of the previous body
def transform(ds, ctx):
  prev_body = ds.body
  ds.body = [['Number of Movies', len(prev_body.index)]]
