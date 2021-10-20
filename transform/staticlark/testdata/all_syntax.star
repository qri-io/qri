load("math.star", "math")

def some_vals():
  xs = [add_one(e) for e in range(4)]
  return xs

def add_one(value):
  return value + 1

def mult(a, b):
  return a * b

def half(n):
  return n / 2

def calc():
  def fact_iter(n):
    acc = 1
    for i in range(1, 10):
      acc = mult(acc, i)
      if i >= n:
        break
    return add_one(acc)
  return fact_iter(4)

def yet_more():
  def fact_iter(n):
    return half(n-1)
  return math.floor(fact_iter(4.3))

def do_math():
  num = 0
  for v in some_vals():
    num += v
  num += calc()
  num += yet_more()
  print(num)


do_math()
