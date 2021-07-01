
def use_branch():
  a = 1
  b = 2
  if a < b:
    c = b + 1
  else:
    c = a + 1
  print('%d' % c)


def branch_multiple():
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


def branch_no_else():
  a = 1
  b = 2
  if a < b:
    c = b + 1
    print('%d' % c)
  print('%d' % b)


def branch_nested():
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
