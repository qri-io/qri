ds = dataset.latest()
ds.body = ds.body.append([['Batman', 126]])
dataset.commit(ds)
