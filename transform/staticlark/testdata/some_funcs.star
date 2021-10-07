
def use_branch(container):
  a = 1
  b = 2
  if a < b:
    c = b + 1
  else:
    c = a + 1
  print('%d' % c)


def branch_multiple(container):
  a = 1
  b = 2
  if a < b:
    c = b + 1
    d = a
    e = a + b
  else:
    c = a + 1
    print(c)
    e = c + 2
  print('%d' % e)


def branch_no_else(container):
  a = 1
  b = 2
  if a < b:
    c = b + 1
    print('%d' % c)
  print('%d' % b)


def branch_nested(container):
  a = 1
  b = 2
  if a < b:
    c = b + 1
    d = a
    if d > c:
      c = d + 2
    e = c + 2
  else:
    c = a + 1
    print(c)
    e = c + 2
  print('%d' % e)


def top_level_func():
  items = []
  use_branch(items)
  if len(items) > 0:
    branch_multiple(items)
  else:
    branch_no_else(items)
  another_function()


def another_function():
  more = []
  branch_nested(more)
  branch_no_else(more)
