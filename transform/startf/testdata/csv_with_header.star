load("dataframe.star", "dataframe")

content = """name,sound
cat,meow
dog,bark
"""
result = dataframe.parse_csv(content)

ds = dataset.latest()
ds.body = result
dataset.commit(ds)
