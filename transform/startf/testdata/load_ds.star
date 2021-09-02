load('assert.star', 'assert')

movies = load_dataset("peer/movies")


assert.eq(movies.get_meta("title"), {"title": "example movie data", "path": "/mem/QmZQNhYYVRx8LyMmPV9mqzVZVEeZKpso4Ywu7nwyWvT4X4", "qri": "md:0" })