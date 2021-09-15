ds = dataset.latest()

ds.set_meta("title", "new title")
dataset.commit(ds)
