load('assert.star', 'assert')

movies = load_dataset("peer/movies")

def transform(ds,ctx):
	assert.eq(movies.get_meta("title"), {"title": "example movie data", "qri": "md:0" })