# transform that sets a simple body
ds = dataset.latest()
print("setting body")
ds.body = [[1,2,3]]
dataset.commit(ds)
