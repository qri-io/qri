# predeclared globals for test: ctx, dl_ctx 
load("assert.star", "assert")

assert.eq(ctx.get_config("foo"), "bar")
assert.eq(ctx.get_secret("baz"), "bat")

ctx.set("foo", "bar")
assert.eq(ctx.get("foo"), "bar")
assert.eq(dl_ctx.download, True)