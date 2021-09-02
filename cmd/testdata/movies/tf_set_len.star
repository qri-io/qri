# Set a body containing the length of the previous body
ds = dataset.latest()
prev_body = ds.body
ds.body = [['Number of Movies', len(prev_body.index)]]
dataset.commit(ds)
